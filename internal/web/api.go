package web

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/stats"
)

func (s *Server) registerAPI() {
	api := s.engine.Group("/api")
	api.GET("/stats", s.handleGetStats)
	api.GET("/repos", s.handleGetRepos)
	api.GET("/commits", s.handleGetCommits) // Renamed from /prs
	api.GET("/comparison", s.handleGetComparison)
	api.GET("/comparison/selected", s.handleGetSelectedComparison)
	api.POST("/sync", s.handleSync)
}

func (s *Server) handleGetStats(c *gin.Context) {
	repo := c.Query("repo")
	if repo == "" {
		values := make([]statsResponse, 0, len(s.repoNames))
		for _, repoID := range s.repoNames {
			value, err := s.statsCache.Load(repoID)
			if err == nil {
				values = append(values, newStatsResponse(value))
			}
		}
		c.JSON(http.StatusOK, gin.H{"stats": values})
		return
	}
	for _, repoID := range s.repoNames {
		if repoID == repo {
			value, err := s.statsCache.Load(repoID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, newStatsResponse(value))
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
}

type statsResponse struct {
	Repo         string                             `json:"repo"`
	ComputedAt   time.Time                          `json:"computed_at"`
	Contributors map[string]*cache.ContributorStats `json:"contributors"`
	Weekly       map[string]*cache.WeeklyEntry      `json:"weekly"`
}

func newStatsResponse(value *cache.StatsData) statsResponse {
	return statsResponse{
		Repo:         value.Repo,
		ComputedAt:   value.ComputedAt,
		Contributors: value.Contributors,
		Weekly:       value.Weekly,
	}
}

func (s *Server) handleGetRepos(c *gin.Context) {
	defaultRepositories := make([]string, len(s.defaultRepos))
	for i, repository := range s.defaultRepos {
		defaultRepositories[i] = repository.ID()
	}
	c.JSON(http.StatusOK, gin.H{
		"repos":                s.repoNames,
		"default_repositories": defaultRepositories,
	})
}

func (s *Server) handleGetComparison(c *gin.Context) {
	granularity := stats.Granularity(c.DefaultQuery("gran", string(stats.GranularityWeek)))
	switch granularity {
	case stats.GranularityWeek, stats.GranularityMonth, stats.GranularityQuarter, stats.GranularityYear:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid granularity"})
		return
	}

	repositories := queryList(c, "repo")
	if len(repositories) == 0 {
		repositories = s.repoNames
	}
	data := make([]*cache.StatsData, 0, len(repositories))
	for _, repository := range repositories {
		value, err := s.statsCache.Load(repository)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data = append(data, value)
	}

	location := time.Local
	if s.cfg != nil {
		configured, err := s.cfg.ReportingLocation()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		location = configured
	}
	result := stats.BuildComparisons(data, stats.ComparisonOptions{
		Granularity: granularity,
		Logins:      queryList(c, "login"),
		Location:    location,
	})
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleGetSelectedComparison(c *gin.Context) {
	granularity := stats.Granularity(c.DefaultQuery("gran", string(stats.GranularityWeek)))
	switch granularity {
	case stats.GranularityWeek, stats.GranularityMonth, stats.GranularityQuarter, stats.GranularityYear:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid granularity"})
		return
	}

	anchor := c.Query("anchor")
	comparisonKeys := queryList(c, "compare")
	if anchor == "" || len(comparisonKeys) == 0 || len(comparisonKeys) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "select an anchor and one to five comparison periods"})
		return
	}

	repositories := queryList(c, "repo")
	if len(repositories) == 0 {
		repositories = s.repoNames
	}
	data := make([]*cache.StatsData, 0, len(repositories))
	for _, repository := range repositories {
		value, err := s.statsCache.Load(repository)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data = append(data, value)
	}

	location := time.Local
	if s.cfg != nil {
		configured, err := s.cfg.ReportingLocation()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		location = configured
	}
	result := stats.CompareSelectedPeriods(data, stats.ComparisonOptions{
		Granularity: granularity,
		Logins:      queryList(c, "login"),
		Location:    location,
	}, anchor, comparisonKeys)
	c.JSON(http.StatusOK, result)
}

type syncRequest struct {
	Repositories []string `json:"repositories"`
}

func (s *Server) handleSync(c *gin.Context) {
	repoID := c.Query("repo")
	request := syncRequest{}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sync request"})
			return
		}
	}
	if repoID != "" {
		request.Repositories = []string{repoID}
	}

	targets := s.defaultRepos
	if len(request.Repositories) > 0 {
		available := make(map[string]config.Repository, len(s.repos))
		for _, repository := range s.repos {
			available[repository.ID()] = repository
		}

		targets = make([]config.Repository, 0, len(request.Repositories))
		seen := make(map[string]struct{}, len(request.Repositories))
		for _, id := range request.Repositories {
			if _, ok := seen[id]; ok {
				continue
			}
			repository, ok := available[id]
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown repository: " + id})
				return
			}
			seen[id] = struct{}{}
			targets = append(targets, repository)
		}
	}
	if len(targets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "select at least one repository"})
		return
	}

	if errors := s.syncer.SyncAll(context.Background(), targets, nil, 5); len(errors) > 0 {
		messages := make([]string, len(errors))
		for i, err := range errors {
			messages[i] = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": strings.Join(messages, "; ")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sync completed"})
}

// CommitInfo 是对外暴露的提交摘要。
type CommitInfo struct {
	Repo         string    `json:"repo"`
	SHA          string    `json:"sha"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	Participants []string  `json:"participants"`
	Date         time.Time `json:"date"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
}

func (s *Server) handleGetCommits(c *gin.Context) {
	selectedRepos := queryList(c, "repo")
	selectedLogins := queryList(c, "login")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var hasFrom, hasTo bool
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from, expect RFC3339"})
			return
		}
		from, hasFrom = t, true
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to, expect RFC3339"})
			return
		}
		to, hasTo = t, true
	}

	repos := s.repoNames
	if len(selectedRepos) > 0 {
		repos = selectedRepos
	}
	logins := make(map[string]struct{}, len(selectedLogins))
	for _, login := range selectedLogins {
		logins[login] = struct{}{}
	}

	var result []CommitInfo
	for _, r := range repos {
		raw, err := s.rawCache.Load(r)
		if err != nil {
			continue
		}
		for _, commit := range raw.Commits {
			if hasFrom && commit.Date.Before(from) {
				continue
			}
			if hasTo && !commit.Date.Before(to) {
				continue
			}
			if len(logins) > 0 {
				found := false
				parts := commit.Participants
				if len(parts) == 0 {
					parts = []string{commit.Author}
				}
				for _, p := range parts {
					if _, ok := logins[p]; ok {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			participants := commit.Participants
			if len(participants) == 0 {
				participants = []string{commit.Author}
			}

			result = append(result, CommitInfo{
				Repo:         r,
				SHA:          commit.SHA,
				Title:        commit.Message,
				Author:       commit.Author,
				Participants: participants,
				Date:         commit.Date,
				Additions:    commit.Additions,
				Deletions:    commit.Deletions,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Date.Equal(result[j].Date) {
			if result[i].Repo == result[j].Repo {
				return result[i].SHA < result[j].SHA
			}
			return result[i].Repo < result[j].Repo
		}
		return result[i].Date.After(result[j].Date)
	})

	total := len(result)

	page := 1
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && p > 0 {
		page = p
	}
	perPage := 100
	if pp, err := strconv.Atoi(c.DefaultQuery("per_page", "100")); err == nil && pp > 0 && pp <= 100 {
		perPage = pp
	}

	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, gin.H{
		"commits":  result[start:end], // Changed from "prs"
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func queryList(c *gin.Context, key string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, value := range c.QueryArray(key) {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

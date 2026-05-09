package web

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/config"
)

func (s *Server) registerAPI() {
	api := s.engine.Group("/api")
	api.GET("/stats", s.handleGetStats)
	api.GET("/repos", s.handleGetRepos)
	api.GET("/commits", s.handleGetCommits) // Renamed from /prs
	api.POST("/sync", s.handleSync)
}

func (s *Server) handleGetStats(c *gin.Context) {
	repo := c.Query("repo")
	if repo == "" {
		c.JSON(http.StatusOK, gin.H{"stats": s.stats})
		return
	}
	for _, st := range s.stats {
		if st.Repo == repo {
			c.JSON(http.StatusOK, st)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
}

func (s *Server) handleGetRepos(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"repos": s.repoNames})
}

func (s *Server) handleSync(c *gin.Context) {
	repoID := c.Query("repo")
	go func() {
		if repoID != "" {
			var target *config.Repository
			for _, r := range s.repos {
				if r.ID() == repoID {
					target = &r
					break
				}
			}
			if target != nil {
				_ = s.syncer.SyncRepo(context.Background(), *target)
			}
		} else {
			s.syncer.SyncAll(context.Background(), s.repos, nil, 5)
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started"})
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
	repo := c.Query("repo")   // 可选；空 = 全部仓库
	login := c.Query("login") // 可选；空 = 不按贡献者过滤
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
	if repo != "" {
		repos = []string{repo}
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
			if login != "" {
				found := false
				parts := commit.Participants
				if len(parts) == 0 {
					parts = []string{commit.Author}
				}
				for _, p := range parts {
					if p == login {
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

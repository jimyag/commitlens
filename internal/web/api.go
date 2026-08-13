package web

import (
	"context"
	"fmt"
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
	api.GET("/identities", s.handleGetIdentities)
	api.PUT("/identities", s.handleUpdateIdentities)
	api.GET("/sync/status", s.handleGetSyncStatus)
	api.POST("/sync", s.handleSync)
	api.GET("/github/contributions", s.handleGetGitHubContributions)
	api.GET("/github/contribution-organizations", s.handleGetGitHubContributionOrganizations)
}

func (s *Server) handleGetGitHubContributionOrganizations(c *gin.Context) {
	if s.githubClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub contribution client is unavailable"})
		return
	}
	currentYear := time.Now().UTC().Year()
	fromYear, err := queryYear(c, "from_year", currentYear-5)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	toYear, err := queryYear(c, "to_year", currentYear)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if fromYear > toYear || toYear-fromYear > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year range must be ordered and no longer than 20 years"})
		return
	}
	user, organizations, err := s.githubClient.ContributionOrganizations(
		c.Request.Context(), strings.TrimSpace(c.Query("user")), fromYear, toYear,
	)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "token is required") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "organizations": organizations})
}

func (s *Server) handleGetGitHubContributions(c *gin.Context) {
	if s.githubClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub contribution client is unavailable"})
		return
	}
	organization := strings.TrimSpace(c.Query("organization"))
	currentYear := time.Now().UTC().Year()
	fromYear, err := queryYear(c, "from_year", currentYear-5)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	toYear, err := queryYear(c, "to_year", currentYear)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if fromYear > toYear || toYear-fromYear > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year range must be ordered and no longer than 20 years"})
		return
	}
	report, err := s.githubClient.Contributions(
		c.Request.Context(),
		strings.TrimSpace(c.Query("user")),
		organization,
		fromYear,
		toYear,
	)
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "token is required") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func queryYear(c *gin.Context, name string, defaultValue int) (int, error) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return defaultValue, nil
	}
	year, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a year", name)
	}
	return year, nil
}

type repositorySyncStatus struct {
	Repo        string     `json:"repo"`
	State       string     `json:"state"`
	LastLog     string     `json:"last_log"`
	Error       string     `json:"error,omitempty"`
	LastAttempt *time.Time `json:"last_attempt,omitempty"`
	LastSuccess *time.Time `json:"last_success,omitempty"`
	DurationMS  int64      `json:"duration_ms"`
	CommitCount int        `json:"commit_count"`
	HasData     bool       `json:"has_data"`
}

func (s *Server) handleGetSyncStatus(c *gin.Context) {
	statuses := make([]repositorySyncStatus, 0, len(s.repoNames))
	for _, repoID := range s.repoNames {
		status, err := s.rawCache.LoadStatus(repoID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		raw, err := s.rawCache.Load(repoID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		state := status.State
		switch {
		case status.Syncing:
			state = "syncing"
		case state != "":
		case status.Error != "":
			state = "failed"
		case !raw.LastUpdated.IsZero():
			state = "success"
		default:
			state = "not_synced"
		}
		lastSuccess := status.LastSuccess
		if lastSuccess.IsZero() && state == "success" {
			lastSuccess = raw.LastUpdated
		}
		statuses = append(statuses, repositorySyncStatus{
			Repo:        repoID,
			State:       state,
			LastLog:     status.LastLog,
			Error:       status.Error,
			LastAttempt: optionalTime(status.LastAttempt),
			LastSuccess: optionalTime(lastSuccess),
			DurationMS:  status.Duration.Milliseconds(),
			CommitCount: len(raw.Commits),
			HasData:     !raw.LastUpdated.IsZero(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"statuses": statuses})
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

type contributorIdentity struct {
	Canonical   string   `json:"canonical"`
	Aliases     []string `json:"aliases"`
	Emails      []string `json:"emails"`
	CommitCount int      `json:"commit_count"`
	Hidden      bool     `json:"hidden"`
	LikelyBot   bool     `json:"likely_bot"`
}

type identitySettingsResponse struct {
	Identities   []contributorIdentity `json:"identities"`
	UserMap      map[string][]string   `json:"user_map"`
	IgnoredUsers []string              `json:"ignored_users"`
}

type updateIdentitySettingsRequest struct {
	UserMap      map[string][]string `json:"user_map"`
	IgnoredUsers []string            `json:"ignored_users"`
}

func (s *Server) handleGetIdentities(c *gin.Context) {
	c.JSON(http.StatusOK, s.identitySettingsResponse())
}

func (s *Server) handleUpdateIdentities(c *gin.Context) {
	var request updateIdentitySettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid identity settings"})
		return
	}
	if err := validateIdentitySettings(request.UserMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.cfg.UpdateIdentitySettings(request.UserMap, request.IgnoredUsers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, repoID := range s.repoNames {
		raw, err := s.rawCache.Load(repoID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := s.statsCache.Save(stats.Aggregate(raw, s.cfg)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, s.identitySettingsResponse())
}

func (s *Server) identitySettingsResponse() identitySettingsResponse {
	userMap, ignoredUsers := s.cfg.IdentitySettings()
	ignored := make(map[string]struct{}, len(ignoredUsers))
	aliasMap := make(map[string]string)
	for canonical, aliases := range userMap {
		aliasMap[strings.ToLower(canonical)] = canonical
		for _, alias := range aliases {
			aliasMap[strings.ToLower(alias)] = canonical
		}
	}
	resolveUser := func(name string) string {
		if canonical, ok := aliasMap[strings.ToLower(name)]; ok {
			return canonical
		}
		return name
	}
	for _, name := range ignoredUsers {
		ignored[strings.ToLower(name)] = struct{}{}
	}

	type identityAccumulator struct {
		canonical string
		aliases   map[string]string
		emails    map[string]string
		commits   int
	}
	accumulators := make(map[string]*identityAccumulator)
	getAccumulator := func(canonical string) *identityAccumulator {
		key := strings.ToLower(canonical)
		if accumulators[key] == nil {
			accumulators[key] = &identityAccumulator{
				canonical: canonical,
				aliases:   make(map[string]string),
				emails:    make(map[string]string),
			}
		}
		return accumulators[key]
	}

	for canonical, aliases := range userMap {
		acc := getAccumulator(canonical)
		for _, alias := range aliases {
			acc.aliases[strings.ToLower(alias)] = alias
		}
	}
	for _, repoID := range s.repoNames {
		raw, err := s.rawCache.Load(repoID)
		if err != nil {
			continue
		}
		for _, commit := range raw.Commits {
			participants := commit.Participants
			if len(participants) == 0 {
				participants = []string{commit.Author}
			}
			seen := make(map[string]struct{}, len(participants))
			for _, participant := range participants {
				canonical := resolveUser(participant)
				key := strings.ToLower(canonical)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				acc := getAccumulator(canonical)
				acc.commits++
				if !strings.EqualFold(participant, canonical) {
					acc.aliases[strings.ToLower(participant)] = participant
				}
				if strings.EqualFold(participant, commit.Author) && commit.AuthorEmail != "" {
					acc.emails[strings.ToLower(commit.AuthorEmail)] = commit.AuthorEmail
				}
			}
		}
	}

	identities := make([]contributorIdentity, 0, len(accumulators))
	for _, acc := range accumulators {
		canonical := acc.canonical
		aliases := mapValues(acc.aliases)
		emails := mapValues(acc.emails)
		_, hidden := ignored[strings.ToLower(canonical)]
		identities = append(identities, contributorIdentity{
			Canonical:   canonical,
			Aliases:     aliases,
			Emails:      emails,
			CommitCount: acc.commits,
			Hidden:      hidden,
			LikelyBot:   isLikelyBot(canonical, aliases),
		})
	}
	sort.Slice(identities, func(i, j int) bool {
		if identities[i].CommitCount != identities[j].CommitCount {
			return identities[i].CommitCount > identities[j].CommitCount
		}
		return strings.ToLower(identities[i].Canonical) < strings.ToLower(identities[j].Canonical)
	})
	return identitySettingsResponse{
		Identities:   identities,
		UserMap:      userMap,
		IgnoredUsers: ignoredUsers,
	}
}

func mapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}

func isLikelyBot(canonical string, aliases []string) bool {
	names := append([]string{canonical}, aliases...)
	for _, name := range names {
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, "[bot]") || strings.HasSuffix(lower, "-bot") ||
			strings.HasSuffix(lower, "_bot") || strings.Contains(lower, "automation") {
			return true
		}
	}
	return false
}

func validateIdentitySettings(userMap map[string][]string) error {
	owners := make(map[string]string)
	for canonical, aliases := range userMap {
		canonical = strings.TrimSpace(canonical)
		if canonical == "" {
			return fmt.Errorf("canonical identity cannot be empty")
		}
		for _, name := range append([]string{canonical}, aliases...) {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if owner, exists := owners[key]; exists && !strings.EqualFold(owner, canonical) {
				return fmt.Errorf("identity %q belongs to both %q and %q", name, owner, canonical)
			}
			owners[key] = canonical
		}
	}
	return nil
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
	Repo           string                                          `json:"repo"`
	ComputedAt     time.Time                                       `json:"computed_at"`
	Contributors   map[string]*cache.ContributorStats              `json:"contributors"`
	Weekly         map[string]*cache.WeeklyEntry                   `json:"weekly"`
	RepositorySize map[string]map[string]cache.RepositorySizePoint `json:"repository_size,omitempty"`
}

func newStatsResponse(value *cache.StatsData) statsResponse {
	return statsResponse{
		Repo:           value.Repo,
		ComputedAt:     value.ComputedAt,
		Contributors:   value.Contributors,
		Weekly:         value.Weekly,
		RepositorySize: value.RepositorySize,
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

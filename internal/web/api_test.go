package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	gitdata "github.com/jimyag/commitlens/internal/git"
	statdata "github.com/jimyag/commitlens/internal/stats"
)

func TestGetReposIncludesDefaultRepositories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{
		engine: gin.New(),
		repoNames: []string{
			"example-org/alpha",
			"example-org/beta",
		},
		defaultRepos: []config.Repository{{Owner: "example-org", Repo: "alpha"}},
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/repos status = %d", response.Code)
	}
	var body struct {
		Repositories        []string `json:"repos"`
		DefaultRepositories []string `json:"default_repositories"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Repositories) != 2 || len(body.DefaultRepositories) != 1 {
		t.Fatalf("unexpected response: %#v", body)
	}
	if body.DefaultRepositories[0] != "example-org/alpha" {
		t.Fatalf("unexpected defaults: %#v", body.DefaultRepositories)
	}
}

func TestSyncRejectsUnknownRepository(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{
		engine: gin.New(),
		repos:  []config.Repository{{Owner: "example-org", Repo: "alpha"}},
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sync",
		strings.NewReader(`{"repositories":["example-org/missing"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/sync status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGetStatsReadsLatestCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	statsCache := cache.NewStatsCache(t.TempDir())
	server := &Server{
		engine:     gin.New(),
		repoNames:  []string{"example-org/alpha"},
		statsCache: statsCache,
	}
	server.registerAPI()

	if err := statsCache.Save(&cache.StatsData{
		Repo:         "example-org/alpha",
		Contributors: map[string]*cache.ContributorStats{},
		Weekly:       map[string]*cache.WeeklyEntry{},
	}); err != nil {
		t.Fatalf("save stats: %v", err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "example-org/alpha") {
		t.Fatalf("GET /api/stats status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGetCommitsSupportsMultipleRepositoriesAndContributors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawCache := cache.NewRawCache(t.TempDir())
	for _, raw := range []*cache.RawData{
		{
			Repo: "sample/alpha",
			Commits: []gitdata.Commit{{
				SHA:          "alpha-sha",
				Author:       "alice",
				Participants: []string{"alice"},
				Date:         time.Date(2026, time.July, 2, 0, 0, 0, 0, time.UTC),
			}},
		},
		{
			Repo: "sample/beta",
			Commits: []gitdata.Commit{{
				SHA:          "beta-sha",
				Author:       "bob",
				Participants: []string{"bob"},
				Date:         time.Date(2026, time.July, 2, 0, 0, 0, 0, time.UTC),
			}},
		},
	} {
		if err := rawCache.Save(raw); err != nil {
			t.Fatalf("save raw cache: %v", err)
		}
	}

	server := &Server{
		engine:    gin.New(),
		repoNames: []string{"sample/alpha", "sample/beta"},
		rawCache:  rawCache,
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/commits?repo=sample%2Fbeta%2Csample%2Falpha&login=alice%2Cbob",
		nil,
	)
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/commits status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Commits []CommitInfo `json:"commits"`
		Total   int          `json:"total"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total != 2 || len(body.Commits) != 2 {
		t.Fatalf("unexpected commits response: %#v", body)
	}
	if body.Commits[0].Repo != "sample/alpha" || body.Commits[1].Repo != "sample/beta" {
		t.Fatalf("unexpected commit order: %#v", body.Commits)
	}
}

func TestGetComparisonUsesSelectedRepositories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	statsCache := cache.NewStatsCache(t.TempDir())
	for _, data := range []*cache.StatsData{
		{
			Repo: "sample/alpha",
			Daily: map[string]*cache.WeeklyEntry{
				"2026-06-10": {TotalCommits: 2, Contributors: map[string]*cache.ContributorWeeklyStats{}},
				"2026-07-10": {TotalCommits: 3, Contributors: map[string]*cache.ContributorWeeklyStats{}},
			},
			Contributors: map[string]*cache.ContributorStats{},
			Weekly:       map[string]*cache.WeeklyEntry{},
		},
		{
			Repo: "sample/beta",
			Daily: map[string]*cache.WeeklyEntry{
				"2026-07-10": {TotalCommits: 7, Contributors: map[string]*cache.ContributorWeeklyStats{}},
			},
			Contributors: map[string]*cache.ContributorStats{},
			Weekly:       map[string]*cache.WeeklyEntry{},
		},
	} {
		if err := statsCache.Save(data); err != nil {
			t.Fatalf("save stats: %v", err)
		}
	}

	server := &Server{
		engine:     gin.New(),
		repoNames:  []string{"sample/alpha", "sample/beta"},
		statsCache: statsCache,
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/comparison?repo=sample%2Falpha&gran=month",
		nil,
	)
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/comparison status = %d, body = %s", response.Code, response.Body.String())
	}
	var body statdata.ComparisonSet
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body.Comparisons["2026-07"].Anchor.Metrics.Commits; got != 3 {
		t.Fatalf("selected repository commits = %d, want 3", got)
	}
}

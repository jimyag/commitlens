package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	gitdata "github.com/jimyag/commitlens/internal/git"
	ghapi "github.com/jimyag/commitlens/internal/github"
	statdata "github.com/jimyag/commitlens/internal/stats"
)

type fakeContributionClient struct {
	report ghapi.ContributionReport
	err    error
}

func (f fakeContributionClient) Contributions(_ context.Context, user, organization string, fromYear, toYear int) (ghapi.ContributionReport, error) {
	report := f.report
	report.User = user
	report.Organization.Login = organization
	report.FromYear = fromYear
	report.ToYear = toYear
	return report, f.err
}

func (f fakeContributionClient) ContributionOrganizations(_ context.Context, user string, _, _ int) (string, []ghapi.Organization, error) {
	if user == "" {
		user = "viewer"
	}
	return user, []ghapi.Organization{{Login: "alpha"}, {Login: "zeta"}}, f.err
}

func TestGetGitHubContributions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{
		engine: gin.New(),
		githubClient: fakeContributionClient{report: ghapi.ContributionReport{
			TotalContributions: 12,
			Years:              []ghapi.YearContributions{{Year: 2024, TotalContributions: 12}},
		}},
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/github/contributions?organization=qbox&user=jimyag&from_year=2023&to_year=2024", nil)
	server.engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET contributions status = %d, body = %s", response.Code, response.Body.String())
	}
	var report ghapi.ContributionReport
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.User != "jimyag" || report.Organization.Login != "qbox" || report.FromYear != 2023 || len(report.Years) != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestGetGitHubContributionsValidatesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{engine: gin.New(), githubClient: fakeContributionClient{}}
	server.registerAPI()

	response := httptest.NewRecorder()
	target := "/api/github/contributions?organization=qbox&from_year=2025&to_year=2024"
	server.engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET %s status = %d", target, response.Code)
	}
}

func TestGetGitHubContributionsAllowsAllOrganizations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{engine: gin.New(), githubClient: fakeContributionClient{}}
	server.registerAPI()
	response := httptest.NewRecorder()
	server.engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/github/contributions?from_year=2024&to_year=2024", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET all contributions status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGetGitHubContributionOrganizations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{engine: gin.New(), githubClient: fakeContributionClient{}}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/github/contribution-organizations?from_year=2023&to_year=2024", nil)
	server.engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"login":"alpha"`) {
		t.Fatalf("GET contribution organizations status = %d, body = %s", response.Code, response.Body.String())
	}
}

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

func TestUpdateIdentitiesPersistsAndRebuildsStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("repositories: []\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	rawCache := cache.NewRawCache(filepath.Join(dir, "cache"))
	statsCache := cache.NewStatsCache(filepath.Join(dir, "cache"))
	if err := rawCache.Save(&cache.RawData{
		Repo: "sample/alpha",
		Commits: []gitdata.Commit{{
			Author:       "Alice Example",
			AuthorEmail:  "alice@example.test",
			Participants: []string{"Alice Example", "automation[bot]"},
			Date:         time.Now().UTC(),
		}},
	}); err != nil {
		t.Fatalf("save raw cache: %v", err)
	}

	server := &Server{
		engine:     gin.New(),
		repoNames:  []string{"sample/alpha"},
		cfg:        cfg,
		rawCache:   rawCache,
		statsCache: statsCache,
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/identities",
		strings.NewReader(`{"user_map":{"alice":["Alice Example"]},"ignored_users":["automation[bot]"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("PUT /api/identities status = %d, body = %s", response.Code, response.Body.String())
	}
	rebuilt, err := statsCache.Load("sample/alpha")
	if err != nil {
		t.Fatalf("load rebuilt stats: %v", err)
	}
	if rebuilt.Contributors["alice"] == nil {
		t.Fatalf("canonical contributor missing: %#v", rebuilt.Contributors)
	}
	if rebuilt.Contributors["automation[bot]"] != nil {
		t.Fatalf("ignored contributor remains: %#v", rebuilt.Contributors)
	}
	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if len(reloaded.UserMap["alice"]) != 1 || len(reloaded.IgnoredUsers) != 1 {
		t.Fatalf("identity settings not persisted: %#v %#v", reloaded.UserMap, reloaded.IgnoredUsers)
	}
}

func TestGetSyncStatusDistinguishesFailureFromEmptyRepository(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawCache := cache.NewRawCache(t.TempDir())
	now := time.Now().UTC()
	if err := rawCache.Save(&cache.RawData{Repo: "sample/empty", LastUpdated: now}); err != nil {
		t.Fatalf("save empty raw cache: %v", err)
	}
	if err := rawCache.SaveStatus(&cache.SyncStatus{
		Repo:        "sample/empty",
		State:       "success",
		LastSuccess: now,
		Duration:    2 * time.Second,
	}); err != nil {
		t.Fatalf("save success status: %v", err)
	}
	if err := rawCache.SaveStatus(&cache.SyncStatus{
		Repo:  "sample/failing",
		State: "failed",
		Error: "authentication failed",
	}); err != nil {
		t.Fatalf("save failure status: %v", err)
	}

	server := &Server{
		engine:    gin.New(),
		repoNames: []string{"sample/empty", "sample/failing"},
		rawCache:  rawCache,
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/sync/status status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Statuses []repositorySyncStatus `json:"statuses"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Statuses) != 2 {
		t.Fatalf("unexpected statuses: %#v", body.Statuses)
	}
	if body.Statuses[0].State != "success" || body.Statuses[0].HasData != true || body.Statuses[0].CommitCount != 0 {
		t.Fatalf("empty repository should be a successful sync: %#v", body.Statuses[0])
	}
	if body.Statuses[1].State != "failed" || body.Statuses[1].Error == "" || body.Statuses[1].HasData {
		t.Fatalf("failure status not preserved: %#v", body.Statuses[1])
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

func TestGetSelectedComparisonUsesRequestedPeriods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	statsCache := cache.NewStatsCache(t.TempDir())
	if err := statsCache.Save(&cache.StatsData{
		Repo: "sample/alpha",
		Daily: map[string]*cache.WeeklyEntry{
			"2026-06-10": {TotalCommits: 2, Contributors: map[string]*cache.ContributorWeeklyStats{}},
			"2026-07-10": {TotalCommits: 3, Contributors: map[string]*cache.ContributorWeeklyStats{}},
		},
		Contributors: map[string]*cache.ContributorStats{},
		Weekly:       map[string]*cache.WeeklyEntry{},
	}); err != nil {
		t.Fatalf("save stats: %v", err)
	}

	server := &Server{
		engine:     gin.New(),
		repoNames:  []string{"sample/alpha"},
		statsCache: statsCache,
	}
	server.registerAPI()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/comparison/selected?gran=month&anchor=2026-07&compare=2026-06",
		nil,
	)
	server.engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/comparison/selected status = %d, body = %s", response.Code, response.Body.String())
	}
	var body statdata.SelectedComparison
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Anchor.Metrics.Commits != 3 || body.Comparisons[0].Metrics.Commits != 2 {
		t.Fatalf("unexpected selected comparison: %#v", body)
	}
}

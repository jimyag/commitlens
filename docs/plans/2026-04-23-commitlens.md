# CommitLens Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 构建一个统计 GitHub 仓库各贡献者 PR/commit/增删行的命令行工具，支持 TUI（默认）和 WebUI（--web）两种模式。

**Architecture:** 单二进制，cobra 处理 CLI，bubbletea 驱动 TUI，gin 提供 HTTP API，React+ECharts 前端 embed 进二进制。GitHub API 拉取 merged PR 数据，JSON 文件分层缓存（raw + stats），启动时自动增量更新。

**Tech Stack:** Go 1.26, cobra, bubbletea v2 (charm.land/bubbletea/v2), lipgloss, ntcharts, gin, React 19, Apache ECharts, Vite, gopkg.in/yaml.v3

---

## 项目结构

```
commitlens/
├── main.go
├── cmd/
│   └── root.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── github/
│   │   └── client.go
│   ├── cache/
│   │   ├── raw.go
│   │   └── stats.go
│   ├── stats/
│   │   └── aggregator.go
│   ├── sync/
│   │   └── syncer.go
│   ├── tui/
│   │   ├── app.go
│   │   ├── keys.go
│   │   └── views/
│   │       ├── summary.go
│   │       ├── repo.go
│   │       └── trend.go
│   └── web/
│       ├── server.go
│       └── api.go
├── frontend/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── api.ts
│   │   └── components/
│   │       ├── ContributorTable.tsx
│   │       └── TrendChart.tsx
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts
└── go.mod
```

---

## Task 1: 初始化项目

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

**Step 1: 初始化 go module**

```bash
cd /Users/jimyag/src/github/jimyag/commitlens
go mod init github.com/jimyag/commitlens
```

**Step 2: 安装 Go 依赖**

```bash
go get github.com/spf13/cobra@latest
go get charm.land/bubbletea/v2@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/NimbleMarkets/ntcharts@latest
go get github.com/gin-gonic/gin@latest
go get gopkg.in/yaml.v3@latest
go get github.com/google/go-github/v72@latest
go get golang.org/x/oauth2@latest
```

**Step 3: 创建 main.go**

```go
package main

import (
	"embed"

	"github.com/jimyag/commitlens/cmd"
)

//go:embed frontend/dist
var frontendFS embed.FS

func main() {
	cmd.ExecuteWithAssets(frontendFS)
}
```

**Step 4: 创建 cmd/root.go**

```go
package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	globalAssets embed.FS
	webMode      bool
	webPort      int
	configFile   string
)

var rootCmd = &cobra.Command{
	Use:   "commitlens",
	Short: "GitHub contribution stats viewer",
	RunE:  run,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func ExecuteWithAssets(assets embed.FS) {
	globalAssets = assets
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().BoolVar(&webMode, "web", false, "Start web UI mode")
	rootCmd.Flags().IntVar(&webPort, "port", 8080, "Web UI port")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Config file path (default: ~/.commitlens/config.yaml)")
}

func run(cmd *cobra.Command, args []string) error {
	if webMode {
		fmt.Printf("Starting web UI on http://localhost:%d\n", webPort)
		// TODO: start web server
		return nil
	}
	fmt.Fprintln(os.Stderr, "TUI mode not yet implemented")
	return nil
}
```

**Step 5: 验证编译**

```bash
go build ./...
```

Expected: 无报错。

**Step 6: Commit**

```bash
git add .
git commit -m "feat: init project structure"
```

---

## Task 2: config 包

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: 写失败测试**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jimyag/commitlens/internal/config"
)

func TestLoad_defaults(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	content := `
repositories:
  - owner: jimyag
    repo: commitlens
`
	os.WriteFile(cfgPath, []byte(content), 0644)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Repositories) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repositories))
	}
	if cfg.Repositories[0].Owner != "jimyag" {
		t.Errorf("expected owner jimyag, got %s", cfg.Repositories[0].Owner)
	}
	if cfg.Cache.Dir == "" {
		t.Error("expected default cache dir to be set")
	}
	if cfg.Web.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Web.Port)
	}
}
```

**Step 2: 跑测试确认失败**

```bash
go test ./internal/config/...
```

Expected: FAIL "cannot find package"

**Step 3: 实现 config.go**

```go
// internal/config/config.go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Repository struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
}

type GitHub struct {
	Token string `yaml:"token"`
}

type Cache struct {
	Dir string `yaml:"dir"`
}

type Web struct {
	Port int `yaml:"port"`
}

type Config struct {
	GitHub       GitHub       `yaml:"github"`
	Repositories []Repository `yaml:"repositories"`
	Cache        Cache        `yaml:"cache"`
	Web          Web          `yaml:"web"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	setDefaults(cfg)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Cache.Dir == "" {
		home, _ := os.UserHomeDir()
		cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	}
	if cfg.Web.Port == 0 {
		cfg.Web.Port = 8080
	}
	return cfg, nil
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".commitlens", "config.yaml")
}

func setDefaults(cfg *Config) {
	home, _ := os.UserHomeDir()
	cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	cfg.Web.Port = 8080
}
```

**Step 4: 跑测试确认通过**

```bash
go test ./internal/config/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package"
```

---

## Task 3: GitHub API 客户端

**Files:**
- Create: `internal/github/client.go`
- Create: `internal/github/client_test.go`

**Step 1: 写失败测试（用 httptest mock）**

```go
// internal/github/client_test.go
package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/github"
)

func TestClient_GetMergedPRsSince(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prs := []map[string]any{
			{
				"number":    1,
				"title":     "test pr",
				"user":      map[string]any{"login": "jimyag", "avatar_url": "https://example.com/avatar"},
				"merged_at": time.Now().Format(time.RFC3339),
				"additions": 100,
				"deletions": 20,
			},
		}
		json.NewEncoder(w).Encode(prs)
	}))
	defer srv.Close()

	client := github.NewClient("fake-token")
	client.SetBaseURL(srv.URL)

	since := time.Now().Add(-24 * time.Hour)
	prs, err := client.GetMergedPRsSince(context.Background(), "jimyag", "commitlens", since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) == 0 {
		t.Fatal("expected at least 1 PR")
	}
}
```

**Step 2: 跑测试确认失败**

```bash
go test ./internal/github/... 2>&1 | head -5
```

Expected: FAIL "cannot find package"

**Step 3: 实现 client.go**

```go
// internal/github/client.go
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type PR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatar_url"`
	MergedAt  time.Time `json:"merged_at"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
	Commits   []Commit  `json:"commits"`
}

type Commit struct {
	SHA       string `json:"sha"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	if token == "" {
		token = tokenFromGH()
	}
	return &Client{
		token:      token,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

func tokenFromGH() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github API error: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) GetMergedPRsSince(ctx context.Context, owner, repo string, since time.Time) ([]PR, error) {
	var result []PR
	page := 1
	for {
		var raw []struct {
			Number   int    `json:"number"`
			Title    string `json:"title"`
			User     struct {
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
			} `json:"user"`
			MergedAt  *time.Time `json:"merged_at"`
			Additions int        `json:"additions"`
			Deletions int        `json:"deletions"`
		}
		path := fmt.Sprintf("/repos/%s/%s/pulls?state=closed&per_page=100&page=%d&sort=updated&direction=desc", owner, repo, page)
		if err := c.get(ctx, path, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			break
		}
		done := false
		for _, r := range raw {
			if r.MergedAt == nil {
				continue
			}
			if r.MergedAt.Before(since) {
				done = true
				break
			}
			pr := PR{
				Number:    r.Number,
				Title:     r.Title,
				Author:    r.User.Login,
				AvatarURL: r.User.AvatarURL,
				MergedAt:  *r.MergedAt,
				Additions: r.Additions,
				Deletions: r.Deletions,
			}
			commits, _ := c.GetPRCommits(ctx, owner, repo, r.Number)
			pr.Commits = commits
			result = append(result, pr)
		}
		if done {
			break
		}
		page++
	}
	return result, nil
}

func (c *Client) GetPRCommits(ctx context.Context, owner, repo string, prNumber int) ([]Commit, error) {
	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commit"`
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
		Stats struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
		} `json:"stats"`
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/commits?per_page=100", owner, repo, prNumber)
	if err := c.get(ctx, path, &raw); err != nil {
		return nil, err
	}
	commits := make([]Commit, 0, len(raw))
	for _, r := range raw {
		author := r.Commit.Author.Name
		if r.Author != nil {
			author = r.Author.Login
		}
		commits = append(commits, Commit{
			SHA:       r.SHA,
			Author:    author,
			Message:   r.Commit.Message,
			Additions: r.Stats.Additions,
			Deletions: r.Stats.Deletions,
		})
	}
	return commits, nil
}
```

**Step 4: 跑测试**

```bash
go test ./internal/github/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/github/
git commit -m "feat: add github api client"
```

---

## Task 4: cache 包（raw + stats）

**Files:**
- Create: `internal/cache/raw.go`
- Create: `internal/cache/stats.go`
- Create: `internal/cache/cache_test.go`

**Step 1: 写失败测试**

```go
// internal/cache/cache_test.go
package cache_test

import (
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	gh "github.com/jimyag/commitlens/internal/github"
)

func TestRawCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	rc := cache.NewRawCache(dir)

	raw := &cache.RawData{
		Repo:        "jimyag/commitlens",
		LastUpdated: time.Now().UTC(),
		PRs: []gh.PR{
			{Number: 1, Author: "jimyag", Additions: 100, Deletions: 20},
		},
	}
	if err := rc.Save(raw); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := rc.Load("jimyag/commitlens")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.PRs) != 1 {
		t.Errorf("expected 1 PR, got %d", len(loaded.PRs))
	}
}

func TestStatsCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	sc := cache.NewStatsCache(dir)

	stats := &cache.StatsData{
		Repo:       "jimyag/commitlens",
		ComputedAt: time.Now().UTC(),
		Contributors: map[string]*cache.ContributorStats{
			"jimyag": {Login: "jimyag", PRCount: 5, CommitCount: 20, Additions: 500, Deletions: 100},
		},
	}
	if err := sc.Save(stats); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := sc.Load("jimyag/commitlens")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Contributors["jimyag"].PRCount != 5 {
		t.Errorf("expected PRCount 5, got %d", loaded.Contributors["jimyag"].PRCount)
	}
}
```

**Step 2: 跑测试确认失败**

```bash
go test ./internal/cache/... 2>&1 | head -5
```

Expected: FAIL

**Step 3: 实现 raw.go**

```go
// internal/cache/raw.go
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	gh "github.com/jimyag/commitlens/internal/github"
)

type RawData struct {
	Repo        string    `json:"repo"`
	LastUpdated time.Time `json:"last_updated"`
	PRs         []gh.PR   `json:"prs"`
}

type RawCache struct {
	dir string
}

func NewRawCache(dir string) *RawCache {
	return &RawCache{dir: dir}
}

func repoKey(repo string) string {
	return strings.ReplaceAll(repo, "/", "_")
}

func (c *RawCache) path(repo string) string {
	return filepath.Join(c.dir, repoKey(repo)+"_raw.json")
}

func (c *RawCache) Load(repo string) (*RawData, error) {
	data, err := os.ReadFile(c.path(repo))
	if os.IsNotExist(err) {
		return &RawData{Repo: repo, LastUpdated: time.Time{}, PRs: nil}, nil
	}
	if err != nil {
		return nil, err
	}
	var raw RawData
	return &raw, json.Unmarshal(data, &raw)
}

func (c *RawCache) Save(raw *RawData) error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(raw.Repo), data, 0644)
}
```

**Step 4: 实现 stats.go**

```go
// internal/cache/stats.go
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ContributorStats struct {
	Login       string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
	PRCount     int    `json:"pr_count"`
	CommitCount int    `json:"commit_count"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
}

type WeeklyEntry struct {
	TotalPRs     int            `json:"total_prs"`
	Contributors map[string]int `json:"contributors"`
}

type StatsData struct {
	Repo         string                       `json:"repo"`
	ComputedAt   time.Time                    `json:"computed_at"`
	Contributors map[string]*ContributorStats `json:"contributors"`
	Weekly       map[string]*WeeklyEntry      `json:"weekly"`
}

type StatsCache struct {
	dir string
}

func NewStatsCache(dir string) *StatsCache {
	return &StatsCache{dir: dir}
}

func (c *StatsCache) path(repo string) string {
	return filepath.Join(c.dir, repoKey(repo)+"_stats.json")
}

func (c *StatsCache) Load(repo string) (*StatsData, error) {
	data, err := os.ReadFile(c.path(repo))
	if os.IsNotExist(err) {
		return &StatsData{
			Repo:         repo,
			Contributors: make(map[string]*ContributorStats),
			Weekly:       make(map[string]*WeeklyEntry),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var stats StatsData
	return &stats, json.Unmarshal(data, &stats)
}

func (c *StatsCache) Save(stats *StatsData) error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(stats.Repo), data, 0644)
}
```

**Step 5: 跑测试**

```bash
go test ./internal/cache/... -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/cache/
git commit -m "feat: add cache package (raw + stats)"
```

---

## Task 5: stats 聚合器

**Files:**
- Create: `internal/stats/aggregator.go`
- Create: `internal/stats/aggregator_test.go`

**Step 1: 写失败测试**

```go
// internal/stats/aggregator_test.go
package stats_test

import (
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	gh "github.com/jimyag/commitlens/internal/github"
	"github.com/jimyag/commitlens/internal/stats"
)

func TestAggregate(t *testing.T) {
	now := time.Now().UTC()
	raw := &cache.RawData{
		Repo:        "jimyag/commitlens",
		LastUpdated: now,
		PRs: []gh.PR{
			{
				Number:    1,
				Author:    "jimyag",
				AvatarURL: "https://example.com/avatar",
				MergedAt:  now.Add(-24 * time.Hour),
				Additions: 100,
				Deletions: 20,
				Commits: []gh.Commit{
					{SHA: "abc", Author: "jimyag", Additions: 60, Deletions: 10},
					{SHA: "def", Author: "jimyag", Additions: 40, Deletions: 10},
				},
			},
			{
				Number:    2,
				Author:    "alice",
				AvatarURL: "https://example.com/alice",
				MergedAt:  now.Add(-48 * time.Hour),
				Additions: 50,
				Deletions: 5,
				Commits: []gh.Commit{
					{SHA: "ghi", Author: "alice", Additions: 50, Deletions: 5},
				},
			},
		},
	}

	result := stats.Aggregate(raw)

	if result.Contributors["jimyag"].PRCount != 1 {
		t.Errorf("jimyag PRCount: want 1, got %d", result.Contributors["jimyag"].PRCount)
	}
	if result.Contributors["jimyag"].CommitCount != 2 {
		t.Errorf("jimyag CommitCount: want 2, got %d", result.Contributors["jimyag"].CommitCount)
	}
	if result.Contributors["jimyag"].Additions != 100 {
		t.Errorf("jimyag Additions: want 100, got %d", result.Contributors["jimyag"].Additions)
	}
	if result.Contributors["alice"].PRCount != 1 {
		t.Errorf("alice PRCount: want 1, got %d", result.Contributors["alice"].PRCount)
	}
	if len(result.Weekly) == 0 {
		t.Error("expected weekly data")
	}
}
```

**Step 2: 跑测试确认失败**

```bash
go test ./internal/stats/... 2>&1 | head -5
```

Expected: FAIL

**Step 3: 实现 aggregator.go**

```go
// internal/stats/aggregator.go
package stats

import (
	"fmt"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
)

func Aggregate(raw *cache.RawData) *cache.StatsData {
	result := &cache.StatsData{
		Repo:         raw.Repo,
		ComputedAt:   time.Now().UTC(),
		Contributors: make(map[string]*cache.ContributorStats),
		Weekly:       make(map[string]*cache.WeeklyEntry),
	}

	for _, pr := range raw.PRs {
		c := getOrCreate(result.Contributors, pr.Author, pr.AvatarURL)
		c.PRCount++
		c.CommitCount += len(pr.Commits)
		c.Additions += pr.Additions
		c.Deletions += pr.Deletions

		week := weekKey(pr.MergedAt)
		w := getOrCreateWeek(result.Weekly, week)
		w.TotalPRs++
		w.Contributors[pr.Author]++
	}

	return result
}

func getOrCreate(m map[string]*cache.ContributorStats, login, avatarURL string) *cache.ContributorStats {
	if v, ok := m[login]; ok {
		return v
	}
	v := &cache.ContributorStats{Login: login, AvatarURL: avatarURL}
	m[login] = v
	return v
}

func getOrCreateWeek(m map[string]*cache.WeeklyEntry, key string) *cache.WeeklyEntry {
	if v, ok := m[key]; ok {
		return v
	}
	v := &cache.WeeklyEntry{Contributors: make(map[string]int)}
	m[key] = v
	return v
}

func weekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func MonthKey(t time.Time) string {
	return t.Format("2006-01")
}

func QuarterKey(t time.Time) string {
	q := (int(t.Month())-1)/3 + 1
	return fmt.Sprintf("%d-Q%d", t.Year(), q)
}

func YearKey(t time.Time) string {
	return fmt.Sprintf("%d", t.Year())
}
```

**Step 4: 跑测试**

```bash
go test ./internal/stats/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/stats/
git commit -m "feat: add stats aggregator"
```

---

## Task 6: sync 同步器

**Files:**
- Create: `internal/sync/syncer.go`

**Step 1: 实现 syncer.go**

```go
// internal/sync/syncer.go
package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	gh "github.com/jimyag/commitlens/internal/github"
	"github.com/jimyag/commitlens/internal/stats"
)

type Syncer struct {
	client     *gh.Client
	rawCache   *cache.RawCache
	statsCache *cache.StatsCache
}

func New(client *gh.Client, rawCache *cache.RawCache, statsCache *cache.StatsCache) *Syncer {
	return &Syncer{client: client, rawCache: rawCache, statsCache: statsCache}
}

type Progress struct {
	Repo    string
	Current int
	Total   int
	Err     error
}

func (s *Syncer) SyncAll(ctx context.Context, repos []string, progress chan<- Progress) {
	for i, repo := range repos {
		if progress != nil {
			progress <- Progress{Repo: repo, Current: i + 1, Total: len(repos)}
		}
		if err := s.SyncRepo(ctx, repo); err != nil {
			if progress != nil {
				progress <- Progress{Repo: repo, Current: i + 1, Total: len(repos), Err: err}
			}
		}
	}
	if progress != nil {
		close(progress)
	}
}

func (s *Syncer) SyncRepo(ctx context.Context, repo string) error {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, name := parts[0], parts[1]

	raw, err := s.rawCache.Load(repo)
	if err != nil {
		return fmt.Errorf("load raw cache: %w", err)
	}

	since := raw.LastUpdated
	if since.IsZero() {
		since = time.Now().AddDate(-1, 0, 0) // default: last 1 year
	}

	newPRs, err := s.client.GetMergedPRsSince(ctx, owner, name, since)
	if err != nil {
		return fmt.Errorf("fetch PRs: %w", err)
	}

	raw.PRs = append(newPRs, raw.PRs...)
	raw.LastUpdated = time.Now().UTC()
	if err := s.rawCache.Save(raw); err != nil {
		return fmt.Errorf("save raw cache: %w", err)
	}

	computed := stats.Aggregate(raw)
	if err := s.statsCache.Save(computed); err != nil {
		return fmt.Errorf("save stats cache: %w", err)
	}
	return nil
}
```

**Step 2: 验证编译**

```bash
go build ./...
```

Expected: 无报错

**Step 3: Commit**

```bash
git add internal/sync/
git commit -m "feat: add sync orchestrator"
```

---

## Task 7: TUI 框架（app + keys）

**Files:**
- Create: `internal/tui/keys.go`
- Create: `internal/tui/app.go`

**Step 1: 实现 keys.go**

```go
// internal/tui/keys.go
package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab      key.Binding
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Refresh  key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch view")),
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev granularity")),
	Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next granularity")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
```

**Step 2: 实现 app.go**

```go
// internal/tui/app.go
package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
	isync "github.com/jimyag/commitlens/internal/sync"
)

type viewMode int

const (
	viewSummary viewMode = iota
	viewRepo
	viewTrend
)

type syncDoneMsg struct{ err error }

type App struct {
	mode       viewMode
	stats      []*cache.StatsData
	repoNames  []string
	selectedRepo int
	selectedContributor int
	granularity int // 0=week 1=month 2=quarter 3=year
	syncer     *isync.Syncer
	syncing    bool
	width      int
	height     int
	err        error
}

var granularityLabels = []string{"周", "月", "季度", "年"}

func New(syncer *isync.Syncer, stats []*cache.StatsData, repos []string) *App {
	return &App{
		syncer:    syncer,
		stats:     stats,
		repoNames: repos,
	}
}

func (a *App) Init() (tea.Model, tea.Cmd) {
	return a, nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			a.mode = (a.mode + 1) % 3
		case "r":
			if !a.syncing {
				a.syncing = true
				return a, a.doSync()
			}
		case "up", "k":
			if a.mode == viewRepo && a.selectedRepo > 0 {
				a.selectedRepo--
			}
			if a.mode == viewTrend && a.selectedContributor > 0 {
				a.selectedContributor--
			}
		case "down", "j":
			if a.mode == viewRepo && a.selectedRepo < len(a.repoNames)-1 {
				a.selectedRepo++
			}
			if a.mode == viewTrend {
				maxC := a.maxContributors()
				if a.selectedContributor < maxC-1 {
					a.selectedContributor++
				}
			}
		case "left", "h":
			if a.granularity > 0 {
				a.granularity--
			}
		case "right", "l":
			if a.granularity < 3 {
				a.granularity++
			}
		}
	case syncDoneMsg:
		a.syncing = false
		a.err = msg.err
	}
	return a, nil
}

func (a *App) doSync() tea.Cmd {
	return func() tea.Msg {
		// Sync is handled externally; TUI triggers re-load via message
		return syncDoneMsg{err: nil}
	}
}

func (a *App) maxContributors() int {
	total := 0
	for _, s := range a.stats {
		total += len(s.Contributors)
	}
	return total
}

func (a *App) View() tea.View {
	header := a.renderHeader()
	var body string
	switch a.mode {
	case viewSummary:
		body = a.renderSummary()
	case viewRepo:
		body = a.renderRepo()
	case viewTrend:
		body = a.renderTrend()
	}
	status := a.renderStatus()
	return tea.NewView(fmt.Sprintf("%s\n%s\n%s", header, body, status))
}

func (a *App) renderHeader() string {
	tabs := []string{"[汇总]", "[单仓库]", "[趋势]"}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	line := ""
	for i, t := range tabs {
		if viewMode(i) == a.mode {
			line += style.Render(t) + " "
		} else {
			line += t + " "
		}
	}
	return "CommitLens  " + line + "         r:刷新  q:退出"
}

func (a *App) renderStatus() string {
	if a.syncing {
		return "状态: 同步中..."
	}
	if a.err != nil {
		return fmt.Sprintf("错误: %v", a.err)
	}
	return ""
}

func (a *App) renderSummary() string  { return renderSummaryView(a) }
func (a *App) renderRepo() string     { return renderRepoView(a) }
func (a *App) renderTrend() string    { return renderTrendView(a) }

func Run(syncer *isync.Syncer, stats []*cache.StatsData, repos []string) error {
	app := New(syncer, stats, repos)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI app scaffold"
```

---

## Task 8: TUI 三个视图

**Files:**
- Create: `internal/tui/views/summary.go`
- Create: `internal/tui/views/repo.go`
- Create: `internal/tui/views/trend.go`

注意：这三个文件内的函数被 `app.go` 的 `renderSummaryView`/`renderRepoView`/`renderTrendView` 调用，放在同一个 `tui` package 内。

**Step 1: 创建 internal/tui/summary.go**

```go
// internal/tui/summary.go
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
)

func renderSummaryView(a *App) string {
	// 合并所有仓库贡献者数据
	merged := make(map[string]*cache.ContributorStats)
	for _, s := range a.stats {
		for login, c := range s.Contributors {
			if existing, ok := merged[login]; ok {
				existing.PRCount += c.PRCount
				existing.CommitCount += c.CommitCount
				existing.Additions += c.Additions
				existing.Deletions += c.Deletions
			} else {
				cp := *c
				merged[login] = &cp
			}
		}
	}

	contributors := sortedContributors(merged)
	return renderContributorTable(contributors)
}

func sortedContributors(m map[string]*cache.ContributorStats) []*cache.ContributorStats {
	list := make([]*cache.ContributorStats, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].PRCount > list[j].PRCount
	})
	return list
}

func renderContributorTable(contributors []*cache.ContributorStats) string {
	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%-20s %6s %8s %8s %8s", "贡献者", "PR数", "Commit数", "新增行", "删除行"),
	)
	sep := strings.Repeat("─", 60)
	rows := []string{header, sep}
	for _, c := range contributors {
		row := fmt.Sprintf("%-20s %6d %8d %8d %8d",
			c.Login, c.PRCount, c.CommitCount, c.Additions, c.Deletions)
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}
```

**Step 2: 创建 internal/tui/repo.go**

```go
// internal/tui/repo.go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderRepoView(a *App) string {
	if len(a.repoNames) == 0 {
		return "无仓库配置"
	}

	// 左侧仓库列表
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	repoList := make([]string, len(a.repoNames))
	for i, r := range a.repoNames {
		if i == a.selectedRepo {
			repoList[i] = selectedStyle.Render("> " + r)
		} else {
			repoList[i] = "  " + r
		}
	}
	left := strings.Join(repoList, "\n")

	// 右侧贡献者表
	right := "无数据"
	if a.selectedRepo < len(a.stats) {
		s := a.stats[a.selectedRepo]
		contributors := sortedContributors(s.Contributors)
		right = fmt.Sprintf("%s\n%s", a.repoNames[a.selectedRepo], renderContributorTable(contributors))
	}

	// 简单左右拼接
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	var sb strings.Builder
	for i := 0; i < maxLines; i++ {
		l, r2 := "", ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r2 = rightLines[i]
		}
		sb.WriteString(fmt.Sprintf("%-25s  %s\n", l, r2))
	}
	return sb.String()
}
```

**Step 3: 创建 internal/tui/trend.go**

```go
// internal/tui/trend.go
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/NimbleMarkets/ntcharts/linechart/streamlinechart"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/stats"
)

func renderTrendView(a *App) string {
	granLabel := fmt.Sprintf("粒度: %s  (←→切换)", granularityLabels[a.granularity])

	// 聚合周期数据
	periodData := aggregatePeriods(a)
	periods := sortedKeys(periodData)

	if len(periods) == 0 {
		return granLabel + "\n\n无数据"
	}

	// 全仓库折线图数据
	totalValues := make([]float64, len(periods))
	for i, p := range periods {
		totalValues[i] = float64(periodData[p].total)
	}

	chartWidth := 60
	if a.width > 20 {
		chartWidth = a.width - 20
	}
	chart := streamlinechart.New(chartWidth, 8)
	for _, v := range totalValues {
		chart.Push(v)
	}
	chart.Draw()
	totalChart := chart.View()

	// 贡献者列表
	allLogins := allContributorLogins(a.stats)
	sort.Strings(allLogins)

	var selectedLogin string
	if a.selectedContributor < len(allLogins) {
		selectedLogin = allLogins[a.selectedContributor]
	}

	loginList := make([]string, len(allLogins))
	sel := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	for i, l := range allLogins {
		if i == a.selectedContributor {
			loginList[i] = sel.Render("> " + l)
		} else {
			loginList[i] = "  " + l
		}
	}

	// 个人折线图
	var personChart string
	if selectedLogin != "" {
		personValues := make([]float64, len(periods))
		for i, p := range periods {
			personValues[i] = float64(periodData[p].byContributor[selectedLogin])
		}
		pc := streamlinechart.New(chartWidth, 6)
		for _, v := range personValues {
			pc.Push(v)
		}
		pc.Draw()
		personChart = fmt.Sprintf("%s PR 趋势\n%s", selectedLogin, pc.View())
	}

	return strings.Join([]string{
		granLabel,
		"",
		"全仓库合并 PR 趋势",
		totalChart,
		"",
		"按贡献者 ↑↓ 选择",
		strings.Join(loginList, "\n"),
		"",
		personChart,
	}, "\n")
}

type periodEntry struct {
	total         int
	byContributor map[string]int
}

func aggregatePeriods(a *App) map[string]*periodEntry {
	result := make(map[string]*periodEntry)
	for _, s := range a.stats {
		for weekKey, w := range s.Weekly {
			period := toPeriodKey(weekKey, a.granularity)
			if _, ok := result[period]; !ok {
				result[period] = &periodEntry{byContributor: make(map[string]int)}
			}
			result[period].total += w.TotalPRs
			for login, count := range w.Contributors {
				result[period].byContributor[login] += count
			}
		}
	}
	return result
}

func toPeriodKey(weekKey string, granularity int) string {
	// weekKey format: "2026-W15" - parse year/week then convert
	switch granularity {
	case 0:
		return weekKey
	case 1:
		// Convert week to month approximation
		_ = stats.MonthKey
		return weekKey[:7] // use year-W prefix as proxy; ideally parse date
	case 2:
		return weekKey[:5] + "Q" // simplified
	case 3:
		return weekKey[:4]
	}
	return weekKey
}

func sortedKeys(m map[string]*periodEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func allContributorLogins(stats []*cache.StatsData) []string {
	seen := make(map[string]struct{})
	for _, s := range stats {
		for login := range s.Contributors {
			seen[login] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for l := range seen {
		result = append(result, l)
	}
	return result
}
```

**Step 4: 验证编译**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI views (summary, repo, trend)"
```

---

## Task 9: Web 服务器

**Files:**
- Create: `internal/web/server.go`
- Create: `internal/web/api.go`

**Step 1: 实现 server.go**

```go
// internal/web/server.go
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/cache"
	isync "github.com/jimyag/commitlens/internal/sync"
)

type Server struct {
	engine     *gin.Engine
	syncer     *isync.Syncer
	stats      []*cache.StatsData
	repos      []string
	frontendFS http.FileSystem
}

func New(assets embed.FS, syncer *isync.Syncer, stats []*cache.StatsData, repos []string) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		engine: gin.New(),
		syncer: syncer,
		stats:  stats,
		repos:  repos,
	}
	s.mountFrontend(assets)
	s.registerAPI()
	return s
}

func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}

func (s *Server) mountFrontend(assets embed.FS) {
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return
	}
	s.frontendFS = http.FS(sub)

	s.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := s.frontendFS.Open(path)
		if err != nil {
			f, _ = s.frontendFS.Open("index.html")
		}
		if f != nil {
			defer f.Close()
			info, _ := f.Stat()
			http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
		}
	})
}
```

**Step 2: 实现 api.go**

```go
// internal/web/api.go
package web

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerAPI() {
	api := s.engine.Group("/api")
	api.GET("/stats", s.handleGetStats)
	api.GET("/repos", s.handleGetRepos)
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
	c.JSON(http.StatusOK, gin.H{"repos": s.repos})
}

func (s *Server) handleSync(c *gin.Context) {
	go func() {
		_ = s.syncer.SyncRepo(context.Background(), c.Query("repo"))
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started"})
}
```

**Step 3: 验证编译**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/web/
git commit -m "feat: add web server with API"
```

---

## Task 10: React 前端初始化

**Files:**
- Create: `frontend/` 目录及 React 项目

**Step 1: 初始化 React + Vite + TypeScript**

```bash
cd /Users/jimyag/src/github/jimyag/commitlens
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
```

**Step 2: 安装依赖**

```bash
npm install echarts echarts-for-react axios
npm install -D @types/react @types/react-dom
```

**Step 3: 配置 vite.config.ts（API 代理用于开发）**

```typescript
// frontend/vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
})
```

**Step 4: 验证前端构建**

```bash
cd frontend && npm run build
```

Expected: dist/ 目录生成

**Step 5: Commit**

```bash
cd ..
git add frontend/
git commit -m "feat: init react frontend"
```

---

## Task 11: React 前端核心组件

**Files:**
- Create: `frontend/src/api.ts`
- Create: `frontend/src/components/ContributorTable.tsx`
- Create: `frontend/src/components/TrendChart.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: 实现 api.ts**

```typescript
// frontend/src/api.ts
import axios from 'axios'

export interface ContributorStats {
  login: string
  avatar_url: string
  pr_count: number
  commit_count: number
  additions: number
  deletions: number
}

export interface WeeklyEntry {
  total_prs: number
  contributors: Record<string, number>
}

export interface StatsData {
  repo: string
  computed_at: string
  contributors: Record<string, ContributorStats>
  weekly: Record<string, WeeklyEntry>
}

export const api = {
  getRepos: () => axios.get<{ repos: string[] }>('/api/repos'),
  getStats: (repo?: string) =>
    axios.get<{ stats: StatsData[] } | StatsData>('/api/stats', { params: repo ? { repo } : {} }),
  sync: (repo?: string) =>
    axios.post('/api/sync', null, { params: repo ? { repo } : {} }),
}
```

**Step 2: 实现 ContributorTable.tsx**

```typescript
// frontend/src/components/ContributorTable.tsx
import { ContributorStats } from '../api'

interface Props {
  contributors: Record<string, ContributorStats>
}

export function ContributorTable({ contributors }: Props) {
  const sorted = Object.values(contributors).sort((a, b) => b.pr_count - a.pr_count)

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
      <thead>
        <tr style={{ borderBottom: '2px solid #333' }}>
          <th style={{ textAlign: 'left', padding: '8px' }}>贡献者</th>
          <th style={{ textAlign: 'right', padding: '8px' }}>PR数</th>
          <th style={{ textAlign: 'right', padding: '8px' }}>Commit数</th>
          <th style={{ textAlign: 'right', padding: '8px' }}>新增行</th>
          <th style={{ textAlign: 'right', padding: '8px' }}>删除行</th>
        </tr>
      </thead>
      <tbody>
        {sorted.map(c => (
          <tr key={c.login} style={{ borderBottom: '1px solid #eee' }}>
            <td style={{ padding: '8px', display: 'flex', alignItems: 'center', gap: 8 }}>
              <img src={c.avatar_url} alt={c.login} width={24} height={24} style={{ borderRadius: '50%' }} />
              {c.login}
            </td>
            <td style={{ textAlign: 'right', padding: '8px' }}>{c.pr_count}</td>
            <td style={{ textAlign: 'right', padding: '8px' }}>{c.commit_count}</td>
            <td style={{ textAlign: 'right', padding: '8px', color: '#22c55e' }}>+{c.additions}</td>
            <td style={{ textAlign: 'right', padding: '8px', color: '#ef4444' }}>-{c.deletions}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
```

**Step 3: 实现 TrendChart.tsx**

```typescript
// frontend/src/components/TrendChart.tsx
import ReactECharts from 'echarts-for-react'
import { WeeklyEntry } from '../api'

type Granularity = 'week' | 'month' | 'quarter' | 'year'

interface Props {
  weekly: Record<string, WeeklyEntry>
  granularity: Granularity
  selectedLogin?: string
}

function toPeriodKey(weekKey: string, gran: Granularity): string {
  // weekKey: "2026-W15"
  const [yearStr, wStr] = weekKey.split('-W')
  const year = parseInt(yearStr)
  const week = parseInt(wStr)
  if (gran === 'week') return weekKey
  if (gran === 'year') return yearStr
  // Approximate month from week number
  const month = Math.ceil(week / 4.33)
  if (gran === 'month') return `${year}-${String(month).padStart(2, '0')}`
  const quarter = Math.ceil(month / 3)
  return `${year}-Q${quarter}`
}

export function TrendChart({ weekly, granularity, selectedLogin }: Props) {
  // Aggregate by period
  const periodMap: Record<string, { total: number; byLogin: Record<string, number> }> = {}
  for (const [weekKey, entry] of Object.entries(weekly)) {
    const period = toPeriodKey(weekKey, granularity)
    if (!periodMap[period]) periodMap[period] = { total: 0, byLogin: {} }
    periodMap[period].total += entry.total_prs
    for (const [login, count] of Object.entries(entry.contributors)) {
      periodMap[period].byLogin[login] = (periodMap[period].byLogin[login] ?? 0) + count
    }
  }

  const periods = Object.keys(periodMap).sort()
  const totalData = periods.map(p => periodMap[p].total)
  const personData = selectedLogin
    ? periods.map(p => periodMap[p].byLogin[selectedLogin] ?? 0)
    : []

  const series = [
    {
      name: '全仓库',
      type: 'line',
      data: totalData,
      smooth: true,
      areaStyle: { opacity: 0.3 },
      lineStyle: { width: 2 },
    },
    ...(selectedLogin ? [{
      name: selectedLogin,
      type: 'line',
      data: personData,
      smooth: true,
      areaStyle: { opacity: 0.2 },
      lineStyle: { width: 2, type: 'dashed' },
    }] : []),
  ]

  const option = {
    tooltip: { trigger: 'axis' },
    legend: { data: selectedLogin ? ['全仓库', selectedLogin] : ['全仓库'] },
    xAxis: { type: 'category', data: periods, axisLabel: { rotate: 30 } },
    yAxis: { type: 'value', name: 'PR数' },
    series,
    grid: { left: 60, right: 20, bottom: 60 },
  }

  return <ReactECharts option={option} style={{ height: 320 }} />
}
```

**Step 4: 实现 App.tsx**

```typescript
// frontend/src/App.tsx
import { useEffect, useState } from 'react'
import { api, StatsData, ContributorStats } from './api'
import { ContributorTable } from './components/ContributorTable'
import { TrendChart } from './components/TrendChart'

type Granularity = 'week' | 'month' | 'quarter' | 'year'

export default function App() {
  const [repos, setRepos] = useState<string[]>([])
  const [allStats, setAllStats] = useState<StatsData[]>([])
  const [selectedRepo, setSelectedRepo] = useState<string>('')
  const [granularity, setGranularity] = useState<Granularity>('week')
  const [selectedLogin, setSelectedLogin] = useState<string>('')
  const [syncing, setSyncing] = useState(false)

  useEffect(() => {
    api.getRepos().then(r => setRepos(r.data.repos))
    api.getStats().then(r => {
      const data = r.data as { stats: StatsData[] }
      setAllStats(data.stats ?? [])
    })
  }, [])

  const currentStats = selectedRepo
    ? allStats.find(s => s.repo === selectedRepo)
    : allStats[0]

  const mergedContributors: Record<string, ContributorStats> = {}
  for (const s of (selectedRepo ? (currentStats ? [currentStats] : []) : allStats)) {
    for (const [login, c] of Object.entries(s?.contributors ?? {})) {
      if (!mergedContributors[login]) {
        mergedContributors[login] = { ...c }
      } else {
        mergedContributors[login].pr_count += c.pr_count
        mergedContributors[login].commit_count += c.commit_count
        mergedContributors[login].additions += c.additions
        mergedContributors[login].deletions += c.deletions
      }
    }
  }

  const mergedWeekly = (selectedRepo ? [currentStats] : allStats).reduce((acc, s) => {
    if (!s?.weekly) return acc
    for (const [k, v] of Object.entries(s.weekly)) {
      if (!acc[k]) acc[k] = { total_prs: 0, contributors: {} }
      acc[k].total_prs += v.total_prs
      for (const [login, count] of Object.entries(v.contributors)) {
        acc[k].contributors[login] = (acc[k].contributors[login] ?? 0) + count
      }
    }
    return acc
  }, {} as Record<string, any>)

  const handleSync = async () => {
    setSyncing(true)
    await api.sync(selectedRepo || undefined)
    setTimeout(() => setSyncing(false), 2000)
  }

  return (
    <div style={{ maxWidth: 1200, margin: '0 auto', padding: 24, fontFamily: 'sans-serif' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 24 }}>
        <h1 style={{ margin: 0 }}>CommitLens</h1>
        <select value={selectedRepo} onChange={e => setSelectedRepo(e.target.value)}>
          <option value="">全部仓库</option>
          {repos.map(r => <option key={r} value={r}>{r}</option>)}
        </select>
        <select value={granularity} onChange={e => setGranularity(e.target.value as Granularity)}>
          <option value="week">周</option>
          <option value="month">月</option>
          <option value="quarter">季度</option>
          <option value="year">年</option>
        </select>
        <button onClick={handleSync} disabled={syncing}>
          {syncing ? '同步中...' : '刷新'}
        </button>
      </div>

      <h2>贡献者排行</h2>
      <ContributorTable contributors={mergedContributors} />

      <h2 style={{ marginTop: 32 }}>趋势图</h2>
      <div style={{ marginBottom: 12 }}>
        按贡献者：
        <select value={selectedLogin} onChange={e => setSelectedLogin(e.target.value)}>
          <option value="">不叠加个人</option>
          {Object.keys(mergedContributors).sort().map(l => (
            <option key={l} value={l}>{l}</option>
          ))}
        </select>
      </div>
      <TrendChart weekly={mergedWeekly} granularity={granularity} selectedLogin={selectedLogin || undefined} />
    </div>
  )
}
```

**Step 5: 构建前端**

```bash
cd frontend && npm run build && cd ..
```

Expected: frontend/dist/ 目录生成

**Step 6: Commit**

```bash
git add frontend/
git commit -m "feat: add react frontend with echarts"
```

---

## Task 12: 整合 main.go + cmd/root.go

**Files:**
- Modify: `cmd/root.go`

**Step 1: 更新 run 函数**

```go
// cmd/root.go - 完整版
package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	gh "github.com/jimyag/commitlens/internal/github"
	isync "github.com/jimyag/commitlens/internal/sync"
	"github.com/jimyag/commitlens/internal/tui"
	"github.com/jimyag/commitlens/internal/web"
)

var (
	globalAssets embed.FS
	webMode      bool
	webPort      int
	configFile   string
)

var rootCmd = &cobra.Command{
	Use:   "commitlens",
	Short: "GitHub contribution stats viewer",
	RunE:  run,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func ExecuteWithAssets(assets embed.FS) {
	globalAssets = assets
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().BoolVar(&webMode, "web", false, "Start web UI mode")
	rootCmd.Flags().IntVar(&webPort, "port", 8080, "Web UI port")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Config file (default: ~/.commitlens/config.yaml)")
}

func run(cmd *cobra.Command, args []string) error {
	cfgPath := configFile
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "config file not found: %s\nCreate one with:\n\nrepositories:\n  - owner: your-org\n    repo: your-repo\n", cfgPath)
			os.Exit(1)
		}
		return err
	}

	if err := os.MkdirAll(cfg.Cache.Dir, 0755); err != nil {
		return err
	}

	client := gh.NewClient(cfg.GitHub.Token)
	rawCache := cache.NewRawCache(cfg.Cache.Dir)
	statsCache := cache.NewStatsCache(cfg.Cache.Dir)
	syncer := isync.New(client, rawCache, statsCache)

	// Build repo list
	repos := make([]string, len(cfg.Repositories))
	for i, r := range cfg.Repositories {
		repos[i] = r.Owner + "/" + r.Repo
	}

	// Auto-sync on startup
	fmt.Fprintln(os.Stderr, "同步数据中...")
	for _, repo := range repos {
		if err := syncer.SyncRepo(cmd.Context(), repo); err != nil {
			fmt.Fprintf(os.Stderr, "sync %s: %v\n", repo, err)
		}
	}

	// Load stats
	var allStats []*cache.StatsData
	for _, repo := range repos {
		s, err := statsCache.Load(repo)
		if err != nil {
			continue
		}
		allStats = append(allStats, s)
	}

	if webMode {
		port := webPort
		if cfg.Web.Port != 8080 {
			port = cfg.Web.Port
		}
		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("CommitLens Web UI: http://localhost%s\n", addr)
		srv := web.New(globalAssets, syncer, allStats, repos)
		return srv.Run(addr)
	}

	return tui.Run(syncer, allStats, repos)
}

func init() {
	_ = filepath.Abs // ensure import used
}
```

**Step 2: 验证完整编译**

```bash
go build ./...
```

Expected: 无报错（注意：embed 的 frontend/dist 需要先 build）

**Step 3: 端到端测试**

先确保 frontend/dist 存在：
```bash
cd frontend && npm run build && cd ..
```

然后：
```bash
go build -o commitlens .
./commitlens --help
```

Expected: 显示帮助信息

**Step 4: Commit**

```bash
git add cmd/ main.go
git commit -m "feat: wire up full app (TUI + web mode)"
```

---

## Task 13: 配置示例文件 + README

**Files:**
- Create: `config.example.yaml`
- Create: `Makefile`

**Step 1: 创建 config.example.yaml**

```yaml
github:
  token: ""  # 留空则自动使用 gh auth token

repositories:
  - owner: jimyag
    repo: commitlens
  - owner: jimyag
    repo: jvp

cache:
  dir: ~/.commitlens/cache

web:
  port: 8080
```

**Step 2: 创建 Makefile**

```makefile
.PHONY: build frontend clean

frontend:
	cd frontend && npm run build

build: frontend
	go build -o commitlens .

clean:
	rm -f commitlens
	rm -rf frontend/dist
```

**Step 3: 验证 make build**

```bash
make build
```

Expected: 生成 commitlens 二进制

**Step 4: Commit**

```bash
git add config.example.yaml Makefile
git commit -m "chore: add makefile and example config"
```

---

## 验证清单

- [ ] `go test ./...` 全部通过
- [ ] `make build` 成功生成二进制
- [ ] `./commitlens --help` 输出帮助
- [ ] 配置真实仓库后 `./commitlens` 进入 TUI，显示贡献者数据
- [ ] `./commitlens --web` 启动后浏览器可访问，折线图正常渲染
- [ ] 手动触发 `r` / WebUI 刷新按钮，增量同步正常
- [ ] 关闭再重开，缓存数据复用，不重新全量拉取

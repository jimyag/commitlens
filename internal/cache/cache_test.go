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

func TestRawCache_Load_NotExist(t *testing.T) {
	dir := t.TempDir()
	rc := cache.NewRawCache(dir)

	raw, err := rc.Load("nonexistent/repo")
	if err != nil {
		t.Fatalf("Load() should not error on missing file, got: %v", err)
	}
	if raw.LastUpdated.IsZero() != true {
		t.Error("expected zero LastUpdated for missing cache")
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
		Weekly: make(map[string]*cache.WeeklyEntry),
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

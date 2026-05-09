package cache_test

import (
	"os"
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/git"
)

func TestRawCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "commitlens-raw-cache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	c := cache.NewRawCache(tmpDir)
	repo := "jimyag/commitlens"
	now := time.Now().UTC()
	raw := &cache.RawData{
		Repo:        repo,
		LastUpdated: now,
		Commits: []git.Commit{
			{SHA: "abc", Author: "jimyag", Message: "feat: something", Date: now},
		},
	}

	if err := c.Save(raw); err != nil {
		t.Fatal(err)
	}

	loaded, err := c.Load(repo)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Repo != repo {
		t.Errorf("expected repo %s, got %s", repo, loaded.Repo)
	}
	if len(loaded.Commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(loaded.Commits))
	}
}

func TestStatsCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "commitlens-stats-cache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	c := cache.NewStatsCache(tmpDir)
	repo := "jimyag/commitlens"
	stats := &cache.StatsData{
		Repo: repo,
		Contributors: map[string]*cache.ContributorStats{
			"jimyag": {Login: "jimyag", CommitCount: 5, Additions: 500, Deletions: 100},
		},
	}

	if err := c.Save(stats); err != nil {
		t.Fatal(err)
	}

	loaded, err := c.Load(repo)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Repo != repo {
		t.Errorf("expected repo %s, got %s", repo, loaded.Repo)
	}
	if loaded.Contributors["jimyag"].CommitCount != 5 {
		t.Errorf("expected CommitCount 5, got %d", loaded.Contributors["jimyag"].CommitCount)
	}
}

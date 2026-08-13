package cache_test

import (
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/git"
)

func TestRawCache(t *testing.T) {
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

	c := cache.NewStatsCache(tmpDir)
	repo := "jimyag/commitlens"
	stats := &cache.StatsData{
		Repo: repo,
		Contributors: map[string]*cache.ContributorStats{
			"jimyag": {Login: "jimyag", CommitCount: 5, Additions: 500, Deletions: 100},
		},
		RepositorySize: map[string]map[string]cache.RepositorySizePoint{
			"year": {"2026": {Lines: 1234, Commit: "abc", Date: time.Now().UTC()}},
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
	if loaded.RepositorySize["year"]["2026"].Lines != 1234 {
		t.Errorf("expected repository size 1234, got %d", loaded.RepositorySize["year"]["2026"].Lines)
	}
}

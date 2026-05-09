package stats_test

import (
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/git"
	"github.com/jimyag/commitlens/internal/stats"
)

func TestAggregate(t *testing.T) {
	now := time.Now().UTC()
	raw := &cache.RawData{
		Repo:        "jimyag/commitlens",
		LastUpdated: now,
		Commits: []git.Commit{
			{
				SHA:          "abc",
				Author:       "jimyag",
				Participants: []string{"jimyag"},
				Date:         now.Add(-24 * time.Hour),
				Additions:    60,
				Deletions:    10,
			},
			{
				SHA:          "def",
				Author:       "jimyag",
				Participants: []string{"jimyag"},
				Date:         now.Add(-20 * time.Hour),
				Additions:    40,
				Deletions:    10,
			},
			{
				SHA:          "ghi",
				Author:       "alice",
				Participants: []string{"alice"},
				Date:         now.Add(-48 * time.Hour),
				Additions:    50,
				Deletions:    5,
			},
		},
	}

	result := stats.Aggregate(raw, nil)

	if result.Contributors["jimyag"].CommitCount != 2 {
		t.Errorf("jimyag CommitCount: want 2, got %d", result.Contributors["jimyag"].CommitCount)
	}
	if result.Contributors["jimyag"].Additions != 100 {
		t.Errorf("jimyag Additions: want 100, got %d", result.Contributors["jimyag"].Additions)
	}
	if result.Contributors["alice"].CommitCount != 1 {
		t.Errorf("alice CommitCount: want 1, got %d", result.Contributors["alice"].CommitCount)
	}
	if len(result.Weekly) == 0 {
		t.Error("expected weekly data")
	}
}

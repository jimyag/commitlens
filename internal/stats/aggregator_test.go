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

package stats

import (
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
)

func TestBuildComparisonsAlignsWeekWithinPreviousMonths(t *testing.T) {
	location := time.UTC
	data := []*cache.StatsData{{
		Repo: "sample/alpha",
		Daily: map[string]*cache.WeeklyEntry{
			"2026-04-13": dailyCommits(6),
			"2026-05-11": dailyCommits(5),
			"2026-06-08": dailyCommits(4),
			"2026-07-06": dailyCommits(2),
			"2026-07-07": dailyCommits(2),
			"2026-07-08": dailyCommits(2),
			"2026-07-13": dailyCommits(1),
			"2026-07-14": dailyCommits(1),
			"2026-07-15": dailyCommits(1),
		},
	}}

	result := BuildComparisons(data, ComparisonOptions{
		Granularity: GranularityWeek,
		Location:    location,
		Now:         time.Date(2026, time.July, 15, 12, 0, 0, 0, location),
	})
	comparison := result.Comparisons["2026-W29"]

	if comparison.Anchor.Metrics.Commits != 3 {
		t.Fatalf("anchor commits = %d, want 3", comparison.Anchor.Metrics.Commits)
	}
	if comparison.Previous.Metrics.Commits != 6 {
		t.Fatalf("previous commits = %d, want 6", comparison.Previous.Metrics.Commits)
	}
	wantHistory := []int{4, 5, 6}
	for i, want := range wantHistory {
		if got := comparison.Historical[i].Metrics.Commits; got != want {
			t.Fatalf("historical[%d] commits = %d, want %d", i, got, want)
		}
	}
}

func TestBuildComparisonsUsesPreviousAndSameCalendarPeriods(t *testing.T) {
	data := []*cache.StatsData{{
		Repo: "sample/alpha",
		Daily: map[string]*cache.WeeklyEntry{
			"2024-01-10": dailyCommits(2),
			"2025-01-10": dailyCommits(3),
			"2025-12-10": dailyCommits(4),
			"2026-01-10": dailyCommits(5),
		},
	}}

	result := BuildComparisons(data, ComparisonOptions{
		Granularity: GranularityMonth,
		Location:    time.UTC,
		Now:         time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC),
	})
	comparison := result.Comparisons["2026-01"]

	if comparison.Anchor.Metrics.Commits != 5 || comparison.Previous.Metrics.Commits != 4 {
		t.Fatalf("unexpected anchor/previous values: %#v", comparison)
	}
	if comparison.Historical[0].Key != "2025-01" || comparison.Historical[0].Metrics.Commits != 3 {
		t.Fatalf("unexpected first historical period: %#v", comparison.Historical[0])
	}
	if comparison.Historical[1].Key != "2024-01" || comparison.Historical[1].Metrics.Commits != 2 {
		t.Fatalf("unexpected second historical period: %#v", comparison.Historical[1])
	}
}

func TestBuildComparisonsFiltersContributors(t *testing.T) {
	data := []*cache.StatsData{{
		Repo: "sample/alpha",
		Daily: map[string]*cache.WeeklyEntry{
			"2026-07-01": {
				TotalCommits: 3,
				Contributors: map[string]*cache.ContributorWeeklyStats{
					"alice": {Commits: 2, Additions: 20, Deletions: 4},
					"bob":   {Commits: 1, Additions: 5, Deletions: 1},
				},
			},
		},
	}}

	result := BuildComparisons(data, ComparisonOptions{
		Granularity: GranularityMonth,
		Logins:      []string{"alice"},
		Location:    time.UTC,
		Now:         time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
	})
	metrics := result.Comparisons["2026-07"].Anchor.Metrics
	if metrics.Commits != 2 || metrics.Additions != 20 || metrics.Deletions != 4 {
		t.Fatalf("unexpected contributor metrics: %#v", metrics)
	}
}

func dailyCommits(commits int) *cache.WeeklyEntry {
	return &cache.WeeklyEntry{
		TotalCommits: commits,
		Contributors: make(map[string]*cache.ContributorWeeklyStats),
	}
}

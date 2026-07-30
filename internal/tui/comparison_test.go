package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/locale"
)

func TestTrendViewsRenderPeriodComparison(t *testing.T) {
	locale.Init("en")
	statsData := []*cache.StatsData{{
		Repo: "sample/alpha",
		Daily: map[string]*cache.WeeklyEntry{
			"2026-06-08": comparisonDaily(2, 20, 4),
			"2026-07-06": comparisonDaily(3, 30, 6),
			"2026-07-13": comparisonDaily(4, 40, 8),
		},
		Weekly: map[string]*cache.WeeklyEntry{
			"2026-W24": comparisonDaily(2, 20, 4),
			"2026-W28": comparisonDaily(3, 30, 6),
			"2026-W29": comparisonDaily(4, 40, 8),
		},
		Contributors: map[string]*cache.ContributorStats{},
	}}
	app := New(
		nil,
		statsData,
		[]config.Repository{{Owner: "sample", Repo: "alpha"}},
		nil,
		nil,
		time.UTC,
	)
	defer app.zones.Close()
	app.width = 100
	app.globalGranularity = 0

	commitView := renderTrendView(app)
	if !strings.Contains(commitView, "Period comparison") || !strings.Contains(commitView, "C  2026-W29") {
		t.Fatalf("commit comparison chart missing:\n%s", commitView)
	}
	if strings.Contains(commitView, "%s") || !strings.Contains(commitView, "scope: all  gran: [week]") {
		t.Fatalf("commit trend hint is not formatted:\n%s", commitView)
	}

	linesView := renderLinesTrendView(app)
	if !strings.Contains(linesView, "Period comparison") || !strings.Contains(linesView, "+") {
		t.Fatalf("lines comparison chart missing:\n%s", linesView)
	}
	if strings.Contains(linesView, "%s") || !strings.Contains(linesView, "scope: all  gran: [week]") {
		t.Fatalf("lines trend hint is not formatted:\n%s", linesView)
	}
}

func comparisonDaily(commits, additions, deletions int) *cache.WeeklyEntry {
	return &cache.WeeklyEntry{
		TotalCommits:   commits,
		TotalAdditions: additions,
		TotalDeletions: deletions,
		Contributors:   map[string]*cache.ContributorWeeklyStats{},
	}
}

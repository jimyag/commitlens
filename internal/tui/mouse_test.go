package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	zone "github.com/lrstanley/bubblezone/v2"
)

func TestMouseTabAndGranularityClicks(t *testing.T) {
	app := New(nil, nil, nil, nil, nil)
	defer app.zones.Close()
	app.width = 100
	app.height = 30

	view := app.View()
	if view.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("mouse mode = %v, want cell motion", view.MouseMode)
	}

	clickZone(t, app, tabZoneID(int(viewTrend)))
	if app.mode != viewTrend {
		t.Fatalf("mode = %v, want trend", app.mode)
	}

	app.View()
	clickZone(t, app, zoneGranularityNext)
	if app.globalGranularity != 1 {
		t.Fatalf("granularity = %d, want 1", app.globalGranularity)
	}

	app.View()
	wheelZone(t, app, zoneGranularityFilter, tea.MouseWheelDown)
	if app.globalGranularity != 2 {
		t.Fatalf("granularity after wheel = %d, want 2", app.globalGranularity)
	}
}

func TestMouseRepositoryFilterAndCommitRow(t *testing.T) {
	repos := []config.Repository{
		{Owner: "example", Repo: "one"},
		{Owner: "example", Repo: "two"},
	}
	statsData := []*cache.StatsData{
		{Repo: "example/one", Contributors: map[string]*cache.ContributorStats{}},
		{Repo: "example/two", Contributors: map[string]*cache.ContributorStats{}},
	}
	app := New(nil, statsData, repos, nil, nil)
	defer app.zones.Close()
	app.width = 120
	app.height = 30

	app.View()
	clickZone(t, app, zoneRepoFilter)
	if !app.globalRepoExpanded {
		t.Fatal("repository filter did not expand")
	}

	app.View()
	clickZone(t, app, repoOptionZoneID(0))
	if _, selected := app.globalRepoMulti[0]; selected {
		t.Fatal("repository click did not toggle the first repository")
	}
	if _, selected := app.globalRepoMulti[1]; !selected {
		t.Fatal("repository click changed the second repository")
	}

	app.mode = viewCommitList
	app.globalRepoExpanded = false
	app.commitList = []commitItem{{SHA: "one"}, {SHA: "two"}}
	app.View()
	clickZone(t, app, commitRowZoneID(1))
	if app.commitListCursor != 1 || app.commitListFocus != 1 {
		t.Fatalf("commit selection = cursor %d focus %d, want cursor 1 focus 1", app.commitListCursor, app.commitListFocus)
	}
}

func TestTrendPeriodIndexAt(t *testing.T) {
	tests := []struct {
		name   string
		scroll int
		x      int
		want   int
	}{
		{name: "first bar", x: 0, want: 0},
		{name: "first gap", x: 2, want: -1},
		{name: "second bar", x: 4, want: 1},
		{name: "scrolled second bar", scroll: 4, x: 0, want: 1},
		{name: "outside", x: 20, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trendPeriodIndexAt(3, 10, tt.scroll, tt.x); got != tt.want {
				t.Fatalf("trendPeriodIndexAt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func clickZone(t *testing.T, app *App, id string) {
	t.Helper()
	info := waitForZone(t, app.zones, id)
	msg := tea.MouseClickMsg(tea.Mouse{
		X:      info.StartX,
		Y:      info.StartY,
		Button: tea.MouseLeft,
	})
	app.Update(msg)
}

func wheelZone(t *testing.T, app *App, id string, button tea.MouseButton) {
	t.Helper()
	info := waitForZone(t, app.zones, id)
	msg := tea.MouseWheelMsg(tea.Mouse{
		X:      info.StartX,
		Y:      info.StartY,
		Button: button,
	})
	app.Update(msg)
}

func waitForZone(t *testing.T, manager *zone.Manager, id string) *zone.ZoneInfo {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if info := manager.Get(id); !info.IsZero() {
			return info
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("zone %q was not scanned", id)
	return nil
}

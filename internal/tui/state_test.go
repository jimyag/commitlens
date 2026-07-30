package tui

import (
	"testing"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
)

func TestNormalizeStateClampsDynamicIndices(t *testing.T) {
	statsData := []*cache.StatsData{
		{Repo: "example/one", Contributors: map[string]*cache.ContributorStats{}},
	}
	repos := []config.Repository{{Owner: "example", Repo: "one"}}
	app := New(nil, statsData, repos, nil, nil, nil)
	defer app.zones.Close()

	app.mode = 99
	app.globalFocus = 99
	app.globalGranularity = 99
	app.globalRepoMulti = map[int]struct{}{-1: {}, 0: {}, 17: {}}
	app.globalRepoCursor = 17
	app.globalLoginIdx = 17
	app.globalLoginCursor = 17
	app.repoViewSelected = 17
	app.trendPeriodCursor = 17
	app.commitListFocus = 17
	app.commitListPeriodIdx = 17
	app.commitList = []commitItem{{SHA: "one"}}
	app.commitListCursor = 17

	app.normalizeState()

	if app.mode != viewCommitList || app.globalFocus != 3 || app.globalGranularity != 3 {
		t.Fatalf("top-level state was not clamped: mode=%d focus=%d granularity=%d", app.mode, app.globalFocus, app.globalGranularity)
	}
	if app.repoViewSelected != 0 || app.globalRepoCursor != 0 {
		t.Fatalf("repository state was not clamped: selected=%d cursor=%d", app.repoViewSelected, app.globalRepoCursor)
	}
	if len(app.globalRepoMulti) != 1 {
		t.Fatalf("invalid repository selections remain: %v", app.globalRepoMulti)
	}
	if _, ok := app.globalRepoMulti[0]; !ok {
		t.Fatalf("valid repository selection was removed: %v", app.globalRepoMulti)
	}
	if app.globalLoginIdx != 0 || app.globalLoginCursor != 0 {
		t.Fatalf("login state was not clamped: index=%d cursor=%d", app.globalLoginIdx, app.globalLoginCursor)
	}
	if app.trendPeriodCursor != 0 || app.commitListPeriodIdx != 0 || app.commitListCursor != 0 || app.commitListFocus != 1 {
		t.Fatalf(
			"period/list state was not clamped: trend=%d period=%d cursor=%d focus=%d",
			app.trendPeriodCursor,
			app.commitListPeriodIdx,
			app.commitListCursor,
			app.commitListFocus,
		)
	}
}

func TestViewNormalizesInvalidIndicesBeforeRendering(t *testing.T) {
	app := New(nil, nil, nil, nil, nil, nil)
	defer app.zones.Close()
	app.mode = viewCommitList
	app.commitListPeriodIdx = 17
	app.commitListCursor = 17

	app.View()

	if app.commitListPeriodIdx != 0 || app.commitListCursor != 0 {
		t.Fatalf("view left invalid commit state: period=%d cursor=%d", app.commitListPeriodIdx, app.commitListCursor)
	}
}

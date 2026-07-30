package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jimyag/commitlens/internal/cache"
)

func TestSortedContributorsUsesLoginToBreakCommitCountTies(t *testing.T) {
	contributors := map[string]*cache.ContributorStats{
		"charlie": {Login: "charlie", CommitCount: 2, Additions: 10},
		"alice":   {Login: "alice", CommitCount: 2, Additions: 30},
		"bob":     {Login: "bob", CommitCount: 2, Additions: 20},
		"zoe":     {Login: "zoe", CommitCount: 3},
	}
	want := []string{"zoe", "alice", "bob", "charlie"}

	for range 100 {
		sorted := sortedContributors(contributors)
		got := make([]string, len(sorted))
		for i, contributor := range sorted {
			got[i] = contributor.Login
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sorted contributors = %v, want %v", got, want)
		}
	}
}

func TestRefreshCommitListResetsUnavailablePeriod(t *testing.T) {
	app := New(nil, nil, nil, nil, nil, nil)
	defer app.zones.Close()
	app.mode = viewCommitList
	app.commitListPeriodIdx = 15

	app.refreshCommitList()

	if app.commitListPeriodIdx != 0 {
		t.Fatalf("period index = %d, want 0", app.commitListPeriodIdx)
	}
	app.renderCommitList()
}

func TestSummaryContributorTableUsesWideTerminal(t *testing.T) {
	contributors := []*cache.ContributorStats{
		{Login: "alice", CommitCount: 10},
		{Login: "bob", CommitCount: 5},
	}

	wide := renderContributorTableWithWidth(contributors, 100)
	if !strings.Contains(wide, "█") || !strings.Contains(wide, "░") {
		t.Fatal("wide summary does not contain the relative commit chart")
	}

	narrow := renderContributorTableWithWidth(contributors, 60)
	if strings.Contains(narrow, "█") || strings.Contains(narrow, "░") {
		t.Fatal("narrow summary unexpectedly contains the relative commit chart")
	}
}

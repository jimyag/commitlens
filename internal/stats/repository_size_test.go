package stats

import (
	"testing"
	"time"

	gitrepo "github.com/jimyag/commitlens/internal/git"
)

func TestBuildRepositorySizeCarriesForwardPeriodEndSnapshot(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	history := []gitrepo.LineCountPoint{
		{SHA: "first", Date: time.Date(2025, 1, 3, 8, 0, 0, 0, location), Lines: 10},
		{SHA: "second", Date: time.Date(2025, 1, 20, 8, 0, 0, 0, location), Lines: 25},
	}

	result := BuildRepositorySize(history, location, time.Date(2025, 3, 15, 12, 0, 0, 0, location))
	monthly := result[string(GranularityMonth)]

	if got := monthly["2025-01"].Lines; got != 25 {
		t.Fatalf("January lines = %d, want 25", got)
	}
	if got := monthly["2025-02"].Lines; got != 25 {
		t.Fatalf("February lines = %d, want carried-forward 25", got)
	}
	if got := monthly["2025-03"].Commit; got != "second" {
		t.Fatalf("March commit = %q, want second", got)
	}
}

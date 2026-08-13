package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseLineCountHistory(t *testing.T) {
	data := []byte("COMMIT\x00aaa\x001700000000\n\n3\t0\tmain.go\n-\t-\tlogo.png\n" +
		"COMMIT\x00bbb\x001700000100\n\n2\t1\tmain.go\n0\t4\told.txt\n")

	points, err := parseLineCountHistory(data)
	if err != nil {
		t.Fatalf("parseLineCountHistory() error = %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}
	if points[0].SHA != "aaa" || points[0].Lines != 3 {
		t.Fatalf("points[0] = %+v", points[0])
	}
	if points[1].SHA != "bbb" || points[1].Lines != 0 {
		t.Fatalf("points[1] = %+v", points[1])
	}
	if want := time.Unix(1700000100, 0).UTC(); !points[1].Date.Equal(want) {
		t.Fatalf("points[1].Date = %v, want %v", points[1].Date, want)
	}
}

func TestGetLineCountHistoryUsesFirstParentTreeChanges(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init", "-b", "main")
	runGit(t, repository, "config", "user.name", "CommitLens Test")
	runGit(t, repository, "config", "user.email", "commitlens@example.com")

	writeTestFile(t, filepath.Join(repository, "main.txt"), []byte("one\n\ntwo\n"))
	writeTestFile(t, filepath.Join(repository, "image.bin"), []byte{0, 1, 2})
	runGit(t, repository, "add", ".")
	runGit(t, repository, "commit", "-m", "root")

	runGit(t, repository, "checkout", "-b", "feature")
	writeTestFile(t, filepath.Join(repository, "feature.txt"), []byte("a\nb\n"))
	runGit(t, repository, "add", "feature.txt")
	runGit(t, repository, "commit", "-m", "feature")

	runGit(t, repository, "checkout", "main")
	writeTestFile(t, filepath.Join(repository, "main.txt"), []byte("one\n\ntwo\nthree\n"))
	runGit(t, repository, "add", "main.txt")
	runGit(t, repository, "commit", "-m", "main change")
	runGit(t, repository, "merge", "--no-ff", "feature", "-m", "merge feature")

	points, err := GetLineCountHistory(context.Background(), filepath.Join(repository, ".git"))
	if err != nil {
		t.Fatalf("GetLineCountHistory() error = %v", err)
	}
	want := []int{3, 4, 6}
	if len(points) != len(want) {
		t.Fatalf("len(points) = %d, want %d: %+v", len(points), len(want), points)
	}
	for index, lines := range want {
		if points[index].Lines != lines {
			t.Fatalf("points[%d].Lines = %d, want %d", index, points[index].Lines, lines)
		}
	}
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = directory
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

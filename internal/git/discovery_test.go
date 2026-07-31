package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tmpDir := t.TempDir()
	mkdirAll := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Standard repo
	stdRepo := filepath.Join(tmpDir, "std-repo")
	mkdirAll(filepath.Join(stdRepo, ".git"))

	// 2. Bare repo
	bareRepo := filepath.Join(tmpDir, "bare-repo")
	mkdirAll(bareRepo)
	writeFile(filepath.Join(bareRepo, "config"), "")
	mkdirAll(filepath.Join(bareRepo, "refs"))
	mkdirAll(filepath.Join(bareRepo, "objects"))
	writeFile(filepath.Join(bareRepo, "HEAD"), "")

	// 3. Worktree (should be ignored)
	wtRepo := filepath.Join(tmpDir, "worktree-repo")
	mkdirAll(wtRepo)
	writeFile(filepath.Join(wtRepo, ".git"), "gitdir: ...")

	// 4. Nested repo
	nestedParent := filepath.Join(tmpDir, "nested")
	mkdirAll(nestedParent)
	nestedRepo := filepath.Join(nestedParent, "inner-repo")
	mkdirAll(filepath.Join(nestedRepo, ".git"))

	repos, err := Discover([]string{tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	expected := 3 // std, bare, nested/inner
	if len(repos) != expected {
		t.Errorf("expected %d repos, got %d", expected, len(repos))
		for _, r := range repos {
			t.Logf("Found: %s", r.LocalPath)
		}
	}

	foundStd := false
	foundBare := false
	foundNested := false
	for _, r := range repos {
		if r.LocalPath == stdRepo {
			foundStd = true
		}
		if r.LocalPath == bareRepo {
			foundBare = true
		}
		if r.LocalPath == nestedRepo {
			foundNested = true
		}
		if r.LocalPath == wtRepo {
			t.Errorf("found worktree repo %s, which should be ignored", wtRepo)
		}
	}

	if !foundStd {
		t.Error("did not find standard repo")
	}
	if !foundBare {
		t.Error("did not find bare repo")
	}
	if !foundNested {
		t.Error("did not find nested repo")
	}
}

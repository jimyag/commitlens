package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "commitlens-discovery-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Standard repo
	stdRepo := filepath.Join(tmpDir, "std-repo")
	os.MkdirAll(filepath.Join(stdRepo, ".git"), 0o755)

	// 2. Bare repo
	bareRepo := filepath.Join(tmpDir, "bare-repo")
	os.MkdirAll(bareRepo, 0o755)
	os.WriteFile(filepath.Join(bareRepo, "config"), []byte(""), 0o644)
	os.MkdirAll(filepath.Join(bareRepo, "refs"), 0o755)
	os.MkdirAll(filepath.Join(bareRepo, "objects"), 0o755)
	os.WriteFile(filepath.Join(bareRepo, "HEAD"), []byte(""), 0o644)

	// 3. Worktree (should be ignored)
	wtRepo := filepath.Join(tmpDir, "worktree-repo")
	os.MkdirAll(wtRepo, 0o755)
	os.WriteFile(filepath.Join(wtRepo, ".git"), []byte("gitdir: ..."), 0o644)

	// 4. Nested repo
	nestedParent := filepath.Join(tmpDir, "nested")
	os.MkdirAll(nestedParent, 0o755)
	nestedRepo := filepath.Join(nestedParent, "inner-repo")
	os.MkdirAll(filepath.Join(nestedRepo, ".git"), 0o755)

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

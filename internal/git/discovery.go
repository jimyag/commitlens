package git

import (
	"os"
	"path/filepath"

	"github.com/jimyag/commitlens/internal/config"
)

// Discover scans the provided root directories for Git repositories.
// It stops searching deeper once a repository is found.
// It ignores Git worktrees.
func Discover(roots []string) ([]config.Repository, error) {
	var repos []config.Repository
	for _, root := range roots {
		err := discover(root, &repos)
		if err != nil {
			return nil, err
		}
	}
	return repos, nil
}

func discover(path string, repos *[]config.Repository) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil // Ignore inaccessible directories
	}

	isRepo, stopRecursion := checkGitRepo(path, entries)
	if isRepo {
		*repos = append(*repos, config.Repository{LocalPath: path})
		return nil
	}
	if stopRecursion {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			err := discover(filepath.Join(path, entry.Name()), repos)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// checkGitRepo checks if a directory is a Git repository and if recursion should stop.
// Returns (isRepo, stopRecursion).
func checkGitRepo(path string, entries []os.DirEntry) (bool, bool) {
	hasConfig := false
	hasRefs := false
	hasObjects := false
	hasHEAD := false

	for _, entry := range entries {
		if entry.Name() == ".git" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.IsDir() {
				// Standard Git Repo
				return true, true
			} else {
				// .git is a file, likely a worktree. Stop recursion but don't count as repo.
				return false, true
			}
		}
		switch entry.Name() {
		case "config":
			hasConfig = true
		case "refs":
			hasRefs = true
		case "objects":
			hasObjects = true
		case "HEAD":
			hasHEAD = true
		}
	}

	// Check for Bare Git Repo
	if hasConfig && hasRefs && hasObjects && hasHEAD {
		return true, true
	}

	return false, false
}

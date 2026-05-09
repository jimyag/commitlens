package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jimyag/commitlens/internal/config"
)

// EnsureRepo ensures that the repository is available locally.
// If config.Repository has LocalPath set, it validates and returns it.
// If it's a remote GitHub repo, it bare-clones or fetches it into cacheDir.
func EnsureRepo(ctx context.Context, repo config.Repository, token, cacheDir string, skipFetch bool, onProgress func(string)) (string, error) {
	if repo.LocalPath != "" {
		if _, err := os.Stat(repo.LocalPath); err != nil {
			return "", fmt.Errorf("local path error: %w", err)
		}
		// ensure it's a git repo
		cmd := exec.CommandContext(ctx, "git", "--git-dir="+filepath.Join(repo.LocalPath, ".git"), "rev-parse")
		if err := cmd.Run(); err != nil {
			// Try bare repo
			cmd = exec.CommandContext(ctx, "git", "--git-dir="+repo.LocalPath, "rev-parse")
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("not a valid git repository: %s", repo.LocalPath)
			}
			return repo.LocalPath, nil
		}
		return filepath.Join(repo.LocalPath, ".git"), nil
	}

	destDir := filepath.Join(cacheDir, "repos", "github.com", repo.Owner, repo.Repo)

	url := fmt.Sprintf("https://github.com/%s/%s.git", repo.Owner, repo.Repo)
	if token != "" {
		url = fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, repo.Owner, repo.Repo)
	}

	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if onProgress != nil {
			onProgress("cloning repository...")
		}
		// Clone bare
		if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
			return "", err
		}
		cmd := exec.CommandContext(ctx, "git", "clone", "--bare", url, destDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git clone failed: %s, %w", string(out), err)
		}
	} else if !skipFetch {
		if onProgress != nil {
			onProgress("fetching updates...")
		}
		// Fetch
		cmd := exec.CommandContext(ctx, "git", "--git-dir="+destDir, "fetch", "origin", "+refs/heads/*:refs/heads/*")
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git fetch failed: %s, %w", string(out), err)
		}
	}

	// Ensure HEAD is updated to remote default branch
	// This is important for bare clones to see new commits if the default branch name changed or if HEAD was detached.
	cmd := exec.CommandContext(ctx, "git", "--git-dir="+destDir, "remote", "set-head", "origin", "-a")
	_ = cmd.Run() // non-critical if it fails

	return destDir, nil
}

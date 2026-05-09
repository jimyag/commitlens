package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/git"
	"github.com/jimyag/commitlens/internal/stats"
)

type Syncer struct {
	cfg        *config.Config
	rawCache   *cache.RawCache
	statsCache *cache.StatsCache
}

func New(cfg *config.Config, rawCache *cache.RawCache, statsCache *cache.StatsCache) *Syncer {
	return &Syncer{cfg: cfg, rawCache: rawCache, statsCache: statsCache}
}

// Progress is emitted during sync to report real-time status.
type Progress struct {
	Repo        string
	RepoIndex   int
	RepoTotal   int
	PRsFetched  int // Will be used for commits now
	PRsTotal    int
	ListPage    int
	Log         string
	Err         error
	Done        bool
}

type FetchProgress struct {
	Log string
}

func (s *Syncer) SyncAll(ctx context.Context, repos []config.Repository, progress chan<- Progress, repoWorkers int) {
	if repoWorkers <= 0 {
		repoWorkers = 3
	}

	sem := make(chan struct{}, repoWorkers)
	var wg sync.WaitGroup

	for i, repo := range repos {
		i, repo := i, repo
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			repoID := repo.ID()
			var onProg func(FetchProgress)
			if progress != nil {
				onProg = func(p FetchProgress) {
					progress <- Progress{
						Repo:       repoID,
						RepoIndex:  i + 1,
						RepoTotal:  len(repos),
						Log:        p.Log,
					}
				}
			}

			if err := s.syncRepo(ctx, repo, onProg); err != nil {
				if progress != nil {
					progress <- Progress{
						Repo:      repoID,
						RepoIndex: i + 1,
						RepoTotal: len(repos),
						Err:       err,
					}
				}
				return
			}

			if progress != nil {
				progress <- Progress{
					Repo:      repoID,
					RepoIndex: i + 1,
					RepoTotal: len(repos),
					Done:      true,
				}
			}
		}()
	}

	wg.Wait()
	if progress != nil {
		close(progress)
	}
}

func (s *Syncer) SyncRepo(ctx context.Context, repo config.Repository) error {
	return s.syncRepo(ctx, repo, nil)
}

func (s *Syncer) syncRepo(ctx context.Context, repo config.Repository, onProgress func(FetchProgress)) error {
	repoID := repo.ID()
	
	raw, err := s.rawCache.Load(repoID)
	if err != nil {
		return fmt.Errorf("load raw cache: %w", err)
	}

	// Short Cooldown: if updated within the last 30 seconds, skip git fetch.
	skipFetch := false
	if !raw.LastUpdated.IsZero() && time.Since(raw.LastUpdated) < 30*time.Second {
		skipFetch = true
	}

	if onProgress != nil {
		log := "ensuring local repository..."
		if skipFetch {
			log = "using local repository (recent cache)..."
		}
		onProgress(FetchProgress{Log: log})
	}
	
	gitDir, err := git.EnsureRepo(ctx, repo, s.cfg.GitHub.Token, s.cfg.Cache.Dir, skipFetch)
	if err != nil {
		return fmt.Errorf("ensure repo failed: %w", err)
	}

	revRange := ""
	if len(raw.Commits) > 0 {
		// Incremental: get commits from latest_sha..HEAD
		// raw.Commits is newest to oldest, so the first one is the latest.
		latestSHA := raw.Commits[0].SHA
		revRange = fmt.Sprintf("%s..HEAD", latestSHA)
	}

	if onProgress != nil {
		onProgress(FetchProgress{Log: "extracting commits..."})
	}

	newCommits, err := git.GetCommits(ctx, gitDir, revRange)
	if err != nil {
		// If revRange failed (e.g. latestSHA no longer in repo), fallback to full scan
		newCommits, err = git.GetCommits(ctx, gitDir, "")
		if err != nil {
			return fmt.Errorf("get commits failed: %w", err)
		}
		raw.Commits = nil // reset existing
	}

	if len(newCommits) > 0 {
		// prepend new commits
		raw.Commits = append(newCommits, raw.Commits...)
	}

	raw.Repo = repoID
	raw.LastUpdated = time.Now().UTC()

	if err := s.rawCache.Save(raw); err != nil {
		return fmt.Errorf("save raw cache: %w", err)
	}

	computed := stats.Aggregate(raw)
	if err := s.statsCache.Save(computed); err != nil {
		return fmt.Errorf("save stats cache: %w", err)
	}
	return nil
}

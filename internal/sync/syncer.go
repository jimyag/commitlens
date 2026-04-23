package sync

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	gh "github.com/jimyag/commitlens/internal/github"
	"github.com/jimyag/commitlens/internal/stats"
)

type Syncer struct {
	client     *gh.Client
	rawCache   *cache.RawCache
	statsCache *cache.StatsCache
}

func New(client *gh.Client, rawCache *cache.RawCache, statsCache *cache.StatsCache) *Syncer {
	return &Syncer{client: client, rawCache: rawCache, statsCache: statsCache}
}

// Progress is emitted during sync to report real-time status.
type Progress struct {
	Repo        string
	RepoIndex   int
	RepoTotal   int
	PRsFetched  int
	PRsTotal    int // -1 = still counting
	Err         error
	Done        bool
}

// SyncAll syncs all repos concurrently (up to repoWorkers at a time).
// Progress events are sent to the progress channel; the channel is closed when done.
func (s *Syncer) SyncAll(ctx context.Context, repos []string, progress chan<- Progress, repoWorkers int) {
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

			var onPR func(gh.FetchProgress)
			if progress != nil {
				onPR = func(p gh.FetchProgress) {
					progress <- Progress{
						Repo:       repo,
						RepoIndex:  i + 1,
						RepoTotal:  len(repos),
						PRsFetched: p.PRsFetched,
						PRsTotal:   p.PRsTotal,
					}
				}
			}

			if err := s.syncRepo(ctx, repo, onPR); err != nil {
				if progress != nil {
					progress <- Progress{
						Repo:      repo,
						RepoIndex: i + 1,
						RepoTotal: len(repos),
						Err:       err,
					}
				}
				return
			}

			if progress != nil {
				progress <- Progress{
					Repo:      repo,
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

// SyncRepo syncs a single repository (no progress reporting).
func (s *Syncer) SyncRepo(ctx context.Context, repo string) error {
	return s.syncRepo(ctx, repo, nil)
}

// mergePRs merges newPRs into existing, deduplicating by PR number.
// newPRs are placed first (they have fresher data).
func mergePRs(newPRs, existing []gh.PR) []gh.PR {
	seen := make(map[int]struct{}, len(newPRs))
	for _, pr := range newPRs {
		seen[pr.Number] = struct{}{}
	}
	result := make([]gh.PR, 0, len(newPRs)+len(existing))
	result = append(result, newPRs...)
	for _, pr := range existing {
		if _, ok := seen[pr.Number]; !ok {
			result = append(result, pr)
		}
	}
	return result
}

func (s *Syncer) syncRepo(ctx context.Context, repo string, onProgress func(gh.FetchProgress)) error {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, name := parts[0], parts[1]

	raw, err := s.rawCache.Load(repo)
	if err != nil {
		return fmt.Errorf("load raw cache: %w", err)
	}

	// Zero time means fetch all history (first sync).
	// Subsequent syncs use last_updated as the cutoff.
	since := raw.LastUpdated

	newPRs, err := s.client.GetMergedPRsSince(ctx, owner, name, since, onProgress)
	if err != nil {
		return fmt.Errorf("fetch PRs: %w", err)
	}

	raw.PRs = mergePRs(newPRs, raw.PRs)
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

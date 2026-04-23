package sync

import (
	"context"
	"fmt"
	"strings"
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

type Progress struct {
	Repo    string
	Current int
	Total   int
	Err     error
}

func (s *Syncer) SyncAll(ctx context.Context, repos []string, progress chan<- Progress) {
	for i, repo := range repos {
		if progress != nil {
			progress <- Progress{Repo: repo, Current: i + 1, Total: len(repos)}
		}
		if err := s.SyncRepo(ctx, repo); err != nil {
			if progress != nil {
				progress <- Progress{Repo: repo, Current: i + 1, Total: len(repos), Err: err}
			}
		}
	}
	if progress != nil {
		close(progress)
	}
}

func (s *Syncer) SyncRepo(ctx context.Context, repo string) error {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, name := parts[0], parts[1]

	raw, err := s.rawCache.Load(repo)
	if err != nil {
		return fmt.Errorf("load raw cache: %w", err)
	}

	since := raw.LastUpdated
	if since.IsZero() {
		since = time.Now().AddDate(-1, 0, 0)
	}

	newPRs, err := s.client.GetMergedPRsSince(ctx, owner, name, since)
	if err != nil {
		return fmt.Errorf("fetch PRs: %w", err)
	}

	raw.PRs = append(newPRs, raw.PRs...)
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

package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ContributorStats struct {
	Login       string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
	PRCount     int    `json:"pr_count"`
	CommitCount int    `json:"commit_count"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
}

type WeeklyEntry struct {
	TotalPRs     int            `json:"total_prs"`
	Contributors map[string]int `json:"contributors"`
}

type StatsData struct {
	Repo         string                       `json:"repo"`
	ComputedAt   time.Time                    `json:"computed_at"`
	Contributors map[string]*ContributorStats `json:"contributors"`
	Weekly       map[string]*WeeklyEntry      `json:"weekly"`
}

type StatsCache struct {
	dir string
}

func NewStatsCache(dir string) *StatsCache {
	return &StatsCache{dir: dir}
}

func (c *StatsCache) path(repo string) string {
	return filepath.Join(c.dir, repoKey(repo)+"_stats.json")
}

func (c *StatsCache) Load(repo string) (*StatsData, error) {
	data, err := os.ReadFile(c.path(repo))
	if os.IsNotExist(err) {
		return &StatsData{
			Repo:         repo,
			Contributors: make(map[string]*ContributorStats),
			Weekly:       make(map[string]*WeeklyEntry),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var stats StatsData
	return &stats, json.Unmarshal(data, &stats)
}

func (c *StatsCache) Save(stats *StatsData) error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(stats.Repo), data, 0644)
}

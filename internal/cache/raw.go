package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jimyag/commitlens/internal/git"
)

type RawData struct {
	Repo        string       `json:"repo"`
	LastUpdated time.Time    `json:"last_updated"`
	Commits     []git.Commit `json:"commits"`
}

type RawCache struct {
	dir string
}

func NewRawCache(dir string) *RawCache {
	return &RawCache{dir: dir}
}

func repoKey(repo string) string {
	return strings.ReplaceAll(repo, "/", "_")
}

func (c *RawCache) Dir() string { return c.dir }

type SyncStatus struct {
	Repo        string    `json:"repo"`
	LastLog     string    `json:"last_log"`
	LastUpdated time.Time `json:"last_updated"`
	Syncing     bool      `json:"syncing"`
	PID         int       `json:"pid"`
}

func (c *RawCache) statusPath(repo string) string {
	return filepath.Join(c.dir, repoKey(repo)+"_status.json")
}

func (c *RawCache) LoadStatus(repo string) (*SyncStatus, error) {
	data, err := os.ReadFile(c.statusPath(repo))
	if os.IsNotExist(err) {
		return &SyncStatus{Repo: repo}, nil
	}
	if err != nil {
		return nil, err
	}
	var st SyncStatus
	return &st, json.Unmarshal(data, &st)
}

func (c *RawCache) SaveStatus(st *SyncStatus) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.statusPath(st.Repo), data, 0o644)
}

func (c *RawCache) path(repo string) string {
	return filepath.Join(c.dir, repoKey(repo)+"_raw.json")
}

func (c *RawCache) Load(repo string) (*RawData, error) {
	data, err := os.ReadFile(c.path(repo))
	if os.IsNotExist(err) {
		return &RawData{Repo: repo, LastUpdated: time.Time{}, Commits: nil}, nil
	}
	if err != nil {
		return nil, err
	}
	var raw RawData
	return &raw, json.Unmarshal(data, &raw)
}

func (c *RawCache) Save(raw *RawData) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(raw.Repo), data, 0o644)
}

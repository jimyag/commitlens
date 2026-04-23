package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	gh "github.com/jimyag/commitlens/internal/github"
)

type RawData struct {
	Repo        string    `json:"repo"`
	LastUpdated time.Time `json:"last_updated"`
	PRs         []gh.PR   `json:"prs"`
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

func (c *RawCache) path(repo string) string {
	return filepath.Join(c.dir, repoKey(repo)+"_raw.json")
}

func (c *RawCache) Load(repo string) (*RawData, error) {
	data, err := os.ReadFile(c.path(repo))
	if os.IsNotExist(err) {
		return &RawData{Repo: repo, LastUpdated: time.Time{}, PRs: nil}, nil
	}
	if err != nil {
		return nil, err
	}
	var raw RawData
	return &raw, json.Unmarshal(data, &raw)
}

func (c *RawCache) Save(raw *RawData) error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(raw.Repo), data, 0644)
}

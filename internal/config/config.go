package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Repository struct {
	Owner     string `yaml:"owner"`
	Repo      string `yaml:"repo"`
	LocalPath string `yaml:"local_path"`
}

// IsOrganization reports whether this entry discovers repositories from an
// organization instead of naming one repository directly.
func (r Repository) IsOrganization() bool {
	return r.LocalPath == "" && r.Owner != "" && r.Repo == ""
}

// IsSyncable reports whether this entry identifies a concrete repository.
func (r Repository) IsSyncable() bool {
	return r.LocalPath != "" || (r.Owner != "" && r.Repo != "")
}

// ID returns a unique identifier for the repository.
func (r Repository) ID() string {
	if r.LocalPath != "" {
		return filepath.Base(r.LocalPath)
	}
	return r.Owner + "/" + r.Repo
}

type GitHub struct {
	Token string `yaml:"token"`
}

type Cache struct {
	Dir string `yaml:"dir"`
}

type Web struct {
	Port int `yaml:"port"`
}

type Reporting struct {
	Timezone string `yaml:"timezone"`
}

type Config struct {
	GitHub       GitHub              `yaml:"github"`
	Repositories []Repository        `yaml:"repositories"`
	Cache        Cache               `yaml:"cache"`
	Web          Web                 `yaml:"web"`
	Reporting    Reporting           `yaml:"reporting"`
	UserMap      map[string][]string `yaml:"userMap"`
	// DiscoveryRoots: Local directories to scan for Git repositories.
	DiscoveryRoots []string `yaml:"discoveryRoots"`
	// Language: UI language for TUI and CLI (en, zh). Empty means COMMITLENS_LANG / LANG.
	Language string `yaml:"language"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	setDefaults(cfg)

	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.Cache.Dir = expandHome(cfg.Cache.Dir)
	if cfg.Cache.Dir == "" {
		home, _ := os.UserHomeDir()
		cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	}
	if cfg.Web.Port == 0 {
		cfg.Web.Port = 8080
	}
	if _, err := cfg.ReportingLocation(); err != nil {
		return nil, err
	}
	for i, root := range cfg.DiscoveryRoots {
		cfg.DiscoveryRoots[i] = expandHome(root)
	}
	return cfg, nil
}

func (c *Config) ReportingLocation() (*time.Location, error) {
	if c == nil || c.Reporting.Timezone == "" || c.Reporting.Timezone == "Local" {
		return time.Local, nil
	}
	location, err := time.LoadLocation(c.Reporting.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid reporting timezone %q: %w", c.Reporting.Timezone, err)
	}
	return location, nil
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".commitlens", "config.yaml")
}

func setDefaults(cfg *Config) {
	home, _ := os.UserHomeDir()
	cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	cfg.Web.Port = 8080
	cfg.Reporting.Timezone = "Local"
}

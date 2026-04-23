package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Repository struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
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

type Config struct {
	GitHub       GitHub       `yaml:"github"`
	Repositories []Repository `yaml:"repositories"`
	Cache        Cache        `yaml:"cache"`
	Web          Web          `yaml:"web"`
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
	return cfg, nil
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
}

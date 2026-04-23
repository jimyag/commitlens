package config

import (
	"os"
	"path/filepath"

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

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Cache.Dir == "" {
		home, _ := os.UserHomeDir()
		cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	}
	if cfg.Web.Port == 0 {
		cfg.Web.Port = 8080
	}
	return cfg, nil
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

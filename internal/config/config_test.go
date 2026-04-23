package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jimyag/commitlens/internal/config"
)

func TestLoad_defaults(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	content := `
repositories:
  - owner: jimyag
    repo: commitlens
`
	os.WriteFile(cfgPath, []byte(content), 0644)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Repositories) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repositories))
	}
	if cfg.Repositories[0].Owner != "jimyag" {
		t.Errorf("expected owner jimyag, got %s", cfg.Repositories[0].Owner)
	}
	if cfg.Cache.Dir == "" {
		t.Error("expected default cache dir to be set")
	}
	if cfg.Web.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Web.Port)
	}
}

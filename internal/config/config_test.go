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
	os.WriteFile(cfgPath, []byte(content), 0o644)

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

func TestRepositoryKinds(t *testing.T) {
	organization := config.Repository{Owner: "example-org"}
	if !organization.IsOrganization() || organization.IsSyncable() {
		t.Fatalf("owner-only entry should be an organization: %#v", organization)
	}

	remote := config.Repository{Owner: "example-org", Repo: "example-repo"}
	if remote.IsOrganization() || !remote.IsSyncable() {
		t.Fatalf("owner/repo entry should be syncable: %#v", remote)
	}

	local := config.Repository{LocalPath: "/tmp/las"}
	if local.IsOrganization() || !local.IsSyncable() {
		t.Fatalf("local_path entry should be syncable: %#v", local)
	}
}

func TestLoadRejectsInvalidReportingTimezone(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	content := `
reporting:
  timezone: Not/A_Timezone
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := config.Load(cfgPath); err == nil {
		t.Fatal("expected invalid reporting timezone error")
	}
}

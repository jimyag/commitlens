package sync_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	syncdata "github.com/jimyag/commitlens/internal/sync"
)

func TestSyncRepoPersistsFailureStatus(t *testing.T) {
	dir := t.TempDir()
	rawCache := cache.NewRawCache(dir)
	statsCache := cache.NewStatsCache(dir)
	cfg := &config.Config{Cache: config.Cache{Dir: dir}}
	syncer := syncdata.New(cfg, rawCache, statsCache)
	repository := config.Repository{LocalPath: filepath.Join(dir, "missing")}

	err := syncer.SyncRepo(context.Background(), repository)
	if err == nil {
		t.Fatal("expected sync failure")
	}
	status, loadErr := rawCache.LoadStatus(repository.ID())
	if loadErr != nil {
		t.Fatalf("load sync status: %v", loadErr)
	}
	if status.State != "failed" || status.Syncing || status.Error == "" {
		t.Fatalf("unexpected failure status: %#v", status)
	}
	if status.LastAttempt.IsZero() || status.FinishedAt.IsZero() {
		t.Fatalf("sync timestamps missing: %#v", status)
	}
	if !strings.Contains(status.Error, "ensure repo failed") {
		t.Fatalf("unexpected error: %q", status.Error)
	}
}

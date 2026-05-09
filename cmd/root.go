package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/locale"
	isync "github.com/jimyag/commitlens/internal/sync"
	"github.com/jimyag/commitlens/internal/tui"
	"github.com/jimyag/commitlens/internal/web"
)

var (
	globalAssets embed.FS
	webMode      bool
	webPort      int
	configFile   string
)

var rootCmd = &cobra.Command{
	Use:   "commitlens",
	Short: "Universal Git Code Contribution Analyzer",
	Long:  "CommitLens shows per-contributor commit and line-change stats for any local or remote Git repository.",
	RunE:  run,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func ExecuteWithAssets(assets embed.FS) {
	globalAssets = assets
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().BoolVar(&webMode, "web", false, "start web UI")
	rootCmd.Flags().IntVar(&webPort, "port", 8080, "web server listen port")
	rootCmd.Flags().StringVar(&configFile, "config", "", "config file (default: ~/.commitlens/config.yaml)")
}

func run(cmd *cobra.Command, args []string) error {
	cfgPath := configFile
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "config file not found: %s\n\ncreate a config file, for example:\n\nrepositories:\n  - local_path: /path/to/my/repo\n  - owner: your-org\n    repo: your-repo\n", cfgPath)
			os.Exit(1)
		}
		return err
	}

	if err := os.MkdirAll(cfg.Cache.Dir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	locale.Init(cfg.Language)

	rawCache := cache.NewRawCache(cfg.Cache.Dir)
	statsCache := cache.NewStatsCache(cfg.Cache.Dir)
	syncer := isync.New(cfg, rawCache, statsCache)

	repos := cfg.Repositories
	var repoNames []string
	for _, r := range repos {
		repoNames = append(repoNames, r.ID())
	}

	var allStats []*cache.StatsData
	hasCache := false
	for _, repoName := range repoNames {
		s, err := statsCache.Load(repoName)
		if err == nil && len(s.Contributors) > 0 {
			allStats = append(allStats, s)
			hasCache = true
		} else {
			allStats = append(allStats, &cache.StatsData{Repo: repoName})
		}
	}

	// Only run blocking sync UI if we have no cached data at all.
	if !hasCache {
		runSync(cmd.Context(), syncer, repos)
		// Reload stats after sync
		allStats = nil
		for _, repoName := range repoNames {
			s, _ := statsCache.Load(repoName)
			allStats = append(allStats, s)
		}
	}

	if webMode {
		port := webPort
		if cfg.Web.Port != 0 && cfg.Web.Port != 8080 {
			port = cfg.Web.Port
		}
		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("CommitLens web UI: http://localhost%s\n", addr)
		srv := web.New(globalAssets, syncer, allStats, repoNames, rawCache)
		return srv.Run(addr)
	}

	return tui.Run(syncer, allStats, repos, rawCache)
}

// runSync launches a bubbletea progress UI while syncing all repos concurrently.
func runSync(ctx context.Context, syncer *isync.Syncer, repos []config.Repository) {
	// Large buffer so sync never blocks on progress while TUI is drawing.
	progress := make(chan isync.Progress, 65536)

	go func() {
		syncer.SyncAll(ctx, repos, progress, 5)
	}()

	var repoNames []string
	for _, r := range repos {
		repoNames = append(repoNames, r.ID())
	}

	if err := tui.RunSyncProgress(repoNames, progress); err != nil {
		// Drain remaining events so SyncAll can finish
		for range progress {
		}
	}
}

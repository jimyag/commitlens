package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/git"
	"github.com/jimyag/commitlens/internal/locale"
	"github.com/jimyag/commitlens/internal/stats"
	isync "github.com/jimyag/commitlens/internal/sync"
	"github.com/jimyag/commitlens/internal/tui"
	"github.com/jimyag/commitlens/internal/web"
	"github.com/spf13/cobra"
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

	if err := os.MkdirAll(cfg.Cache.Dir, 0o755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	locale.Init(cfg.Language)

	rawCache := cache.NewRawCache(cfg.Cache.Dir)
	statsCache := cache.NewStatsCache(cfg.Cache.Dir)
	syncer := isync.New(cfg, rawCache, statsCache)

	repos := cfg.Repositories
	// Discovery
	if len(cfg.DiscoveryRoots) > 0 {
		discovered, err := git.Discover(cfg.DiscoveryRoots)
		if err != nil {
			return fmt.Errorf("discovery failed: %w", err)
		}
		repos = append(repos, discovered...)
	}

	// Deduplicate by ID
	uniqueRepos := make([]config.Repository, 0, len(repos))
	seenIDs := make(map[string]struct{})
	for _, r := range repos {
		id := r.ID()
		if _, ok := seenIDs[id]; !ok {
			seenIDs[id] = struct{}{}
			uniqueRepos = append(uniqueRepos, r)
		}
	}
	repos = uniqueRepos

	var allStats []*cache.StatsData
	hasRawCache := false
	for _, repo := range repos {
		repoID := repo.ID()
		raw, err := rawCache.Load(repoID)
		if err == nil && len(raw.Commits) > 0 {
			// Always re-aggregate from raw data to respect latest userMap config
			s := stats.Aggregate(raw, cfg)
			_ = statsCache.Save(s)
			allStats = append(allStats, s)
			hasRawCache = true
		} else {
			allStats = append(allStats, &cache.StatsData{Repo: repoID})
		}
	}

	// Only run blocking sync UI if we have no raw cached data at all.
	if !hasRawCache {
		runSync(cmd.Context(), syncer, repos)
		// Reload and aggregate after sync
		allStats = nil
		for _, repo := range repos {
			raw, _ := rawCache.Load(repo.ID())
			s := stats.Aggregate(raw, cfg)
			_ = statsCache.Save(s)
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
		srv := web.New(globalAssets, syncer, allStats, repos, rawCache)
		return srv.Run(addr)
	}

	return tui.Run(syncer, allStats, repos, rawCache, statsCache)
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

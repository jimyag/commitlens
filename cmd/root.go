package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	ghapi "github.com/jimyag/commitlens/internal/github"
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
			fmt.Fprintf(os.Stderr, "config file not found: %s\n\ncreate a config file, for example:\n\nrepositories:\n  - local_path: /path/to/my/repo\n  - owner: your-org\n    repo: your-repo\n  - owner: another-org  # Web: select repositories before syncing\n", cfgPath)
			os.Exit(1)
		}
		return err
	}

	if err := os.MkdirAll(cfg.Cache.Dir, 0o755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	locale.Init(cfg.Language)
	cfg.GitHub.Token = ghapi.ResolveToken(cfg.GitHub.Token)

	rawCache := cache.NewRawCache(cfg.Cache.Dir)
	statsCache := cache.NewStatsCache(cfg.Cache.Dir)
	syncer := isync.New(cfg, rawCache, statsCache)

	repositorySet, err := resolveRepositories(cmd.Context(), cfg, webMode)
	if err != nil {
		return err
	}
	repos := repositorySet.available
	startupRepos := repositorySet.startup
	startupRepoIDs := make(map[string]struct{}, len(startupRepos))
	for _, repo := range startupRepos {
		startupRepoIDs[repo.ID()] = struct{}{}
	}

	var allStats []*cache.StatsData
	hasStartupRawCache := false
	for _, repo := range repos {
		repoID := repo.ID()
		raw, err := rawCache.Load(repoID)
		if err == nil && len(raw.Commits) > 0 {
			// Always re-aggregate from raw data to respect latest userMap config
			s := stats.Aggregate(raw, cfg)
			_ = statsCache.Save(s)
			allStats = append(allStats, s)
			if _, ok := startupRepoIDs[repoID]; ok {
				hasStartupRawCache = true
			}
		} else {
			allStats = append(allStats, &cache.StatsData{Repo: repoID})
		}
	}

	// Organization repositories are selectable in Web mode and are never
	// synchronized automatically on startup.
	if len(startupRepos) > 0 && !hasStartupRawCache {
		runSync(cmd.Context(), syncer, startupRepos)
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
		srv := web.New(globalAssets, syncer, repos, startupRepos, cfg, rawCache, statsCache)
		return srv.Run(addr)
	}

	reportingLocation, _ := cfg.ReportingLocation()
	return tui.Run(syncer, allStats, repos, rawCache, statsCache, reportingLocation)
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

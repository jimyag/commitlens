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
	gh "github.com/jimyag/commitlens/internal/github"
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
	Short: "GitHub contribution stats viewer",
	Long:  "CommitLens shows per-contributor merged PR, commit, and line-change stats for GitHub repositories.",
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
			fmt.Fprintf(os.Stderr, "config file not found: %s\n\ncreate a config file, for example:\n\nrepositories:\n  - owner: your-org\n    repo: your-repo\n", cfgPath)
			os.Exit(1)
		}
		return err
	}

	if err := os.MkdirAll(cfg.Cache.Dir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	locale.Init(cfg.Language)

	client := gh.NewClient(cfg.GitHub.Token)
	rawCache := cache.NewRawCache(cfg.Cache.Dir)
	statsCache := cache.NewStatsCache(cfg.Cache.Dir)
	syncer := isync.New(client, rawCache, statsCache)

	repos := make([]string, len(cfg.Repositories))
	for i, r := range cfg.Repositories {
		repos[i] = r.Owner + "/" + r.Repo
	}

	runSync(cmd.Context(), syncer, repos)

	var allStats []*cache.StatsData
	for _, repo := range repos {
		s, err := statsCache.Load(repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load stats for %s: %v\n", repo, err)
			continue
		}
		allStats = append(allStats, s)
	}

	if webMode {
		port := webPort
		if cfg.Web.Port != 0 && cfg.Web.Port != 8080 {
			port = cfg.Web.Port
		}
		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("CommitLens web UI: http://localhost%s\n", addr)
		srv := web.New(globalAssets, syncer, allStats, repos)
		return srv.Run(addr)
	}

	return tui.Run(syncer, allStats, repos)
}

// runSync launches a bubbletea progress UI while syncing all repos concurrently.
func runSync(ctx context.Context, syncer *isync.Syncer, repos []string) {
	progress := make(chan isync.Progress, len(repos)*64)

	go func() {
		syncer.SyncAll(ctx, repos, progress, 5)
	}()

	if err := tui.RunSyncProgress(repos, progress); err != nil {
		// Drain remaining events so SyncAll can finish
		for range progress {
		}
	}
}

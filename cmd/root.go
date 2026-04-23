package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
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
	Long:  "CommitLens 统计 GitHub 仓库各贡献者的 PR、commit 和增删行数据。",
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
	rootCmd.Flags().BoolVar(&webMode, "web", false, "启动 Web UI 模式")
	rootCmd.Flags().IntVar(&webPort, "port", 8080, "Web UI 监听端口")
	rootCmd.Flags().StringVar(&configFile, "config", "", "配置文件路径（默认: ~/.commitlens/config.yaml）")
}

func run(cmd *cobra.Command, args []string) error {
	cfgPath := configFile
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "配置文件不存在: %s\n\n请创建配置文件，示例内容:\n\nrepositories:\n  - owner: your-org\n    repo: your-repo\n", cfgPath)
			os.Exit(1)
		}
		return err
	}

	if err := os.MkdirAll(cfg.Cache.Dir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	client := gh.NewClient(cfg.GitHub.Token)
	rawCache := cache.NewRawCache(cfg.Cache.Dir)
	statsCache := cache.NewStatsCache(cfg.Cache.Dir)
	syncer := isync.New(client, rawCache, statsCache)

	repos := make([]string, len(cfg.Repositories))
	for i, r := range cfg.Repositories {
		repos[i] = r.Owner + "/" + r.Repo
	}

	runSync(cmd.Context(), syncer, repos)

	// 加载聚合统计数据
	var allStats []*cache.StatsData
	for _, repo := range repos {
		s, err := statsCache.Load(repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 加载 %s 统计数据失败: %v\n", repo, err)
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
		fmt.Printf("CommitLens Web UI 已启动: http://localhost%s\n", addr)
		srv := web.New(globalAssets, syncer, allStats, repos)
		return srv.Run(addr)
	}

	return tui.Run(syncer, allStats, repos)
}

// runSync runs all repos concurrently and prints real-time progress to stderr.
func runSync(ctx context.Context, syncer *isync.Syncer, repos []string) {
	fmt.Fprintln(os.Stderr, "正在同步数据...")

	progress := make(chan isync.Progress, len(repos)*32)

	// Track per-repo last line length for \r overwrite
	var mu sync.Mutex
	lines := make(map[string]int) // repo -> last printed line length

	printProgress := func(p isync.Progress) {
		mu.Lock()
		defer mu.Unlock()

		var line string
		switch {
		case p.Err != nil:
			line = fmt.Sprintf("  [%d/%d] %-30s 失败: %v", p.RepoIndex, p.RepoTotal, p.Repo, p.Err)
		case p.Done:
			line = fmt.Sprintf("  [%d/%d] %-30s 完成", p.RepoIndex, p.RepoTotal, p.Repo)
		case p.PRsTotal > 0:
			line = fmt.Sprintf("  [%d/%d] %-30s 拉取 commit %d/%d", p.RepoIndex, p.RepoTotal, p.Repo, p.PRsFetched, p.PRsTotal)
		default:
			line = fmt.Sprintf("  [%d/%d] %-30s 拉取 PR 列表...", p.RepoIndex, p.RepoTotal, p.Repo)
		}

		prev := lines[p.Repo]
		if prev > 0 {
			// Clear previous line and rewrite
			fmt.Fprintf(os.Stderr, "\r%-*s\r%s", prev, "", line)
		} else {
			fmt.Fprintf(os.Stderr, "%s", line)
		}

		if p.Done || p.Err != nil {
			fmt.Fprintln(os.Stderr)
			delete(lines, p.Repo)
		} else {
			lines[p.Repo] = len(line)
		}
	}

	// Drain progress channel in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for p := range progress {
			printProgress(p)
		}
	}()

	syncer.SyncAll(ctx, repos, progress, 3)
	wg.Wait()
}

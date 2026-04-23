package cmd

import (
	"embed"
	"fmt"
	"os"

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

	// 启动时自动增量同步
	fmt.Fprintln(os.Stderr, "正在同步数据...")
	for _, repo := range repos {
		fmt.Fprintf(os.Stderr, "  同步 %s ...\n", repo)
		if err := syncer.SyncRepo(cmd.Context(), repo); err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 同步 %s 失败: %v\n", repo, err)
		}
	}

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

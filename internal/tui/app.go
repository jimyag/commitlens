package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
	isync "github.com/jimyag/commitlens/internal/sync"
)

type viewMode int

const (
	viewSummary viewMode = iota
	viewRepo
	viewTrend
)

type syncDoneMsg struct{ err error }

type App struct {
	mode                viewMode
	stats               []*cache.StatsData
	repoNames           []string
	selectedRepo        int
	selectedContributor int
	granularity         int // 0=week 1=month 2=quarter 3=year
	syncer              *isync.Syncer
	syncing             bool
	width               int
	height              int
	err                 error
}

var granularityLabels = []string{"周", "月", "季度", "年"}

func New(syncer *isync.Syncer, stats []*cache.StatsData, repos []string) *App {
	return &App{
		syncer:    syncer,
		stats:     stats,
		repoNames: repos,
	}
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			a.mode = (a.mode + 1) % 3
		case "r":
			if !a.syncing {
				a.syncing = true
				return a, a.doSync()
			}
		case "up", "k":
			if a.mode == viewRepo && a.selectedRepo > 0 {
				a.selectedRepo--
			}
			if a.mode == viewTrend && a.selectedContributor > 0 {
				a.selectedContributor--
			}
		case "down", "j":
			if a.mode == viewRepo && a.selectedRepo < len(a.repoNames)-1 {
				a.selectedRepo++
			}
			if a.mode == viewTrend {
				maxC := a.maxContributors()
				if a.selectedContributor < maxC-1 {
					a.selectedContributor++
				}
			}
		case "left", "h":
			if a.granularity > 0 {
				a.granularity--
			}
		case "right", "l":
			if a.granularity < 3 {
				a.granularity++
			}
		}
	case syncDoneMsg:
		a.syncing = false
		a.err = msg.err
	}
	return a, nil
}

func (a *App) doSync() tea.Cmd {
	return func() tea.Msg {
		return syncDoneMsg{err: nil}
	}
}

func (a *App) maxContributors() int {
	total := 0
	for _, s := range a.stats {
		total += len(s.Contributors)
	}
	return total
}

func (a *App) View() tea.View {
	header := a.renderHeader()
	var body string
	switch a.mode {
	case viewSummary:
		body = a.renderSummary()
	case viewRepo:
		body = a.renderRepo()
	case viewTrend:
		body = a.renderTrend()
	}
	status := a.renderStatus()
	v := tea.NewView(fmt.Sprintf("%s\n%s\n%s", header, body, status))
	v.AltScreen = true
	return v
}

func (a *App) renderHeader() string {
	tabs := []string{"[汇总]", "[单仓库]", "[趋势]"}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	line := ""
	for i, t := range tabs {
		if viewMode(i) == a.mode {
			line += style.Render(t) + " "
		} else {
			line += t + " "
		}
	}
	return "CommitLens  " + line + "        tab:切换  r:刷新  q:退出"
}

func (a *App) renderStatus() string {
	if a.syncing {
		return "状态: 同步中..."
	}
	if a.err != nil {
		return fmt.Sprintf("错误: %v", a.err)
	}
	return ""
}

func (a *App) renderSummary() string { return renderSummaryView(a) }
func (a *App) renderRepo() string    { return renderRepoView(a) }
func (a *App) renderTrend() string   { return renderTrendView(a) }

func Run(syncer *isync.Syncer, stats []*cache.StatsData, repos []string) error {
	app := New(syncer, stats, repos)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}

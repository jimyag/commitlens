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

// trend 仓库列表与贡献者列表的焦点：0=仓库 1=贡献者
const (
	trendFocusRepo = iota
	trendFocusContributors
)

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
	// 趋势视图中按仓库筛数据：默认单仓；m 可切多选，多选时用 trendRepoMultiPick
	trendListFocus   int
	trendSelectMulti bool
	trendOneRepo     int
	trendRepoCursor  int
	trendRepoMulti   map[int]struct{}
	trendHScroll      int // 趋势图柱区横向视口左偏移（rune 列）
	trendLastPeriodN  int // 用于周期数变化时重置横滚
}

var granularityLabels = []string{"周", "月", "季度", "年"}

func New(syncer *isync.Syncer, stats []*cache.StatsData, repos []string) *App {
	return &App{
		syncer:          syncer,
		stats:           stats,
		repoNames:       repos,
		trendListFocus:  trendFocusRepo,
		trendOneRepo:    0,
		trendRepoCursor: 0,
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
		if a.mode == viewTrend && a.trendHScrollKey(msg) {
			return a, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			if a.mode == viewTrend {
				a.trendHScroll = 0
			}
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
			if a.mode == viewTrend {
				a.trendViewUp()
			}
		case "down", "j":
			if a.mode == viewRepo && a.selectedRepo < len(a.repoNames)-1 {
				a.selectedRepo++
			}
			if a.mode == viewTrend {
				a.trendViewDown()
			}
		case " ":
			if a.mode == viewTrend {
				a.trendViewSpace()
			}
		case "m", "M":
			if a.mode == viewTrend {
				a.trendViewToggleMode()
			}
		case ",":
			if a.mode == viewTrend {
				a.trendListFocus = trendFocusRepo
			}
		case ".":
			if a.mode == viewTrend {
				a.trendListFocus = trendFocusContributors
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
	seen := make(map[string]struct{})
	for _, s := range a.stats {
		for login := range s.Contributors {
			seen[login] = struct{}{}
		}
	}
	return len(seen)
}

func (a *App) nRepos() int { return len(a.repoNames) }

const trendHScrollStep = 8

// trendHScrollKey 处理趋势内横向滚动，true 表示已消费该键（不再走粒度等逻辑）。
func (a *App) trendHScrollKey(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	if k.Code == tea.KeyLeft && (k.Mod&tea.ModShift) != 0 {
		a.trendHScroll -= trendHScrollStep
		if a.trendHScroll < 0 {
			a.trendHScroll = 0
		}
		return true
	}
	if k.Code == tea.KeyRight && (k.Mod&tea.ModShift) != 0 {
		a.trendHScroll += trendHScrollStep
		return true
	}
	if k.Code == tea.KeyHome {
		a.trendHScroll = 0
		return true
	}
	if k.Code == tea.KeyEnd {
		a.trendHScroll = 1e9
		return true
	}
	s := msg.String()
	if s == "<" {
		a.trendHScroll -= trendHScrollStep
		if a.trendHScroll < 0 {
			a.trendHScroll = 0
		}
		return true
	}
	if s == ">" {
		a.trendHScroll += trendHScrollStep
		return true
	}
	return false
}

func (a *App) trendViewUp() {
	if a.trendListFocus == trendFocusRepo {
		if a.trendSelectMulti {
			if a.trendRepoCursor > 0 {
				a.trendRepoCursor--
			}
		} else {
			if a.trendOneRepo > 0 {
				a.trendOneRepo--
				a.trendRepoCursor = a.trendOneRepo
			}
		}
		return
	}
	if a.selectedContributor > 0 {
		a.selectedContributor--
	}
}

func (a *App) trendViewDown() {
	if a.trendListFocus == trendFocusRepo {
		max := a.nRepos() - 1
		if a.trendSelectMulti {
			if a.trendRepoCursor < max {
				a.trendRepoCursor++
			}
		} else {
			if a.trendOneRepo < max {
				a.trendOneRepo++
				a.trendRepoCursor = a.trendOneRepo
			}
		}
		return
	}
	maxC := a.maxTrendContributors()
	if a.selectedContributor < maxC-1 {
		a.selectedContributor++
	}
}

func (a *App) trendViewSpace() {
	if a.trendListFocus != trendFocusRepo || !a.trendSelectMulti {
		return
	}
	if a.trendRepoMulti == nil {
		return
	}
	_, in := a.trendRepoMulti[a.trendRepoCursor]
	if in {
		if len(a.trendRepoMulti) <= 1 {
			return
		}
		delete(a.trendRepoMulti, a.trendRepoCursor)
	} else {
		a.trendRepoMulti[a.trendRepoCursor] = struct{}{}
	}
}

func (a *App) trendViewToggleMode() {
	if a.nRepos() == 0 {
		return
	}
	if !a.trendSelectMulti {
		a.trendSelectMulti = true
		if a.trendRepoMulti == nil {
			a.trendRepoMulti = make(map[int]struct{})
		} else {
			for k := range a.trendRepoMulti {
				delete(a.trendRepoMulti, k)
			}
		}
		i := a.trendOneRepo
		if i < 0 {
			i = 0
		} else if i > a.nRepos()-1 {
			i = a.nRepos() - 1
		}
		a.trendRepoMulti[i] = struct{}{}
		if a.trendRepoCursor < 0 || a.trendRepoCursor > a.nRepos()-1 {
			a.trendRepoCursor = i
		}
		return
	}
	a.trendSelectMulti = false
	first := -1
	for j := 0; j < a.nRepos(); j++ {
		if _, ok := a.trendRepoMulti[j]; ok {
			if first < 0 {
				first = j
			}
		}
	}
	if first < 0 {
		first = 0
	}
	a.trendOneRepo = first
	a.trendRepoCursor = first
}

func (a *App) maxTrendContributors() int {
	return len(contributorLoginsForTrend(a))
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

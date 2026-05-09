package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/locale"
	"github.com/jimyag/commitlens/internal/stats"
	isync "github.com/jimyag/commitlens/internal/sync"
)

type viewMode int

const (
	viewSummary viewMode = iota
	viewRepo
	viewTrend
	viewCommitList
)

type syncDoneMsg struct{ err error }

type App struct {
	mode      viewMode
	stats     []*cache.StatsData
	repoNames []string
	repos     []config.Repository
	rawCache  *cache.RawCache
	syncer    *isync.Syncer
	syncing   bool
	width     int
	height    int
	err       error

	// Global Filters
	globalFocus         int              // 0=Repo, 1=Granularity, 2=Contributor, 3=ViewContent
	globalRepoMulti     map[int]struct{} // nil means ALL, otherwise specific repos
	globalRepoExpanded  bool             // true if repo multi-select list is open
	globalRepoCursor    int              // cursor for multi-select list
	globalGranularity   int              // 0=Week, 1=Month, 2=Quarter, 3=Year
	globalLoginIdx      int              // 0=ALL, 1..N
	globalLoginExpanded bool
	globalLoginCursor   int
	globalLoginSearch   string

	// Repo view specific
	repoViewSelected int

	// Trend specific
	trendHScroll      int // 趋势图柱区横向视口左偏移（rune 列）
	trendLastPeriodN  int // 用于周期数变化时重置横滚
	trendPeriodCursor int // 趋势图中当前选中的周期索引

	// Commit List view state
	commitListFocus     int // 0=Period, 1=Table
	commitListPeriodIdx int // 0=All, 1...N
	commitList          []commitItem
	commitListCursor    int
}

type commitItem struct {
	Repo         string
	SHA          string
	Title        string
	Author       string
	Participants []string
	Date         time.Time
	Additions    int
	Deletions    int
}

func New(syncer *isync.Syncer, stats []*cache.StatsData, repos []config.Repository, rawCache *cache.RawCache) *App {
	repoNames := make([]string, len(repos))
	for i, r := range repos {
		repoNames[i] = r.ID()
	}
	return &App{
		syncer:            syncer,
		stats:             stats,
		repoNames:         repoNames,
		repos:             repos,
		rawCache:          rawCache,
		globalFocus:       0,
		globalGranularity: 0,
		globalLoginIdx:    0,
		commitListFocus:   0,
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
		s := msg.String()
		
		// Handle typing for search if expanded
		if a.globalFocus == 2 && a.globalLoginExpanded && len(s) == 1 {
			// Only alphanumeric or punctuation (printable)
			c := s[0]
			if c >= 32 && c <= 126 {
				a.globalLoginSearch += s
				a.globalLoginCursor = 0
				return a, nil
			}
		}

		switch s {
		case "q", "ctrl+c":
			if a.globalLoginExpanded {
				// Don't quit if just typing 'q' in search
				a.globalLoginSearch += "q"
				a.globalLoginCursor = 0
				return a, nil
			}
			return a, tea.Quit
		case "tab":
			if a.globalFocus == 3 {
				if a.mode == viewCommitList {
					a.commitListFocus = (a.commitListFocus + 1) % 2
					if a.commitListFocus == 0 {
						a.globalFocus = 0
					}
				} else {
					a.globalFocus = 0
				}
			} else {
				a.globalFocus = (a.globalFocus + 1) % 4
			}
			if a.globalFocus != 0 {
				a.globalRepoExpanded = false
			}
			if a.globalFocus != 2 {
				a.globalLoginExpanded = false
				a.globalLoginSearch = ""
			}
		case "1", "2", "3", "4":
			if a.globalLoginExpanded {
				a.globalLoginSearch += s
				a.globalLoginCursor = 0
				return a, nil
			}
			m := viewMode(s[0] - '1')
			if m != a.mode {
				a.mode = m
				if a.mode == viewCommitList {
					a.refreshCommitList()
				}
			}
		case "[":
			a.mode = (a.mode + 3) % 4
			if a.mode == viewCommitList {
				a.refreshCommitList()
			}
		case "]":
			a.mode = (a.mode + 1) % 4
			if a.mode == viewCommitList {
				a.refreshCommitList()
			}
		case "esc":
			if a.globalRepoExpanded {
				a.globalRepoExpanded = false
			} else if a.globalLoginExpanded {
				a.globalLoginExpanded = false
				a.globalLoginSearch = ""
			} else if a.mode == viewCommitList {
				a.mode = viewTrend
			}
		case "backspace", "delete":
			if a.globalFocus == 2 && a.globalLoginExpanded {
				if len(a.globalLoginSearch) > 0 {
					a.globalLoginSearch = a.globalLoginSearch[:len(a.globalLoginSearch)-1]
					a.globalLoginCursor = 0
				}
			} else if a.globalRepoExpanded {
				a.globalRepoExpanded = false
			} else if a.mode == viewCommitList {
				a.mode = viewTrend
			}
		case "enter":
			if a.globalFocus == 0 {
				if a.globalRepoExpanded {
					// Exclusively select this repo
					a.preserveLoginIdxDuring(func() {
						visible := a.visibleRepoIndices()
						if a.globalRepoCursor >= 0 && a.globalRepoCursor < len(visible) {
							repoIdx := visible[a.globalRepoCursor]
							a.globalRepoMulti = map[int]struct{}{repoIdx: {}}
						}
						a.globalRepoExpanded = false
					})
				} else {
					a.globalRepoExpanded = true
					a.globalRepoCursor = 0
				}
			} else if a.globalFocus == 2 {
				if a.globalLoginExpanded {
					filtered := a.filteredLogins()
					if a.globalLoginCursor >= 0 && a.globalLoginCursor < len(filtered) {
						selected := filtered[a.globalLoginCursor]
						all := a.availableGlobalLogins()
						for i, l := range all {
							if l == selected {
								a.globalLoginIdx = i
								break
							}
						}
					}
					a.globalLoginExpanded = false
					a.globalLoginSearch = ""
					a.onGlobalFilterChange()
				} else {
					a.globalLoginExpanded = true
					a.globalLoginSearch = ""
					a.globalLoginCursor = 0
				}
			} else if a.mode == viewTrend {
				a.openCommitListFromTrend()
			}
		case "space", " ":
			if a.globalFocus == 0 {
				if a.globalRepoExpanded {
					visible := a.visibleRepoIndices()
					if a.globalRepoCursor >= 0 && a.globalRepoCursor < len(visible) {
						repoIdx := visible[a.globalRepoCursor]
						a.toggleGlobalRepo(repoIdx)
					}
				} else {
					a.globalRepoExpanded = true
					a.globalRepoCursor = 0
				}
			} else if a.globalFocus == 2 && a.globalLoginExpanded {
				a.globalLoginSearch += " "
				a.globalLoginCursor = 0
			}
		case "r":
			if a.globalLoginExpanded {
				a.globalLoginSearch += "r"
				a.globalLoginCursor = 0
				return a, nil
			}
			if !a.syncing {
				a.syncing = true
				return a, a.doSync()
			}
		case "up", "k":
			if a.globalLoginExpanded {
				if a.globalFocus == 2 {
					if a.globalLoginCursor > 0 {
						a.globalLoginCursor--
					}
				}
			} else if a.globalFocus == 0 && a.globalRepoExpanded {
				if a.globalRepoCursor > 0 {
					a.globalRepoCursor--
				}
			} else if a.globalFocus == 3 {
				if a.mode == viewRepo {
					if a.repoViewSelected > 0 {
						a.repoViewSelected--
					}
				} else if a.mode == viewCommitList {
					if a.commitListFocus == 1 && a.commitListCursor > 0 {
						a.commitListCursor--
					}
				}
			}
		case "down", "j":
			if a.globalLoginExpanded {
				if a.globalFocus == 2 {
					filtered := a.filteredLogins()
					if a.globalLoginCursor < len(filtered)-1 {
						a.globalLoginCursor++
					}
				}
			} else if a.globalFocus == 0 && a.globalRepoExpanded {
				visible := a.visibleRepoIndices()
				if a.globalRepoCursor < len(visible)-1 {
					a.globalRepoCursor++
				}
			} else if a.globalFocus == 3 {
				if a.mode == viewRepo {
					if a.repoViewSelected < len(a.repoNames)-1 {
						a.repoViewSelected++
					}
				} else if a.mode == viewCommitList {
					if a.commitListFocus == 1 && a.commitListCursor < len(a.commitList)-1 {
						a.commitListCursor++
					}
				}
			}
		case "left", "h":
			if a.globalFocus == 1 {
				if a.globalGranularity > 0 {
					a.globalGranularity--
					a.onGlobalFilterChange()
				}
			} else if a.globalFocus == 2 {
				if !a.globalLoginExpanded {
					if a.globalLoginIdx > 0 {
						a.globalLoginIdx--
						a.onGlobalFilterChange()
					}
				}
			} else if a.globalFocus == 3 {
				if a.mode == viewTrend {
					if a.trendPeriodCursor > 0 {
						a.trendPeriodCursor--
					}
				} else if a.mode == viewCommitList {
					if a.commitListFocus == 0 {
						if a.commitListPeriodIdx > 0 {
							a.commitListPeriodIdx--
							a.refreshCommitList()
						}
					}
				}
			}
		case "right", "l":
			if a.globalFocus == 1 {
				if a.globalGranularity < 3 {
					a.globalGranularity++
					a.onGlobalFilterChange()
				}
			} else if a.globalFocus == 2 {
				if !a.globalLoginExpanded {
					if a.globalLoginIdx < len(a.availableGlobalLogins())-1 {
						a.globalLoginIdx++
						a.onGlobalFilterChange()
					}
				}
			} else if a.globalFocus == 3 {
				if a.mode == viewTrend {
					maxP := len(sortedPeriodKeys(aggregatePeriods(a)))
					if a.trendPeriodCursor < maxP-1 {
						a.trendPeriodCursor++
					}
				} else if a.mode == viewCommitList {
					if a.commitListFocus == 0 {
						if a.commitListPeriodIdx < len(a.availableCommitListPeriods())-1 {
							a.commitListPeriodIdx++
							a.refreshCommitList()
						}
					}
				}
			}
		}
	case syncDoneMsg:
		a.syncing = false
		a.err = msg.err
	}
	return a, nil
}

func (a *App) preserveLoginIdxDuring(changeFn func()) {
	oldLogins := a.availableGlobalLogins()
	var oldLogin string
	if a.globalLoginIdx >= 0 && a.globalLoginIdx < len(oldLogins) {
		oldLogin = oldLogins[a.globalLoginIdx]
	}

	changeFn()

	newLogins := a.availableGlobalLogins()
	newIdx := 0
	if oldLogin != "" && oldLogin != locale.T("tui.prlist.filterAll") {
		for i, l := range newLogins {
			if l == oldLogin {
				newIdx = i
				break
			}
		}
	}
	a.globalLoginIdx = newIdx
	a.onGlobalFilterChange()
}

func (a *App) toggleGlobalRepo(idx int) {
	a.preserveLoginIdxDuring(func() {
		if a.globalRepoMulti == nil {
			a.globalRepoMulti = make(map[int]struct{})
			for i := range a.repoNames {
				a.globalRepoMulti[i] = struct{}{}
			}
		}
		if _, ok := a.globalRepoMulti[idx]; ok {
			if len(a.globalRepoMulti) > 1 {
				delete(a.globalRepoMulti, idx)
			}
		} else {
			a.globalRepoMulti[idx] = struct{}{}
		}
	})
}

func (a *App) onGlobalFilterChange() {
	if a.mode == viewCommitList {
		a.refreshCommitList()
	}
}

func (a *App) availableGlobalLogins() []string {
	logins := contributorsSortedByCommitCount(trendFilteredStats(a))
	out := []string{locale.T("tui.prlist.filterAll")}
	out = append(out, logins...)
	return out
}

func (a *App) filteredLogins() []string {
	all := a.availableGlobalLogins()
	if a.globalLoginSearch == "" {
		return all
	}
	search := strings.ToLower(a.globalLoginSearch)
	var out []string
	for _, l := range all {
		if strings.Contains(strings.ToLower(l), search) {
			out = append(out, l)
		}
	}
	return out
}

func (a *App) availableCommitListPeriods() []string {
	periods := sortedPeriodKeys(aggregatePeriods(a))
	out := []string{locale.T("tui.prlist.filterAll")}
	out = append(out, periods...)
	return out
}

func (a *App) openCommitListFromTrend() {
	periods := sortedPeriodKeys(aggregatePeriods(a))
	if len(periods) == 0 || a.trendPeriodCursor >= len(periods) {
		return
	}
	period := periods[a.trendPeriodCursor]

	a.mode = viewCommitList
	a.globalFocus = 3
	a.commitListFocus = 1 // Focus on table

	// Find indices for the selected period
	a.commitListPeriodIdx = 0
	for i, p := range a.availableCommitListPeriods() {
		if p == period {
			a.commitListPeriodIdx = i
			break
		}
	}
	a.refreshCommitList()
}

func (a *App) refreshCommitList() {
	periods := a.availableCommitListPeriods()

	var period string
	if a.commitListPeriodIdx > 0 && a.commitListPeriodIdx < len(periods) {
		period = periods[a.commitListPeriodIdx]
	}

	logins := a.availableGlobalLogins()
	var login string
	if a.globalLoginIdx > 0 && a.globalLoginIdx < len(logins) {
		login = logins[a.globalLoginIdx]
	}

	a.commitList = a.fetchCommits(period, login)
	a.commitListCursor = 0
}

func (a *App) fetchCommits(period, login string) []commitItem {
	var out []commitItem
	stats_ := trendFilteredStats(a)
	for _, s := range stats_ {
		raw, err := a.rawCache.Load(s.Repo)
		if err != nil {
			continue
		}
		for _, commit := range raw.Commits {
			if period != "" {
				if toPeriodKey(stats.WeekKey(commit.Date), a.globalGranularity) != period {
					continue
				}
			}
			if login != "" {
				found := false
				participants := commit.Participants
				if len(participants) == 0 {
					participants = []string{commit.Author}
				}
				for _, p := range participants {
					if p == login {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			participants := commit.Participants
			if len(participants) == 0 {
				participants = []string{commit.Author}
			}
			
			out = append(out, commitItem{
				Repo:         s.Repo,
				SHA:          commit.SHA,
				Title:        commit.Message,
				Author:       commit.Author,
				Participants: participants,
				Date:         commit.Date,
				Additions:    commit.Additions,
				Deletions:    commit.Deletions,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Date.After(out[j].Date)
	})
	return out
}

func (a *App) doSync() tea.Cmd {
	return func() tea.Msg {
		a.syncer.SyncAll(context.Background(), a.repos, nil, 5)
		return syncDoneMsg{err: nil}
	}
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

func (a *App) View() tea.View {
	header := a.renderHeader()
	filters := a.renderGlobalFilters()
	var body string
	switch a.mode {
	case viewSummary:
		body = a.renderSummary()
	case viewRepo:
		body = a.renderRepo()
	case viewTrend:
		body = a.renderTrend()
	case viewCommitList:
		body = a.renderCommitList()
	}
	status := a.renderStatus()
	v := tea.NewView(fmt.Sprintf("%s\n%s\n\n%s\n%s", header, filters, body, status))
	v.AltScreen = true
	return v
}

func (a *App) renderHeader() string {
	tabs := []string{
		locale.T("tui.tab.summary"),
		locale.T("tui.tab.repos"),
		locale.T("tui.tab.trend"),
		locale.T("tui.tab.prlist"),
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	line := ""
	for i, t := range tabs {
		if viewMode(i) == a.mode {
			line += style.Render(t) + " "
		} else {
			line += t + " "
		}
	}
	return "CommitLens  " + line + locale.T("tui.header.hints")
}

func (a *App) renderStatus() string {
	if a.syncing {
		return locale.T("tui.status.syncing")
	}
	if a.err != nil {
		return fmt.Sprintf(locale.T("tui.status.error"), a.err)
	}
	return ""
}

func (a *App) renderSummary() string { return renderSummaryView(a) }
func (a *App) renderRepo() string    { return renderRepoView(a) }
func (a *App) renderTrend() string   { return renderTrendView(a) }

func Run(syncer *isync.Syncer, stats []*cache.StatsData, repos []config.Repository, rawCache *cache.RawCache) error {
	app := New(syncer, stats, repos, rawCache)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}

var (
	globalFilterBoxStyle       = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	globalFilterBoxActiveStyle = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))
	globalFilterLabelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	globalFilterValueStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
)

func (a *App) visibleRepoIndices() []int {
	var selectedLogin string
	logins := a.availableGlobalLogins()
	if a.globalLoginIdx > 0 && a.globalLoginIdx < len(logins) {
		selectedLogin = logins[a.globalLoginIdx]
	}

	var out []int
	for i, s := range a.stats {
		if selectedLogin == "" {
			out = append(out, i)
			continue
		}
		if _, ok := s.Contributors[selectedLogin]; ok {
			out = append(out, i)
		}
	}
	return out
}

func (a *App) renderGlobalFilters() string {
	// Repo filter
	repoVal := locale.T("tui.prlist.filterAll")
	if a.globalRepoMulti != nil {
		if len(a.globalRepoMulti) == 1 {
			for idx := range a.globalRepoMulti {
				repoVal = a.repoNames[idx]
			}
		} else {
			repoVal = fmt.Sprintf(locale.T("tui.scope.multifmt"), len(a.globalRepoMulti))
		}
	}
	repoStyle := globalFilterBoxStyle
	if a.globalFocus == 0 {
		repoStyle = globalFilterBoxActiveStyle
	}
	repoFilter := repoStyle.Render(fmt.Sprintf("%s: %s", globalFilterLabelStyle.Render(locale.T("tui.focus.repo")), globalFilterValueStyle.Render(repoVal)))

	// Granularity filter
	granVal := locale.GranularityLabel(a.globalGranularity)
	granStyle := globalFilterBoxStyle
	if a.globalFocus == 1 {
		granStyle = globalFilterBoxActiveStyle
	}
	granFilter := granStyle.Render(fmt.Sprintf("%s: < %s >", globalFilterLabelStyle.Render(locale.T("granularity.period")), globalFilterValueStyle.Render(granVal)))

	// Contributor filter
	logins := a.availableGlobalLogins()
	loginVal := logins[a.globalLoginIdx]
	loginStyle := globalFilterBoxStyle
	if a.globalFocus == 2 {
		loginStyle = globalFilterBoxActiveStyle
	}
	loginFilter := loginStyle.Render(fmt.Sprintf("%s: < %s >", globalFilterLabelStyle.Render(locale.T("tui.table.contributor")), globalFilterValueStyle.Render(loginVal)))

	bar := lipgloss.JoinHorizontal(lipgloss.Top, repoFilter, "  ", granFilter, "  ", loginFilter)

	if a.globalRepoExpanded && a.globalFocus == 0 {
		// Append repo list
		lines := []string{bar, ""}
		visible := a.visibleRepoIndices()

		for i, repoIdx := range visible {
			name := a.repoNames[repoIdx]
			mark := "[ ] "
			if a.globalRepoMulti == nil {
				mark = "[x] "
			} else if _, ok := a.globalRepoMulti[repoIdx]; ok {
				mark = "[x] "
			}
			line := "  " + mark + name
			if i == a.globalRepoCursor {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("> " + mark + name)
			}
			lines = append(lines, line)
		}
		if len(visible) == 0 {
			lines = append(lines, "  (No relevant repositories)")
		}
		return strings.Join(lines, "\n")
	} else if a.globalLoginExpanded && a.globalFocus == 2 {
		// Append login search list
		lines := []string{bar, ""}
		lines = append(lines, fmt.Sprintf("  Search: [ %s_ ]", a.globalLoginSearch))
		filtered := a.filteredLogins()
		if len(filtered) == 0 {
			lines = append(lines, "  (No matches)")
		} else {
			// Calculate visible range to avoid making the screen too tall
			start := 0
			if a.globalLoginCursor >= 10 {
				start = a.globalLoginCursor - 9
			}
			end := start + 10
			if end > len(filtered) {
				end = len(filtered)
			}
			for i := start; i < end; i++ {
				l := filtered[i]
				if i == a.globalLoginCursor {
					lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("> "+l))
				} else {
					lines = append(lines, "  "+l)
				}
			}
			if end < len(filtered) {
				lines = append(lines, fmt.Sprintf("  ... and %d more", len(filtered)-end))
			}
		}
		return strings.Join(lines, "\n")
	}

	return bar
}

// trendFilteredStats 趋势图当前统计范围：默认全仓库；多选为 globalRepoMulti 内仓库的并集。
func trendFilteredStats(a *App) []*cache.StatsData {
	n := len(a.stats)
	if n == 0 {
		return nil
	}
	if a.globalRepoMulti == nil {
		return a.stats
	}
	var out []*cache.StatsData
	for i := 0; i < n; i++ {
		if _, ok := a.globalRepoMulti[i]; ok {
			out = append(out, a.stats[i])
		}
	}
	if len(out) == 0 {
		return a.stats
	}
	return out
}

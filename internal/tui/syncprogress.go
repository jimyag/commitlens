package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	isync "github.com/jimyag/commitlens/internal/sync"
)

// syncProgressModel is a short-lived bubbletea program shown during sync.
type syncProgressModel struct {
	repos    []string
	states   map[string]*repoSyncState
	bars     map[string]progress.Model
	progCh   <-chan isync.Progress
	allDone  bool
	width    int
}

type repoSyncState struct {
	prsFetched int
	prsTotal   int
	done       bool
	err        error
}

// progressTickMsg carries one progress event from the sync channel.
type progressTickMsg struct {
	p  isync.Progress
	ok bool
}

func waitForProgress(ch <-chan isync.Progress) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		return progressTickMsg{p: p, ok: ok}
	}
}

func newSyncProgressModel(repos []string, ch <-chan isync.Progress) syncProgressModel {
	states := make(map[string]*repoSyncState, len(repos))
	bars := make(map[string]progress.Model, len(repos))
	for _, r := range repos {
		states[r] = &repoSyncState{prsTotal: -1}
		bars[r] = progress.New(
			progress.WithGradient("#5A56E0", "#EE6FF8"),
			progress.WithWidth(40),
			progress.WithoutPercentage(),
		)
	}
	return syncProgressModel{
		repos:  repos,
		states: states,
		bars:   bars,
		progCh: ch,
		width:  80,
	}
}

func (m syncProgressModel) Init() tea.Cmd {
	return waitForProgress(m.progCh)
}

func (m syncProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		barWidth := m.width - 36
		if barWidth < 20 {
			barWidth = 20
		}
		for r, bar := range m.bars {
			bar.Width = barWidth
			m.bars[r] = bar
		}

	case progressTickMsg:
		if !msg.ok {
			// Channel closed → all done
			m.allDone = true
			return m, tea.Quit
		}
		p := msg.p
		if st, ok := m.states[p.Repo]; ok {
			if p.PRsTotal > 0 {
				st.prsTotal = p.PRsTotal
			}
			if p.PRsFetched > 0 {
				st.prsFetched = p.PRsFetched
			}
			st.done = p.Done
			st.err = p.Err
		}
		// Check if all repos finished
		allDone := true
		for _, st := range m.states {
			if !st.done && st.err == nil {
				allDone = false
				break
			}
		}
		if allDone {
			m.allDone = true
			return m, tea.Quit
		}
		return m, waitForProgress(m.progCh)
	}
	return m, nil
}

var (
	labelStyle   = lipgloss.NewStyle().Width(30)
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m syncProgressModel) View() tea.View {
	var sb strings.Builder
	sb.WriteString("正在同步数据...\n\n")

	for _, repo := range m.repos {
		st := m.states[repo]
		bar := m.bars[repo]

		label := labelStyle.Render(repo)

		switch {
		case st.err != nil:
			sb.WriteString(fmt.Sprintf("  %s %s\n", label, errStyle.Render("失败: "+st.err.Error())))

		case st.done:
			sb.WriteString(fmt.Sprintf("  %s %s %s\n",
				label,
				bar.ViewAs(1.0),
				doneStyle.Render(fmt.Sprintf("完成 (%d PR)", st.prsFetched)),
			))

		case st.prsTotal > 0:
			pct := float64(st.prsFetched) / float64(st.prsTotal)
			sb.WriteString(fmt.Sprintf("  %s %s %s\n",
				label,
				bar.ViewAs(pct),
				fmt.Sprintf("%d/%d", st.prsFetched, st.prsTotal),
			))

		default:
			sb.WriteString(fmt.Sprintf("  %s %s\n",
				label,
				pendingStyle.Render("拉取 PR 列表..."),
			))
		}
	}

	return tea.NewView(sb.String())
}

// RunSyncProgress runs the sync progress TUI until all repos finish.
func RunSyncProgress(repos []string, ch <-chan isync.Progress) error {
	m := newSyncProgressModel(repos, ch)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

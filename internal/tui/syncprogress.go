package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/jimyag/commitlens/internal/locale"
	isync "github.com/jimyag/commitlens/internal/sync"
)

const (
	syncRepoW     = 38
	maxLogBuffer  = 48
	maxLogDisplay = 8
)

// syncProgressModel is a short-lived bubbletea program shown during sync.
type syncProgressModel struct {
	repos    []string
	states   map[string]*repoSyncState
	progCh   <-chan isync.Progress
	logLines []string // append-only execution log; older lines stay, new lines add
}

type repoSyncState struct {
	// prsTotal: 0 = no progress yet, -1 = listing PRs, >0 = detail total
	prsFetched int
	prsTotal   int
	listPage   int // GitHub list API page while prsTotal == -1
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

func (m *syncProgressModel) appendLog(line string) {
	if line == "" {
		return
	}
	n := len(m.logLines)
	if n > 0 && m.logLines[n-1] == line {
		return
	}
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > maxLogBuffer {
		m.logLines = m.logLines[len(m.logLines)-maxLogBuffer:]
	}
}

func newSyncProgressModel(repos []string, ch <-chan isync.Progress) *syncProgressModel {
	states := make(map[string]*repoSyncState, len(repos))
	for _, r := range repos {
		states[r] = &repoSyncState{prsTotal: 0, prsFetched: 0}
	}
	return &syncProgressModel{
		repos:  repos,
		states: states,
		progCh: ch,
	}
}

func (m *syncProgressModel) Init() tea.Cmd {
	return waitForProgress(m.progCh)
}

func (m *syncProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressTickMsg:
		if !msg.ok {
			return m, tea.Quit
		}
		p := msg.p
		if p.Log != "" {
			m.appendLog(fmt.Sprintf("%s  %s", p.Repo, p.Log))
		}
		if st, ok := m.states[p.Repo]; ok {
			st.err = p.Err
			st.done = p.Done
			if p.PRsTotal < 0 {
				st.prsTotal = -1
				st.prsFetched = p.PRsFetched
				if p.ListPage > 0 {
					st.listPage = p.ListPage
				}
			} else {
				st.prsTotal = p.PRsTotal
				st.prsFetched = p.PRsFetched
				st.listPage = 0
			}
		}
		allDone := true
		for _, st := range m.states {
			if !st.done && st.err == nil {
				allDone = false
				break
			}
		}
		if allDone {
			return m, tea.Quit
		}
		return m, waitForProgress(m.progCh)
	}
	return m, nil
}

func clipRepoName(name string) string {
	if len(name) <= syncRepoW {
		return name
	}
	return name[:syncRepoW-3] + "..."
}

func (m *syncProgressModel) View() tea.View {
	var sb strings.Builder
	sb.WriteString(locale.T("tui.sync.title"))
	if n := len(m.logLines); n > 0 {
		sb.WriteString(locale.T("tui.sync.logHeader") + "\n")
		start := 0
		if n > maxLogDisplay {
			start = n - maxLogDisplay
		}
		for _, line := range m.logLines[start:] {
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}
	for _, repo := range m.repos {
		st := m.states[repo]
		name := clipRepoName(repo)
		if st.err != nil {
			sb.WriteString(fmt.Sprintf("  %-38s  %s\n", name, fmt.Sprintf(locale.T("tui.sync.fail"), st.err.Error())))
			continue
		}
		if st.done {
			sb.WriteString(fmt.Sprintf("  %-38s  100%%  %s\n", name, fmt.Sprintf(locale.T("tui.sync.done"), st.prsFetched)))
			continue
		}
		var pctS string
		if st.prsTotal > 0 {
			p := 100 * st.prsFetched / st.prsTotal
			if p > 100 {
				p = 100
			}
			pctS = fmt.Sprintf("%3d%%", p)
		} else if st.prsTotal < 0 {
			// Listing: no total yet; show list page as monotonic progress (not a % of total).
			if st.listPage > 0 {
				pctS = fmt.Sprintf("p%3d", st.listPage)
			} else {
				pctS = "p  -"
			}
		} else {
			pctS = "  0%"
		}
		var rest string
		if st.prsTotal < 0 {
			rest = fmt.Sprintf(locale.T("tui.sync.listingN"), st.prsFetched)
		} else if st.prsTotal > 0 {
			rest = fmt.Sprintf("%d/%d", st.prsFetched, st.prsTotal)
		} else {
			rest = locale.T("tui.sync.fetch")
		}
		sb.WriteString(fmt.Sprintf("  %-38s  %5s  %s\n", name, pctS, rest))
	}
	return tea.NewView(sb.String())
}

// RunSyncProgress runs the sync progress TUI until all repos finish.
func RunSyncProgress(repos []string, ch <-chan isync.Progress) error {
	prog := newSyncProgressModel(repos, ch)
	p := tea.NewProgram(prog)
	_, err := p.Run()
	return err
}

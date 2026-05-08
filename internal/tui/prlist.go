package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jimyag/commitlens/internal/locale"
)

var (
	prListHeaderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	prListSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	prListTitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	prListHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	prListFilterBoxStyle      = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	prListFilterBoxActiveStyle = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))
	prListFilterLabelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	prListFilterValueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)

	prListColRepoStyle   = lipgloss.NewStyle().Width(15).PaddingRight(2)
	prListColPRStyle     = lipgloss.NewStyle().Width(8).PaddingRight(2)
	prListColTitleStyle  = lipgloss.NewStyle().Width(40).PaddingRight(2)
	prListColAuthorStyle = lipgloss.NewStyle().Width(15).PaddingRight(2)
	prListColDateStyle   = lipgloss.NewStyle().Width(12).PaddingRight(2)
	prListColLinesStyle  = lipgloss.NewStyle().Width(12).Align(lipgloss.Right)
)

func (a *App) renderPRList() string {
	periods := a.availablePRListPeriods()

	periodVal := periods[a.prListPeriodIdx]

	// Filters
	periodStyle := prListFilterBoxStyle
	if a.prListFocus == 0 {
		periodStyle = prListFilterBoxActiveStyle
	}
	periodFilter := periodStyle.Render(fmt.Sprintf("%s: < %s >", prListFilterLabelStyle.Render(locale.T("granularity.period")), prListFilterValueStyle.Render(periodVal)))

	header := periodFilter
	hint := prListHintStyle.Render(locale.T("tui.prlist.hint"))

	if len(a.prListPRs) == 0 {
		return header + "\n" + hint + "\n\n" + locale.T("tui.prlist.empty")
	}

	// Table header
	multiRepo := (a.globalRepoMulti != nil && len(a.globalRepoMulti) > 1) || len(a.repoNames) > 1

	cols := []string{}
	if multiRepo {
		cols = append(cols, prListColRepoStyle.Bold(true).Render(locale.T("tui.prlist.col.repo")))
	}
	cols = append(cols, prListColPRStyle.Bold(true).Render(locale.T("tui.prlist.col.pr")))
	cols = append(cols, prListColTitleStyle.Bold(true).Render(locale.T("tui.prlist.col.title")))
	cols = append(cols, prListColAuthorStyle.Bold(true).Render(locale.T("tui.prlist.col.author")))
	cols = append(cols, prListColDateStyle.Bold(true).Render(locale.T("tui.prlist.col.date")))
	cols = append(cols, prListColLinesStyle.Bold(true).Render(locale.T("tui.prlist.col.lines")))

	tableHeaderLine := "  " + strings.Join(cols, "")

	var lines []string
	lines = append(lines, header, hint, "", tableHeaderLine)

	// Calculate visible range
	visibleHeight := a.height - 10 // Header, filters, hint, status, etc.
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	start := 0
	if a.prListCursor >= visibleHeight {
		start = a.prListCursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(a.prListPRs) {
		end = len(a.prListPRs)
	}

	for i := start; i < end; i++ {
		pr := a.prListPRs[i]
		var row strings.Builder

		if multiRepo {
			row.WriteString(prListColRepoStyle.Render(ansi.Truncate(pr.Repo, 13, "...")))
		}
		row.WriteString(prListColPRStyle.Render(fmt.Sprintf("#%d", pr.Number)))
		row.WriteString(prListColTitleStyle.Render(ansi.Truncate(pr.Title, 38, "...")))

		authors := strings.Join(pr.Participants, ", ")
		row.WriteString(prListColAuthorStyle.Render(ansi.Truncate(authors, 13, "...")))

		row.WriteString(prListColDateStyle.Render(pr.MergedAt.Format("2006-01-02")))

		linesText := fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("+%d", pr.Additions)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("-%d", pr.Deletions)),
		)
		row.WriteString(prListColLinesStyle.Render(linesText))

		if i == a.prListCursor && a.prListFocus == 1 {
			lines = append(lines, prListSelectedStyle.Render("> "+row.String()))
		} else {
			lines = append(lines, "  "+row.String())
		}
	}

	return strings.Join(lines, "\n")
}

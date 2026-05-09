package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jimyag/commitlens/internal/locale"
)

var (
	commitListSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	commitListHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	commitListFilterBoxStyle       = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	commitListFilterBoxActiveStyle = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))
	commitListFilterLabelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	commitListFilterValueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)

	// Use fixed Width and MaxWidth to prevent wrapping and ensure alignment, especially with CJK.
	commitListColRepoStyle   = lipgloss.NewStyle().Width(15).MaxWidth(15).PaddingRight(2)
	commitListColPRStyle     = lipgloss.NewStyle().Width(12).MaxWidth(12).PaddingRight(2)
	commitListColTitleStyle  = lipgloss.NewStyle().Width(40).MaxWidth(40).PaddingRight(2)
	commitListColAuthorStyle = lipgloss.NewStyle().Width(15).MaxWidth(15).PaddingRight(2)
	commitListColDateStyle   = lipgloss.NewStyle().Width(12).MaxWidth(12).PaddingRight(2)
	commitListColLinesStyle  = lipgloss.NewStyle().Width(12).MaxWidth(12).Align(lipgloss.Right)
)

func (a *App) renderCommitList() string {
	periods := a.availableCommitListPeriods()

	periodVal := periods[a.commitListPeriodIdx]

	// Filters
	periodStyle := commitListFilterBoxStyle
	if a.commitListFocus == 0 {
		periodStyle = commitListFilterBoxActiveStyle
	}
	periodFilter := periodStyle.Render(fmt.Sprintf("%s: < %s >", commitListFilterLabelStyle.Render(locale.T("granularity.period")), commitListFilterValueStyle.Render(periodVal)))

	header := periodFilter
	hint := commitListHintStyle.Render(locale.T("tui.prlist.hint"))

	if len(a.commitList) == 0 {
		return header + "\n" + hint + "\n\n" + locale.T("tui.prlist.empty")
	}

	// Table header
	multiRepo := len(a.globalRepoMulti) > 1 || len(a.repoNames) > 1

	cols := []string{}
	if multiRepo {
		cols = append(cols, commitListColRepoStyle.Bold(true).Render(locale.T("tui.prlist.col.repo")))
	}
	cols = append(cols, commitListColPRStyle.Bold(true).Render(locale.T("tui.prlist.col.pr"))) // Map to "ID" or "SHA"
	cols = append(cols, commitListColTitleStyle.Bold(true).Render(locale.T("tui.prlist.col.title")))
	cols = append(cols, commitListColAuthorStyle.Bold(true).Render(locale.T("tui.prlist.col.author")))
	cols = append(cols, commitListColDateStyle.Bold(true).Render(locale.T("tui.prlist.col.date")))
	cols = append(cols, commitListColLinesStyle.Bold(true).Render(locale.T("tui.prlist.col.lines")))

	tableHeaderLine := "  " + strings.Join(cols, "")

	var lines []string
	lines = append(lines, header, hint, "", tableHeaderLine)

	// Calculate visible range
	visibleHeight := a.height - 10 // Header, filters, hint, status, etc.
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	start := 0
	if a.commitListCursor >= visibleHeight {
		start = a.commitListCursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(a.commitList) {
		end = len(a.commitList)
	}

	for i := start; i < end; i++ {
		c := a.commitList[i]
		var row strings.Builder

		if multiRepo {
			row.WriteString(commitListColRepoStyle.Render(ansi.Truncate(c.Repo, 13, "...")))
		}

		shortSHA := c.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		// Wrap SHA in brackets. Width is 12, content is "[1234567]" (9 chars). Padding is 2.
		// Total content width 11. 11 < 12, so it fits.
		row.WriteString(commitListColPRStyle.Render(fmt.Sprintf("[%s]", shortSHA)))

		// Sanitize title to remove potential newlines/control characters
		title := strings.Map(func(r rune) rune {
			if r < 32 {
				return -1
			}
			return r
		}, c.Title)
		row.WriteString(commitListColTitleStyle.Render(ansi.Truncate(title, 38, "...")))

		authors := strings.Join(c.Participants, ", ")
		row.WriteString(commitListColAuthorStyle.Render(ansi.Truncate(authors, 13, "...")))

		row.WriteString(commitListColDateStyle.Render(c.Date.Format("2006-01-02")))

		linesText := fmt.Sprintf(
			"%s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("+%d", c.Additions)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("-%d", c.Deletions)),
		)
		row.WriteString(commitListColLinesStyle.Render(linesText))

		if i == a.commitListCursor && a.commitListFocus == 1 {
			lines = append(lines, commitListSelectedStyle.Render("> "+row.String()))
		} else {
			lines = append(lines, "  "+row.String())
		}
	}

	return strings.Join(lines, "\n")
}

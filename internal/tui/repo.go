package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/locale"
)

func renderRepoView(a *App) string {
	if len(a.repoNames) == 0 {
		return locale.T("tui.reposelect.noconfig")
	}

	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	repoList := make([]string, len(a.repoNames))
	for i, r := range a.repoNames {
		if i == a.selectedRepo {
			repoList[i] = selectedStyle.Render("> " + r)
		} else {
			repoList[i] = "  " + r
		}
	}
	left := strings.Join(repoList, "\n")

	right := locale.T("tui.reposelect.nodata")
	if a.selectedRepo < len(a.stats) {
		s := a.stats[a.selectedRepo]
		contributors := sortedContributors(s.Contributors)
		right = fmt.Sprintf("%s\n%s", a.repoNames[a.selectedRepo], renderContributorTable(contributors))
	}

	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	var sb strings.Builder
	for i := 0; i < maxLines; i++ {
		l, r := "", ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		sb.WriteString(fmt.Sprintf("%-30s  %s\n", l, r))
	}
	return sb.String()
}

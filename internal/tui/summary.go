package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/locale"
)

func renderSummaryView(a *App) string {
	merged := make(map[string]*cache.ContributorStats)
	for _, s := range a.stats {
		for login, c := range s.Contributors {
			if existing, ok := merged[login]; ok {
				existing.PRCount += c.PRCount
				existing.CommitCount += c.CommitCount
				existing.Additions += c.Additions
				existing.Deletions += c.Deletions
			} else {
				cp := *c
				merged[login] = &cp
			}
		}
	}

	contributors := sortedContributors(merged)
	return renderContributorTable(contributors)
}

func sortedContributors(m map[string]*cache.ContributorStats) []*cache.ContributorStats {
	list := make([]*cache.ContributorStats, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].PRCount > list[j].PRCount
	})
	return list
}

func renderContributorTable(contributors []*cache.ContributorStats) string {
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	header := headerStyle.Render(
		fmt.Sprintf("%-22s %6s %8s %9s %9s",
			locale.T("tui.table.contributor"),
			locale.T("tui.table.pr"),
			locale.T("tui.table.commits"),
			locale.T("tui.table.added"),
			locale.T("tui.table.deleted"),
		),
	)
	sep := strings.Repeat("─", 56)
	rows := []string{header, sep}

	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	for _, c := range contributors {
		row := fmt.Sprintf("%-20s %6d %8d %s %s",
			c.Login,
			c.PRCount,
			c.CommitCount,
			addStyle.Render(fmt.Sprintf("%+9d", c.Additions)),
			delStyle.Render(fmt.Sprintf("%+9d", -c.Deletions)),
		)
		rows = append(rows, row)
	}
	if len(contributors) == 0 {
		rows = append(rows, locale.T("tui.table.nodata"))
	}
	return strings.Join(rows, "\n")
}

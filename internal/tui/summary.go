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
	selectedLogins := a.availableGlobalLogins()
	var filterLogin string
	if a.globalLoginIdx > 0 && a.globalLoginIdx < len(selectedLogins) {
		filterLogin = selectedLogins[a.globalLoginIdx]
	}

	for _, s := range trendFilteredStats(a) {
		for login, c := range s.Contributors {
			if filterLogin != "" && login != filterLogin {
				continue
			}
			if existing, ok := merged[login]; ok {
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
	return renderContributorTableWithWidth(contributors, a.width)
}

func sortedContributors(m map[string]*cache.ContributorStats) []*cache.ContributorStats {
	list := make([]*cache.ContributorStats, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].CommitCount != list[j].CommitCount {
			return list[i].CommitCount > list[j].CommitCount
		}
		return list[i].Login < list[j].Login
	})
	return list
}

func renderContributorTable(contributors []*cache.ContributorStats) string {
	return renderContributorTableWithWidth(contributors, 0)
}

func renderContributorTableWithWidth(contributors []*cache.ContributorStats, width int) string {
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	baseHeader := fmt.Sprintf(
		"%-22s %8s %9s %9s",
		locale.T("tui.table.contributor"),
		locale.T("tui.table.commits"),
		locale.T("tui.table.added"),
		locale.T("tui.table.deleted"),
	)
	barWidth := width - 52
	showActivity := barWidth >= 16 && len(contributors) > 0
	header := baseHeader
	if showActivity {
		header += "  " + locale.T("tui.table.activity")
	}
	header = headerStyle.Render(header)

	tableWidth := 50
	if showActivity {
		tableWidth = width
	}
	sep := strings.Repeat("─", tableWidth)
	rows := []string{header, sep}

	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	activityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	activityTrackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	maxCommits := 0
	for _, contributor := range contributors {
		if contributor.CommitCount > maxCommits {
			maxCommits = contributor.CommitCount
		}
	}

	for _, c := range contributors {
		row := fmt.Sprintf(
			"%-20s %8d %s %s",
			c.Login,
			c.CommitCount,
			addStyle.Render(fmt.Sprintf("%+9d", c.Additions)),
			delStyle.Render(fmt.Sprintf("%+9d", -c.Deletions)),
		)
		if showActivity {
			filled := 0
			if maxCommits > 0 {
				filled = c.CommitCount * barWidth / maxCommits
			}
			filled = clamp(filled, 0, barWidth)
			if c.CommitCount > 0 && filled == 0 {
				filled = 1
			}
			row += "  " +
				activityStyle.Render(strings.Repeat("█", filled)) +
				activityTrackStyle.Render(strings.Repeat("░", barWidth-filled))
		}
		rows = append(rows, row)
	}
	if len(contributors) == 0 {
		rows = append(rows, locale.T("tui.table.nodata"))
	}
	return strings.Join(rows, "\n")
}

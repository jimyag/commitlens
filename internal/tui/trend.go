package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/linechart/streamlinechart"
	"github.com/charmbracelet/lipgloss"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/stats"
)

func renderTrendView(a *App) string {
	granLabel := fmt.Sprintf("粒度: [%s]  ←→切换", granularityLabels[a.granularity])

	periodData := aggregatePeriods(a)
	periods := sortedPeriodKeys(periodData)

	if len(periods) == 0 {
		return granLabel + "\n\n暂无数据，按 r 刷新"
	}

	chartWidth := 60
	if a.width > 25 {
		chartWidth = a.width - 25
	}
	if chartWidth < 20 {
		chartWidth = 20
	}

	// 全仓库总 PR 折线图
	totalValues := make([]float64, len(periods))
	for i, p := range periods {
		totalValues[i] = float64(periodData[p].total)
	}
	totalChart := renderLineChart(totalValues, chartWidth, 6)
	xAxis := buildXAxis(periods, chartWidth)

	// 贡献者列表
	allLogins := allContributorLogins(a.stats)
	sort.Strings(allLogins)

	var selectedLogin string
	if len(allLogins) > 0 && a.selectedContributor < len(allLogins) {
		selectedLogin = allLogins[a.selectedContributor]
	}

	sel := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	loginList := make([]string, len(allLogins))
	for i, l := range allLogins {
		if i == a.selectedContributor {
			loginList[i] = sel.Render("> " + l)
		} else {
			loginList[i] = "  " + l
		}
	}

	// 个人折线图
	var personSection string
	if selectedLogin != "" {
		personValues := make([]float64, len(periods))
		for i, p := range periods {
			personValues[i] = float64(periodData[p].byContributor[selectedLogin])
		}
		personChart := renderLineChart(personValues, chartWidth, 5)
		personSection = fmt.Sprintf("\n%s PR 趋势\n%s\n%s", selectedLogin, personChart, xAxis)
	}

	return strings.Join([]string{
		granLabel,
		"",
		"全仓库合并 PR 趋势",
		totalChart,
		xAxis,
		"",
		"贡献者 (↑↓ 选择):",
		strings.Join(loginList, "  "),
		personSection,
	}, "\n")
}

func renderLineChart(values []float64, width, height int) string {
	chart := streamlinechart.New(width, height)
	for _, v := range values {
		chart.Push(v)
	}
	chart.Draw()
	return chart.View()
}

func buildXAxis(periods []string, width int) string {
	if len(periods) == 0 {
		return ""
	}
	step := 1
	if len(periods) > width/8 {
		step = len(periods)/(width/8) + 1
	}
	var labels []string
	for i, p := range periods {
		if i%step == 0 {
			if len(p) > 7 {
				p = p[len(p)-7:]
			}
			labels = append(labels, p)
		}
	}
	return strings.Join(labels, "  ")
}

type periodEntry struct {
	total         int
	byContributor map[string]int
}

func aggregatePeriods(a *App) map[string]*periodEntry {
	result := make(map[string]*periodEntry)
	for _, s := range a.stats {
		for weekKey, w := range s.Weekly {
			period := toPeriodKey(weekKey, a.granularity)
			if _, ok := result[period]; !ok {
				result[period] = &periodEntry{byContributor: make(map[string]int)}
			}
			result[period].total += w.TotalPRs
			for login, count := range w.Contributors {
				result[period].byContributor[login] += count
			}
		}
	}
	return result
}

// toPeriodKey converts a "YYYY-Www" week key to the requested granularity.
// It parses the week key to get an approximate date, then formats accordingly.
func toPeriodKey(weekKey string, granularity int) string {
	if granularity == 0 {
		return weekKey
	}
	// Parse "YYYY-Www" → approximate date using Jan 1 + week offset
	var year, week int
	fmt.Sscanf(weekKey, "%d-W%d", &year, &week)
	// Jan 4 is always in week 1 per ISO 8601
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, -weekday+1)
	approx := monday.AddDate(0, 0, (week-1)*7)

	switch granularity {
	case 1:
		return stats.MonthKey(approx)
	case 2:
		return stats.QuarterKey(approx)
	case 3:
		return stats.YearKey(approx)
	}
	return weekKey
}

func sortedPeriodKeys(m map[string]*periodEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func allContributorLogins(statsData []*cache.StatsData) []string {
	seen := make(map[string]struct{})
	for _, s := range statsData {
		for login := range s.Contributors {
			seen[login] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for l := range seen {
		result = append(result, l)
	}
	return result
}

package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/locale"
	"github.com/jimyag/commitlens/internal/stats"
)

func renderTrendView(a *App) string {
	granLabel := locale.T("tui.trend.hint")

	periodData := aggregatePeriods(a)
	periods := sortedPeriodKeys(periodData)

	if len(periods) == 0 {
		return granLabel + "\n\n" + locale.T("tui.trend.nodata")
	}

	vpW := a.width
	if vpW < 1 {
		vpW = 80
	}
	// 周期多时加宽 barchart canvas，终端内用视口 + trendHScroll 横移查看
	chartWidth := trendChartCanvasW(len(periods), vpW)
	if n := len(periods); a.trendLastPeriodN != n {
		a.trendHScroll = 0
		a.trendLastPeriodN = n
	}

	// 合并 PR：竖向条形图（ntcharts barchart，同 examples/barchart/vertical）+ 各点 PR 数
	chartTitle := locale.T("tui.mergedPrTrend")
	if a.globalRepoMulti != nil {
		if len(a.globalRepoMulti) == 1 {
			for idx := range a.globalRepoMulti {
				chartTitle = fmt.Sprintf(locale.T("tui.repoPrTrend"), a.repoNames[idx])
			}
		} else {
			chartTitle = fmt.Sprintf(locale.T("tui.mergedNReposFmt"), len(a.globalRepoMulti))
		}
	}

	if a.globalFocus == 3 {
		chartTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("> " + chartTitle)
	}

	totalCommitsValues := make([]float64, len(periods))
	for i, p := range periods {
		totalCommitsValues[i] = float64(periodData[p].totalCommits)
	}
	selIdx := -1
	if a.globalFocus == 3 {
		selIdx = a.trendPeriodCursor
	}
	totalCommitsChart := renderBarChart(periods, totalCommitsValues, chartWidth, 12, false, selIdx, a.globalGranularity)

	// 如果选中了具体贡献者，显示该贡献者的趋势
	logins := a.availableGlobalLogins()
	var selectedLogin string
	if a.globalLoginIdx > 0 && a.globalLoginIdx < len(logins) {
		selectedLogin = logins[a.globalLoginIdx]
	}

	personBlock := ""
	if selectedLogin != "" {
		personValues := make([]float64, len(periods))
		for i, p := range periods {
			personValues[i] = float64(periodData[p].byContributor[selectedLogin])
		}
		pChart := renderBarChart(periods, personValues, chartWidth, 8, true, selIdx, a.globalGranularity)
		personBlock = "\n" + fmt.Sprintf(locale.T("tui.trend.personTitle"), selectedLogin) + "\n" + pChart
	}

	block1 := chartTitle + "\n" + totalCommitsChart + personBlock
	max1 := trendHScrollMaxForBlock(block1, vpW)
	maxOff := max1
	if a.trendHScroll > maxOff {
		a.trendHScroll = maxOff
	}

	clip1 := block1
	if maxOff > 0 {
		st := trendScrollStatusLine(a.trendHScroll, maxOff, vpW)
		clip1 = st + "\n" + clipViewHorizontal(block1, a.trendHScroll, vpW)
	}

	return strings.Join([]string{
		granLabel,
		"",
		clip1,
	}, "\n")
}

// 与 ntcharts examples/barchart/vertical 示例一致的轴/标签/色块风格。
var (
	barAxisStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
	barLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63")) // purple
	// 全仓柱色（红块）
	barValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Background(lipgloss.Color("9"))
	// 个人柱色（绿块），与示例中 Name2 系列一致
	barValueStylePerson = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("2"))
	// 选中柱色（蓝块）
	barValueStyleSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Background(lipgloss.Color("12"))
)

// trendChartCanvasW 周期多时让柱图逻辑宽度超过视口，便于横滚阅读；否则用视口宽即可。
func trendChartCanvasW(periodN, viewW int) int {
	if viewW < 1 {
		viewW = 80
	}
	if periodN <= 0 {
		return viewW
	}
	g := 2
	if periodN > 1 {
		g = pickBarGap(periodN, 10000)
	}
	// 每柱至少约 2 格，列间 g，总宽可能大于视口
	minW := periodN*2 + (periodN-1)*g
	if minW < viewW {
		return viewW
	}
	return minW
}

func trendHScrollMaxForBlock(s string, viewportW int) int {
	if viewportW < 1 {
		return 0
	}
	maxW := 0
	for _, line := range strings.Split(s, "\n") {
		n := ansi.StringWidth(line)
		if n > maxW {
			maxW = n
		}
	}
	if maxW <= viewportW {
		return 0
	}
	return maxW - viewportW
}

func clipViewHorizontal(s string, offset, viewportW int) string {
	if viewportW < 1 {
		return s
	}
	// 按「终端列」裁剪，与 lipgloss.JoinHorizontal 的 StringWidth 一致，避免在 ANSI
	// 转义符中间截断，并防止右侧被误判为超宽、垫空格把整行撑到几千列而换行错乱。
	var b strings.Builder
	lines := strings.Split(s, "\n")
	right := offset + viewportW
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ansi.Cut(line, offset, right))
	}
	return b.String()
}

func trendScrollStatusLine(off, maxOff, viewportW int) string {
	if maxOff <= 0 {
		return ""
	}
	var parts []string
	if off > 0 {
		parts = append(parts, locale.T("tui.scroll.left"))
	}
	if off < maxOff {
		parts = append(parts, locale.T("tui.scroll.right"))
	}
	side := ""
	if len(parts) > 0 {
		sep := "，"
		o, c := "（", "）"
		if locale.Current() == locale.En {
			sep = ", "
			o, c = "(", ")"
		}
		side = o + strings.Join(parts, sep) + c
	}
	line := fmt.Sprintf(locale.T("tui.scroll.status"), off, maxOff, side)
	if ansi.StringWidth(line) > viewportW && viewportW > 8 {
		line = ansi.Truncate(line, viewportW, "")
	}
	return line
}

// pickBarGap 在终端宽度下尽量为列与列之间留出间隔：优先 2 格，再 1 格，不得已才 0。
func pickBarGap(n, chartW int) int {
	if n <= 1 {
		return 0
	}
	for _, g := range []int{2, 1} {
		gTotal := (n - 1) * g
		if chartW <= gTotal {
			continue
		}
		if (chartW-gTotal)/n >= 1 {
			return g
		}
	}
	return 0
}

// centerInWidth 将数字在「柱宽」内居中，过长则取右侧截断，保证与 barchart 列对齐。
func centerInWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) > w {
		if w == 1 {
			return s[len(s)-1:]
		}
		return s[len(s)-w:]
	}
	pad := w - len(s)
	l := pad / 2
	r := pad - l
	return strings.Repeat(" ", l) + s + strings.Repeat(" ", r)
}

// prValuesLineAboveChart 与 ntcharts 柱宽、柱间空隙一致，使数字与柱顶对齐。
func prValuesLineAboveChart(values []float64, cellW, gap int) string {
	n := len(values)
	if n == 0 {
		return ""
	}
	if cellW < 1 {
		cellW = 1
	}
	var b strings.Builder
	for i, v := range values {
		s := strconv.Itoa(int(math.Round(v)))
		b.WriteString(centerInWidth(s, cellW))
		if i < n-1 {
			b.WriteString(strings.Repeat(" ", gap))
		}
	}
	return b.String()
}

func estimateBarWidth(n, chartW, gap int) int {
	if n <= 0 {
		return 1
	}
	g := (n - 1) * gap
	if chartW <= g {
		return 1
	}
	return (chartW - g) / n
}

func formatVerticalLabel(period string, gran int, barW int) []string {
	if gran == 0 {
		var year, week int
		if n, _ := fmt.Sscanf(period, "%d-W%d", &year, &week); n == 2 {
			jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
			weekday := int(jan4.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			monday := jan4.AddDate(0, 0, -weekday+1)
			start := monday.AddDate(0, 0, (week-1)*7)
			end := start.AddDate(0, 0, 6)

			if barW >= 5 {
				return []string{
					fmt.Sprintf("%04d", year),
					start.Format("01-02"),
					"~",
					end.Format("01-02"),
				}
			} else if barW >= 2 {
				return []string{
					fmt.Sprintf("%02d", year/100),
					fmt.Sprintf("%02d", year%100),
					start.Format("01"),
					start.Format("02"),
					"~",
					end.Format("01"),
					end.Format("02"),
				}
			} else {
				s := fmt.Sprintf("W%02d", week)
				var lines []string
				for _, c := range s {
					lines = append(lines, string(c))
				}
				return lines
			}
		}
	}
	
	if barW >= len(period) {
		return []string{period}
	} else if barW >= 4 && len(period) == 7 { 
		return []string{period[:4], period[5:]}
	} else if barW >= 2 {
		var lines []string
		for i := 0; i < len(period); i += barW {
			e := i + barW
			if e > len(period) {
				e = len(period)
			}
			lines = append(lines, period[i:e])
		}
		return lines
	} else {
		var lines []string
		for _, c := range period {
			lines = append(lines, string(c))
		}
		return lines
	}
}

func buildCustomLabels(periods []string, gran int, barW, barGap int) string {
	var cols [][]string
	maxLines := 0
	for _, p := range periods {
		lines := formatVerticalLabel(p, gran, barW)
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
		cols = append(cols, lines)
	}

	var sb strings.Builder
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		for pIdx, col := range cols {
			s := ""
			if lineIdx < len(col) {
				s = col[lineIdx]
			}
			s = centerInWidth(s, barW)
			styled := barLabelStyle.Render(s)
			sb.WriteString(styled)
			if pIdx < len(cols)-1 {
				sb.WriteString(strings.Repeat(" ", barGap))
			}
		}
		if lineIdx < maxLines-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func barDataFromPeriods(periods []string, values []float64, width, gap int, person bool, selIdx int) []barchart.BarData {
	n := len(periods)
	block := barValueStyle
	if person {
		block = barValueStylePerson
	}
	out := make([]barchart.BarData, 0, n)
	for i := 0; i < n; i++ {
		style := block
		if i == selIdx {
			style = barValueStyleSelected
		}
		out = append(out, barchart.BarData{
			Label: " ",
			Values: []barchart.BarValue{
				{Name: locale.T("tui.barchart.pr"), Value: values[i], Style: style},
			},
		})
	}
	return out
}

func renderBarChart(periods []string, values []float64, width, height int, person bool, selIdx int, gran int) string {
	n := len(values)
	if n == 0 {
		return ""
	}
	gap := pickBarGap(n, width)
	data := barDataFromPeriods(periods, values, width, gap, person, selIdx)
	m := barchart.New(width, height,
		barchart.WithDataSet(data),
		barchart.WithStyles(barAxisStyle, barLabelStyle),
		barchart.WithBarGap(gap),
	)
	m.Draw()
	
	view := m.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	cellW := m.BarWidth()
	if cellW < 1 {
		cellW = 1
	}
	g := m.BarGap()
	
	customLabels := buildCustomLabels(periods, gran, cellW, g)
	lines = append(lines, customLabels)

	topLine := prValuesLineAboveChart(values, cellW, g)
	return locale.T("tui.bar.topPrCount") + "\n" + topLine + "\n" + strings.Join(lines, "\n")
}

type periodEntry struct {
	totalCommits  int
	byContributor map[string]int
}

func aggregatePeriods(a *App) map[string]*periodEntry {
	result := make(map[string]*periodEntry)
	for _, s := range trendFilteredStats(a) {
		for weekKey, w := range s.Weekly {
			period := toPeriodKey(weekKey, a.globalGranularity)
			if _, ok := result[period]; !ok {
				result[period] = &periodEntry{byContributor: make(map[string]int)}
			}
			result[period].totalCommits += w.TotalCommits
			for login, stats := range w.Contributors {
				result[period].byContributor[login] += stats.Commits
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

// contributorsSortedByCommitCount 按各登录在全仓库合并后的 PR 总数降序；同 PR 数时按登录名升序，顺序稳定。
func contributorsSortedByCommitCount(statsData []*cache.StatsData) []string {
	merged := make(map[string]int)
	for _, s := range statsData {
		for login, c := range s.Contributors {
			merged[login] += c.CommitCount
		}
	}
	type row struct {
		login string
		prs   int
	}
	rows := make([]row, 0, len(merged))
	for login, n := range merged {
		rows = append(rows, row{login: login, prs: n})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].prs != rows[j].prs {
			return rows[i].prs > rows[j].prs
		}
		return rows[i].login < rows[j].login
	})
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.login)
	}
	return out
}

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
	"github.com/jimyag/commitlens/internal/stats"
)

func renderTrendView(a *App) string {
	normalizeTrendState(a)

	scopeLine := formatTrendScopeLine(a)
	granLabel := fmt.Sprintf("范围: %s  粒度: [%s]  ← →\n,. 切焦点(仓库/贡献)  m 单仓|多选  多选+空格  ↑↓\n柱图区宽可横移 shift+←/→  < >  Home End", scopeLine, granularityLabels[a.granularity])

	periodData := aggregatePeriods(a)
	periods := sortedPeriodKeys(periodData)

	if len(periods) == 0 {
		return granLabel + "\n\n暂无数据，按 r 刷新"
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
	chartTitle := "合并 PR 趋势"
	if a.nRepos() > 0 {
		if a.trendSelectMulti {
			chartTitle = fmt.Sprintf("已选 %d 个仓库 合并 PR 趋势", countTrendSelectedRepos(a))
		} else {
			chartTitle = fmt.Sprintf("%s  合并 PR 趋势", a.repoNames[a.trendOneRepo])
		}
	}
	totalValues := make([]float64, len(periods))
	for i, p := range periods {
		totalValues[i] = float64(periodData[p].total)
	}
	totalChart := renderBarChart(periods, totalValues, chartWidth, 8, false)

	// 贡献者：当前范围内按 PR 数降序
	allLogins := contributorLoginsForTrend(a)

	var selectedLogin string
	if len(allLogins) > 0 && a.selectedContributor < len(allLogins) {
		selectedLogin = allLogins[a.selectedContributor]
	}

	// 贡献者列表：竖向排列，高亮当前选中项
	sel := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	var loginLines []string
	for i, l := range allLogins {
		if i == a.selectedContributor {
			loginLines = append(loginLines, sel.Render("> "+l))
		} else {
			loginLines = append(loginLines, "  "+l)
		}
	}

	// 左右分栏：左贡献者列表、右个人 PR 柱图（同一行，宽度见 trendContributorLayout）
	leftW, rightW, gutter := trendContributorLayout(vpW)
	personBlock := ""
	if selectedLogin != "" {
		personValues := make([]float64, len(periods))
		for i, p := range periods {
			personValues[i] = float64(periodData[p].byContributor[selectedLogin])
		}
		personCanvasW := trendChartCanvasW(len(periods), rightW)
		pChart := renderBarChart(periods, personValues, personCanvasW, 6, true)
		personBlock = selectedLogin + " PR 趋势\n" + pChart
	} else {
		personBlock = ""
	}

	block1 := chartTitle + "\n" + totalChart
	max1 := trendHScrollMaxForBlock(block1, vpW)
	max2 := 0
	if personBlock != "" {
		max2 = trendHScrollMaxForBlock(personBlock, rightW)
	}
	maxOff := max1
	if max2 > maxOff {
		maxOff = max2
	}
	if a.trendHScroll > maxOff {
		a.trendHScroll = maxOff
	}
	clip1 := block1
	if maxOff > 0 {
		st := trendScrollStatusLine(a.trendHScroll, maxOff, vpW)
		if max1 > 0 {
			clip1 = st + "\n" + clipViewHorizontal(block1, a.trendHScroll, vpW)
		} else {
			clip1 = st + "\n" + block1
		}
	}
	clipPerson := personBlock
	if personBlock != "" && maxOff > 0 {
		clipPerson = clipViewHorizontal(personBlock, a.trendHScroll, rightW)
	}

	contributorBlock := "贡献者 (↑↓) [. 焦点]:\n" + strings.Join(loginLines, "\n")
	leftCol := lipgloss.NewStyle().Width(leftW).Render(contributorBlock)
	var rightCol string
	if selectedLogin == "" {
		rightCol = lipgloss.NewStyle().Width(rightW).Foreground(lipgloss.Color("240")).Render("选择贡献者后显示个人 PR 柱图")
	} else {
		rightCol = clipPerson
	}
	gap := lipgloss.NewStyle().Width(gutter).Render(strings.Repeat(" ", gutter))
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, gap, rightCol)

	return strings.Join([]string{
		granLabel,
		"",
		renderRepoPanel(a),
		"",
		clip1,
		"",
		bottomRow,
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
)

// trendContributorLayout 将一行分为「贡献者 | 个人 PR 柱图」；gutter 为中间间隔列数。
func trendContributorLayout(vpW int) (leftW, rightW, gutter int) {
	gutter = 2
	if vpW < 1 {
		vpW = 80
	}
	leftW = 30
	if vpW < 64 {
		leftW = 22
	}
	rightW = vpW - leftW - gutter
	if rightW < 14 {
		rightW = 14
		leftW = vpW - gutter - rightW
		if leftW < 12 {
			leftW = 12
			rightW = vpW - leftW - gutter
			if rightW < 10 {
				rightW = 10
			}
		}
	}
	return leftW, rightW, gutter
}

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
		parts = append(parts, "左侧还有")
	}
	if off < maxOff {
		parts = append(parts, "右侧还有")
	}
	side := ""
	if len(parts) > 0 {
		side = "（" + strings.Join(parts, "，") + "）"
	}
	line := fmt.Sprintf("视口 %d/%d 列 %s shift+←→ < > Home End", off, maxOff, side)
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

func compactPeriod(p string) string {
	if p == "" {
		return "?"
	}
	if parts := strings.Split(p, "-"); len(parts) >= 2 {
		y := parts[0]
		if len(y) == 4 {
			return y[2:4] + parts[1]
		}
	}
	if len(p) > 5 {
		return p[:5]
	}
	return p
}

// xAxisLabel 横轴标签：柱变窄时缩短，库会再按 barWidth 截断。
func xAxisLabel(period string, i, n, barW int) string {
	if barW >= 7 {
		s := compactPeriod(period)
		if len(s) > barW {
			return s[:barW]
		}
		return s
	}
	if barW >= 4 {
		s := period
		if len(s) > barW {
			return s[len(s)-barW:]
		}
		return s
	}
	if barW >= 2 {
		s := fmt.Sprintf("%02d", i+1)
		if len(s) > barW {
			return s[:barW]
		}
		return s
	}
	return fmt.Sprintf("%d", i%10)
}

func barDataFromPeriods(periods []string, values []float64, width, gap int, person bool) []barchart.BarData {
	n := len(periods)
	bw := estimateBarWidth(n, width, gap)
	block := barValueStyle
	if person {
		block = barValueStylePerson
	}
	out := make([]barchart.BarData, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, barchart.BarData{
			Label: xAxisLabel(periods[i], i, n, bw),
			Values: []barchart.BarValue{
				{Name: "PR", Value: values[i], Style: block},
			},
		})
	}
	return out
}

// renderBarChart 参考 github.com/NimbleMarkets/ntcharts examples/barchart/vertical：柱顶一行展示具体 PR 数，列间距由 pickBarGap 控制。
func renderBarChart(periods []string, values []float64, width, height int, person bool) string {
	n := len(values)
	if n == 0 {
		return ""
	}
	gap := pickBarGap(n, width)
	data := barDataFromPeriods(periods, values, width, gap, person)
	m := barchart.New(width, height,
		barchart.WithDataSet(data),
		barchart.WithStyles(barAxisStyle, barLabelStyle),
		barchart.WithBarGap(gap),
	)
	m.Draw()
	cellW := m.BarWidth()
	if cellW < 1 {
		cellW = 1
	}
	g := m.BarGap()
	topLine := prValuesLineAboveChart(values, cellW, g)
	return "柱顶 PR 数\n" + topLine + "\n" + m.View()
}

func formatTrendScopeLine(a *App) string {
	if a.nRepos() == 0 {
		return "无仓库"
	}
	if !a.trendSelectMulti {
		return "单 " + a.repoNames[a.trendOneRepo]
	}
	return fmt.Sprintf("多 已选 %d 个", countTrendSelectedRepos(a))
}

func countTrendSelectedRepos(a *App) int {
	if !a.trendSelectMulti {
		return 1
	}
	if a.trendRepoMulti == nil {
		return 0
	}
	return len(a.trendRepoMulti)
}

// trendFilteredStats 趋势图当前统计范围：默认仅 trendOneRepo；多选为 trendRepoMulti 内仓库的并集。
func trendFilteredStats(a *App) []*cache.StatsData {
	n := len(a.stats)
	if n == 0 {
		return nil
	}
	if !a.trendSelectMulti {
		i := a.trendOneRepo
		if i < 0 {
			i = 0
		} else if i >= n {
			i = n - 1
		}
		return []*cache.StatsData{a.stats[i]}
	}
	if a.trendRepoMulti == nil {
		return []*cache.StatsData{a.stats[0]}
	}
	var out []*cache.StatsData
	for i := 0; i < n; i++ {
		if _, ok := a.trendRepoMulti[i]; ok {
			out = append(out, a.stats[i])
		}
	}
	if len(out) == 0 {
		return []*cache.StatsData{a.stats[0]}
	}
	return out
}

func contributorLoginsForTrend(a *App) []string {
	return contributorsSortedByPRCount(trendFilteredStats(a))
}

func normalizeTrendState(a *App) {
	n := a.nRepos()
	if n == 0 {
		a.selectedContributor = 0
		return
	}
	if a.trendOneRepo < 0 {
		a.trendOneRepo = 0
	} else if a.trendOneRepo > n-1 {
		a.trendOneRepo = n - 1
	}
	if a.trendRepoCursor < 0 {
		a.trendRepoCursor = 0
	} else if a.trendRepoCursor > n-1 {
		a.trendRepoCursor = n - 1
	}
	if a.trendSelectMulti {
		if a.trendRepoMulti == nil {
			a.trendRepoMulti = make(map[int]struct{})
		}
		if len(a.trendRepoMulti) == 0 {
			a.trendRepoMulti[a.trendOneRepo] = struct{}{}
		}
	}
	if a.trendListFocus != trendFocusRepo && a.trendListFocus != trendFocusContributors {
		a.trendListFocus = trendFocusRepo
	}
	m := len(contributorLoginsForTrend(a))
	if m == 0 {
		a.selectedContributor = 0
		return
	}
	if a.selectedContributor >= m {
		a.selectedContributor = m - 1
	}
	if a.selectedContributor < 0 {
		a.selectedContributor = 0
	}
}

func renderRepoPanel(a *App) string {
	n := a.nRepos()
	if n == 0 {
		return "仓库: (无配置)"
	}
	sel := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	mode := "单选  m→多选"
	if a.trendSelectMulti {
		mode = "多选  空格=勾选/取消(至少保留1个)  m→单选"
	}
	focus := "贡献者"
	if a.trendListFocus == trendFocusRepo {
		focus = "仓库"
	}
	header := fmt.Sprintf("仓库  [焦点:%s]  %s  [,]=仓库  [.]=贡献者  ↑↓=移动", focus, mode)
	lines := []string{header}
	for i, name := range a.repoNames {
		cursor := a.trendRepoCursor == i && a.trendListFocus == trendFocusRepo
		var mark string
		if a.trendSelectMulti {
			if a.trendRepoMulti != nil {
				if _, ok := a.trendRepoMulti[i]; ok {
					mark = "[x] "
				} else {
					mark = "[ ] "
				}
			}
		} else if i == a.trendOneRepo {
			mark = "• "
		} else {
			mark = "  "
		}
		if cursor {
			lines = append(lines, sel.Render("> "+mark+name))
		} else {
			lines = append(lines, "  "+mark+name)
		}
	}
	return strings.Join(lines, "\n")
}

type periodEntry struct {
	total         int
	byContributor map[string]int
}

func aggregatePeriods(a *App) map[string]*periodEntry {
	result := make(map[string]*periodEntry)
	for _, s := range trendFilteredStats(a) {
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

// contributorsSortedByPRCount 按各登录在全仓库合并后的 PR 总数降序；同 PR 数时按登录名升序，顺序稳定。
func contributorsSortedByPRCount(statsData []*cache.StatsData) []string {
	merged := make(map[string]int)
	for _, s := range statsData {
		for login, c := range s.Contributors {
			merged[login] += c.PRCount
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

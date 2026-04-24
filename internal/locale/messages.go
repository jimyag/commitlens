package locale

var bundles = map[Tag]map[string]string{
	En: enMessages,
	Zh: zhMessages,
}

var enMessages = map[string]string{
	"granularity.week":    "week",
	"granularity.month":   "month",
	"granularity.quarter": "quarter",
	"granularity.year":    "year",

	"tui.tab.summary":  "[Summary]",
	"tui.tab.repos":    "[Single repo]",
	"tui.tab.trend":    "[Trends]",
	"tui.header.hints": "        tab: switch  r: refresh  q: quit",

	"tui.status.syncing": "State: syncing...",
	"tui.status.error": "Error: %s",

	"tui.trend.hint": "Scope: %s  Step: [%s]  left/right\n,. focus repos|contributors  m single|multi  multi+space  up/down\nChart: wide pan shift+left/right  < >  Home End",
	"tui.trend.nodata": "No data, press r to refresh",

	"tui.mergedPrTrend":   "merged PR trend",
	"tui.mergedNReposFmt": "merged PR trend for %d selected repos",
	"tui.repoPrTrend":     "%s  merged PR trend",
	"tui.trend.contributorHeader":  "Contributors (↑↓) [. focus]:",
	"tui.trend.selectPersonHint":   "Select a contributor to see their PR bar chart",
	"tui.trend.personTitle":        "%s  merged PR trend",
	"tui.bar.topPrCount":            "PR count on top",
	"tui.barchart.pr":              "PR",

	"tui.scope.norepos":  "no repo",
	"tui.scope.single":   "one ",
	"tui.scope.multifmt":   "multi: %d selected",

	"tui.scroll.left":  "more on left",
	"tui.scroll.right": "more on right",
	"tui.scroll.status": "viewport %d/%d cols %s shift+arrows < > Home End",

	"tui.repoPanel.noconfig":     "repo: (none configured)",
	"tui.repoPanel.modeSingle":   "single  m: multi  space in multi: toggle",
	"tui.repoPanel.modeMulti":    "multi  space: on/off (keep ≥1)  m: single",
	"tui.focus.repo":             "repo",
	"tui.focus.contributor":     "contributors",
	"tui.repoPanel.header":       "repo  [focus: %s]  %s  ,=repo  .=contributors  up/down: move",

	"tui.table.contributor": "contributor",
	"tui.table.pr":         "PRs",
	"tui.table.commits":     "commits",
	"tui.table.added":      "+lines",
	"tui.table.deleted":    "−lines",
	"tui.table.nodata":     "  (no data)",

	"tui.reposelect.noconfig": "no repository configured",
	"tui.reposelect.nodata":   "no data",

	"tui.sync.title": "Syncing data...\n\n",
	"tui.sync.fail":  "failed: %s",
	"tui.sync.done":  "done (%d PRs)",
	"tui.sync.fetch": "fetching PR list...",
	"tui.sync.listing": "listing PRs…",
	"tui.sync.logHeader": "Execution log:",
	"tui.sync.listingN":  "list: %d PRs",
}

var zhMessages = map[string]string{
	"granularity.week":    "周",
	"granularity.month":   "月",
	"granularity.quarter": "季度",
	"granularity.year":    "年",

	"tui.tab.summary":  "[汇总]",
	"tui.tab.repos":    "[单仓库]",
	"tui.tab.trend":    "[趋势]",
	"tui.header.hints": "        tab:切换  r:刷新  q:退出",

	"tui.status.syncing": "状态: 同步中...",
	"tui.status.error": "错误: %s",

	"tui.trend.hint": "范围: %s  粒度: [%s]  ← →\n,. 切焦点(仓库/贡献)  m 单仓|多选  多选+空格  ↑↓\n柱图区宽可横移 shift+←/→  < >  Home End",
	"tui.trend.nodata": "暂无数据，按 r 刷新",

	"tui.mergedPrTrend":   "合并 PR 趋势",
	"tui.mergedNReposFmt": "已选 %d 个仓库 合并 PR 趋势",
	"tui.repoPrTrend":     "%s  合并 PR 趋势",
	"tui.trend.contributorHeader":  "贡献者 (↑↓) [. 焦点]:",
	"tui.trend.selectPersonHint":   "选择贡献者后显示个人 PR 柱图",
	"tui.trend.personTitle":        "%s PR 趋势",
	"tui.bar.topPrCount":            "柱顶 PR 数",
	"tui.barchart.pr":              "PR",

	"tui.scope.norepos":  "无仓库",
	"tui.scope.single":   "单 ",
	"tui.scope.multifmt":   "多 已选 %d 个",

	"tui.scroll.left":  "左侧还有",
	"tui.scroll.right": "右侧还有",
	"tui.scroll.status": "视口 %d/%d 列 %s shift+←→ < > Home End",

	"tui.repoPanel.noconfig":   "仓库: (无配置)",
	"tui.repoPanel.modeSingle":  "单选  m→多选",
	"tui.repoPanel.modeMulti":  "多选  空格=勾选/取消(至少保留1个)  m→单选",
	"tui.focus.repo":         "仓库",
	"tui.focus.contributor":  "贡献者",
	"tui.repoPanel.header":  "仓库  [焦点:%s]  %s  [,]=仓库  [.]=贡献者  ↑↓=移动",

	"tui.table.contributor": "贡献者",
	"tui.table.pr":         "PR数",
	"tui.table.commits":     "Commit数",
	"tui.table.added":      "新增行",
	"tui.table.deleted":    "删除行",
	"tui.table.nodata":     "  暂无数据",

	"tui.reposelect.noconfig": "无仓库配置",
	"tui.reposelect.nodata":  "无数据",

	"tui.sync.title":  "正在同步数据...\n\n",
	"tui.sync.fail":  "失败: %s",
	"tui.sync.done":  "完成 (%d PR)",
	"tui.sync.fetch": "拉取 PR 列表...",
	"tui.sync.listing": "正在列出 PR…",
	"tui.sync.logHeader": "执行日志：",
	"tui.sync.listingN":  "列表: %d 个 PR",
}

// GranularityLabel returns the label for TUI index 0..3 (week..year).
func GranularityLabel(i int) string {
	keys := []string{"granularity.week", "granularity.month", "granularity.quarter", "granularity.year"}
	if i < 0 || i >= len(keys) {
		return T("granularity.week")
	}
	return T(keys[i])
}

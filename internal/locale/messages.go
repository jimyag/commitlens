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
	"granularity.period":  "Period",

	"tui.tab.summary":  "[Summary]",
	"tui.tab.repos":    "[Single repo]",
	"tui.tab.trend":    "[Trends]",
	"tui.tab.prlist":   "[PRs]",
	"tui.header.hints": "        1-4/[]: tab  tab: focus  r: refresh  q: quit\nFilters: Enter: toggle/select  Space: multi select  ←/→: change",

	"tui.status.syncing": "State: syncing...",
	"tui.status.error": "Error: %s",

	"tui.trend.hint": "Scope: %s  Step: [%s]  ← →\ntab: focus cycle (repos|contributors|chart)  m: single|multi  space: toggle  ↑↓\nChart: Enter: PR list  shift+←/→: pan  < > Home End",
	"tui.trend.nodata": "No data, press r to refresh",

	"tui.prlist.title":  "PR List · %s",
	"tui.prlist.titleWithLogin": "PR List · %s · %s",
	"tui.prlist.hint":   "        tab: focus cycle  ←/→: change filter  ↑/↓: scroll table",
	"tui.prlist.filterAll":  "All",
	"tui.prlist.col.repo":   "Repo",
	"tui.prlist.col.pr":     "PR",
	"tui.prlist.col.title":  "Title",
	"tui.prlist.col.author": "Author",
	"tui.prlist.col.date":   "Merged At",
	"tui.prlist.col.lines":  "Lines",
	"tui.prlist.empty":      "No PRs found for this period.",

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
	"granularity.period":  "周期",

	"tui.tab.summary":  "[汇总]",
	"tui.tab.repos":    "[单仓库]",
	"tui.tab.trend":    "[趋势]",
	"tui.tab.prlist":   "[PR列表]",
	"tui.header.hints": "        1-4/[]:切换  tab:切焦点  r:刷新  q:退出\n过滤器: Enter:展开/选择  空格:多选  ←/→:切换",

	"tui.status.syncing": "状态: 同步中...",
	"tui.status.error": "错误: %s",

	"tui.trend.hint": "范围: %s  粒度: [%s]  ← →\ntab: 切焦点(仓库/贡献者/柱图)  m: 单仓|多选  空格: 勾选  ↑↓\n柱图: Enter: PR列表  shift+←/→: 横移  < > Home End",
	"tui.trend.nodata": "暂无数据，按 r 刷新",

	"tui.prlist.title":  "PR 列表 · %s",
	"tui.prlist.titleWithLogin": "PR 列表 · %s · %s",
	"tui.prlist.hint":   "        tab: 切焦点  ←/→: 切换筛选  ↑/↓: 滚动表格",
	"tui.prlist.filterAll":  "全部",
	"tui.prlist.col.repo":   "仓库",
	"tui.prlist.col.pr":     "PR",
	"tui.prlist.col.title":  "标题",
	"tui.prlist.col.author": "作者",
	"tui.prlist.col.date":   "合并时间",
	"tui.prlist.col.lines":  "增删行",
	"tui.prlist.empty":      "该时间段内暂无 PR。",

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

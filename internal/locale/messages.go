package locale

var bundles = map[Tag]map[string]string{
	En: {
		"granularity.week":    "week",
		"granularity.month":   "month",
		"granularity.quarter": "quarter",
		"granularity.year":    "year",
		"granularity.period":  "period",

		"tui.tab.summary":    "[summary]",
		"tui.tab.repos":      "[repos]",
		"tui.tab.trend":      "[commit trend]",
		"tui.tab.trendLines": "[lines trend]",
		"tui.tab.prlist":     "[commits]",
		"tui.header.hints":   "        1-5/[]:switch  tab:focus  r:sync  q:quit\nfilters: Enter:expand/select  Space:multi  ←/→:cycle",

		"tui.status.syncing": "status: syncing...",
		"tui.status.error":   "error: %s",

		"tui.trend.hint":   "scope: %s  gran: [%s]  ← →\ntab: focus(repo/user/chart)  space in user: toggle  ↑↓\nchart: Enter: list  shift+arrows < > Home End",
		"tui.trend.nodata": "no data, press r to sync",

		"tui.prlist.title":          "commits · %s",
		"tui.prlist.titleWithLogin": "commits · %s · %s",
		"tui.prlist.hint":           "        tab: focus  ←/→: filter  ↑/↓: scroll",
		"tui.prlist.filterAll":      "all",
		"tui.prlist.col.repo":       "repo",
		"tui.prlist.col.pr":         "sha",
		"tui.prlist.col.title":      "title",
		"tui.prlist.col.author":     "author",
		"tui.prlist.col.date":       "date",
		"tui.prlist.col.lines":      "lines",
		"tui.prlist.empty":          "no commits in this period.",

		"tui.mergedPrTrend":           "commit trend",
		"tui.mergedNReposFmt":         "selected %d repos commit trend",
		"tui.repoPrTrend":             "%s commit trend",
		"tui.trend.contributorHeader": "contributors (↑↓) [. focus]:",
		"tui.trend.selectPersonHint":  "select user to see personal trend",
		"tui.trend.personTitle":       "%s",
		"tui.trend.personTitleMulti":  "Selected Contributors",
		"tui.trend.linesAdd":          "Repo Additions (+)",
		"tui.trend.linesDel":          "Repo Deletions (-)",
		"filter.multiUsers":           "%d users",
		"tui.bar.topPrCount":          "values",
		"tui.barchart.pr":             "value",

		"tui.scope.norepos":  "no repo",
		"tui.scope.single":   "single ",
		"tui.scope.multifmt": "multi: %d selected",

		"tui.scroll.left":   "more on left",
		"tui.scroll.right":  "more on right",
		"tui.scroll.status": "viewport %d/%d cols %s shift+arrows < > Home End",

		"tui.repoPanel.noconfig":   "repo: (none configured)",
		"tui.repoPanel.modeSingle": "single  m: multi  space in multi: toggle",
		"tui.repoPanel.modeMulti":  "multi  space: on/off (keep ≥1)  m: single",
		"tui.focus.repo":           "repo",
		"tui.focus.contributor":    "contributors",
		"tui.repoPanel.header":     "repo  [focus: %s]  %s  ,=repo  .=contributors  up/down: move",

		"tui.table.contributor": "contributor",
		"tui.table.pr":          "Commits",
		"tui.table.commits":     "Commits",
		"tui.table.added":       "+lines",
		"tui.table.deleted":     "−lines",
		"tui.table.nodata":      "  (no data)",

		"tui.reposelect.noconfig": "no repository configured",
		"tui.reposelect.nodata":   "no data",

		"tui.sync.title":     "Syncing data...\n\n",
		"tui.sync.fail":      "failed: %s",
		"tui.sync.done":      "done (%d commits)",
		"tui.sync.fetch":     "fetching commit history...",
		"tui.sync.listing":   "extracting commits…",
		"tui.sync.logHeader": "Execution log:",
		"tui.sync.listingN":  "list: %d commits",
	},
	Zh: {
		"granularity.week":    "周",
		"granularity.month":   "月",
		"granularity.quarter": "季度",
		"granularity.year":    "年",
		"granularity.period":  "周期",

		"tui.tab.summary":    "[汇总]",
		"tui.tab.repos":      "[单仓库]",
		"tui.tab.trend":      "[提交趋势]",
		"tui.tab.trendLines": "[行数趋势]",
		"tui.tab.prlist":     "[提交列表]",
		"tui.header.hints":   "        1-5/[]:切换  tab:切焦点  r:刷新  q:退出\n过滤器: Enter:展开/选择  空格:多选  ←/→:切换",

		"tui.status.syncing": "状态: 同步中...",
		"tui.status.error":   "错误: %s",

		"tui.trend.hint":   "范围: %s  粒度: [%s]  ← →\ntab: 切焦点(仓库/贡献者/柱图)  m: 单仓|多选  空格: 勾选  ↑↓\n柱图: Enter: 提交列表  shift+←/→: 横移  < > Home End",
		"tui.trend.nodata": "暂无数据，按 r 刷新",

		"tui.prlist.title":          "提交记录 · %s",
		"tui.prlist.titleWithLogin": "提交记录 · %s · %s",
		"tui.prlist.hint":           "        tab: 切焦点  ←/→: 切换筛选  ↑/↓: 滚动表格",
		"tui.prlist.filterAll":      "全部",
		"tui.prlist.col.repo":       "仓库",
		"tui.prlist.col.pr":         "SHA",
		"tui.prlist.col.title":      "描述",
		"tui.prlist.col.author":     "作者",
		"tui.prlist.col.date":       "日期",
		"tui.prlist.col.lines":      "增删行",
		"tui.prlist.empty":          "该时间段内暂无提交记录。",

		"tui.mergedPrTrend":           "提交数趋势",
		"tui.mergedNReposFmt":         "已选 %d 个仓库 提交数趋势",
		"tui.repoPrTrend":             "%s  提交数趋势",
		"tui.trend.contributorHeader": "贡献者 (↑↓) [. 焦点]:",
		"tui.trend.selectPersonHint":  "选择贡献者后显示个人柱状图",
		"tui.trend.personTitle":       "%s",
		"tui.trend.personTitleMulti":  "已选贡献者合集",
		"tui.trend.linesAdd":          "全仓库新增行 (+)",
		"tui.trend.linesDel":          "全仓库删除行 (-)",
		"filter.multiUsers":           "%d 位用户",
		"tui.bar.topPrCount":          "数值",
		"tui.barchart.pr":             "数值",

		"tui.scope.norepos":  "无仓库",
		"tui.scope.single":   "单 ",
		"tui.scope.multifmt": "多 已选 %d 个",

		"tui.scroll.left":   "左侧还有",
		"tui.scroll.right":  "右侧还有",
		"tui.scroll.status": "视口 %d/%d 列 %s shift+←→ < > Home End",

		"tui.repoPanel.noconfig":   "仓库: (无配置)",
		"tui.repoPanel.modeSingle": "单选  m→多选",
		"tui.repoPanel.modeMulti":  "多选  空格=勾选/取消(至少保留1个)  m→单选",
		"tui.focus.repo":           "仓库",
		"tui.focus.contributor":    "贡献者",
		"tui.repoPanel.header":     "仓库  [焦点:%s]  %s  [,]=仓库  [.]=贡献者  ↑↓=移动",

		"tui.table.contributor": "贡献者",
		"tui.table.pr":          "提交数",
		"tui.table.commits":     "提交数",
		"tui.table.added":       "新增行",
		"tui.table.deleted":     "删除行",
		"tui.table.nodata":      "  暂无数据",

		"tui.reposelect.noconfig": "无仓库配置",
		"tui.reposelect.nodata":   "无数据",

		"tui.sync.title":     "正在同步数据...\n\n",
		"tui.sync.fail":      "失败: %s",
		"tui.sync.done":      "完成 (%d 次提交)",
		"tui.sync.fetch":     "拉取提交历史...",
		"tui.sync.listing":   "正在提取提交…",
		"tui.sync.logHeader": "执行日志：",
		"tui.sync.listingN":  "提取到 %d 个提交",
	},
}

// GranularityLabel returns the label for TUI index 0..3 (week..year).
func GranularityLabel(i int) string {
	keys := []string{"granularity.week", "granularity.month", "granularity.quarter", "granularity.year"}
	if i < 0 || i >= len(keys) {
		return T("granularity.week")
	}
	return T(keys[i])
}

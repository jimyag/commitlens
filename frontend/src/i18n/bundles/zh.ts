import type { en } from './en'

export const zh: Record<keyof typeof en, string> = {
  'lang.en': 'English',
  'lang.zh': '中文',

  'app.language': '语言',
  'app.allRepos': '全部仓库',
  'app.granularity.week': '按周',
  'app.granularity.month': '按月',
  'app.granularity.quarter': '按季度',
  'app.granularity.year': '按年',
  'app.syncing': '同步中…',
  'app.refresh': '刷新',
  'app.lastSync': '最后同步：',
  'app.section.trendTitle': '代码提交趋势',
  'app.section.trendDesc':
    '上方：各周期全仓库提交总数（柱顶数字）。下方：各贡献者明细。悬停可查看详情；底部滑块可缩放时间范围。',
  'app.section.rankTitle': '贡献者排行榜',

  'table.contributor': '贡献者',
  'table.prs': 'PR 数',
  'table.commits': '提交数',
  'table.added': '新增行',
  'table.deleted': '删除行',
  'table.empty': '暂无数据',

  'trend.emptyWeekly': '暂无统计数据，请先从 TUI/CLI 执行同步或点击刷新。',
  'trend.truncated': '为保持图表清晰，仅展示提交数前 {n} 名的贡献者（共 {total} 人）；完整排名见下方表格。',

  'chart.repoWide': '全仓库',
  'chart.totalSeries': '全站总计',
  'chart.sub': '粒度：{g} · 每一行时间轴对齐；悬停查看数值；底部滑块可左右滚动。',
  'chart.prOnTop': '柱顶显示提交数',
  'chart.tooltipLine': '{who}: <span style="font-weight:600">{value}</span> 次提交',

  'nav.dashboard': '仪表盘',
  'nav.prs': '提交历史',

  'filter.allUsers': '全部贡献者',

  'prPage.back': '← 返回',
  'prPage.allPRs': '代码提交记录',
  'prPage.title': '提交记录 · {period}',
  'prPage.filterAll': '全部贡献者',
  'prPage.loading': '加载中…',
  'prPage.empty': '该时间段内暂无提交记录。',
  'prPage.colRepo': '仓库',
  'prPage.colPR': '标识',
  'prPage.colAuthors': '作者',
  'prPage.colDate': '日期',
  'prPage.colLines': '增删行',
  'prPage.total': '共 {n} 次提交',
  'prPage.allPeriods': '全部时间',
  'prPage.pagePrev': '← 上一页',
  'prPage.pageNext': '下一页 →',
  'prPage.pageOf': '第 {page} / {total} 页',
}

export const en = {
  'lang.en': 'English',
  'lang.zh': '中文',

  'app.language': 'Language',
  'app.allRepos': 'All repositories',
  'app.granularity.week': 'Weekly',
  'app.granularity.month': 'Monthly',
  'app.granularity.quarter': 'Quarterly',
  'app.granularity.year': 'Yearly',
  'app.syncing': 'Syncing…',
  'app.refresh': 'Refresh',
  'app.lastSync': 'Last updated:',
  'app.section.trendTitle': 'Merged PR trend',
  'app.section.trendDesc':
    'Top: repo-wide PR counts per period (labels on bars). Below: one row per contributor. Hover a bar for that period; bottom slider pans/zooms the time range.',
  'app.section.rankTitle': 'Contributors',

  'table.contributor': 'Contributor',
  'table.prs': 'PRs',
  'table.commits': 'Commits',
  'table.added': 'Lines +',
  'table.deleted': 'Lines −',
  'table.empty': 'No data',

  'trend.emptyWeekly': 'No weekly data yet; run sync from the TUI/CLI or click Refresh.',
  'trend.truncated': 'For readability, the chart shows the top {n} contributors by PR count ({total} people); see the ranking table below for everyone.',

  'chart.repoWide': 'Repo-wide',
  'chart.totalSeries': 'Repo total',
  'chart.sub':
    'Granularity: {g} · same periods in each row; hover a bar; bottom slider scrolls long ranges.',
  'chart.prOnTop': 'PR count on top',
  'chart.tooltipLine': '{who}: <span style="font-weight:600">{value}</span> PRs',
} as const

export type MessageKey = keyof typeof en

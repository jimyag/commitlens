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
  'app.section.trendTitle': 'Code submission trend',
  'app.section.trendDesc':
    'Top: repo-wide commit counts per period (labels on bars). Below: one row per contributor. Hover a bar for that period; bottom slider pans/zooms the time range.',
  'app.section.rankTitle': 'Contributors',

  'table.contributor': 'Contributor',
  'table.prs': 'PRs', // We might still use this in some contexts, but let's hide it in ranking
  'table.commits': 'Commits',
  'table.added': 'Lines +',
  'table.deleted': 'Lines −',
  'table.empty': 'No data',

  'trend.emptyWeekly': 'No weekly data yet; run sync from the TUI/CLI or click Refresh.',
  'trend.truncated': 'For readability, the chart shows the top {n} contributors by commit count ({total} people); see the ranking table below for everyone.',

  'chart.repoWide': 'Repo-wide',
  'chart.totalSeries': 'Repo total',
  'chart.sub':
    'Granularity: {g} · same periods in each row; hover a bar; bottom slider scrolls long ranges.',
  'chart.prOnTop': 'Commit count on top',
  'chart.tooltipLine': '{who}: <span style="font-weight:600">{value}</span> commits',

  'nav.dashboard': 'Dashboard',
  'nav.prs': 'Commits',

  'filter.allUsers': 'All contributors',

  'prPage.back': '← Back',
  'prPage.allPRs': 'Commit History',
  'prPage.title': 'Commits · {period}',
  'prPage.filterAll': 'All contributors',
  'prPage.loading': 'Loading…',
  'prPage.empty': 'No commits found for this period.',
  'prPage.colRepo': 'Repo',
  'prPage.colPR': 'SHA',
  'prPage.colAuthors': 'Author(s)',
  'prPage.colDate': 'Date',
  'prPage.colLines': 'Lines',
  'prPage.total': '{n} commits',
  'prPage.allPeriods': 'All periods',
  'prPage.pagePrev': '← Prev',
  'prPage.pageNext': 'Next →',
  'prPage.pageOf': 'Page {page} / {total}',
} as const

export type MessageKey = keyof typeof en

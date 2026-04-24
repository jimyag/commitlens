import type { en } from './en'

type Keys = { [K in keyof typeof en]: string }

export const zh: Keys = {
  'lang.en': 'English',
  'lang.zh': '中文',

  'app.language': '语言',
  'app.allRepos': '全部仓库',
  'app.granularity.week': '按周',
  'app.granularity.month': '按月',
  'app.granularity.quarter': '按季度',
  'app.granularity.year': '按年',
  'app.syncing': '同步中...',
  'app.refresh': '刷新数据',
  'app.lastSync': '上次更新:',
  'app.section.trendTitle': '合并 PR 趋势',
  'app.section.trendDesc':
    '上方为全仓库各周期 PR 数（柱顶为数量），其下按贡献者各一行。悬停单柱可查看该周期与 PR 数；底部可横向缩放平移多周期数据。',
  'app.section.rankTitle': '贡献者排行',

  'table.contributor': '贡献者',
  'table.prs': 'PR 数',
  'table.commits': 'Commit 数',
  'table.added': '新增行',
  'table.deleted': '删除行',
  'table.empty': '暂无数据',

  'trend.emptyWeekly': '暂无周度数据，请先在 TUI/CLI 同步或点击刷新。',
  'trend.truncated':
    '为便于浏览，图内仅显示 PR 数前 {n} 位（共 {total} 人），其余见下方「贡献者排行」。',

  'chart.repoWide': '全仓库',
  'chart.totalSeries': '全仓库',
  'chart.sub': '粒度: {g} · 与下方各贡献者同周期；悬停柱条查看数值；底栏滑块可横移长周期。',
  'chart.prOnTop': '柱顶 PR 数',
  'chart.tooltipLine': '{who}：<span style="font-weight:600">{value}</span> 个 PR',
}

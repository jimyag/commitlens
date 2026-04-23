import ReactECharts from 'echarts-for-react'
import type { WeeklyEntry } from '../api'

export type Granularity = 'week' | 'month' | 'quarter' | 'year'

interface Props {
  weekly: Record<string, WeeklyEntry>
  granularity: Granularity
  selectedLogin?: string
}

function toPeriodKey(weekKey: string, gran: Granularity): string {
  const m = weekKey.match(/^(\d{4})-W(\d{2})$/)
  if (!m) return weekKey
  const year = parseInt(m[1])
  const week = parseInt(m[2])
  if (gran === 'week') return weekKey
  if (gran === 'year') return `${year}`
  // ISO week to approximate month: Jan 4 is always week 1
  const jan4 = new Date(year, 0, 4)
  const weekday = jan4.getDay() || 7
  const monday = new Date(jan4)
  monday.setDate(jan4.getDate() - weekday + 1)
  const approx = new Date(monday)
  approx.setDate(monday.getDate() + (week - 1) * 7)
  const mo = approx.getMonth() + 1
  if (gran === 'month') return `${year}-${String(mo).padStart(2, '0')}`
  const quarter = Math.ceil(mo / 3)
  return `${year}-Q${quarter}`
}

export function TrendChart({ weekly, granularity, selectedLogin }: Props) {
  const periodMap: Record<string, { total: number; byLogin: Record<string, number> }> = {}

  for (const [weekKey, entry] of Object.entries(weekly)) {
    const period = toPeriodKey(weekKey, granularity)
    if (!periodMap[period]) periodMap[period] = { total: 0, byLogin: {} }
    periodMap[period].total += entry.total_prs
    for (const [login, count] of Object.entries(entry.contributors)) {
      periodMap[period].byLogin[login] = (periodMap[period].byLogin[login] ?? 0) + count
    }
  }

  const periods = Object.keys(periodMap).sort()
  const totalData = periods.map(p => periodMap[p].total)
  const personData = selectedLogin
    ? periods.map(p => periodMap[p].byLogin[selectedLogin] ?? 0)
    : []

  const series: object[] = [
    {
      name: '全仓库',
      type: 'line',
      data: totalData,
      smooth: true,
      symbol: 'circle',
      symbolSize: 5,
      areaStyle: { opacity: 0.15 },
      lineStyle: { width: 2.5 },
      itemStyle: { color: '#6366f1' },
    },
  ]

  if (selectedLogin) {
    series.push({
      name: selectedLogin,
      type: 'line',
      data: personData,
      smooth: true,
      symbol: 'circle',
      symbolSize: 5,
      areaStyle: { opacity: 0.1 },
      lineStyle: { width: 2, type: 'dashed' },
      itemStyle: { color: '#f59e0b' },
    })
  }

  const option = {
    backgroundColor: '#fff',
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross', label: { backgroundColor: '#6a7985' } },
    },
    legend: {
      data: selectedLogin ? ['全仓库', selectedLogin] : ['全仓库'],
      top: 8,
    },
    grid: { left: 48, right: 24, bottom: 56, top: 48 },
    xAxis: {
      type: 'category',
      data: periods,
      axisLabel: { rotate: periods.length > 12 ? 30 : 0, fontSize: 12 },
      boundaryGap: false,
    },
    yAxis: {
      type: 'value',
      name: 'PR 数',
      nameTextStyle: { fontSize: 12 },
      minInterval: 1,
    },
    series,
  }

  return <ReactECharts option={option} style={{ height: 340 }} notMerge />
}

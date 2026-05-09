import { useMemo, useCallback } from 'react'
import type { EChartsOption } from 'echarts'
import type { BarSeriesOption, GridComponentOption, XAXisComponentOption, YAXisComponentOption } from 'echarts'
import ReactECharts from 'echarts-for-react'
import type { MessageKey } from '../i18n/bundles/en'
import { useI18n } from '../i18n/I18nContext'
import type { ContributorStats, WeeklyEntry } from '../api'
import { toPeriodKey, type Granularity } from '../utils/periodUtils'

export type { Granularity }

export type Metric = 'commits' | 'lines'

/** 与单页性能平衡；超出可在下方表格中查看 */
const MAX_CONTRIBUTOR_CHARTS = 100

interface Props {
  weekly: Record<string, WeeklyEntry>
  granularity: Granularity
  metric?: Metric
  /** 用于趋势图左侧头像、贡献者表格等 */
  contributors: Record<string, ContributorStats>
  /** 当前选中的用户列表，用于过滤趋势图行 */
  selectedLogins?: string[]
  /** 点击柱子时的回调；login 为 undefined 表示总量柱 */
  onBarClick?: (period: string, login: string | undefined) => void
}
const BAR_COLOR_TOTAL = '#6366f1'
const BAR_COLOR_PERSON = '#16a34a'
const BAR_COLOR_ADD = '#10b981'
const BAR_COLOR_DEL = '#ef4444'

const AXIS = '#6b7280'
const AXISLINE = '#e5e7eb'

/** 柱顶显示数值；为 0 的柱不标字，避免过密 */
function barValueLabel(opts: { tiny?: boolean; color?: string }) {
  return {
    show: true,
    position: 'top' as const,
    color: opts.color ?? '#374151',
    fontSize: opts.tiny ? 8 : 9,
    fontWeight: 500 as const,
    formatter: (p: { value: unknown }) => {
      const n = Number(p.value)
      if (Number.isNaN(n) || n === 0) return ''
      return String(Math.abs(n))
    },
  }
}
const L_LEFT = 12
const L_NAME = 148
const R_PAD = 16
/** 全仓库图与下方各人图之间的空隙 (留给 DataZoom) */
const GAP_AFTER_TOTAL = 60

export function TrendChart({ weekly, granularity, metric = 'commits', contributors, selectedLogins = [], onBarClick }: Props) {
  const { t } = useI18n()
  const { option, heightPx, truncated, totalLoginCount, personRowLabels } = useMemo(
    () => buildTrendOption(weekly, granularity, metric, contributors, selectedLogins, { t }),
    [weekly, granularity, metric, contributors, selectedLogins, t],
  )

  const totalSeriesName = t('chart.totalSeries')
  const handleBarClick = useCallback(
    (params: { name?: string; seriesName?: string }) => {
      const period = params.name
      if (!period || !onBarClick) return
      // Match series names for HandleBarClick
      const login = (params.seriesName === totalSeriesName || params.seriesName?.includes('Total')) ? undefined : params.seriesName
      onBarClick(period, login)
    },
    [onBarClick, totalSeriesName],
  )

  if (!option) {
    return <div style={{ color: '#6b7280', padding: 24 }}>{t('trend.emptyWeekly')}</div>
  }

  return (
    <div>
      {truncated > 0 && (
        <p style={{ fontSize: 12, color: '#9ca3af', margin: '0 0 8px' }}>
          {t('trend.truncated').replace('{n}', String(MAX_CONTRIBUTOR_CHARTS)).replace('{total}', String(totalLoginCount))}
        </p>
      )}
      <div style={{ 
        position: 'relative', 
        width: '100%', 
        maxHeight: '700px', 
        overflowY: 'auto',
        border: '1px solid #f3f4f6',
        borderRadius: 8,
      }}>
        {personRowLabels.map(row => (
          <div
            key={row.login}
            style={{
              position: 'absolute',
              left: L_LEFT,
              top: row.topPx,
              transform: 'translateY(-50%)',
              zIndex: 2,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'flex-start',
              gap: 8,
              width: L_NAME,
              maxWidth: L_NAME,
              pointerEvents: 'none',
            }}
            aria-hidden
          >
            {row.avatarUrl ? (
              <img
                src={row.avatarUrl}
                alt=""
                width={28}
                height={28}
                style={{ borderRadius: '50%', border: '1px solid #e5e7eb', flexShrink: 0 }}
              />
            ) : (
              <span
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: '50%',
                  background: '#e5e7eb',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  color: '#6b7280',
                  flexShrink: 0,
                }}
              >
                {row.login.slice(0, 1).toUpperCase()}
              </span>
            )}
            <span
              title={row.login}
              style={{
                fontSize: 12,
                color: '#374151',
                fontWeight: 500,
                lineHeight: 1.2,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                minWidth: 0,
              }}
            >
              {row.login}
            </span>
          </div>
        ))}
        <ReactECharts
          option={option as EChartsOption}
          style={{ width: '100%', height: heightPx, minWidth: 400 }}
          notMerge
          onEvents={{ click: handleBarClick }}
        />
      </div>
    </div>
  )
}

type L10n = {
  t: (k: MessageKey) => string
}

function buildTrendOption(
  weekly: Record<string, WeeklyEntry>,
  granularity: Granularity,
  metric: Metric,
  contributors: Record<string, ContributorStats>,
  selectedLogins: string[],
  l10n: L10n,
) {
  const { t } = l10n
  const periodMap: Record<string, { 
    total_c: number; 
    total_a: number; 
    total_d: number; 
    byLogin: Record<string, {c:number, a:number, d:number}> 
  }> = {}

  for (const [weekKey, entry] of Object.entries(weekly)) {
    const period = toPeriodKey(weekKey, granularity)
    if (!periodMap[period]) periodMap[period] = { total_c: 0, total_a: 0, total_d: 0, byLogin: {} }
    periodMap[period].total_c += entry.total_commits
    periodMap[period].total_a += entry.total_additions
    periodMap[period].total_d += entry.total_deletions
    
    for (const [login, stats] of Object.entries(entry.contributors)) {
      if (!periodMap[period].byLogin[login]) periodMap[period].byLogin[login] = {c:0, a:0, d:0}
      periodMap[period].byLogin[login].c += stats.commits
      periodMap[period].byLogin[login].a += stats.additions
      periodMap[period].byLogin[login].d += stats.deletions
    }
  }

  const periods = Object.keys(periodMap).sort()
  if (periods.length === 0) {
    return {
      option: null as EChartsOption | null,
      heightPx: 0,
      truncated: 0,
      totalLoginCount: 0,
      personRowLabels: [] as { login: string; topPx: number; avatarUrl: string }[],
    }
  }

  const loginTotals: Record<string, number> = {}
  for (const p of periods) {
    for (const [l, stats] of Object.entries(periodMap[p].byLogin)) {
      loginTotals[l] = (loginTotals[l] ?? 0) + (metric === 'commits' ? stats.c : (stats.a + stats.d))
    }
  }
  const allSorted = Object.keys(loginTotals).sort(
    (a, b) => (loginTotals[b] ?? 0) - (loginTotals[a] ?? 0) || a.localeCompare(b),
  )

  let logins: string[]
  let truncated = 0
  if (selectedLogins.length > 0) {
    logins = allSorted.filter(l => selectedLogins.includes(l))
  } else {
    logins = allSorted.slice(0, MAX_CONTRIBUTOR_CHARTS)
    truncated = allSorted.length - logins.length
  }
  
  const totalLoginCount = allSorted.length

  const nRows = 1 + logins.length
  const totalH = 260
  const personH = 60
  const gapY = 16
  const topStart = 10
  const xLabelExtra = 22
  const sliderH = 30
  const footPad = 10

  const leftInner = L_LEFT + L_NAME
  const grids: GridComponentOption[] = []
  const xAxes: XAXisComponentOption[] = []
  const yAxes: YAXisComponentOption[] = []
  const series: BarSeriesOption[] = []
  const xIdx = Array.from({ length: nRows }, (_, j) => j)
  const rotateX = periods.length > 20 ? 45 : 0
  const personRowLabels: { login: string; topPx: number; avatarUrl: string }[] = []

  // 全仓
  let y = topStart
  grids.push({ left: leftInner, right: R_PAD, top: y, width: 'auto', height: totalH, containLabel: false })
  xAxes.push({
    type: 'category',
    data: periods,
    gridIndex: 0,
    axisLabel: { show: true, color: AXIS, fontSize: 10, interval: 0, rotate: rotateX },
    axisLine: { lineStyle: { color: AXISLINE } },
  })
  yAxes.push({
    type: 'value',
    gridIndex: 0,
    name: metric === 'commits' ? t('chart.repoWide') : 'Lines (+/-)',
    nameTextStyle: { color: AXIS, fontSize: 11, fontWeight: 600 },
    min: 0,
    splitLine: { lineStyle: { color: '#f3f4f6' } },
  })

  if (metric === 'commits') {
    series.push({
      name: 'Total Commits',
      type: 'bar',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: periods.map(p => periodMap[p].total_c),
      itemStyle: { color: BAR_COLOR_TOTAL },
      barMaxWidth: 20,
      label: barValueLabel({ color: '#4f46e5' }),
    })
  } else {
    series.push({
      name: 'Total Additions',
      type: 'bar',
      stack: 'total',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: periods.map(p => periodMap[p].total_a),
      itemStyle: { color: BAR_COLOR_ADD },
      barMaxWidth: 20,
    })
    series.push({
      name: 'Total Deletions',
      type: 'bar',
      stack: 'total',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: periods.map(p => -(periodMap[p].total_d ?? 0)),
      itemStyle: { color: BAR_COLOR_DEL },
      barMaxWidth: 20,
    })
  }

  y += totalH + gapY + GAP_AFTER_TOTAL

  for (let i = 0; i < logins.length; i++) {
    const gi = 1 + i
    const login = logins[i]
    
    const showXLabel = i === logins.length - 1
    const h = showXLabel ? personH + xLabelExtra : personH
    grids.push({ left: leftInner, right: R_PAD, top: y, width: 'auto', height: h, containLabel: false })
    xAxes.push({
      type: 'category',
      data: periods,
      gridIndex: gi,
      axisLabel: { show: showXLabel, color: AXIS, fontSize: 9, interval: 0, rotate: rotateX },
      axisLine: { show: true, lineStyle: { color: AXISLINE } },
    })
    yAxes.push({
      type: 'value',
      gridIndex: gi,
      min: 0,
      splitLine: { lineStyle: { color: '#f3f4f6' } },
    })
    const avatarUrl = contributors[login]?.avatar_url?.trim() ?? ''
    personRowLabels.push({ login, topPx: y + h / 2, avatarUrl })
    
    if (metric === 'commits') {
      series.push({
        name: login,
        type: 'bar',
        xAxisIndex: gi,
        yAxisIndex: gi,
        data: periods.map(p => periodMap[p].byLogin[login]?.c ?? 0),
        itemStyle: { color: BAR_COLOR_PERSON, opacity: 0.9 },
        barMaxWidth: 16,
        label: barValueLabel({ tiny: true }),
      })
    } else {
      series.push({
        name: login + ' (+)',
        type: 'bar',
        stack: login,
        xAxisIndex: gi,
        yAxisIndex: gi,
        data: periods.map(p => periodMap[p].byLogin[login]?.a ?? 0),
        itemStyle: { color: BAR_COLOR_ADD, opacity: 0.8 },
        barMaxWidth: 16,
      })
      series.push({
        name: login + ' (-)',
        type: 'bar',
        stack: login,
        xAxisIndex: gi,
        yAxisIndex: gi,
        data: periods.map(p => -(periodMap[p].byLogin[login]?.d ?? 0)),
        itemStyle: { color: BAR_COLOR_DEL, opacity: 0.8 },
        barMaxWidth: 16,
      })
    }
    y += h + gapY
  }

  const heightPx = Math.ceil(y + footPad)
  
  // Default Zoom: show last 30 periods
  const zoomStart = periods.length > 30 ? ((periods.length - 30) / periods.length) * 100 : 0

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#374151' },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      confine: true,
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#f9fafb', fontSize: 12 },
      formatter(params: any) {
        if (!params || params.length === 0) return ''
        const period = params[0].name
        let res = `<div style="font-weight:600;margin-bottom:6px">${period}</div>`
        params.forEach((p: any) => {
          if (p.value === 0) return
          const val = Math.abs(p.value)
          res += `<div>${p.seriesName}: <span style="font-weight:600">${val}</span></div>`
        })
        return res
      }
    },
    dataZoom: [
      {
        type: 'slider',
        xAxisIndex: xIdx,
        left: leftInner,
        right: R_PAD,
        top: topStart + totalH + 10, // Sticky below total chart
        height: sliderH,
        start: zoomStart,
        end: 100,
        filterMode: 'none',
        borderColor: '#d1d5db',
        handleStyle: { color: BAR_COLOR_TOTAL },
        dataBackground: { lineStyle: { color: AXISLINE }, areaStyle: { color: 'rgba(99, 102, 241, 0.08)' } },
        textStyle: { color: AXIS, fontSize: 10 },
      },
      { type: 'inside', xAxisIndex: xIdx },
    ],
    grid: grids,
    xAxis: xAxes,
    yAxis: yAxes,
    series: series,
  }

  return { option, heightPx, truncated, totalLoginCount, personRowLabels }
}

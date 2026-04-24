import { useMemo } from 'react'
import type { EChartsOption } from 'echarts'
import type { BarSeriesOption, GridComponentOption, XAXisComponentOption, YAXisComponentOption } from 'echarts'
import ReactECharts from 'echarts-for-react'
import type { MessageKey } from '../i18n/bundles/en'
import { useI18n } from '../i18n/I18nContext'
import type { ContributorStats, WeeklyEntry } from '../api'

export type Granularity = 'week' | 'month' | 'quarter' | 'year'

/** 与单页性能平衡；超出可在下方表格中查看 */
const MAX_CONTRIBUTOR_CHARTS = 48

interface Props {
  weekly: Record<string, WeeklyEntry>
  granularity: Granularity
  /** 用于趋势图左侧头像、贡献者表格等 */
  contributors: Record<string, ContributorStats>
}

function toPeriodKey(weekKey: string, gran: Granularity): string {
  const m = weekKey.match(/^(\d{4})-W(\d{2})$/)
  if (!m) return weekKey
  const year = parseInt(m[1])
  const week = parseInt(m[2])
  if (gran === 'week') return weekKey
  if (gran === 'year') return `${year}`
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

const BAR_COLOR_TOTAL = '#6366f1'
const BAR_COLOR_PERSON = '#16a34a'
const AXIS = '#6b7280'
const AXISLINE = '#e5e7eb'
/** 柱顶显示 PR 数；为 0 的柱不标字，避免过密一墙 0 */
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
      return String(n)
    },
  }
}
const L_LEFT = 12
const L_NAME = 148
const R_PAD = 16
/** 全仓库图与下方各人图之间的空隙 */
const GAP_AFTER_TOTAL = 44

export function TrendChart({ weekly, granularity, contributors }: Props) {
  const { t, tf } = useI18n()
  const { option, heightPx, truncated, totalLoginCount, personRowLabels } = useMemo(
    () => buildTrendOption(weekly, granularity, contributors, { t, tf }),
    [weekly, granularity, contributors, t, tf],
  )

  if (!option) {
    return <div style={{ color: '#6b7280', padding: 24 }}>{t('trend.emptyWeekly')}</div>
  }

  return (
    <div>
      {truncated > 0 && (
        <p style={{ fontSize: 12, color: '#9ca3af', margin: '0 0 8px' }}>
          {tf('trend.truncated', { n: MAX_CONTRIBUTOR_CHARTS, total: totalLoginCount })}
        </p>
      )}
      <div style={{ position: 'relative', width: '100%' }}>
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
        />
      </div>
    </div>
  )
}

type L10n = {
  t: (k: MessageKey) => string
  tf: (k: MessageKey, vars: Record<string, string | number>) => string
}

function buildTrendOption(
  weekly: Record<string, WeeklyEntry>,
  granularity: Granularity,
  contributors: Record<string, ContributorStats>,
  l10n: L10n,
) {
  const { t, tf } = l10n
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
  if (periods.length === 0) {
    return {
      option: null as EChartsOption | null,
      heightPx: 0,
      truncated: 0,
      totalLoginCount: 0,
      personRowLabels: [] as { login: string; topPx: number; avatarUrl: string }[],
    }
  }

  const totalData = periods.map(p => periodMap[p].total)
  const loginTotals: Record<string, number> = {}
  for (const p of periods) {
    for (const [l, c] of Object.entries(periodMap[p].byLogin)) {
      loginTotals[l] = (loginTotals[l] ?? 0) + c
    }
  }
  const allSorted = Object.keys(loginTotals).sort(
    (a, b) => (loginTotals[b] ?? 0) - (loginTotals[a] ?? 0) || a.localeCompare(b),
  )
  const totalLoginCount = allSorted.length
  const logins = allSorted.slice(0, MAX_CONTRIBUTOR_CHARTS)
  const truncated = totalLoginCount - logins.length

  const nRows = 1 + logins.length
  const totalH = 300
  const personH = 58
  const gapY = 14
  const topStart = 6
  const xLabelExtra = 22
  const sliderH = 24
  const footPad = 14

  const leftInner = L_LEFT + L_NAME
  const grids: GridComponentOption[] = []
  const xAxes: XAXisComponentOption[] = []
  const yAxes: YAXisComponentOption[] = []
  const series: BarSeriesOption[] = []
  const xIdx = Array.from({ length: nRows }, (_, j) => j)
  const rotateX = periods.length > 20 ? 50 : periods.length > 12 ? 32 : 0
  const personRowLabels: { login: string; topPx: number; avatarUrl: string }[] = []

  // 全仓（柱区加高，等效「更宽」的可视与点击区域）
  let y = topStart
  let contentBottom = topStart + totalH
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
    name: t('chart.repoWide'),
    nameTextStyle: { color: AXIS, fontSize: 11, fontWeight: 600 },
    nameLocation: 'end',
    min: 0,
    minInterval: 1,
    splitLine: { lineStyle: { color: '#f3f4f6' } },
  })
  series.push({
    name: t('chart.totalSeries'),
    type: 'bar',
    xAxisIndex: 0,
    yAxisIndex: 0,
    data: totalData,
    itemStyle: { color: BAR_COLOR_TOTAL },
    barMaxWidth: 16,
    label: barValueLabel({ color: '#4f46e5' }),
  })
  y += totalH + gapY + GAP_AFTER_TOTAL

  for (let i = 0; i < logins.length; i++) {
    const gi = 1 + i
    const login = logins[i]
    const rowData = periods.map(p => periodMap[p].byLogin[login] ?? 0)
    const showXLabel = i === logins.length - 1
    const h = showXLabel ? personH + xLabelExtra : personH
    grids.push({ left: leftInner, right: R_PAD, top: y, width: 'auto', height: h, containLabel: false })
    xAxes.push({
      type: 'category',
      data: periods,
      gridIndex: gi,
      axisLabel: {
        show: showXLabel,
        color: AXIS,
        fontSize: 9,
        interval: 0,
        rotate: rotateX,
      },
      axisLine: { show: true, lineStyle: { color: AXISLINE } },
    })
    yAxes.push({
      type: 'value',
      gridIndex: gi,
      min: 0,
      minInterval: 1,
      splitLine: { lineStyle: { color: '#f3f4f6' } },
    })
    const avatarUrl = contributors[login]?.avatar_url?.trim() ?? ''
    personRowLabels.push({
      login,
      topPx: y + h / 2,
      avatarUrl,
    })
    series.push({
      name: login,
      type: 'bar',
      xAxisIndex: gi,
      yAxisIndex: gi,
      data: rowData,
      itemStyle: { color: BAR_COLOR_PERSON, opacity: 0.92 },
      barMaxWidth: 12,
      label: barValueLabel({ tiny: periods.length > 30 }),
    })
    contentBottom = y + h
    y += h + gapY
  }

  const heightPx = Math.ceil(contentBottom + footPad + sliderH + 4)

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#374151' },
    tooltip: {
      trigger: 'item',
      confine: true,
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#f9fafb', fontSize: 12 },
      formatter(params) {
        const p = params as { name?: string; value?: number; seriesName?: string; dataIndex?: number }
        const period = (p.name as string) || (p.dataIndex != null ? (periods[p.dataIndex] ?? '') : '')
        const v = p.value
        const who = p.seriesName ?? ''
        if (v == null || !who) return ''
        const line = tf('chart.tooltipLine', { who: escapeHtml(who), value: v })
        return [
          `<div style="font-weight:600;margin-bottom:6px">${escapeHtml(period)}</div>`,
          line,
        ].join('<br/>')
      },
    },
    dataZoom: [
      {
        type: 'slider',
        xAxisIndex: xIdx,
        left: leftInner,
        right: R_PAD,
        bottom: 4,
        height: sliderH,
        filterMode: 'none',
        borderColor: '#d1d5db',
        handleStyle: { color: BAR_COLOR_TOTAL },
        dataBackground: { lineStyle: { color: AXISLINE }, areaStyle: { color: 'rgba(99, 102, 241, 0.08)' } },
        textStyle: { color: AXIS, fontSize: 10 },
        showDetail: true,
        brushSelect: true,
        labelFormatter: (v: string | number) => (String(v).length > 8 ? '…' : String(v)),
      },
    ],
    grid: grids,
    xAxis: xAxes,
    yAxis: yAxes,
    series: series,
  }

  return { option, heightPx, truncated, totalLoginCount, personRowLabels }
}

function escapeHtml(s: string) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

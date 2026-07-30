import { useMemo, useCallback, useEffect, useRef } from 'react'
import type { EChartsOption } from 'echarts'
import type { BarSeriesOption, GridComponentOption, XAXisComponentOption, YAXisComponentOption } from 'echarts'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { DataZoomComponent, GridComponent, TooltipComponent } from 'echarts/components'
import { LabelLayout } from 'echarts/features'
import { CanvasRenderer } from 'echarts/renderers'
import ReactEChartsCore from 'echarts-for-react/esm/core'
import type { MessageKey } from '../i18n/bundles/en'
import { useI18n } from '../i18n/I18nContext'
import type { ContributorStats, PeriodComparison, WeeklyEntry } from '../api'
import { toPeriodKey, type Granularity } from '../utils/periodUtils'

echarts.use([
  BarChart,
  DataZoomComponent,
  GridComponent,
  TooltipComponent,
  LabelLayout,
  CanvasRenderer,
])

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
  /** 各周期的环比与历史同期结果，用于 Tooltip */
  comparisons?: Record<string, PeriodComparison>
  /** 点击柱子时的回调；login 为 undefined 表示总量柱 */
  onBarClick?: (period: string, login: string | undefined) => void
}
const BAR_COLOR_TOTAL = '#6366f1'
const BAR_COLOR_PERSON = '#16a34a'
const BAR_COLOR_ADD = '#10b981'
const BAR_COLOR_DEL = '#ef4444'

const AXIS = '#6b7280'
const AXISLINE = '#e5e7eb'
const COMPRESSED_SCALE_RATIO = 100

type TrendBarData = {
  value: number
  rawValue: number
}

function shouldCompressScale(values: number[]) {
  const nonZero = values.map(Math.abs).filter(value => value > 0)
  if (nonZero.length < 2) return false
  return Math.max(...nonZero) / Math.min(...nonZero) >= COMPRESSED_SCALE_RATIO
}

function signedLog1p(value: number) {
  return Math.sign(value) * Math.log10(1 + Math.abs(value))
}

function lineBarData(value: number, compressed: boolean): number | TrendBarData {
  if (!compressed) return value
  return { value: signedLog1p(value), rawValue: value }
}

function formatAxisLines(value: number) {
  if (value === 0) return '0'
  const approximateValue = Math.sign(value) * 10 ** Math.abs(value)
  return new Intl.NumberFormat(undefined, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(approximateValue)
}

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
const MIN_ZOOM_RANGE = 5
const MIN_PERIOD_LABEL_WIDTH = 18

export function TrendChart({ weekly, granularity, metric = 'commits', contributors, selectedLogins = [], comparisons, onBarClick }: Props) {
  const { t } = useI18n()
  const chartRef = useRef<ReactEChartsCore>(null)
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const periodLabelsVisibleRef = useRef<boolean | null>(null)
  const { option, heightPx, truncated, totalLoginCount, personRowLabels } = useMemo(
    () => buildTrendOption(weekly, granularity, metric, contributors, selectedLogins, comparisons, { t }),
    [weekly, granularity, metric, contributors, selectedLogins, comparisons, t],
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

  const updatePeriodLabelVisibility = useCallback(() => {
    const instance = chartRef.current?.getEchartsInstance()
    const container = chartContainerRef.current
    if (!instance || !container) return

    const currentOption = instance.getOption()
    const zoom = (currentOption.dataZoom as { start?: number; end?: number }[] | undefined)?.[0]
    const xAxes = currentOption.xAxis as { data?: unknown[] }[] | undefined
    const totalPeriods = xAxes?.[0]?.data?.length ?? 0
    if (
      !xAxes
      || totalPeriods === 0
      || typeof zoom?.start !== 'number'
      || typeof zoom.end !== 'number'
    ) return

    const visiblePeriods = Math.max(1, Math.ceil(totalPeriods * (zoom.end - zoom.start) / 100))
    const plotWidth = Math.max(1, container.clientWidth - L_LEFT - L_NAME - R_PAD)
    const visible = plotWidth / visiblePeriods >= MIN_PERIOD_LABEL_WIDTH
    if (periodLabelsVisibleRef.current === visible) return
    periodLabelsVisibleRef.current = visible

    instance.setOption({
      xAxis: xAxes.map((_, index) => ({
        axisLabel: {
          show: visible && (index === 0 || index === xAxes.length - 1),
        },
      })),
    })
  }, [])

  useEffect(() => {
    const container = chartContainerRef.current
    if (!container) return
    periodLabelsVisibleRef.current = null

    const handleWheel = (event: WheelEvent) => {
      const isHorizontalPan = !event.shiftKey && Math.abs(event.deltaX) > Math.abs(event.deltaY)
      if (!event.shiftKey && !isHorizontalPan) return

      const instance = chartRef.current?.getEchartsInstance()
      const zoom = (instance?.getOption().dataZoom as { start?: number; end?: number }[] | undefined)?.[0]
      if (!instance || typeof zoom?.start !== 'number' || typeof zoom.end !== 'number') return

      event.preventDefault()

      const rawDelta = isHorizontalPan ? event.deltaX : event.deltaY || event.deltaX
      const deltaPixels = event.deltaMode === WheelEvent.DOM_DELTA_LINE
        ? rawDelta * 16
        : event.deltaMode === WheelEvent.DOM_DELTA_PAGE
          ? rawDelta * window.innerHeight
          : rawDelta
      const currentRange = zoom.end - zoom.start

      const rect = container.getBoundingClientRect()
      const plotLeft = rect.left + L_LEFT + L_NAME
      const plotWidth = Math.max(1, rect.width - L_LEFT - L_NAME - R_PAD)

      if (isHorizontalPan) {
        const nextStart = Math.max(
          0,
          Math.min(100 - currentRange, zoom.start + deltaPixels / plotWidth * currentRange),
        )
        instance.dispatchAction({
          type: 'dataZoom',
          dataZoomIndex: 0,
          start: nextStart,
          end: nextStart + currentRange,
        })
        return
      }

      const scale = Math.exp(Math.max(-100, Math.min(100, deltaPixels)) * 0.0025)
      const nextRange = Math.max(MIN_ZOOM_RANGE, Math.min(100, currentRange * scale))
      const pointerRatio = Math.max(0, Math.min(1, (event.clientX - plotLeft) / plotWidth))
      const anchor = zoom.start + currentRange * pointerRatio
      let start = anchor - nextRange * pointerRatio
      start = Math.max(0, Math.min(100 - nextRange, start))

      instance.dispatchAction({
        type: 'dataZoom',
        dataZoomIndex: 0,
        start,
        end: start + nextRange,
      })
    }

    const resizeObserver = new ResizeObserver(updatePeriodLabelVisibility)
    resizeObserver.observe(container)
    const frame = requestAnimationFrame(updatePeriodLabelVisibility)
    container.addEventListener('wheel', handleWheel, { passive: false })
    return () => {
      cancelAnimationFrame(frame)
      resizeObserver.disconnect()
      container.removeEventListener('wheel', handleWheel)
    }
  }, [option, updatePeriodLabelVisibility])

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
      <div ref={chartContainerRef} style={{
        position: 'relative',
        width: '100%',
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
        <ReactEChartsCore
          ref={chartRef}
          echarts={echarts}
          option={option as EChartsOption}
          style={{ width: '100%', height: heightPx, minWidth: 400 }}
          notMerge
          onEvents={{
            click: handleBarClick,
            datazoom: updatePeriodLabelVisibility,
          }}
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
  comparisons: Record<string, PeriodComparison> | undefined,
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
  const compressedLineScale = metric === 'lines' && shouldCompressScale(
    periods.flatMap(period => [
      periodMap[period].total_a,
      periodMap[period].total_d,
    ]),
  )

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
    name: metric === 'commits'
      ? t('chart.repoWide')
      : `${t('chart.lines')} ${compressedLineScale ? `· ${t('chart.compressedScale')}` : ''}`,
    nameTextStyle: { color: AXIS, fontSize: 11, fontWeight: 600 },
    min: metric === 'commits' ? 0 : undefined,
    axisLabel: compressedLineScale
      ? { formatter: formatAxisLines, color: AXIS, fontSize: 10 }
      : undefined,
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
      data: periods.map(p => lineBarData(periodMap[p].total_a, compressedLineScale)),
      itemStyle: { color: BAR_COLOR_ADD },
      barMaxWidth: 20,
    })
    series.push({
      name: 'Total Deletions',
      type: 'bar',
      stack: 'total',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: periods.map(p => lineBarData(-(periodMap[p].total_d ?? 0), compressedLineScale)),
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
      min: metric === 'commits' ? 0 : undefined,
      axisLabel: compressedLineScale
        ? { formatter: formatAxisLines, color: AXIS, fontSize: 9 }
        : undefined,
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
        data: periods.map(p => lineBarData(periodMap[p].byLogin[login]?.a ?? 0, compressedLineScale)),
        itemStyle: { color: BAR_COLOR_ADD, opacity: 0.8 },
        barMaxWidth: 16,
      })
      series.push({
        name: login + ' (-)',
        type: 'bar',
        stack: login,
        xAxisIndex: gi,
        yAxisIndex: gi,
        data: periods.map(p => lineBarData(-(periodMap[p].byLogin[login]?.d ?? 0), compressedLineScale)),
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
      formatter(params: unknown) {
        const pList = params as {
          name: string
          seriesName: string
          value: number
          data?: number | TrendBarData
        }[]
        if (!pList || pList.length === 0) return ''
        const period = pList[0].name
        let res = `<div style="font-weight:600;margin-bottom:6px">${period}</div>`
        pList.forEach((p) => {
          const rawValue = typeof p.data === 'object' && p.data !== null
            ? p.data.rawValue
            : p.value
          if (rawValue === 0) return
          res += `<div>${p.seriesName}: <span style="font-weight:600">${Math.abs(rawValue).toLocaleString()}</span></div>`
        })
        const comparison = comparisons?.[period]
        if (comparison?.previous.available) {
          const previous = comparison.previous.metrics
          const detail = metric === 'commits'
            ? `${previous.commits} ${t('table.commits')}`
            : `+${previous.additions} / -${previous.deletions}`
          res += `<div style="border-top:1px solid rgba(255,255,255,.18);margin-top:6px;padding-top:6px">${t('comparison.previous')} ${comparison.previous.label}: ${detail}</div>`
        }
        const samePeriod = comparison?.historical.find(point => point.available)
        if (samePeriod) {
          const detail = metric === 'commits'
            ? `${samePeriod.metrics.commits} ${t('table.commits')}`
            : `+${samePeriod.metrics.additions} / -${samePeriod.metrics.deletions}`
          res += `<div>${t('comparison.samePeriod')} ${samePeriod.label}: ${detail}</div>`
        }
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
    ],
    grid: grids,
    xAxis: xAxes,
    yAxis: yAxes,
    series: series,
  }

  return { option, heightPx, truncated, totalLoginCount, personRowLabels }
}

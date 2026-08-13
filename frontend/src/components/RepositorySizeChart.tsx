import { useMemo } from 'react'
import type { EChartsOption } from 'echarts'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { DataZoomComponent, GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import ReactEChartsCore from 'echarts-for-react/esm/core'
import type { StatsData } from '../api'
import { useI18n } from '../i18n/I18nContext'
import type { Granularity } from '../utils/periodUtils'

echarts.use([LineChart, DataZoomComponent, GridComponent, TooltipComponent, CanvasRenderer])

interface Props {
  stats: StatsData[]
  granularity: Granularity
}

export function RepositorySizeChart({ stats, granularity }: Props) {
  const { t } = useI18n()
  const option = useMemo(() => buildOption(stats, granularity, t('repositorySize.series')), [granularity, stats, t])

  if (!option) {
    return (
      <div style={{ color: '#6b7280', padding: 24 }}>
        {t('repositorySize.empty')}
      </div>
    )
  }

  return (
    <ReactEChartsCore
      echarts={echarts}
      option={option}
      style={{ width: '100%', height: 340, minWidth: 400 }}
      notMerge
    />
  )
}

function buildOption(stats: StatsData[], granularity: Granularity, seriesName: string): EChartsOption | null {
  const totals = new Map<string, number>()
  for (const repository of stats) {
    const periods = repository.repository_size?.[granularity]
    if (!periods) continue
    for (const [period, point] of Object.entries(periods)) {
      totals.set(period, (totals.get(period) ?? 0) + point.lines)
    }
  }

  const periods = [...totals.keys()].sort()
  if (periods.length === 0) return null
  const zoomStart = periods.length > 30 ? (periods.length - 30) / periods.length * 100 : 0

  return {
    animationDuration: 300,
    tooltip: {
      trigger: 'axis',
      confine: true,
      backgroundColor: 'rgba(17, 24, 39, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#f9fafb', fontSize: 12 },
      valueFormatter: value => Number(value).toLocaleString(),
    },
    grid: { left: 72, right: 24, top: 24, bottom: 72 },
    dataZoom: [
      {
        type: 'slider',
        start: zoomStart,
        end: 100,
        bottom: 14,
        height: 26,
        borderColor: '#d1d5db',
        handleStyle: { color: '#6366f1' },
      },
      { type: 'inside', start: zoomStart, end: 100 },
    ],
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: periods,
      axisLabel: { color: '#6b7280', hideOverlap: true },
      axisLine: { lineStyle: { color: '#e5e7eb' } },
    },
    yAxis: {
      type: 'value',
      min: value => Math.max(0, Math.floor(value.min * 0.95)),
      axisLabel: {
        color: '#6b7280',
        formatter: value => new Intl.NumberFormat(undefined, {
          notation: 'compact',
          maximumFractionDigits: 1,
        }).format(Number(value)),
      },
      splitLine: { lineStyle: { color: '#f3f4f6' } },
    },
    series: [{
      name: seriesName,
      type: 'line',
      data: periods.map(period => totals.get(period) ?? 0),
      showSymbol: periods.length <= 30,
      symbolSize: 6,
      smooth: false,
      lineStyle: { color: '#6366f1', width: 2 },
      itemStyle: { color: '#6366f1' },
      areaStyle: { color: 'rgba(99, 102, 241, 0.10)' },
    }],
  }
}

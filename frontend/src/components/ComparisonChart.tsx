import { useMemo } from 'react'
import type { EChartsOption } from 'echarts'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { LabelLayout } from 'echarts/features'
import { CanvasRenderer } from 'echarts/renderers'
import ReactEChartsCore from 'echarts-for-react/esm/core'
import type { ComparisonMetrics, ComparisonPoint, PeriodComparison } from '../api'
import { useI18n } from '../i18n/I18nContext'

echarts.use([
  BarChart,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  LabelLayout,
  CanvasRenderer,
])

type Metric = 'commits' | 'lines'

interface Props {
  comparison?: PeriodComparison
  metric: Metric
  filteredByContributors: boolean
}

function metricValue(metrics: ComparisonMetrics, metric: keyof ComparisonMetrics) {
  return metrics[metric]
}

function formatChange(current: number, baseline: number) {
  const delta = current - baseline
  const sign = delta > 0 ? '+' : ''
  if (baseline === 0) return `${sign}${delta.toLocaleString()}`
  return `${sign}${delta.toLocaleString()} (${sign}${(delta / baseline * 100).toFixed(1)}%)`
}

export function ComparisonChart({ comparison, metric, filteredByContributors }: Props) {
  const { t } = useI18n()
  const option = useMemo<EChartsOption | null>(() => {
    if (!comparison?.anchor.available) return null

    const points: ComparisonPoint[] = [
      comparison.anchor,
      comparison.previous,
      ...comparison.historical,
    ]
    const categories = points.map((point, index) => {
      if (index === 0) return `${t('comparison.current')}\n${point.label}`
      if (index === 1) return `${t('comparison.previous')}\n${point.label}`
      return `${t('comparison.samePeriod')}\n${point.label}`
    })
    const current = comparison.anchor.metrics
    const makeData = (key: keyof ComparisonMetrics) =>
      points.map(point => point.available ? metricValue(point.metrics, key) : null)

    const series = metric === 'commits'
      ? [{
          name: t('table.commits'),
          type: 'bar' as const,
          data: makeData('commits'),
          itemStyle: { color: '#6366f1' },
          barMaxWidth: 42,
          label: { show: true, position: 'top' as const, color: '#4f46e5', fontSize: 11 },
        }]
      : [
          {
            name: t('table.added'),
            type: 'bar' as const,
            data: makeData('additions'),
            itemStyle: { color: '#10b981' },
            barMaxWidth: 32,
            label: { show: true, position: 'top' as const, color: '#059669', fontSize: 10 },
          },
          {
            name: t('table.deleted'),
            type: 'bar' as const,
            data: makeData('deletions'),
            itemStyle: { color: '#ef4444' },
            barMaxWidth: 32,
            label: { show: true, position: 'top' as const, color: '#dc2626', fontSize: 10 },
          },
        ]

    return {
      animationDuration: 250,
      grid: { left: 56, right: 24, top: metric === 'lines' ? 42 : 24, bottom: 54 },
      legend: metric === 'lines' ? { top: 4 } : undefined,
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        confine: true,
        formatter(params: unknown) {
          const list = params as { dataIndex: number; seriesName: string; value: number | null }[]
          if (!list?.length) return ''
          const point = points[list[0].dataIndex]
          let output = `<div style="font-weight:600;margin-bottom:6px">${point.label}</div>`
          if (!point.available) {
            return output + `<div>${t('comparison.noData')}</div>`
          }
          for (const item of list) {
            if (item.value == null) continue
            const key = metric === 'commits'
              ? 'commits'
              : item.seriesName === t('table.added') ? 'additions' : 'deletions'
            output += `<div>${item.seriesName}: <b>${item.value.toLocaleString()}</b></div>`
            if (list[0].dataIndex > 0) {
              output += `<div style="color:#d1d5db">${t('comparison.currentVsPeriod')}: ${formatChange(metricValue(current, key), item.value)}</div>`
            }
          }
          return output
        },
      },
      xAxis: {
        type: 'category',
        data: categories,
        axisTick: { show: false },
        axisLabel: { color: '#6b7280', fontSize: 10, lineHeight: 15 },
        axisLine: { lineStyle: { color: '#e5e7eb' } },
      },
      yAxis: {
        type: 'value',
        min: 0,
        splitLine: { lineStyle: { color: '#f3f4f6' } },
        axisLabel: { color: '#9ca3af', fontSize: 10 },
      },
      series,
    }
  }, [comparison, metric, t])

  if (!comparison || !option) return null

  return (
    <section style={{
      marginBottom: 16,
      padding: '14px 16px 8px',
      background: '#fff',
      border: '1px solid #e5e7eb',
      borderRadius: 8,
    }}>
      <div style={{ marginBottom: 4 }}>
        <h3 style={{ margin: 0, fontSize: 14, color: '#374151' }}>
          {t('comparison.title')} · {comparison.anchor.label}
        </h3>
        <p style={{ margin: '4px 0 0', fontSize: 12, color: '#6b7280' }}>
          {filteredByContributors ? t('comparison.selectedContributors') : t('comparison.repoScope')}
          {!comparison.anchor.complete ? ` · ${t('comparison.inProgress')}` : ''}
        </p>
      </div>
      <ReactEChartsCore
        echarts={echarts}
        option={option}
        notMerge
        style={{ width: '100%', height: 230 }}
      />
    </section>
  )
}

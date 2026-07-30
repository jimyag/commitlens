import { useMemo } from 'react'
import type { EChartsOption } from 'echarts'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { LabelLayout } from 'echarts/features'
import { CanvasRenderer } from 'echarts/renderers'
import ReactEChartsCore from 'echarts-for-react/esm/core'
import type {
  ComparisonMetrics,
  ComparisonPoint,
  ComparisonSet,
  SelectedComparison,
} from '../api'
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
  comparisonSet: ComparisonSet | null
  selection: SelectedComparison | null
  anchorKey: string
  comparisonKeys: string[]
  metric: Metric
  filteredByContributors: boolean
  onAnchorChange: (period: string) => void
  onComparisonKeysChange: (periods: string[]) => void
}

const MAX_COMPARISON_PERIODS = 5

function metricValue(metrics: ComparisonMetrics, metric: keyof ComparisonMetrics) {
  return metrics[metric]
}

function formatChange(current: number, baseline: number) {
  const delta = current - baseline
  const sign = delta > 0 ? '+' : ''
  if (baseline === 0) return `${sign}${delta.toLocaleString()}`
  return `${sign}${delta.toLocaleString()} (${sign}${(delta / baseline * 100).toFixed(1)}%)`
}

export function ComparisonChart({
  comparisonSet,
  selection,
  anchorKey,
  comparisonKeys,
  metric,
  filteredByContributors,
  onAnchorChange,
  onComparisonKeysChange,
}: Props) {
  const { t } = useI18n()
  const anchorOptions = useMemo(
    () => comparisonSet?.periods.slice(1).reverse() ?? [],
    [comparisonSet],
  )
  const comparisonOptions = useMemo(() => {
    if (!comparisonSet || !anchorKey) return []
    const anchorIndex = comparisonSet.periods.indexOf(anchorKey)
    const points = new Map<string, ComparisonPoint>()
    comparisonSet.periods.slice(0, Math.max(0, anchorIndex)).forEach(period => {
      const point = comparisonSet.comparisons[period]?.anchor
      if (point?.available) points.set(point.key, point)
    })
    const automatic = comparisonSet.comparisons[anchorKey]
    if (automatic) {
      [automatic.previous, ...automatic.historical].forEach(point => {
        if (point.available && point.key !== anchorKey) points.set(point.key, point)
      })
    }
    return Array.from(points.values()).sort((a, b) => b.key.localeCompare(a.key))
  }, [anchorKey, comparisonSet])

  const option = useMemo<EChartsOption | null>(() => {
    if (!selection?.anchor.available) return null

    const points: ComparisonPoint[] = [
      selection.anchor,
      ...selection.comparisons,
    ]
    const categories = points.map((point, index) => {
      if (index === 0) return `${t('comparison.anchor')}\n${point.label}`
      return `${t('comparison.baseline')}\n${point.label}`
    })
    const anchor = selection.anchor.metrics
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
              output += `<div style="color:#d1d5db">${t('comparison.anchorDifference')}: ${formatChange(metricValue(anchor, key), item.value)}</div>`
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
  }, [metric, selection, t])

  if (!comparisonSet || anchorOptions.length === 0) return null

  const toggleComparison = (period: string) => {
    if (comparisonKeys.includes(period)) {
      if (comparisonKeys.length === 1) return
      onComparisonKeysChange(comparisonKeys.filter(key => key !== period))
      return
    }
    if (comparisonKeys.length >= MAX_COMPARISON_PERIODS) return
    onComparisonKeysChange([...comparisonKeys, period])
  }

  return (
    <section style={{
      marginBottom: 16,
      padding: '14px 16px 8px',
      background: '#fff',
      border: '1px solid #e5e7eb',
      borderRadius: 8,
    }}>
      <div style={{
        marginBottom: 8,
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
        gap: 16,
        flexWrap: 'wrap',
      }}>
        <div>
          <h3 style={{ margin: 0, fontSize: 14, color: '#374151' }}>
            {t('comparison.title')} · {selection?.anchor.label ?? anchorKey}
          </h3>
          <p style={{ margin: '4px 0 0', fontSize: 12, color: '#6b7280' }}>
            {filteredByContributors ? t('comparison.selectedContributors') : t('comparison.repoScope')}
            {selection?.anchor && !selection.anchor.complete ? ` · ${t('comparison.inProgress')}` : ''}
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'flex-end', gap: 8, flexWrap: 'wrap' }}>
          <label style={{ display: 'grid', gap: 3, fontSize: 11, color: '#6b7280' }}>
            {t('comparison.anchorPeriod')}
            <select
              value={anchorKey}
              onChange={event => onAnchorChange(event.target.value)}
              style={{
                height: 32,
                minWidth: 132,
                padding: '0 28px 0 10px',
                border: '1px solid #d1d5db',
                borderRadius: 6,
                background: '#fff',
                color: '#374151',
              }}
            >
              {anchorOptions.map(period => (
                <option key={period} value={period}>{period}</option>
              ))}
            </select>
          </label>
          <div style={{ display: 'grid', gap: 3, fontSize: 11, color: '#6b7280' }}>
            <span>{t('comparison.comparePeriods')}</span>
            <details style={{ position: 'relative' }}>
              <summary style={{
                boxSizing: 'border-box',
                height: 32,
                minWidth: 132,
                padding: '7px 10px',
                border: '1px solid #d1d5db',
                borderRadius: 6,
                background: '#fff',
                color: '#374151',
                cursor: 'pointer',
                listStylePosition: 'inside',
                userSelect: 'none',
              }}>
                {t('comparison.selectedPeriods').replace('{n}', String(comparisonKeys.length))}
              </summary>
              <div style={{
                position: 'absolute',
                zIndex: 20,
                top: 36,
                right: 0,
                width: 220,
                maxHeight: 280,
                overflowY: 'auto',
                padding: 6,
                border: '1px solid #d1d5db',
                borderRadius: 6,
                background: '#fff',
                boxShadow: '0 8px 24px rgba(15, 23, 42, 0.12)',
              }}>
                {comparisonOptions.map(point => {
                  const checked = comparisonKeys.includes(point.key)
                  const disabled = (checked && comparisonKeys.length === 1)
                    || (!checked && comparisonKeys.length >= MAX_COMPARISON_PERIODS)
                  return (
                    <label
                      key={point.key}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        padding: '7px 8px',
                        color: disabled ? '#9ca3af' : '#374151',
                        cursor: disabled ? 'not-allowed' : 'pointer',
                      }}
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        disabled={disabled}
                        onChange={() => toggleComparison(point.key)}
                      />
                      <span>{point.label}</span>
                    </label>
                  )
                })}
                <div style={{
                  borderTop: '1px solid #f3f4f6',
                  marginTop: 4,
                  padding: '7px 8px 3px',
                  color: '#9ca3af',
                  fontSize: 10,
                }}>
                  {t('comparison.selectionLimit')}
                </div>
              </div>
            </details>
          </div>
        </div>
      </div>
      {option ? (
        <ReactEChartsCore
          echarts={echarts}
          option={option}
          notMerge
          style={{ width: '100%', height: 230 }}
        />
      ) : (
        <div style={{ height: 230, display: 'grid', placeItems: 'center', color: '#9ca3af', fontSize: 12 }}>
          {t('comparison.loading')}
        </div>
      )}
    </section>
  )
}

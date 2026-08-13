import { useEffect, useMemo, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router'
import { useApp } from './context/AppContext'
import { mergeContributors, mergeWeekly } from './utils/statsUtils'
import {
  api,
  type ComparisonSet,
  type ContributorStats,
  type PeriodComparison,
  type SelectedComparison,
} from './api'
import { TrendChart } from './components/TrendChart'
import type { Granularity } from './components/TrendChart'
import { ComparisonChart } from './components/ComparisonChart'
import { RepositorySizeChart } from './components/RepositorySizeChart'
import { useI18n } from './i18n/I18nContext'
import type { MessageKey } from './i18n/bundles/en'

const granularityKey: Record<Granularity, MessageKey> = {
  week: 'app.granularity.week',
  month: 'app.granularity.month',
  quarter: 'app.granularity.quarter',
  year: 'app.granularity.year',
}

const MAX_COMPARISON_PERIODS = 5

function defaultComparisonKeys(comparison?: PeriodComparison) {
  if (!comparison) return []
  return [comparison.previous, ...comparison.historical]
    .filter(point => point.available)
    .map(point => point.key)
    .filter((key, index, keys) => key !== comparison.anchor.key && keys.indexOf(key) === index)
    .slice(0, MAX_COMPARISON_PERIODS)
}

export default function App({ metric = 'commits' }: { metric?: 'commits' | 'lines' }) {
  const { allStats } = useApp()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { t, tf } = useI18n()

  const selectedRepos = useMemo(() => {
    const val = searchParams.get('repo')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  const gran = (searchParams.get('gran') ?? 'week') as Granularity

  const selectedLogins = useMemo(() => {
    const val = searchParams.get('login')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  const filteredStats = useMemo(() => {
    if (selectedRepos.length === 0) return allStats
    return allStats.filter(s => selectedRepos.includes(s.repo))
  }, [allStats, selectedRepos])

  const contributors = useMemo(() => {
    const merged = mergeContributors(filteredStats)
    if (selectedLogins.length === 0) return merged

    const filtered: Record<string, ContributorStats> = {}
    selectedLogins.forEach(l => {
      if (merged[l]) filtered[l] = merged[l]
    })
    return filtered
  }, [filteredStats, selectedLogins])

  const weekly = mergeWeekly(filteredStats)
  const [comparisonSet, setComparisonSet] = useState<ComparisonSet | null>(null)
  const [selectedComparisonResult, setSelectedComparisonResult] = useState<{
    key: string
    value: SelectedComparison
  } | null>(null)
  const repositoryKey = selectedRepos.join(',')
  const loginKey = selectedLogins.join(',')

  useEffect(() => {
    let cancelled = false
    void api.getComparison({
      repositories: selectedRepos,
      logins: selectedLogins,
      granularity: gran,
    }).then(response => {
      if (!cancelled) setComparisonSet(response.data)
    }).catch(() => {
      if (!cancelled) setComparisonSet(null)
    })
    return () => { cancelled = true }
  }, [allStats, gran, loginKey, repositoryKey, selectedLogins, selectedRepos])

  const selectedAnchor = useMemo(() => {
    const requested = searchParams.get('compareAnchor')
    if (requested && (comparisonSet?.periods.indexOf(requested) ?? -1) > 0) return requested
    return (comparisonSet?.periods.length ?? 0) > 1 ? comparisonSet?.periods.at(-1) ?? '' : ''
  }, [comparisonSet, searchParams])

  const selectedComparisonPeriods = useMemo(() => {
    if (!comparisonSet || !selectedAnchor) return []
    const anchorIndex = comparisonSet.periods.indexOf(selectedAnchor)
    const automatic = defaultComparisonKeys(comparisonSet.comparisons[selectedAnchor])
    const allowed = new Set([
      ...comparisonSet.periods.slice(0, Math.max(0, anchorIndex)),
      ...automatic,
    ])
    const requested = searchParams.get('comparePeriods')
    if (requested === null) return automatic
    const selected = requested
      .split(',')
      .filter((key, index, keys) =>
        allowed.has(key) && key !== selectedAnchor && keys.indexOf(key) === index,
      )
      .slice(0, MAX_COMPARISON_PERIODS)
    return selected.length > 0 ? selected : automatic
  }, [comparisonSet, searchParams, selectedAnchor])
  const selectedComparisonKey = selectedComparisonPeriods.join(',')
  const selectionRequestKey = [
    gran,
    repositoryKey,
    loginKey,
    selectedAnchor,
    selectedComparisonKey,
  ].join('|')

  useEffect(() => {
    if (!selectedAnchor || selectedComparisonPeriods.length === 0) return

    let cancelled = false
    void api.getSelectedComparison({
      repositories: selectedRepos,
      logins: selectedLogins,
      granularity: gran,
      anchor: selectedAnchor,
      comparisons: selectedComparisonPeriods,
    }).then(response => {
      if (!cancelled) setSelectedComparisonResult({
        key: selectionRequestKey,
        value: response.data,
      })
    }).catch(() => undefined)
    return () => { cancelled = true }
  }, [
    allStats,
    gran,
    loginKey,
    repositoryKey,
    selectedAnchor,
    selectedComparisonKey,
    selectedComparisonPeriods,
    selectedLogins,
    selectedRepos,
    selectionRequestKey,
  ])
  const selectedComparison = selectedComparisonResult?.key === selectionRequestKey
    ? selectedComparisonResult.value
    : null

  const setComparisonSearch = (anchor: string, periods: string[]) => {
    const next = new URLSearchParams(searchParams)
    next.set('compareAnchor', anchor)
    next.set('comparePeriods', periods.join(','))
    navigate({ search: `?${next.toString()}` }, { replace: true })
  }

  const handleComparisonAnchorChange = (anchor: string) => {
    setComparisonSearch(anchor, defaultComparisonKeys(comparisonSet?.comparisons[anchor]))
  }

  const handleBarClick = (period: string, login: string | undefined) => {
    const next = new URLSearchParams(searchParams)
    next.set('period', period)
    if (login) {
      next.set('login', login)
    }
    navigate('/prs?' + next.toString())
  }

  const titleKey: MessageKey = metric === 'commits' ? 'app.section.trendTitle' : 'app.section.linesTrend'

  return (
    <div style={{
      boxSizing: 'border-box',
      minHeight: 'calc(100svh - 64px)',
      padding: '20px 5% 32px',
    }}>
      <section>
        {metric === 'lines' && (
          <div style={{ marginBottom: 24 }}>
            <div style={{ marginBottom: 12 }}>
              <h2 style={{ fontSize: 18, fontWeight: 600, margin: '0 0 6px', color: '#374151' }}>
                {t('repositorySize.title')}
              </h2>
              <p style={{ margin: 0, fontSize: 13, color: '#6b7280' }}>
                {t('repositorySize.description')}
              </p>
            </div>
            <div style={{
              background: '#fff',
              border: '1px solid #e5e7eb',
              borderRadius: 8,
              padding: 16,
            }}>
              <RepositorySizeChart stats={filteredStats} granularity={gran} />
            </div>
          </div>
        )}
        <div style={{ marginBottom: 12 }}>
          <h2 style={{ fontSize: 18, fontWeight: 600, margin: '0 0 6px', color: '#374151' }}>
            {t(titleKey)}
          </h2>
          <p style={{ margin: 0, fontSize: 13, color: '#6b7280' }}>{t('app.section.trendDesc')}</p>
          <p style={{ margin: '8px 0 0', fontSize: 13, color: '#6b7280' }}>
            {tf('chart.sub', { g: t(granularityKey[gran]) })}
          </p>
        </div>
        <ComparisonChart
          comparisonSet={comparisonSet}
          selection={selectedComparison}
          anchorKey={selectedAnchor}
          comparisonKeys={selectedComparisonPeriods}
          metric={metric}
          filteredByContributors={selectedLogins.length > 0}
          onAnchorChange={handleComparisonAnchorChange}
          onComparisonKeysChange={periods => setComparisonSearch(selectedAnchor, periods)}
        />
        <div style={{
          background: '#fff',
          border: '1px solid #e5e7eb',
          borderRadius: 8,
          padding: 16,
        }}>
          <TrendChart
            weekly={weekly}
            granularity={gran}
            metric={metric}
            contributors={contributors}
            selectedLogins={selectedLogins}
            comparisons={comparisonSet?.comparisons}
            onBarClick={handleBarClick}
          />
        </div>
      </section>
    </div>
  )
}

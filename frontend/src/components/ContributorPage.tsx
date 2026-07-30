import { useMemo } from 'react'
import { useSearchParams } from 'react-router'
import type { ContributorStats } from '../api'
import { useApp } from '../context/AppContext'
import { useI18n } from '../i18n/I18nContext'
import { toPeriodKey, type Granularity } from '../utils/periodUtils'
import { mergeContributors, mergeWeekly } from '../utils/statsUtils'
import { ContributorTable } from './ContributorTable'

export function ContributorPage() {
  const { allStats } = useApp()
  const [searchParams] = useSearchParams()
  const { t } = useI18n()

  const { contributors, commitFrequency } = useMemo(() => {
    const selectedRepos = searchParams.get('repo')?.split(',').filter(Boolean) ?? []
    const selectedLogins = searchParams.get('login')?.split(',').filter(Boolean) ?? []
    const granularity = (searchParams.get('gran') ?? 'week') as Granularity
    const stats = selectedRepos.length === 0
      ? allStats
      : allStats.filter(item => selectedRepos.includes(item.repo))
    const merged = mergeContributors(stats)
    const weekly = mergeWeekly(stats)
    const periods = [...new Set(Object.keys(weekly).map(period => toPeriodKey(period, granularity)))].sort()
    const frequencyByPeriod: Record<string, Record<string, number>> = {}

    Object.entries(weekly).forEach(([week, entry]) => {
      const period = toPeriodKey(week, granularity)
      if (!frequencyByPeriod[period]) frequencyByPeriod[period] = {}
      Object.entries(entry.contributors).forEach(([login, values]) => {
        frequencyByPeriod[period][login] = (frequencyByPeriod[period][login] ?? 0) + values.commits
      })
    })

    const commitFrequency: Record<string, number[]> = {}
    Object.keys(merged).forEach(login => {
      commitFrequency[login] = periods.map(period => frequencyByPeriod[period]?.[login] ?? 0)
    })

    if (selectedLogins.length === 0) return { contributors: merged, commitFrequency }

    const filtered: Record<string, ContributorStats> = {}
    selectedLogins.forEach(login => {
      if (merged[login]) filtered[login] = merged[login]
    })
    return { contributors: filtered, commitFrequency }
  }, [allStats, searchParams])

  return (
    <div style={{ padding: '20px 5% 40px' }}>
      <section>
        <h2 style={{ fontSize: 18, fontWeight: 600, margin: '0 0 12px', color: '#374151' }}>
          {t('app.section.rankTitle')}
        </h2>
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden' }}>
          <ContributorTable contributors={contributors} commitFrequency={commitFrequency} />
        </div>
      </section>
    </div>
  )
}

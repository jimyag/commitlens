import { useMemo } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useApp, mergeContributors, mergeWeekly } from './context/AppContext'
import { ContributorTable } from './components/ContributorTable'
import { TrendChart } from './components/TrendChart'
import type { Granularity } from './components/TrendChart'
import { useI18n } from './i18n/I18nContext'
import type { MessageKey } from './i18n/bundles/en'

const granularityKey: Record<Granularity, MessageKey> = {
  week: 'app.granularity.week',
  month: 'app.granularity.month',
  quarter: 'app.granularity.quarter',
  year: 'app.granularity.year',
}

export default function App() {
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
    
    // If specific logins are selected, we might want to filter the table too, 
    // but the user mostly complained about the chart. 
    // Let's filter the table to match the chart's focus.
    const filtered: Record<string, any> = {}
    selectedLogins.forEach(l => {
      if (merged[l]) filtered[l] = merged[l]
    })
    return filtered
  }, [filteredStats, selectedLogins])

  const weekly = mergeWeekly(filteredStats)

  const handleBarClick = (period: string, login: string | undefined) => {
    const next = new URLSearchParams(searchParams)
    next.set('period', period)
    if (login) {
      // If we clicked a personal bar, we probably want to focus on that person
      next.set('login', login)
    }
    navigate('/prs?' + next.toString())
  }

  return (
    <div style={{ padding: '20px 5% 32px' }}>
      <section style={{ marginBottom: 40 }}>
        <div style={{ marginBottom: 12 }}>
          <h2 style={{ fontSize: 18, fontWeight: 600, margin: '0 0 6px', color: '#374151' }}>
            {t('app.section.trendTitle')}
          </h2>
          <p style={{ margin: 0, fontSize: 13, color: '#6b7280' }}>{t('app.section.trendDesc')}</p>
          <p style={{ margin: '8px 0 0', fontSize: 13, color: '#6b7280' }}>
            {tf('chart.sub', { g: t(granularityKey[gran]) })}
          </p>
        </div>
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, padding: 16, background: '#fff' }}>
          <TrendChart
            weekly={weekly}
            granularity={gran}
            contributors={contributors}
            selectedLogins={selectedLogins}
            onBarClick={handleBarClick}
          />
        </div>
      </section>

      <section>
        <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 12, color: '#374151' }}>
          {t('app.section.rankTitle')}
        </h2>
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden' }}>
          <ContributorTable contributors={contributors} />
        </div>
      </section>
    </div>
  )
}

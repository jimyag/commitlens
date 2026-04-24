import { useEffect, useState } from 'react'
import { api } from './api'
import type { StatsData, ContributorStats, WeeklyEntry } from './api'
import { ContributorTable } from './components/ContributorTable'
import { TrendChart } from './components/TrendChart'
import type { Granularity } from './components/TrendChart'
import { useI18n, type Lang } from './i18n/I18nContext'
import type { MessageKey } from './i18n/bundles/en'

function mergeContributors(statsList: StatsData[]): Record<string, ContributorStats> {
  const merged: Record<string, ContributorStats> = {}
  for (const s of statsList) {
    for (const [login, c] of Object.entries(s?.contributors ?? {})) {
      if (!merged[login]) {
        merged[login] = { ...c }
      } else {
        merged[login].pr_count += c.pr_count
        merged[login].commit_count += c.commit_count
        merged[login].additions += c.additions
        merged[login].deletions += c.deletions
      }
    }
  }
  return merged
}

function mergeWeekly(statsList: (StatsData | undefined)[]): Record<string, WeeklyEntry> {
  const merged: Record<string, WeeklyEntry> = {}
  for (const s of statsList) {
    if (!s?.weekly) continue
    for (const [k, v] of Object.entries(s.weekly)) {
      if (!merged[k]) merged[k] = { total_prs: 0, contributors: {} }
      merged[k].total_prs += v.total_prs
      for (const [login, count] of Object.entries(v.contributors)) {
        merged[k].contributors[login] = (merged[k].contributors[login] ?? 0) + count
      }
    }
  }
  return merged
}

const granularityKey: Record<Granularity, MessageKey> = {
  week: 'app.granularity.week',
  month: 'app.granularity.month',
  quarter: 'app.granularity.quarter',
  year: 'app.granularity.year',
}

export default function App() {
  const { t, tf, lang, setLang } = useI18n()
  const [repos, setRepos] = useState<string[]>([])
  const [allStats, setAllStats] = useState<StatsData[]>([])
  const [selectedRepo, setSelectedRepo] = useState<string>('')
  const [granularity, setGranularity] = useState<Granularity>('week')
  const [syncing, setSyncing] = useState(false)
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null)

  const loadData = () => {
    api.getRepos().then(r => setRepos(r.data.repos))
    api.getStats().then(r => {
      const data = r.data as { stats: StatsData[] }
      setAllStats(data.stats ?? [])
      setLastSyncAt(Date.now())
    })
  }

  useEffect(() => {
    loadData()
  }, [])

  const timeLocale = lang === 'zh' ? 'zh-CN' : 'en-US'
  const lastSyncStr =
    lastSyncAt != null ? new Date(lastSyncAt).toLocaleTimeString(timeLocale) : ''

  const filteredStats = selectedRepo
    ? allStats.filter(s => s.repo === selectedRepo)
    : allStats

  const contributors = mergeContributors(filteredStats)
  const weekly = mergeWeekly(filteredStats)

  const handleSync = async () => {
    setSyncing(true)
    await api.sync(selectedRepo || undefined)
    setTimeout(() => {
      loadData()
      setSyncing(false)
    }, 3000)
  }

  return (
    <div
      style={{
        width: '100%',
        maxWidth: '100%',
        boxSizing: 'border-box',
        margin: 0,
        padding: '20px 5% 32px',
        fontFamily: 'system-ui, sans-serif',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 28, flexWrap: 'wrap' }}>
        <h1 style={{ margin: 0, fontSize: 24, fontWeight: 700, color: '#111' }}>CommitLens</h1>

        <label style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 14, color: '#374151' }}>
          <span>{t('app.language')}</span>
          <select
            value={lang}
            onChange={e => setLang(e.target.value as Lang)}
            style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14 }}
          >
            <option value="en">{t('lang.en')}</option>
            <option value="zh">{t('lang.zh')}</option>
          </select>
        </label>

        <select
          value={selectedRepo}
          onChange={e => setSelectedRepo(e.target.value)}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14 }}
        >
          <option value="">{t('app.allRepos')}</option>
          {repos.map(r => <option key={r} value={r}>{r}</option>)}
        </select>

        <select
          value={granularity}
          onChange={e => setGranularity(e.target.value as Granularity)}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14 }}
        >
          <option value="week">{t('app.granularity.week')}</option>
          <option value="month">{t('app.granularity.month')}</option>
          <option value="quarter">{t('app.granularity.quarter')}</option>
          <option value="year">{t('app.granularity.year')}</option>
        </select>

        <button
          onClick={handleSync}
          disabled={syncing}
          style={{
            padding: '6px 16px',
            borderRadius: 6,
            border: 'none',
            background: syncing ? '#9ca3af' : '#6366f1',
            color: '#fff',
            fontSize: 14,
            cursor: syncing ? 'not-allowed' : 'pointer',
          }}
        >
          {syncing ? t('app.syncing') : t('app.refresh')}
        </button>

        {lastSyncAt != null && (
          <span style={{ color: '#9ca3af', fontSize: 13 }}>
            {t('app.lastSync')} {lastSyncStr}
          </span>
        )}
      </div>

      <section style={{ marginBottom: 40 }}>
        <div style={{ marginBottom: 12 }}>
          <h2 style={{ fontSize: 18, fontWeight: 600, margin: '0 0 6px', color: '#374151' }}>
            {t('app.section.trendTitle')}
          </h2>
          <p style={{ margin: 0, fontSize: 13, color: '#6b7280' }}>{t('app.section.trendDesc')}</p>
          <p style={{ margin: '8px 0 0', fontSize: 13, color: '#6b7280' }}>
            {tf('chart.sub', { g: t(granularityKey[granularity]) })}
          </p>
        </div>
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, padding: 16, background: '#fff' }}>
          <TrendChart weekly={weekly} granularity={granularity} contributors={contributors} />
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

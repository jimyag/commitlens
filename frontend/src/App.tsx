import { useEffect, useState } from 'react'
import { api } from './api'
import type { StatsData, ContributorStats, WeeklyEntry } from './api'
import { ContributorTable } from './components/ContributorTable'
import { TrendChart } from './components/TrendChart'
import type { Granularity } from './components/TrendChart'

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

export default function App() {
  const [repos, setRepos] = useState<string[]>([])
  const [allStats, setAllStats] = useState<StatsData[]>([])
  const [selectedRepo, setSelectedRepo] = useState<string>('')
  const [granularity, setGranularity] = useState<Granularity>('week')
  const [selectedLogin, setSelectedLogin] = useState<string>('')
  const [syncing, setSyncing] = useState(false)
  const [lastSync, setLastSync] = useState<string>('')

  const loadData = () => {
    api.getRepos().then(r => setRepos(r.data.repos))
    api.getStats().then(r => {
      const data = r.data as { stats: StatsData[] }
      setAllStats(data.stats ?? [])
      setLastSync(new Date().toLocaleTimeString())
    })
  }

  useEffect(() => { loadData() }, [])

  const filteredStats = selectedRepo
    ? allStats.filter(s => s.repo === selectedRepo)
    : allStats

  const contributors = mergeContributors(filteredStats)
  const weekly = mergeWeekly(filteredStats)
  const allLogins = Object.keys(contributors).sort()

  const handleSync = async () => {
    setSyncing(true)
    await api.sync(selectedRepo || undefined)
    setTimeout(() => {
      loadData()
      setSyncing(false)
    }, 3000)
  }

  return (
    <div style={{ maxWidth: 1200, margin: '0 auto', padding: '24px 16px', fontFamily: 'system-ui, sans-serif' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 32, flexWrap: 'wrap' }}>
        <h1 style={{ margin: 0, fontSize: 24, fontWeight: 700, color: '#111' }}>CommitLens</h1>

        <select
          value={selectedRepo}
          onChange={e => setSelectedRepo(e.target.value)}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14 }}
        >
          <option value="">全部仓库</option>
          {repos.map(r => <option key={r} value={r}>{r}</option>)}
        </select>

        <select
          value={granularity}
          onChange={e => setGranularity(e.target.value as Granularity)}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14 }}
        >
          <option value="week">按周</option>
          <option value="month">按月</option>
          <option value="quarter">按季度</option>
          <option value="year">按年</option>
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
          {syncing ? '同步中...' : '刷新数据'}
        </button>

        {lastSync && (
          <span style={{ color: '#9ca3af', fontSize: 13 }}>上次更新: {lastSync}</span>
        )}
      </div>

      <section style={{ marginBottom: 40 }}>
        <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 12, color: '#374151' }}>贡献者排行</h2>
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden' }}>
          <ContributorTable contributors={contributors} />
        </div>
      </section>

      <section>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 12, flexWrap: 'wrap' }}>
          <h2 style={{ fontSize: 18, fontWeight: 600, margin: 0, color: '#374151' }}>合并 PR 趋势</h2>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 13, color: '#6b7280' }}>叠加贡献者:</span>
            <select
              value={selectedLogin}
              onChange={e => setSelectedLogin(e.target.value)}
              style={{ padding: '4px 8px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 13 }}
            >
              <option value="">不叠加</option>
              {allLogins.map(l => <option key={l} value={l}>{l}</option>)}
            </select>
          </div>
        </div>
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, padding: 16, background: '#fff' }}>
          <TrendChart
            weekly={weekly}
            granularity={granularity}
            selectedLogin={selectedLogin || undefined}
          />
        </div>
      </section>
    </div>
  )
}

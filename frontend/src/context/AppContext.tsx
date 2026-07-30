/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useState } from 'react'
import { api } from '../api'
import type { StatsData, ContributorStats } from '../api'
import { mergeContributors } from '../utils/statsUtils'

interface AppContextValue {
  repos: string[]
  defaultSyncRepos: string[]
  allStats: StatsData[]
  allContributors: Record<string, ContributorStats>
  syncing: boolean
  lastSyncAt: number | null
  loadData: () => Promise<void>
  syncRepositories: (repositories?: string[]) => Promise<void>
}

const AppContext = createContext<AppContextValue | null>(null)

async function fetchAppData() {
  const [repositoryResponse, statsResponse] = await Promise.all([api.getRepos(), api.getStats()])
  const stats = statsResponse.data as { stats: StatsData[] }
  return {
    repositories: repositoryResponse.data.repos,
    defaultSyncRepos: repositoryResponse.data.default_repositories ?? [],
    stats: stats.stats ?? [],
  }
}

export function AppProvider({ children }: { children: React.ReactNode }) {
  const [repos, setRepos] = useState<string[]>([])
  const [defaultSyncRepos, setDefaultSyncRepos] = useState<string[]>([])
  const [allStats, setAllStats] = useState<StatsData[]>([])
  const [syncing, setSyncing] = useState(false)
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null)

  const loadData = async () => {
    const data = await fetchAppData()
    setRepos(data.repositories)
    setDefaultSyncRepos(data.defaultSyncRepos)
    setAllStats(data.stats)
    setLastSyncAt(Date.now())
  }

  useEffect(() => {
    let cancelled = false
    void fetchAppData().then(data => {
      if (cancelled) return
      setRepos(data.repositories)
      setDefaultSyncRepos(data.defaultSyncRepos)
      setAllStats(data.stats)
      setLastSyncAt(Date.now())
    })
    return () => { cancelled = true }
  }, [])

  const syncRepositories = async (repositories?: string[]) => {
    setSyncing(true)
    try {
      await api.sync(repositories)
      await loadData()
    } finally {
      setSyncing(false)
    }
  }

  const allContributors = mergeContributors(allStats)

  return (
    <AppContext.Provider value={{
      repos,
      defaultSyncRepos,
      allStats,
      allContributors,
      syncing,
      lastSyncAt,
      loadData,
      syncRepositories,
    }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp must be used within AppProvider')
  return ctx
}

/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useState } from 'react'
import { api } from '../api'
import type { StatsData, ContributorStats } from '../api'
import { mergeContributors } from '../utils/statsUtils'

interface AppContextValue {
  repos: string[]
  allStats: StatsData[]
  allContributors: Record<string, ContributorStats>
  syncing: boolean
  lastSyncAt: number | null
  loadData: () => void
  syncRepo: (repo?: string) => Promise<void>
}

const AppContext = createContext<AppContextValue | null>(null)

export function AppProvider({ children }: { children: React.ReactNode }) {
  const [repos, setRepos] = useState<string[]>([])
  const [allStats, setAllStats] = useState<StatsData[]>([])
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

  useEffect(() => { loadData() }, [])

  const syncRepo = async (repo?: string) => {
    setSyncing(true)
    await api.sync(repo)
    setTimeout(() => {
      loadData()
      setSyncing(false)
    }, 3000)
  }

  const allContributors = mergeContributors(allStats)

  return (
    <AppContext.Provider value={{ repos, allStats, allContributors, syncing, lastSyncAt, loadData, syncRepo }}>
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const ctx = useContext(AppContext)
  if (!ctx) throw new Error('useApp must be used within AppProvider')
  return ctx
}

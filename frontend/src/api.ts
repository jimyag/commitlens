import axios from 'axios'

export interface ContributorStats {
  login: string
  avatar_url: string
  pr_count: number
  commit_count: number
  additions: number
  deletions: number
}

export interface WeeklyEntry {
  total_prs: number
  contributors: Record<string, number>
}

export interface StatsData {
  repo: string
  computed_at: string
  contributors: Record<string, ContributorStats>
  weekly: Record<string, WeeklyEntry>
}

export const api = {
  getRepos: () => axios.get<{ repos: string[] }>('/api/repos'),
  getStats: (repo?: string) =>
    repo
      ? axios.get<StatsData>('/api/stats', { params: { repo } })
      : axios.get<{ stats: StatsData[] }>('/api/stats'),
  sync: (repo?: string) =>
    axios.post('/api/sync', null, { params: repo ? { repo } : {} }),
}

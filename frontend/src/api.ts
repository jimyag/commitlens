import axios from 'axios'

export interface ContributorStats {
  login: string
  avatar_url: string
  commit_count: number
  additions: number
  deletions: number
}

export interface WeeklyEntry {
  total_commits: number
  contributors: Record<string, number>
}

export interface StatsData {
  repo: string
  computed_at: string
  contributors: Record<string, ContributorStats>
  weekly: Record<string, WeeklyEntry>
}

export interface CommitInfo {
  repo: string
  sha: string
  title: string
  author: string
  participants: string[]
  date: string
  additions: number
  deletions: number
}

export const api = {
  getRepos: () => axios.get<{ repos: string[] }>('/api/repos'),
  getStats: (repo?: string) =>
    repo
      ? axios.get<StatsData>('/api/stats', { params: { repo } })
      : axios.get<{ stats: StatsData[] }>('/api/stats'),
  sync: (repo?: string) =>
    axios.post('/api/sync', null, { params: repo ? { repo } : {} }),
  getCommits: (params: { repo?: string; from?: string; to?: string; login?: string; page?: number; per_page?: number }) =>
    axios.get<{ commits: CommitInfo[]; total: number; page: number; per_page: number }>('/api/commits', { params }),
}

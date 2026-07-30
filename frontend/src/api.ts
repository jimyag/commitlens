import axios from 'axios'

export interface ContributorStats {
  login: string
  avatar_url: string
  commit_count: number
  additions: number
  deletions: number
}

export interface ContributorWeeklyStats {
  commits: number
  additions: number
  deletions: number
}

export interface WeeklyEntry {
  total_commits: number
  total_additions: number
  total_deletions: number
  contributors: Record<string, ContributorWeeklyStats>
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

export interface RepositoryList {
  repos: string[]
  default_repositories: string[]
}

export interface ComparisonMetrics {
  commits: number
  additions: number
  deletions: number
}

export interface ComparisonPoint {
  key: string
  label: string
  available: boolean
  complete: boolean
  metrics: ComparisonMetrics
}

export interface PeriodComparison {
  anchor: ComparisonPoint
  previous: ComparisonPoint
  historical: ComparisonPoint[]
}

export interface ComparisonSet {
  granularity: 'week' | 'month' | 'quarter' | 'year'
  periods: string[]
  comparisons: Record<string, PeriodComparison>
}

export const api = {
  getRepos: () => axios.get<RepositoryList>('/api/repos'),
  getStats: (repo?: string) =>
    repo
      ? axios.get<StatsData>('/api/stats', { params: { repo } })
      : axios.get<{ stats: StatsData[] }>('/api/stats'),
  sync: (repositories?: string[]) =>
    axios.post('/api/sync', { repositories: repositories ?? [] }),
  getComparison: (params: { repositories?: string[]; logins?: string[]; granularity: string }) =>
    axios.get<ComparisonSet>('/api/comparison', {
      params: {
        repo: params.repositories?.join(',') || undefined,
        login: params.logins?.join(',') || undefined,
        gran: params.granularity,
      },
    }),
  getCommits: (params: { repo?: string; from?: string; to?: string; login?: string; page?: number; per_page?: number }) =>
    axios.get<{ commits: CommitInfo[]; total: number; page: number; per_page: number }>('/api/commits', { params }),
}

import type { StatsData, ContributorStats, WeeklyEntry } from '../api'

export function mergeContributors(statsList: StatsData[]): Record<string, ContributorStats> {
  const merged: Record<string, ContributorStats> = {}
  for (const s of statsList) {
    for (const [login, c] of Object.entries(s?.contributors ?? {})) {
      if (!merged[login]) {
        merged[login] = { ...c }
      } else {
        merged[login].commit_count += c.commit_count
        merged[login].additions += c.additions
        merged[login].deletions += c.deletions
      }
    }
  }
  return merged
}

export function mergeWeekly(statsList: (StatsData | undefined)[]): Record<string, WeeklyEntry> {
  const merged: Record<string, WeeklyEntry> = {}
  for (const s of statsList) {
    if (!s?.weekly) continue
    for (const [k, v] of Object.entries(s.weekly)) {
      if (!merged[k]) merged[k] = { total_commits: 0, total_additions: 0, total_deletions: 0, contributors: {} }
      merged[k].total_commits += v.total_commits
      merged[k].total_additions += v.total_additions
      merged[k].total_deletions += v.total_deletions
      for (const [login, stats] of Object.entries(v.contributors)) {
        if (!merged[k].contributors[login]) {
          merged[k].contributors[login] = { ...stats }
        } else {
          merged[k].contributors[login].commits += stats.commits
          merged[k].contributors[login].additions += stats.additions
          merged[k].contributors[login].deletions += stats.deletions
        }
      }
    }
  }
  return merged
}

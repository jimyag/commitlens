import { useMemo, useEffect, useState, useRef } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useApp } from '../context/AppContext'
import { useI18n } from '../i18n/I18nContext'
import { toPeriodKey, type Granularity, periodToDateRange } from '../utils/periodUtils'
import { api, type CommitInfo } from '../api'

const PER_PAGE = 30

/** 头像小组件 */
function Avatar({ login, avatarUrl }: { login: string; avatarUrl?: string }) {
  if (avatarUrl) {
    return (
      <img
        src={avatarUrl}
        alt={login}
        style={{ width: 22, height: 22, borderRadius: '50%', border: '1px solid #e5e7eb' }}
      />
    )
  }
  return (
    <span
      style={{
        width: 22,
        height: 22,
        borderRadius: '50%',
        background: '#e5e7eb',
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 10,
        color: '#6b7280',
      }}
    >
      {login.slice(0, 1).toUpperCase()}
    </span>
  )
}

export function PRListPage() {
  const { allStats, allContributors } = useApp()
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const { t, tf } = useI18n()

  const repo = searchParams.get('repo') || ''
  const gran = (searchParams.get('gran') || 'week') as Granularity
  const period = searchParams.get('period') || ''
  const selectedLogins = useMemo(() => {
    const val = searchParams.get('login')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  const [commits, setCommits] = useState<CommitInfo[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  // Reset page to 1 when any filter (except page itself) changes
  const filterKey = `${period}|${gran}|${repo}|${selectedLogins.join(',')}`
  const prevFilterKey = useRef(filterKey)

  useEffect(() => {
    if (prevFilterKey.current !== filterKey) {
      prevFilterKey.current = filterKey
      setPage(1)
    }
  }, [filterKey])

  useEffect(() => {
    let ignore = false
    // Use async update to avoid "setState synchronously within effect" lint error
    Promise.resolve().then(() => {
      if (!ignore) setLoading(true)
    })
    const params: { repo?: string; from?: string; to?: string; login?: string; page?: number; per_page?: number } = {
      repo: repo || undefined,
      login: selectedLogins.length > 0 ? selectedLogins.join(',') : undefined,
      page,
      per_page: PER_PAGE,
    }
    if (period) {
      const range = periodToDateRange(period, gran)
      params.from = range.from
      params.to = range.to
    }
    api.getCommits(params)
      .then(res => {
        if (!ignore) {
          setCommits(res.data.commits ?? [])
          setTotal(res.data.total ?? 0)
          setLoading(false)
        }
      })
      .catch(() => {
        if (!ignore) setLoading(false)
      })
    return () => { ignore = true }
  }, [filterKey, page, gran, period, repo, selectedLogins])

  // 从 stats 数据推导当前 gran + repo 下所有可用 period
  const availablePeriods = useMemo(() => {
    const statsToUse = repo ? allStats.filter(s => s.repo === repo) : allStats
    const set = new Set<string>()
    for (const s of statsToUse) {
      for (const wk of Object.keys(s.weekly ?? {})) {
        set.add(toPeriodKey(wk, gran))
      }
    }
    return Array.from(set).sort().reverse()
  }, [allStats, repo, gran])

  const multiRepo = useMemo(() => new Set(commits.map(c => c.repo)).size > 1, [commits])
  const totalPages = Math.ceil(total / PER_PAGE)

  const patch = (key: string, value: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value)
      else next.delete(key)
      return next
    })
  }

  const handleBack = () => {
    const next = new URLSearchParams(searchParams)
    next.delete('period')
    navigate('/?' + next.toString())
  }

  const getAvatarUrl = (login: string) =>
    allContributors[login]?.avatar_url

  return (
    <div style={{ padding: '20px 5% 40px' }}>
      {/* 页头 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <button
            onClick={handleBack}
            style={{
              padding: '6px 12px', borderRadius: 6, border: '1px solid #d1d5db',
              background: '#fff', cursor: 'pointer', fontSize: 13,
            }}
          >
            {t('prPage.back')}
          </button>
          <h1 style={{ fontSize: 20, fontWeight: 700, color: '#111827', margin: 0 }}>
            {t('prPage.title').replace('{period}', period || t('prPage.allPeriods'))}
          </h1>
        </div>
        {!loading && (
          <span style={{ fontSize: 13, color: '#6b7280' }}>
            {tf('prPage.total', { n: total })}
          </span>
        )}
      </div>

      {/* 过滤栏：仅保留时间段选择器 */}
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'flex-start', gap: 12, flexWrap: 'wrap' }}>
        {/* 时间段下拉 */}
        <select
          value={period}
          onChange={e => patch('period', e.target.value)}
          style={{
            padding: '5px 10px', borderRadius: 6, border: '1px solid #d1d5db',
            fontSize: 13, background: '#fff', color: '#374151', flexShrink: 0,
          }}
        >
          <option value="">{t('prPage.allPeriods')}</option>
          {availablePeriods.map(p => (
            <option key={p} value={p}>{p}</option>
          ))}
        </select>
      </div>

      {/* Commit 表格 */}
      <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden', background: '#fff' }}>
        {loading ? (
          <div style={{ padding: 40, textAlign: 'center', color: '#9ca3af' }}>{t('app.syncing')}</div>
        ) : commits.length === 0 ? (
          <div style={{ padding: 40, textAlign: 'center', color: '#9ca3af' }}>{t('prPage.empty')}</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead style={{ background: '#f9fafb', borderBottom: '1px solid #e5e7eb' }}>
              <tr>
                {multiRepo && <th style={thStyle}>{t('prPage.colRepo')}</th>}
                <th style={thStyle}>{t('prPage.colPR')}</th>
                <th style={thStyle}>{t('table.commits')}</th>
                <th style={thStyle}>{t('prPage.colAuthors')}</th>
                <th style={thStyle}>{t('prPage.colDate')}</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>{t('prPage.colLines')}</th>
              </tr>
            </thead>
            <tbody>
              {commits.map(c => (
                <tr key={c.sha} style={{ borderBottom: '1px solid #f3f4f6' }}>
                  {multiRepo && <td style={tdStyle}>{c.repo}</td>}
                  <td style={tdStyle}>
                    <code style={{ color: '#4f46e5', fontWeight: 600 }}>{c.sha.slice(0, 7)}</code>
                  </td>
                  <td style={{ ...tdStyle, fontWeight: 500, color: '#111827' }}>{c.title}</td>
                  <td style={tdStyle}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <Avatar login={c.author} avatarUrl={getAvatarUrl(c.author)} />
                      {c.author}
                    </div>
                  </td>
                  <td style={{ ...tdStyle, color: '#6b7280' }}>
                    {new Date(c.date).toLocaleDateString()}
                  </td>
                  <td style={{ ...tdStyle, textAlign: 'right', whiteSpace: 'nowrap' }}>
                    <span style={{ color: '#059669', marginRight: 8 }}>+{c.additions}</span>
                    <span style={{ color: '#dc2626' }}>-{c.deletions}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* 分页 */}
      {!loading && totalPages > 1 && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12, marginTop: 24 }}>
          <button
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
            style={pageBtn(page <= 1)}
          >
            {t('prPage.pagePrev')}
          </button>
          <span style={{ fontSize: 13, color: '#374151' }}>
            {tf('prPage.pageOf', { page, total: totalPages })}
          </span>
          <button
            disabled={page >= totalPages}
            onClick={() => setPage(page + 1)}
            style={pageBtn(page >= totalPages)}
          >
            {t('prPage.pageNext')}
          </button>
        </div>
      )}
    </div>
  )
}

const thStyle: React.CSSProperties = {
  padding: '12px 16px', textAlign: 'left', fontSize: 12,
  fontWeight: 600, color: '#4b5563', textTransform: 'uppercase', letterSpacing: '0.05em',
}

const tdStyle: React.CSSProperties = {
  padding: '12px 16px', color: '#374151',
}

function pageBtn(disabled: boolean): React.CSSProperties {
  return {
    padding: '5px 14px', borderRadius: 6, border: '1px solid #d1d5db',
    background: disabled ? '#f9fafb' : '#fff',
    color: disabled ? '#9ca3af' : '#374151',
    cursor: disabled ? 'not-allowed' : 'pointer',
    fontSize: 13, fontWeight: 500,
  }
}

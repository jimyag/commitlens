import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import type { PRInfo } from '../api'
import { api } from '../api'
import { toPeriodKey, periodToDateRange, type Granularity } from '../utils/periodUtils'
import { useI18n } from '../i18n/I18nContext'
import { useApp } from '../context/AppContext'

const PER_PAGE = 100

function formatMergedAt(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function Avatar({ login, avatarUrl }: { login: string; avatarUrl?: string }) {
  if (avatarUrl) {
    return (
      <img src={avatarUrl} alt={login} title={login} width={22} height={22}
        style={{ borderRadius: '50%', flexShrink: 0, border: '1px solid #e5e7eb' }} />
    )
  }
  return (
    <span title={login} style={{
      width: 22, height: 22, borderRadius: '50%', background: '#e5e7eb',
      display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
      fontSize: 10, color: '#6b7280', flexShrink: 0,
    }}>
      {login.slice(0, 1).toUpperCase()}
    </span>
  )
}

export function PRListPage() {
  const { allStats, allContributors } = useApp()
  const { t, tf } = useI18n()
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  const period = searchParams.get('period') ?? ''
  const gran = (searchParams.get('gran') ?? 'week') as Granularity
  const repo = searchParams.get('repo') ?? ''
  const selectedLogin = searchParams.get('login') ?? ''

  const [prs, setPRs] = useState<PRInfo[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  // Reset page to 1 when any filter (except page itself) changes
  const filterKey = `${period}|${gran}|${repo}|${selectedLogin}`
  const prevFilterKey = useRef(filterKey)
  useEffect(() => {
    if (prevFilterKey.current !== filterKey) {
      prevFilterKey.current = filterKey
      if (page !== 1) {
        setPage(1)
        return
      }
    }
    setLoading(true)
    const params: Parameters<typeof api.getPRs>[0] = {
      repo: repo || undefined,
      login: selectedLogin || undefined,
      page,
      per_page: PER_PAGE,
    }
    if (period) {
      const range = periodToDateRange(period, gran)
      params.from = range.from
      params.to = range.to
    }
    api.getPRs(params)
      .then(res => {
        setPRs(res.data.prs ?? [])
        setTotal(res.data.total ?? 0)
      })
      .finally(() => setLoading(false))
  }, [filterKey, page]) // eslint-disable-line react-hooks/exhaustive-deps

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

  // 从 stats 数据推导当前 period 下的参与者（用于 pill 过滤器）
  const participantsInPeriod = useMemo(() => {
    const statsToUse = repo ? allStats.filter(s => s.repo === repo) : allStats
    const loginSet = new Set<string>()
    for (const s of statsToUse) {
      for (const [wk, entry] of Object.entries(s.weekly ?? {})) {
        const p = toPeriodKey(wk, gran)
        if (!period || p === period) {
          for (const login of Object.keys(entry.contributors)) {
            loginSet.add(login)
          }
        }
      }
    }
    return Array.from(loginSet).sort()
  }, [allStats, repo, gran, period])

  const multiRepo = useMemo(() => new Set(prs.map(pr => pr.repo)).size > 1, [prs])
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

  const getAvatarUrl = (login: string, pr: PRInfo) =>
    login === pr.author ? pr.avatar_url : allContributors[login]?.avatar_url

  return (
    <div style={{ padding: '20px 5% 40px' }}>
      {/* 页头 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
        <button onClick={handleBack} style={backBtnStyle}>
          {t('prPage.back')}
        </button>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600, color: '#111827' }}>
          {t('prPage.allPRs')}
        </h2>
        {!loading && (
          <span style={{ fontSize: 13, color: '#6b7280' }}>
            {tf('prPage.total', { n: total })}
          </span>
        )}
      </div>

      {/* 过滤栏：时间段选择器 + 贡献者 pills */}
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

        {/* 贡献者 pills */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
          <button onClick={() => patch('login', '')} style={pillStyle(!selectedLogin)}>
            {t('prPage.filterAll')}
          </button>
          {participantsInPeriod.map(login => {
            const avatarUrl = allContributors[login]?.avatar_url
            return (
              <button key={login} onClick={() => patch('login', login)} style={pillStyle(selectedLogin === login)}>
                <Avatar login={login} avatarUrl={avatarUrl} />
                {login}
              </button>
            )
          })}
        </div>
      </div>

      {/* PR 表格 */}
      <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden', background: '#fff' }}>
        {loading ? (
          <div style={{ padding: '40px 24px', textAlign: 'center', color: '#9ca3af', fontSize: 14 }}>
            {t('prPage.loading')}
          </div>
        ) : prs.length === 0 ? (
          <div style={{ padding: '40px 24px', textAlign: 'center', color: '#9ca3af', fontSize: 14 }}>
            {t('prPage.empty')}
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr style={{ background: '#f9fafb', borderBottom: '1px solid #e5e7eb' }}>
                {multiRepo && <th style={thStyle}>{t('prPage.colRepo')}</th>}
                <th style={thStyle}>{t('prPage.colPR')}</th>
                <th style={{ ...thStyle, whiteSpace: 'nowrap' }}>{t('prPage.colAuthors')}</th>
                <th style={{ ...thStyle, whiteSpace: 'nowrap' }}>{t('prPage.colDate')}</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>{t('prPage.colLines')}</th>
              </tr>
            </thead>
            <tbody>
              {prs.map((pr, idx) => {
                const participants = pr.participants?.length ? pr.participants : [pr.author]
                return (
                  <tr key={`${pr.repo}-${pr.number}`} style={{
                    borderBottom: idx < prs.length - 1 ? '1px solid #f3f4f6' : 'none',
                    background: idx % 2 === 0 ? '#fff' : '#fafafa',
                  }}>
                    {multiRepo && (
                      <td style={{ ...tdStyle, color: '#6b7280', whiteSpace: 'nowrap' }}>{pr.repo}</td>
                    )}
                    <td style={{ ...tdStyle, minWidth: 240 }}>
                      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
                        <a
                          href={`https://github.com/${pr.repo}/pull/${pr.number}`}
                          target="_blank" rel="noopener noreferrer"
                          style={{ color: '#6366f1', textDecoration: 'none', fontWeight: 500, whiteSpace: 'nowrap', flexShrink: 0 }}
                          onMouseOver={e => (e.currentTarget.style.textDecoration = 'underline')}
                          onMouseOut={e => (e.currentTarget.style.textDecoration = 'none')}
                        >
                          #{pr.number}
                        </a>
                        <span style={{ color: '#111827', lineHeight: 1.5 }}>{pr.title}</span>
                      </div>
                    </td>
                    <td style={{ ...tdStyle, whiteSpace: 'nowrap' }}>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {participants.map(login => (
                          <div key={login} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <Avatar login={login} avatarUrl={getAvatarUrl(login, pr)} />
                            <span style={{ color: '#374151' }}>{login}</span>
                          </div>
                        ))}
                      </div>
                    </td>
                    <td style={{ ...tdStyle, color: '#6b7280', whiteSpace: 'nowrap' }}>
                      {formatMergedAt(pr.merged_at)}
                    </td>
                    <td style={{ ...tdStyle, textAlign: 'right', whiteSpace: 'nowrap' }}>
                      <span style={{ color: '#16a34a' }}>+{pr.additions}</span>
                      {' '}
                      <span style={{ color: '#dc2626' }}>-{pr.deletions}</span>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* 分页控件 */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 16, justifyContent: 'center' }}>
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page <= 1 || loading}
            style={pageBtn(page <= 1 || loading)}
          >
            {t('prPage.pagePrev')}
          </button>
          <span style={{ fontSize: 13, color: '#6b7280' }}>
            {tf('prPage.pageOf', { page, total: totalPages })}
          </span>
          <button
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages || loading}
            style={pageBtn(page >= totalPages || loading)}
          >
            {t('prPage.pageNext')}
          </button>
        </div>
      )}
    </div>
  )
}

function pillStyle(active: boolean): React.CSSProperties {
  return {
    padding: '5px 12px', borderRadius: 16, border: '1px solid',
    borderColor: active ? '#6366f1' : '#d1d5db',
    background: active ? '#eef2ff' : '#fff',
    color: active ? '#4f46e5' : '#374151',
    fontSize: 13, cursor: 'pointer', fontWeight: active ? 600 : 400,
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }
}

function pageBtn(disabled: boolean): React.CSSProperties {
  return {
    padding: '5px 14px', borderRadius: 6, border: '1px solid #d1d5db',
    background: disabled ? '#f9fafb' : '#fff',
    color: disabled ? '#9ca3af' : '#374151',
    fontSize: 13, cursor: disabled ? 'not-allowed' : 'pointer',
  }
}

const backBtnStyle: React.CSSProperties = {
  background: 'none', border: '1px solid #d1d5db', borderRadius: 6,
  padding: '5px 14px', fontSize: 13, cursor: 'pointer', color: '#374151',
}

const thStyle: React.CSSProperties = {
  padding: '10px 16px', textAlign: 'left', fontWeight: 500, color: '#6b7280',
}

const tdStyle: React.CSSProperties = {
  padding: '10px 16px', verticalAlign: 'top',
}

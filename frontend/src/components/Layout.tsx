import { Outlet, Link, NavLink, useSearchParams } from 'react-router-dom'
import { useApp } from '../context/AppContext'
import { useI18n, type Lang } from '../i18n/I18nContext'
import type { Granularity } from './TrendChart'

function navLinkStyle(isActive: boolean): React.CSSProperties {
  return {
    padding: '4px 10px',
    borderRadius: 6,
    fontSize: 13,
    fontWeight: isActive ? 600 : 400,
    color: isActive ? '#4f46e5' : '#6b7280',
    background: isActive ? '#eef2ff' : 'transparent',
    textDecoration: 'none',
    display: 'inline-block',
  }
}

const selectStyle: React.CSSProperties = {
  padding: '5px 10px',
  borderRadius: 6,
  border: '1px solid #d1d5db',
  fontSize: 13,
  background: '#fff',
  color: '#374151',
}

export function Layout() {
  const { repos, allContributors, syncing, lastSyncAt, syncRepo } = useApp()
  const { t, lang, setLang } = useI18n()
  const [searchParams, setSearchParams] = useSearchParams()

  const repo = searchParams.get('repo') ?? ''
  const gran = (searchParams.get('gran') ?? 'week') as Granularity
  const login = searchParams.get('login') ?? ''

  const patch = (key: string, value: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value)
      else next.delete(key)
      return next
    })
  }

  const timeLocale = lang === 'zh' ? 'zh-CN' : 'en-US'
  const lastSyncStr = lastSyncAt != null ? new Date(lastSyncAt).toLocaleTimeString(timeLocale) : ''

  const contributorList = Object.entries(allContributors)
    .sort((a, b) => b[1].pr_count - a[1].pr_count)
    .map(([l]) => l)

  return (
    <div style={{ fontFamily: 'system-ui, sans-serif', minHeight: '100vh', background: '#f9fafb' }}>
      <header
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 10,
          background: '#fff',
          borderBottom: '1px solid #e5e7eb',
          padding: '10px 5%',
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          flexWrap: 'wrap',
        }}
      >
        <Link
          to="/"
          style={{ fontWeight: 700, fontSize: 18, color: '#111', textDecoration: 'none', marginRight: 2 }}
        >
          CommitLens
        </Link>

        {/* 主导航 */}
        <nav style={{ display: 'flex', gap: 2, marginRight: 8 }}>
          <NavLink
            to={{ pathname: '/', search: searchParams.toString() }}
            style={({ isActive }) => navLinkStyle(isActive)}
          >
            {t('nav.dashboard')}
          </NavLink>
          <NavLink
            to={{ pathname: '/prs', search: (() => { const p = new URLSearchParams(searchParams); p.delete('period'); return p.toString() })() }}
            style={({ isActive }) => navLinkStyle(isActive)}
          >
            {t('nav.prs')}
          </NavLink>
        </nav>

        <select value={repo} onChange={e => patch('repo', e.target.value)} style={selectStyle}>
          <option value="">{t('app.allRepos')}</option>
          {repos.map(r => <option key={r} value={r}>{r}</option>)}
        </select>

        <select value={gran} onChange={e => patch('gran', e.target.value)} style={selectStyle}>
          <option value="week">{t('app.granularity.week')}</option>
          <option value="month">{t('app.granularity.month')}</option>
          <option value="quarter">{t('app.granularity.quarter')}</option>
          <option value="year">{t('app.granularity.year')}</option>
        </select>

        <select value={login} onChange={e => patch('login', e.target.value)} style={selectStyle}>
          <option value="">{t('filter.allUsers')}</option>
          {contributorList.map(l => <option key={l} value={l}>{l}</option>)}
        </select>

        <button
          onClick={() => syncRepo(repo || undefined)}
          disabled={syncing}
          style={{
            padding: '5px 14px',
            borderRadius: 6,
            border: 'none',
            background: syncing ? '#9ca3af' : '#6366f1',
            color: '#fff',
            fontSize: 13,
            cursor: syncing ? 'not-allowed' : 'pointer',
          }}
        >
          {syncing ? t('app.syncing') : t('app.refresh')}
        </button>

        {lastSyncAt != null && (
          <span style={{ fontSize: 12, color: '#9ca3af' }}>
            {t('app.lastSync')} {lastSyncStr}
          </span>
        )}

        <select
          value={lang}
          onChange={e => setLang(e.target.value as Lang)}
          style={{ ...selectStyle, marginLeft: 'auto' }}
        >
          <option value="en">{t('lang.en')}</option>
          <option value="zh">{t('lang.zh')}</option>
        </select>
      </header>

      <main>
        <Outlet />
      </main>
    </div>
  )
}

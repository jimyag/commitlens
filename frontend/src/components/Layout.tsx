import { useMemo, useState, useRef, useEffect } from 'react'
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
  cursor: 'pointer',
}

export function Layout() {
  const { repos: allRepoNames, allStats, allContributors, syncing, lastSyncAt, syncRepo } = useApp()
  const { t, tf, lang, setLang } = useI18n()
  const [searchParams, setSearchParams] = useSearchParams()

  const selectedRepos = useMemo(() => {
    const val = searchParams.get('repo')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  const gran = (searchParams.get('gran') ?? 'week') as Granularity

  const selectedLogins = useMemo(() => {
    const val = searchParams.get('login')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  // Dropdown States
  const [showRepoDropdown, setShowRepoDropdown] = useState(false)
  const [repoSearch, setRepoSearch] = useState('')
  const [showLoginDropdown, setShowLoginDropdown] = useState(false)
  const [loginSearch, setLoginSearch] = useState('')
  
  const repoDropdownRef = useRef<HTMLDivElement>(null)
  const loginDropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (repoDropdownRef.current && !repoDropdownRef.current.contains(event.target as Node)) {
        setShowRepoDropdown(false)
      }
      if (loginDropdownRef.current && !loginDropdownRef.current.contains(event.target as Node)) {
        setShowLoginDropdown(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const toggleItem = (key: string, currentValues: string[], value: string) => {
    let nextValues: string[]
    if (value === '') {
      nextValues = [] // Clear all
    } else if (currentValues.includes(value)) {
      nextValues = currentValues.filter(v => v !== value)
    } else {
      nextValues = [...currentValues, value]
    }

    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      if (nextValues.length > 0) {
        next.set(key, nextValues.join(','))
      } else {
        next.delete(key)
      }
      return next
    })
  }

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

  // 1. 过滤可选的仓库列表
  const fullRepoList = useMemo(() => {
    if (selectedLogins.length === 0) return allRepoNames
    return allStats
      .filter(s => {
        const repoLogins = Object.keys(s.contributors)
        return selectedLogins.some(l => repoLogins.includes(l))
      })
      .map(s => s.repo)
  }, [allRepoNames, allStats, selectedLogins])

  const filteredRepoNames = useMemo(() => {
    const list = repoSearch 
      ? fullRepoList.filter(r => r.toLowerCase().includes(repoSearch.toLowerCase()))
      : fullRepoList
    return [...list].sort()
  }, [fullRepoList, repoSearch])

  // 2. 过滤可选的用户列表
  const fullContributorList = useMemo(() => {
    let source = allContributors
    if (selectedRepos.length > 0) {
      const merged: Record<string, any> = {}
      allStats
        .filter(s => selectedRepos.includes(s.repo))
        .forEach(s => {
          Object.keys(s.contributors).forEach(l => { merged[l] = true })
        })
      source = merged
    }
    return Object.keys(source)
      .sort((a, b) => (allContributors[b]?.commit_count ?? 0) - (allContributors[a]?.commit_count ?? 0))
  }, [allStats, allContributors, selectedRepos])

  // 3. 应用用户搜索过滤
  const filteredContributorList = useMemo(() => {
    const list = loginSearch
      ? fullContributorList.filter(l => l.toLowerCase().includes(loginSearch.toLowerCase()))
      : fullContributorList
    return list
  }, [fullContributorList, loginSearch])

  const repoLabel = useMemo(() => {
    if (selectedRepos.length === 0) return t('app.allRepos')
    if (selectedRepos.length === 1) return selectedRepos[0]
    return tf('app.scope.multifmt', { n: selectedRepos.length })
  }, [selectedRepos, t, tf])

  const loginLabel = useMemo(() => {
    if (selectedLogins.length === 0) return t('filter.allUsers')
    if (selectedLogins.length === 1) return selectedLogins[0]
    return tf('filter.multiUsers', { n: selectedLogins.length })
  }, [selectedLogins, t, tf])

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
            to={{ pathname: '/lines', search: searchParams.toString() }}
            style={({ isActive }) => navLinkStyle(isActive)}
          >
            {t('app.section.linesTrend')}
          </NavLink>
          <NavLink
            to={{ pathname: '/prs', search: (() => { const p = new URLSearchParams(searchParams); p.delete('period'); return p.toString() })() }}
            style={({ isActive }) => navLinkStyle(isActive)}
          >
            {t('nav.prs')}
          </NavLink>
        </nav>

        {/* Searchable Repo Select (Multi) */}
        <div ref={repoDropdownRef} style={{ position: 'relative' }}>
          <div
            onClick={() => setShowRepoDropdown(!showRepoDropdown)}
            style={{ ...selectStyle, minWidth: 160, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
          >
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {repoLabel}
            </span>
            <span style={{ fontSize: 10, marginLeft: 8, opacity: 0.5 }}>▼</span>
          </div>

          {showRepoDropdown && (
            <div style={{
              position: 'absolute', top: '100%', left: 0, marginTop: 4,
              background: '#fff', border: '1px solid #d1d5db', borderRadius: 8,
              boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)', minWidth: 240, zIndex: 20,
              padding: 4,
            }}>
              <input
                autoFocus
                placeholder={t('app.allRepos') + '...'}
                value={repoSearch}
                onChange={e => setRepoSearch(e.target.value)}
                style={{
                  width: '100%', padding: '6px 8px', border: '1px solid #e5e7eb',
                  borderRadius: 6, fontSize: 13, marginBottom: 4, outline: 'none',
                  boxSizing: 'border-box'
                }}
              />
              <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                <div
                  onClick={() => { toggleItem('repo', selectedRepos, ''); setShowRepoDropdown(false); setRepoSearch('') }}
                  style={{
                    padding: '6px 12px', fontSize: 13, cursor: 'pointer',
                    background: selectedRepos.length === 0 ? '#f3f4f6' : 'transparent',
                    borderRadius: 4,
                  }}
                >
                  {t('app.allRepos')}
                </div>
                {filteredRepoNames.map(r => (
                  <div
                    key={r}
                    onClick={() => { toggleItem('repo', selectedRepos, r); setRepoSearch('') }}
                    style={{
                      padding: '6px 12px', fontSize: 13, cursor: 'pointer',
                      background: selectedRepos.includes(r) ? '#eef2ff' : 'transparent',
                      color: selectedRepos.includes(r) ? '#4f46e5' : 'inherit',
                      fontWeight: selectedRepos.includes(r) ? 600 : 400,
                      borderRadius: 4,
                      whiteSpace: 'nowrap',
                      display: 'flex', alignItems: 'center', gap: 8
                    }}
                  >
                    <input type="checkbox" checked={selectedRepos.includes(r)} readOnly style={{ pointerEvents: 'none' }} />
                    {r}
                  </div>
                ))}
                {filteredRepoNames.length === 0 && repoSearch && (
                  <div style={{ padding: '6px 12px', fontSize: 12, color: '#9ca3af' }}>No results</div>
                )}
              </div>
            </div>
          )}
        </div>

        <select value={gran} onChange={e => patch('gran', e.target.value)} style={selectStyle}>
          <option value="week">{t('app.granularity.week')}</option>
          <option value="month">{t('app.granularity.month')}</option>
          <option value="quarter">{t('app.granularity.quarter')}</option>
          <option value="year">{t('app.granularity.year')}</option>
        </select>

        {/* Searchable Contributor Select (Multi) */}
        <div ref={loginDropdownRef} style={{ position: 'relative' }}>
          <div
            onClick={() => setShowLoginDropdown(!showLoginDropdown)}
            style={{ ...selectStyle, minWidth: 140, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
          >
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {loginLabel}
            </span>
            <span style={{ fontSize: 10, marginLeft: 8, opacity: 0.5 }}>▼</span>
          </div>

          {showLoginDropdown && (
            <div style={{
              position: 'absolute', top: '100%', left: 0, marginTop: 4,
              background: '#fff', border: '1px solid #d1d5db', borderRadius: 8,
              boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)', minWidth: 200, zIndex: 20,
              padding: 4,
            }}>
              <input
                autoFocus
                placeholder={t('filter.allUsers') + '...'}
                value={loginSearch}
                onChange={e => setLoginSearch(e.target.value)}
                style={{
                  width: '100%', padding: '6px 8px', border: '1px solid #e5e7eb',
                  borderRadius: 6, fontSize: 13, marginBottom: 4, outline: 'none',
                  boxSizing: 'border-box'
                }}
              />
              <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                <div
                  onClick={() => { toggleItem('login', selectedLogins, ''); setShowLoginDropdown(false); setLoginSearch('') }}
                  style={{
                    padding: '6px 12px', fontSize: 13, cursor: 'pointer',
                    background: selectedLogins.length === 0 ? '#f3f4f6' : 'transparent',
                    borderRadius: 4,
                  }}
                >
                  {t('filter.allUsers')}
                </div>
                {filteredContributorList.map(l => (
                  <div
                    key={l}
                    onClick={() => { toggleItem('login', selectedLogins, l); setLoginSearch('') }}
                    style={{
                      padding: '6px 12px', fontSize: 13, cursor: 'pointer',
                      background: selectedLogins.includes(l) ? '#eef2ff' : 'transparent',
                      color: selectedLogins.includes(l) ? '#4f46e5' : 'inherit',
                      fontWeight: selectedLogins.includes(l) ? 600 : 400,
                      borderRadius: 4,
                      whiteSpace: 'nowrap',
                      display: 'flex', alignItems: 'center', gap: 8
                    }}
                  >
                    <input type="checkbox" checked={selectedLogins.includes(l)} readOnly style={{ pointerEvents: 'none' }} />
                    {l}
                  </div>
                ))}
                {filteredContributorList.length === 0 && loginSearch && (
                  <div style={{ padding: '6px 12px', fontSize: 12, color: '#9ca3af' }}>No results</div>
                )}
              </div>
            </div>
          )}
        </div>

        <button
          onClick={() => syncRepo(selectedRepos.length === 1 ? selectedRepos[0] : undefined)}
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

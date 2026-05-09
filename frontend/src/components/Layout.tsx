import { useMemo, useState, useRef, useEffect } from 'react'
import { Outlet, Link, NavLink, useSearchParams } from 'react-router-dom'
import { useApp } from '../context/AppContext'
import { useI18n, type Lang } from '../i18n/I18nContext'
import type { MessageKey } from '../i18n/bundles/en'
import type { ContributorStats } from '../api'
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
    transition: 'all 0.2s',
  }
}

export function Layout() {
  const { repos: allRepoIDs, allStats, allContributors, syncing, syncRepo } = useApp()
  const { lang, setLang, t, tf } = useI18n()
  const [searchParams, setSearchParams] = useSearchParams()

  const selectedRepos = useMemo(() => {
    const val = searchParams.get('repo')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  const selectedLogins = useMemo(() => {
    const val = searchParams.get('login')
    return val ? val.split(',').filter(Boolean) : []
  }, [searchParams])

  const gran = (searchParams.get('gran') ?? 'week') as Granularity

  const [repoOpen, setRepoOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const [repoSearch, setRepoSearch] = useState('')
  const [userSearch, setUserSearch] = useState('')

  const repoRef = useRef<HTMLDivElement>(null)
  const userRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const clickOutside = (e: MouseEvent) => {
      if (repoRef.current && !repoRef.current.contains(e.target as Node)) setRepoOpen(false)
      if (userRef.current && !userRef.current.contains(e.target as Node)) setUserOpen(false)
    }
    document.addEventListener('mousedown', clickOutside)
    return () => document.removeEventListener('mousedown', clickOutside)
  }, [])

  const toggleRepo = (id: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      const current = prev.get('repo')?.split(',').filter(Boolean) ?? []
      let nextList: string[]
      if (current.includes(id)) {
        nextList = current.filter(v => v !== id)
      } else {
        nextList = [...current, id]
      }
      if (nextList.length > 0) next.set('repo', nextList.join(','))
      else next.delete('repo')
      return next
    })
  }

  const toggleLogin = (login: string) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      const current = prev.get('login')?.split(',').filter(Boolean) ?? []
      let nextList: string[]
      if (current.includes(login)) {
        nextList = current.filter(v => v !== login)
      } else {
        nextList = [...current, login]
      }
      if (nextList.length > 0) next.set('login', nextList.join(','))
      else next.delete('login')
      return next
    })
  }

  const setGran = (g: Granularity) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.set('gran', g)
      return next
    })
  }

  const fullRepoList = useMemo(() => {
    const list = repoSearch
      ? allRepoIDs.filter(r => r.toLowerCase().includes(repoSearch.toLowerCase()))
      : allRepoIDs
    return [...list].sort()
  }, [allRepoIDs, repoSearch])

  // 2. 过滤可选的用户列表
  const fullContributorList = useMemo(() => {
    let source: Record<string, ContributorStats | boolean> = allContributors
    if (selectedRepos.length > 0) {
      const merged: Record<string, boolean> = {}
      allStats
        .filter(s => selectedRepos.includes(s.repo))
        .forEach(s => {
          Object.keys(s.contributors).forEach(l => { merged[l] = true })
        })
      source = merged
    }
    const list = Object.keys(source)
      .sort((a, b) => (allContributors[b]?.commit_count ?? 0) - (allContributors[a]?.commit_count ?? 0))
    
    if (userSearch) {
      return list.filter(l => l.toLowerCase().includes(userSearch.toLowerCase()))
    }
    return list
  }, [allStats, allContributors, selectedRepos, userSearch])

  return (
    <div style={{ minHeight: '100vh', background: '#f9fafb', color: '#111827', fontFamily: 'Inter, system-ui, sans-serif' }}>
      {/* 顶部导航 */}
      <header style={{
        height: 64, background: '#fff', borderBottom: '1px solid #e5e7eb',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        position: 'sticky', top: 0, zIndex: 50, padding: '0 5%',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
          <Link to="/" style={{ textDecoration: 'none', display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ fontSize: 20, fontWeight: 800, color: '#4f46e5', letterSpacing: '-0.025em' }}>CommitLens</span>
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

          {/* 仓库多选 */}
          <div ref={repoRef} style={{ position: 'relative' }}>
            <button
              onClick={() => setRepoOpen(!repoOpen)}
              style={dropdownBtn(selectedRepos.length > 0)}
            >
              {selectedRepos.length === 0 ? t('app.allRepos') : tf('app.scope.multifmt', { n: selectedRepos.length })}
              <ChevronDown />
            </button>
            {repoOpen && (
              <div style={dropdownListStyle}>
                <input
                  autoFocus
                  placeholder="Search repos..."
                  value={repoSearch}
                  onChange={e => setRepoSearch(e.target.value)}
                  style={searchInputStyle}
                />
                <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                  {fullRepoList.map(r => (
                    <label key={r} style={itemStyle(selectedRepos.includes(r))}>
                      <input
                        type="checkbox"
                        checked={selectedRepos.includes(r)}
                        onChange={() => toggleRepo(r)}
                        style={{ marginRight: 8 }}
                      />
                      {r}
                    </label>
                  ))}
                  {fullRepoList.length === 0 && <div style={{ padding: '10px 12px', color: '#9ca3af', fontSize: 13 }}>No results</div>}
                </div>
              </div>
            )}
          </div>

          {/* 粒度选择 */}
          <div style={{ display: 'flex', background: '#f3f4f6', padding: 3, borderRadius: 8, gap: 1 }}>
            {(['week', 'month', 'quarter', 'year'] as Granularity[]).map(g => (
              <button
                key={g}
                onClick={() => setGran(g)}
                style={granBtn(gran === g)}
              >
                {t(`app.granularity.${g}` as MessageKey)}
              </button>
            ))}
          </div>

          {/* 用户多选 */}
          <div ref={userRef} style={{ position: 'relative' }}>
            <button
              onClick={() => setUserOpen(!userOpen)}
              style={dropdownBtn(selectedLogins.length > 0)}
            >
              {selectedLogins.length === 0 ? t('filter.allUsers') : tf('filter.multiUsers', { n: selectedLogins.length })}
              <ChevronDown />
            </button>
            {userOpen && (
              <div style={dropdownListStyle}>
                <input
                  autoFocus
                  placeholder="Search contributors..."
                  value={userSearch}
                  onChange={e => setUserSearch(e.target.value)}
                  style={searchInputStyle}
                />
                <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                  {fullContributorList.map(l => (
                    <label key={l} style={itemStyle(selectedLogins.includes(l))}>
                      <input
                        type="checkbox"
                        checked={selectedLogins.includes(l)}
                        onChange={() => toggleLogin(l)}
                        style={{ marginRight: 8 }}
                      />
                      {l}
                      <span style={{ marginLeft: 'auto', fontSize: 11, color: '#9ca3af' }}>
                        {allContributors[l]?.commit_count ?? 0}
                      </span>
                    </label>
                  ))}
                  {fullContributorList.length === 0 && <div style={{ padding: '10px 12px', color: '#9ca3af', fontSize: 13 }}>No results</div>}
                </div>
              </div>
            )}
          </div>

          <button
            onClick={() => syncRepo()}
            disabled={syncing}
            style={{
              padding: '6px 14px', borderRadius: 8, background: '#4f46e5', color: '#fff',
              border: 'none', fontSize: 13, fontWeight: 600, cursor: syncing ? 'wait' : 'pointer',
              opacity: syncing ? 0.7 : 1,
            }}
          >
            {syncing ? t('app.syncing') : t('app.refresh')}
          </button>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <select
            value={lang}
            onChange={e => setLang(e.target.value as Lang)}
            style={{
              padding: '4px 8px', borderRadius: 6, border: '1px solid #d1d5db',
              fontSize: 13, background: '#fff', outline: 'none',
            }}
          >
            <option value="en">English</option>
            <option value="zh">中文</option>
          </select>
        </div>
      </header>

      {/* 主体内容 */}
      <main style={{ minHeight: 'calc(100vh - 64px)' }}>
        <Outlet />
      </main>
    </div>
  )
}

function dropdownBtn(active: boolean): React.CSSProperties {
  return {
    padding: '6px 12px', borderRadius: 8, border: '1px solid #d1d5db',
    background: '#fff', fontSize: 13, color: '#374151', cursor: 'pointer',
    display: 'flex', alignItems: 'center', gap: 8, minWidth: 120,
    fontWeight: active ? 600 : 400,
    borderColor: active ? '#4f46e5' : '#d1d5db',
  }
}

const dropdownListStyle: React.CSSProperties = {
  position: 'absolute', top: '110%', left: 0, width: 240,
  background: '#fff', border: '1px solid #e5e7eb', borderRadius: 8,
  boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)', zIndex: 100,
  padding: '4px 0',
}

const searchInputStyle: React.CSSProperties = {
  width: 'calc(100% - 24px)', margin: '8px 12px', padding: '6px 10px',
  borderRadius: 6, border: '1px solid #e5e7eb', fontSize: 13, outline: 'none',
}

function itemStyle(selected: boolean): React.CSSProperties {
  return {
    display: 'flex', alignItems: 'center', padding: '8px 12px',
    fontSize: 13, cursor: 'pointer', background: selected ? '#f5f3ff' : 'transparent',
    color: selected ? '#4f46e5' : '#374151',
    userSelect: 'none',
  }
}

function granBtn(active: boolean): React.CSSProperties {
  return {
    padding: '4px 12px', borderRadius: 6, border: 'none',
    background: active ? '#fff' : 'transparent',
    color: active ? '#111827' : '#6b7280',
    fontSize: 12, fontWeight: active ? 600 : 500, cursor: 'pointer',
    boxShadow: active ? '0 1px 2px rgba(0,0,0,0.05)' : 'none',
  }
}

function ChevronDown() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M6 9l6 6 6-6" />
    </svg>
  )
}

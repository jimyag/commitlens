import { useEffect, useMemo, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { useI18n } from '../i18n/I18nContext'
import type { MessageKey } from '../i18n/bundles/en'

const STORAGE_KEY = 'commitlens.savedViews.v1'

type SavedView = {
  id: string
  name: string
  pathname: string
  search: string
  createdAt: number
}

type Translate = (key: MessageKey) => string
type Format = (key: MessageKey, values: Record<string, string | number>) => string

export function SavedViewsPanel() {
  const location = useLocation()
  const navigate = useNavigate()
  const { t, tf } = useI18n()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [views, setViews] = useState<SavedView[]>(loadSavedViews)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const clickOutside = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', clickOutside)
    return () => document.removeEventListener('mousedown', clickOutside)
  }, [open])

  const summary = useMemo(
    () => describeSearch(location.search, tf, t),
    [location.search, t, tf],
  )

  const persist = (next: SavedView[]) => {
    setViews(next)
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
    } catch {
      // Keep the in-memory list usable when browser storage is unavailable.
    }
  }

  const saveCurrent = () => {
    const trimmed = name.trim()
    if (!trimmed) return
    const existing = views.find(view => view.name.toLowerCase() === trimmed.toLowerCase())
    const nextView: SavedView = {
      id: existing?.id ?? `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name: trimmed,
      pathname: location.pathname,
      search: location.search,
      createdAt: existing?.createdAt ?? Date.now(),
    }
    const next = existing
      ? views.map(view => view.id === existing.id ? nextView : view)
      : [nextView, ...views]
    persist(next)
    setName('')
  }

  const applyView = (view: SavedView) => {
    void navigate({ pathname: view.pathname, search: view.search })
    setOpen(false)
  }

  const deleteView = (id: string) => {
    persist(views.filter(view => view.id !== id))
  }

  return (
    <div ref={rootRef} style={{ position: 'relative' }}>
      <button onClick={() => setOpen(value => !value)} style={triggerStyle}>
        {t('savedViews.title')}
        <ChevronDown />
      </button>
      {open && (
        <div style={panelStyle}>
          <div style={{ padding: '12px 14px', borderBottom: '1px solid #e5e7eb' }}>
            <strong style={{ display: 'block', fontSize: 13, marginBottom: 4 }}>{t('savedViews.saved')}</strong>
            <div style={{ color: '#6b7280', fontSize: 11, lineHeight: 1.4 }}>
              {tf('savedViews.current', { summary })}
            </div>
            <div style={{ display: 'flex', gap: 6, marginTop: 9 }}>
              <input
                value={name}
                onChange={event => setName(event.target.value)}
                onKeyDown={event => {
                  if (event.key === 'Enter') saveCurrent()
                }}
                placeholder={t('savedViews.name')}
                style={inputStyle}
              />
              <button disabled={!name.trim()} onClick={saveCurrent} style={{ ...saveButton, opacity: name.trim() ? 1 : 0.45 }}>
                {t('savedViews.saveCurrent')}
              </button>
            </div>
          </div>
          <div style={{ maxHeight: 320, overflowY: 'auto' }}>
            {views.map(view => (
              <div key={view.id} style={viewRow}>
                <button onClick={() => applyView(view)} style={viewButton}>
                  <strong style={{ display: 'block', fontSize: 12, fontWeight: 600 }}>{view.name}</strong>
                  <span style={{ display: 'block', marginTop: 2, color: '#9ca3af', fontSize: 10 }}>
                    {describeSearch(view.search, tf, t)} · {routeLabel(view.pathname, t)}
                  </span>
                </button>
                <button
                  onClick={() => deleteView(view.id)}
                  title={t('savedViews.delete')}
                  aria-label={`${t('savedViews.delete')} ${view.name}`}
                  style={deleteButton}
                >
                  ×
                </button>
              </div>
            ))}
            {views.length === 0 && <div style={emptyStyle}>{t('savedViews.empty')}</div>}
          </div>
        </div>
      )}
    </div>
  )
}

function loadSavedViews(): SavedView[] {
  try {
    const parsed: unknown = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '[]')
    if (!Array.isArray(parsed)) return []
    return parsed.filter(isSavedView)
  } catch {
    return []
  }
}

function isSavedView(value: unknown): value is SavedView {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<SavedView>
  return typeof candidate.id === 'string'
    && typeof candidate.name === 'string'
    && typeof candidate.pathname === 'string'
    && typeof candidate.search === 'string'
    && typeof candidate.createdAt === 'number'
}

function describeSearch(search: string, tf: Format, t: Translate): string {
  const params = new URLSearchParams(search)
  const repos = params.get('repo')?.split(',').filter(Boolean).length ?? 0
  const users = params.get('login')?.split(',').filter(Boolean).length ?? 0
  const granularity = params.get('gran') ?? 'week'
  const granularityKey = `app.granularity.${granularity}` as MessageKey
  return [
    repos > 0 ? tf('savedViews.repos', { n: repos }) : t('savedViews.allRepos'),
    users > 0 ? tf('savedViews.users', { n: users }) : t('savedViews.allUsers'),
    t(granularityKey),
  ].join(' · ')
}

function routeLabel(pathname: string, t: Translate): string {
  switch (pathname) {
    case '/lines': return t('savedViews.route.lines')
    case '/contributors': return t('savedViews.route.contributors')
    case '/prs': return t('savedViews.route.history')
    case '/identities': return t('savedViews.route.identities')
    default: return t('savedViews.route.commits')
  }
}

const triggerStyle: React.CSSProperties = {
  padding: '5px 9px',
  borderRadius: 7,
  border: '1px solid #d1d5db',
  background: '#fff',
  color: '#374151',
  fontSize: 12,
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  gap: 5,
  whiteSpace: 'nowrap',
}
const panelStyle: React.CSSProperties = {
  position: 'absolute',
  top: 'calc(100% + 8px)',
  right: 0,
  width: 330,
  background: '#fff',
  border: '1px solid #e5e7eb',
  borderRadius: 9,
  boxShadow: '0 12px 28px rgba(15, 23, 42, 0.16)',
  zIndex: 120,
  overflow: 'hidden',
}
const inputStyle: React.CSSProperties = {
  flex: 1,
  minWidth: 0,
  padding: '6px 8px',
  border: '1px solid #d1d5db',
  borderRadius: 5,
  fontSize: 12,
}
const saveButton: React.CSSProperties = {
  border: 0,
  borderRadius: 5,
  padding: '6px 8px',
  background: '#4f46e5',
  color: '#fff',
  fontSize: 11,
  cursor: 'pointer',
}
const viewRow: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  borderTop: '1px solid #f3f4f6',
}
const viewButton: React.CSSProperties = {
  flex: 1,
  minWidth: 0,
  border: 0,
  background: 'transparent',
  textAlign: 'left',
  padding: '9px 14px',
  color: '#374151',
  cursor: 'pointer',
}
const deleteButton: React.CSSProperties = {
  border: 0,
  background: 'transparent',
  color: '#9ca3af',
  fontSize: 18,
  padding: '8px 12px',
  cursor: 'pointer',
}
const emptyStyle: React.CSSProperties = { padding: 24, textAlign: 'center', color: '#9ca3af', fontSize: 12 }

function ChevronDown() {
  return (
    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M6 9l6 6 6-6" />
    </svg>
  )
}

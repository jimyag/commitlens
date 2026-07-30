import { useEffect, useMemo, useRef, useState } from 'react'
import axios from 'axios'
import { api, type RepositorySyncStatus } from '../api'
import { useApp } from '../context/AppContext'
import { useI18n } from '../i18n/I18nContext'
import type { MessageKey } from '../i18n/bundles/en'

type Props = {
  open: boolean
  onToggle: () => void
  onClose: () => void
}

export function SyncStatusPanel({ open, onToggle, onClose }: Props) {
  const { syncing, syncRepositories } = useApp()
  const { t, tf } = useI18n()
  const [statuses, setStatuses] = useState<RepositorySyncStatus[]>([])
  const [error, setError] = useState('')
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open && !syncing) return
    let cancelled = false
    let timer: number | undefined
    const load = async () => {
      try {
        const response = await api.getSyncStatus()
        if (!cancelled) {
          setStatuses(response.data.statuses)
          setError('')
        }
      } catch (err) {
        if (!cancelled) setError(errorMessage(err))
      }
    }
    void load()
    if (syncing) timer = window.setInterval(() => { void load() }, 1000)
    return () => {
      cancelled = true
      if (timer !== undefined) window.clearInterval(timer)
    }
  }, [open, syncing])

  useEffect(() => {
    if (!open) return
    const clickOutside = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) onClose()
    }
    document.addEventListener('mousedown', clickOutside)
    return () => document.removeEventListener('mousedown', clickOutside)
  }, [onClose, open])

  const sorted = useMemo(() => {
    const order: Record<RepositorySyncStatus['state'], number> = {
      failed: 0,
      syncing: 1,
      success: 2,
      not_synced: 3,
    }
    return [...statuses].sort((a, b) => order[a.state] - order[b.state] || a.repo.localeCompare(b.repo))
  }, [statuses])
  const failedRepos = statuses.filter(status => status.state === 'failed').map(status => status.repo)
  const activeCount = statuses.filter(status => status.state === 'syncing' || status.state === 'failed').length

  const retryFailed = () => {
    if (failedRepos.length === 0) return
    void syncRepositories(failedRepos).catch(() => undefined)
  }

  return (
    <div ref={rootRef} style={{ position: 'relative' }}>
      <button onClick={onToggle} style={statusButton(activeCount > 0)}>
        <span style={{ ...statusDot, background: syncing ? '#f59e0b' : activeCount > 0 ? '#ef4444' : statuses.length > 0 ? '#22c55e' : '#9ca3af' }} />
        {t('syncStatus.title')}
      </button>
      {open && (
        <div style={panelStyle}>
          <div style={{ padding: '12px 14px 9px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #e5e7eb' }}>
            <strong style={{ fontSize: 13 }}>{t('syncStatus.title')}</strong>
            <button
              onClick={retryFailed}
              disabled={syncing || failedRepos.length === 0}
              title={failedRepos.length === 0 ? t('syncStatus.noFailures') : undefined}
              style={{ ...retryButton, opacity: syncing || failedRepos.length === 0 ? 0.45 : 1 }}
            >
              {t('syncStatus.retryFailed')}
            </button>
          </div>
          {error && <div style={{ padding: '9px 14px', color: '#b91c1c', background: '#fef2f2', fontSize: 12 }}>{error}</div>}
          {!error && statuses.length === 0 && <div style={emptyStyle}>{t('syncStatus.loading')}</div>}
          <div style={{ maxHeight: 420, overflowY: 'auto' }}>
            {sorted.map(status => (
              <div key={status.repo} style={{ padding: '10px 14px', borderTop: '1px solid #f3f4f6' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ ...statusDot, background: stateColor(status.state) }} />
                  <strong style={{ fontSize: 12, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis' }}>{status.repo}</strong>
                  <span style={{ marginLeft: 'auto', color: stateColor(status.state), fontSize: 11, fontWeight: 600 }}>
                    {t(`syncStatus.${status.state === 'not_synced' ? 'notSynced' : status.state}` as MessageKey)}
                  </span>
                </div>
                <div style={{ margin: '5px 0 0 14px', color: '#6b7280', fontSize: 11, lineHeight: 1.45 }}>
                  {status.state === 'failed' && status.error ? (
                    <div style={{ color: '#b91c1c', overflowWrap: 'anywhere' }}>{status.error}</div>
                  ) : status.state === 'syncing' && status.last_log ? (
                    <div>{status.last_log}</div>
                  ) : !status.has_data && status.state === 'success' ? (
                    <div>{t('syncStatus.emptyRepo')}</div>
                  ) : (
                    <div>{tf('syncStatus.commits', { n: status.commit_count })}</div>
                  )}
                  <div style={{ display: 'flex', gap: 10 }}>
                    {status.last_success && <span>{tf('syncStatus.lastSuccess', { time: formatTime(status.last_success) })}</span>}
                    {status.duration_ms > 0 && <span>{tf('syncStatus.duration', { duration: formatDuration(status.duration_ms) })}</span>}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function stateColor(state: RepositorySyncStatus['state']): string {
  switch (state) {
    case 'syncing': return '#f59e0b'
    case 'success': return '#22c55e'
    case 'failed': return '#ef4444'
    default: return '#9ca3af'
  }
}

function formatTime(value: string): string {
  return new Date(value).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
}

function formatDuration(milliseconds: number): string {
  if (milliseconds < 1000) return `${milliseconds} ms`
  if (milliseconds < 60_000) return `${(milliseconds / 1000).toFixed(1)} s`
  return `${Math.floor(milliseconds / 60_000)}m ${Math.round(milliseconds % 60_000 / 1000)}s`
}

function errorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    return String(error.response?.data?.error ?? error.message)
  }
  return error instanceof Error ? error.message : String(error)
}

function statusButton(active: boolean): React.CSSProperties {
  return {
    padding: '6px 10px',
    borderRadius: 8,
    border: `1px solid ${active ? '#fecaca' : '#d1d5db'}`,
    background: '#fff',
    color: '#374151',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    fontSize: 12,
    whiteSpace: 'nowrap',
  }
}

const statusDot: React.CSSProperties = {
  width: 7,
  height: 7,
  borderRadius: 999,
  flex: '0 0 auto',
}
const panelStyle: React.CSSProperties = {
  position: 'absolute',
  top: 'calc(100% + 8px)',
  right: 0,
  width: 390,
  background: '#fff',
  border: '1px solid #e5e7eb',
  borderRadius: 9,
  boxShadow: '0 12px 28px rgba(15, 23, 42, 0.16)',
  zIndex: 120,
  overflow: 'hidden',
}
const retryButton: React.CSSProperties = {
  border: 0,
  borderRadius: 5,
  padding: '4px 7px',
  background: '#eef2ff',
  color: '#4f46e5',
  fontSize: 11,
  cursor: 'pointer',
}
const emptyStyle: React.CSSProperties = { padding: 24, textAlign: 'center', color: '#9ca3af', fontSize: 12 }

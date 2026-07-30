import { useEffect, useMemo, useState } from 'react'
import axios from 'axios'
import { api, type ContributorIdentity, type IdentitySettings } from '../api'
import { useApp } from '../context/AppContext'
import { useI18n } from '../i18n/I18nContext'

type Editor = {
  original: string
  canonical: string
  aliases: string
}

export function IdentityPage() {
  const { loadData } = useApp()
  const { t, tf } = useI18n()
  const [settings, setSettings] = useState<IdentitySettings | null>(null)
  const [query, setQuery] = useState('')
  const [editor, setEditor] = useState<Editor | null>(null)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    void api.getIdentities()
      .then(response => {
        if (!cancelled) setSettings(response.data)
      })
      .catch(err => {
        if (!cancelled) setError(errorMessage(err))
      })
    return () => { cancelled = true }
  }, [])

  const identities = useMemo(() => {
    if (!settings) return []
    const needle = query.trim().toLowerCase()
    if (!needle) return settings.identities
    return settings.identities.filter(identity =>
      [identity.canonical, ...identity.aliases, ...identity.emails]
        .some(value => value.toLowerCase().includes(needle)),
    )
  }, [query, settings])

  const unmappedCount = settings?.identities.filter(identity => identity.aliases.length === 0).length ?? 0
  const hiddenCount = settings?.identities.filter(identity => identity.hidden).length ?? 0

  const update = async (next: Pick<IdentitySettings, 'user_map' | 'ignored_users'>) => {
    setBusy(true)
    setError('')
    setMessage('')
    try {
      const response = await api.updateIdentities(next)
      setSettings(response.data)
      await loadData()
      setMessage(t('identity.saved'))
    } catch (err) {
      setError(errorMessage(err))
      throw err
    } finally {
      setBusy(false)
    }
  }

  const saveEditor = async () => {
    if (!settings || !editor) return
    const canonical = editor.canonical.trim()
    if (!canonical) return

    const enteredAliases = splitAliases(editor.aliases)
    const identity = settings.identities.find(item => item.canonical === editor.original)
    const mergeNames = new Set(
      [editor.original, canonical, ...(identity?.aliases ?? []), ...enteredAliases]
        .map(value => value.toLowerCase()),
    )
    const nextMap: Record<string, string[]> = {}
    const mergedAliases = new Map<string, string>()

    for (const [existingCanonical, aliases] of Object.entries(settings.user_map)) {
      const names = [existingCanonical, ...aliases]
      if (names.some(name => mergeNames.has(name.toLowerCase()))) {
        for (const name of names) mergedAliases.set(name.toLowerCase(), name)
        continue
      }
      nextMap[existingCanonical] = aliases
    }
    for (const name of [editor.original, ...(identity?.aliases ?? []), ...enteredAliases]) {
      mergedAliases.set(name.toLowerCase(), name)
    }
    mergedAliases.delete(canonical.toLowerCase())
    nextMap[canonical] = [...mergedAliases.values()].sort((a, b) => a.localeCompare(b))

    const ignored = new Set(settings.ignored_users.map(value => value.toLowerCase()))
    if (ignored.delete(editor.original.toLowerCase())) ignored.add(canonical.toLowerCase())
    const ignoredUsers = [...ignored].map(key => key === canonical.toLowerCase() ? canonical : (
      settings.ignored_users.find(value => value.toLowerCase() === key) ?? key
    ))

    try {
      await update({ user_map: nextMap, ignored_users: ignoredUsers })
      setEditor(null)
    } catch {
      // The error is rendered above the table.
    }
  }

  const toggleHidden = async (identity: ContributorIdentity) => {
    if (!settings) return
    const ignored = settings.ignored_users.filter(name => name.toLowerCase() !== identity.canonical.toLowerCase())
    const nextIgnored = identity.hidden
      ? ignored
      : [...ignored, identity.canonical]
    try {
      await update({ user_map: settings.user_map, ignored_users: nextIgnored })
    } catch {
      // The error is rendered above the table.
    }
  }

  return (
    <section style={{ padding: '24px 5% 48px' }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 24, marginBottom: 18 }}>
        <div>
          <h1 style={{ fontSize: 22, margin: '0 0 6px', fontWeight: 650 }}>{t('identity.title')}</h1>
          <p style={{ color: '#6b7280', fontSize: 13 }}>{t('identity.description')}</p>
        </div>
        <div style={{ display: 'flex', gap: 8, color: '#6b7280', fontSize: 12 }}>
          <span style={summaryPill}>{tf('identity.unmapped', { n: unmappedCount })}</span>
          <span style={summaryPill}>{tf('identity.hiddenCount', { n: hiddenCount })}</span>
        </div>
      </div>

      <input
        value={query}
        onChange={event => setQuery(event.target.value)}
        placeholder={t('identity.search')}
        style={{ width: 320, maxWidth: '100%', padding: '8px 11px', border: '1px solid #d1d5db', borderRadius: 7, fontSize: 13, marginBottom: 12 }}
      />

      {message && <div style={{ ...noticeStyle, color: '#166534', background: '#f0fdf4', borderColor: '#bbf7d0' }}>{message}</div>}
      {error && <div style={{ ...noticeStyle, color: '#991b1b', background: '#fef2f2', borderColor: '#fecaca' }}>{error}</div>}

      <div style={{ background: '#fff', border: '1px solid #e5e7eb', borderRadius: 9, overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            <tr style={{ background: '#f9fafb', color: '#4b5563', textAlign: 'left' }}>
              <th style={headerCell}>{t('identity.name')}</th>
              <th style={headerCell}>{t('identity.aliases')}</th>
              <th style={headerCell}>{t('identity.emails')}</th>
              <th style={{ ...headerCell, textAlign: 'right' }}>{t('identity.commits')}</th>
              <th style={headerCell}>{t('identity.status')}</th>
              <th style={{ ...headerCell, textAlign: 'right' }}>{t('identity.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {identities.map(identity => {
              const editing = editor?.original === identity.canonical
              return (
                <tr key={identity.canonical} style={{ borderTop: '1px solid #e5e7eb', opacity: identity.hidden ? 0.58 : 1 }}>
                  <td style={bodyCell}>
                    {editing ? (
                      <input
                        value={editor.canonical}
                        onChange={event => setEditor({ ...editor, canonical: event.target.value })}
                        style={editorInput}
                      />
                    ) : <strong style={{ fontWeight: 600 }}>{identity.canonical}</strong>}
                  </td>
                  <td style={bodyCell}>
                    {editing ? (
                      <input
                        value={editor.aliases}
                        onChange={event => setEditor({ ...editor, aliases: event.target.value })}
                        placeholder={t('identity.aliasHint')}
                        style={{ ...editorInput, width: '100%' }}
                      />
                    ) : identity.aliases.length > 0 ? identity.aliases.join(', ') : <span style={{ color: '#9ca3af' }}>—</span>}
                  </td>
                  <td style={{ ...bodyCell, color: '#6b7280' }}>
                    {identity.emails.length > 0 ? identity.emails.join(', ') : '—'}
                  </td>
                  <td style={{ ...bodyCell, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>{identity.commit_count}</td>
                  <td style={bodyCell}>
                    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                      {identity.hidden && <span style={hiddenBadge}>{t('identity.hidden')}</span>}
                      {identity.likely_bot && <span style={botBadge}>{t('identity.bot')}</span>}
                    </div>
                  </td>
                  <td style={{ ...bodyCell, textAlign: 'right', whiteSpace: 'nowrap' }}>
                    {editing ? (
                      <>
                        <button disabled={busy} onClick={() => void saveEditor()} style={primaryButton}>{t('identity.save')}</button>
                        <button disabled={busy} onClick={() => setEditor(null)} style={plainButton}>{t('identity.cancel')}</button>
                      </>
                    ) : (
                      <>
                        <button
                          disabled={busy}
                          onClick={() => setEditor({ original: identity.canonical, canonical: identity.canonical, aliases: identity.aliases.join(', ') })}
                          style={plainButton}
                        >
                          {t('identity.edit')}
                        </button>
                        <button disabled={busy} onClick={() => void toggleHidden(identity)} style={plainButton}>
                          {identity.hidden ? t('identity.show') : t('identity.hide')}
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              )
            })}
            {settings && identities.length === 0 && (
              <tr><td colSpan={6} style={{ padding: 40, color: '#9ca3af', textAlign: 'center' }}>{t('identity.empty')}</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function splitAliases(value: string): string[] {
  const seen = new Set<string>()
  return value
    .split(/[,，\n]/)
    .map(item => item.trim())
    .filter(item => {
      const key = item.toLowerCase()
      if (!item || seen.has(key)) return false
      seen.add(key)
      return true
    })
}

function errorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    return String(error.response?.data?.error ?? error.message)
  }
  return error instanceof Error ? error.message : String(error)
}

const summaryPill: React.CSSProperties = {
  padding: '3px 8px',
  border: '1px solid #e5e7eb',
  borderRadius: 999,
  background: '#fff',
}
const noticeStyle: React.CSSProperties = {
  border: '1px solid',
  borderRadius: 7,
  padding: '8px 11px',
  fontSize: 13,
  marginBottom: 12,
}
const headerCell: React.CSSProperties = { padding: '10px 12px', fontWeight: 600 }
const bodyCell: React.CSSProperties = { padding: '10px 12px', verticalAlign: 'top' }
const editorInput: React.CSSProperties = {
  minWidth: 150,
  padding: '5px 7px',
  border: '1px solid #c7d2fe',
  borderRadius: 5,
  fontSize: 13,
  boxSizing: 'border-box',
}
const plainButton: React.CSSProperties = {
  border: 0,
  background: 'transparent',
  color: '#4f46e5',
  cursor: 'pointer',
  padding: '4px 7px',
  fontSize: 12,
}
const primaryButton: React.CSSProperties = {
  ...plainButton,
  background: '#4f46e5',
  color: '#fff',
  borderRadius: 5,
}
const hiddenBadge: React.CSSProperties = { color: '#991b1b', background: '#fee2e2', borderRadius: 999, padding: '2px 7px', fontSize: 11 }
const botBadge: React.CSSProperties = { color: '#92400e', background: '#fef3c7', borderRadius: 999, padding: '2px 7px', fontSize: 11 }

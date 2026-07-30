import type { ContributorStats } from '../api'
import { useI18n } from '../i18n/I18nContext'

interface Props {
  contributors: Record<string, ContributorStats>
  commitFrequency: Record<string, number[]>
}

const MIN_SPARKLINE_POINTS = 3
const SPARKLINE_WIDTH = 240
const SPARKLINE_HEIGHT = 28
const SPARKLINE_PADDING = 3

function CommitFrequencySparkline({ values }: { values: number[] }) {
  if (values.length < MIN_SPARKLINE_POINTS) return null

  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min
  const points = values.map((value, index) => {
    const x = values.length === 1 ? SPARKLINE_WIDTH / 2 : index * SPARKLINE_WIDTH / (values.length - 1)
    const y = range === 0
      ? SPARKLINE_HEIGHT / 2
      : SPARKLINE_PADDING + (max - value) / range * (SPARKLINE_HEIGHT - SPARKLINE_PADDING * 2)
    return `${x.toFixed(2)},${y.toFixed(2)}`
  }).join(' ')

  return (
    <svg
      aria-hidden
      viewBox={`0 0 ${SPARKLINE_WIDTH} ${SPARKLINE_HEIGHT}`}
      preserveAspectRatio="none"
      style={{ display: 'block', width: '100%', height: SPARKLINE_HEIGHT }}
    >
      <polyline
        points={points}
        fill="none"
        stroke="#6366f1"
        strokeOpacity={0.75}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  )
}

export function ContributorTable({ contributors, commitFrequency }: Props) {
  const { t } = useI18n()
  const sorted = Object.values(contributors).sort((a, b) => b.commit_count - a.commit_count)

  if (sorted.length === 0) {
    return <p style={{ color: '#888' }}>{t('table.empty')}</p>
  }

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
      <thead>
        <tr style={{ borderBottom: '2px solid #e5e7eb', background: '#f9fafb' }}>
          <th style={{ textAlign: 'left', padding: '10px 12px', verticalAlign: 'middle' }}>
            {t('table.contributor')}
          </th>
          <th style={{ width: '30%', textAlign: 'left', padding: '10px 12px', verticalAlign: 'middle' }}>
            {t('table.frequency')}
          </th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>{t('table.commits')}</th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>{t('table.added')}</th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>{t('table.deleted')}</th>
        </tr>
      </thead>
      <tbody>
        {sorted.map((c, idx) => (
          <tr
            key={c.login}
            style={{
              borderBottom: '1px solid #f0f0f0',
              background: idx % 2 === 0 ? '#fff' : '#fafafa',
            }}
          >
            <td style={{ padding: '10px 12px', verticalAlign: 'middle', textAlign: 'left' }}>
              <div
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'flex-start',
                  gap: 8,
                  minHeight: 32,
                }}
              >
                {c.avatar_url ? (
                  <img
                    src={c.avatar_url}
                    alt={c.login}
                    width={28}
                    height={28}
                    style={{ borderRadius: '50%', border: '1px solid #e5e7eb', flexShrink: 0 }}
                  />
                ) : (
                  <span style={{
                    width: 28, height: 28, borderRadius: '50%', background: '#e5e7eb',
                    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 12, color: '#6b7280', flexShrink: 0,
                  }}>
                    {c.login.slice(0, 1).toUpperCase()}
                  </span>
                )}
                <span style={{ fontWeight: 500 }}>{c.login}</span>
              </div>
            </td>
            <td style={{ width: '30%', padding: '10px 12px', verticalAlign: 'middle' }}>
              <CommitFrequencySparkline values={commitFrequency[c.login] ?? []} />
            </td>
            <td
              style={{
                textAlign: 'right',
                padding: '10px 12px',
                verticalAlign: 'middle',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {c.commit_count}
            </td>
            <td
              style={{
                textAlign: 'right',
                padding: '10px 12px',
                verticalAlign: 'middle',
                color: '#16a34a',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              +{c.additions.toLocaleString()}
            </td>
            <td
              style={{
                textAlign: 'right',
                padding: '10px 12px',
                verticalAlign: 'middle',
                color: '#dc2626',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              -{c.deletions.toLocaleString()}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

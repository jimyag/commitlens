import type { ContributorStats } from '../api'

interface Props {
  contributors: Record<string, ContributorStats>
}

export function ContributorTable({ contributors }: Props) {
  const sorted = Object.values(contributors).sort((a, b) => b.pr_count - a.pr_count)

  if (sorted.length === 0) {
    return <p style={{ color: '#888' }}>暂无数据</p>
  }

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
      <thead>
        <tr style={{ borderBottom: '2px solid #e5e7eb', background: '#f9fafb' }}>
          <th style={{ textAlign: 'left', padding: '10px 12px', verticalAlign: 'middle' }}>贡献者</th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>PR 数</th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>Commit 数</th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>新增行</th>
          <th style={{ textAlign: 'right', padding: '10px 12px', verticalAlign: 'middle' }}>删除行</th>
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
                <img
                  src={c.avatar_url}
                  alt={c.login}
                  width={28}
                  height={28}
                  style={{ borderRadius: '50%', border: '1px solid #e5e7eb', flexShrink: 0 }}
                />
                <span style={{ fontWeight: 500 }}>{c.login}</span>
              </div>
            </td>
            <td
              style={{
                textAlign: 'right',
                padding: '10px 12px',
                verticalAlign: 'middle',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {c.pr_count}
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

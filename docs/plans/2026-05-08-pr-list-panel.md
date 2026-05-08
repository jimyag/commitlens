# PR List Panel Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 点击趋势图中的柱子（总量柱或个人柱），在图表下方展开一个 PR 列表面板，显示该时间段内匹配的 PR 标题、编号、作者、合并日期和增删行数。

**Architecture:** 后端新增 `GET /api/prs` 接口，从 raw cache 中按时间区间和可选 login 过滤 PR；前端在 `TrendChart` 内监听 ECharts 点击事件，计算时间区间后请求该接口，在图表正下方渲染展开面板。Login 过滤口径与聚合时一致（主作者 + Co-authored-by 解析）。

**Tech Stack:** Go 1.21+, gin, React 18, TypeScript, ECharts (echarts-for-react)

---

### Task 1: 导出 `PRParticipants`，供 web handler 复用

**Files:**
- Modify: `internal/stats/coauthor.go`
- Modify: `internal/stats/aggregator.go`

**Step 1: 在 `coauthor.go` 中导出该函数（重命名）**

将第 17 行的 `uniquePRParticipants` → `PRParticipants`：

```go
// PRParticipants 返回该 PR 的参与者 GitHub 登录名：主作者 + Co-authored-by 合著者。
func PRParticipants(pr *gh.PR) []string {
    seen := make(map[string]struct{}, 8+len(pr.Commits))
    var out []string
    add := func(login string) {
        login = strings.TrimSpace(login)
        if login == "" {
            return
        }
        if _, ok := seen[login]; ok {
            return
        }
        seen[login] = struct{}{}
        out = append(out, login)
    }
    add(pr.Author)
    for _, login := range coauthorLoginsFromPR(pr) {
        add(login)
    }
    return out
}
```

**Step 2: 同步更新 `aggregator.go` 中的调用**

将第 19 行 `uniquePRParticipants(&pr)` 改为 `PRParticipants(&pr)`。

**Step 3: 确认编译通过**

```bash
cd /Users/jimyag/src/github/jimyag/commitlens
go build ./...
```

Expected: 无报错。

**Step 4: Commit**

```bash
git add internal/stats/coauthor.go internal/stats/aggregator.go
git commit -m "refactor(stats): export PRParticipants for reuse"
```

---

### Task 2: 后端 — 给 Server 加 rawCache，注册 `/api/prs`

**Files:**
- Modify: `internal/web/server.go`
- Modify: `internal/web/api.go`
- Modify: `cmd/root.go`

**Step 1: `server.go` — 给 `Server` struct 加 `rawCache` 字段和 `repos` 字段**

`Server` 已有 `repos []string`，只需加 `rawCache *cache.RawCache`：

```go
type Server struct {
    engine     *gin.Engine
    syncer     *isync.Syncer
    stats      []*cache.StatsData
    repos      []string
    rawCache   *cache.RawCache   // ← 新增
    frontendFS http.FileSystem
}
```

同时更新 `New` 函数签名：

```go
func New(assets embed.FS, syncer *isync.Syncer, stats []*cache.StatsData, repos []string, rawCache *cache.RawCache) *Server {
    gin.SetMode(gin.ReleaseMode)
    s := &Server{
        engine:   gin.New(),
        syncer:   syncer,
        stats:    stats,
        repos:    repos,
        rawCache: rawCache,
    }
    s.mountFrontend(assets)
    s.registerAPI()
    return s
}
```

**Step 2: `api.go` — 新增路由和 handler**

在 `registerAPI` 里加一行：

```go
api.GET("/prs", s.handleGetPRs)
```

新增 `handleGetPRs`：

```go
// PRInfo 是对外暴露的 PR 摘要（不含 commits 代码细节）。
type PRInfo struct {
    Repo      string    `json:"repo"`
    Number    int       `json:"number"`
    Title     string    `json:"title"`
    Author    string    `json:"author"`
    AvatarURL string    `json:"avatar_url"`
    MergedAt  time.Time `json:"merged_at"`
    Additions int       `json:"additions"`
    Deletions int       `json:"deletions"`
}

func (s *Server) handleGetPRs(c *gin.Context) {
    repo := c.Query("repo")   // 可选；空 = 所有仓库
    login := c.Query("login") // 可选；空 = 不过滤贡献者
    fromStr := c.Query("from")
    toStr := c.Query("to")

    from, err1 := time.Parse(time.RFC3339, fromStr)
    to, err2 := time.Parse(time.RFC3339, toStr)
    if err1 != nil || err2 != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from/to, expect RFC3339"})
        return
    }

    repos := s.repos
    if repo != "" {
        repos = []string{repo}
    }

    var result []PRInfo
    for _, r := range repos {
        raw, err := s.rawCache.Load(r)
        if err != nil {
            continue
        }
        for _, pr := range raw.PRs {
            if pr.MergedAt.Before(from) || !pr.MergedAt.Before(to) {
                continue
            }
            if login != "" {
                found := false
                for _, p := range stats.PRParticipants(&pr) {
                    if p == login {
                        found = true
                        break
                    }
                }
                if !found {
                    continue
                }
            }
            result = append(result, PRInfo{
                Repo:      r,
                Number:    pr.Number,
                Title:     pr.Title,
                Author:    pr.Author,
                AvatarURL: pr.AvatarURL,
                MergedAt:  pr.MergedAt,
                Additions: pr.Additions,
                Deletions: pr.Deletions,
            })
        }
    }
    // 按合并时间降序
    sort.Slice(result, func(i, j int) bool {
        return result[i].MergedAt.After(result[j].MergedAt)
    })
    c.JSON(http.StatusOK, gin.H{"prs": result, "total": len(result)})
}
```

需要在文件顶部 import：

```go
import (
    "context"
    "net/http"
    "sort"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jimyag/commitlens/internal/stats"
)
```

**Step 3: `cmd/root.go` — 传入 rawCache**

将第 97 行改为：

```go
srv := web.New(globalAssets, syncer, allStats, repos, rawCache)
```

**Step 4: 编译验证**

```bash
go build ./...
```

**Step 5: 手动测试接口**（需先运行服务）

```bash
# 启动（另一个终端）
go run main.go --web

# 按月查某人 PR
curl "http://localhost:8080/api/prs?repo=qbox/las&from=2026-04-01T00:00:00Z&to=2026-05-01T00:00:00Z&login=jimyag"
```

Expected: 返回 `{"prs":[...],"total":N}`，每条含 repo/number/title/author/avatar_url/merged_at/additions/deletions。

**Step 6: Commit**

```bash
git add internal/web/server.go internal/web/api.go cmd/root.go
git commit -m "feat(web): add /api/prs endpoint with time range and login filter"
```

---

### Task 3: 前端 — API 类型 + `getPRs` 函数

**Files:**
- Modify: `frontend/src/api.ts`

**Step 1: 加 `PRInfo` 接口和 `getPRs` 方法**

```typescript
export interface PRInfo {
  repo: string
  number: number
  title: string
  author: string
  avatar_url: string
  merged_at: string   // ISO 8601
  additions: number
  deletions: number
}

// 在 api 对象里加：
getPRs: (params: { repo?: string; from: string; to: string; login?: string }) =>
  axios.get<{ prs: PRInfo[]; total: number }>('/api/prs', { params }),
```

**Step 2: 编译检查（TypeScript 严格）**

```bash
cd /Users/jimyag/src/github/jimyag/commitlens/frontend
npm run build 2>&1 | tail -20
```

Expected: 无 TS 错误。

**Step 3: Commit**

```bash
git add frontend/src/api.ts
git commit -m "feat(web): add PRInfo type and getPRs API helper"
```

---

### Task 4: 前端 — i18n 新增字符串

**Files:**
- Modify: `frontend/src/i18n/bundles/en.ts`
- Modify: `frontend/src/i18n/bundles/zh.ts`

**Step 1: `en.ts` 加入以下键**

```typescript
'prPanel.title': 'PRs · {period}',
'prPanel.titleWithLogin': 'PRs · {period} · {login}',
'prPanel.close': 'Close',
'prPanel.loading': 'Loading…',
'prPanel.empty': 'No PRs found for this period.',
'prPanel.mergedAt': 'Merged',
'prPanel.colRepo': 'Repo',
'prPanel.colPR': 'PR',
'prPanel.colAuthor': 'Author',
'prPanel.colDate': 'Merged',
'prPanel.colLines': 'Lines',
```

**Step 2: `zh.ts` 加入对应翻译**

```typescript
'prPanel.title': 'PR 列表 · {period}',
'prPanel.titleWithLogin': 'PR 列表 · {period} · {login}',
'prPanel.close': '关闭',
'prPanel.loading': '加载中...',
'prPanel.empty': '该时间段内暂无 PR。',
'prPanel.mergedAt': '合并时间',
'prPanel.colRepo': '仓库',
'prPanel.colPR': 'PR',
'prPanel.colAuthor': '作者',
'prPanel.colDate': '合并时间',
'prPanel.colLines': '增删行',
```

**Step 3: Commit**

```bash
git add frontend/src/i18n/bundles/en.ts frontend/src/i18n/bundles/zh.ts
git commit -m "feat(i18n): add PR panel strings (en + zh)"
```

---

### Task 5: 前端 — `PRListPanel` 组件

**Files:**
- Create: `frontend/src/components/PRListPanel.tsx`

组件接受以下 props：

```typescript
interface Props {
  period: string           // e.g. "2026-04"
  login?: string           // 个人柱时有值
  prs: PRInfo[]
  loading: boolean
  onClose: () => void
  multiRepo: boolean       // 是否显示 repo 列
}
```

完整实现：

```tsx
import type { PRInfo } from '../api'
import { useI18n } from '../i18n/I18nContext'

interface Props {
  period: string
  login?: string
  prs: PRInfo[]
  loading: boolean
  onClose: () => void
  multiRepo: boolean
}

export function PRListPanel({ period, login, prs, loading, onClose, multiRepo }: Props) {
  const { t, tf } = useI18n()
  const title = login
    ? tf('prPanel.titleWithLogin', { period, login })
    : tf('prPanel.title', { period })

  return (
    <div
      style={{
        marginTop: 16,
        border: '1px solid #e5e7eb',
        borderRadius: 8,
        background: '#f9fafb',
        overflow: 'hidden',
      }}
    >
      {/* 头部 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '10px 16px',
          background: '#f3f4f6',
          borderBottom: '1px solid #e5e7eb',
        }}
      >
        <span style={{ fontWeight: 600, fontSize: 14, color: '#374151' }}>{title}</span>
        <button
          onClick={onClose}
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            fontSize: 16,
            color: '#6b7280',
            lineHeight: 1,
            padding: '0 4px',
          }}
          aria-label={t('prPanel.close')}
        >
          ×
        </button>
      </div>

      {/* 内容 */}
      <div style={{ padding: '8px 0', maxHeight: 360, overflowY: 'auto' }}>
        {loading ? (
          <div style={{ padding: '24px 16px', color: '#9ca3af', textAlign: 'center', fontSize: 13 }}>
            {t('prPanel.loading')}
          </div>
        ) : prs.length === 0 ? (
          <div style={{ padding: '24px 16px', color: '#9ca3af', textAlign: 'center', fontSize: 13 }}>
            {t('prPanel.empty')}
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr style={{ color: '#6b7280', borderBottom: '1px solid #e5e7eb' }}>
                {multiRepo && (
                  <th style={{ padding: '4px 16px', textAlign: 'left', fontWeight: 500 }}>
                    {t('prPanel.colRepo')}
                  </th>
                )}
                <th style={{ padding: '4px 16px', textAlign: 'left', fontWeight: 500 }}>
                  {t('prPanel.colPR')}
                </th>
                <th style={{ padding: '4px 8px', textAlign: 'left', fontWeight: 500 }}>
                  {t('prPanel.colAuthor')}
                </th>
                <th style={{ padding: '4px 8px', textAlign: 'left', fontWeight: 500 }}>
                  {t('prPanel.colDate')}
                </th>
                <th style={{ padding: '4px 16px', textAlign: 'right', fontWeight: 500 }}>
                  {t('prPanel.colLines')}
                </th>
              </tr>
            </thead>
            <tbody>
              {prs.map((pr, idx) => (
                <tr
                  key={`${pr.repo}-${pr.number}`}
                  style={{ borderBottom: idx < prs.length - 1 ? '1px solid #f3f4f6' : 'none' }}
                >
                  {multiRepo && (
                    <td style={{ padding: '6px 16px', color: '#6b7280', whiteSpace: 'nowrap' }}>
                      {pr.repo}
                    </td>
                  )}
                  <td style={{ padding: '6px 16px', minWidth: 220 }}>
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
                      <span style={{ color: '#9ca3af', whiteSpace: 'nowrap' }}>#{pr.number}</span>
                      <span style={{ color: '#111827', lineHeight: 1.4 }}>{pr.title}</span>
                    </div>
                  </td>
                  <td style={{ padding: '6px 8px', whiteSpace: 'nowrap' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      {pr.avatar_url ? (
                        <img
                          src={pr.avatar_url}
                          alt=""
                          width={20}
                          height={20}
                          style={{ borderRadius: '50%', flexShrink: 0 }}
                        />
                      ) : (
                        <span
                          style={{
                            width: 20,
                            height: 20,
                            borderRadius: '50%',
                            background: '#e5e7eb',
                            display: 'inline-flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: 10,
                            color: '#6b7280',
                            flexShrink: 0,
                          }}
                        >
                          {pr.author.slice(0, 1).toUpperCase()}
                        </span>
                      )}
                      <span style={{ color: '#374151' }}>{pr.author}</span>
                    </div>
                  </td>
                  <td style={{ padding: '6px 8px', color: '#6b7280', whiteSpace: 'nowrap' }}>
                    {new Date(pr.merged_at).toLocaleDateString()}
                  </td>
                  <td style={{ padding: '6px 16px', textAlign: 'right', whiteSpace: 'nowrap' }}>
                    <span style={{ color: '#16a34a' }}>+{pr.additions}</span>
                    {' '}
                    <span style={{ color: '#dc2626' }}>-{pr.deletions}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
```

**Step 1: 创建文件，写入上述内容**

**Step 2: 编译检查**

```bash
cd frontend && npm run build 2>&1 | tail -20
```

Expected: 无错误。

**Step 3: Commit**

```bash
git add frontend/src/components/PRListPanel.tsx
git commit -m "feat(web): add PRListPanel component"
```

---

### Task 6: 前端 — TrendChart 加 click handler + panel 状态 + period→date range

**Files:**
- Modify: `frontend/src/components/TrendChart.tsx`

**Step 1: 加 `periodToDateRange` 函数**（与 `toPeriodKey` 伴生，放在同一文件）

```typescript
/** 将 period key 转为 [from, to) 的 ISO 8601 UTC 字符串，供 /api/prs 使用 */
function periodToDateRange(period: string, gran: Granularity): { from: string; to: string } {
  const toISO = (d: Date) => d.toISOString()

  if (gran === 'week') {
    const m = period.match(/^(\d{4})-W(\d{2})$/)
    if (!m) return { from: period, to: period }
    const year = parseInt(m[1])
    const week = parseInt(m[2])
    const jan4 = new Date(year, 0, 4)
    const weekday = jan4.getDay() || 7
    const monday = new Date(jan4)
    monday.setDate(jan4.getDate() - weekday + 1)
    const from = new Date(monday)
    from.setDate(monday.getDate() + (week - 1) * 7)
    from.setHours(0, 0, 0, 0)
    const to = new Date(from)
    to.setDate(from.getDate() + 7)
    return { from: toISO(from), to: toISO(to) }
  }

  if (gran === 'month') {
    const m = period.match(/^(\d{4})-(\d{2})$/)
    if (!m) return { from: period, to: period }
    const year = parseInt(m[1])
    const month = parseInt(m[2]) - 1  // 0-indexed
    const from = new Date(Date.UTC(year, month, 1))
    const to = new Date(Date.UTC(year, month + 1, 1))
    return { from: toISO(from), to: toISO(to) }
  }

  if (gran === 'quarter') {
    const m = period.match(/^(\d{4})-Q(\d)$/)
    if (!m) return { from: period, to: period }
    const year = parseInt(m[1])
    const q = parseInt(m[2])
    const startMonth = (q - 1) * 3  // 0-indexed
    const from = new Date(Date.UTC(year, startMonth, 1))
    const to = new Date(Date.UTC(year, startMonth + 3, 1))
    return { from: toISO(from), to: toISO(to) }
  }

  // year
  const year = parseInt(period)
  if (isNaN(year)) return { from: period, to: period }
  const from = new Date(Date.UTC(year, 0, 1))
  const to = new Date(Date.UTC(year + 1, 0, 1))
  return { from: toISO(from), to: toISO(to) }
}
```

**Step 2: 给 `TrendChart` 加 props**

```typescript
interface Props {
  weekly: Record<string, WeeklyEntry>
  granularity: Granularity
  contributors: Record<string, ContributorStats>
  selectedRepo?: string   // ← 新增；空字符串/undefined = 全仓库
  repos?: string[]        // ← 新增；全仓库模式时需要
}
```

**Step 3: 加 panel 状态和 click handler**

在 `TrendChart` 函数体内加：

```typescript
const { t, tf } = useI18n()
// panel state
const [panel, setPanel] = useState<{
  period: string
  login: string | undefined
  prs: PRInfo[]
  loading: boolean
} | null>(null)

const handleBarClick = useCallback(
  async (params: { name?: string; seriesName?: string }) => {
    const period = params.name
    if (!period) return
    const isTotalSeries = params.seriesName === t('chart.totalSeries')
    const login = isTotalSeries ? undefined : (params.seriesName ?? undefined)
    const { from, to } = periodToDateRange(period, granularity)
    setPanel({ period, login, prs: [], loading: true })
    try {
      const repoParam = selectedRepo || undefined
      const res = await api.getPRs({ repo: repoParam, from, to, login })
      setPanel(prev => prev ? { ...prev, prs: res.data.prs ?? [], loading: false } : null)
    } catch {
      setPanel(prev => prev ? { ...prev, prs: [], loading: false } : null)
    }
  },
  [granularity, selectedRepo, t],
)
```

需要在文件顶部加 import：

```typescript
import { useState, useCallback } from 'react'
import { api } from '../api'
import type { PRInfo } from '../api'
import { PRListPanel } from './PRListPanel'
```

**Step 4: 给 `ReactECharts` 加 `onEvents`**

```tsx
<ReactECharts
  option={option as EChartsOption}
  style={{ width: '100%', height: heightPx, minWidth: 400 }}
  notMerge
  onEvents={{ click: handleBarClick }}
/>
```

**Step 5: 在 `ReactECharts` 后渲染面板**

```tsx
{panel && (
  <PRListPanel
    period={panel.period}
    login={panel.login}
    prs={panel.prs}
    loading={panel.loading}
    onClose={() => setPanel(null)}
    multiRepo={!selectedRepo && (repos?.length ?? 0) > 1}
  />
)}
```

**Step 6: 更新 App.tsx 传 selectedRepo 和 repos**

在 `App.tsx` 第 171 行的 `<TrendChart>` 调用处加两个 props：

```tsx
<TrendChart
  weekly={weekly}
  granularity={granularity}
  contributors={contributors}
  selectedRepo={selectedRepo}
  repos={repos}
/>
```

**Step 7: 编译验证**

```bash
cd frontend && npm run build 2>&1 | tail -20
```

Expected: 无 TS 错误。

**Step 8: 测试**

启动后端：`go run main.go --web`，
启动前端开发服务器：`cd frontend && npm run dev`，
用浏览器打开后点击不同柱子，验证：
- 总量柱 → 面板标题无 login，列出所有人 PR
- 个人柱 → 面板标题带 login，只列该人 PR
- 再次点击同一柱或点 × 可关闭面板
- 切换粒度后面板自动关闭（因 `panel` state 不跟随保留）

**Step 9: Commit**

```bash
git add frontend/src/components/TrendChart.tsx frontend/src/App.tsx
git commit -m "feat(web): click bar to show PR list panel"
```

---

### Task 7: 最终整体验证

**Step 1: 全量构建**

```bash
cd /Users/jimyag/src/github/jimyag/commitlens
go build ./...
cd frontend && npm run build
```

Expected: 无报错。

**Step 2: 运行现有测试**

```bash
cd /Users/jimyag/src/github/jimyag/commitlens
go test ./...
```

Expected: 全部通过（本次改动不涉及现有测试文件）。

**Step 3: 边缘情况手动验证**
- 切换"全部仓库" vs 单仓库：multiRepo 列的显示/隐藏
- 空数据段（该段无 PR）：显示空态文案
- 粒度切换后点击新柱子：显示新数据，旧面板消失

**Step 4: 最终 Commit（如上各步都已单独 commit，此步跳过）**

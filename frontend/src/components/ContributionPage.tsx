import axios from 'axios'
import { useEffect, useMemo, useRef, useState } from 'react'
import { api, type ContributionBreakdown, type ContributionReport, type ContributionWeek, type YearContributions } from '../api'
import { useApp } from '../context/AppContext'
import { useI18n } from '../i18n/I18nContext'
import './ContributionPage.css'

const SVG_WIDTH = 1180
const CALENDAR_LEFT = 70
const GRID_WIDTH = 742
const ROW_HEIGHT = 202
const CELL_SIZE = 11
const BREAKDOWN_X = 1000
const BREAKDOWN_RX = 108
const BREAKDOWN_RY = 34

export function ContributionPage() {
  const { repos } = useApp()
  const { t, tf } = useI18n()
  const currentYear = new Date().getUTCFullYear()
  const configuredOrganizations = useMemo(
    () => [...new Set(repos.filter(repo => repo.includes('/')).map(repo => repo.split('/')[0]))].sort(),
    [repos],
  )
  const [organizations, setOrganizations] = useState<string[]>([])
  const [selectedOrganization, setSelectedOrganization] = useState('')
  const [user, setUser] = useState('')
  const [fromYear, setFromYear] = useState(currentYear - 5)
  const [toYear, setToYear] = useState(currentYear)
  const [report, setReport] = useState<ContributionReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [loadingOrganizations, setLoadingOrganizations] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState('')
  const svgRef = useRef<SVGSVGElement>(null)

  const organization = selectedOrganization

  useEffect(() => {
    let active = true
    api.getGitHubContributionOrganizations({ user: '', fromYear: currentYear - 5, toYear: currentYear })
      .then(response => {
        if (!active) return
        setOrganizations(mergeOrganizations(configuredOrganizations, response.data.organizations.map(item => item.login)))
      })
      .catch(() => {
        if (active) setOrganizations(configuredOrganizations)
      })
    return () => { active = false }
  }, [configuredOrganizations, currentYear])

  const loadOrganizations = async () => {
    setLoadingOrganizations(true)
    setError('')
    try {
      const response = await api.getGitHubContributionOrganizations({ user: user.trim(), fromYear, toYear })
      const next = mergeOrganizations(configuredOrganizations, response.data.organizations.map(item => item.login))
      setOrganizations(next)
      if (organization && !next.includes(organization)) setSelectedOrganization('')
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setLoadingOrganizations(false)
    }
  }

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.getGitHubContributions({ organization, user: user.trim(), fromYear, toYear })
      setReport(response.data)
    } catch (requestError) {
      setReport(null)
      setError(errorMessage(requestError))
    } finally {
      setLoading(false)
    }
  }

  const downloadPNG = async () => {
    if (!svgRef.current || !report) return
    setDownloading(true)
    try {
      await saveSVGAsPNG(svgRef.current, contributionFilename(report))
    } catch (downloadError) {
      setError(errorMessage(downloadError))
    } finally {
      setDownloading(false)
    }
  }

  return (
    <section className="contribution-page">
      <div className="contribution-heading">
        <div>
          <h1>{t('contributions.title')}</h1>
          <p>{t('contributions.description')}</p>
        </div>
        <button className="contribution-download" disabled={!report || report.years.length === 0 || downloading} onClick={() => void downloadPNG()}>
          {downloading ? t('contributions.downloading') : t('contributions.download')}
        </button>
      </div>

      <div className="contribution-filters">
        <label>
          <span>{t('contributions.organization')}</span>
          <select value={organization} onChange={event => setSelectedOrganization(event.target.value)} disabled={loadingOrganizations}>
            <option value="">{t('contributions.allOrganizations')}</option>
            {mergeOrganizations(configuredOrganizations, organizations).map(org => <option key={org} value={org}>{org}</option>)}
          </select>
        </label>
        <label className="contribution-user">
          <span>{t('contributions.user')}</span>
          <input value={user} onChange={event => setUser(event.target.value)} placeholder={t('contributions.currentUser')} />
        </label>
        <label>
          <span>{t('contributions.fromYear')}</span>
          <input type="number" min="2008" max={toYear} value={fromYear} onChange={event => setFromYear(Number(event.target.value))} />
        </label>
        <label>
          <span>{t('contributions.toYear')}</span>
          <input type="number" min={fromYear} max={currentYear} value={toYear} onChange={event => setToYear(Number(event.target.value))} />
        </label>
        <button disabled={loading || fromYear > toYear} onClick={() => void load()}>
          {loading ? t('contributions.loading') : t('contributions.load')}
        </button>
        <button className="contribution-secondary" disabled={loadingOrganizations || fromYear > toYear} onClick={() => void loadOrganizations()}>
          {loadingOrganizations ? t('contributions.loadingOrganizations') : t('contributions.refreshOrganizations')}
        </button>
      </div>

      {error && <div className="contribution-error">{error}</div>}
      {!error && loading && !report && <div className="contribution-state">{t('contributions.loading')}</div>}
      {!loading && report && report.years.length === 0 && <div className="contribution-state">{t('contributions.empty')}</div>}
      {report && report.years.length > 0 && (
        <div className="contribution-canvas">
          <ContributionSVG ref={svgRef} report={report} labels={{
            total: value => tf('contributions.total', { n: value }),
            commits: value => tf('contributions.commits', { n: value }),
            commit: t('contributions.commit'),
            pulls: t('contributions.pullRequests'),
            issues: t('contributions.issues'),
            reviews: t('contributions.codeReviews'),
            less: t('contributions.less'),
            more: t('contributions.more'),
          }} scope={report.organization.login === 'all' ? t('contributions.allOrganizations') : report.organization.login} />
        </div>
      )}
    </section>
  )
}

function mergeOrganizations(...groups: string[][]) {
  return [...new Set(groups.flat().filter(Boolean))].sort((a, b) => a.localeCompare(b))
}

type Labels = {
  total: (value: number) => string
  commits: (value: number) => string
  commit: string
  pulls: string
  issues: string
  reviews: string
  less: string
  more: string
}

const ContributionSVG = ({ report, labels, scope, ref }: { report: ContributionReport; labels: Labels; scope: string; ref: React.Ref<SVGSVGElement> }) => {
  const height = 76 + report.years.length * ROW_HEIGHT + 28
  return (
    <svg ref={ref} xmlns="http://www.w3.org/2000/svg" viewBox={`0 0 ${SVG_WIDTH} ${height}`} role="img" aria-label={`${labels.total(report.total_contributions)} · ${scope}`}>
      <style>{`
        .svg-title { fill: #1f2328; font-size: 15px; font-weight: 650; }
        .svg-subtitle { fill: #636c76; font-size: 11px; }
        .svg-year { fill: #1f2328; font-size: 14px; font-weight: 650; }
        .svg-year-meta { fill: #636c76; font-size: 11px; }
        .svg-axis-label { fill: #636c76; font-size: 10px; }
        .svg-breakdown-label { fill: #3d444d; font-size: 10px; }
      `}</style>
      <rect width={SVG_WIDTH} height={height} rx="8" fill="#fff" />
      <text x="24" y="30" className="svg-title">{report.user} · {scope}</text>
      <text x="24" y="51" className="svg-subtitle">{labels.total(report.total_contributions)} · {report.from_year}–{report.to_year}</text>
      {report.years.map((year, index) => (
        <YearRow key={year.year} year={year} top={82 + index * ROW_HEIGHT} labels={labels} showDivider={index < report.years.length - 1} />
      ))}
    </svg>
  )
}

function YearRow({ year, top, labels, showDivider }: { year: YearContributions; top: number; labels: Labels; showDivider: boolean }) {
  const gridTop = top + 36
  return (
    <g>
      <text x="24" y={top} className="svg-year">{year.year}</text>
      <text x={CALENDAR_LEFT} y={top} className="svg-year-meta">{labels.commits(year.breakdown.commits)} · {labels.total(year.total_contributions)}</text>
      <Calendar weeks={year.weeks} top={gridTop} labels={labels} />
      <line x1="835" y1={top - 15} x2="835" y2={top + 112} stroke="#d8dee4" />
      <Breakdown breakdown={year.breakdown} centerY={gridTop + 43} labels={labels} />
      {showDivider && <line x1="24" y1={top + 165} x2={SVG_WIDTH - 24} y2={top + 165} stroke="#d8dee4" />}
    </g>
  )
}

function Calendar({ weeks, top, labels }: { weeks: ContributionWeek[]; top: number; labels: Labels }) {
  const months = calendarMonthLabels(weeks)
  return (
    <g>
      {[[1, 'Mon'], [3, 'Wed'], [5, 'Fri']].map(([row, label]) => (
        <text key={label} x={CALENDAR_LEFT - 8} y={top + Number(row) * 14 + 10} textAnchor="end" className="svg-axis-label">{label}</text>
      ))}
      {weeks.map((week, weekIndex) => {
        return (
          <g key={weekIndex}>
            {months.has(weekIndex) && <text x={CALENDAR_LEFT + weekIndex * 14} y={top - 10} className="svg-axis-label">{months.get(weekIndex)}</text>}
            {week.days.map(day => {
              const weekday = new Date(`${day.date}T00:00:00Z`).getUTCDay()
              return (
                <rect key={day.date} x={CALENDAR_LEFT + weekIndex * 14} y={top + weekday * 14} width={CELL_SIZE} height={CELL_SIZE} rx="2" fill={day.color || '#ebedf0'}>
                  <title>{day.date}: {labels.total(day.count)}</title>
                </rect>
              )
            })}
          </g>
        )
      })}
      <Legend x={CALENDAR_LEFT + GRID_WIDTH - 130} y={top + 106} labels={labels} />
    </g>
  )
}

function calendarMonthLabels(weeks: ContributionWeek[]) {
  const result = new Map<number, string>()
  let lastMonth = -1
  weeks.forEach((week, weekIndex) => {
    const monthDay = week.days.map(day => new Date(`${day.date}T00:00:00Z`)).find(date => date.getUTCDate() <= 7)
    const month = monthDay?.getUTCMonth() ?? -1
    if (!monthDay || month < 0 || month === lastMonth) return
    result.set(weekIndex, monthDay.toLocaleString('en', { month: 'short', timeZone: 'UTC' }))
    lastMonth = month
  })
  return result
}

function Legend({ x, y, labels }: { x: number; y: number; labels: Labels }) {
  const colors = ['#ebedf0', '#9be9a8', '#40c463', '#30a14e', '#216e39']
  return (
    <g>
      <text x={x} y={y + 10} className="svg-axis-label">{labels.less}</text>
      {colors.map((color, index) => <rect key={color} x={x + 28 + index * 14} y={y} width={CELL_SIZE} height={CELL_SIZE} rx="2" fill={color} />)}
      <text x={x + 103} y={y + 10} className="svg-axis-label">{labels.more}</text>
    </g>
  )
}

function Breakdown({ breakdown, centerY, labels }: { breakdown: ContributionBreakdown; centerY: number; labels: Labels }) {
  const values = [breakdown.code_reviews, breakdown.issues, breakdown.pull_requests, breakdown.commits]
  const percentages = exactPercentages(values)
  const [reviewPct, issuePct, pullPct, commitPct] = percentages
  const points = [
    `${BREAKDOWN_X},${centerY - BREAKDOWN_RY * reviewPct / 100}`,
    `${BREAKDOWN_X + BREAKDOWN_RX * issuePct / 100},${centerY}`,
    `${BREAKDOWN_X},${centerY + BREAKDOWN_RY * pullPct / 100}`,
    `${BREAKDOWN_X - BREAKDOWN_RX * commitPct / 100},${centerY}`,
  ].join(' ')
  const labelsData = [
    { x: BREAKDOWN_X, y: centerY - BREAKDOWN_RY - 16, anchor: 'middle' as const, percent: reviewPct, name: labels.reviews },
    { x: BREAKDOWN_X + BREAKDOWN_RX + 13, y: centerY - 4, anchor: 'start' as const, percent: issuePct, name: labels.issues },
    { x: BREAKDOWN_X, y: centerY + BREAKDOWN_RY + 17, anchor: 'middle' as const, percent: pullPct, name: labels.pulls },
    { x: BREAKDOWN_X - BREAKDOWN_RX - 13, y: centerY - 4, anchor: 'end' as const, percent: commitPct, name: labels.commit },
  ]
  return (
    <g>
      <line x1={BREAKDOWN_X - BREAKDOWN_RX} y1={centerY} x2={BREAKDOWN_X + BREAKDOWN_RX} y2={centerY} stroke="#1f883d" strokeWidth="1.5" />
      <line x1={BREAKDOWN_X} y1={centerY - BREAKDOWN_RY} x2={BREAKDOWN_X} y2={centerY + BREAKDOWN_RY} stroke="#1f883d" strokeWidth="1.5" />
      <polygon points={points} fill="#9be9a8" fillOpacity="0.78" />
      {labelsData.map(item => (
        <text key={item.name} x={item.x} y={item.y} textAnchor={item.anchor} className="svg-breakdown-label">
          <tspan x={item.x}>{item.percent}%</tspan>
          <tspan x={item.x} dy="12">{item.name}</tspan>
        </text>
      ))}
    </g>
  )
}

function exactPercentages(values: number[]) {
  const total = values.reduce((sum, value) => sum + value, 0)
  if (total === 0) return values.map(() => 0)
  const result = values.map(value => Math.floor(value * 100 / total))
  const remainders = values.map((value, index) => ({ index, value: value * 100 % total }))
    .sort((a, b) => b.value - a.value || a.index - b.index)
  const allocated = result.reduce((sum, value) => sum + value, 0)
  for (let index = 0; index < 100 - allocated; index += 1) result[remainders[index].index] += 1
  return result
}

function errorMessage(error: unknown) {
  if (axios.isAxiosError<{ error?: string }>(error)) return error.response?.data?.error || error.message
  return error instanceof Error ? error.message : String(error)
}

function contributionFilename(report: ContributionReport) {
  return `${report.organization.login}-${report.user}-${report.from_year}-${report.to_year}-contributions.png`
}

async function saveSVGAsPNG(svg: SVGSVGElement, filename: string) {
  const viewBox = svg.viewBox.baseVal
  const serialized = new XMLSerializer().serializeToString(svg)
  const source = new Blob([serialized], { type: 'image/svg+xml;charset=utf-8' })
  const sourceURL = URL.createObjectURL(source)
  try {
    const image = new Image()
    image.decoding = 'async'
    image.src = sourceURL
    await image.decode()
    const scale = pngExportScale(viewBox.width, viewBox.height)
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(viewBox.width * scale)
    canvas.height = Math.round(viewBox.height * scale)
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas is unavailable')
    context.fillStyle = '#ffffff'
    context.fillRect(0, 0, canvas.width, canvas.height)
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    const png = await new Promise<Blob>((resolve, reject) => canvas.toBlob(blob => blob ? resolve(blob) : reject(new Error('Could not render PNG')), 'image/png'))
    const downloadURL = URL.createObjectURL(png)
    const link = document.createElement('a')
    link.href = downloadURL
    link.download = filename
    link.click()
    URL.revokeObjectURL(downloadURL)
  } finally {
    URL.revokeObjectURL(sourceURL)
  }
}

function pngExportScale(width: number, height: number) {
  const preferredScale = 4
  const maxDimension = 16_384
  const maxPixels = 40_000_000
  return Math.max(1, Math.min(
    preferredScale,
    maxDimension / width,
    maxDimension / height,
    Math.sqrt(maxPixels / (width * height)),
  ))
}

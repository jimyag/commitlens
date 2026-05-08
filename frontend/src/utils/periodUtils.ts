export type Granularity = 'week' | 'month' | 'quarter' | 'year'

/** 将 ISO 周键 "YYYY-Www" 转为对应粒度的 period key */
export function toPeriodKey(weekKey: string, gran: Granularity): string {
  const m = weekKey.match(/^(\d{4})-W(\d{2})$/)
  if (!m) return weekKey
  const year = parseInt(m[1])
  const week = parseInt(m[2])
  if (gran === 'week') return weekKey
  if (gran === 'year') return `${year}`
  const jan4 = new Date(year, 0, 4)
  const weekday = jan4.getDay() || 7
  const monday = new Date(jan4)
  monday.setDate(jan4.getDate() - weekday + 1)
  const approx = new Date(monday)
  approx.setDate(monday.getDate() + (week - 1) * 7)
  const approxYear = approx.getFullYear()
  const mo = approx.getMonth() + 1
  if (gran === 'month') return `${approxYear}-${String(mo).padStart(2, '0')}`
  const quarter = Math.ceil(mo / 3)
  return `${approxYear}-Q${quarter}`
}

/** 将 period key 转为 [from, to) 的 ISO 8601 UTC 字符串，供 /api/prs 使用 */
export function periodToDateRange(period: string, gran: Granularity): { from: string; to: string } {
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
    const month = parseInt(m[2]) - 1
    const from = new Date(Date.UTC(year, month, 1))
    const to = new Date(Date.UTC(year, month + 1, 1))
    return { from: toISO(from), to: toISO(to) }
  }

  if (gran === 'quarter') {
    const m = period.match(/^(\d{4})-Q(\d)$/)
    if (!m) return { from: period, to: period }
    const year = parseInt(m[1])
    const q = parseInt(m[2])
    const startMonth = (q - 1) * 3
    const from = new Date(Date.UTC(year, startMonth, 1))
    const to = new Date(Date.UTC(year, startMonth + 3, 1))
    return { from: toISO(from), to: toISO(to) }
  }

  const year = parseInt(period)
  if (isNaN(year)) return { from: period, to: period }
  const from = new Date(Date.UTC(year, 0, 1))
  const to = new Date(Date.UTC(year + 1, 0, 1))
  return { from: toISO(from), to: toISO(to) }
}

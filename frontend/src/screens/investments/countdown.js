import { todayStr } from '../../data/schedule.js'

// daysLeft(dateStr) — number of days until dateStr, or null if invalid
export function daysLeft(dateStr) {
  if (!dateStr) return null
  return Math.ceil((new Date(dateStr) - new Date()) / 864e5)
}

// fmtCountdown(days) — human-readable countdown string
export function fmtCountdown(days) {
  if (days === null) return '—'
  if (days < 0) return 'Matured'
  if (days === 0) return 'Today'
  if (days < 30) return `${days}d left`
  if (days < 365) return `${Math.round(days / 30)}mo left`
  const y = Math.floor(days / 365)
  const m = Math.round((days % 365) / 30)
  return m > 0 ? `${y}y ${m}mo left` : `${y}y left`
}

// computeMatures(openedDateStr, tenureMonths) — returns maturity date string "YYYY-MM-DD"
export function computeMatures(openedStr, tenureMonths) {
  const d = new Date(openedStr || todayStr())
  d.setMonth(d.getMonth() + Math.round(tenureMonths))
  return d.toISOString().slice(0, 10)
}

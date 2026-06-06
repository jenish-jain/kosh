// Formatters and aggregation helpers — Indian numbering, ₹ INR

// ── Formatters ──────────────────────────────────────────────

function groupIndian(intStr) {
  if (intStr.length <= 3) return intStr
  const last3 = intStr.slice(-3)
  let other = intStr.slice(0, -3)
  other = other.replace(/\B(?=(\d{2})+(?!\d))/g, ',')
  return other + ',' + last3
}

export function fmtINR(n) {
  const neg = n < 0
  const v = Math.round(Math.abs(n))
  const s = '₹' + groupIndian(v.toString())
  return neg ? '−' + s : s
}

// ₹19.6L, ₹1.06Cr, ₹62K
export function fmtCompact(n) {
  const neg = n < 0
  const v = Math.abs(n)
  let out
  if (v >= 1e7) out = (v / 1e7).toFixed(2).replace(/\.?0+$/, '') + 'Cr'
  else if (v >= 1e5) out = (v / 1e5).toFixed(2).replace(/\.?0+$/, '') + 'L'
  else if (v >= 1e3) out = (v / 1e3).toFixed(1).replace(/\.?0+$/, '') + 'K'
  else out = Math.round(v).toString()
  return (neg ? '−₹' : '₹') + out
}

export function fmtPct(n, dp = 1) {
  return (n >= 0 ? '+' : '') + n.toFixed(dp) + '%'
}

export function fmtDate(d) {
  if (!d) return ''
  const dt = typeof d === 'string' ? new Date(d) : d
  const m = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']
  return dt.getDate() + ' ' + m[dt.getMonth()] + ' ' + dt.getFullYear()
}

// ── Aggregation ─────────────────────────────────────────────

export function holdingsFor(data, memberId) {
  const f = arr => memberId ? arr.filter(x => x.member === memberId) : arr
  return {
    mf:        f(data.mf        || []),
    stocks:    f(data.stocks    || []),
    metals:    f(data.metals    || []),
    fixed:     f(data.fixed     || []),
    insurance: f(data.insurance || []),
  }
}

export function classTotals(data, memberId) {
  const h = holdingsFor(data, memberId)

  const mfCur  = h.mf.reduce((a, x) => a + (x.current  || 0), 0)
  const mfInv  = h.mf.reduce((a, x) => a + (x.invested || 0), 0)

  const stCur  = h.stocks.reduce((a, x) => a + (x.qty || 0) * (x.last_price || 0), 0)
  const stInv  = h.stocks.reduce((a, x) => a + (x.qty || 0) * (x.avg_price  || 0), 0)

  const meCur  = h.metals.reduce((a, x) => a + (x.grams || 0) * (x.today_price || 0), 0)
  const meInv  = h.metals.reduce((a, x) => a + (x.grams || 0) * (x.buy_rate    || 0), 0)

  const fiCur  = h.fixed.reduce((a, x) => a + (x.current_value || 0), 0)
  const fiInv  = h.fixed.reduce((a, x) => a + (x.principal    || 0), 0)

  const inCur  = h.insurance.reduce((a, x) => a + (x.value || 0), 0)
  const inInv  = h.insurance.reduce((a, x) => a + (x.paid  || 0), 0)

  return {
    mf:        { cur: mfCur, inv: mfInv },
    stocks:    { cur: stCur, inv: stInv },
    metals:    { cur: meCur, inv: meInv },
    fixed:     { cur: fiCur, inv: fiInv },
    insurance: { cur: inCur, inv: inInv },
    total:     {
      cur: mfCur + stCur + meCur + fiCur + inCur,
      inv: mfInv + stInv + meInv + fiInv + inInv,
    },
  }
}

export function memberTotal(data, memberId) {
  return classTotals(data, memberId).total.cur
}

export const CLASS_META = {
  mf:        { label: 'Mutual Funds',       color: 'var(--accent)' },
  stocks:    { label: 'Stocks',             color: '#B5603E' },
  metals:    { label: 'Gold & Silver',      color: '#B0822C' },
  fixed:     { label: 'FD / RD',            color: '#9AA79E' },
  insurance: { label: 'Insurance & Plans',  color: '#5E7D72' },
}

export const ED_ORDER = ['mf', 'stocks', 'fixed', 'insurance', 'metals']
export const ED_COL = {
  mf: 'var(--accent)', stocks: '#B5603E', fixed: '#9AA79E', insurance: '#5E7D72', metals: '#B0822C'
}
export const ED_LABEL = {
  mf: 'Mutual funds', stocks: 'Stocks', fixed: 'FD / RD', insurance: 'Insurance', metals: 'Gold & silver'
}

// Today's date (static — refresh app to update)
export const TODAY = new Date()
export const TODAY_STR = TODAY.toISOString().slice(0, 10) // YYYY-MM-DD
export const TODAY_DISPLAY = fmtDate(TODAY)
export const TODAY_DAY = TODAY.getDate()
export const TODAY_MONTH = TODAY.getMonth() // 0-indexed
export const TODAY_YEAR = TODAY.getFullYear()

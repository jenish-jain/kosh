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
    loans:     f(data.loans     || []),
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

  const loanOutstanding = h.loans.reduce((a, x) => a + (x.outstanding || 0), 0)
  const loanEmiMonthly  = h.loans.reduce((a, x) => a + (x.emi || 0), 0)

  return {
    mf:        { cur: mfCur, inv: mfInv },
    stocks:    { cur: stCur, inv: stInv },
    metals:    { cur: meCur, inv: meInv },
    fixed:     { cur: fiCur, inv: fiInv },
    insurance: { cur: inCur, inv: inInv },
    liabilities: { cur: loanOutstanding, monthly: loanEmiMonthly },
    total:     {
      cur: mfCur + stCur + meCur + fiCur + inCur,
      inv: mfInv + stInv + meInv + fiInv + inInv,
    },
  }
}

// Gross asset total for a scope — used for allocation percentages, etc.
export function memberTotal(data, memberId) {
  return classTotals(data, memberId).total.cur
}

// True net worth — gross assets minus outstanding loan balances.
export function netWorth(data, memberId) {
  const c = classTotals(data, memberId)
  return c.total.cur - c.liabilities.cur
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

// Next date an insurance premium falls due, given its anchor `due_date` and
// `freq` (annual reuses the anchor's month+day each year, monthly reuses its
// day-of-month each month, single is a one-off that's null once it's past).
export function nextPremiumDue(policy, from = TODAY) {
  if (!policy?.due_date) return null
  const anchor = new Date(policy.due_date)
  if (isNaN(anchor)) return null

  if (policy.freq === 'single') {
    return anchor >= from ? anchor : null
  }
  if (policy.freq === 'monthly') {
    const day = anchor.getDate()
    let d = new Date(from.getFullYear(), from.getMonth(), day)
    if (d < from) d = new Date(from.getFullYear(), from.getMonth() + 1, day)
    return d
  }
  // annual
  const month = anchor.getMonth(), day = anchor.getDate()
  let d = new Date(from.getFullYear(), month, day)
  if (d < from) d = new Date(from.getFullYear() + 1, month, day)
  return d
}

// Next date a loan EMI falls due, given its `emi_day` (day-of-month it debits).
// Null once the loan's tenure has run out (started + tenure_months in the past).
export function nextEmiDue(loan, from = TODAY) {
  if (!loan?.emi_day) return null
  if (loan.started && loan.tenure_months) {
    const end = new Date(loan.started)
    end.setMonth(end.getMonth() + Math.round(loan.tenure_months))
    if (end < from) return null
  }
  let d = new Date(from.getFullYear(), from.getMonth(), loan.emi_day)
  if (d < from) d = new Date(from.getFullYear(), from.getMonth() + 1, loan.emi_day)
  return d
}

// Recurring fixed outflows (active SIP debits, insurance premiums, loan EMIs)
// due within the next `days` — sorted soonest-first, for an "upcoming" widget.
export function upcomingOutflows(data, memberId, days = 30) {
  const horizon = new Date(TODAY.getTime() + days * 864e5)
  const items = []

  for (const s of data.sips || []) {
    if (s.status !== 'active' || (memberId && s.member !== memberId)) continue
    let d = new Date(TODAY_YEAR, TODAY_MONTH, s.day)
    if (d < TODAY) d = new Date(TODAY_YEAR, TODAY_MONTH + 1, s.day)
    if (d <= horizon) items.push({ id: s.id, kind: 'sip', label: s.fund, amount: s.amount, date: d })
  }

  for (const p of data.insurance || []) {
    if (memberId && p.member !== memberId) continue
    const d = nextPremiumDue(p)
    if (d && d <= horizon) items.push({ id: p.id, kind: 'insurance', label: p.name, amount: p.premium, date: d })
  }

  for (const l of data.loans || []) {
    if (memberId && l.member !== memberId) continue
    const d = nextEmiDue(l)
    if (d && d <= horizon) items.push({ id: l.id, kind: 'loan', label: l.lender, amount: l.emi, date: d })
  }

  return items.sort((a, b) => a.date - b.date)
}

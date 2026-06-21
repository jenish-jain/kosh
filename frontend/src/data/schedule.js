// Schedule and date helpers — premium due dates, EMI dates, upcoming outflows
// TODAY coupling fix: no module-level date constants; use functions that call
// new Date() at call time, and accept an optional `from` parameter for testability.

// ── Today convenience functions ──────────────────────────────

export const todayStr = () => new Date().toISOString().slice(0, 10)

export const todayDisplay = () => {
  const d = new Date()
  return d.toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

export const todayDay = () => new Date().getDate()

// Next date an insurance premium falls due, given its anchor `due_date` and
// `freq` (annual reuses the anchor's month+day each year, monthly reuses its
// day-of-month each month, single is a one-off that's null once it's past).
export function nextPremiumDue(policy, from = new Date()) {
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
export function nextEmiDue(loan, from = new Date()) {
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
export function upcomingOutflows(data, memberId, days = 30, from = new Date()) {
  const horizon = new Date(from.getTime() + days * 864e5)
  const items = []

  for (const s of data.sips || []) {
    if (s.status !== 'active' || (memberId && s.member !== memberId)) continue
    let d = new Date(from.getFullYear(), from.getMonth(), s.day)
    if (d < from) d = new Date(from.getFullYear(), from.getMonth() + 1, s.day)
    if (d <= horizon) items.push({ id: s.id, kind: 'sip', label: s.fund, amount: s.amount, date: d })
  }

  for (const p of data.insurance || []) {
    if (memberId && p.member !== memberId) continue
    const d = nextPremiumDue(p, from)
    if (d && d <= horizon) items.push({ id: p.id, kind: 'insurance', label: p.name, amount: p.premium, date: d })
  }

  for (const l of data.loans || []) {
    if (memberId && l.member !== memberId) continue
    const d = nextEmiDue(l, from)
    if (d && d <= horizon) items.push({ id: l.id, kind: 'loan', label: l.lender, amount: l.emi, date: d })
  }

  return items.sort((a, b) => a.date - b.date)
}

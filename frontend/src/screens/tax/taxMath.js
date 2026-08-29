// Tax math — regime slabs, effective rate, and the Indian financial-year window.
// Mirrors the deterministic Go equivalents in backend/internal/models (fy.go,
// finance.go) used server-side for the AI recommendation facts — kept as a
// separate JS copy here so the Tax screen renders instantly with no round trip.

export function taxOldRegime(income) {
  if (income <= 250000) return 0
  let t = 0
  if (income > 1000000) t += (income - 1000000) * 0.30
  if (income > 500000)  t += (Math.min(income, 1000000) - 500000) * 0.20
  if (income > 250000)  t += (Math.min(income, 500000) - 250000) * 0.05
  let surcharge = 0
  if (income > 10000000) surcharge = t * 0.15
  else if (income > 5000000) surcharge = t * 0.10
  const cess = (t + surcharge) * 0.04
  return Math.round(t + surcharge + cess)
}

// New regime FY 2025-26: std deduction ₹75K, 87A rebate if taxable ≤ ₹12L
export const NEW_STD_DEDUCTION = 75000

export function taxNewRegime(gross) {
  const taxable = Math.max(0, gross - NEW_STD_DEDUCTION)
  let t = 0
  if (taxable > 2400000) t += (taxable - 2400000) * 0.30
  if (taxable > 2000000) t += (Math.min(taxable, 2400000) - 2000000) * 0.25
  if (taxable > 1600000) t += (Math.min(taxable, 2000000) - 1600000) * 0.20
  if (taxable > 1200000) t += (Math.min(taxable, 1600000) - 1200000) * 0.15
  if (taxable >  800000) t += (Math.min(taxable, 1200000) -  800000) * 0.10
  if (taxable >  400000) t += (Math.min(taxable,  800000) -  400000) * 0.05
  if (taxable <= 1200000) t = 0  // Section 87A rebate
  let surcharge = 0
  if (gross > 10000000) surcharge = t * 0.15
  else if (gross > 5000000) surcharge = t * 0.10
  const cess = (t + surcharge) * 0.04
  return Math.round(t + surcharge + cess)
}

export function slabLabelOld(income) {
  if (income <= 250000) return '0%'
  if (income <= 500000) return '5%'
  if (income <= 1000000) return '20%'
  if (income <= 5000000) return '30%'
  return '30% + surcharge'
}

export function slabLabelNew(gross) {
  const taxable = Math.max(0, gross - NEW_STD_DEDUCTION)
  if (taxable <= 400000)  return '0%'
  if (taxable <= 800000)  return '5%'
  if (taxable <= 1200000) return '10% · 0 via 87A'
  if (taxable <= 1600000) return '15%'
  if (taxable <= 2000000) return '20%'
  if (taxable <= 2400000) return '25%'
  return '30%'
}

export function effectiveRate(income, taxFn) {
  const t = taxFn(income)
  return income > 0 ? (t / income * 100).toFixed(1) : '0.0'
}

// ── Financial-year window (1 Apr – 31 Mar) ─────────────────────
export function currentFY(from = new Date()) {
  const startYear = from.getMonth() < 3 /* Jan-Mar = 0-2 */ ? from.getFullYear() - 1 : from.getFullYear()
  const start = new Date(startYear, 3, 1)
  const end = new Date(startYear + 1, 2, 31, 23, 59, 59)
  const label = `FY ${startYear}-${String((startYear + 1) % 100).padStart(2, '0')}`
  return { label, start, end }
}

function monthsBetween(from, to) {
  return (to.getFullYear() - from.getFullYear()) * 12 + (to.getMonth() - from.getMonth())
}

// ── FD/RD projected value at its own maturity date ─────────────
// RDs compound each monthly installment separately (mirrors the Go
// RDCurrentValue logic, projected to maturity instead of to now) — a plain
// FD-style compound on `principal` overstates/understates an RD's payout.
export function fdValueAtMaturity(f) {
  if (!f.opened || !f.matures) return f.principal || 0
  const opened = new Date(f.opened)
  const matures = new Date(f.matures)
  const r = (f.rate || 0) / 100

  if (f.kind === 'RD') {
    const tenure = monthsBetween(opened, matures)
    if (tenure <= 0 || !f.monthly) return 0
    let total = 0
    for (let i = 0; i < tenure; i++) {
      total += f.monthly * Math.pow(1 + r, (tenure - i) / 12)
    }
    return total
  }

  if (!f.principal) return 0
  const years = (matures - opened) / (1000 * 60 * 60 * 24 * 365.25)
  if (years <= 0) return f.principal
  return f.principal * Math.pow(1 + r, years)
}

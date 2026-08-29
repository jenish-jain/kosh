// Tax math — a generic slab/surcharge/cess interpreter driven by data
// (data.tax_rules), plus the Indian financial-year window and FD/RD
// maturity-value helpers.
//
// computeTax() mirrors backend/internal/models/tax_rules.go's ComputeTax —
// kept in sync manually (same pattern as the rest of this file: the frontend
// copy renders instantly with no round trip). Neither this file nor the Go
// side hardcodes slab boundaries or deduction caps any more — those live in
// TaxRuleSet rows (one per FY+regime) so a Budget-driven change is a data
// update reviewed and approved by the account owner, not a code change.

// ── Generic tax engine ──────────────────────────────────────────
export function computeTax(rules, grossIncome) {
  const taxable = Math.max(0, grossIncome - (rules.stdDeduction || 0))

  let tax = 0
  let prev = 0
  for (const band of rules.slabs || []) {
    const upto = band.upto ?? Infinity
    const upper = Math.min(taxable, upto)
    if (upper > prev) tax += (upper - prev) * band.rate
    if (upto === Infinity) break
    prev = upto
  }

  if (rules.rebateThreshold > 0 && taxable <= rules.rebateThreshold) {
    tax = Math.max(0, tax - (rules.rebateAmount || 0))
  }

  let surchargeRate = 0
  let maxAbove = -1
  for (const band of rules.surcharge || []) {
    if (grossIncome > band.above && band.above > maxAbove) {
      maxAbove = band.above
      surchargeRate = band.rate
    }
  }
  const surcharge = tax * surchargeRate

  const cess = (tax + surcharge) * (rules.cessRate || 0)
  return Math.round(tax + surcharge + cess)
}

export function slabLabel(rules, grossIncome) {
  const taxable = Math.max(0, grossIncome - (rules.stdDeduction || 0))
  let rate = 0
  let prev = 0
  for (const band of rules.slabs || []) {
    const upto = band.upto ?? Infinity
    if (taxable > prev) rate = band.rate
    if (taxable <= upto) break
    prev = upto
  }
  let label = `${Math.round(rate * 100)}%`
  const inRebate = rules.rebateThreshold > 0 && taxable <= rules.rebateThreshold
  if (inRebate && rate > 0) label += ' · 0 via 87A'
  const inSurcharge = (rules.surcharge || []).some(b => grossIncome > b.above)
  if (inSurcharge && !inRebate) label += ' + surcharge'
  return label
}

export function effectiveRate(income, taxFn) {
  const t = taxFn(income)
  return income > 0 ? (t / income * 100).toFixed(1) : '0.0'
}

// ── Bundled default rules (fallback only — mirrors Go's DefaultTaxRuleSet) ─
// Used if data.tax_rules is empty (e.g. before the backend's one-time
// migration seed has run) so the screen never breaks.
const FULL_REBATE = 999999999
const DEDUCTION_CAPS = { section80C: 150000, section80DSelf: 25000, section80DSenior: 50000, nps80CCD1B: 50000 }

export const DEFAULT_RULES = {
  old: {
    schemaVersion: 1,
    slabs: [{ upto: 250000, rate: 0 }, { upto: 500000, rate: 0.05 }, { upto: 1000000, rate: 0.20 }, { rate: 0.30 }],
    stdDeduction: 0,
    rebateThreshold: 500000,
    rebateAmount: FULL_REBATE,
    surcharge: [{ above: 5000000, rate: 0.10 }, { above: 10000000, rate: 0.15 }],
    cessRate: 0.04,
    deductionCaps: DEDUCTION_CAPS,
  },
  new: {
    schemaVersion: 1,
    slabs: [
      { upto: 400000, rate: 0 }, { upto: 800000, rate: 0.05 }, { upto: 1200000, rate: 0.10 },
      { upto: 1600000, rate: 0.15 }, { upto: 2000000, rate: 0.20 }, { upto: 2400000, rate: 0.25 },
      { rate: 0.30 },
    ],
    stdDeduction: 75000,
    rebateThreshold: 1200000,
    rebateAmount: FULL_REBATE,
    surcharge: [{ above: 5000000, rate: 0.10 }, { above: 10000000, rate: 0.15 }],
    cessRate: 0.04,
    deductionCaps: DEDUCTION_CAPS,
  },
}

// activeRuleSet resolves the parsed rules to use for `regime`, preferring an
// "active" TaxRules row matching `fy`, falling back to any active row for
// the regime (any FY), then to the bundled default. Mirrors
// backend/internal/models/tax_rules.go's activeRuleSet/ActiveTaxRules.
export function activeRuleSet(rows, fy, regime) {
  const pick = matchFY => {
    let best = null
    for (const r of rows || []) {
      if (r.status !== 'active' || (r.regime || '').toLowerCase() !== regime) continue
      if (matchFY && r.fy !== fy) continue
      if (!best || (r.activated_date || '') > (best.activated_date || '')) best = r
    }
    return best
  }
  const row = pick(true) || pick(false)
  if (!row) return DEFAULT_RULES[regime] || DEFAULT_RULES.old
  try {
    return JSON.parse(row.rules_json)
  } catch {
    return DEFAULT_RULES[regime] || DEFAULT_RULES.old
  }
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

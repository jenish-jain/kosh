import { describe, it, expect } from 'vitest'
import { computeTax, slabLabel, currentFY, activeRuleSet, DEFAULT_RULES } from './taxMath.js'

// Reference implementations of the pre-rules-engine hardcoded tax functions
// that used to live in this file, before the generic computeTax() engine
// replaced them. Kept here ONLY to regression-test that the migration to
// data-driven rules preserves behavior — except one documented, intentional
// fix (see "old regime: fixes the missing 87A rebate" below). Mirrors
// backend/internal/models/tax_rules_migration_test.go.
function oldTaxOldRegime(income) {
  if (income <= 250000) return 0
  let t = 0
  if (income > 1000000) t += (income - 1000000) * 0.30
  if (income > 500000) t += (Math.min(income, 1000000) - 500000) * 0.20
  if (income > 250000) t += (Math.min(income, 500000) - 250000) * 0.05
  let surcharge = 0
  if (income > 10000000) surcharge = t * 0.15
  else if (income > 5000000) surcharge = t * 0.10
  const cess = (t + surcharge) * 0.04
  return Math.round(t + surcharge + cess)
}

function oldTaxNewRegime(gross) {
  const taxable = Math.max(0, gross - 75000)
  let t = 0
  if (taxable > 2400000) t += (taxable - 2400000) * 0.30
  if (taxable > 2000000) t += (Math.min(taxable, 2400000) - 2000000) * 0.25
  if (taxable > 1600000) t += (Math.min(taxable, 2000000) - 1600000) * 0.20
  if (taxable > 1200000) t += (Math.min(taxable, 1600000) - 1200000) * 0.15
  if (taxable > 800000) t += (Math.min(taxable, 1200000) - 800000) * 0.10
  if (taxable > 400000) t += (Math.min(taxable, 800000) - 400000) * 0.05
  if (taxable <= 1200000) t = 0
  let surcharge = 0
  if (gross > 10000000) surcharge = t * 0.15
  else if (gross > 5000000) surcharge = t * 0.10
  const cess = (t + surcharge) * 0.04
  return Math.round(t + surcharge + cess)
}

const incomeSamples = [
  0, 1, 100000,
  249999, 250000, 250001,
  399999, 400000, 400001,
  499999, 500000, 500001,
  799999, 800000, 800001,
  999999, 1000000, 1000001,
  1199999, 1200000, 1200001,
  1599999, 1600000, 1600001,
  1999999, 2000000, 2000001,
  2399999, 2400000, 2400001,
  4999999, 5000000, 5000001,
  9999999, 10000000, 10000001,
  1234567, 3456789, 7654321, 15000000,
]

describe('computeTax — migration parity with the old hardcoded functions', () => {
  it('new regime matches the old hardcoded function for every sample (no behavior change)', () => {
    for (const income of incomeSamples) {
      expect(computeTax(DEFAULT_RULES.new, income)).toBe(oldTaxNewRegime(income))
    }
  })

  it('old regime matches the old hardcoded function above the rebate band', () => {
    for (const income of incomeSamples) {
      if (income <= 500000) continue // covered separately — see the next test
      expect(computeTax(DEFAULT_RULES.old, income)).toBe(oldTaxOldRegime(income))
    }
  })

  it('old regime: fixes the missing 87A rebate (intentional difference, not a regression)', () => {
    // The old hardcoded oldTaxOldRegime() never zeroed tax for taxable
    // income <= Rs 5L, even though Section 87A requires it. The new engine's
    // default rule set includes the rebate — this is a real bug fix.
    for (const income of [300000, 400000, 499999, 500000]) {
      expect(oldTaxOldRegime(income)).toBeGreaterThan(0) // sanity: old function was indeed buggy here
      expect(computeTax(DEFAULT_RULES.old, income)).toBe(0)
    }
    expect(computeTax(DEFAULT_RULES.old, 500001)).toBeGreaterThan(0)
  })
})

describe('slabLabel', () => {
  it('labels the new-regime rebate band', () => {
    expect(slabLabel(DEFAULT_RULES.new, 1000000)).toBe('10% · 0 via 87A')
  })
  it('labels the old-regime top band with surcharge', () => {
    expect(slabLabel(DEFAULT_RULES.old, 6000000)).toBe('30% + surcharge')
  })
})

describe('activeRuleSet', () => {
  const fy = currentFY().label

  it('falls back to the bundled default when no rows are given', () => {
    expect(activeRuleSet([], fy, 'old')).toEqual(DEFAULT_RULES.old)
  })

  it('picks the active row for the given FY+regime', () => {
    const rows = [
      { fy, regime: 'old', status: 'active', activated_date: '2026-04-01', rules_json: JSON.stringify({ ...DEFAULT_RULES.old, stdDeduction: 12345 }) },
      { fy, regime: 'new', status: 'active', activated_date: '2026-04-01', rules_json: JSON.stringify(DEFAULT_RULES.new) },
    ]
    expect(activeRuleSet(rows, fy, 'old').stdDeduction).toBe(12345)
  })

  it('prefers the most recently activated row when more than one is active for the same FY+regime', () => {
    const rows = [
      { fy, regime: 'old', status: 'active', activated_date: '2026-01-01', rules_json: JSON.stringify({ ...DEFAULT_RULES.old, stdDeduction: 1 }) },
      { fy, regime: 'old', status: 'active', activated_date: '2026-06-01', rules_json: JSON.stringify({ ...DEFAULT_RULES.old, stdDeduction: 2 }) },
    ]
    expect(activeRuleSet(rows, fy, 'old').stdDeduction).toBe(2)
  })

  it('ignores pending/rejected/superseded rows', () => {
    const rows = [
      { fy, regime: 'old', status: 'pending', activated_date: '', rules_json: JSON.stringify({ ...DEFAULT_RULES.old, stdDeduction: 999 }) },
    ]
    expect(activeRuleSet(rows, fy, 'old')).toEqual(DEFAULT_RULES.old)
  })
})

import { describe, it, expect } from 'vitest'
import {
  fmtINR,
  fmtCompact,
  fmtPct,
  fmtDate,
  holdingsFor,
  classTotals,
  memberTotal,
  netWorth,
  nextPremiumDue,
  nextEmiDue,
  upcomingOutflows,
} from './helpers.js'

// ── Shared fixture ───────────────────────────────────────────

const fixedDate = new Date('2025-06-15')

const sampleData = {
  members: [{ id: 'you', name: 'You' }, { id: 'mom', name: 'Mom' }],
  mf: [
    { id: 'mf1', member: 'you', invested: 10000, current: 12000 },
    { id: 'mf2', member: 'mom', invested: 20000, current: 22000 },
  ],
  stocks: [
    { id: 'st1', member: 'you', qty: 10, avg_price: 100, last_price: 120 },
  ],
  metals: [
    { id: 'me1', member: 'you', grams: 10, buy_rate: 5000, today_price: 6000 },
  ],
  fixed: [
    { id: 'fd1', member: 'you', principal: 100000, rate: 7, current_value: 107000 },
  ],
  insurance: [
    { id: 'in1', member: 'you', premium: 10000, freq: 'annual', paid: 50000, value: 60000, due_date: '2020-08-01' },
    { id: 'in2', member: 'mom', premium: 5000, freq: 'monthly', paid: 20000, value: 25000, due_date: '2020-03-10' },
  ],
  loans: [
    { id: 'ln1', member: 'you', principal: 500000, outstanding: 300000, rate: 8, emi: 15000, emi_day: 7, started: '2022-01-01', tenure_months: 180 },
  ],
  nps: [
    { id: 'nps1', member: 'you', units: 100, nav: 50, invested: 4000 },
  ],
  sips: [
    { id: 'sip1', member: 'you', amount: 5000, day: 10, status: 'active', fund: 'Test Fund' },
    { id: 'sip2', member: 'you', amount: 3000, day: 15, status: 'paused', fund: 'Paused Fund' },
  ],
}

// ── fmtINR ───────────────────────────────────────────────────

describe('fmtINR', () => {
  it('should format 0 as ₹0', () => {
    expect(fmtINR(0)).toBe('₹0')
  })

  it('should format 500 as ₹500', () => {
    expect(fmtINR(500)).toBe('₹500')
  })

  it('should format 1000 as ₹1,000', () => {
    expect(fmtINR(1000)).toBe('₹1,000')
  })

  it('should format 100000 as ₹1,00,000 (Indian grouping)', () => {
    expect(fmtINR(100000)).toBe('₹1,00,000')
  })

  it('should format 1234567 as ₹12,34,567', () => {
    expect(fmtINR(1234567)).toBe('₹12,34,567')
  })

  it('should format -500 as −₹500 (Unicode minus)', () => {
    expect(fmtINR(-500)).toBe('−₹500')
  })

  it('should format -1234 as −₹1,234', () => {
    expect(fmtINR(-1234)).toBe('−₹1,234')
  })
})

// ── fmtCompact ───────────────────────────────────────────────

describe('fmtCompact', () => {
  it('should format 500 as ₹500', () => {
    expect(fmtCompact(500)).toBe('₹500')
  })

  it('should format 1500 as ₹1.5K', () => {
    expect(fmtCompact(1500)).toBe('₹1.5K')
  })

  it('should format 100000 as ₹1L', () => {
    expect(fmtCompact(100000)).toBe('₹1L')
  })

  it('should format 1234500 as ₹12.35L', () => {
    expect(fmtCompact(1234500)).toBe('₹12.35L')
  })

  it('should format 10000000 as ₹1Cr', () => {
    expect(fmtCompact(10000000)).toBe('₹1Cr')
  })

  it('should format -500000 as −₹5L (Unicode minus)', () => {
    expect(fmtCompact(-500000)).toBe('−₹5L')
  })
})

// ── fmtPct ───────────────────────────────────────────────────

describe('fmtPct', () => {
  it('should format 12.5 as +12.5%', () => {
    expect(fmtPct(12.5)).toBe('+12.5%')
  })

  it('should format -3.0 as -3.0%', () => {
    expect(fmtPct(-3.0)).toBe('-3.0%')
  })

  it('should format 0 as +0.0%', () => {
    expect(fmtPct(0)).toBe('+0.0%')
  })

  it('should format 100 with dp=0 as +100%', () => {
    expect(fmtPct(100, 0)).toBe('+100%')
  })
})

// ── fmtDate ──────────────────────────────────────────────────

describe('fmtDate', () => {
  it('should format a Date object as "5 Jun 2024"', () => {
    expect(fmtDate(new Date(2024, 5, 5))).toBe('5 Jun 2024')
  })

  it('should format an ISO string "2024-01-01" as "1 Jan 2024"', () => {
    expect(fmtDate('2024-01-01')).toBe('1 Jan 2024')
  })
})

// ── holdingsFor ──────────────────────────────────────────────

describe('holdingsFor', () => {
  it('should return ALL holdings when memberId is null', () => {
    const h = holdingsFor(sampleData, null)
    expect(h.mf).toHaveLength(2)
    expect(h.stocks).toHaveLength(1)
    expect(h.insurance).toHaveLength(2)
  })

  it('should return only "you" items when memberId is "you"', () => {
    const h = holdingsFor(sampleData, 'you')
    expect(h.mf).toHaveLength(1)
    expect(h.mf[0].id).toBe('mf1')
    expect(h.stocks).toHaveLength(1)
    expect(h.insurance).toHaveLength(1)
    expect(h.insurance[0].id).toBe('in1')
  })

  it('should return empty arrays for an unknown memberId', () => {
    const h = holdingsFor(sampleData, 'unknown')
    expect(h.mf).toHaveLength(0)
    expect(h.stocks).toHaveLength(0)
    expect(h.metals).toHaveLength(0)
    expect(h.fixed).toHaveLength(0)
    expect(h.insurance).toHaveLength(0)
    expect(h.loans).toHaveLength(0)
    expect(h.nps).toHaveLength(0)
  })

  it('should handle data missing the nps key gracefully', () => {
    const dataWithoutNps = { ...sampleData, nps: undefined }
    const h = holdingsFor(dataWithoutNps, 'you')
    expect(h.nps).toEqual([])
  })
})

// ── classTotals ──────────────────────────────────────────────

describe('classTotals', () => {
  it('should compute mf totals for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.mf.cur).toBe(12000)
    expect(t.mf.inv).toBe(10000)
  })

  it('should compute stocks totals for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.stocks.cur).toBe(10 * 120)   // 1200
    expect(t.stocks.inv).toBe(10 * 100)   // 1000
  })

  it('should compute metals totals for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.metals.cur).toBe(10 * 6000)  // 60000
    expect(t.metals.inv).toBe(10 * 5000)  // 50000
  })

  it('should compute fixed totals for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.fixed.cur).toBe(107000)
    expect(t.fixed.inv).toBe(100000)
  })

  it('should compute insurance totals for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.insurance.cur).toBe(60000)
    expect(t.insurance.inv).toBe(50000)
  })

  it('should compute nps totals for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.nps.cur).toBe(100 * 50)  // 5000
    expect(t.nps.inv).toBe(4000)
  })

  it('should compute liabilities for "you"', () => {
    const t = classTotals(sampleData, 'you')
    expect(t.liabilities.cur).toBe(300000)
    expect(t.liabilities.monthly).toBe(15000)
  })

  it('should compute total.cur as sum of all asset classes for "you"', () => {
    const t = classTotals(sampleData, 'you')
    // 12000 + 1200 + 60000 + 107000 + 60000 + 5000 = 245200
    expect(t.total.cur).toBe(245200)
  })

  it('should compute total.inv as sum of all asset classes for "you"', () => {
    const t = classTotals(sampleData, 'you')
    // 10000 + 1000 + 50000 + 100000 + 50000 + 4000 = 215000
    expect(t.total.inv).toBe(215000)
  })
})

// ── memberTotal ──────────────────────────────────────────────

describe('memberTotal', () => {
  it('should return total.cur for "you"', () => {
    expect(memberTotal(sampleData, 'you')).toBe(245200)
  })
})

// ── netWorth ─────────────────────────────────────────────────

describe('netWorth', () => {
  it('should return total.cur minus liabilities.cur for "you"', () => {
    // 245200 - 300000 = -54800
    expect(netWorth(sampleData, 'you')).toBe(-54800)
  })
})

// ── nextPremiumDue ───────────────────────────────────────────

describe('nextPremiumDue', () => {
  it('should return Aug 1 2025 for annual policy due Aug 1, queried from Jun 15 2025', () => {
    const policy = { freq: 'annual', due_date: '2020-08-01' }
    const result = nextPremiumDue(policy, new Date('2025-06-15'))
    expect(result).toEqual(new Date(2025, 7, 1))  // Aug = month 7
  })

  it('should return Aug 1 2026 for annual policy due Aug 1, queried from Sep 1 2025 (day has passed)', () => {
    const policy = { freq: 'annual', due_date: '2020-08-01' }
    const result = nextPremiumDue(policy, new Date('2025-09-01'))
    expect(result).toEqual(new Date(2026, 7, 1))
  })

  it('should return Jul 10 2025 for monthly policy due on day 10, queried from Jun 15 2025 (day 10 already passed)', () => {
    const policy = { freq: 'monthly', due_date: '2020-03-10' }
    const result = nextPremiumDue(policy, new Date('2025-06-15'))
    expect(result).toEqual(new Date(2025, 6, 10))  // Jul = month 6
  })

  it('should return the anchor date for single policy with future due_date', () => {
    const policy = { freq: 'single', due_date: '2025-12-01' }
    const result = nextPremiumDue(policy, new Date('2025-06-15'))
    expect(result).toEqual(new Date('2025-12-01'))
  })

  it('should return null for single policy with past due_date', () => {
    const policy = { freq: 'single', due_date: '2025-01-01' }
    const result = nextPremiumDue(policy, new Date('2025-06-15'))
    expect(result).toBeNull()
  })

  it('should return null when due_date is missing', () => {
    expect(nextPremiumDue({ freq: 'annual' }, fixedDate)).toBeNull()
  })

  it('should return null when policy is null', () => {
    expect(nextPremiumDue(null, fixedDate)).toBeNull()
  })
})

// ── nextEmiDue ───────────────────────────────────────────────

describe('nextEmiDue', () => {
  it('should return Jul 7 2025 when emi_day=7 and from=Jun 15 2025 (day 7 already passed)', () => {
    const loan = { emi_day: 7, started: '2022-01-01', tenure_months: 180 }
    const result = nextEmiDue(loan, new Date('2025-06-15'))
    expect(result).toEqual(new Date(2025, 6, 7))  // Jul = month 6
  })

  it('should return Jun 20 2025 when emi_day=20 and from=Jun 15 2025 (day not yet passed)', () => {
    const loan = { emi_day: 20, started: '2022-01-01', tenure_months: 180 }
    const result = nextEmiDue(loan, new Date('2025-06-15'))
    expect(result).toEqual(new Date(2025, 5, 20))  // Jun = month 5
  })

  it('should return null for an expired loan (started Jan 2020, tenure 48 months → ended Jan 2024)', () => {
    const loan = { emi_day: 7, started: '2020-01-01', tenure_months: 48 }
    const result = nextEmiDue(loan, new Date('2025-06-15'))
    expect(result).toBeNull()
  })

  it('should return null when emi_day is null', () => {
    expect(nextEmiDue({ emi_day: null }, fixedDate)).toBeNull()
  })

  it('should return null when loan is null', () => {
    expect(nextEmiDue(null, fixedDate)).toBeNull()
  })
})

// ── upcomingOutflows ─────────────────────────────────────────

describe('upcomingOutflows', () => {
  it('should only include active SIPs, not paused ones', () => {
    const results = upcomingOutflows(sampleData, 'you', 60)
    const sipItems = results.filter(i => i.kind === 'sip')
    expect(sipItems.every(i => i.id !== 'sip2')).toBe(true)
  })

  it('should return items sorted soonest-first', () => {
    const results = upcomingOutflows(sampleData, 'you', 60)
    for (let i = 1; i < results.length; i++) {
      expect(results[i].date.getTime()).toBeGreaterThanOrEqual(results[i - 1].date.getTime())
    }
  })

  it('should include kind, label, amount, date, and id on each item', () => {
    const results = upcomingOutflows(sampleData, 'you', 60)
    for (const item of results) {
      expect(item).toHaveProperty('id')
      expect(item).toHaveProperty('kind')
      expect(item).toHaveProperty('amount')
      expect(item).toHaveProperty('date')
    }
  })

  it('should filter by memberId — "mom" should not see "you" SIPs', () => {
    const results = upcomingOutflows(sampleData, 'mom', 60)
    const sipItems = results.filter(i => i.kind === 'sip')
    expect(sipItems.every(i => i.id !== 'sip1')).toBe(true)
  })

  it('should return an empty array when data has no sips, insurance, or loans', () => {
    const emptyData = { sips: [], insurance: [], loans: [] }
    expect(upcomingOutflows(emptyData, null, 30)).toEqual([])
  })
})

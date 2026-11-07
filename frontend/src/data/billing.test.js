import { planRunning, daysLeft, canUse, missingFeatures } from './billing.js'

// Mirrors backend/internal/billing/billing_test.go TestTierFor (true for
// the trial and paid tiers) — the UI must agree with the server.
describe('planRunning', () => {
  const now = Date.parse('2026-11-07T12:00:00Z')
  const later = '2026-11-07T13:00:00Z', earlier = '2026-11-07T11:00:00Z'
  it.each([
    ['trial running', { status: 'trialing', trial_ends_at: later }, true],
    ['trial over', { status: 'trialing', trial_ends_at: earlier }, false],
    ['active', { status: 'active' }, true],
    ['payment retrying', { status: 'past_due' }, true],
    ['cancelled, paid time left', { status: 'cancelled', current_period_end: later }, true],
    ['cancelled, period over', { status: 'cancelled', current_period_end: earlier }, false],
    ['lapsed', { status: 'lapsed', current_period_end: later }, false],
  ])('%s', (_, b, want) => expect(planRunning(b, now)).toBe(want))
})

describe('daysLeft', () => {
  it('rounds up and never goes negative', () => {
    const now = Date.parse('2026-11-07T12:00:00Z')
    expect(daysLeft('2026-11-08T00:00:00Z', now)).toBe(1)
    expect(daysLeft('2026-11-01T00:00:00Z', now)).toBe(0)
  })
})

describe('canUse', () => {
  it('follows the household features, allowing anything not listed', () => {
    const h = { features: { share: false, edit: true } }
    expect(canUse(h, 'share')).toBe(false)
    expect(canUse(h, 'edit')).toBe(true)
    expect(canUse(h, 'ask')).toBe(true)
    expect(canUse(null, 'share')).toBe(true)
    expect(missingFeatures(h)).toEqual(['Family sharing'])
  })
})

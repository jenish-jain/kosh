// What the marketing site says about the hosted plan. Must match the
// Razorpay plans (the app itself shows prices fetched from Razorpay) and
// backend/internal/hosted.MonthlyQuotas / TrialLength.
export const PRICING = {
  monthly: 149,
  annual: 1499,
  trialDays: 14,
  maxPeople: 6,
  quotas: { ask: 200, parse: 30, tax: 20 },
}

// Where "Start free trial" goes. The marketing pages ship in every build,
// so a deploy that isn't the hosted service (e.g. a self-hosted instance)
// sets VITE_HOSTED_APP_URL to the hosted app's address at build time.
export const HOSTED_APP_URL = import.meta.env.VITE_HOSTED_APP_URL || '/app'

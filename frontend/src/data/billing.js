// Hosted billing: API calls, Razorpay Checkout, and what the household's
// plan includes (the server enforces it; this only decides what to show).

async function request(path, opts = {}) {
  const res = await fetch(path, { credentials: 'include', ...opts })
  if (!res.ok) {
    const err = new Error((await res.text().catch(() => '')).trim() || `Server error ${res.status}`)
    err.status = res.status
    throw err
  }
  return res.json()
}
const post = (path, body) => request(path, {
  method: 'POST', headers: { 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined,
})

export const fetchPlans = () => request('/api/billing/plans')
export const startCheckout = plan => post('/api/billing/checkout', { plan })
export const verifyPayment = resp => post('/api/billing/verify', resp)
export const cancelSubscription = () => post('/api/billing/cancel')

// Feature labels, mirroring backend/internal/billing/features.go.
export const FEATURE_LABELS = {
  edit: 'Adding and editing data',
  share: 'Family sharing',
  invite: 'Household invites',
  ask: 'Ask Kosh',
  parse: 'Document scanning',
  tax: 'Tax recommendations',
}

// canUse: whether the household's tier includes feature. Self-hosted (no
// household) and older responses without features allow everything.
export const canUse = (household, feature) => household?.features?.[feature] !== false

// The features the household's tier leaves out, as labels.
export const missingFeatures = household =>
  Object.keys(FEATURE_LABELS).filter(f => !canUse(household, f)).map(f => FEATURE_LABELS[f])

// planRunning: a trial or paid period is in effect (the server's tier is
// "trial" or "paid", not "free"). Mirrors billing.TierFor.
export function planRunning(b, now = Date.now()) {
  if (!b) return true
  switch (b.status) {
    case 'trialing': return now < Date.parse(b.trial_ends_at)
    case 'active': case 'past_due': return true
    case 'cancelled': return now < Date.parse(b.current_period_end)
    default: return false
  }
}

export const daysLeft = (iso, now = Date.now()) => Math.max(0, Math.ceil((Date.parse(iso) - now) / 86_400_000))

export const fmtRupees = paise => '₹' + (paise / 100).toLocaleString('en-IN', { maximumFractionDigits: 0 })

function loadCheckoutScript() {
  if (window.Razorpay) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = 'https://checkout.razorpay.com/v1/checkout.js'
    s.onload = resolve
    s.onerror = () => reject(new Error("Couldn't load the payment window. Check your connection and try again."))
    document.head.appendChild(s)
  })
}

// Runs the whole subscribe flow and resolves with the updated billing
// state once the payment is verified.
export async function subscribe(plan, { devLogin, householdName }) {
  const co = await startCheckout(plan)
  let resp
  if (devLogin) {
    // hosteddev: no Razorpay — the dev endpoint "pays" and sends the webhook.
    resp = await post('/api/dev/pay', { subscription_id: co.subscription_id })
  } else {
    await loadCheckoutScript()
    resp = await new Promise((resolve, reject) => {
      const rzp = new window.Razorpay({
        key: co.key_id,
        subscription_id: co.subscription_id,
        name: 'Kosh',
        description: `Kosh Family (${plan}) · ${householdName}`,
        prefill: { email: co.email },
        theme: { color: '#1C4A3A' },
        handler: resolve,
        modal: { ondismiss: () => reject(new Error('Checkout was closed before finishing.')) },
      })
      rzp.on('payment.failed', r => reject(new Error(r.error?.description || 'The payment failed.')))
      rzp.open()
    })
  }
  return (await verifyPayment(resp)).billing
}

import { planRunning, canUse, daysLeft } from '../data/billing.js'

// Hosted: a slim notice above the content when the plan needs attention.
export default function PlanBanner({ household, onOpen }) {
  const b = household?.billing
  if (!b) return null
  const isOwner = household.role === 'owner'
  let text, tone = 'info'
  if (!planRunning(b) && !canUse(household, 'edit')) {
    tone = 'warn'
    text = isOwner ? 'Your household is read-only. Subscribe to keep editing.' : `Read-only for now. Ask ${household.owner_email} to subscribe.`
  } else if (!planRunning(b)) {
    text = isOwner ? "You're on the free plan. Subscribe to unlock everything." : `On the free plan. ${household.owner_email} can subscribe to unlock everything.`
  } else if (b.status === 'past_due') {
    tone = 'warn'
    text = 'A payment failed. Razorpay will retry it, and everything keeps working meanwhile.'
  } else if (b.status === 'trialing') {
    const d = daysLeft(b.trial_ends_at)
    if (d > 7) return null
    text = `Free trial: ${d} day${d === 1 ? '' : 's'} left.`
  } else {
    return null
  }
  return (
    <div className={`plan-banner ${tone}`} role="status">
      <span>{text}</span>
      <button className="btn sm" onClick={onOpen}>{isOwner ? 'See plans' : 'Details'}</button>
    </div>
  )
}

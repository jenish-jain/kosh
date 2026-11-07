import { useEffect, useState } from 'react'
import { Modal } from './Primitives.jsx'
import { KICK } from '../data/tokens.js'
import { fetchPlans, subscribe, cancelSubscription, hasFullAccess, daysLeft, fmtRupees } from '../data/billing.js'
import { getUsage } from '../data/hostedApi.js'

const USAGE_LABELS = [['ask', 'Ask Kosh questions'], ['parse', 'Document scans'], ['tax', 'Tax recommendation runs']]

const fmtDate = iso => new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' })

function statusLine(b) {
  switch (b.status) {
    case 'trialing':
      return hasFullAccess(b) ? `Free trial: ${daysLeft(b.trial_ends_at)} day(s) left.` : 'Your free trial has ended.'
    case 'active':
      return b.cancel_at_period_end
        ? `Cancelled. Full access until ${fmtDate(b.current_period_end)}.`
        : `Subscribed (${b.plan}). Next renewal ${fmtDate(b.current_period_end)}.`
    case 'past_due': return 'Your last payment failed. Razorpay will retry, and everything keeps working meanwhile.'
    case 'cancelled':
      return hasFullAccess(b) ? `Cancelled. Full access until ${fmtDate(b.current_period_end)}.` : 'Your subscription has ended.'
    default: return 'Your subscription has ended.'
  }
}

// Owner: subscribe (monthly/annual) or cancel. Members: status only.
export default function PlanModal({ household, devLogin, onClose, onChanged, showToast }) {
  const b = household.billing
  const isOwner = household.role === 'owner'
  const [plans, setPlans] = useState(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const subscribed = (b.status === 'active' || b.status === 'past_due') && !b.cancel_at_period_end

  const [usage, setUsage] = useState(null)
  useEffect(() => { if (isOwner) fetchPlans().then(p => setPlans(p.plans)).catch(e => setError(e.message)) }, [isOwner])
  useEffect(() => { getUsage().then(setUsage).catch(() => {}) }, [])

  const pay = async plan => {
    setBusy(true)
    setError(null)
    try {
      await subscribe(plan, { devLogin, householdName: household.name })
      showToast?.('Subscribed. Thank you!')
      onChanged()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  const cancel = async () => {
    if (!window.confirm('Cancel your subscription? You keep full access until the end of the period you paid for, and your data stays in your Google Sheet either way.')) return
    setBusy(true)
    try { await cancelSubscription(); showToast?.('Subscription cancelled'); onChanged() } catch (e) { setError(e.message) } finally { setBusy(false) }
  }

  return (
    <Modal title="Plan" onClose={onClose} width={480}>
      <p className="share-intro"><strong>{statusLine(b)}</strong></p>
      {!hasFullAccess(b) && (
        <p className="share-intro">Your household is read-only: you can still view and export everything, and your data is always in your own Google Sheet.</p>
      )}
      {error && <div className="share-error">{error}</div>}
      {!isOwner ? (
        <p className="share-intro">The household owner, {household.owner_email}, manages the subscription.</p>
      ) : subscribed ? (
        <div className="share-actions">
          <button className="btn" onClick={cancel} disabled={busy}>Cancel subscription</button>
        </div>
      ) : !plans ? <div style={KICK}>Loading plans…</div> : (
        <div className="plan-grid">
          {plans.map(p => (
            <div key={p.id} className="plan-card">
              <div style={KICK}>{p.id === 'annual' ? 'Annual' : 'Monthly'}</div>
              <div className="plan-price">{fmtRupees(p.amount)}<span>/{p.id === 'annual' ? 'year' : 'month'}</span></div>
              {p.id === 'annual' && <div className="cell-sub">About 2 months free</div>}
              <button className="btn primary" style={{ width: '100%', justifyContent: 'center', marginTop: 12 }} onClick={() => pay(p.id)} disabled={busy}>
                {busy ? 'Opening…' : 'Subscribe'}
              </button>
            </div>
          ))}
        </div>
      )}
      {usage && (
        <div className="usage-box">
          <div style={{ ...KICK, marginBottom: 8 }}>AI allowance this month</div>
          {USAGE_LABELS.map(([k, label]) => (
            <div key={k} className="usage-row">
              <span>{label}</span>
              <span className="num">{usage.used[k] || 0} / {usage.limits[k]}</span>
            </div>
          ))}
          <div className="cell-sub" style={{ marginTop: 6 }}>Resets {new Date(usage.resets_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'long' })}. Everything else is unlimited.</div>
        </div>
      )}
      {isOwner && b.status === 'trialing' && hasFullAccess(b) && !subscribed && (
        <p className="hosted-fine">Subscribing now doesn't cut your trial short. The first charge happens when it ends.</p>
      )}
    </Modal>
  )
}

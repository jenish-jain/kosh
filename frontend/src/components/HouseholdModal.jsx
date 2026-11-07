import { useCallback, useEffect, useState } from 'react'
import { Modal, Field } from './Primitives.jsx'
import { Icon } from './Icons.jsx'
import { KICK } from '../data/tokens.js'
import { hasFullAccess, daysLeft } from '../data/billing.js'
import { getHousehold, inviteMember, cancelInvite, removeMember, leaveHousehold } from '../data/hostedApi.js'

function planSummary(b) {
  if (!hasFullAccess(b)) return 'Read-only (no active plan)'
  if (b.status === 'trialing') return `Free trial, ${daysLeft(b.trial_ends_at)} day(s) left`
  if (b.cancel_at_period_end || b.status === 'cancelled') return 'Cancelled, active until the paid period ends'
  if (b.status === 'past_due') return 'Payment being retried'
  return `Subscribed (${b.plan})`
}

const fmtDate = iso => new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })

// Hosted mode: who's in the household. The owner invites (by link, bound to
// the invitee's Google account) and removes members; members can leave.
export default function HouseholdModal({ onClose, onLeft, onOpenPlan, showToast }) {
  const [state, setState] = useState(null)
  const [error, setError] = useState(null)
  const [email, setEmail] = useState('')
  const [busy, setBusy] = useState(false)
  const [newLink, setNewLink] = useState(null)

  const refresh = useCallback(() => {
    getHousehold().then(setState).catch(e => setError(e.message))
  }, [])
  useEffect(refresh, [refresh])

  const run = async (fn, ok) => {
    setError(null)
    setBusy(true)
    try { await ok(await fn()) } catch (e) { setError(e.message) } finally { setBusy(false) }
  }

  const invite = e => {
    e.preventDefault()
    if (!email.trim()) return
    run(() => inviteMember(email.trim()), res => {
      setNewLink({ email: res.invite.email, url: window.location.origin + res.link_path })
      setEmail('')
      refresh()
    })
  }

  const copy = async url => {
    try { await navigator.clipboard.writeText(url); showToast?.('Invite link copied') } catch { window.prompt('Copy this link:', url) }
  }

  const isOwner = state?.household.role === 'owner'

  return (
    <Modal title="Household" onClose={onClose} width={520}>
      {error && <div className="share-error">{error}</div>}
      {!state ? <div style={KICK}>Loading…</div> : (
        <>
          <p className="share-intro">
            Everyone here signs in with their own Google account and sees the same household data.
            {isOwner && <> Your data lives in <a href={state.household.sheet_url} target="_blank" rel="noopener noreferrer">your Google Sheet</a>.</>}
          </p>

          <div className="share-row" style={{ borderTop: '1px solid var(--line)' }}>
            <div className="share-row-main">
              <div className="cell-strong">Plan</div>
              <div className="cell-sub">{planSummary(state.household.billing)}</div>
            </div>
            <div className="share-row-actions"><button className="btn ghost sm" onClick={onOpenPlan}>{isOwner ? 'Manage' : 'View'}</button></div>
          </div>

          <ul className="share-list" style={{ marginTop: 12 }}>
            {state.members.map(m => (
              <li key={m.email} className="share-row">
                <div className="share-row-main">
                  <div className="cell-strong" style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.email}</div>
                  <div className="cell-sub">
                    <span className={`pill ${m.role === 'owner' ? 'accent' : 'neutral'}`} style={{ marginRight: 8 }}>{m.role}</span>
                    since {fmtDate(m.joined_at)}
                  </div>
                </div>
                {isOwner && m.role !== 'owner' && (
                  <div className="share-row-actions">
                    <button className="btn ghost sm" disabled={busy} onClick={() => {
                      if (!window.confirm(`Remove ${m.email}? They lose access immediately.`)) return
                      run(() => removeMember(m.email), () => { showToast?.(`${m.email} removed`); refresh() })
                    }}>Remove</button>
                  </div>
                )}
              </li>
            ))}
            {(state.invites || []).map(inv => (
              <li key={inv.id} className="share-row">
                <div className="share-row-main">
                  <div className="cell-strong" style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{inv.email}</div>
                  <div className="cell-sub"><span className="pill gold" style={{ marginRight: 8 }}>invited</span>link expires {fmtDate(inv.expires_at)}</div>
                </div>
                <div className="share-row-actions">
                  <button className="btn ghost sm" disabled={busy} onClick={() => run(() => cancelInvite(inv.id), refresh)}>Cancel</button>
                </div>
              </li>
            ))}
          </ul>

          {isOwner && (
            <form onSubmit={invite} style={{ marginTop: 18 }}>
              <Field label="Invite by Google account email">
                <div className="enrol-link">
                  <input className="input" type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="spouse@gmail.com" disabled={busy} />
                  <button className="btn primary" disabled={busy || !email.trim()}><Icon name="plus" size={14} /> Invite</button>
                </div>
              </Field>
            </form>
          )}

          {newLink && (
            <div className="share-warn">
              Send this link to <strong>{newLink.email}</strong>. It only works when they sign in with that Google account, and it expires in 7 days.
              <div className="enrol-link" style={{ marginTop: 8 }}>
                <input className="input" value={newLink.url} readOnly onFocus={e => e.target.select()} />
                <button className="btn" onClick={() => copy(newLink.url)}>Copy</button>
              </div>
            </div>
          )}

          {!isOwner && (
            <div className="share-actions">
              <button className="btn" disabled={busy} onClick={() => {
                if (!window.confirm(`Leave ${state.household.name}? You'll lose access to its data.`)) return
                run(leaveHousehold, onLeft)
              }}>Leave household</button>
            </div>
          )}
        </>
      )}
    </Modal>
  )
}

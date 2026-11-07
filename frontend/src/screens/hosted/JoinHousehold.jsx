import { useState } from 'react'
import CenterCard, { ErrorNote } from './CenterCard.jsx'
import { acceptInvite } from '../../data/hostedApi.js'

// Shown when the account arrived through an invite link (/join#<token>).
export default function JoinHousehold({ token, email, hasHousehold, onDone, onDismiss, onSignOut }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  const join = async () => {
    setBusy(true)
    setError(null)
    try {
      const res = await acceptInvite(token)
      onDone(res.household)
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <CenterCard kicker="Invitation" title="Join a household on Kosh" email={email} onSignOut={onSignOut}>
      <p className="hosted-text">
        You've been invited to see and help manage a family's finances on Kosh as <strong>{email}</strong>.
        Invites only work for the Google account they were sent to.
      </p>
      {hasHousehold && (
        <p className="hosted-text">You're already part of a household. You'd need to leave it before joining another.</p>
      )}
      <ErrorNote>{error}</ErrorNote>
      <button className="btn primary hosted-cta" onClick={join} disabled={busy}>{busy ? 'Joining…' : 'Join household'}</button>
      <button className="btn ghost hosted-cta" onClick={onDismiss} disabled={busy}>
        {hasHousehold ? 'Go to my household' : 'Not now'}
      </button>
    </CenterCard>
  )
}

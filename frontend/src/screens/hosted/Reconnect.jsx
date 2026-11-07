import { useState } from 'react'
import CenterCard, { ErrorNote } from './CenterCard.jsx'
import { reconnectDrive } from '../../data/hostedApi.js'
import { requestDriveCode } from '../../auth/driveCode.js'

// Google stopped accepting the household's stored grant (access removed in
// the owner's Google account, password change...). The data is safe in
// their Sheet; Kosh just can't reach it until the owner reconnects.
export default function Reconnect({ household, email, clientId, devLogin, onDone, onSignOut }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const isOwner = household.role === 'owner'

  const reconnect = () => {
    setError(null)
    requestDriveCode({ clientId, email, devLogin })
      .then(code => { setBusy(true); return reconnectDrive(code) })
      .then(res => onDone(res.household))
      .catch(e => setError(e.message))
      .finally(() => setBusy(false))
  }

  return (
    <CenterCard kicker={household.name} title="Kosh lost access to your Google Drive" email={email} onSignOut={onSignOut}>
      <p className="hosted-text">
        Your data is safe in your Google Sheet. Kosh just can't reach it right now, which usually happens after
        access was removed in Google account settings.
      </p>
      {isOwner ? (
        <>
          <ErrorNote>{error}</ErrorNote>
          <button className="btn primary hosted-cta" onClick={reconnect} disabled={busy}>
            {busy ? 'Reconnecting…' : 'Reconnect Google Drive'}
          </button>
        </>
      ) : (
        <p className="hosted-text">Ask <strong>{household.owner_email}</strong> to sign in to Kosh and reconnect.</p>
      )}
    </CenterCard>
  )
}

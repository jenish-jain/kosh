import { useState } from 'react'
import CenterCard, { ErrorNote } from './CenterCard.jsx'
import { createHousehold } from '../../data/hostedApi.js'
import { requestDriveCode } from '../../auth/driveCode.js'

// First-run for a new hosted account: name the household, then grant Kosh
// access to create its spreadsheet in the customer's own Google Drive.
export default function Onboarding({ email, clientId, devLogin, onDone, onSignOut }) {
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  const start = () => {
    const trimmed = name.trim()
    if (!trimmed) { setError('Give your household a name first.'); return }
    setError(null)
    // No await before requestDriveCode — the popup must open inside the click.
    requestDriveCode({ clientId, email, devLogin })
      .then(code => { setBusy(true); return createHousehold(code, trimmed) })
      .then(res => onDone(res.household))
      .catch(e => setError(e.message))
      .finally(() => setBusy(false))
  }

  return (
    <CenterCard kicker="Welcome to Kosh" title="Set up your household" email={email} onSignOut={onSignOut}>
      <p className="hosted-text">
        Kosh keeps your family's finances in a Google Sheet <strong>in your own Drive</strong>, not in our database.
        You can open it, export it, or take it with you to a self-hosted Kosh any time.
      </p>
      <label className="field" style={{ textAlign: 'left' }}>
        <span className="field-label">Household name</span>
        <input className="input" value={name} onChange={e => setName(e.target.value)} maxLength={60}
          placeholder="e.g. The Sharmas" autoFocus disabled={busy}
          onKeyDown={e => { if (e.key === 'Enter') start() }} />
      </label>
      <ErrorNote>{error}</ErrorNote>
      <button className="btn primary hosted-cta" onClick={start} disabled={busy}>
        {busy ? 'Creating your spreadsheet…' : 'Connect Google Drive & start free trial'}
      </button>
      <p className="hosted-fine">
        Google will ask to let Kosh “see, edit, create and delete only the specific Google Drive files you use with this app”.
        That's the narrowest access there is. Kosh can't see anything else in your Drive.
        Your 14-day trial starts now, and no card is needed.
      </p>
    </CenterCard>
  )
}

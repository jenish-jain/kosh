import { Icon } from '../Icons.jsx'

export default function ErrorScreen({ message }) {
  // Hosted: the household's Google grant was revoked mid-session.
  const reconnect = message === 'reconnect_required'
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: 12, padding: 32 }}>
      <div style={{ fontFamily: 'var(--serif)', fontSize: 40, color: 'var(--neg)' }}>!</div>
      <div style={{ fontSize: 16, fontWeight: 700 }}>{reconnect ? 'Kosh lost access to your Google Drive' : 'Could not connect to backend'}</div>
      <div style={{ fontSize: 13, color: 'var(--ink-3)', maxWidth: 380, textAlign: 'center', lineHeight: 1.6 }}>
        {reconnect ? 'Your data is safe in its Google Sheet. Reload to see how to reconnect.' : message}
      </div>
      <button className="btn" onClick={() => window.location.reload()}>
        <Icon name="refresh" size={15} /> Retry
      </button>
    </div>
  )
}

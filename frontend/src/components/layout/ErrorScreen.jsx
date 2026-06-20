import { Icon } from '../Icons.jsx'

export default function ErrorScreen({ message }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: 12, padding: 32 }}>
      <div style={{ fontFamily: 'var(--serif)', fontSize: 40, color: 'var(--neg)' }}>!</div>
      <div style={{ fontSize: 16, fontWeight: 700 }}>Could not connect to backend</div>
      <div style={{ fontSize: 13, color: 'var(--ink-3)', maxWidth: 380, textAlign: 'center', lineHeight: 1.6 }}>{message}</div>
      <button className="btn" onClick={() => window.location.reload()}>
        <Icon name="refresh" size={15} /> Retry
      </button>
    </div>
  )
}

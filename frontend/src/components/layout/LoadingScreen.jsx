import { SERIF } from '../../data/tokens.js'

export default function LoadingScreen() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100vh', gap: 16, color: 'var(--ink-3)' }}>
      <div style={{ fontFamily: SERIF, fontSize: 64, color: 'var(--accent)', lineHeight: 1 }}>क</div>
      <div style={{ fontSize: 14, fontWeight: 600, letterSpacing: '.08em', textTransform: 'uppercase' }}>Loading…</div>
    </div>
  )
}

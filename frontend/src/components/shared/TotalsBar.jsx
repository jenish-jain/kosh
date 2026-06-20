import { fmtINR } from '../../data/format.js'
import { KICK, SERIF } from '../../data/tokens.js'

// ── Totals bar ───────────────────────────────────────────────
// Renders a footer bar showing total invested + current value.
// Extracted from Investments.jsx for reuse across table components.
export default function TotalsBar({ inv, cur, label, curLabel = 'Current value' }) {
  return (
    <div className="totals-bar">
      <span style={{ fontWeight: 700, fontSize: 13.5 }}>{label}</span>
      <div style={{ flex: 1 }} />
      <div style={{ textAlign: 'right' }}>
        <div style={KICK}>Total invested</div>
        <div className="num" style={{ fontFamily: SERIF, fontWeight: 500, fontSize: 19, marginTop: 2 }}>{fmtINR(inv)}</div>
      </div>
      <div style={{ textAlign: 'right' }}>
        <div style={KICK}>{curLabel}</div>
        <div className="num" style={{ fontFamily: SERIF, fontWeight: 500, fontSize: 19, marginTop: 2 }}>{cur > 0 ? fmtINR(cur) : '—'}</div>
      </div>
    </div>
  )
}

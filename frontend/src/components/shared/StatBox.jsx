import { KICK, SERIF } from '../../data/tokens.js'

// ── StatBox ───────────────────────────────────────────────────
// The headline stat pattern from Dashboard.jsx.
// KICK-styled label, large serif number, optional subtitle.
//
// Props:
//   label    — section label (KICK-styled, uppercase)
//   value    — main numeric / text value (large serif)
//   subtitle — optional smaller line below the value
//   style    — optional additional container styles
export default function StatBox({ label, value, subtitle, style }) {
  return (
    <div style={style}>
      <div style={KICK}>{label}</div>
      <div className="num serif-num" style={{ fontFamily: SERIF, fontSize: 28, marginTop: 8 }}>
        {value}
      </div>
      {subtitle && (
        <div style={{ fontSize: 12.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
          {subtitle}
        </div>
      )}
    </div>
  )
}

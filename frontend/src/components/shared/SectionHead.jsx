import { KICK } from '../../data/tokens.js'

// ── SectionHead ───────────────────────────────────────────────
// The section header pattern (KICK-styled label used everywhere).
// Wraps children with the KICK uppercase/small-caps label style.
//
// Props:
//   children — the heading text or elements
//   style    — optional additional styles merged onto the KICK base
export default function SectionHead({ children, style }) {
  return (
    <div style={{ ...KICK, ...style }}>
      {children}
    </div>
  )
}

// ── FormGrid ──────────────────────────────────────────────────
// The 2-or-3-column field grid used in every add modal.
// Matches the `display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12` pattern
// used throughout Investments.jsx, Expenses.jsx, and SIPs.jsx.
//
// Props:
//   cols     — number of columns (default 2)
//   children — Field components / other content
export default function FormGrid({ cols = 2, children }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: `repeat(${cols}, 1fr)`, gap: 12 }}>
      {children}
    </div>
  )
}

import { Avatar } from '../Primitives.jsx'

// ── DataListItem ──────────────────────────────────────────────
// The clickable list row from Dashboard.jsx (member list, asset class list).
// Renders: [avatar?] | title / subtitle | value / percent
//
// Props:
//   avatar   — optional member object; if provided, renders <Avatar member={avatar} size={32} />
//   title    — main label / name
//   subtitle — smaller text below title (optional)
//   value    — main value on the right (string or JSX)
//   percent  — smaller percentage text below value (optional)
//   onClick  — click handler (makes the row cursor:pointer)
export default function DataListItem({ avatar, title, subtitle, value, percent, onClick }) {
  return (
    <div
      className="ed-li"
      onClick={onClick}
      style={onClick ? { cursor: 'pointer' } : undefined}
    >
      {avatar && <Avatar member={avatar} size={32} />}
      <div style={{ marginLeft: avatar ? 12 : 0 }}>
        <div style={{ fontSize: 15, fontWeight: 700 }}>{title}</div>
        {subtitle && (
          <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>{subtitle}</div>
        )}
      </div>
      <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
        <div className="num serif-num" style={{ fontSize: 22 }}>{value}</div>
        {percent && (
          <div className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>
            {percent}
          </div>
        )}
      </div>
    </div>
  )
}

// ── ColorLabel ────────────────────────────────────────────────
// The color swatch + label pattern from Dashboard/Family allocation grids.
// Renders a small square with background=color + text label.
//
// Props:
//   color  — CSS color value for the swatch
//   label  — text label next to the swatch
//   size   — swatch size in px (default 10, but uses the `.sw` CSS class)
//   style  — optional additional container styles
export default function ColorLabel({ color, label, size = 10, style }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 7, ...style }}>
      <span className="sw" style={{ background: color, width: size, height: size }} />
      <span style={{ fontSize: 11.5, fontWeight: 700, color: 'var(--ink-2)', whiteSpace: 'nowrap' }}>
        {label}
      </span>
    </div>
  )
}

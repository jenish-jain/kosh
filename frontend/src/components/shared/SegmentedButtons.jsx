// ── SegmentedButtons ─────────────────────────────────────────
// A categorical toggle/selector used in add modals across multiple screens.
// Matches the `.seg` button pattern from Investments.jsx / SIPs.jsx.
//
// Props:
//   options  — array of { value: string, label: string }
//   value    — currently selected value
//   onChange — (value: string) => void
export default function SegmentedButtons({ options, value, onChange }) {
  return (
    <div className="seg" style={{ display: 'flex' }}>
      {options.map(opt => (
        <button
          key={opt.value}
          className={value === opt.value ? 'active' : ''}
          onClick={() => onChange(opt.value)}
          style={{ flex: 1 }}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

import { fmtINR } from '../../data/format.js'
import { KICK } from '../../data/tokens.js'

// Statutory advance-tax instalment structure (Sec 211): cumulative % of the
// full-year tax payable due by each date. Month is 0-indexed (Jan=0); Jan-Mar
// dates fall in the FY's second calendar year.
const INSTALMENTS = [
  { label: '15 Jun', month: 5, day: 15, cumPct: 0.15 },
  { label: '15 Sep', month: 8, day: 15, cumPct: 0.45 },
  { label: '15 Dec', month: 11, day: 15, cumPct: 0.75 },
  { label: '15 Mar', month: 2, day: 15, cumPct: 1.00 },
]

function resolveDate(fy, month, day) {
  const year = month < 3 ? fy.start.getFullYear() + 1 : fy.start.getFullYear()
  return new Date(year, month, day)
}

export default function AdvanceTaxSchedule({ taxPayable, fy }) {
  const today = new Date()
  let prevCum = 0
  const rows = INSTALMENTS.map(inst => {
    const date = resolveDate(fy, inst.month, inst.day)
    const cumAmount = Math.round(taxPayable * inst.cumPct)
    const thisInstalment = cumAmount - prevCum
    prevCum = cumAmount
    return { ...inst, date, thisInstalment, past: date < today }
  })

  return (
    <div>
      <div style={{ ...KICK, marginBottom: 14 }}>Advance tax schedule</div>
      <div>
        {rows.map(r => (
          <div key={r.label} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 0', borderBottom: '1px solid var(--line)', opacity: r.past ? 0.5 : 1 }}>
            <div>
              <div style={{ fontSize: 13, fontWeight: 700 }}>{r.label} — {Math.round(r.cumPct * 100)}% cumulative</div>
              <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 2 }}>
                {r.past ? 'Past due date' : 'Upcoming'}
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <div className="num serif-num" style={{ fontSize: 15 }}>{fmtINR(r.thisInstalment)}</div>
              <div style={{ fontSize: 11, color: 'var(--ink-3)', fontWeight: 600 }}>this instalment</div>
            </div>
          </div>
        ))}
      </div>
      <div style={{ fontSize: 11, color: 'var(--ink-3)', fontStyle: 'italic', marginTop: 14 }}>
        Statutory instalment structure applied to your full projected annual tax payable — does not net out TDS already deducted this FY. Consult a qualified CA for your actual advance-tax liability.
      </div>
    </div>
  )
}

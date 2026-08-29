import { useState } from 'react'
import { fmtINR, fmtCompact, fmtDate } from '../../data/format.js'
import { KICK } from '../../data/tokens.js'
import { Modal } from '../../components/Primitives.jsx'
import { currentFY, fdValueAtMaturity } from './taxMath.js'

// FDs/RDs maturing inside the FY window contribute their accrued interest —
// taxable as "income from other sources", not a capital gain. Insurance
// maturity years matching the FY's spanning calendar years are bucketed as
// typically exempt under Sec 10(10D); Insurance.maturity is year-only in the
// data model, so this can't be pinned to an exact date.
export function computePotentialInflows(data, fy, today = new Date()) {
  const fixed = data.fixed || []
  const insurance = data.insurance || []

  const taxable = fixed
    .filter(f => {
      if (!f.matures) return false
      const m = new Date(f.matures)
      return m >= today && m <= fy.end
    })
    .map(f => ({
      id: f.id,
      name: f.name,
      when: fmtDate(f.matures),
      amount: Math.max(0, fdValueAtMaturity(f) - (f.principal || 0)),
    }))
    .filter(r => r.amount > 0)

  const fyYears = new Set([fy.start.getFullYear(), fy.end.getFullYear()])
  const exempt = insurance
    .filter(i => fyYears.has(i.maturity) && (i.value || 0) > 0)
    .map(i => ({ id: i.id, name: i.name, when: `Matures ${i.maturity}`, amount: i.value }))

  const totalTaxable = taxable.reduce((a, r) => a + r.amount, 0)
  const totalExempt = exempt.reduce((a, r) => a + r.amount, 0)

  return { taxable, exempt, totalTaxable, totalExempt }
}

function InflowList({ rows, emptyHint }) {
  if (rows.length === 0) return <div className="empty-hint" style={{ marginTop: 6 }}>{emptyHint}</div>
  return (
    <div style={{ marginTop: 10 }}>
      {rows.map(r => (
        <div key={r.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', padding: '9px 0', borderBottom: '1px solid var(--line)' }}>
          <div>
            <div style={{ fontSize: 13, fontWeight: 700 }}>{r.name}</div>
            <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 2 }}>{r.when}</div>
          </div>
          <div className="num serif-num" style={{ fontSize: 15 }}>{fmtINR(r.amount)}</div>
        </div>
      ))}
    </div>
  )
}

export default function PotentialInflowsTile({ data }) {
  const [open, setOpen] = useState(false)
  const fy = currentFY()
  const { taxable, exempt, totalTaxable, totalExempt } = computePotentialInflows(data, fy)
  const total = totalTaxable + totalExempt

  return (
    <>
      <div
        className="tax-tile"
        role="button"
        tabIndex={0}
        onClick={() => setOpen(true)}
        onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') setOpen(true) }}
        style={{ cursor: 'pointer' }}
      >
        <div style={KICK}>Potential inflows next FY</div>
        <div className="num serif-num" style={{ fontSize: 36, marginTop: 10 }}>{fmtCompact(total)}</div>
        <div style={{ fontSize: 12, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
          {fmtCompact(totalTaxable)} taxable · {fmtCompact(totalExempt)} typically exempt
        </div>
      </div>

      {open && (
        <Modal title={`Potential inflows — ${fy.label}`} onClose={() => setOpen(false)} width={520}>
          <div style={KICK}>Taxable — interest income, not a capital gain</div>
          <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
            FD/RD interest accrued to maturity, taxed as income from other sources at your slab rate.
          </div>
          <InflowList rows={taxable} emptyHint="No FDs or RDs maturing this financial year." />

          <div style={{ height: 1, background: 'var(--line)', margin: '20px 0' }} />

          <div style={KICK}>Typically exempt*</div>
          <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
            Insurance and other maturities, usually exempt under Sec 10(10D).
          </div>
          <InflowList rows={exempt} emptyHint="No insurance or other maturities this financial year." />

          <div style={{ fontSize: 11, color: 'var(--ink-3)', fontStyle: 'italic', marginTop: 18 }}>
            *Not a guarantee — exemption depends on premium-to-cover ratio and policy terms. Confirm against your policy document.
          </div>
        </Modal>
      )}
    </>
  )
}

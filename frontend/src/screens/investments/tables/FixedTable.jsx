import { useState } from 'react'
import { Icon } from '../../../components/Icons.jsx'
import { MemberTag, Modal } from '../../../components/Primitives.jsx'
import TotalsBar from '../../../components/shared/TotalsBar.jsx'
import { fmtINR, fmtDate } from '../../../data/format.js'
import { memberOf } from '../../../data/aggregate.js'
import { FIXED_URGENCY_DAYS } from '../../../data/constants.js'
import { daysLeft, fmtCountdown } from '../countdown.js'
import { useData } from '../../../data/context.jsx'

export default function FixedTable({ data, rows, all, showToast }) {
  const { remove } = useData()
  const [deleting, setDeleting] = useState(null)
  const inv = rows.reduce((a, x) => a + (x.principal     || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.current_value || 0), 0)
  const kindPill = { FD: 'accent', RD: 'gold' }

  const handleDelete = async (id) => {
    await remove('Fixed', id)
    setDeleting(null)
    showToast?.('Deposit removed', 'error')
  }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Deposit</th>
          {all && <th>Owner</th>}
          <th>Kind</th>
          <th className="r">Rate</th>
          <th className="r">Monthly</th>
          <th className="r">Principal</th>
          <th className="r">Matures</th>
          <th className="r">Value today</th>
          <th />
        </tr></thead>
        <tbody>
          {rows.map(r => {
            const dl     = daysLeft(r.matures)
            const urgent = dl !== null && dl >= 0 && dl < FIXED_URGENCY_DAYS
            const gain   = r.current_value - r.principal
            return (
              <tr key={r.id}>
                <td style={{ minWidth: 200 }}>
                  <div className="cell-strong" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    {r.name}
                    {r.doc_link && (
                      <a href={r.doc_link} target="_blank" rel="noopener noreferrer" title="View document"
                        style={{ color: 'var(--ink-3)', display: 'inline-flex' }}>
                        <Icon name="link" size={13} />
                      </a>
                    )}
                  </div>
                  <div className="cell-sub">{fmtDate(r.opened)}</div>
                </td>
                {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
                <td><span className={'pill ' + (kindPill[r.kind] || 'neutral')}>{r.kind}</span></td>
                <td className="r num">{r.rate}%</td>
                <td className="r num">{r.monthly ? fmtINR(r.monthly) : '—'}</td>
                <td className="r num">{fmtINR(r.principal)}</td>
                <td className="r">
                  <div className="num" style={{ fontWeight: 600, fontSize: 13 }}>{r.matures}</div>
                  <div className="cell-sub" style={{ color: urgent ? 'var(--warn)' : undefined }}>
                    {fmtCountdown(dl)}
                  </div>
                </td>
                <td className="r">
                  <div className="num cell-strong">{fmtINR(r.current_value)}</div>
                  {gain > 0 && <div className="cell-sub" style={{ color: 'var(--pos)' }}>+{fmtINR(gain)}</div>}
                </td>
                <td className="r">
                  <button className="btn ghost sm" title="Remove deposit" onClick={() => setDeleting(r.id)}>
                    <Icon name="trash" size={13} />
                  </button>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} deposit${rows.length !== 1 ? 's' : ''}`} curLabel="Value today" />

      {deleting && (
        <Modal title="Remove deposit?" onClose={() => setDeleting(null)} width={380}>
          <p style={{ fontSize: 14, color: 'var(--ink-2)', margin: '0 0 20px' }}>
            This will permanently remove the deposit — use this if it's been redeemed or broken.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
            <button className="btn" onClick={() => setDeleting(null)}>Cancel</button>
            <button className="btn danger" onClick={() => handleDelete(deleting)}>Remove</button>
          </div>
        </Modal>
      )}
    </div>
  )
}

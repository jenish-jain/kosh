import { EditCell, MemberTag } from '../../../components/Primitives.jsx'
import TotalsBar from '../../../components/shared/TotalsBar.jsx'
import { Icon } from '../../../components/Icons.jsx'
import { fmtINR, fmtCompact, fmtDate } from '../../../data/format.js'
import { memberOf } from '../../../data/aggregate.js'
import { nextPremiumDue } from '../../../data/schedule.js'
import { INSURANCE_URGENCY_DAYS } from '../../../data/constants.js'
import { daysLeft, fmtCountdown } from '../countdown.js'

export default function InsuranceTable({ data, rows, all, dirty, setDirty }) {
  const inv = rows.reduce((a, x) => a + (x.paid  || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.value || 0), 0)
  const mark = (id, field, val) => setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))
  const freqLabel = { annual: '/yr', monthly: '/mo', single: 'one-time' }
  const typePill  = { Term: 'neutral', ULIP: 'accent', Endowment: 'gold', Income: 'silver' }

  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Plan</th>
          {all && <th>Owner</th>}
          <th>Type</th>
          <th className="r">Premium</th>
          <th className="r">Next due</th>
          <th className="r">Paid so far</th>
          <th className="r">Cover</th>
          <th className="r">Value</th>
          <th className="r">Matures</th>
        </tr></thead>
        <tbody>
          {rows.map(r => {
            const due    = nextPremiumDue(r)
            const dl     = due ? daysLeft(due.toISOString().slice(0, 10)) : null
            const urgent = dl !== null && dl >= 0 && dl < INSURANCE_URGENCY_DAYS
            return (
              <tr key={r.id}>
                <td style={{ minWidth: 180 }}>
                  <div className="cell-strong" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    {r.name}
                    {r.doc_link && (
                      <a href={r.doc_link} target="_blank" rel="noopener noreferrer" title="View document"
                        style={{ color: 'var(--ink-3)', display: 'inline-flex' }}>
                        <Icon name="link" size={13} />
                      </a>
                    )}
                  </div>
                </td>
                {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
                <td><span className={'pill ' + (typePill[r.type] || 'neutral')}>{r.type}</span></td>
                <td className="r">
                  <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end', alignItems: 'baseline' }}>
                    <EditCell value={dirty[r.id]?.premium ?? r.premium} type="number" align="right"
                      format={v => fmtINR(v)} onChange={v => mark(r.id, 'premium', v)} />
                    <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--ink-3)' }}>{freqLabel[r.freq]}</span>
                  </div>
                </td>
                <td className="r">
                  {due ? (
                    <>
                      <div className="num" style={{ fontWeight: 600, fontSize: 13 }}>{fmtDate(due)}</div>
                      <div className="cell-sub" style={{ color: urgent ? 'var(--warn)' : undefined }}>{fmtCountdown(dl)}</div>
                    </>
                  ) : <span className="faint">—</span>}
                </td>
                <td className="r">
                  <EditCell value={dirty[r.id]?.paid ?? r.paid} type="number" align="right"
                    format={v => fmtINR(v)} onChange={v => mark(r.id, 'paid', v)} />
                </td>
                <td className="r num">{r.cover ? fmtCompact(r.cover) : '—'}</td>
                <td className="r">
                  <EditCell value={dirty[r.id]?.value ?? r.value} type="number" align="right"
                    format={v => v ? fmtINR(v) : '—'} onChange={v => mark(r.id, 'value', v)} />
                </td>
                <td className="r num faint">{r.maturity}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} plan${rows.length !== 1 ? 's' : ''}`} curLabel="Total value" />
    </div>
  )
}

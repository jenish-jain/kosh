import { EditCell, MemberTag } from '../../../components/Primitives.jsx'
import TotalsBar from '../../../components/shared/TotalsBar.jsx'
import { fmtINR, fmtCompact } from '../../../data/format.js'
import { memberOf } from '../../../data/aggregate.js'
import { AC_COLOR, AC_LABEL } from '../../../data/constants.js'

export default function NPSTable({ data, rows, all, dirty, setDirty }) {
  const inv = rows.reduce((a, x) => a + (x.invested || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.units || 0) * (x.nav || 0), 0)

  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Scheme</th>
          {all && <th>Owner</th>}
          <th>Tier</th>
          <th>Class</th>
          <th className="r">Units</th>
          <th className="r">NAV</th>
          <th className="r">Invested</th>
          <th className="r">Value</th>
        </tr></thead>
        <tbody>
          {rows.map(r => {
            const curVal = (r.units || 0) * (r.nav || 0)
            const gain   = curVal - (r.invested || 0)
            const k      = r.id
            return (
              <tr key={k}>
                <td style={{ minWidth: 220 }}>
                  <div className="cell-strong">{r.scheme || r.fund_manager}</div>
                  {r.scheme && r.fund_manager && <div className="cell-sub">{r.fund_manager}</div>}
                  {r.pran && <div className="cell-sub">PRAN {r.pran}</div>}
                </td>
                {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
                <td><span className="pill neutral">{r.tier || '—'}</span></td>
                <td>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5, fontSize: 12.5, fontWeight: 700 }}>
                    <span style={{ width: 8, height: 8, borderRadius: '50%', background: AC_COLOR[r.asset_class] || 'var(--line)', flexShrink: 0 }} />
                    {r.asset_class} · {AC_LABEL[r.asset_class] || r.asset_class}
                  </span>
                </td>
                <td className="r">
                  <EditCell value={dirty[k]?.units ?? r.units} type="number"
                    onChange={v => setDirty(d => ({ ...d, [k]: { ...r, ...(d[k] || {}), units: Number(v) } }))} />
                </td>
                <td className="r">
                  <EditCell value={dirty[k]?.nav ?? r.nav} type="number"
                    onChange={v => setDirty(d => ({ ...d, [k]: { ...r, ...(d[k] || {}), nav: Number(v) } }))} />
                </td>
                <td className="r">
                  <EditCell value={dirty[k]?.invested ?? r.invested} type="number"
                    onChange={v => setDirty(d => ({ ...d, [k]: { ...r, ...(d[k] || {}), invested: Number(v) } }))} />
                </td>
                <td className="r">
                  <div className="num cell-strong">{fmtINR(curVal)}</div>
                  {Math.abs(gain) > 0 && (
                    <div className="cell-sub" style={{ color: gain >= 0 ? 'var(--pos)' : 'var(--neg)' }}>
                      {gain >= 0 ? '+' : '−'}{fmtCompact(Math.abs(gain))}
                    </div>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} scheme${rows.length !== 1 ? 's' : ''}`} curLabel="Current value" />
    </div>
  )
}

import { classTotals } from '../../data/aggregate.js'
import { fmtCompact } from '../../data/format.js'
import { KICK } from '../../data/tokens.js'

export default function SummaryTiles({ data, memberId, tab, setTab }) {
  const c = classTotals(data, memberId)
  const tiles = [
    { k: 'mf',        label: 'Mutual funds',   inv: c.mf.inv,        cur: c.mf.cur },
    { k: 'stocks',    label: 'Stocks',          inv: c.stocks.inv,    cur: c.stocks.cur },
    { k: 'nps',       label: 'NPS',             inv: c.nps.inv,       cur: c.nps.cur },
    { k: 'fixed',     label: 'FD / RD',         inv: c.fixed.inv,     cur: c.fixed.cur },
    { k: 'metals',    label: 'Gold & silver',   inv: c.metals.inv,    cur: c.metals.cur },
    { k: 'insurance', label: 'Insurance',       inv: c.insurance.inv, cur: c.insurance.cur },
  ]
  return (
    <div style={{ display: 'grid', gridTemplateColumns: `repeat(${tiles.length},1fr)`, gap: 12, marginBottom: 26 }}>
      {tiles.map((t, i) => (
        <div key={t.k} onClick={() => setTab(t.k)}
          style={{ paddingLeft: i ? 16 : 0, borderLeft: i ? '1px solid var(--line)' : 'none', cursor: 'pointer' }}>
          <div style={KICK}>{t.label}</div>
          <div className="num serif-num" style={{ fontSize: 22, marginTop: 8, color: tab === t.k ? 'var(--ink)' : 'var(--ink-2)' }}>
            {t.cur > 0 ? fmtCompact(t.cur) : t.inv > 0 ? fmtCompact(t.inv) : '—'}
          </div>
          {t.inv > 0 && (
            <div className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 2 }}>
              invested {fmtCompact(t.inv)}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

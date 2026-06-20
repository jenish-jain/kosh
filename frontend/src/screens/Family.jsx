import { classTotals, memberTotal, holdingsFor } from '../data/aggregate.js'
import { fmtINR, fmtCompact } from '../data/format.js'
import { ED_ORDER, ED_COL, ED_LABEL } from '../data/constants.js'
import { todayDisplay } from '../data/schedule.js'
import { KICK, SERIF } from '../data/tokens.js'
import { EdStack, EdRule, Avatar } from '../components/Primitives.jsx'
import { Icon } from '../components/Icons.jsx'

// ── Family overview ──────────────────────────────────────────
function FamilyOverview({ data, onSelect }) {
  const household = memberTotal(data, null)
  const hc = classTotals(data, null)
  const hgain = hc.total.cur - hc.total.inv
  const memSegs = (data.members || []).map(m => ({ value: memberTotal(data, m.id), color: m.color, label: m.name }))

  return (
    <div className="fade-in">
      <div className="stmt-band">
        <div style={{ ...KICK, letterSpacing: '.18em' }}>Family portfolios</div>
        <div className="stmt-meta">{(data.members || []).length} members · As on {todayDisplay()}</div>
      </div>
      <EdRule thick />

      {/* Hero */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 40, flexWrap: 'wrap' }}>
        <div>
          <div style={KICK}>Household net worth</div>
          <div className="num hero-num" style={{ fontSize: 64, marginTop: 12 }}>{fmtINR(household)}</div>
        </div>
        <div style={{ marginLeft: 'auto', textAlign: 'right', paddingBottom: 6 }}>
          <div style={KICK}>Invested</div>
          <div className="num col-num" style={{ marginTop: 8 }}>{fmtINR(hc.total.inv)}</div>
          <div style={{ fontSize: 12.5, color: 'var(--pos)', fontWeight: 700, marginTop: 8 }}>+{fmtCompact(hgain)} since cost</div>
        </div>
      </div>

      <div style={{ marginTop: 24 }}>
        <div style={{ ...KICK, marginBottom: 12 }}>Share by member</div>
        <EdStack segs={memSegs} h={13} />
        <div style={{ display: 'grid', gridTemplateColumns: `repeat(${Math.max(memSegs.length, 1)},1fr)`, gap: 18, marginTop: 18 }}>
          {(data.members || []).map(m => {
            const v = memberTotal(data, m.id)
            return (
              <div key={m.id}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
                  <span className="sw" style={{ background: m.color }} />
                  <span style={{ fontSize: 11.5, fontWeight: 700, color: 'var(--ink-2)' }}>{m.name}</span>
                </div>
                <div className="num serif-num" style={{ fontSize: 22, marginTop: 8 }}>{fmtCompact(v)}</div>
                <div className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 1 }}>
                  {Math.round(v / household * 100)}% share
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <EdRule />

      {/* Member statement rows */}
      <div style={{ ...KICK, marginBottom: 4 }}>Member statements</div>
      {(data.members || []).map(m => {
        const c = classTotals(data, m.id)
        const gain = c.total.cur - c.total.inv
        const sip = (data.sips || []).filter(s => s.member === m.id && s.status === 'active').reduce((a, s) => a + (s.amount || 0), 0)
        const segs = ED_ORDER.map(k => ({ value: c[k].cur, color: ED_COL[k], label: ED_LABEL[k] })).filter(s => s.value > 0)
        const h = holdingsFor(data, m.id)
        const count = h.mf.length + h.stocks.length + h.fixed.length + h.metals.length + h.insurance.length
        return (
          <div key={m.id} onClick={() => onSelect(m.id)}
            style={{ display: 'grid', gridTemplateColumns: 'minmax(180px,1.1fr) 1.4fr minmax(150px,auto)', gap: 28, alignItems: 'center', padding: '20px 0', borderBottom: '1px solid var(--line)', cursor: 'pointer' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 13 }}>
              <Avatar member={m} size={40} />
              <div>
                <div style={{ fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}>{m.full_name || m.name}</div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, whiteSpace: 'nowrap' }}>{m.relation} · slab {m.slab}%</div>
              </div>
            </div>
            <div>
              {segs.length > 0 && <EdStack segs={segs} h={9} />}
              <div style={{ display: 'flex', gap: 18, marginTop: 10 }}>
                <div><span style={KICK}>SIP/mo </span><span className="num serif-num" style={{ fontSize: 14 }}>{sip ? fmtCompact(sip) : '—'}</span></div>
                <div><span style={KICK}>Holdings </span><span className="num serif-num" style={{ fontSize: 14 }}>{count}</span></div>
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <div className="num serif-num" style={{ fontSize: 28 }}>{fmtINR(c.total.cur)}</div>
              <div style={{ fontSize: 12, color: 'var(--pos)', fontWeight: 700, marginTop: 3 }}>+{fmtCompact(gain)} · view ›</div>
            </div>
          </div>
        )
      })}
    </div>
  )
}

// ── Member profile ───────────────────────────────────────────
function MemberProfile({ data, memberId, setScreen }) {
  const m = (data.members || []).find(x => x.id === memberId)
  if (!m) return <div className="empty-hint">Member not found.</div>
  const c = classTotals(data, memberId)
  const gain = c.total.cur - c.total.inv
  const h = holdingsFor(data, memberId)
  const sips = (data.sips || []).filter(s => s.member === memberId && s.status === 'active')
  const sipMonthly = sips.reduce((a, s) => a + (s.amount || 0), 0)
  const segs = ED_ORDER.map(k => ({ k, color: ED_COL[k], label: ED_LABEL[k], value: c[k].cur })).filter(s => s.value > 0)

  return (
    <div className="fade-in">
      <div className="stmt-band">
        <div style={{ ...KICK, letterSpacing: '.18em' }}>Member statement</div>
        <div className="stmt-meta">{m.relation} · slab {m.slab}% · As on {todayDisplay()}</div>
      </div>
      <EdRule thick />

      {/* Hero */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 20, flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <Avatar member={m} size={52} />
          <div>
            <div style={KICK}>Net worth</div>
            <div className="num hero-num" style={{ fontSize: 60, marginTop: 6 }}>{fmtINR(c.total.cur)}</div>
          </div>
        </div>
        <div style={{ marginLeft: 'auto', textAlign: 'right', paddingBottom: 6 }}>
          <div style={KICK}>Invested</div>
          <div className="num col-num" style={{ marginTop: 8 }}>{fmtINR(c.total.inv)}</div>
          <div style={{ fontSize: 12.5, color: 'var(--pos)', fontWeight: 700, marginTop: 8 }}>+{fmtCompact(gain)} since cost</div>
        </div>
      </div>

      <EdRule />

      {/* Allocation */}
      <div style={{ ...KICK, marginBottom: 14 }}>Allocation · {segs.length} asset class{segs.length !== 1 ? 'es' : ''}</div>
      {segs.length > 0 && <EdStack segs={segs} h={13} />}
      <div style={{ display: 'grid', gridTemplateColumns: `repeat(${Math.max(segs.length, 1)},1fr)`, gap: 18, marginTop: 18 }}>
        {segs.map(s => (
          <div key={s.k}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
              <span className="sw" style={{ background: s.color }} />
              <span style={{ fontSize: 11.5, fontWeight: 700, color: 'var(--ink-2)', whiteSpace: 'nowrap' }}>{s.label}</span>
            </div>
            <div className="num serif-num" style={{ fontSize: 21, marginTop: 8 }}>{fmtCompact(s.value)}</div>
            <div className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 1 }}>
              {Math.round(s.value / c.total.cur * 100)}%
            </div>
          </div>
        ))}
      </div>

      <EdRule />

      {/* Two columns: SIPs + top holdings */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1px 1.2fr', gap: 38 }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'baseline', marginBottom: 4 }}>
            <div style={KICK}>Active SIPs</div>
            <span className="num serif-num" style={{ marginLeft: 'auto', fontSize: 15 }}>
              {sipMonthly ? fmtINR(sipMonthly) + '/mo' : '—'}
            </span>
          </div>
          {sips.length
            ? sips.sort((a, b) => a.day - b.day).map(s => (
                <div key={s.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderBottom: '1px solid var(--line)' }}>
                  <div className="num serif-num" style={{ width: 26, fontSize: 17 }}>{s.day}</div>
                  <div style={{ flex: 1, fontSize: 13.5, fontWeight: 600 }}>{s.fund}</div>
                  <div className="num serif-num" style={{ fontSize: 16 }}>{fmtINR(s.amount)}</div>
                </div>
              ))
            : <div className="empty-hint">No active SIPs.</div>}
        </div>

        <div style={{ background: 'var(--line)' }} />

        <div>
          <div style={{ display: 'flex', alignItems: 'baseline', marginBottom: 4 }}>
            <div style={KICK}>Top holdings</div>
            <button className="btn ghost sm" style={{ marginLeft: 'auto' }} onClick={() => setScreen('investments')}>All ›</button>
          </div>
          {[...h.mf].sort((a, b) => (b.current || 0) - (a.current || 0)).slice(0, 5).map(f => (
            <div key={f.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderBottom: '1px solid var(--line)' }}>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{f.name}</div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>{f.platform}</div>
              </div>
              <div className="num serif-num" style={{ fontSize: 16 }}>{fmtINR(f.current)}</div>
            </div>
          ))}
          {h.mf.length === 0 && <div className="empty-hint">No mutual funds.</div>}
        </div>
      </div>
    </div>
  )
}

// ── Main ─────────────────────────────────────────────────────
export default function Family({ data, memberId, setScreen, onSelect }) {
  return memberId
    ? <MemberProfile data={data} memberId={memberId} setScreen={setScreen} />
    : <FamilyOverview data={data} onSelect={onSelect} />
}

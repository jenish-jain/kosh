import { classTotals, memberTotal, scope } from '../data/aggregate.js'
import { fmtINR, fmtCompact, fmtDate } from '../data/format.js'
import { ED_ORDER, ED_COL, ED_LABEL, OUTFLOW_COLOR, OUTFLOW_LABEL } from '../data/constants.js'
import { todayDisplay, todayDay, upcomingOutflows } from '../data/schedule.js'
import { KICK, SERIF } from '../data/tokens.js'
import { AreaChart, EdStack, EdRule, GainPill } from '../components/Primitives.jsx'
import { Avatar } from '../components/Primitives.jsx'

export default function Dashboard({ data, memberId, setScreen, onSelectMember }) {
  const c = classTotals(data, memberId)
  const total = c.total.cur, invested = c.total.inv, gain = total - invested
  const debt = c.liabilities.cur
  const netW = total - debt
  const segs = ED_ORDER
    .map(k => ({ k, label: ED_LABEL[k], value: c[k].cur, color: ED_COL[k] }))
    .filter(s => s.value > 0)

  const history = data.history || []
  const monthDelta = history.length >= 2
    ? history[history.length - 1].value - history[history.length - 2].value
    : 0

  const activeSips = (data.sips || []).filter(s => s.status === 'active' && (!memberId || s.member === memberId))
  const sipMonthly = activeSips.reduce((a, s) => a + (s.amount || 0), 0)
  const cover = (data.insurance || [])
    .filter(x => !memberId || x.member === memberId)
    .reduce((a, x) => a + (x.cover || 0), 0)
  const planCount = (data.insurance || []).filter(x => !memberId || x.member === memberId).length

  // Find next SIP day
  const _todayDay = todayDay()
  const nextSip = [...activeSips].sort((a, b) => {
    const da = a.day >= _todayDay ? a.day - _todayDay : a.day + 30 - _todayDay
    const db = b.day >= _todayDay ? b.day - _todayDay : b.day + 30 - _todayDay
    return da - db
  })[0]

  // Recurring outflows (SIP debits, insurance premiums, loan EMIs) due in the next 30 days
  const upcoming = upcomingOutflows(data, memberId, 30)
  const upcomingTotal = upcoming.reduce((a, x) => a + (x.amount || 0), 0)

  return (
    <div className="fade-in">
      {/* Statement header */}
      <div className="stmt-band">
        <div style={{ ...KICK, letterSpacing: '.18em', whiteSpace: 'nowrap' }}>Statement of net worth</div>
        <div className="stmt-meta">{scope(data, memberId)} · As on {todayDisplay()}</div>
      </div>
      <EdRule thick />

      {/* Hero */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 40, flexWrap: 'wrap' }}>
        <div>
          <div style={KICK}>Total net worth</div>
          <div className="num hero-num" style={{ marginTop: 14 }}>{fmtINR(netW)}</div>
          {debt > 0 && (
            <div style={{ fontSize: 12.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 8 }}>
              {fmtCompact(total)} assets · −{fmtCompact(debt)} debt
            </div>
          )}
        </div>
        <div style={{ marginLeft: 'auto', textAlign: 'right', paddingBottom: 6 }}>
          <div style={KICK}>Invested</div>
          <div className="num col-num" style={{ marginTop: 8 }}>{fmtINR(invested)}</div>
          <div style={{ fontSize: 12.5, color: 'var(--pos)', fontWeight: 700, marginTop: 8 }}>
            +{fmtCompact(gain)} since cost
            {monthDelta !== 0 && (
              <> · {monthDelta >= 0 ? '▲' : '▼'} {fmtCompact(Math.abs(monthDelta))} this month</>
            )}
          </div>
        </div>
      </div>

      {/* Sparkline */}
      {history.length > 1 && (
        <div style={{ marginTop: 22, opacity: .9 }}>
          <AreaChart data={history} height={100} />
        </div>
      )}

      <EdRule />

      {/* Allocation */}
      <div style={{ ...KICK, marginBottom: 14 }}>Allocation · {segs.length} asset class{segs.length !== 1 ? 'es' : ''}</div>
      <EdStack segs={segs} h={13} />
      <div className="alloc-grid" style={{ display: 'grid', gridTemplateColumns: `repeat(${Math.max(segs.length, 1)}, 1fr)`, gap: 18, marginTop: 18 }}>
        {segs.map(s => (
          <div key={s.k}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
              <span className="sw" style={{ background: s.color }} />
              <span style={{ fontSize: 11.5, fontWeight: 700, color: 'var(--ink-2)', whiteSpace: 'nowrap' }}>{s.label}</span>
            </div>
            <div className="num serif-num" style={{ fontSize: 22, marginTop: 8 }}>{fmtCompact(s.value)}</div>
            <div className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 1 }}>
              {Math.round(s.value / total * 100)}% of total
            </div>
          </div>
        ))}
      </div>

      <EdRule />

      {/* Two-column lower */}
      <div className="dash-lower" style={{ display: 'grid', gridTemplateColumns: '1.25fr 1px 1fr', gap: 38 }}>
        {/* Left: members or asset classes */}
        <div>
          <div style={{ ...KICK, marginBottom: 4 }}>
            {memberId ? 'Holdings by asset class' : 'By family member'}
          </div>
          {memberId
            ? segs.map(s => (
                <div key={s.k} className="ed-li">
                  <span className="sw" style={{ background: s.color, marginRight: 12 }} />
                  <span style={{ fontSize: 14.5, fontWeight: 600, whiteSpace: 'nowrap' }}>{s.label}</span>
                  <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
                    <span className="num serif-num" style={{ fontSize: 20 }}>{fmtCompact(s.value)}</span>
                    <span className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600, marginLeft: 10 }}>
                      {Math.round(s.value / total * 100)}%
                    </span>
                  </div>
                </div>
              ))
            : (data.members || []).map(m => {
                const v = memberTotal(data, m.id)
                return (
                  <div key={m.id} className="ed-li" onClick={() => onSelectMember(m.id)}
                    style={{ cursor: 'pointer' }}>
                    <Avatar member={m} size={32} />
                    <div style={{ marginLeft: 12 }}>
                      <div style={{ fontSize: 15, fontWeight: 700 }}>{(m.full_name || m.name).replace(' (You)', '')}</div>
                      <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>{m.relation}</div>
                    </div>
                    <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
                      <div className="num serif-num" style={{ fontSize: 22 }}>{fmtCompact(v)}</div>
                      <div className="num" style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>
                        {Math.round(v / total * 100)}% share
                      </div>
                    </div>
                  </div>
                )
              })}
        </div>

        {/* Divider */}
        <div className="dash-divider" style={{ background: 'var(--line)' }} />

        {/* Right: SIP + protection */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 22 }}>
          <div onClick={() => setScreen('sips')} style={{ cursor: 'pointer' }}>
            <div style={KICK}>Monthly commitment</div>
            <div className="num col-num" style={{ marginTop: 8 }}>{fmtINR(sipMonthly)}</div>
            <div style={{ fontSize: 12.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
              {activeSips.length} active SIP{activeSips.length !== 1 ? 's' : ''}
              {nextSip ? ` · next on ${nextSip.day} ${['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][new Date().getMonth()]}` : ''}
            </div>
          </div>
          {cover > 0 && (
            <div>
              <div style={KICK}>Protection in force</div>
              <div className="num col-num" style={{ marginTop: 8 }}>{fmtINR(cover)}</div>
              <div style={{ fontSize: 12.5, color: 'var(--ink-3)', fontWeight: 600, marginTop: 4 }}>
                sum assured across {planCount} plan{planCount !== 1 ? 's' : ''}
              </div>
            </div>
          )}
        </div>
      </div>

      {upcoming.length > 0 && (
        <>
          <EdRule />
          <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 14 }}>
            <div style={KICK}>Expected outflows · next 30 days</div>
            <div className="num serif-num" style={{ fontSize: 17 }}>{fmtINR(upcomingTotal)}</div>
          </div>
          {upcoming.map(x => (
            <div key={`${x.kind}-${x.id}`} className="ed-li">
              <span className="sw" style={{ background: OUTFLOW_COLOR[x.kind], marginRight: 12 }} />
              <div>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{x.label}</div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>
                  {OUTFLOW_LABEL[x.kind]} · {fmtDate(x.date)}
                </div>
              </div>
              <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
                <span className="num serif-num" style={{ fontSize: 17 }}>{fmtINR(x.amount)}</span>
              </div>
            </div>
          ))}
        </>
      )}

      <EdRule />
      <div style={{ fontFamily: SERIF, fontSize: 13.5, fontStyle: 'italic', color: 'var(--ink-3)' }}>
        Figures maintained locally and entered by hand — a quiet ledger, not a live feed.
      </div>
      <a
        href="https://github.com/jenish-jain/kosh" target="_blank" rel="noopener noreferrer"
        style={{ ...KICK, display: 'inline-block', marginTop: 10, textDecoration: 'none' }}
      >
        View source & self-host →
      </a>
    </div>
  )
}

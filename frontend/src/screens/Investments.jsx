import { useState, useRef } from 'react'
import { useData } from '../data/context.jsx'
import { holdingsFor, classTotals, fmtINR, fmtCompact, fmtDate, ED_COL, ED_LABEL, TODAY_DISPLAY } from '../data/helpers.js'
import { EdRule, Modal, Field, EditCell, MemberTag, SaveBar } from '../components/Primitives.jsx'
import { Icon } from '../components/Icons.jsx'

const KICK = { textTransform: 'uppercase', letterSpacing: '.16em', fontWeight: 700, fontSize: 10.5, color: 'var(--ink-3)' }
const SERIF = "var(--serif)"

function scope(data, memberId) {
  if (!memberId) return 'Whole family'
  const m = data.members?.find(m => m.id === memberId)
  return m ? (m.full_name || m.name).replace(' (You)', '') : '—'
}

function memberOf(data, id) {
  return (data.members || []).find(m => m.id === id)
}

// ── Summary tiles ────────────────────────────────────────────
function SummaryTiles({ data, memberId, activeTab, setTab }) {
  const c = classTotals(data, memberId)
  const tiles = [
    { k: 'mf',        label: 'Mutual funds',   inv: c.mf.inv,        cur: c.mf.cur },
    { k: 'stocks',    label: 'Stocks',          inv: c.stocks.inv,    cur: c.stocks.cur },
    { k: 'metals',    label: 'Gold & silver',   inv: c.metals.inv,    cur: c.metals.cur },
    { k: 'insurance', label: 'Insurance',       inv: c.insurance.inv, cur: c.insurance.cur },
  ]
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 18, marginBottom: 26 }}>
      {tiles.map((t, i) => (
        <div key={t.k} onClick={() => setTab(t.k)}
          style={{ paddingLeft: i ? 24 : 0, borderLeft: i ? '1px solid var(--line)' : 'none', cursor: 'pointer' }}>
          <div style={KICK}>{t.label}</div>
          <div className="num serif-num" style={{ fontSize: 28, marginTop: 8, color: activeTab === t.k ? 'var(--ink)' : 'var(--ink-2)' }}>
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

// ── Totals bar ───────────────────────────────────────────────
function TotalsBar({ inv, cur, label, curLabel = 'Current value' }) {
  return (
    <div className="totals-bar">
      <span style={{ fontWeight: 700, fontSize: 13.5 }}>{label}</span>
      <div style={{ flex: 1 }} />
      <div style={{ textAlign: 'right' }}>
        <div style={KICK}>Total invested</div>
        <div className="num" style={{ fontFamily: SERIF, fontWeight: 500, fontSize: 19, marginTop: 2 }}>{fmtINR(inv)}</div>
      </div>
      <div style={{ textAlign: 'right' }}>
        <div style={KICK}>{curLabel}</div>
        <div className="num" style={{ fontFamily: SERIF, fontWeight: 500, fontSize: 19, marginTop: 2 }}>{cur > 0 ? fmtINR(cur) : '—'}</div>
      </div>
    </div>
  )
}

// ── MF Table ─────────────────────────────────────────────────
function MFTable({ data, rows, all, dirty, setDirty }) {
  const inv = rows.reduce((a, x) => a + (x.invested || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.current || 0), 0)
  const mark = (id, field, val) => {
    setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))
  }
  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Fund</th>
          {all && <th>Owner</th>}
          <th className="r">Monthly SIP</th>
          <th className="r">Invested</th>
          <th className="r">Current value</th>
        </tr></thead>
        <tbody>
          {rows.map(r => (
            <tr key={r.id}>
              <td style={{ minWidth: 240 }}>
                <div className="cell-strong">{r.name}</div>
                <div className="cell-sub">{r.plan} · {r.platform}</div>
              </td>
              {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
              <td className="r">
                <EditCell value={dirty[r.id]?.sip ?? r.sip} type="number" align="right"
                  format={v => v ? fmtINR(v) : '—'}
                  onChange={v => mark(r.id, 'sip', v)} />
              </td>
              <td className="r">
                <EditCell value={dirty[r.id]?.invested ?? r.invested} type="number" align="right"
                  format={v => fmtINR(v)}
                  onChange={v => mark(r.id, 'invested', v)} />
              </td>
              <td className="r">
                <EditCell value={dirty[r.id]?.current ?? r.current} type="number" align="right"
                  format={v => v ? fmtINR(v) : '—'}
                  onChange={v => mark(r.id, 'current', v)} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} fund${rows.length !== 1 ? 's' : ''}`} />
    </div>
  )
}

// ── Stocks Table ─────────────────────────────────────────────
function StockTable({ data, rows, all, dirty, setDirty }) {
  const inv = rows.reduce((a, x) => a + (x.qty || 0) * (x.avg_price || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.qty || 0) * (x.last_price || 0), 0)
  const mark = (id, field, val) => setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))
  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Stock</th>
          {all && <th>Owner</th>}
          <th className="r">Qty</th>
          <th className="r">Avg price</th>
          <th className="r">Last price</th>
          <th className="r">Invested</th>
          <th className="r">Current value</th>
        </tr></thead>
        <tbody>
          {rows.map(r => {
            const dR = dirty[r.id] || {}
            const qty = dR.qty ?? r.qty
            const avg = dR.avg_price ?? r.avg_price
            const ltp = dR.last_price ?? r.last_price
            return (
              <tr key={r.id}>
                <td>
                  <div className="cell-strong">{r.name}</div>
                  <div className="cell-sub">{r.ticker}</div>
                </td>
                {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
                <td className="r"><EditCell value={qty} type="number" align="right" onChange={v => mark(r.id, 'qty', v)} /></td>
                <td className="r"><EditCell value={avg} type="number" align="right" format={v => fmtINR(v)} onChange={v => mark(r.id, 'avg_price', v)} /></td>
                <td className="r"><EditCell value={ltp} type="number" align="right" format={v => fmtINR(v)} onChange={v => mark(r.id, 'last_price', v)} /></td>
                <td className="r num">{fmtINR(qty * avg)}</td>
                <td className="r num cell-strong">{fmtINR(qty * ltp)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} stock${rows.length !== 1 ? 's' : ''}`} />
    </div>
  )
}

// ── Metals Table ─────────────────────────────────────────────
function MetalTable({ data, rows, all, dirty, setDirty }) {
  const inv = rows.reduce((a, x) => a + (x.grams || 0) * (x.buy_rate || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.grams || 0) * (x.today_price || 0), 0)
  const mark = (id, field, val) => setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))
  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Metal</th>
          {all && <th>Owner</th>}
          <th className="r">Grams</th>
          <th className="r">Buy rate /g</th>
          <th className="r">Today /g</th>
          <th className="r">Invested</th>
          <th className="r">Current value</th>
        </tr></thead>
        <tbody>
          {rows.map(r => {
            const dR = dirty[r.id] || {}
            const grams = dR.grams ?? r.grams
            const buy = dR.buy_rate ?? r.buy_rate
            const today = dR.today_price ?? r.today_price
            return (
              <tr key={r.id}>
                <td>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
                    <span className={'pill ' + (r.type === 'Gold' ? 'gold' : 'silver')}>{r.type}</span>
                  </div>
                  <div className="cell-sub">{fmtDate(r.date_purchased)} · {r.place}</div>
                </td>
                {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
                <td className="r"><EditCell value={grams} type="number" align="right" format={v => v + ' g'} onChange={v => mark(r.id, 'grams', v)} /></td>
                <td className="r"><EditCell value={buy} type="number" align="right" format={v => fmtINR(v)} onChange={v => mark(r.id, 'buy_rate', v)} /></td>
                <td className="r"><EditCell value={today} type="number" align="right" format={v => fmtINR(v)} onChange={v => mark(r.id, 'today_price', v)} /></td>
                <td className="r num">{fmtINR(grams * buy)}</td>
                <td className="r num cell-strong">{fmtINR(grams * today)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} holding${rows.length !== 1 ? 's' : ''}`} />
    </div>
  )
}

// ── Insurance Table ──────────────────────────────────────────
function InsuranceTable({ data, rows, all, dirty, setDirty }) {
  const inv = rows.reduce((a, x) => a + (x.paid || 0), 0)
  const cur = rows.reduce((a, x) => a + (x.value || 0), 0)
  const mark = (id, field, val) => setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))
  const freqLabel = { annual: '/yr', monthly: '/mo', single: 'one-time' }
  const typePill = { Term: 'neutral', ULIP: 'accent', Endowment: 'gold', Income: 'silver' }
  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="tbl">
        <thead><tr>
          <th>Plan</th>
          {all && <th>Owner</th>}
          <th>Type</th>
          <th className="r">Premium</th>
          <th className="r">Paid so far</th>
          <th className="r">Cover</th>
          <th className="r">Value</th>
          <th className="r">Matures</th>
        </tr></thead>
        <tbody>
          {rows.map(r => (
            <tr key={r.id}>
              <td style={{ minWidth: 180 }}><div className="cell-strong">{r.name}</div></td>
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
          ))}
        </tbody>
      </table>
      <TotalsBar inv={inv} cur={cur} label={`${rows.length} plan${rows.length !== 1 ? 's' : ''}`} curLabel="Total value" />
    </div>
  )
}

// ── Add Modal ────────────────────────────────────────────────
function AddModal({ tab, memberId, data, onClose }) {
  const { add } = useData()
  const [form, setForm] = useState({ member: memberId || (data.members?.[0]?.id || 'you'), freq: 'annual', metalType: 'Gold', insType: 'Endowment' })
  const up = (k, v) => setForm(f => ({ ...f, [k]: v }))
  const n = v => Number(v) || 0

  const submit = async () => {
    const id = tab + Date.now()
    if (tab === 'mf') {
      await add('MF', { id, name: form.name || 'New Fund', plan: 'Direct · Growth', platform: form.platform || '—', member: form.member, invested: n(form.invested), current: n(form.current) || n(form.invested), sip: n(form.sip), notes: form.notes || '' })
    } else if (tab === 'stocks') {
      await add('Stocks', { id, name: form.name || 'New Stock', ticker: (form.ticker || form.name || 'NEW').toUpperCase().slice(0, 8), qty: n(form.qty), avg_price: n(form.avg), last_price: n(form.ltp) || n(form.avg), member: form.member })
    } else if (tab === 'metals') {
      await add('Metals', { id, type: form.metalType, date_purchased: new Date().toISOString().slice(0, 10), grams: n(form.grams), buy_rate: n(form.buyRate), today_price: n(form.buyRate), place: form.place || '—', member: form.member })
    } else {
      await add('Insurance', { id, name: form.name || 'New plan', type: form.insType, member: form.member, premium: n(form.premium), freq: form.freq, paid: n(form.paid), value: n(form.value), cover: n(form.cover), maturity: n(form.maturity) || 2040 })
    }
    onClose()
  }

  const memberPicker = (
    <Field label="Owner">
      <div className="seg" style={{ display: 'flex' }}>
        {(data.members || []).map(m => (
          <button key={m.id} className={form.member === m.id ? 'active' : ''} onClick={() => up('member', m.id)} style={{ flex: 1 }}>{m.name}</button>
        ))}
      </div>
    </Field>
  )

  const titles = { mf: 'Add mutual fund', stocks: 'Add stock', metals: 'Add metal', insurance: 'Add insurance / plan' }

  return (
    <Modal title={titles[tab]} onClose={onClose}>
      {tab === 'mf' && <>
        <Field label="Fund name"><input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. Quant Small Cap Fund" /></Field>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Invested so far"><input className="input" type="number" value={form.invested || ''} onChange={e => up('invested', e.target.value)} /></Field>
          <Field label="Current value"><input className="input" type="number" value={form.current || ''} onChange={e => up('current', e.target.value)} /></Field>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Monthly SIP (0 if lumpsum)"><input className="input" type="number" value={form.sip || ''} onChange={e => up('sip', e.target.value)} /></Field>
          <Field label="Platform"><input className="input" value={form.platform || ''} onChange={e => up('platform', e.target.value)} placeholder="e.g. Groww" /></Field>
        </div>
      </>}

      {tab === 'stocks' && <>
        <Field label="Company"><input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. Reliance Industries" /></Field>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Ticker"><input className="input" value={form.ticker || ''} onChange={e => up('ticker', e.target.value)} placeholder="e.g. RELIANCE" /></Field>
          <Field label="Qty"><input className="input" type="number" value={form.qty || ''} onChange={e => up('qty', e.target.value)} /></Field>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Avg price"><input className="input" type="number" value={form.avg || ''} onChange={e => up('avg', e.target.value)} /></Field>
          <Field label="Last price"><input className="input" type="number" value={form.ltp || ''} onChange={e => up('ltp', e.target.value)} /></Field>
        </div>
      </>}

      {tab === 'metals' && <>
        <Field label="Metal">
          <div className="seg" style={{ display: 'flex' }}>
            {['Gold', 'Silver'].map(x => <button key={x} className={form.metalType === x ? 'active' : ''} onClick={() => up('metalType', x)} style={{ flex: 1 }}>{x}</button>)}
          </div>
        </Field>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Grams"><input className="input" type="number" value={form.grams || ''} onChange={e => up('grams', e.target.value)} /></Field>
          <Field label="Buy rate / g"><input className="input" type="number" value={form.buyRate || ''} onChange={e => up('buyRate', e.target.value)} /></Field>
        </div>
        <Field label="Place of purchase"><input className="input" value={form.place || ''} onChange={e => up('place', e.target.value)} placeholder="e.g. Kalamandir" /></Field>
      </>}

      {tab === 'insurance' && <>
        <Field label="Plan name"><input className="input" value={form.name || ''} onChange={e => up('name', e.target.value)} placeholder="e.g. LIC Jeevan Anand" /></Field>
        <Field label="Type">
          <div className="seg" style={{ display: 'flex', flexWrap: 'wrap' }}>
            {['Endowment', 'Term', 'ULIP', 'Income'].map(x => <button key={x} className={form.insType === x ? 'active' : ''} onClick={() => up('insType', x)} style={{ flex: 1 }}>{x}</button>)}
          </div>
        </Field>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Premium"><input className="input" type="number" value={form.premium || ''} onChange={e => up('premium', e.target.value)} /></Field>
          <Field label="Frequency">
            <select className="input" value={form.freq} onChange={e => up('freq', e.target.value)}>
              <option value="annual">Annual</option><option value="monthly">Monthly</option><option value="single">Single</option>
            </select>
          </Field>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Paid so far"><input className="input" type="number" value={form.paid || ''} onChange={e => up('paid', e.target.value)} /></Field>
          <Field label="Sum assured / cover"><input className="input" type="number" value={form.cover || ''} onChange={e => up('cover', e.target.value)} /></Field>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Current / fund value"><input className="input" type="number" value={form.value || ''} onChange={e => up('value', e.target.value)} placeholder="optional" /></Field>
          <Field label="Maturity year"><input className="input" type="number" value={form.maturity || ''} onChange={e => up('maturity', e.target.value)} placeholder="e.g. 2040" /></Field>
        </div>
      </>}

      {memberPicker}
      <div style={{ display: 'flex', gap: 10, marginTop: 8, justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={submit}>Add holding</button>
      </div>
    </Modal>
  )
}

// ── Main screen ──────────────────────────────────────────────
export default function Investments({ data, memberId, showToast }) {
  const { update } = useData()
  const [tab, setTab] = useState('mf')
  const [showAdd, setShowAdd] = useState(false)
  const [dirty, setDirty] = useState({})
  const [saving, setSaving] = useState(false)

  const h = holdingsFor(data, memberId)
  const all = !memberId
  const tabs = [
    ['mf', 'Mutual Funds', h.mf.length],
    ['stocks', 'Stocks', h.stocks.length],
    ['metals', 'Gold & Silver', h.metals.length],
    ['insurance', 'Insurance & Plans', h.insurance.length],
  ]

  // sheet name map
  const sheetMap = { mf: 'MF', stocks: 'Stocks', metals: 'Metals', insurance: 'Insurance' }

  const hasDirty = Object.keys(dirty).length > 0

  const save = async () => {
    setSaving(true)
    try {
      const promises = Object.entries(dirty).map(([id, patch]) =>
        update(sheetMap[tab], id, patch)
      )
      await Promise.all(promises)
      setDirty({})
      showToast('Saved to Google Sheet ✓')
    } catch (e) {
      showToast('Save failed — check connection', 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fade-in">
      {/* Header */}
      <div className="stmt-band">
        <div style={{ ...KICK, letterSpacing: '.18em' }}>Schedule of holdings</div>
        <div className="stmt-meta">{scope(data, memberId)} · As on {TODAY_DISPLAY}</div>
      </div>
      <EdRule thick />
      <SummaryTiles data={data} memberId={memberId} activeTab={tab} setTab={t => { setTab(t); setDirty({}) }} />
      <EdRule />

      {/* Tab bar + actions */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 18, flexWrap: 'wrap' }}>
        <div className="seg">
          {tabs.map(([k, label, n]) => (
            <button key={k} className={tab === k ? 'active' : ''} onClick={() => { setTab(k); setDirty({}) }}>
              {label} <span style={{ opacity: .55, fontWeight: 700 }}>{n}</span>
            </button>
          ))}
        </div>
        <div style={{ flex: 1 }} />
        <button className="btn primary" onClick={() => setShowAdd(true)}><Icon name="plus" size={15} /> Add</button>
      </div>

      {/* Tables */}
      {tab === 'mf'        && <MFTable       data={data} rows={h.mf}        all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'stocks'    && <StockTable    data={data} rows={h.stocks}    all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'metals'    && <MetalTable    data={data} rows={h.metals}    all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'insurance' && <InsuranceTable data={data} rows={h.insurance} all={all} dirty={dirty} setDirty={setDirty} />}

      <SaveBar dirty={hasDirty} saving={saving} onSave={save} />

      {showAdd && <AddModal tab={tab} memberId={memberId} data={data} onClose={() => setShowAdd(false)} />}
    </div>
  )
}

import { useState } from 'react'
import { useData } from '../data/context.jsx'
import { fmtINR, fmtCompact, fmtDate, TODAY_DISPLAY, TODAY_DAY, TODAY_MONTH, TODAY_YEAR } from '../data/helpers.js'
import { EdRule, Modal, Field, MemberTag, SaveBar } from '../components/Primitives.jsx'
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

// ── Monthly calendar ─────────────────────────────────────────
function SipCalendar({ sips }) {
  const first = new Date(TODAY_YEAR, TODAY_MONTH, 1).getDay()
  const days = new Date(TODAY_YEAR, TODAY_MONTH + 1, 0).getDate()
  const byDay = {}
  sips.forEach(s => { (byDay[s.day] = byDay[s.day] || []).push(s) })
  const dows = ['S', 'M', 'T', 'W', 'T', 'F', 'S']
  const monthNames = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']
  return (
    <div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)' }}>
        {dows.map((d, i) => (
          <div key={'h' + i} style={{ textAlign: 'center', fontSize: 10, fontWeight: 700, letterSpacing: '.1em', color: 'var(--ink-3)', padding: '0 0 8px' }}>{d}</div>
        ))}
        {Array.from({ length: first }, (_, i) => (
          <div key={'e' + i} style={{ borderTop: '1px solid var(--line-2)' }} />
        ))}
        {Array.from({ length: days }, (_, i) => {
          const d = i + 1
          const hits = byDay[d] || []
          const sum = hits.reduce((a, s) => a + (s.amount || 0), 0)
          const isToday = d === TODAY_DAY
          return (
            <div key={d} style={{
              minHeight: 60, padding: '6px 7px',
              borderTop: '1px solid var(--line-2)',
              background: hits.length ? 'var(--accent-soft)' : 'transparent',
            }}>
              <div className="num serif-num" style={{
                fontSize: 15, color: hits.length ? 'var(--accent-ink)' : 'var(--ink-3)',
                display: 'inline-block', lineHeight: 1,
                ...(isToday ? { borderBottom: '2px solid var(--accent)' } : {})
              }}>{d}</div>
              {hits.length > 0 && (
                <div style={{ marginTop: 4 }}>
                  <div className="num serif-num" style={{ fontSize: 13, color: 'var(--accent-ink)', whiteSpace: 'nowrap' }}>{fmtCompact(sum)}</div>
                  <div style={{ fontSize: 9, fontWeight: 700, letterSpacing: '.04em', textTransform: 'uppercase', color: 'var(--accent)', whiteSpace: 'nowrap' }}>
                    {hits.length} SIP{hits.length > 1 ? 's' : ''}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ── Edit SIP modal ───────────────────────────────────────────
function EditSipModal({ sip, onClose, onSave }) {
  const [amount, setAmount] = useState(sip.amount)
  const [day, setDay] = useState(sip.day)
  const [status, setStatus] = useState(sip.status)
  return (
    <Modal title="Edit SIP" onClose={onClose}>
      <div style={{ fontWeight: 700, fontSize: 15, marginBottom: 4 }}>{sip.fund}</div>
      <div style={{ fontSize: 12.5, color: 'var(--ink-3)', marginBottom: 16 }}>{sip.platform} · started {fmtDate(sip.start_date)}</div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="Monthly amount"><input className="input" type="number" value={amount} onChange={e => setAmount(Number(e.target.value))} /></Field>
        <Field label="Debit day (1–28)"><input className="input" type="number" min="1" max="28" value={day} onChange={e => setDay(Number(e.target.value))} /></Field>
      </div>
      <div style={{ display: 'flex', gap: 10, marginTop: 8, justifyContent: 'space-between' }}>
        <button className="btn" onClick={() => {
          setStatus(s => s === 'active' ? 'paused' : 'active')
        }}>
          {status === 'active' ? <><Icon name="pause" size={14} /> Pause SIP</> : <><Icon name="play" size={14} /> Resume</>}
        </button>
        <div style={{ display: 'flex', gap: 10 }}>
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn primary" onClick={() => onSave({ ...sip, amount, day, status })}>Save</button>
        </div>
      </div>
    </Modal>
  )
}

// ── Add SIP modal ────────────────────────────────────────────
function AddSipModal({ memberId, data, onClose }) {
  const { add } = useData()
  const mfOptions = (data.mf || []).filter(f => !memberId || f.member === memberId)
  const [useExisting, setUseExisting] = useState(mfOptions.length > 0)
  const [fundId, setFundId] = useState(mfOptions[0]?.id || '')
  const [fundName, setFundName] = useState('')
  const [member, setMember] = useState(memberId || data.members?.[0]?.id || 'you')
  const [amount, setAmount] = useState('')
  const [day, setDay] = useState(1)
  const [platform, setPlatform] = useState('')
  const [startDate, setStartDate] = useState(new Date().toISOString().slice(0, 10))

  const resolvedFund = useExisting
    ? (data.mf || []).find(x => x.id === fundId)
    : null

  const submit = async () => {
    const name = useExisting ? (resolvedFund?.name || fundId) : fundName
    const owner = useExisting ? (resolvedFund?.member || member) : member
    if (!name || !amount || !day) return
    await add('SIPs', {
      id: 'sip' + Date.now(),
      fund: name,
      member: owner,
      amount: Number(amount),
      day: Number(day),
      status: 'active',
      start_date: startDate,
      platform: useExisting ? (resolvedFund?.platform || platform) : platform,
    })
    onClose()
  }

  return (
    <Modal title="Add SIP" onClose={onClose}>
      <Field label="Fund">
        <div className="seg" style={{ display: 'flex', marginBottom: 10 }}>
          <button className={useExisting ? 'active' : ''} onClick={() => setUseExisting(true)} style={{ flex: 1 }} disabled={mfOptions.length === 0}>Pick from my MFs</button>
          <button className={!useExisting ? 'active' : ''} onClick={() => setUseExisting(false)} style={{ flex: 1 }}>Enter manually</button>
        </div>
        {useExisting ? (
          <select className="input" value={fundId} onChange={e => setFundId(e.target.value)}>
            {mfOptions.map(f => {
              const m = (data.members || []).find(m => m.id === f.member)
              return <option key={f.id} value={f.id}>{f.name}{m ? ` — ${m.name}` : ''}</option>
            })}
          </select>
        ) : (
          <input className="input" value={fundName} onChange={e => setFundName(e.target.value)} placeholder="e.g. Quant Small Cap Fund" />
        )}
      </Field>

      {!useExisting && (
        <Field label="Owner">
          <div className="seg" style={{ display: 'flex' }}>
            {(data.members || []).map(m => (
              <button key={m.id} className={member === m.id ? 'active' : ''} onClick={() => setMember(m.id)} style={{ flex: 1 }}>{m.name}</button>
            ))}
          </div>
        </Field>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="Monthly amount"><input className="input" type="number" value={amount} onChange={e => setAmount(e.target.value)} placeholder="e.g. 5000" /></Field>
        <Field label="Debit day (1–28)"><input className="input" type="number" min="1" max="28" value={day} onChange={e => setDay(e.target.value)} /></Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="Platform"><input className="input" value={platform} onChange={e => setPlatform(e.target.value)} placeholder={useExisting ? resolvedFund?.platform || 'e.g. Groww' : 'e.g. Groww'} /></Field>
        <Field label="Start date"><input className="input" type="date" value={startDate} onChange={e => setStartDate(e.target.value)} /></Field>
      </div>
      <div style={{ display: 'flex', gap: 10, marginTop: 8, justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={submit}>Add SIP</button>
      </div>
    </Modal>
  )
}

// ── Lumpsum modal ────────────────────────────────────────────
function LumpsumModal({ memberId, data, onClose }) {
  const { add } = useData()
  const mfOptions = (data.mf || []).filter(f => !memberId || f.member === memberId)
  const [fundId, setFundId] = useState(mfOptions[0]?.id || '')
  const [amount, setAmount] = useState('')

  const submit = async () => {
    if (!fundId || !amount) return
    const f = (data.mf || []).find(x => x.id === fundId)
    await add('Lumpsums', {
      id: 'lp' + Date.now(),
      fund: f?.name || fundId,
      member: f?.member || memberId || 'you',
      amount: Number(amount),
      date: new Date().toISOString().slice(0, 10),
    })
    onClose()
  }

  return (
    <Modal title="Add lumpsum top-up" onClose={onClose}>
      <Field label="Into fund">
        <select className="input" value={fundId} onChange={e => setFundId(e.target.value)}>
          {mfOptions.map(f => {
            const m = memberOf(data, f.member)
            return <option key={f.id} value={f.id}>{f.name}{m ? ` — ${m.name}` : ''}</option>
          })}
        </select>
      </Field>
      <Field label="Amount"><input className="input" type="number" value={amount} onChange={e => setAmount(e.target.value)} placeholder="e.g. 50000" /></Field>
      <p style={{ fontSize: 12.5, color: 'var(--ink-3)', lineHeight: 1.5, marginBottom: 16 }}>
        A one-off investment outside the monthly cycle. Will be recorded in the Lumpsums sheet.
      </p>
      <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={submit}>Add top-up</button>
      </div>
    </Modal>
  )
}

// ── Main screen ──────────────────────────────────────────────
export default function SIPs({ data, memberId, showToast }) {
  const { update } = useData()
  const [editSip, setEditSip] = useState(null)
  const [showLump, setShowLump] = useState(false)
  const [showAdd, setShowAdd] = useState(false)
  const [dirty, setDirty] = useState({}) // id → full updated sip object
  const [saving, setSaving] = useState(false)

  const monthNames = ['January','February','March','April','May','June','July','August','September','October','November','December']
  const sips = (data.sips || []).filter(s => !memberId || s.member === memberId)
  const active = sips.filter(s => s.status === 'active')
  const monthly = active.reduce((a, s) => a + (s.amount || 0), 0)
  const annual = monthly * 12
  const lumpsums = (data.lumpsums || []).filter(l => !memberId || l.member === memberId)
  const lumpsTotal = lumpsums.reduce((a, l) => a + (l.amount || 0), 0)
  const sorted = [...sips].sort((a, b) => a.status === b.status ? a.day - b.day : a.status === 'active' ? -1 : 1)

  const hasDirty = Object.keys(dirty).length > 0

  const handleEditSave = (updated) => {
    setDirty(d => ({ ...d, [updated.id]: updated }))
    setEditSip(null)
  }

  const save = async () => {
    setSaving(true)
    try {
      await Promise.all(Object.values(dirty).map(sip => update('SIPs', sip.id, sip)))
      setDirty({})
      showToast('Saved to Google Sheet ✓')
    } catch {
      showToast('Save failed — check connection', 'error')
    } finally {
      setSaving(false)
    }
  }

  const figs = [
    { label: 'Monthly outflow', value: fmtINR(monthly) },
    { label: 'Active SIPs', value: `${active.length} / ${sips.length}` },
    { label: 'Annual commitment', value: fmtCompact(annual) },
    { label: 'Lumpsums added', value: fmtCompact(lumpsTotal) },
  ]

  return (
    <div className="fade-in">
      <div className="stmt-band">
        <div style={{ ...KICK, letterSpacing: '.18em' }}>SIP schedule &amp; commitments</div>
        <div className="stmt-meta">{scope(data, memberId)} · As on {TODAY_DISPLAY}</div>
      </div>
      <EdRule thick />

      {/* Headline figures */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr) auto', gap: 18, alignItems: 'center' }}>
        {figs.map((f, i) => (
          <div key={i} style={{ paddingLeft: i ? 24 : 0, borderLeft: i ? '1px solid var(--line)' : 'none' }}>
            <div style={KICK}>{f.label}</div>
            <div className="num serif-num" style={{ fontSize: 30, marginTop: 8 }}>{f.value}</div>
          </div>
        ))}
        <div style={{ paddingLeft: 24, borderLeft: '1px solid var(--line)', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <button className="btn primary" onClick={() => setShowAdd(true)}><Icon name="plus" size={15} /> Add SIP</button>
          <button className="btn" onClick={() => setShowLump(true)}><Icon name="plus" size={15} /> Lumpsum top-up</button>
        </div>
      </div>

      <EdRule />

      {/* Two-column: manage + calendar */}
      <div style={{ display: 'grid', gridTemplateColumns: '1.45fr 1px 0.95fr', gap: 36, alignItems: 'start' }}>
        <div>
          <div style={{ ...KICK, marginBottom: 4 }}>Manage SIPs</div>
          <table className="tbl">
            <thead><tr>
              <th>Fund</th>
              {!memberId && <th>Owner</th>}
              <th className="r">Amount</th>
              <th className="r">Day</th>
              <th>Status</th>
              <th />
            </tr></thead>
            <tbody>
              {sorted.map(s => {
                const disp = dirty[s.id] || s
                const m = memberOf(data, s.member)
                return (
                  <tr key={s.id} style={{ opacity: disp.status === 'paused' ? .55 : 1 }}>
                    <td>
                      <div className="cell-strong" style={{ minWidth: 140 }}>{s.fund}</div>
                      <div className="cell-sub">{s.platform}</div>
                    </td>
                    {!memberId && <td><MemberTag member={m} /></td>}
                    <td className="r num">{disp.amount ? fmtINR(disp.amount) : '—'}</td>
                    <td className="r num">{disp.day}</td>
                    <td>
                      {disp.status === 'active'
                        ? <span className="pill pos">Active</span>
                        : <span className="pill neutral">Paused</span>}
                    </td>
                    <td className="r">
                      <button className="btn ghost sm" onClick={() => setEditSip(disp)} style={{ padding: '5px 8px' }}>
                        <Icon name="edit" size={14} />
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          <SaveBar dirty={hasDirty} saving={saving} onSave={save} />
        </div>

        <div style={{ background: 'var(--line)', alignSelf: 'stretch' }} />

        <div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
            <div style={KICK}>{monthNames[TODAY_MONTH]} {TODAY_YEAR}</div>
            <span className="tag" style={{ marginLeft: 'auto' }}>
              <span className="dot" style={{ background: 'var(--accent)' }} /> SIP debit
            </span>
          </div>
          <SipCalendar sips={active} />

          <div style={{ ...KICK, margin: '28px 0 4px' }}>Recent lumpsum top-ups</div>
          {lumpsums.slice(0, 5).map(l => {
            const m = memberOf(data, l.member)
            return (
              <div key={l.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '12px 0', borderBottom: '1px solid var(--line)' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13.5, fontWeight: 600 }}>{l.fund}</div>
                  <div style={{ fontSize: 11.5, color: 'var(--ink-3)', fontWeight: 600 }}>{fmtDate(l.date)}{m ? ` · ${m.name}` : ''}</div>
                </div>
                <div className="num serif-num" style={{ fontSize: 16 }}>{fmtINR(l.amount)}</div>
              </div>
            )
          })}
          {lumpsums.length === 0 && <div className="empty-hint">No lumpsum top-ups yet.</div>}
        </div>
      </div>

      {editSip && <EditSipModal sip={editSip} onClose={() => setEditSip(null)} onSave={handleEditSave} />}
      {showLump && <LumpsumModal memberId={memberId} data={data} onClose={() => setShowLump(false)} />}
      {showAdd  && <AddSipModal  memberId={memberId} data={data} onClose={() => setShowAdd(false)} />}
    </div>
  )
}

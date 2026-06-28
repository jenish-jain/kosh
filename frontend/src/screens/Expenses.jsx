import { useState } from 'react'
import { useData } from '../data/context.jsx'
import { holdingsFor, classTotals, scope, memberOf } from '../data/aggregate.js'
import { fmtINR, fmtCompact, fmtDate } from '../data/format.js'
import { todayStr, todayDisplay, nextEmiDue } from '../data/schedule.js'
import { KICK, SERIF } from '../data/tokens.js'
import { EdRule, Modal, Field, EditCell, MemberTag, SaveBar } from '../components/Primitives.jsx'
import { Icon } from '../components/Icons.jsx'

function daysLeft(dateStr) {
  if (!dateStr) return null
  return Math.ceil((new Date(dateStr) - new Date()) / 864e5)
}

function fmtCountdown(days) {
  if (days === null) return '—'
  if (days < 0) return 'Closed'
  if (days === 0) return 'Today'
  if (days < 30) return `${days}d left`
  if (days < 365) return `${Math.round(days / 30)}mo left`
  const y = Math.floor(days / 365)
  const m = Math.round((days % 365) / 30)
  return m > 0 ? `${y}y ${m}mo left` : `${y}y left`
}

function computeEnds(startedStr, tenureMonths) {
  if (!tenureMonths) return ''
  const d = new Date(startedStr || todayStr())
  d.setMonth(d.getMonth() + Math.round(tenureMonths))
  return d.toISOString().slice(0, 10)
}

// ── Totals bar ───────────────────────────────────────────────
function TotalsBar({ principal, outstanding, label }) {
  return (
    <div className="totals-bar">
      <span style={{ fontWeight: 700, fontSize: 13.5 }}>{label}</span>
      <div style={{ flex: 1 }} />
      <div style={{ textAlign: 'right' }}>
        <div style={KICK}>Total borrowed</div>
        <div className="num" style={{ fontFamily: SERIF, fontWeight: 500, fontSize: 19, marginTop: 2 }}>{fmtINR(principal)}</div>
      </div>
      <div style={{ width: 1, alignSelf: 'stretch', background: 'var(--line)', margin: '0 22px' }} />
      <div style={{ textAlign: 'right' }}>
        <div style={KICK}>Outstanding</div>
        <div className="num" style={{ fontFamily: SERIF, fontWeight: 700, fontSize: 19, marginTop: 2, color: 'var(--neg)' }}>{fmtINR(outstanding)}</div>
      </div>
    </div>
  )
}

// ── Loan Table ───────────────────────────────────────────────
function LoanTable({ data, rows, all, dirty, setDirty }) {
  const principal = rows.reduce((a, x) => a + (x.principal || 0), 0)
  const outstanding = rows.reduce((a, x) => a + (dirty[x.id]?.outstanding ?? x.outstanding ?? 0), 0)
  const mark = (id, field, val) => setDirty(d => ({ ...d, [id]: { ...(d[id] || {}), [field]: val } }))
  const typePill = { Home: 'accent', Car: 'gold', Personal: 'neutral', Education: 'silver', Other: 'neutral' }

  return (
    <>
      <div className="tbl-scroll">
        <table className="tbl" style={{ minWidth: 640 }}>
          <thead><tr>
            <th>Loan</th>
            {all && <th>Owner</th>}
            <th>Type</th>
            <th className="r">Rate</th>
            <th className="r">EMI</th>
            <th className="r">Outstanding</th>
            <th className="r">Ends</th>
          </tr></thead>
          <tbody>
            {rows.map(r => {
              const ends = computeEnds(r.started, r.tenure_months)
              const dl = daysLeft(ends)
              const urgent = dl !== null && dl >= 0 && dl < 90
              const due = nextEmiDue(r)
              return (
                <tr key={r.id}>
                  <td style={{ minWidth: 200 }}>
                    <div className="cell-strong">{r.lender}</div>
                    <div className="cell-sub">{r.started ? fmtDate(r.started) : ''}</div>
                  </td>
                  {all && <td><MemberTag member={memberOf(data, r.member)} /></td>}
                  <td><span className={'pill ' + (typePill[r.type] || 'neutral')}>{r.type}</span></td>
                  <td className="r num">{r.rate ? `${r.rate}%` : '—'}</td>
                  <td className="r">
                    <div className="num cell-strong">{r.emi ? fmtINR(r.emi) : '—'}</div>
                    {due && <div className="cell-sub">next {fmtDate(due)}</div>}
                  </td>
                  <td className="r">
                    <EditCell value={dirty[r.id]?.outstanding ?? r.outstanding} type="number" align="right"
                      format={v => fmtINR(v)} onChange={v => mark(r.id, 'outstanding', v)} />
                  </td>
                  <td className="r">
                    {ends ? (
                      <>
                        <div className="num" style={{ fontWeight: 600, fontSize: 13 }}>{fmtDate(ends)}</div>
                        <div className="cell-sub" style={{ color: urgent ? 'var(--warn)' : undefined }}>{fmtCountdown(dl)}</div>
                      </>
                    ) : <span className="faint">—</span>}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <TotalsBar principal={principal} outstanding={outstanding} label={`${rows.length} loan${rows.length !== 1 ? 's' : ''}`} />
    </>
  )
}

// ── Add Modal ────────────────────────────────────────────────
function AddLoanModal({ memberId, data, onClose }) {
  const { add } = useData()
  const defaults = { member: memberId || (data.members?.[0]?.id || 'you'), type: 'Home', started: todayStr() }
  const [form, setForm] = useState(defaults)
  const up = (k, v) => setForm(f => ({ ...f, [k]: v }))
  const n = v => Number(v) || 0

  const submit = async () => {
    const id = 'loan' + Date.now()
    await add('Loans', {
      id,
      lender: form.lender || 'New loan',
      type: form.type,
      member: form.member,
      principal: n(form.principal),
      outstanding: n(form.outstanding) || n(form.principal),
      rate: n(form.rate),
      emi: n(form.emi),
      emi_day: n(form.emiDay) || 1,
      started: form.started || todayStr(),
      tenure_months: n(form.tenure),
    })
    onClose()
  }

  return (
    <Modal title="Add loan" onClose={onClose}>
      <Field label="Lender"><input className="input" value={form.lender || ''} onChange={e => up('lender', e.target.value)} placeholder="e.g. HDFC Home Loan" /></Field>
      <Field label="Type">
        <div className="seg" style={{ display: 'flex', flexWrap: 'wrap' }}>
          {['Home', 'Car', 'Personal', 'Education', 'Other'].map(x => (
            <button key={x} className={form.type === x ? 'active' : ''} onClick={() => up('type', x)} style={{ flex: 1 }}>{x}</button>
          ))}
        </div>
      </Field>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="Principal (original amount)"><input className="input" type="number" value={form.principal || ''} onChange={e => up('principal', e.target.value)} /></Field>
        <Field label="Outstanding balance"><input className="input" type="number" value={form.outstanding || ''} onChange={e => up('outstanding', e.target.value)} placeholder="defaults to principal" /></Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="Interest rate (% p.a.)"><input className="input" type="number" step="0.01" value={form.rate || ''} onChange={e => up('rate', e.target.value)} placeholder="e.g. 8.5" /></Field>
        <Field label="EMI amount"><input className="input" type="number" value={form.emi || ''} onChange={e => up('emi', e.target.value)} /></Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="EMI day of month"><input className="input" type="number" min="1" max="31" value={form.emiDay || ''} onChange={e => up('emiDay', e.target.value)} placeholder="e.g. 5" /></Field>
        <Field label="Tenure (months)"><input className="input" type="number" value={form.tenure || ''} onChange={e => up('tenure', e.target.value)} placeholder="e.g. 240" /></Field>
      </div>
      <Field label="Started"><input className="input" type="date" value={form.started || todayStr()} onChange={e => up('started', e.target.value)} /></Field>
      {(form.tenure > 0) && (
        <div style={{ fontSize: 12, color: 'var(--ink-3)', padding: '8px 12px', background: 'var(--bg-2,#f7f5f0)', borderRadius: 4, lineHeight: 1.6 }}>
          Ends: <strong>{computeEnds(form.started || todayStr(), n(form.tenure))}</strong>
        </div>
      )}
      <Field label="Owner">
        <div className="seg" style={{ display: 'flex' }}>
          {(data.members || []).map(m => (
            <button key={m.id} className={form.member === m.id ? 'active' : ''} onClick={() => up('member', m.id)} style={{ flex: 1 }}>{m.name}</button>
          ))}
        </div>
      </Field>
      <div style={{ display: 'flex', gap: 10, marginTop: 8, justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={submit}>Add loan</button>
      </div>
    </Modal>
  )
}

// ── Main screen ──────────────────────────────────────────────
export default function Expenses({ data, memberId, showToast }) {
  const { update } = useData()
  const [showAdd, setShowAdd] = useState(false)
  const [dirty, setDirty] = useState({})
  const [saving, setSaving] = useState(false)

  const h = holdingsFor(data, memberId)
  const all = !memberId
  const c = classTotals(data, memberId)
  const hasDirty = Object.keys(dirty).length > 0

  const save = async () => {
    setSaving(true)
    try {
      await Promise.all(Object.entries(dirty).map(([id, patch]) => update('Loans', id, patch)))
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
        <div style={{ ...KICK, letterSpacing: '.18em' }}>Schedule of liabilities</div>
        <div className="stmt-meta">{scope(data, memberId)} · As on {todayDisplay()}</div>
      </div>
      <EdRule thick />

      {/* Summary tiles */}
      <div style={{ display: 'flex', gap: 40, marginBottom: 26 }}>
        <div>
          <div style={KICK}>Outstanding debt</div>
          <div className="num serif-num" style={{ fontSize: 28, marginTop: 8 }}>
            {c.liabilities.cur > 0 ? fmtCompact(c.liabilities.cur) : '—'}
          </div>
        </div>
        <div style={{ paddingLeft: 24, borderLeft: '1px solid var(--line)' }}>
          <div style={KICK}>Monthly EMIs</div>
          <div className="num serif-num" style={{ fontSize: 28, marginTop: 8 }}>
            {c.liabilities.monthly > 0 ? fmtINR(c.liabilities.monthly) : '—'}
          </div>
        </div>
      </div>
      <EdRule />

      {/* Actions */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 18 }}>
        <div style={{ ...KICK, marginBottom: 0 }}>Loans · {h.loans.length}</div>
        <div style={{ flex: 1 }} />
        <button className="btn primary" onClick={() => setShowAdd(true)}>
          <Icon name="plus" size={15} /> Add loan
        </button>
      </div>

      {h.loans.length > 0 ? (
        <LoanTable data={data} rows={h.loans} all={all} dirty={dirty} setDirty={setDirty} />
      ) : (
        <div style={{ fontFamily: SERIF, fontSize: 13.5, fontStyle: 'italic', color: 'var(--ink-3)', padding: '24px 0' }}>
          No loans recorded yet — add one to track EMIs and outstanding balances here.
        </div>
      )}

      <SaveBar dirty={hasDirty} saving={saving} onSave={save} />

      {showAdd && <AddLoanModal memberId={memberId} data={data} onClose={() => setShowAdd(false)} />}
    </div>
  )
}

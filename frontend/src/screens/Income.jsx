import { useState } from 'react'
import { useData } from '../data/context.jsx'
import { fmtINR } from '../data/format.js'
import { KICK } from '../data/tokens.js'
import { Modal, Field, EdRule } from '../components/Primitives.jsx'
import { Icon } from '../components/Icons.jsx'
import UploadZone from '../components/UploadZone.jsx'
import { getDriveAccessToken } from '../data/driveAuth.js'

const TYPES = ['salary', 'freelance', 'rental', 'other']

function currentPeriod() {
  const d = new Date()
  return d.toLocaleString('en-IN', { month: 'short', year: 'numeric' })
}

// Sort income rows newest-first by parsing "MMM YYYY"
function sortedIncome(rows) {
  return [...rows].sort((a, b) => {
    const ta = new Date(a.period + ' 01') || 0
    const tb = new Date(b.period + ' 01') || 0
    return tb - ta
  })
}

// Latest period totals
function latestTotals(rows) {
  if (!rows.length) return { period: null, gross: 0, net: 0 }
  const sorted = sortedIncome(rows)
  const period = sorted[0].period
  let gross = 0, net = 0
  rows.forEach(r => { if (r.period === period) { gross += r.gross; net += r.net } })
  return { period, gross, net }
}

// ── Income form modal ──────────────────────────────────────────
function IncomeModal({ initial, onClose, onSave, title }) {
  const [form, setForm] = useState({
    source: '', type: 'salary', period: currentPeriod(),
    gross: '', net: '', pf_deduction: '', tax_deduction: '',
    other_deductions: '', notes: '', ...initial,
  })
  const [showUpload, setShowUpload] = useState(false)
  const [saving, setSaving] = useState(false)

  const set = (k, v) => setForm(f => ({ ...f, [k]: v }))
  const num = k => Number(form[k]) || 0

  const handleExtracted = (fields) => {
    setShowUpload(false)
    setForm(f => ({
      ...f,
      source:           fields.source           || f.source,
      type:             fields.type             || f.type,
      period:           fields.period           || f.period,
      gross:            fields.gross            ?? f.gross,
      net:              fields.net              ?? f.net,
      pf_deduction:     fields.pf_deduction     ?? f.pf_deduction,
      tax_deduction:    fields.tax_deduction     ?? f.tax_deduction,
      other_deductions: fields.other_deductions ?? f.other_deductions,
      notes:            fields.notes            || f.notes,
    }))
  }

  const submit = async () => {
    if (!form.source || !form.period || !form.gross) return
    setSaving(true)
    await onSave({
      ...form,
      gross:            num('gross'),
      net:              num('net'),
      pf_deduction:     num('pf_deduction'),
      tax_deduction:    num('tax_deduction'),
      other_deductions: num('other_deductions'),
    })
    setSaving(false)
    onClose()
  }

  return (
    <>
      <Modal title={title} onClose={onClose} width={520}>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
          <button
            className="btn sm"
            onClick={() => { getDriveAccessToken().catch(() => {}); setShowUpload(true) }}
          >
            <Icon name="upload" size={14} /> Upload payslip
          </button>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Source / Employer">
            <input className="input" value={form.source} onChange={e => set('source', e.target.value)} placeholder="e.g. Accenture" />
          </Field>
          <Field label="Type">
            <select className="input" value={form.type} onChange={e => set('type', e.target.value)}>
              {TYPES.map(t => <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>)}
            </select>
          </Field>
        </div>

        <Field label="Period (e.g. Jun 2026)">
          <input className="input" value={form.period} onChange={e => set('period', e.target.value)} placeholder="Jun 2026" />
        </Field>

        <EdRule />

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginTop: 4 }}>
          <Field label="Gross salary (₹)">
            <input className="input" type="number" value={form.gross} onChange={e => set('gross', e.target.value)} placeholder="200000" />
          </Field>
          <Field label="Net take-home (₹)">
            <input className="input" type="number" value={form.net} onChange={e => set('net', e.target.value)} placeholder="152000" />
          </Field>
        </div>

        <div style={{ marginTop: 4 }}>
          <div style={{ ...KICK, marginBottom: 8 }}>Deductions (optional)</div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
            <Field label="PF / EPF (₹)">
              <input className="input" type="number" value={form.pf_deduction} onChange={e => set('pf_deduction', e.target.value)} placeholder="0" />
            </Field>
            <Field label="TDS / Tax (₹)">
              <input className="input" type="number" value={form.tax_deduction} onChange={e => set('tax_deduction', e.target.value)} placeholder="0" />
            </Field>
            <Field label="Other (₹)">
              <input className="input" type="number" value={form.other_deductions} onChange={e => set('other_deductions', e.target.value)} placeholder="0" />
            </Field>
          </div>
        </div>

        <Field label="Notes (optional)">
          <input className="input" value={form.notes} onChange={e => set('notes', e.target.value)} placeholder="Bonus, arrears, variable pay…" />
        </Field>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn primary" onClick={submit} disabled={saving || !form.source || !form.period || !form.gross}>
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </Modal>

      {showUpload && (
        <UploadZone docType="income" onExtracted={handleExtracted} onClose={() => setShowUpload(false)} />
      )}
    </>
  )
}

// ── Main screen ────────────────────────────────────────────────
export default function Income({ data, showToast }) {
  const { add, update, remove } = useData()
  const [modal, setModal] = useState(null) // null | { mode: 'add' } | { mode: 'edit', row }
  const [deleting, setDeleting] = useState(null)

  const rows = data?.income || []
  const sorted = sortedIncome(rows)
  const { period: latestPeriod, gross: latestGross, net: latestNet } = latestTotals(rows)

  const sipMonthly = (data?.sips || []).filter(s => s.status === 'active').reduce((a, s) => a + s.amount, 0)
  const emiMonthly = (data?.loans || []).reduce((a, l) => a + l.emi, 0)
  const surplus = latestNet > 0 ? latestNet - sipMonthly - emiMonthly : null
  const committedRate = latestNet > 0 ? ((sipMonthly + emiMonthly) / latestNet * 100) : null

  const handleAdd = async (row) => {
    await add('Income', { id: 'inc' + Date.now(), ...row })
    showToast('Income entry added')
  }

  const handleEdit = async (row) => {
    await update('Income', row.id, row)
    showToast('Income entry updated')
  }

  const handleDelete = async (id) => {
    await remove('Income', id)
    setDeleting(null)
    showToast('Entry deleted', 'error')
  }

  return (
    <div>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 28 }}>
        <div>
          <div style={KICK}>Income & Payslips</div>
          <h1 style={{ fontSize: 26, fontWeight: 800, margin: '6px 0 4px' }}>Income sources</h1>
          <div style={{ fontSize: 13, color: 'var(--ink-3)' }}>
            Upload payslips or add entries manually — used by the AI advisor for savings analysis.
          </div>
        </div>
        <button className="btn primary" onClick={() => setModal({ mode: 'add' })}>
          <Icon name="plus" size={15} /> Add income
        </button>
      </div>

      {/* Summary tiles */}
      {latestPeriod && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 32 }} className="income-tiles">
          <Tile label={`Gross · ${latestPeriod}`} value={fmtINR(latestGross)} />
          <Tile label={`Net take-home · ${latestPeriod}`} value={fmtINR(latestNet)} />
          <Tile
            label="Surplus after SIPs + EMIs"
            value={surplus !== null ? fmtINR(surplus) : '—'}
            sub={surplus !== null && surplus < 0 ? 'outflow exceeds take-home' : null}
            neg={surplus !== null && surplus < 0}
          />
          <Tile
            label="Committed rate"
            value={committedRate !== null ? committedRate.toFixed(1) + '%' : '—'}
            sub="of net goes to SIPs + EMIs"
          />
        </div>
      )}

      {/* Table */}
      {sorted.length === 0 ? (
        <Empty onAdd={() => setModal({ mode: 'add' })} />
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="tbl" style={{ minWidth: 640 }}>
            <thead>
              <tr>
                <th>Period</th>
                <th>Source</th>
                <th>Type</th>
                <th className="num">Gross</th>
                <th className="num">Net</th>
                <th className="num">Deductions</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {sorted.map(row => {
                const deductions = (row.pf_deduction || 0) + (row.tax_deduction || 0) + (row.other_deductions || 0)
                return (
                  <tr key={row.id}>
                    <td style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{row.period}</td>
                    <td>{row.source}</td>
                    <td>
                      <span className="tag" style={{ textTransform: 'capitalize' }}>{row.type}</span>
                    </td>
                    <td className="num">{fmtINR(row.gross)}</td>
                    <td className="num">{fmtINR(row.net)}</td>
                    <td className="num" style={{ color: 'var(--ink-3)', fontSize: 12.5 }}>
                      {deductions > 0 ? fmtINR(deductions) : '—'}
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end' }}>
                        <button className="btn ghost sm" title="Edit" onClick={() => setModal({ mode: 'edit', row })}>
                          <Icon name="edit" size={13} />
                        </button>
                        <button className="btn ghost sm" title="Delete" onClick={() => setDeleting(row.id)}>
                          <Icon name="trash" size={13} />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Modals */}
      {modal?.mode === 'add' && (
        <IncomeModal title="Add income" initial={{}} onClose={() => setModal(null)} onSave={handleAdd} />
      )}
      {modal?.mode === 'edit' && (
        <IncomeModal title="Edit income" initial={modal.row} onClose={() => setModal(null)} onSave={handleEdit} />
      )}
      {deleting && (
        <Modal title="Delete entry?" onClose={() => setDeleting(null)} width={380}>
          <p style={{ fontSize: 14, color: 'var(--ink-2)', margin: '0 0 20px' }}>
            This income entry will be permanently removed.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
            <button className="btn" onClick={() => setDeleting(null)}>Cancel</button>
            <button className="btn danger" onClick={() => handleDelete(deleting)}>Delete</button>
          </div>
        </Modal>
      )}
    </div>
  )
}

function Tile({ label, value, sub, neg }) {
  return (
    <div style={{ background: 'var(--surface)', border: '1px solid var(--line)', borderRadius: 'var(--r)', padding: '16px 18px' }}>
      <div style={{ fontSize: 11, fontWeight: 700, color: 'var(--ink-3)', textTransform: 'uppercase', letterSpacing: '.06em', marginBottom: 6 }}>{label}</div>
      <div className="num" style={{ fontSize: 20, fontWeight: 800, color: neg ? 'var(--neg)' : 'var(--ink)' }}>{value}</div>
      {sub && <div style={{ fontSize: 11, color: neg ? 'var(--neg)' : 'var(--ink-3)', marginTop: 3 }}>{sub}</div>}
    </div>
  )
}

function Empty({ onAdd }) {
  return (
    <div style={{ textAlign: 'center', padding: '60px 20px', color: 'var(--ink-3)' }}>
      <div style={{ fontSize: 36, marginBottom: 12 }}>💼</div>
      <div style={{ fontWeight: 700, fontSize: 15, color: 'var(--ink-2)', marginBottom: 6 }}>No income entries yet</div>
      <div style={{ fontSize: 13, marginBottom: 20, lineHeight: 1.6 }}>
        Upload a payslip or add your salary, freelance, or rental income manually.
      </div>
      <button className="btn primary" onClick={onAdd}>
        <Icon name="plus" size={15} /> Add income
      </button>
    </div>
  )
}

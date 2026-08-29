import { useState } from 'react'
import { Modal, Field } from '../../components/Primitives.jsx'
import { Icon } from '../../components/Icons.jsx'
import UploadZone from '../../components/UploadZone.jsx'
import { useData } from '../../data/context.jsx'
import { proposeTaxRules, approveTaxRule, rejectTaxRule } from '../../data/api.js'
import { fmtINR, fmtCompact } from '../../data/format.js'
import { KICK } from '../../data/tokens.js'
import { activeRuleSet } from './taxMath.js'

const EMPTY_CAPS = { section80C: 150000, section80DSelf: 25000, section80DSenior: 50000, nps80CCD1B: 50000 }
const EMPTY_RULES = {
  schemaVersion: 1, slabs: [{ rate: 0 }], stdDeduction: 0, rebateThreshold: 0,
  rebateAmount: 0, surcharge: [], cessRate: 0.04, deductionCaps: EMPTY_CAPS,
}

// "upto:rate" per line, last line "" (unbounded) is the top band — the
// compact editable form for the two variable-length arrays in a rule set.
function bandsText(bands, key) {
  return (bands || []).map(b => (b[key] == null ? `: ${b.rate}` : `${b[key]}: ${b.rate}`)).join('\n')
}
function parseBandsText(text, key) {
  return text.split('\n').map(l => l.trim()).filter(Boolean).map(line => {
    const [left, rateStr] = line.split(':').map(s => s.trim())
    const rate = Number(rateStr)
    if (!left) return { rate }
    return { [key]: Number(left), rate }
  })
}

function RuleSummary({ title, rules }) {
  if (!rules) return <div className="empty-hint">No active rule set yet.</div>
  const caps = rules.deductionCaps || {}
  return (
    <div>
      <div style={{ fontWeight: 700, fontSize: 13, marginBottom: 8 }}>{title}</div>
      <div style={{ fontSize: 12, color: 'var(--ink-3)', lineHeight: 1.9 }}>
        {(rules.slabs || []).map((b, i) => (
          <div key={i}>{b.upto != null ? `Up to ${fmtCompact(b.upto)}` : 'Above'}: {Math.round(b.rate * 100)}%</div>
        ))}
        <div>Std. deduction: {fmtINR(rules.stdDeduction || 0)}</div>
        <div>87A rebate: taxable ≤ {fmtCompact(rules.rebateThreshold || 0)}</div>
        {(rules.surcharge || []).map((b, i) => (
          <div key={i}>Surcharge &gt; {fmtCompact(b.above)}: {Math.round(b.rate * 100)}%</div>
        ))}
        <div>Cess: {Math.round((rules.cessRate || 0) * 100)}%</div>
        <div>80C: {fmtCompact(caps.section80C || 0)} · 80D: {fmtCompact(caps.section80DSelf || 0)}/{fmtCompact(caps.section80DSenior || 0)} · NPS: {fmtCompact(caps.nps80CCD1B || 0)}</div>
      </div>
    </div>
  )
}

function diffRows(oldRules, newRules) {
  const rows = []
  const push = (label, a, b, fmt = v => v) => { if ((a || 0) !== (b || 0)) rows.push({ label, from: fmt(a || 0), to: fmt(b || 0) }) }
  push('Standard deduction', oldRules.stdDeduction, newRules.stdDeduction, fmtINR)
  push('87A rebate threshold', oldRules.rebateThreshold, newRules.rebateThreshold, fmtCompact)
  push('Cess rate', oldRules.cessRate, newRules.cessRate, v => `${Math.round(v * 100)}%`)
  push('80C cap', oldRules.deductionCaps?.section80C, newRules.deductionCaps?.section80C, fmtCompact)
  push('80D cap (self)', oldRules.deductionCaps?.section80DSelf, newRules.deductionCaps?.section80DSelf, fmtCompact)
  push('80D cap (senior)', oldRules.deductionCaps?.section80DSenior, newRules.deductionCaps?.section80DSenior, fmtCompact)
  push('NPS 80CCD(1B) cap', oldRules.deductionCaps?.nps80CCD1B, newRules.deductionCaps?.nps80CCD1B, fmtCompact)
  if (JSON.stringify(oldRules.slabs) !== JSON.stringify(newRules.slabs)) rows.push({ label: 'Slabs', from: 'changed', to: '(see summary above)' })
  if (JSON.stringify(oldRules.surcharge) !== JSON.stringify(newRules.surcharge)) rows.push({ label: 'Surcharge', from: 'changed', to: '(see summary above)' })
  return rows
}

function ProposalRow({ proposal, baseline, onApprove, onReject, busy }) {
  let proposedRules = {}
  try { proposedRules = JSON.parse(proposal.rules_json) } catch { /* malformed row — show empty diff */ }
  const rows = diffRows(baseline || EMPTY_RULES, proposedRules)

  return (
    <div style={{ padding: '14px 0', borderBottom: '1px solid var(--line)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 6, flexWrap: 'wrap', gap: 8 }}>
        <div style={{ fontWeight: 700, fontSize: 13 }}>{proposal.fy} · {proposal.regime} regime</div>
        <span className="pill accent">Pending</span>
      </div>
      {proposal.source && <div style={{ fontSize: 11.5, color: 'var(--ink-3)', marginBottom: 8 }}>{proposal.source}</div>}
      <RuleSummary title="Proposed" rules={proposedRules} />
      {rows.length > 0 && (
        <div style={{ fontSize: 12, lineHeight: 1.8, marginTop: 8, background: 'var(--surface)', border: '1px solid var(--line)', borderRadius: 6, padding: '8px 12px' }}>
          {rows.map((r, i) => (
            <div key={i}>{r.label}: <span style={{ color: 'var(--ink-3)' }}>{String(r.from)}</span> → <strong>{String(r.to)}</strong></div>
          ))}
        </div>
      )}
      {proposal.notes && <div style={{ fontSize: 11.5, color: 'var(--ink-3)', marginTop: 8, fontStyle: 'italic' }}>{proposal.notes}</div>}
      <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
        <button className="btn ghost sm" disabled={busy} onClick={() => onApprove(proposal.id)}>
          <Icon name="check" size={14} /> Approve
        </button>
        <button className="btn ghost sm" disabled={busy} onClick={() => onReject(proposal.id)}>
          <Icon name="x" size={14} /> Reject
        </button>
      </div>
    </div>
  )
}

function ProposeForm({ onClose, onProposed }) {
  const [showUpload, setShowUpload] = useState(false)
  const [form, setForm] = useState(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)
  const [info, setInfo] = useState(null)

  const startBlank = () => setForm({ fy: '', regime: 'old', source: '', notes: '', rules: EMPTY_RULES })

  const handleExtracted = (fields) => {
    setShowUpload(false)
    const f = fields || {}
    setForm({
      fy: f.fy || '',
      regime: f.regime === 'new' ? 'new' : 'old',
      source: f.source || '',
      notes: f.notes || '',
      rules: {
        schemaVersion: 1,
        slabs: Array.isArray(f.slabs) ? f.slabs : [],
        stdDeduction: Number(f.stdDeduction) || 0,
        rebateThreshold: Number(f.rebateThreshold) || 0,
        rebateAmount: Number(f.rebateAmount) || 0,
        surcharge: Array.isArray(f.surcharge) ? f.surcharge : [],
        cessRate: Number(f.cessRate) || 0,
        deductionCaps: { ...EMPTY_CAPS, ...(f.deductionCaps || {}) },
      },
    })
  }

  const set = (path, value) => setForm(f => {
    const next = { ...f, rules: { ...f.rules, deductionCaps: { ...f.rules.deductionCaps } } }
    if (path[0] !== 'rules') { next[path[0]] = value; return next }
    if (path.length === 2) next.rules[path[1]] = value
    else next.rules.deductionCaps[path[2]] = value
    return next
  })

  const submit = async () => {
    setSaving(true)
    setError(null)
    setInfo(null)
    try {
      const result = await proposeTaxRules(form)
      if (result?.note) setInfo(result.note)
      else onProposed()
    } catch (e) {
      setError(e.message || 'Failed to submit proposal.')
    } finally {
      setSaving(false)
    }
  }

  if (!form) {
    return (
      <div>
        <div style={{ fontSize: 13, color: 'var(--ink-3)', marginBottom: 16, lineHeight: 1.6 }}>
          Upload an official document — Budget memorandum, Finance Bill excerpt, CBDT circular — and Claude extracts the new rules for you to review. Nothing takes effect until you approve it.
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button className="btn primary" onClick={() => setShowUpload(true)}>Upload a document</button>
          <button className="btn ghost" onClick={startBlank}>Enter manually instead</button>
        </div>
        {showUpload && (
          <UploadZone docType="tax_rules" onExtracted={handleExtracted} onClose={() => setShowUpload(false)} />
        )}
      </div>
    )
  }

  return (
    <div>
      <div style={{ fontSize: 12, color: 'var(--ink-3)', marginBottom: 14, lineHeight: 1.6 }}>
        Review every field — this only creates a pending proposal.
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 12 }}>
        <Field label="Financial year">
          <input className="input" value={form.fy} onChange={e => set(['fy'], e.target.value)} placeholder="FY 2027-28" />
        </Field>
        <Field label="Regime">
          <select className="input" value={form.regime} onChange={e => set(['regime'], e.target.value)}>
            <option value="old">Old</option>
            <option value="new">New</option>
          </select>
        </Field>
      </div>

      <Field label="Source">
        <input className="input" value={form.source} onChange={e => set(['source'], e.target.value)} placeholder="Union Budget 2027 Memorandum, Part A" />
      </Field>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, margin: '12px 0' }}>
        <Field label="Standard deduction (₹)">
          <input className="input" type="number" value={form.rules.stdDeduction} onChange={e => set(['rules', 'stdDeduction'], Number(e.target.value))} />
        </Field>
        <Field label="87A rebate threshold (₹)">
          <input className="input" type="number" value={form.rules.rebateThreshold} onChange={e => set(['rules', 'rebateThreshold'], Number(e.target.value))} />
        </Field>
        <Field label="87A rebate amount (₹)">
          <input className="input" type="number" value={form.rules.rebateAmount} onChange={e => set(['rules', 'rebateAmount'], Number(e.target.value))} />
        </Field>
        <Field label="Cess rate (0.04 = 4%)">
          <input className="input" type="number" step="0.01" value={form.rules.cessRate} onChange={e => set(['rules', 'cessRate'], Number(e.target.value))} />
        </Field>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 12 }}>
        <Field label="Slabs — one per line, upto:rate (blank upto = top band)">
          <textarea className="input" rows={6} style={{ fontFamily: 'monospace', fontSize: 12 }}
            value={bandsText(form.rules.slabs, 'upto')}
            onChange={e => set(['rules', 'slabs'], parseBandsText(e.target.value, 'upto'))} />
        </Field>
        <Field label="Surcharge — one per line, above:rate">
          <textarea className="input" rows={6} style={{ fontFamily: 'monospace', fontSize: 12 }}
            value={bandsText(form.rules.surcharge, 'above')}
            onChange={e => set(['rules', 'surcharge'], parseBandsText(e.target.value, 'above'))} />
        </Field>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 12, marginBottom: 12 }}>
        <Field label="80C cap"><input className="input" type="number" value={form.rules.deductionCaps.section80C} onChange={e => set(['rules', 'deductionCaps', 'section80C'], Number(e.target.value))} /></Field>
        <Field label="80D self"><input className="input" type="number" value={form.rules.deductionCaps.section80DSelf} onChange={e => set(['rules', 'deductionCaps', 'section80DSelf'], Number(e.target.value))} /></Field>
        <Field label="80D senior"><input className="input" type="number" value={form.rules.deductionCaps.section80DSenior} onChange={e => set(['rules', 'deductionCaps', 'section80DSenior'], Number(e.target.value))} /></Field>
        <Field label="NPS 80CCD(1B)"><input className="input" type="number" value={form.rules.deductionCaps.nps80CCD1B} onChange={e => set(['rules', 'deductionCaps', 'nps80CCD1B'], Number(e.target.value))} /></Field>
      </div>

      <Field label="Notes">
        <textarea className="input" rows={2} value={form.notes} onChange={e => set(['notes'], e.target.value)} />
      </Field>

      {error && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 6, background: 'var(--neg-soft)', color: 'var(--neg)', fontSize: 12.5 }}>
          {error}
        </div>
      )}
      {info && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 6, background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink-2)', fontSize: 12.5, lineHeight: 1.6 }}>
          {info}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 18 }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={submit} disabled={saving || !form.fy}>
          {saving ? 'Submitting…' : 'Submit for review'}
        </button>
      </div>
    </div>
  )
}

export default function TaxRulesAdmin({ data, onClose }) {
  const { reload } = useData()
  const rows = data.tax_rules || []
  const activeOld = activeRuleSet(rows, '', 'old')
  const activeNew = activeRuleSet(rows, '', 'new')
  const pending = rows.filter(r => r.status === 'pending')

  const [showProposeForm, setShowProposeForm] = useState(false)
  const [busyId, setBusyId] = useState(null)
  const [error, setError] = useState(null)

  const handleApprove = async (id) => {
    setBusyId(id)
    setError(null)
    try {
      const result = await approveTaxRule(id)
      if (result?.note) setError(result.note)
      else await reload()
    } catch (e) {
      setError(e.message || 'Failed to approve.')
    } finally {
      setBusyId(null)
    }
  }

  const handleReject = async (id) => {
    setBusyId(id)
    setError(null)
    try {
      const result = await rejectTaxRule(id)
      if (result?.note) setError(result.note)
      else await reload()
    } catch (e) {
      setError(e.message || 'Failed to reject.')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <Modal title="Tax rules" onClose={onClose} width={620}>
      <div style={{ fontSize: 12, color: 'var(--ink-3)', marginBottom: 18, lineHeight: 1.6 }}>
        The slab, surcharge, and deduction figures the tax engine uses. Nothing changes automatically —
        a proposed change only takes effect once you approve it here.
      </div>

      {error && (
        <div style={{ marginBottom: 14, padding: '10px 14px', borderRadius: 6, background: 'var(--neg-soft)', color: 'var(--neg)', fontSize: 12.5 }}>
          {error}
        </div>
      )}

      {!showProposeForm ? (
        <>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
            <RuleSummary title="Old regime (active)" rules={activeOld} />
            <RuleSummary title="New regime (active)" rules={activeNew} />
          </div>

          <div style={{ height: 1, background: 'var(--line)', margin: '18px 0' }} />

          <div style={{ ...KICK, marginBottom: 10 }}>Pending proposals</div>
          {pending.length === 0 ? (
            <div className="empty-hint" style={{ marginBottom: 16 }}>No pending proposals.</div>
          ) : (
            pending.map(p => (
              <ProposalRow
                key={p.id}
                proposal={p}
                baseline={p.regime === 'new' ? activeNew : activeOld}
                onApprove={handleApprove}
                onReject={handleReject}
                busy={busyId === p.id}
              />
            ))
          )}

          <div style={{ marginTop: 18 }}>
            <button className="btn primary" onClick={() => setShowProposeForm(true)}>Propose a change</button>
          </div>
        </>
      ) : (
        <ProposeForm
          onClose={() => setShowProposeForm(false)}
          onProposed={async () => { setShowProposeForm(false); await reload() }}
        />
      )}
    </Modal>
  )
}

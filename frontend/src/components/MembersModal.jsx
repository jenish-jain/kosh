import { useState } from 'react'
import { Modal, Field, Avatar } from './Primitives.jsx'
import { Icon } from './Icons.jsx'
import { useData } from '../data/context.jsx'
import { holdingsFor } from '../data/aggregate.js'

const SLABS = [0, 5, 10, 15, 20, 25, 30]
const COLORS = ['#1C4A3A', '#C98A5E', '#5B6FA6', '#A87C2A', '#8E5B8A', '#4F8A8B', '#B5603E', '#6B7F3A']
const EMPTY = { name: '', full_name: '', relation: '', slab: 0, color: COLORS[0] }

function holdingCount(data, id) {
  const h = holdingsFor(data, id)
  const own = arr => (arr || []).filter(x => x.member === id).length
  return Object.values(h).reduce((n, arr) => n + arr.length, 0) + own(data.sips) + own(data.lumpsums)
}

// The people whose finances the household tracks (Members tab) — distinct
// from who can sign in. Each person's tax slab drives the tax views.
export default function MembersModal({ data, onClose, showToast }) {
  const { add, update, remove } = useData()
  const [editing, setEditing] = useState(null) // null | 'new' | member id
  const [form, setForm] = useState(EMPTY)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const members = data.members || []

  const open = m => {
    setError(null)
    setEditing(m ? m.id : 'new')
    setForm(m ? { ...EMPTY, ...m } : { ...EMPTY, color: COLORS[members.length % COLORS.length] })
  }
  const set = k => e => setForm(f => ({ ...f, [k]: k === 'slab' ? Number(e.target.value) : e.target.value }))

  const save = async e => {
    e.preventDefault()
    const name = form.name.trim()
    if (!name) { setError('Give them a short name.'); return }
    setBusy(true)
    setError(null)
    const row = { name, full_name: form.full_name.trim() || name, relation: form.relation.trim(), slab: form.slab, color: form.color }
    try {
      if (editing === 'new') {
        await add('Members', { id: 'm' + Date.now().toString(36) + Math.random().toString(36).slice(2, 6), ...row })
        showToast?.(`${name} added`)
      } else {
        await update('Members', editing, row)
        showToast?.(`${name} updated`)
      }
      setEditing(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  const del = async m => {
    const n = holdingCount(data, m.id)
    if (n > 0) {
      setError(`${m.name} still has ${n} holding${n === 1 ? '' : 's'} or SIP${n === 1 ? '' : 's'}. Move or delete those first.`)
      return
    }
    if (!window.confirm(`Remove ${m.name}?`)) return
    setBusy(true)
    try { await remove('Members', m.id); showToast?.(`${m.name} removed`) } catch (err) { setError(err.message) } finally { setBusy(false) }
  }

  return (
    <Modal title={editing ? (editing === 'new' ? 'Add family member' : 'Edit family member') : 'Family members'} onClose={onClose} width={480}>
      {error && <div className="share-error">{error}</div>}
      {editing ? (
        <form onSubmit={save}>
          <Field label="Short name"><input className="input" value={form.name} onChange={set('name')} placeholder="e.g. Mom" autoFocus maxLength={30} /></Field>
          <Field label="Full name (optional)"><input className="input" value={form.full_name} onChange={set('full_name')} maxLength={60} /></Field>
          <Field label="Relation (optional)"><input className="input" value={form.relation} onChange={set('relation')} placeholder="e.g. Mother" maxLength={30} /></Field>
          <Field label="Income tax slab">
            <select className="input" value={form.slab} onChange={set('slab')}>
              {SLABS.map(s => <option key={s} value={s}>{s}%</option>)}
            </select>
          </Field>
          <Field label="Colour">
            <div className="color-picker" role="radiogroup" aria-label="Colour">
              {COLORS.map(c => (
                <button type="button" key={c} role="radio" aria-checked={form.color === c} aria-label={c}
                  className={'color-dot' + (form.color === c ? ' active' : '')} style={{ background: c }}
                  onClick={() => setForm(f => ({ ...f, color: c }))} />
              ))}
            </div>
          </Field>
          <div className="share-actions">
            <button type="button" className="btn" onClick={() => setEditing(null)} disabled={busy}>Cancel</button>
            <button className="btn primary" disabled={busy}>{busy ? 'Saving…' : 'Save'}</button>
          </div>
        </form>
      ) : (
        <>
          <p className="share-intro">The people whose money you track. Each holding belongs to one of them, and their tax slab drives the tax views.</p>
          {members.length === 0 ? <div className="share-empty">No one yet. Start with yourself.</div> : (
            <ul className="share-list">
              {members.map(m => (
                <li key={m.id} className="share-row">
                  <Avatar member={m} size={32} />
                  <div className="share-row-main">
                    <div className="cell-strong">{m.full_name || m.name}</div>
                    <div className="cell-sub">{[m.relation, `slab ${m.slab}%`].filter(Boolean).join(' · ')}</div>
                  </div>
                  <div className="share-row-actions">
                    <button className="btn ghost sm" onClick={() => open(m)} disabled={busy}>Edit</button>
                    <button className="btn ghost sm" onClick={() => del(m)} disabled={busy} title="Remove" aria-label={`Remove ${m.name}`}><Icon name="trash" size={14} /></button>
                  </div>
                </li>
              ))}
            </ul>
          )}
          <div className="share-actions">
            <button className="btn primary" onClick={() => open(null)}><Icon name="plus" size={14} /> Add family member</button>
          </div>
        </>
      )}
    </Modal>
  )
}

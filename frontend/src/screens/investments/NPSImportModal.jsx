import { useState } from 'react'
import { useData } from '../../data/context.jsx'
import { Modal, Field } from '../../components/Primitives.jsx'
import { fmtINR } from '../../data/format.js'

export default function NPSImportModal({ batch, data, onDone, onClose }) {
  const { add } = useData()
  const [member, setMember] = useState(batch.member)
  const [selected, setSelected] = useState(() => batch.holdings.map((_, i) => i))
  const [busy, setBusy] = useState(false)
  const toggle = (i) => setSelected(s => s.includes(i) ? s.filter(x => x !== i) : [...s, i])
  const n = Number

  const submit = async () => {
    setBusy(true)
    const toImport = batch.holdings.filter((_, i) => selected.includes(i))
    for (const h of toImport) {
      await add('NPS', {
        id: 'nps' + Date.now() + Math.random().toString(36).slice(2),
        pran: batch.pran || '', member,
        tier: h.tier || 'T1', asset_class: h.asset_class || 'E',
        scheme: h.scheme || '', fund_manager: h.fund_manager || '',
        units: n(h.units) || 0, nav: n(h.nav) || 0, invested: n(h.invested) || 0,
      })
    }
    onDone(toImport.length)
  }

  return (
    <Modal title="Import NPS holdings" onClose={onClose}>
      <div style={{ margin: '0 0 16px', padding: '9px 14px', borderRadius: 4, background: 'var(--accent-soft)', color: 'var(--accent-ink)', fontSize: 12.5, fontWeight: 600 }}>
        ✦ Claude extracted {batch.holdings.length} scheme{batch.holdings.length !== 1 ? 's' : ''} from your statement{batch.pran ? ` · PRAN ${batch.pran}` : ''}.
        {batch.net_total_invested > 0 && <> Total contributions: <span className="num">{fmtINR(batch.net_total_invested)}</span> — invested per scheme is prorated by declared allocation %.</>}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 16 }}>
        {batch.holdings.map((h, i) => {
          const cur = (n(h.units) || 0) * (n(h.nav) || 0)
          const on  = selected.includes(i)
          return (
            <div key={i} onClick={() => toggle(i)} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 12px', borderRadius: 6, border: `1px solid ${on ? 'var(--accent)' : 'var(--line)'}`, background: on ? 'var(--accent-soft)' : 'transparent', cursor: 'pointer' }}>
              <div style={{ width: 16, height: 16, borderRadius: 3, border: `2px solid ${on ? 'var(--accent)' : 'var(--line)'}`, background: on ? 'var(--accent)' : 'transparent', flexShrink: 0 }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 700 }}>
                  {h.tier || 'T1'} · Class {h.asset_class || '?'}{h.alloc_pct != null ? ` · ${h.alloc_pct}%` : ''} · {h.fund_manager || h.scheme}
                </div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-3)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{h.scheme}</div>
              </div>
              <div style={{ textAlign: 'right', flexShrink: 0 }}>
                <div className="num" style={{ fontSize: 13.5, fontWeight: 700 }}>{cur > 0 ? fmtINR(cur) : '—'}</div>
                <div style={{ fontSize: 11, color: 'var(--ink-3)' }}>
                  {h.invested > 0 ? `invested ${fmtINR(n(h.invested))} · ` : ''}{h.units} u × ₹{h.nav}
                </div>
              </div>
            </div>
          )
        })}
      </div>
      <Field label="Assign to member">
        <div className="seg" style={{ display: 'flex' }}>
          {(data.members || []).map(m => (
            <button key={m.id} className={member === m.id ? 'active' : ''} onClick={() => setMember(m.id)} style={{ flex: 1 }}>{m.name}</button>
          ))}
        </div>
      </Field>
      <div style={{ display: 'flex', gap: 10, marginTop: 8, justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose} disabled={busy}>Cancel</button>
        <button className="btn primary" onClick={submit} disabled={busy || selected.length === 0}>
          {busy ? 'Importing…' : `Import ${selected.length} holding${selected.length !== 1 ? 's' : ''}`}
        </button>
      </div>
    </Modal>
  )
}

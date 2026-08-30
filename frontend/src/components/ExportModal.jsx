import { useState } from 'react'
import { Modal } from './Primitives.jsx'
import { Icon } from './Icons.jsx'
import { scope } from '../data/aggregate.js'

const SECTIONS = [
  { key: 'summary', label: 'Net Worth Summary' },
  { key: 'mf', label: 'Mutual Funds' },
  { key: 'stocks', label: 'Stocks' },
  { key: 'metals', label: 'Gold & Silver' },
  { key: 'fixed', label: 'Fixed Deposits' },
  { key: 'insurance', label: 'Insurance' },
  { key: 'nps', label: 'NPS' },
  { key: 'loans', label: 'Loans / Liabilities' },
  { key: 'sips', label: 'SIPs' },
  { key: 'income', label: 'Income' },
]
const ALL_KEYS = SECTIONS.map(s => s.key)
const STORAGE_KEY = 'kosh.export.sections'

function loadSelected() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw && JSON.parse(raw)
    if (Array.isArray(parsed) && parsed.length > 0) return parsed.filter(k => ALL_KEYS.includes(k))
  } catch {
    // ignore malformed localStorage — fall through to the default
  }
  return ALL_KEYS
}

export default function ExportModal({ data, memberId, onClose }) {
  const [selected, setSelected] = useState(loadSelected)
  const [format, setFormat] = useState('pdf')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  const persist = next => {
    setSelected(next)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  }
  const toggle = key => persist(selected.includes(key) ? selected.filter(k => k !== key) : [...selected, key])
  const allChecked = selected.length === ALL_KEYS.length
  const toggleAll = () => persist(allChecked ? [] : ALL_KEYS)

  const download = async () => {
    if (selected.length === 0) {
      setError('Select at least one section.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const params = new URLSearchParams({ format, member: memberId || '', sections: selected.join(',') })
      const res = await fetch(`/api/report/export?${params.toString()}`, { credentials: 'include' })
      if (!res.ok) {
        const msg = await res.text().catch(() => '')
        throw new Error(msg || `Server error ${res.status}`)
      }
      const blob = await res.blob()
      const disposition = res.headers.get('Content-Disposition') || ''
      const match = disposition.match(/filename="?([^"]+)"?/)
      const filename = match ? match[1] : `Kosh-Report.${format}`

      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      onClose()
    } catch (e) {
      setError(e.message || 'Failed to generate report.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="Export report" onClose={onClose} width={440}>
      <div style={{ fontSize: 12.5, color: 'var(--ink-3)', marginBottom: 18, lineHeight: 1.6 }}>
        Scope: <strong style={{ color: 'var(--ink)' }}>{scope(data, memberId)}</strong> — change this from the member switcher at the top of the app.
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
        <span style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '.08em', color: 'var(--ink-3)' }}>Sections</span>
        <button className="btn ghost sm" onClick={toggleAll}>{allChecked ? 'Clear all' : 'Select all'}</button>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '9px 16px', marginBottom: 22 }}>
        {SECTIONS.map(s => (
          <label key={s.key} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, cursor: 'pointer' }}>
            <input type="checkbox" checked={selected.includes(s.key)} onChange={() => toggle(s.key)} />
            {s.label}
          </label>
        ))}
      </div>

      <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '.08em', color: 'var(--ink-3)', marginBottom: 8 }}>Format</div>
      <div className="seg" style={{ display: 'inline-flex', marginBottom: 22 }}>
        {[['pdf', 'PDF'], ['xlsx', 'Excel']].map(([v, label]) => (
          <button key={v} className={format === v ? 'active' : ''} onClick={() => setFormat(v)}>{label}</button>
        ))}
      </div>

      {error && (
        <div style={{ marginBottom: 16, padding: '10px 14px', borderRadius: 6, background: 'var(--neg-soft)', color: 'var(--neg)', fontSize: 12.5, lineHeight: 1.6 }}>
          {error}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
        <button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" onClick={download} disabled={busy}>
          <Icon name="download" size={14} />
          {busy ? 'Preparing…' : 'Download'}
        </button>
      </div>
    </Modal>
  )
}

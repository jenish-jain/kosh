import { useState, useEffect, useRef, useMemo } from 'react'
import { fmtPct } from '../data/format.js'
import { Icon } from './Icons.jsx'

// ── Avatar ──────────────────────────────────────────────────
export function Avatar({ member, size = 26 }) {
  const initials = (member.full_name || member.name || '?')
    .replace(/\s*\(.*\)/, '')
    .split(' ')
    .map(w => w[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
  return (
    <span
      className="avatar"
      style={{ background: member.color, width: size, height: size, flexBasis: size, fontSize: size * 0.42 }}
    >
      {initials}
    </span>
  )
}

// ── Gain pill ────────────────────────────────────────────────
export function GainPill({ value, pct }) {
  const pos = value >= 0
  return (
    <span className={'pill ' + (pos ? 'pos' : 'neg')}>
      <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6"
        strokeLinecap="round" strokeLinejoin="round"
        style={{ transform: pos ? 'none' : 'rotate(180deg)' }}>
        <path d="M12 19V5M6 11l6-6 6 6" />
      </svg>
      {fmtPct(pct)}
    </span>
  )
}

// ── Area chart ───────────────────────────────────────────────
export function AreaChart({ data, height = 150, accent = 'var(--accent)', showAxis = true }) {
  const ref = useRef(null)
  const [w, setW] = useState(600)
  useEffect(() => {
    if (!ref.current) return
    const ro = new ResizeObserver(es => setW(es[0].contentRect.width))
    ro.observe(ref.current)
    return () => ro.disconnect()
  }, [])
  const pad = { l: 8, r: 8, t: 12, b: showAxis ? 22 : 8 }
  const vals = data.map(d => d.value)
  const min = Math.min(...vals) * 0.985
  const max = Math.max(...vals) * 1.01
  const iw = Math.max(w - pad.l - pad.r, 10)
  const ih = height - pad.t - pad.b
  const x = i => pad.l + (i / (data.length - 1)) * iw
  const y = v => pad.t + ih - ((v - min) / (max - min || 1)) * ih
  const line = data.map((d, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)} ${y(d.value).toFixed(1)}`).join(' ')
  const area = line + ` L${x(data.length - 1).toFixed(1)} ${(pad.t + ih).toFixed(1)} L${x(0).toFixed(1)} ${(pad.t + ih).toFixed(1)} Z`
  const gid = useMemo(() => 'g' + Math.random().toString(36).slice(2, 7), [])
  return (
    <div ref={ref} style={{ width: '100%' }}>
      <svg width="100%" height={height} viewBox={`0 0 ${w} ${height}`} preserveAspectRatio="none" style={{ display: 'block' }}>
        <defs>
          <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={accent} stopOpacity="0.18" />
            <stop offset="100%" stopColor={accent} stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={area} fill={`url(#${gid})`} />
        <path d={line} fill="none" stroke={accent} strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" />
        {data.map((d, i) => i === data.length - 1 ? (
          <circle key={i} cx={x(i)} cy={y(d.value)} r="3.5" fill={accent} stroke="#fff" strokeWidth="2" />
        ) : null)}
        {showAxis && data.map((d, i) => (
          (i % 2 === 0 || i === data.length - 1) ? (
            <text key={'t' + i} x={x(i)} y={height - 6} fontSize="10" fill="var(--ink-3)" textAnchor="middle" fontWeight="600">
              {d.month}
            </text>
          ) : null
        ))}
      </svg>
    </div>
  )
}

// ── Ed Stack (allocation bar) ────────────────────────────────
export function EdStack({ segs, h = 13 }) {
  const total = segs.reduce((a, s) => a + s.value, 0) || 1
  return (
    <div className="ed-stack" style={{ height: h }}>
      {segs.map((s, i) => (
        <span key={i} title={s.label}
          style={{ width: (s.value / total * 100) + '%', background: s.color }} />
      ))}
    </div>
  )
}

// ── EdRule (horizontal hairline) ────────────────────────────
export function EdRule({ thick }) {
  return <div className={'ed-rule' + (thick ? ' thick' : '')} />
}

// ── Modal ────────────────────────────────────────────────────
export function Modal({ title, onClose, children, width = 460 }) {
  useEffect(() => {
    const h = e => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onClose])
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card fade-in" style={{ width }} onClick={e => e.stopPropagation()}>
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="btn ghost sm modal-close-btn" onClick={onClose} style={{ padding: 8 }}>
            <Icon name="x" size={18} />
          </button>
        </div>
        <div className="modal-body">{children}</div>
      </div>
    </div>
  )
}

// ── Field + input ────────────────────────────────────────────
export function Field({ label, children }) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      {children}
    </label>
  )
}

// ── Editable cell ────────────────────────────────────────────
export function EditCell({ value, onChange, type = 'text', align = 'left', format }) {
  const [v, setV] = useState(value)
  const [editing, setEditing] = useState(false)
  useEffect(() => setV(value), [value])
  const display = format ? format(value) : (type === 'number' ? value : value)
  return editing ? (
    <input
      className="edit-cell"
      type={type}
      value={v}
      autoFocus
      style={{ textAlign: align }}
      onChange={e => setV(type === 'number' ? e.target.value : e.target.value)}
      onBlur={() => {
        setEditing(false)
        if (onChange) onChange(type === 'number' ? Number(v) : v)
      }}
      onKeyDown={e => {
        if (e.key === 'Enter') e.target.blur()
        if (e.key === 'Escape') { setV(value); setEditing(false) }
      }}
    />
  ) : (
    <button
      className="edit-cell"
      style={{ textAlign: align, cursor: 'text' }}
      onClick={() => setEditing(true)}
    >
      {display}
    </button>
  )
}

// ── Member tag (colored dot + name) ─────────────────────────
export function MemberTag({ member }) {
  if (!member) return null
  return (
    <span className="tag">
      <span className="dot" style={{ background: member.color }} />
      {member.name}
    </span>
  )
}

// ── Toast ────────────────────────────────────────────────────
export function Toast({ message, type, onDone }) {
  useEffect(() => {
    const t = setTimeout(onDone, 3000)
    return () => clearTimeout(t)
  }, [onDone])
  return <div className={`toast ${type}`}>{message}</div>
}

// ── Save button bar ──────────────────────────────────────────
export function SaveBar({ dirty, saving, onSave }) {
  if (!dirty) return null
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16, gap: 10 }}>
      <button className="btn primary" onClick={onSave} disabled={saving}>
        <Icon name="save" size={15} />
        {saving ? 'Saving…' : 'Save changes'}
      </button>
    </div>
  )
}

import { useState, useRef } from 'react'
import { Icon } from './Icons.jsx'
import { getDriveAccessToken } from '../data/driveAuth.js'

const ACCEPT = '.pdf,.jpg,.jpeg,.png'

export default function UploadZone({ docType, clientId, onExtracted, onClose }) {
  const [state, setState] = useState('idle') // idle | connecting | uploading | error
  const [error, setError] = useState(null)
  const [dragging, setDragging] = useState(false)
  const [password, setPassword] = useState('')
  const inputRef = useRef(null)

  const upload = async (file) => {
    if (!file) return
    setError(null)
    try {
      setState('connecting')
      let driveToken = ''
      try {
        driveToken = await getDriveAccessToken(clientId)
      } catch {
        // Drive access is optional — the document still gets parsed and
        // pre-fills the form, it just won't be saved to Drive.
      }

      setState('uploading')
      const form = new FormData()
      form.append('file', file)
      if (driveToken) form.append('drive_token', driveToken)
      if (password) form.append('password', password)
      const res = await fetch(`/api/upload/${docType}`, {
        method: 'POST',
        credentials: 'include',
        body: form,
      })
      if (!res.ok) {
        const msg = await res.text().catch(() => '')
        throw new Error(msg || `Server error ${res.status}`)
      }
      const { fields, drive_url } = await res.json()
      onExtracted(fields, drive_url)
    } catch (e) {
      setError(e.message)
      setState('error')
    }
  }

  const onFiles = (files) => {
    const file = files?.[0]
    if (!file) return
    upload(file)
  }

  const onDrop = (e) => {
    e.preventDefault()
    setDragging(false)
    onFiles(e.dataTransfer.files)
  }

  const labels = {
    fd:        'FD certificate',
    insurance: 'insurance policy / schedule',
    metals:    'gold or silver invoice',
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 200,
      background: 'rgba(34,31,26,.35)', backdropFilter: 'blur(3px)',
      display: 'grid', placeItems: 'center', padding: 20,
    }} onClick={onClose}>
      <div style={{
        background: 'var(--surface)', border: '1px solid var(--line)',
        borderRadius: 'var(--r)', boxShadow: 'var(--shadow-lg)',
        width: 440, padding: 32, textAlign: 'center',
      }} onClick={e => e.stopPropagation()}>

        {state === 'connecting' ? (
          <>
            <div style={{ fontSize: 36, marginBottom: 16 }}>🔐</div>
            <div style={{ fontWeight: 700, fontSize: 15, marginBottom: 6 }}>Connecting to Google Drive…</div>
            <div style={{ fontSize: 13, color: 'var(--ink-3)', lineHeight: 1.6 }}>
              First time only — Kosh needs permission to save a copy of files it uploads.<br />
              You may see a Google sign-in popup.
            </div>
          </>
        ) : state === 'uploading' ? (
          <>
            <div style={{ fontSize: 36, marginBottom: 16 }}>⏳</div>
            <div style={{ fontWeight: 700, fontSize: 15, marginBottom: 6 }}>Reading document…</div>
            <div style={{ fontSize: 13, color: 'var(--ink-3)', lineHeight: 1.6 }}>
              Uploading to Drive and parsing with Claude.<br />This takes about 5–10 seconds.
            </div>
          </>
        ) : (
          <>
            {/* Drop zone */}
            <div
              onDragOver={e => { e.preventDefault(); setDragging(true) }}
              onDragLeave={() => setDragging(false)}
              onDrop={onDrop}
              onClick={() => inputRef.current?.click()}
              style={{
                border: `2px dashed ${dragging ? 'var(--accent)' : 'var(--line)'}`,
                borderRadius: 8,
                padding: '32px 24px',
                cursor: 'pointer',
                background: dragging ? 'var(--accent-soft)' : 'transparent',
                transition: 'all .15s',
                marginBottom: 20,
              }}
            >
              <div style={{ fontSize: 32, marginBottom: 12 }}>📄</div>
              <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 6 }}>
                Drop your {labels[docType] || 'document'} here
              </div>
              <div style={{ fontSize: 12.5, color: 'var(--ink-3)' }}>
                or click to browse · PDF, JPG, PNG · max 20 MB
              </div>
              <input
                ref={inputRef}
                type="file"
                accept={ACCEPT}
                style={{ display: 'none' }}
                onChange={e => onFiles(e.target.files)}
              />
            </div>

            <div style={{ textAlign: 'left', marginBottom: 16 }}>
              <input
                type="password"
                className="input"
                placeholder="Password (only if this PDF is protected)"
                value={password}
                onChange={e => setPassword(e.target.value)}
                autoComplete="off"
              />
              <div style={{ fontSize: 11, color: 'var(--ink-3)', marginTop: 5, lineHeight: 1.5 }}>
                Used only to open the file for reading — never stored or sent anywhere else.
              </div>
            </div>

            {state === 'error' && (
              <div style={{
                padding: '10px 14px', borderRadius: 4, marginBottom: 16,
                background: 'var(--neg-soft)', color: 'var(--neg)',
                fontSize: 13, textAlign: 'left', lineHeight: 1.5,
              }}>
                {error}
              </div>
            )}

            <div style={{ fontSize: 12, color: 'var(--ink-3)', lineHeight: 1.6, marginBottom: 20 }}>
              Claude reads the document, extracts the key details, and pre-fills the form.
              You review everything before it's saved.
            </div>

            <button className="btn" onClick={onClose} style={{ width: '100%' }}>Cancel</button>
          </>
        )}
      </div>
    </div>
  )
}

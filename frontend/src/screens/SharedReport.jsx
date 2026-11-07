import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Icon } from '../components/Icons.jsx'
import { KICK, SERIF } from '../data/tokens.js'
import { verifyShare, fetchSharedReport, lockShare, sharedPdfUrl } from '../data/shareApi.js'

// Public, read-only family report opened from a share link (/s/:token).
// The token identifies the recipient; a 6-digit authenticator code unlocks
// it for a day.
export default function SharedReport() {
  const { token } = useParams()
  const [state, setState] = useState('checking') // checking | code | ready
  const [report, setReport] = useState(null)

  useEffect(() => {
    document.title = 'Kosh · Family report'
    // A no-referrer meta covers the client side too, in case a header is
    // stripped by an intermediate proxy.
    const meta = document.createElement('meta')
    meta.name = 'referrer'
    meta.content = 'no-referrer'
    document.head.appendChild(meta)
    return () => meta.remove()
  }, [])

  const load = useCallback(() => {
    fetchSharedReport(token)
      .then(r => { setReport(r); setState('ready') })
      .catch(() => setState('code'))
  }, [token])
  useEffect(load, [load])

  const lock = async () => {
    await lockShare().catch(() => {})
    setReport(null)
    setState('code')
  }

  if (state === 'checking') return <div className="shared-page"><div style={{ ...KICK, textAlign: 'center', paddingTop: '30vh' }}>Loading…</div></div>
  if (state === 'code') return <CodeEntry token={token} onVerified={load} />
  return <ReportView token={token} report={report} onLock={lock} />
}

function CodeEntry({ token, onVerified }) {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const inputRef = useRef(null)

  const submit = async e => {
    e?.preventDefault()
    if (code.length !== 6) return
    setBusy(true)
    setError(null)
    try {
      await verifyShare(token, code)
      onVerified()
    } catch (err) {
      setError(err.message)
      setCode('')
      inputRef.current?.focus()
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="shared-page shared-center">
      <form className="shared-code-card" onSubmit={submit}>
        <div style={{ fontFamily: SERIF, fontSize: 56, color: 'var(--accent)', lineHeight: 1, marginBottom: 12 }}>क</div>
        <div style={{ fontSize: 20, fontWeight: 800, marginBottom: 4 }}>Kosh family report</div>
        <div style={{ ...KICK, marginBottom: 28 }}>Private · read-only</div>
        <label htmlFor="share-code" className="shared-code-label">
          Open your authenticator app and enter the 6-digit code shown for <strong>Kosh</strong>.
        </label>
        <input
          id="share-code" ref={inputRef}
          className="input shared-code-input"
          inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]*" maxLength={6}
          value={code} autoFocus disabled={busy}
          onChange={e => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
          aria-label="6-digit code"
        />
        {error && <div className="share-error" role="alert">{error}</div>}
        <button className="btn primary shared-code-btn" disabled={busy || code.length !== 6}>
          {busy ? 'Checking…' : 'Unlock'}
        </button>
        <div className="shared-code-foot">
          The code changes every 30 seconds. Lost your phone? Ask the person who shared this to reset your access.
        </div>
      </form>
    </div>
  )
}

const isNumeric = s => /^[−-]?₹|^[−-]?\d[\d,.]*%?$/.test(s)

function ReportView({ token, report, onLock }) {
  const summary = report.tables.find(t => t.title === 'Net Worth Summary')
  const netWorth = summary?.rows.find(r => r[0] === 'Net Worth')?.[2]
  const tables = report.tables.filter(t => t.rows?.length > 0)

  return (
    <div className="shared-page">
      <header className="shared-head">
        <div>
          <div style={KICK}>Kosh · {report.scope}</div>
          <div className="shared-hello">Hi {report.recipient}</div>
        </div>
        <div className="shared-head-actions">
          <a className="btn sm" href={sharedPdfUrl(token)} download><Icon name="download" size={14} /> PDF</a>
          <button className="btn ghost sm" onClick={onLock} title="Lock this device">Lock</button>
        </div>
      </header>

      {netWorth && (
        <div className="shared-hero">
          <div style={KICK}>Household net worth</div>
          <div className="hero-num num">{netWorth}</div>
          <div className="cell-sub">As of {report.generated} · read-only view</div>
        </div>
      )}

      {tables.map(t => (
        <section key={t.title} className="shared-section">
          <h2 className="shared-section-title">{t.title}</h2>
          <div className="tbl-scroll">
            <table className="tbl">
              <thead>
                <tr>{t.headers.map((h, i) => <th key={i} className={i > 0 && t.rows.some(r => isNumeric(r[i] || '')) ? 'r' : ''}>{h}</th>)}</tr>
              </thead>
              <tbody>
                {t.rows.map((r, ri) => (
                  <tr key={ri}>
                    {r.map((c, ci) => <td key={ci} className={ci > 0 && isNumeric(c) ? 'r num' : ''}>{c}</td>)}
                  </tr>
                ))}
              </tbody>
              {t.totals?.some(Boolean) && (
                <tfoot>
                  <tr>{t.totals.map((c, ci) => <td key={ci} className={ci > 0 && isNumeric(c) ? 'r num cell-strong' : 'cell-strong'}>{c}</td>)}</tr>
                </tfoot>
              )}
            </table>
          </div>
        </section>
      ))}

      <footer className="shared-foot">
        Shared privately with {report.recipient}. Please don't forward this page or the PDF.
      </footer>
    </div>
  )
}

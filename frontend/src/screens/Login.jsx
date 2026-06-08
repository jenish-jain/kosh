import { useEffect, useRef, useState } from 'react'

const SERIF = "var(--serif)"
const KICK  = { textTransform: 'uppercase', letterSpacing: '.18em', fontWeight: 700, fontSize: 10.5, color: 'var(--ink-3)' }

export default function Login({ clientId, demoAvailable, onDemo, onLogin }) {
  const btnRef  = useRef(null)
  const [error, setError] = useState(null)
  const [busy,  setBusy]  = useState(false)
  const [demoBusy, setDemoBusy] = useState(false)

  const handleDemo = async () => {
    setDemoBusy(true)
    setError(null)
    try {
      await onDemo()
    } catch {
      setError('Could not start the demo. Is the backend running?')
      setDemoBusy(false)
    }
  }

  useEffect(() => {
    let mounted = true

    const init = () => {
      if (!mounted || !btnRef.current) return
      window.google.accounts.id.initialize({
        client_id: clientId,
        callback: async ({ credential }) => {
          if (!mounted) return
          setBusy(true)
          setError(null)
          try {
            const res = await fetch('/api/auth/login', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ credential }),
            })
            if (res.ok) {
              const user = await res.json()
              onLogin(user)
            } else {
              setError(await res.text())
            }
          } catch {
            setError('Could not reach the server. Is the backend running?')
          } finally {
            setBusy(false)
          }
        },
      })
      window.google.accounts.id.renderButton(btnRef.current, {
        theme: 'outline',
        size: 'large',
        text: 'signin_with',
        shape: 'rectangular',
        logo_alignment: 'left',
      })
    }

    // Google GSI loads async — poll until ready.
    if (window.google?.accounts) {
      init()
    } else {
      const iv = setInterval(() => {
        if (window.google?.accounts) { clearInterval(iv); init() }
      }, 80)
      return () => { mounted = false; clearInterval(iv) }
    }
    return () => { mounted = false }
  }, [clientId, onLogin])

  return (
    <div style={{
      minHeight: '100vh', background: 'var(--bg)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{ width: 360, textAlign: 'center' }}>
        {/* Brand */}
        <div style={{
          fontFamily: SERIF, fontSize: 72, color: 'var(--accent)',
          lineHeight: 1, marginBottom: 16,
        }}>क</div>
        <div style={{ fontSize: 22, fontWeight: 800, letterSpacing: '-.01em', marginBottom: 4 }}>
          Kosh
        </div>
        <div style={{ ...KICK, marginBottom: 40 }}>Family wealth ledger</div>

        {/* Hairline */}
        <div style={{ height: 1, background: 'var(--line)', marginBottom: 40 }} />

        {/* Sign-in area */}
        <div style={{ fontSize: 13.5, color: 'var(--ink-3)', marginBottom: 24, lineHeight: 1.6 }}>
          Sign in with your Google account to continue.<br />
          Access is restricted to invited members.
        </div>

        {busy ? (
          <div style={{ ...KICK, color: 'var(--accent)' }}>Verifying…</div>
        ) : (
          <div ref={btnRef} style={{ display: 'flex', justifyContent: 'center' }} />
        )}

        {error && (
          <div style={{
            marginTop: 20, padding: '12px 16px',
            background: 'var(--neg-soft)', color: 'var(--neg)',
            borderRadius: 4, fontSize: 13, lineHeight: 1.5, textAlign: 'left',
          }}>
            {error}
          </div>
        )}

        {demoAvailable && (
          <>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, margin: '28px 0' }}>
              <div style={{ flex: 1, height: 1, background: 'var(--line)' }} />
              <div style={{ ...KICK, fontSize: 9.5 }}>or</div>
              <div style={{ flex: 1, height: 1, background: 'var(--line)' }} />
            </div>
            <button
              onClick={handleDemo} disabled={demoBusy}
              style={{
                display: 'block', width: '100%', padding: '11px 16px', borderRadius: 4,
                border: '1px solid var(--line)', background: 'none', color: 'var(--ink)',
                fontSize: 13.5, fontWeight: 600, cursor: demoBusy ? 'default' : 'pointer',
              }}
            >
              {demoBusy ? 'Loading demo…' : 'Try the demo →'}
            </button>
            <div style={{ marginTop: 10, fontSize: 12, color: 'var(--ink-3)', lineHeight: 1.5 }}>
              No sign-in needed — explore with sample data, read-only.
            </div>
          </>
        )}

        {/* Footer */}
        <div style={{
          marginTop: 48, fontFamily: SERIF, fontSize: 12.5,
          fontStyle: 'italic', color: 'var(--ink-3)',
        }}>
          A quiet ledger, not a live feed.
        </div>
      </div>
    </div>
  )
}

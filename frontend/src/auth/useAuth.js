import { useState, useEffect } from 'react'

export function useAuth() {
  const [authState, setAuthState] = useState('checking') // 'checking' | 'login' | 'authed'
  const [user, setUser] = useState(null)
  const [clientId, setClientId] = useState(null)
  const [demoAvailable, setDemoAvailable] = useState(false)
  const [aiEnabled, setAiEnabled] = useState(false)

  useEffect(() => {
    fetch('/api/auth/config')
      .then(r => r.json())
      .then(cfg => {
        setDemoAvailable(!!cfg.demo)
        setAiEnabled(!!cfg.ai_enabled)
        if (!cfg.enabled) { setAuthState('authed'); return }
        setClientId(cfg.client_id)
        return fetch('/api/auth/me', { credentials: 'include' })
          .then(r => { if (r.ok) return r.json(); throw new Error() })
          .then(u => { setUser(u); setAuthState('authed') })
          .catch(() => setAuthState('login'))
      })
      .catch(() => setAuthState('authed'))
  }, [])

  const handleLogin = u => { setUser(u); setAuthState('authed') }

  const handleDemo = async () => {
    const res = await fetch('/api/auth/demo-login', { method: 'POST', credentials: 'include' })
    if (res.ok) handleLogin(await res.json())
  }

  const handleLogout = async () => {
    await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {})
    setUser(null)
    setAuthState('login')
  }

  return { authState, user, clientId, demoAvailable, aiEnabled, handleLogin, handleDemo, handleLogout }
}

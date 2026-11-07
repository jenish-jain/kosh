import { useEffect, useState } from 'react'

// The server's mode ("selfhost" | "hosted"), from /api/auth/config. The
// same build runs both, so hosted-only pages (pricing) are switched on at
// runtime rather than at build time. Fetched once per page load.
let pending = null
function fetchMode() {
  pending ??= Promise.resolve()
    .then(() => fetch('/api/auth/config'))
    .then(r => (r.ok ? r.json() : {}))
    .then(cfg => cfg.mode || 'selfhost')
    .catch(() => 'selfhost')
  return pending
}

// useHosted: null while loading, then whether this deploy is hosted Kosh.
export function useHosted() {
  const [hosted, setHosted] = useState(null)
  useEffect(() => {
    let live = true
    fetchMode().then(m => { if (live) setHosted(m === 'hosted') })
    return () => { live = false }
  }, [])
  return hosted
}

export function resetServerModeForTests() { pending = null }

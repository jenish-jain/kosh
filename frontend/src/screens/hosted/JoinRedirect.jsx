import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { JOIN_TOKEN_KEY } from './HostedGate.jsx'

// /join#<token>: stash the invite token (it's in the fragment so it never
// reaches server logs), strip it from the address bar, and continue into the
// app, which signs the person in and then offers to accept it.
export default function JoinRedirect() {
  const navigate = useNavigate()
  useEffect(() => {
    const token = window.location.hash.slice(1)
    if (token) {
      try { sessionStorage.setItem(JOIN_TOKEN_KEY, token) } catch { /* storage unavailable */ }
    }
    navigate('/app', { replace: true })
  }, [navigate])
  return null
}

import { useState, useEffect, useRef, useCallback } from 'react'

const HASH_KEY    = 'kosh.pin.hash'
const TIMEOUT_KEY = 'kosh.pin.timeout'

async function hashPin(pin) {
  const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode('kosh-pin:' + pin))
  return Array.from(new Uint8Array(buf)).map(b => b.toString(16).padStart(2, '0')).join('')
}

export function usePinLock() {
  const [locked, setLockedState]   = useState(false)
  const [pinConfigured, setPinConfigured] = useState(() => !!localStorage.getItem(HASH_KEY))
  const [timeoutMinutes, setTimeoutState] = useState(() => {
    const v = localStorage.getItem(TIMEOUT_KEY)
    return v !== null ? parseInt(v, 10) : 5
  })

  const lockedRef    = useRef(false)
  const lastActivity = useRef(Date.now())

  const setLocked = v => { lockedRef.current = v; setLockedState(v) }

  const updateActivity = useCallback(() => {
    if (!lockedRef.current) lastActivity.current = Date.now()
  }, [])

  useEffect(() => {
    if (!pinConfigured || timeoutMinutes === 0) return
    const EVENTS = ['mousemove', 'mousedown', 'keydown', 'scroll', 'touchstart']
    EVENTS.forEach(e => document.addEventListener(e, updateActivity, { passive: true }))
    const id = setInterval(() => {
      if (!lockedRef.current && Date.now() - lastActivity.current >= timeoutMinutes * 60_000)
        setLocked(true)
    }, 15_000)
    return () => { EVENTS.forEach(e => document.removeEventListener(e, updateActivity)); clearInterval(id) }
  }, [pinConfigured, timeoutMinutes, updateActivity])

  // verify without unlocking — used by settings before changing/removing PIN
  const checkPin = async pin => {
    const hash = await hashPin(pin)
    return hash === localStorage.getItem(HASH_KEY)
  }

  // verify + unlock — used by the lock screen
  const verify = async pin => {
    if (!await checkPin(pin)) return false
    lastActivity.current = Date.now()
    setLocked(false)
    return true
  }

  const setupPin = async pin => {
    localStorage.setItem(HASH_KEY, await hashPin(pin))
    setPinConfigured(true)
  }

  // call only after checkPin has already confirmed the current PIN
  const clearPin = () => {
    localStorage.removeItem(HASH_KEY)
    setPinConfigured(false)
    setLocked(false)
  }

  const setTimeoutMinutes = mins => {
    localStorage.setItem(TIMEOUT_KEY, String(mins))
    setTimeoutState(mins)
  }

  const lockNow = () => { if (pinConfigured) setLocked(true) }

  return { locked, pinConfigured, timeoutMinutes, checkPin, verify, setupPin, clearPin, setTimeoutMinutes, lockNow }
}

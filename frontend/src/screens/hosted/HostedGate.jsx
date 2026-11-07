import { useCallback, useEffect, useState } from 'react'
import { hostedMe } from '../../data/hostedApi.js'
import LoadingScreen from '../../components/layout/LoadingScreen.jsx'
import ErrorScreen from '../../components/layout/ErrorScreen.jsx'
import Onboarding from './Onboarding.jsx'
import JoinHousehold from './JoinHousehold.jsx'
import Reconnect from './Reconnect.jsx'

export const JOIN_TOKEN_KEY = 'kosh.joinToken'

function readJoinToken() {
  try { return sessionStorage.getItem(JOIN_TOKEN_KEY) } catch { return null }
}
function clearJoinToken() {
  try { sessionStorage.removeItem(JOIN_TOKEN_KEY) } catch { /* storage unavailable */ }
}

// Hosted mode: decides what a signed-in account sees before the app proper —
// onboarding, a pending invite, or a Drive reconnect — then renders children
// with the household.
export default function HostedGate({ user, clientId, devLogin, onSignOut, children }) {
  const [state, setState] = useState({ loading: true })
  const [joinToken, setJoinToken] = useState(readJoinToken)

  // The first load shows a spinner; later refreshes (after subscribing,
  // say) update in place so the app — and its toasts — stay mounted.
  const load = useCallback((background = false) => {
    if (!background) setState({ loading: true })
    hostedMe().then(me => setState({ household: me.household, operator: !!me.operator })).catch(e => setState({ error: e.message }))
  }, [])
  useEffect(() => load(), [load])
  const refresh = useCallback(() => load(true), [load])

  const settle = household => { clearJoinToken(); setJoinToken(null); setState(s => ({ household, operator: s.operator })) }
  const common = { email: user.email, clientId, devLogin, onSignOut }

  if (state.loading) return <LoadingScreen />
  if (state.error) return <ErrorScreen message={state.error} />
  if (joinToken) {
    return <JoinHousehold token={joinToken} hasHousehold={!!state.household} {...common}
      onDone={settle} onDismiss={() => { clearJoinToken(); setJoinToken(null) }} />
  }
  if (!state.household) return <Onboarding {...common} onDone={settle} />
  if (state.household.needs_reconnect) return <Reconnect household={state.household} {...common} onDone={settle} />
  return children(state.household, refresh, state.operator)
}

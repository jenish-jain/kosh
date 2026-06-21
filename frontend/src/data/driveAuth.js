// Short-lived Drive access tokens via Google Identity Services (GIS).
// We request the narrow `drive.file` scope — Kosh can only see files it
// creates, never the rest of the user's Drive — and cache the token in
// memory until it's close to expiry. The first request per session shows a
// one-time Google consent prompt; subsequent ones are silent.
//
// IMPORTANT: `requestAccessToken()` opens a popup, and browsers only allow
// that when it's called synchronously inside a user gesture (e.g. a click
// handler). Any `await` before it — even one microtask — can cause the
// popup to be silently blocked, leaving the promise hanging forever. So
// `requestDriveAccessToken` must be invoked directly from an onClick, not
// from inside an async chain.

const SCOPE = 'https://www.googleapis.com/auth/drive.file'

// Factory that creates an isolated drive-auth client.
// In production: createDriveAuth() — uses window.google.accounts.oauth2
// In tests: createDriveAuth(fakeGoogleAccounts) — injectable dependency
export function createDriveAuth(googleAccounts = null) {
  let tokenClient = null
  let cachedToken = null
  let cachedExpiry = 0
  let inFlight = null

  function getGA() {
    return googleAccounts ?? window.google?.accounts?.oauth2
  }

  function isFresh() {
    return cachedToken && Date.now() < cachedExpiry - 60_000
  }

  function ensureTokenClient(clientId) {
    if (tokenClient || !getGA()) return tokenClient
    tokenClient = getGA().initTokenClient({
      client_id: clientId,
      scope: SCOPE,
      callback: () => {},
    })
    return tokenClient
  }

  // Call this directly from a click handler (no awaits beforehand) so the
  // browser treats the resulting popup as user-initiated. Returns a promise
  // that resolves with the access token; safe to fire-and-forget.
  function requestDriveAccessToken(clientId) {
    if (!clientId) return Promise.reject(new Error('Google sign-in is not configured'))
    if (isFresh()) return Promise.resolve(cachedToken)
    if (inFlight) return inFlight

    const client = ensureTokenClient(clientId)
    if (!client) return Promise.reject(new Error('Google sign-in is not ready yet'))

    inFlight = new Promise((resolve, reject) => {
      client.callback = (resp) => {
        inFlight = null
        if (resp.error) {
          reject(new Error(resp.error_description || resp.error))
          return
        }
        cachedToken = resp.access_token
        cachedExpiry = Date.now() + (resp.expires_in || 3600) * 1000
        resolve(cachedToken)
      }
      client.requestAccessToken({ prompt: cachedToken ? '' : 'consent' })
    })
    return inFlight
  }

  // Returns the token if one is already cached or a request is already in
  // flight (started by a prior `requestDriveAccessToken` call from a click).
  // Never opens a popup, so it's safe to call from async chains.
  function getDriveAccessToken() {
    if (isFresh()) return Promise.resolve(cachedToken)
    if (inFlight) return inFlight
    return Promise.resolve('')
  }

  return { requestDriveAccessToken, getDriveAccessToken }
}

// Default singleton for production use
export const driveAuth = createDriveAuth()

// Named exports from singleton for backwards-compatible imports
export const { requestDriveAccessToken, getDriveAccessToken } = driveAuth

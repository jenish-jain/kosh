// Short-lived Drive access tokens via Google Identity Services (GIS).
// We request the narrow `drive.file` scope — Kosh can only see files it
// creates, never the rest of the user's Drive — and cache the token in
// memory until it's close to expiry. The first request per session shows a
// one-time Google consent prompt; subsequent ones are silent.

const SCOPE = 'https://www.googleapis.com/auth/drive.file'

let tokenClient = null
let cachedToken = null
let cachedExpiry = 0

function waitForGis() {
  return new Promise(resolve => {
    if (window.google?.accounts?.oauth2) { resolve(); return }
    const iv = setInterval(() => {
      if (window.google?.accounts?.oauth2) { clearInterval(iv); resolve() }
    }, 80)
  })
}

export async function getDriveAccessToken(clientId) {
  if (!clientId) throw new Error('Google sign-in is not configured')

  const now = Date.now()
  if (cachedToken && now < cachedExpiry - 60_000) return cachedToken

  await waitForGis()
  if (!tokenClient) {
    tokenClient = window.google.accounts.oauth2.initTokenClient({
      client_id: clientId,
      scope: SCOPE,
      callback: () => {},
    })
  }

  return new Promise((resolve, reject) => {
    tokenClient.callback = (resp) => {
      if (resp.error) {
        reject(new Error(resp.error_description || resp.error))
        return
      }
      cachedToken = resp.access_token
      cachedExpiry = Date.now() + (resp.expires_in || 3600) * 1000
      resolve(cachedToken)
    }
    // Silent refresh once the user has granted access at least once this session.
    tokenClient.requestAccessToken({ prompt: cachedToken ? '' : 'consent' })
  })
}

// Hosted-mode API (/api/hosted/*): the signed-in account's household,
// onboarding, Drive reconnect, and membership.

async function request(path, opts = {}) {
  const res = await fetch(path, { credentials: 'include', ...opts })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    const err = new Error(body.trim() || `Server error ${res.status}`)
    err.status = res.status
    throw err
  }
  return res.json()
}

const send = (method, path, body) => request(path, {
  method,
  headers: { 'Content-Type': 'application/json' },
  body: body ? JSON.stringify(body) : undefined,
})

export const hostedMe        = () => request('/api/hosted/me')
export const createHousehold = (code, name) => send('POST', '/api/hosted/households', { code, name })
export const reconnectDrive  = code => send('POST', '/api/hosted/reconnect', { code })
export const getHousehold    = () => request('/api/hosted/household')
export const inviteMember    = email => send('POST', '/api/hosted/invites', { email })
export const cancelInvite    = id => send('DELETE', `/api/hosted/invites/${encodeURIComponent(id)}`)
export const acceptInvite    = token => send('POST', '/api/hosted/invites/accept', { token })
export const removeMember    = email => send('DELETE', `/api/hosted/members/${encodeURIComponent(email)}`)
export const leaveHousehold  = () => send('POST', '/api/hosted/leave')
export const deleteHousehold = confirm => send('DELETE', '/api/hosted/household', { confirm })
export const getUsage        = () => request('/api/hosted/usage')

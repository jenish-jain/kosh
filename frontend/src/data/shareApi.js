// Family-share API — owner management (/api/shares, needs a Kosh session)
// and recipient access (/api/share/*, gated by link token + TOTP code).

async function request(path, opts = {}) {
  const res = await fetch(path, { credentials: 'include', ...opts })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    const err = new Error(body.trim() || `Server error ${res.status}`)
    err.status = res.status
    throw err
  }
  return res.headers.get('Content-Type')?.includes('application/json') ? res.json() : res
}

const post = (path, body) => request(path, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: body ? JSON.stringify(body) : undefined,
})

// ── Owner ──
export const listShares    = () => request('/api/shares')
export const createShare   = recipient => post('/api/shares', recipient)
export const resetShare    = id => post(`/api/shares/${id}/reset`)
export const revokeShare   = id => post(`/api/shares/${id}/revoke`)
export const deleteShare   = id => request(`/api/shares/${id}`, { method: 'DELETE' })
export const shareAccessLog = id => request(`/api/shares/${id}/access`)

// ── Recipient ──
export const verifyShare  = (token, code) => post('/api/share/verify', { token, code })
export const fetchSharedReport = token => request(`/api/share/report?token=${encodeURIComponent(token)}`)
export const lockShare    = () => post('/api/share/logout')
export const sharedPdfUrl = token => `/api/share/export?token=${encodeURIComponent(token)}`

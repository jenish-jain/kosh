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
// The link token travels in a header, never a URL, so it can't end up in
// server or proxy logs.
const TOKEN_HEADER = 'X-Kosh-Share-Token'
const withToken = (token, opts = {}) => ({ ...opts, headers: { ...(opts.headers || {}), [TOKEN_HEADER]: token } })

export const verifyShare = (token, code) => request('/api/share/verify', withToken(token, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ code }),
}))
export const fetchSharedReport = token => request('/api/share/report', withToken(token))
export const lockShare = () => post('/api/share/logout')

export async function downloadSharedPdf(token) {
  const res = await request('/api/share/export', withToken(token))
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `Kosh-Family-Report-${new Date().toISOString().slice(0, 10)}.pdf`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

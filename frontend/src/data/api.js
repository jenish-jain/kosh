// API client — talks to Go backend at /api/*
// In dev mode the Go server returns dev_data.json

const BASE = '/api'

let _cache = null

export async function fetchData() {
  const res = await fetch(`${BASE}/data`)
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body.trim() || `Server error ${res.status}`)
  }
  _cache = await res.json()
  return _cache
}

export function getCached() {
  return _cache
}

export async function addRow(sheet, data) {
  const res = await fetch(`${BASE}/${sheet}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Add failed: ${res.status}`)
  return res.json()
}

export async function updateRow(sheet, id, patch) {
  const res = await fetch(`${BASE}/${sheet}/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) throw new Error(`Update failed: ${res.status}`)
  return res.json()
}

export async function deleteRow(sheet, id) {
  const res = await fetch(`${BASE}/${sheet}/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`Delete failed: ${res.status}`)
  return res.json()
}

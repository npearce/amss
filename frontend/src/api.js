// API client for the AMSS BFF.
// In dev, Vite proxies /api/v1/* to localhost:8080.
// In production/k8s, set VITE_API_URL to the BFF base URL (no trailing slash, no /api/v1).
const BASE = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')
const API = `${BASE}/api/v1`

async function apiFetch(path, options = {}) {
  const res = await fetch(`${API}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const body = await res.json()
  if (body.error) throw new Error(body.error.message || body.error.code)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return body.data
}

function qs(params) {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') p.set(k, String(v))
  }
  const s = p.toString()
  return s ? `?${s}` : ''
}

// ─── Crew ────────────────────────────────────────────────────
export const fetchCrew = (params = {}) =>
  apiFetch(`/crew${qs(params)}`)

export const fetchCrewMember = (id) =>
  apiFetch(`/crew/${id}`)

export const fetchCrewActivity = (id, params = {}) =>
  apiFetch(`/crew/${id}/activity${qs(params)}`)

export const fetchConversations = (params = {}) =>
  apiFetch(`/conversations${qs(params)}`)

// ─── Knowledge Base ──────────────────────────────────────────
// BFF maps /api/v1/kb → kb-store /articles
export const fetchArticles = (params = {}) =>
  apiFetch(`/kb${qs(params)}`)

export const fetchArticle = (id) =>
  apiFetch(`/kb/${id}`)

// ─── Tickets ─────────────────────────────────────────────────
export const fetchTickets = (params = {}) =>
  apiFetch(`/tickets${qs(params)}`)

export const fetchTicket = (id) =>
  apiFetch(`/tickets/${id}`)

// ─── Agent endpoints ──────────────────────────────────────────
export const sendChat = (payload) =>
  apiFetch('/chat', { method: 'POST', body: JSON.stringify(payload) })

export const runCurate = () =>
  apiFetch('/curator', { method: 'POST', body: JSON.stringify({}) })

// ─── Admin ───────────────────────────────────────────────────
export const resetStores = () =>
  apiFetch('/reset', { method: 'POST' })

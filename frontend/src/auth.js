// Keycloak ROPC auth module. Stores the token in memory (not localStorage).
//
// Three modes:
//   enabled  — VITE_KEYCLOAK_URL + CLIENT_ID + CLIENT_SECRET all set
//   disabled — any var missing; isAuthenticated() always true, no token sent
//
// Auto-refresh runs 30 seconds before expiry using the same credentials.

let _token = null
let _expiry = 0
let _credentials = null
let _refreshTimer = null

const KC_URL = import.meta.env.VITE_KEYCLOAK_URL || ''
const KC_CLIENT_ID = import.meta.env.VITE_KEYCLOAK_CLIENT_ID || ''
const KC_CLIENT_SECRET = import.meta.env.VITE_KEYCLOAK_CLIENT_SECRET || ''

export const keycloakEnabled = !!(KC_URL && KC_CLIENT_ID && KC_CLIENT_SECRET)

function parseExpiry(token) {
  try {
    const part = token.split('.')[1]
    const padded = part + '='.repeat((4 - (part.length % 4)) % 4)
    const decoded = JSON.parse(atob(padded.replace(/-/g, '+').replace(/_/g, '/')))
    return decoded.exp ? decoded.exp * 1000 : Date.now() + 3_600_000
  } catch {
    return Date.now() + 3_600_000
  }
}

function scheduleRefresh(expiryMs) {
  if (_refreshTimer) clearTimeout(_refreshTimer)
  const delay = expiryMs - Date.now() - 30_000
  if (delay > 0 && _credentials) {
    _refreshTimer = setTimeout(() => {
      _doLogin(_credentials.username, _credentials.password).catch(() => {})
    }, delay)
  }
}

async function _doLogin(username, password) {
  const res = await fetch(`${KC_URL}/realms/master/protocol/openid-connect/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      client_id: KC_CLIENT_ID,
      client_secret: KC_CLIENT_SECRET,
      username,
      password,
      grant_type: 'password',
    }),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const data = await res.json()
  if (!data.access_token) throw new Error('No access token in response')
  _token = data.access_token
  _expiry = parseExpiry(_token)
  scheduleRefresh(_expiry)
}

export async function login(username, password) {
  _credentials = { username, password }
  await _doLogin(username, password)
}

export function getToken() {
  return _token || ''
}

export function isAuthenticated() {
  if (!keycloakEnabled) return true
  return !!_token && Date.now() < _expiry
}

export function logout() {
  _token = null
  _expiry = 0
  _credentials = null
  if (_refreshTimer) {
    clearTimeout(_refreshTimer)
    _refreshTimer = null
  }
}

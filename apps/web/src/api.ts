import { clearAccessToken, getAccessToken, setAccessToken } from './session'

const configuredBase = import.meta.env.VITE_API_BASE_URL as string | undefined
export const apiBaseUrl = (configuredBase || '/api/v1').replace(/\/$/, '')

export type ApiError = { code?: string; detail?: string; title?: string }
export type SessionRefresh = { access_token: string }

type SessionBroadcast =
  | { type: 'session-refreshed'; accessToken: string }
  | { type: 'session-expired' }

let sessionChannel: BroadcastChannel | null = null

function getSessionChannel() {
  if (typeof BroadcastChannel === 'undefined') return null
  if (sessionChannel) return sessionChannel
  sessionChannel = new BroadcastChannel('atethat-session')
  sessionChannel.addEventListener('message', (event: MessageEvent<SessionBroadcast>) => {
    if (event.data?.type === 'session-refreshed') setAccessToken(event.data.accessToken)
    if (event.data?.type === 'session-expired') expireSession(false)
  })
  return sessionChannel
}

// A feature page should never be responsible for interpreting an expired
// session. It only knows that its request cannot continue; the app shell owns
// the redirect and shows the sign-in flow.
function expireSession(broadcast: boolean) {
  clearAccessToken()
  if (typeof window !== 'undefined') window.dispatchEvent(new Event('atethat:session-expired'))
  if (broadcast) getSessionChannel()?.postMessage({ type: 'session-expired' } satisfies SessionBroadcast)
}

export async function refreshAccessToken(): Promise<SessionRefresh> {
  const channel = getSessionChannel()
  const refresh = async (): Promise<SessionRefresh> => {
    const existing = getAccessToken()
    if (existing) return { access_token: existing }
    const response = await fetch(`${apiBaseUrl}/auth/refresh`, { method: 'POST', credentials: 'include' })
    if (!response.ok) {
      expireSession(true)
      throw new Error('Session expired')
    }
    const session = await response.json() as SessionRefresh
    setAccessToken(session.access_token)
    channel?.postMessage({ type: 'session-refreshed', accessToken: session.access_token } satisfies SessionBroadcast)
    return session
  }
  const locks = typeof navigator === 'undefined' ? undefined : navigator.locks
  // All open AteThat tabs share this browser lock. The waiting tab receives the
  // fresh in-memory token through BroadcastChannel and therefore never reuses
  // the refresh token that the lock holder has just rotated.
  if (locks) return locks.request('atethat-refresh-session', { mode: 'exclusive' }, refresh)
  return refresh()
}

async function requestJson<T>(path: string, method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE', body?: object, authenticated = false, retried = false, extraHeaders?: Record<string, string>): Promise<T> {
  const token = authenticated ? getAccessToken() : null
  const response = await fetch(`${apiBaseUrl}${path}`, {
    method,
    credentials: 'include',
    headers: {
      ...(body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...extraHeaders,
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  })
  if (response.status === 401 && authenticated) {
    if (!retried) {
      // Do not reuse the very access token the server just rejected. A different
      // concurrent request may already have installed a fresh token; preserve it.
      if (getAccessToken() === token) clearAccessToken()
      await refreshAccessToken()
      return requestJson(path, method, body, true, true, extraHeaders)
    }
    expireSession(true)
  }
  if (!response.ok) {
    let error: ApiError = {}
    try { error = (await response.json()) as ApiError } catch { /* Response may have no JSON body. */ }
    throw new Error(error.detail || error.title || `Request failed (${response.status})`)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export function postJson<T>(path: string, body: object): Promise<T> { return requestJson(path, 'POST', body) }
export function getJson<T>(path: string): Promise<T> { return requestJson(path, 'GET') }
export function postAuthenticatedJson<T>(path: string, body: object): Promise<T> { return requestJson(path, 'POST', body, true) }
export function getAuthenticatedJson<T>(path: string): Promise<T> { return requestJson(path, 'GET', undefined, true) }
export function patchAuthenticatedJson<T>(path: string, body: object): Promise<T> { return requestJson(path, 'PATCH', body, true) }
export function putAuthenticatedJson<T>(path: string, body: object): Promise<T> { return requestJson(path, 'PUT', body, true) }
export function deleteAuthenticatedJson<T>(path: string): Promise<T> { return requestJson(path, 'DELETE', undefined, true) }
export function postAuthenticatedIdempotentJson<T>(path: string, body: object, key: string): Promise<T> { return requestJson(path, 'POST', body, true, false, { 'Idempotency-Key': key }) }

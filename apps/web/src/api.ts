import { clearAccessToken, getAccessToken, setAccessToken } from './session'

const configuredBase = import.meta.env.VITE_API_BASE_URL as string | undefined
export const apiBaseUrl = (configuredBase || '/api/v1').replace(/\/$/, '')

export type ApiError = { code?: string; detail?: string; title?: string }

// A feature page should never be responsible for interpreting an expired
// session. It only knows that its request cannot continue; the app shell owns
// the redirect and shows the sign-in flow.
function notifySessionExpired() {
  clearAccessToken()
  if (typeof window !== 'undefined') window.dispatchEvent(new Event('atethat:session-expired'))
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
      const refresh = await fetch(`${apiBaseUrl}/auth/refresh`, { method: 'POST', credentials: 'include' })
      if (refresh.ok) {
        const session = await refresh.json() as { access_token: string }
        setAccessToken(session.access_token)
        return requestJson(path, method, body, true, true, extraHeaders)
      }
    }
    notifySessionExpired()
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

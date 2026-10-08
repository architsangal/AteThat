import { clearAccessToken, getAccessToken, setAccessToken } from './session'

const configuredBase = import.meta.env.VITE_API_BASE_URL as string | undefined
export const apiBaseUrl = (configuredBase || 'http://localhost:18080/api/v1').replace(/\/$/, '')

export type ApiError = { code?: string; detail?: string; title?: string }

async function requestJson<T>(path: string, method: 'GET' | 'POST' | 'PATCH', body?: object, authenticated = false, retried = false): Promise<T> {
  const token = authenticated ? getAccessToken() : null
  const response = await fetch(`${apiBaseUrl}${path}`, {
    method,
    credentials: 'include',
    headers: {
      ...(body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  })
  if (response.status === 401 && authenticated && !retried) {
    const refresh = await fetch(`${apiBaseUrl}/auth/refresh`, { method: 'POST', credentials: 'include' })
    if (refresh.ok) {
      const session = await refresh.json() as { access_token: string }
      setAccessToken(session.access_token)
      return requestJson(path, method, body, true, true)
    }
    clearAccessToken()
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
export function getAuthenticatedJson<T>(path: string): Promise<T> { return requestJson(path, 'GET', undefined, true) }
export function patchAuthenticatedJson<T>(path: string, body: object): Promise<T> { return requestJson(path, 'PATCH', body, true) }

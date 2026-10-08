// Access tokens stay in memory. The refresh token belongs only in the API's HttpOnly cookie.
let accessToken: string | null = null

export function setAccessToken(token: string) { accessToken = token }
export function getAccessToken() { return accessToken }
export function clearAccessToken() { accessToken = null }

import assert from 'node:assert/strict'
import { test } from 'node:test'
import { build } from 'esbuild'

test('401 refresh discards only the rejected token, then retries with a fresh token', async () => {
  const { outputFiles } = await build({ stdin: { contents: "export * from './api'; export * from './session'", resolveDir: new URL('../', import.meta.url).pathname, loader: 'ts' }, bundle: true, format: 'esm', platform: 'node', write: false, define: { 'import.meta.env.VITE_API_BASE_URL': '"/api/v1"' } })
  const api = await import(`data:text/javascript;base64,${Buffer.from(outputFiles[0].text).toString('base64')}`)
  const savedFetch = globalThis.fetch, savedChannel = globalThis.BroadcastChannel
  globalThis.BroadcastChannel = undefined
  try {
    const calls = []
    api.setAccessToken('expired-local-fixture')
    globalThis.fetch = async (url, options) => {
      calls.push([url, options.headers?.Authorization])
      if (url.endsWith('/auth/refresh')) return Response.json({ access_token: 'fresh-local-fixture' })
      return options.headers?.Authorization === 'Bearer expired-local-fixture' ? Response.json({}, { status: 401 }) : Response.json({ ok: true })
    }
    assert.deepEqual(await api.getAuthenticatedJson('/example'), { ok: true })
    assert.deepEqual(calls.map(c => c[0]), ['/api/v1/example','/api/v1/auth/refresh','/api/v1/example'])
    assert.equal(calls[2][1], 'Bearer fresh-local-fixture')
    api.setAccessToken('expired-local-fixture'); calls.length = 0
    globalThis.fetch = async (url, options) => {
      calls.push(url)
      if (options.headers?.Authorization === 'Bearer expired-local-fixture') { api.setAccessToken('newer-concurrent-token'); return Response.json({}, {status:401}) }
      assert.equal(options.headers.Authorization,'Bearer newer-concurrent-token')
      return Response.json({ok:true})
    }
    await api.getAuthenticatedJson('/example')
    assert.equal(calls.length,2, 'concurrent fresh token must not be cleared or refreshed again')
  } finally { globalThis.fetch = savedFetch; globalThis.BroadcastChannel = savedChannel; api.clearAccessToken() }
})

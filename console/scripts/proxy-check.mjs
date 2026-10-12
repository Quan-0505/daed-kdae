import assert from 'node:assert/strict'
import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
const root = fileURLToPath(new URL('../../', import.meta.url))
let disconnected = false
const upstream = createServer(async (req, res) => {
  if (req.url.startsWith('/api/events/logs')) {
    res.writeHead(200, { 'content-type': 'text/event-stream' })
    res.write('event: log.entry\ndata: {"id":1}\n\n')
    const timer = setInterval(() => res.write(': heartbeat\n\n'), 50)
    res.on('close', () => { clearInterval(timer); disconnected = true }); return
  }
  const chunks = []; for await (const chunk of req) chunks.push(chunk)
  res.writeHead(200, { 'content-type': 'application/json', 'content-disposition': 'attachment; filename=fixture.json' })
  res.end(JSON.stringify({ path: req.url, auth: req.headers.authorization, contentType: req.headers['content-type'], body: Buffer.concat(chunks).toString() }))
})
await new Promise((resolve) => upstream.listen(0, '127.0.0.1', resolve))
const reservation = createServer(); await new Promise((resolve) => reservation.listen(0, '127.0.0.1', resolve)); const port = reservation.address().port; await new Promise((resolve) => reservation.close(resolve))
const child = spawn(process.execPath, ['serve-console.js'], { cwd: root, windowsHide: true, stdio: 'ignore', env: { ...process.env, PORT: String(port), DAED_BACKEND_URL: `http://127.0.0.1:${upstream.address().port}/graphql` } })
try {
  const base = `http://127.0.0.1:${port}`
  for (let i = 0; i < 100; i++) { try { await fetch(base + '/graphql'); break } catch { await new Promise((resolve) => setTimeout(resolve, 20)) } }
  const response = await fetch(base + '/api/config/dae?preview=true', { method: 'POST', headers: { Authorization: 'Bearer fixture', 'Content-Type': 'text/plain' }, body: 'global {}' })
  const result = await response.json(); assert.equal(result.path, '/api/config/dae?preview=true'); assert.equal(result.auth, 'Bearer fixture'); assert.equal(result.contentType, 'text/plain'); assert.equal(result.body, 'global {}'); assert(response.headers.get('content-disposition').includes('fixture'))
  const controller = new AbortController(), stream = await fetch(base + '/api/events/logs', { signal: controller.signal })
  const reader = stream.body.getReader(), data = await reader.read(); assert(new TextDecoder().decode(data.value).includes('log.entry')); controller.abort(); await reader.cancel().catch(() => {})
  for (let i = 0; i < 100 && !disconnected; i++) await new Promise((resolve) => setTimeout(resolve, 20))
  assert(disconnected, 'SSE upstream leaked after disconnect')
  console.log('Native API proxy, authorization, dae body, download headers and SSE disconnect: passed')
} finally { child.kill(); upstream.closeAllConnections(); await new Promise((resolve) => upstream.close(resolve)) }

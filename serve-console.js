import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import { dirname, extname, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import { Readable } from 'node:stream'
import { pipeline } from 'node:stream/promises'
const root = resolve(dirname(fileURLToPath(import.meta.url)), 'dist')
const mime = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.css': 'text/css', '.webp': 'image/webp', '.woff': 'font/woff', '.woff2': 'font/woff2', '.svg': 'image/svg+xml', '.json': 'application/json' }
const backend = process.env.DAED_BACKEND_URL
createServer(async (req, res) => {
  try {
    const incoming = new URL(req.url, 'http://localhost')
    const path = incoming.pathname
    if (path === '/graphql' || path.startsWith('/api/')) {
      if (!backend) { res.writeHead(503, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ errors: [{ message: 'Set DAED_BACKEND_URL to your dae-wing GraphQL endpoint.' }] })); return }
      const chunks = []
      let bytes = 0
      for await (const chunk of req) { bytes += chunk.length; if (bytes > 10 * 1024 * 1024) { res.writeHead(413); res.end('Request too large'); return }; chunks.push(chunk) }
      const target = new URL(backend)
      if (path.startsWith('/api/')) target.pathname = target.pathname.replace(/\/graphql\/?$/, '').replace(/\/$/, '') + path
      target.search = incoming.search
      const abort = new AbortController(), timeout = setTimeout(() => abort.abort(), 15000)
      const disconnect = () => abort.abort()
      res.on('close', disconnect)
      try {
        const upstream = await fetch(target, { method: req.method, headers: { 'content-type': req.headers['content-type'] ?? 'application/json', authorization: req.headers.authorization ?? '', accept: req.headers.accept ?? '*/*' }, body: ['GET', 'HEAD'].includes(req.method) ? undefined : Buffer.concat(chunks), signal: abort.signal })
        clearTimeout(timeout)
        const headers = { 'Content-Type': upstream.headers.get('content-type') ?? 'application/json', 'Cache-Control': 'no-store' }
        if (upstream.headers.has('content-disposition')) headers['Content-Disposition'] = upstream.headers.get('content-disposition')
        res.writeHead(upstream.status, headers)
        if (upstream.body) await pipeline(Readable.fromWeb(upstream.body), res)
        else res.end()
      } finally { clearTimeout(timeout); res.off('close', disconnect) }
      return
    }
    const file = resolve(root, `.${decodeURIComponent(path === '/' ? '/index.html' : path)}`)
    if (!file.startsWith(root + sep)) { res.writeHead(403); res.end(); return }
    try { const content = await readFile(file); res.writeHead(200, { 'Content-Type': mime[extname(file)] ?? 'application/octet-stream' }); res.end(content) }
    catch { res.writeHead(404); res.end('Not found') }
  } catch { if (!res.headersSent) { res.writeHead(502); res.end('Backend unavailable') } else res.destroy() }
}).listen(Number(process.env.PORT ?? 3000), '127.0.0.1', () => console.log(`Console: http://127.0.0.1:${process.env.PORT ?? 3000}`))

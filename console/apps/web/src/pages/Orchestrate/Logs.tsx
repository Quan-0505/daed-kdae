import { useStore } from '@nanostores/react'
import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { RefreshCw, Settings2, Trash2 } from 'lucide-react'
import { endpointURLAtom, tokenAtom } from '~/store'
import { Button } from '~/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '~/components/ui/dialog'

type LogEntry = { id: number; ts: string; level: string; message: string; fields?: Record<string, string> }
type Limits = { maxEntries: number; maxBytes: number }
const levels = ['error', 'warn', 'info', 'debug', 'trace']
const names: Record<string, string> = { all: '全部', error: '错误', warn: '警告', info: '信息', debug: '调试', trace: '跟踪' }

export function Logs() {
  const endpoint = useStore(endpointURLAtom), token = useStore(tokenAtom)
  const url = new URL(endpoint || '/graphql', location.origin)
  url.pathname = url.pathname.replace(/\/graphql\/?$/, '').replace(/\/$/, '') + '/api'
  url.search = ''; url.hash = ''
  const base = url.toString().replace(/\/$/, '')
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [filter, setFilter] = useState('all'), [draft, setDraft] = useState(''), [search, setSearch] = useState('')
  const [level, setLevel] = useState('error'), [follow, setFollow] = useState(true)
  const [status, setStatus] = useState('连接中'), [error, setError] = useState(''), [refresh, setRefresh] = useState(0)
  const [settingsOpen, setSettingsOpen] = useState(false), [limits, setLimits] = useState<Limits>({ maxEntries: 500, maxBytes: 5 * 1024 * 1024 })
  const [count, setCount] = useState('500'), [mb, setMb] = useState('5'), [busy, setBusy] = useState(false)
  const viewport = useRef<HTMLDivElement>(null), nearBottom = useRef(true)
  const api = async (path: string, options: RequestInit = {}) => {
    const response = await fetch(base + path, { ...options, headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers } })
    if (!response.ok) throw new Error(response.status === 401 ? '登录已过期，请重新登录' : response.status === 404 ? '当前后端尚未提供日志接口' : `请求失败（${response.status}）`)
    if (!response.headers.get("content-type")?.includes("application/json")) throw new Error("当前后端尚未提供日志接口")
    return response.json()
  }
  useEffect(() => { const timer = setTimeout(() => setSearch(draft.trim()), 300); return () => clearTimeout(timer) }, [draft])
  useEffect(() => {
    const abort = new AbortController()
    setError('')
    void Promise.all([api('/logs/settings', { signal: abort.signal }), api('/runtime/log-level', { signal: abort.signal })])
      .then(([settings, runtime]) => { setLimits(settings); setCount(String(settings.maxEntries)); setMb(String(settings.maxBytes / 1024 / 1024)); setLevel(runtime.level) })
      .catch((e) => { if (!abort.signal.aborted) setError(e.message) })
    return () => abort.abort()
  }, [base, token])
  useEffect(() => {
    const abort = new AbortController(); const query = new URLSearchParams({ level: filter, q: search, limit: '500' })
    let attempts = 0
    setEntries([]); setError(''); setStatus('连接中'); nearBottom.current = true
    const run = async () => {
      while (!abort.signal.aborted) {
        try {
          const snapshot = await api('/logs?' + query, { signal: abort.signal })
          if (abort.signal.aborted) return
          setEntries(snapshot.items); let cursor: number = snapshot.cursor
          const response = await fetch(`${base}/events/logs?${query}&after_id=${cursor}&generation=${snapshot.generation}`, { headers: { Authorization: `Bearer ${token}`, Accept: 'text/event-stream' }, signal: abort.signal })
          if (!response.ok || !response.body) throw new Error(`日志实时连接失败（${response.status}）`)
          setStatus('实时'); setError(''); attempts = 0
          const reader = response.body.getReader(), decoder = new TextDecoder(); let buffer = ''
          try {
            while (!abort.signal.aborted) {
              const { value, done } = await reader.read(); if (done) throw new Error('实时连接中断')
              buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n')
              let index: number
              while ((index = buffer.indexOf('\n\n')) >= 0) {
                const event = buffer.slice(0, index); buffer = buffer.slice(index + 2)
                const data = event.split('\n').filter((line) => line.startsWith('data:')).map((line) => line.slice(5).trimStart()).join('\n')
                if (event.includes('event: log.reset')) { setRefresh((n) => n + 1); return }
                if (event.includes('event: log.entry') && data) {
                  const entry: LogEntry = JSON.parse(data)
                  if (entry.id > cursor) { cursor = entry.id; setEntries((old) => [...old.filter((item) => item.id !== entry.id), entry].slice(-500)) }
                }
              }
              if (buffer.length > 128 * 1024) throw new Error('日志流格式错误')
            }
          } finally { await reader.cancel().catch(() => undefined); reader.releaseLock() }
        } catch (e) {
          if (abort.signal.aborted) return
          setStatus('正在重连'); setError(e instanceof Error ? e.message : '日志连接失败')
          if (e instanceof Error && /过期|尚未提供/.test(e.message)) { setStatus('未连接'); return }
          await new Promise<void>((resolve) => { const timer = setTimeout(finish, Math.min(30000, 1500 * 2 ** attempts++)); function finish() { clearTimeout(timer); abort.signal.removeEventListener('abort', finish); resolve() }; abort.signal.addEventListener('abort', finish, { once: true }) })
        }
      }
    }
    void run(); return () => abort.abort()
  }, [base, token, filter, search, refresh])
  useLayoutEffect(() => { if (follow && nearBottom.current && viewport.current) viewport.current.scrollTop = viewport.current.scrollHeight }, [entries, follow])
  const mutate = async (action: () => Promise<void>) => { setBusy(true); setError(''); try { await action() } catch (e) { setError(e instanceof Error ? e.message : '操作失败') } finally { setBusy(false) } }
  return <section className="console-logs" aria-label="运行日志">
    <header className="log-toolbar">
      <h2>运行日志 <span className="console-badge">{status}</span></h2>
      <div className="log-controls">
        <label>日志等级<select aria-label="日志等级" value={level} disabled={busy} onChange={(e) => { const value = e.target.value; void mutate(async () => { const result = await api('/runtime/log-level', { method: 'PATCH', body: JSON.stringify({ level: value }) }); setLevel(result.level) }) }}>{levels.map((v) => <option key={v} value={v}>{names[v]}</option>)}</select></label>
        <label>级别过滤<select aria-label="级别过滤" value={filter} onChange={(e) => setFilter(e.target.value)}>{['all', ...levels].map((v) => <option key={v} value={v}>{names[v]}</option>)}</select></label>
        <label className="log-search">关键词<input aria-label="日志关键词" placeholder="搜索消息和字段" value={draft} maxLength={256} onChange={(e) => setDraft(e.target.value)} /></label>
        <label className="log-follow"><input type="checkbox" checked={follow} onChange={(e) => { setFollow(e.target.checked); nearBottom.current = e.target.checked }} />自动滚动</label>
        <Button variant="outline" aria-label="刷新日志" onClick={() => setRefresh((n) => n + 1)}><RefreshCw size={16} /></Button>
        <Button variant="outline" aria-label="日志设置" onClick={() => setSettingsOpen(true)}><Settings2 size={16} /></Button>
        <Button variant="outline" aria-label="清空日志" disabled={busy} onClick={() => { if (confirm('清空当前保存的运行日志？')) void mutate(async () => { await api('/logs', { method: 'DELETE' }); setEntries([]); setRefresh((n) => n + 1) }) }}><Trash2 size={16} /></Button>
      </div>
      <small>最多显示最近 500 条 · 保留 {limits.maxEntries.toLocaleString()} 条 / {Math.round(limits.maxBytes / 1024 / 1024)} MB · 日志等级立即生效，并在重启后保留</small>
    </header>
    {error && <div role="alert" className="log-error">{error}<button onClick={() => setRefresh((n) => n + 1)}>重试</button></div>}
    <div className="log-viewport" ref={viewport} onScroll={() => { const v = viewport.current; if (v) nearBottom.current = v.scrollHeight - v.scrollTop - v.clientHeight < 48 }}>
      {!entries.length && <p className="log-empty">{status === '连接中' ? '正在读取日志…' : '暂无匹配日志'}</p>}
      {entries.map((entry) => <article key={entry.id} className="log-entry" data-level={entry.level}>
        <time dateTime={entry.ts} title={new Date(entry.ts).toLocaleString()}>{new Date(entry.ts).toLocaleTimeString()}</time>
        <b>{entry.level.toUpperCase()}</b>
        <div><p>{entry.message}</p><div className="log-fields">{Object.entries(entry.fields || {}).map(([key, value]) => <span key={key}>{key}=<strong>{value || '—'}</strong></span>)}</div></div>
      </article>)}
    </div>
    <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}><DialogContent><DialogHeader><DialogTitle>日志设置</DialogTitle></DialogHeader>
      <form onSubmit={(e) => { e.preventDefault(); void mutate(async () => { const result = await api('/logs/settings', { method: 'PATCH', body: JSON.stringify({ maxEntries: Number(count), maxBytes: Number(mb) * 1024 * 1024 }) }); setLimits(result); setSettingsOpen(false); setRefresh((n) => n + 1) }) }}>
        <label className="log-setting">最大条数<input aria-label="最大条数" type="number" required min={500} max={50000} value={count} onChange={(e) => setCount(e.target.value)} /></label>
        <label className="log-setting">最大文件 MB<input aria-label="最大文件 MB" type="number" required min={5} max={200} value={mb} onChange={(e) => setMb(e.target.value)} /></label>
        <p>条数和文件大小同时生效，较小限制优先。保存后立即裁剪；日志中的订阅链接和密钥会脱敏。</p>
        <Button type="submit" disabled={busy}>保存</Button>
      </form>
    </DialogContent></Dialog>
  </section>
}

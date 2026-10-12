import { useState } from 'react'
import { useStore } from '@nanostores/react'
import { useQuery } from '@tanstack/react-query'
import { Settings, RefreshCw } from 'lucide-react'
import { nativeBase } from '~/apis/native'
import { endpointURLAtom, tokenAtom } from '~/store'
import { Switch } from '~/components/ui/switch'
import { Button } from '~/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '~/components/ui/dialog'
type Schedule = { enabled: boolean; time: string; timezone: string; nextRun: string; lastSuccess?: string; error?: string }
type Geo = { kind: string; version: string; categories: number; entries: number; source: string; useProxy?: boolean }
export function GeoData() {
  const endpoint = useStore(endpointURLAtom), token = useStore(tokenAtom)
  const [editing, setEditing] = useState<Geo | null>(null), [source, setSource] = useState(''), [useProxy,setUseProxy] = useState(false), [busy, setBusy] = useState(''), [error, setError] = useState('')
  const api = async (path: string, data?: object) => {
    const response = await fetch(nativeBase(endpoint) + path, { method: data ? 'POST' : 'GET', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: data ? JSON.stringify(data) : undefined })
    if (!response.headers.get('content-type')?.includes('application/json')) throw new Error('当前后端不支持 geodata 管理')
    const result = await response.json(); if (!response.ok) throw new Error(result.error || 'geodata 请求失败'); return result
  }
  const query = useQuery<{ items: Geo[]; schedule?: Schedule }>({ queryKey: ['native-geodata', endpoint], enabled: !!token, queryFn: () => api('/geodata'), retry: false, staleTime: 60000, refetchInterval: 60000 })
  const act = async (kind: string, fn: () => Promise<void>) => { setBusy(kind); setError(''); try { await fn(); await query.refetch() } catch (e) { setError(e instanceof Error ? e.message : '操作失败') } finally { setBusy('') } }
  return <>
    <div className="config-tiles geodata-tiles">{query.data?.items.map((item) => <div key={item.kind}><header><b>{item.kind === 'geoip' ? 'Geoip' : 'Geosite'}</b><button aria-label={`${item.kind} 更新源`} onClick={() => { setEditing(item); setSource(item.source); setUseProxy(item.useProxy ?? false); setError('') }}><Settings size={14} /></button><button disabled={!!busy} aria-label={`更新 ${item.kind}`} onClick={() => { if (!item.source) { setEditing(item); setSource(''); setUseProxy(item.useProxy ?? false); return }; void act(item.kind, async () => { await api('/geodata/update', { kind: item.kind }) }) }}><RefreshCw size={14} className={busy === item.kind ? 'animate-spin' : ''} /></button></header><small>版本 {item.version}</small><small>分类 {item.categories.toLocaleString()}</small><small>{item.kind === 'geoip' ? 'CIDR' : '规则'} {item.entries.toLocaleString()}</small></div>)}</div>
    {query.data?.schedule && <p className="geo-schedule">每天 {query.data.schedule.time}（北京时间）自动更新并重载<br />{query.data.schedule.lastSuccess ? `最近成功：${new Date(query.data.schedule.lastSuccess).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}` : '首次更新将在下个凌晨执行'}{query.data.schedule.error && <span role="alert">更新失败，已保留旧数据：{query.data.schedule.error}</span>}</p>}
    {query.isError && <p className="console-empty">Geodata 读取失败<button onClick={() => void query.refetch()}>重试</button></p>}{error && <p role="alert">{error}</p>}
    <Dialog open={!!editing} onOpenChange={(v) => { if (!v) setEditing(null) }}><DialogContent><DialogHeader><DialogTitle>{editing?.kind} 更新源</DialogTitle></DialogHeader><form onSubmit={(e) => { e.preventDefault(); if (editing) void act(editing.kind, async () => { await api('/geodata/source', { kind: editing.kind, source, useProxy }); setEditing(null) }) }}><label className="log-setting">HTTPS 下载地址<input type="url" required value={source} onChange={(e) => setSource(e.target.value)} /></label><label className="geo-proxy-toggle" htmlFor="geo-use-proxy"><Switch id="geo-use-proxy" checked={useProxy} onCheckedChange={setUseProxy} disabled={!!busy} />使用代理</label><p>开启后按当前代理路由下载，需代理引擎运行。</p><p>更新前验证文件格式；运行配置拒绝新文件时自动恢复原文件。</p><Button type="submit" disabled={!!busy}>保存</Button></form></DialogContent></Dialog>
  </>
}

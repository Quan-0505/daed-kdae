import { useEffect, useState } from 'react'
import { useStore } from '@nanostores/react'
import { useQuery } from '@tanstack/react-query'
import { Check, ListPlus, Network, Search } from 'lucide-react'
import { useGroupsQuery, useGeneralQuery, useNodesQuery, useSubscriptionsQuery } from '~/apis'
import { nativeBase } from '~/apis/native'
import { endpointURLAtom, tokenAtom } from '~/store'
import { Dialog, DialogTitle } from '~/components/ui/dialog'
import { ScrollableDialogBody, ScrollableDialogContent, ScrollableDialogFooter, ScrollableDialogHeader } from '~/components/ui/scrollable-dialog'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'

type Binding = { id: string; regex: string | null }
type Resources = { nodeIDs: string[]; subscriptions: Binding[]; revision: string }
type Preview = { items: { id: string; nodes: { id: string; name: string; protocol: string }[] }[] }

export function GroupResourceEditor({ id, name, onClose }: { id: string; name: string; onClose: () => void }) {
  const endpoint = useStore(endpointURLAtom), token = useStore(tokenAtom)
  const groups = useGroupsQuery(), general = useGeneralQuery(), nodes = useNodesQuery(), subscriptions = useSubscriptionsQuery()
  const [mode, setMode] = useState<'choose' | 'nodes' | 'subscriptions'>('choose')
  const [selected, setSelected] = useState<string[]>([]), [search, setSearch] = useState('')
  const [regex, setRegex] = useState(''), [uniform, setUniform] = useState(false), [mixed, setMixed] = useState(false)
  const [initial, setInitial] = useState<Resources | null>(null), [busy, setBusy] = useState(false), [error, setError] = useState('')
  const api = async <T,>(path: string, data?: object): Promise<T> => {
    const response = await fetch(nativeBase(endpoint) + path, { method: data ? 'POST' : 'GET', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: data ? JSON.stringify(data) : undefined })
    if (!response.headers.get('content-type')?.includes('application/json')) throw new Error('当前后端不支持群组快捷编辑，请升级后端')
    const result = await response.json(); if (!response.ok) throw new Error(result.error || '群组请求失败'); return result
  }
  const resources = useQuery<Resources>({ queryKey: ['group-resources', endpoint, id], queryFn: () => api(`/groups/resources?id=${encodeURIComponent(id)}`), retry: false, staleTime: 0 })
  useEffect(() => { if (resources.data && !resources.isFetching && !initial) setInitial(resources.data) }, [resources.data, resources.isFetching, initial])
  const choose = (next: 'nodes' | 'subscriptions') => {
    if (!initial) return
    setMode(next); setSearch(''); setError('')
    setSelected(next === 'nodes' ? initial.nodeIDs : initial.subscriptions.map(s => s.id))
    const patterns = new Set(initial.subscriptions.map(s => s.regex || ''))
    setMixed(patterns.size > 1); setRegex(patterns.size === 1 ? [...patterns][0] : ''); setUniform(false)
  }
  const bindings = selected.map(value => ({ id: value, regex: uniform ? regex.trim() || null : initial?.subscriptions.find(s => s.id === value)?.regex || null }))
  const preview = useQuery<Preview>({ queryKey: ['group-resource-preview', endpoint, id, bindings], queryFn: () => api('/groups/preview', { subscriptions: bindings }), enabled: mode === 'subscriptions' && !!initial, retry: false })
  const toggle = (value: string) => { setError(''); setSelected(current => current.includes(value) ? current.filter(v => v !== value) : [...current, value]) }
  const subs = subscriptions.data?.subscriptions ?? []
  const nodePool = [...new Map([
    ...(nodes.data?.nodes.edges ?? []).map(n => ({ id: n.id, name: n.tag || n.name || n.id, protocol: n.protocol, source: '独立节点' })),
    ...subs.flatMap(s => s.nodes.edges.map(n => ({ id: n.id, name: n.name || n.id, protocol: n.protocol, source: s.tag || '未命名订阅' }))),
  ].map(n => [n.id, n])).values()]
  const filter = (text: string) => text.toLowerCase().includes(search.trim().toLowerCase())
  const save = async () => {
    if (!initial) return
    setBusy(true); setError('')
    try {
      await api('/groups/resources', { id, revision: initial.revision, mode, ...(mode === 'nodes' ? { nodeIDs: selected } : { subscriptions: bindings }) })
      await Promise.all([groups.refetch(), general.refetch()]); onClose()
    } catch (e) { setError(e instanceof Error ? e.message : '保存失败') } finally { setBusy(false) }
  }
  const resourceError = resources.error instanceof Error ? resources.error.message : ''
  const previewError = preview.error instanceof Error ? preview.error.message : ''
  const dataUnavailable = mode === 'nodes' ? nodes.isPending || subscriptions.isPending || nodes.isError || subscriptions.isError : subscriptions.isPending || subscriptions.isError
  return <Dialog open onOpenChange={open => { if (!open && !busy) onClose() }}>
    <ScrollableDialogContent className="z-[80]" overlayClassName="z-[70]" size={mode === 'choose' ? 'md' : 'xl'} onEscapeKeyDown={e => { if (busy) e.preventDefault() }} onPointerDownOutside={e => { if (busy) e.preventDefault() }}>
      <ScrollableDialogHeader><DialogTitle>编辑 {name} 的{mode === 'choose' ? '分组资源' : mode === 'nodes' ? '节点' : '订阅组'}</DialogTitle>{mode !== 'choose' && <p className="text-sm text-muted-foreground">已选 {selected.length} 项</p>}</ScrollableDialogHeader>
      <ScrollableDialogBody className="group-resource-body">
        {mode === 'choose' ? <div className="group-resource-options">
          <button disabled={!initial} onClick={() => choose('nodes')}><ListPlus size={28} /><span><b>编辑节点</b><small>{initial?.nodeIDs.length ?? '…'} 个节点</small></span></button>
          <button disabled={!initial} onClick={() => choose('subscriptions')}><Network size={28} /><span><b>编辑订阅组</b><small>{initial?.subscriptions.length ?? '…'} 个订阅组</small></span></button>
          {resourceError && <p role="alert">{resourceError}<Button variant="ghost" onClick={() => void resources.refetch()}>重试</Button></p>}
        </div> : <>
          <Input aria-label={mode === 'nodes' ? '搜索节点' : '搜索订阅组'} placeholder={mode === 'nodes' ? '搜索节点' : '搜索订阅组'} value={search} onChange={e => setSearch(e.target.value)} icon={<Search size={16} />} />
          {dataUnavailable ? <p role="status">资源正在读取或读取失败，请重试后编辑。<Button variant="ghost" onClick={() => { void nodes.refetch(); void subscriptions.refetch() }}>重试</Button></p> : <div className={`group-resource-selections ${mode}`}>
            {mode === 'nodes' ? nodePool.filter(n => filter(`${n.name} ${n.source} ${n.protocol}`)).map(n => <button key={n.id} aria-pressed={selected.includes(n.id)} onClick={() => toggle(n.id)}><span className="resource-check">{selected.includes(n.id) && <Check size={14} />}</span><span><b>{n.name}</b><small>{n.protocol.toUpperCase()} · {n.source}</small></span></button>) : subs.filter(s => filter(s.tag || '未命名订阅')).map(s => <button key={s.id} aria-pressed={selected.includes(s.id)} onClick={() => toggle(s.id)}><span className="resource-check">{selected.includes(s.id) && <Check size={14} />}</span><b>{s.tag || '未命名订阅'}</b><small>{s.nodes.edges.length} 节点</small></button>)}
            {mode === 'nodes' && !nodePool.filter(n => filter(`${n.name} ${n.source} ${n.protocol}`)).length && <p>没有匹配的节点</p>}
            {mode === 'subscriptions' && !subs.filter(s => filter(s.tag || '未命名订阅')).length && <p>没有匹配的订阅组</p>}
          </div>}
          {mode === 'subscriptions' && <>
            <section className="group-resource-panel"><label htmlFor="group-resource-regex">节点名称正则</label><Input id="group-resource-regex" value={regex} maxLength={4096} placeholder="可选，留空则添加整个订阅组" onChange={e => { setRegex(e.target.value); setUniform(true); setError('') }} />
              <small>由 Go 后端校验并应用到所有已选订阅组，不支持前后预查和反向引用。</small>
              {mixed && !uniform && <p>已有订阅使用不同正则，当前保留各自筛选。输入正则可统一修改。<Button variant="ghost" onClick={() => { setRegex(''); setUniform(true) }}>清除全部筛选</Button></p>}
              {previewError && <p role="alert">{previewError}</p>}
            </section>
            <section className="group-resource-panel"><h3>匹配节点预览</h3><small>{preview.isFetching ? '正在由后端校验…' : selected.length ? '留空正则会添加整个订阅组；预览与后端实际筛选一致。' : '请先选择订阅组。'}</small>
              {!preview.isFetching && !preview.isError && preview.data?.items.map(item => <div key={item.id} className="group-resource-preview"><b>{subs.find(s => s.id === item.id)?.tag || '未命名订阅'}</b><small>匹配 {item.nodes.length} 个节点</small><div>{item.nodes.map(n => <span key={n.id}><em>{n.protocol.toUpperCase()}</em>{n.name}</span>)}{!item.nodes.length && <small>没有匹配的节点</small>}</div></div>)}
            </section>
          </>}
        </>}
      </ScrollableDialogBody>
      {mode !== 'choose' && <ScrollableDialogFooter>{error && <p role="alert" className="mr-auto text-sm text-destructive">{error}</p>}<Button variant="ghost" disabled={busy} onClick={onClose}>取消</Button><Button loading={busy} disabled={busy || !initial || dataUnavailable || (mode === 'subscriptions' && (preview.isPending || preview.isFetching || preview.isError))} onClick={() => void save()}>{mode === 'nodes' ? '保存节点选择' : '保存订阅组选择'}</Button></ScrollableDialogFooter>}
    </ScrollableDialogContent>
  </Dialog>
}

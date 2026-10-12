import { useState } from 'react'
import { useStore } from '@nanostores/react'
import { useQueryClient } from '@tanstack/react-query'
import { ArrowRightLeft } from 'lucide-react'
import { endpointURLAtom, tokenAtom } from '~/store'
import { nativeBase } from '~/apis/native'
import { Button } from './ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from './ui/dropdown-menu'

export function ConfigTransfer() {
  const endpoint = useStore(endpointURLAtom), token = useStore(tokenAtom), client = useQueryClient()
  const [open, setOpen] = useState(false), [error, setError] = useState(''), [busy, setBusy] = useState(false)
  const [body, setBody] = useState(''), [counts, setCounts] = useState<Record<string, number> | null>(null), [done, setDone] = useState(false), [daeImport, setDaeImport] = useState(false)
  const request = async (path: string, data?: string) => {
    const response = await fetch(nativeBase(endpoint) + path, { method: data === undefined ? 'GET' : 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: data })
    if (!response.ok) { const result = await response.json().catch(() => ({})); throw new Error(result.error || `请求失败（${response.status}）`) }
    return response
  }
  const action = async (fn: () => Promise<void>) => { setBusy(true); setError(''); try { await fn() } catch (e) { setError(e instanceof Error ? e.message : '操作失败'); setOpen(true) } finally { setBusy(false) } }
  const download = (dae: boolean) => action(async () => {
    const response = await request(dae ? '/config/dae' : '/config/bundle')
    if (!response.headers.get('content-type')?.includes(dae ? 'text/plain' : 'application/json')) throw new Error('当前后端不支持此功能')
    const url = URL.createObjectURL(await response.blob()), link = document.createElement('a')
    link.href = url; link.download = dae ? 'daed-config.dae' : 'daed-kdae-backup.json'; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000)
  })
  return <>
    <DropdownMenu><DropdownMenuTrigger asChild><Button variant="outline" disabled={busy}><ArrowRightLeft size={15} />导入/导出</Button></DropdownMenuTrigger>
      <DropdownMenuContent><DropdownMenuItem onSelect={() => void download(true)}>导出 dae 配置文件</DropdownMenuItem><DropdownMenuItem onSelect={() => void download(false)}>导出备份文件</DropdownMenuItem><DropdownMenuItem onSelect={() => { setDaeImport(true); setOpen(true); setCounts(null); setBody(''); setError(''); setDone(false) }}>导入 dae 配置文件</DropdownMenuItem><DropdownMenuItem onSelect={() => { setDaeImport(false); setOpen(true); setCounts(null); setBody(''); setError(''); setDone(false) }}>导入备份文件</DropdownMenuItem></DropdownMenuContent>
    </DropdownMenu>
    <Dialog open={open} onOpenChange={setOpen}><DialogContent><DialogHeader><DialogTitle>配置备份与导入</DialogTitle></DialogHeader>
      <p>导出的文件包含节点和订阅凭据，请妥善保存。备份不包含账号密码。</p>
      <p>导入会新增资源，保留当前运行配置；导入的订阅定时更新默认关闭。导入完成后，可在配置、DNS 和路由页选择新资源。</p>
      {daeImport && <p>dae 文件以静态节点快照导入。动态订阅、正则群组筛选和群组独立检查参数请使用 JSON 备份。</p>}
      <input type="file" accept={daeImport ? ".dae,.conf,text/plain" : ".json,application/json"} aria-label="选择备份文件" disabled={busy} onChange={(e) => { const file = e.target.files?.[0]; if (!file) return; setDone(false); setCounts(null); void action(async () => { if (file.size > 10 * 1024 * 1024) throw new Error('备份文件不能超过 10 MB'); const data = await file.text(); const response = await request(daeImport ? '/config/dae?preview=true' : '/config/bundle/preview', data); const preview = await response.json(); setBody(data); setCounts(preview.counts) }) }} />
      {counts && <div className="backup-counts">{Object.entries(counts).filter(([, count]) => count > 0).map(([name, count]) => <span key={name}>{name}: {count}</span>)}</div>}
      {error && <p role="alert">{error}</p>}{done && <p role="status">导入成功，当前运行配置已保留。</p>}
      <Button disabled={busy || !counts || done} onClick={() => void action(async () => { await request(daeImport ? '/config/dae' : '/config/bundle', body); setDone(true); setBody(''); await client.invalidateQueries() })}>{busy ? '处理中…' : '导入为新资源'}</Button>
    </DialogContent></Dialog>
  </>
}

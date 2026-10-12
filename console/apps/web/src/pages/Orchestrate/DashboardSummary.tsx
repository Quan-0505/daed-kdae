import { useState, type ReactNode } from 'react'
import { GroupResourceEditor } from './GroupResourceEditor'
import { GeoData } from './GeoData'
import { Settings, Layers, Radio, ChevronRight, ArrowRight, Pencil } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import {
  useConfigsQuery,
  useGeneralQuery,
  useGroupsQuery,
  useNodesQuery,
  useSubscriptionsQuery,
  useNodeLatenciesQuery,
  useTestNodeLatenciesMutation,
} from '~/apis'

function SummaryCard({
  title,
  description,
  icon,
  action,
  children,
}: {
  title: string
  description: string
  icon: ReactNode
  action: ReactNode
  children: ReactNode
}) {
  return (
    <section className="summary-card">
      <header>
        <div className="card-icon">{icon}</div>
        <div>
          <h2>{title}</h2>
          <p>{description}</p>
        </div>
        {action}
      </header>
      <div className="summary-body">{children}</div>
    </section>
  )
}
export function DashboardSummary() {
  const [editingGroup, setEditingGroup] = useState<{ id: string; name: string } | null>(null)
  const [, setParams] = useSearchParams()
  const open = (panel: string) => {
    setParams({ panel })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
  const { data: configs, isError: configError } = useConfigsQuery()
  const { data: general } = useGeneralQuery()
  const { data: groups, isError: groupError } = useGroupsQuery()
  const { data: nodes, isError: nodeError } = useNodesQuery()
  const { data: subs, isError: subError } = useSubscriptionsQuery()
  const { data: latencies } = useNodeLatenciesQuery(30_000)
  const probe = useTestNodeLatenciesMutation()
  const config = configs?.configs.find((c) => c.selected)
  const g = config?.global
  const interfaceValue = (names?: string[]) =>
    names?.length ? (
      names.map((name) => {
        const iface = general?.general.interfaces.find((i) => i.name === name)
        return (
          <div key={name}>
            <b>{name}</b>
            <small>{iface?.ip?.join(', ') || '—'}</small>
          </div>
        )
      })
    ) : (
      <b>—</b>
    )
  const latency = (id: string) => latencies?.find((l) => l.id === id)
  const getLatencyText = (id: string) => {
    const result = latency(id)
    return result ? (!result.alive || result.latencyMs == null || result.latencyMs < 0 ? '不可用' : `${result.latencyMs} ms`) : '未测试'
  }
  return (
    <>
    <div className="dashboard-grid">
      <SummaryCard
        title="配置"
        description="当前生效与关键参数"
        icon={<Settings size={18} />}
        action={
          <button className="console-link-button" onClick={() => open('config')}>
            设置
          </button>
        }
      >
        {config ? (
          <div className="config-tiles">
            <div>
              <small>当前配置</small>
              <b>{config.name}</b>
              <span className="console-badge">{general?.general.dae.modified ? '待应用' : '已选中'}</span>
            </div>
            <div>
              <small>透明代理端口</small>
              <b>{g?.tproxyPort}</b>
            </div>
            <div>
              <small>WAN 接口</small>
              {interfaceValue(g?.wanInterface)}
            </div>
            <div>
              <small>LAN 接口</small>
              {interfaceValue(g?.lanInterface)}
            </div>
            <div>
              <small>拨号模式</small>
              <b>{g?.dialMode}</b>
            </div>
            <div>
              <small>UDP 检查 DNS</small>
              <b className="small-value">{g?.udpCheckDns.join(', ') || '—'}</b>
            </div>
          </div>
        ) : (
          <p className="console-empty">
            {configError ? '配置读取失败，请检查后端连接' : configs ? '尚未选择配置' : '正在读取配置…'}
          </p>
        )}
        <GeoData />
      </SummaryCard>
      <SummaryCard
        title="群组"
        description="群组策略与可用节点"
        icon={<Layers size={18} />}
        action={
          <button className="console-link-button" onClick={() => open('groups')}>
            查看详情
          </button>
        }
      >
        {groups?.groups.map((group) => {
          const candidates = [...group.nodes, ...group.subscriptions.flatMap((s) => s.matchedNodes)]
          const unique = [...new Map(candidates.map((n) => [n.id, n])).values()]
          return (
            <article key={group.id} className="group-summary">
              <button className="group-edit-button" aria-label={`编辑 ${group.name} 的分组资源`} onClick={() => setEditingGroup({id: group.id, name: group.name})}><Pencil size={16} /></button>
              <small>当前分组 · 匹配 {unique.length} 个节点</small>
              <div>
                <b>{group.name}</b>
                <ArrowRight size={14} />
                <span>
                  {unique.length
                    ? unique
                        .map((n) => n.name || n.tag || n.id)
                        .slice(0, 2)
                        .join(' / ')
                    : '尚未绑定节点'}
                </span>
              </div>
              <footer>
                <span>策略 {group.policy}</span>
                <span>{group.subscriptions.length} 个订阅</span>
              </footer>
            </article>
          )
        })}
        {!groups?.groups.length && (
          <p className="console-empty">
            {groupError ? '群组读取失败，请检查后端接口' : groups ? '暂无群组，点击查看详情创建' : '正在读取群组…'}
          </p>
        )}
      </SummaryCard>
      <SummaryCard
        title="节点与订阅"
        description="节点测速排行与订阅状态"
        icon={<Radio size={18} />}
        action={
          <button className="console-link-button" disabled={probe.isPending} onClick={() => probe.mutate(undefined)}>
            {probe.isPending ? '测试中…' : '延迟测试'}
          </button>
        }
      >
        <div className="node-summary-list">
          {[...(nodes?.nodes.edges ?? [])]
            .sort((a, b) => (latency(a.id)?.latencyMs ?? Infinity) - (latency(b.id)?.latencyMs ?? Infinity))
            .map((node, i) => (
              <button className="node-summary" key={node.id} onClick={() => open('nodes')}>
                <span className="node-rank">{i + 1}</span>
                <div>
                  <b>
                    <em>{node.protocol}</em>
                    {node.name || node.tag || '未命名节点'}
                  </b>
                  <small>{node.address || '订阅节点'}</small>
                </div>
                <span className="latency-tag">{getLatencyText(node.id)}</span>
              </button>
            ))}
        </div>
        {!nodes?.nodes.edges.length && (
          <p className="console-empty">{nodeError ? '节点读取失败' : nodes ? '暂无节点' : '正在读取节点…'}</p>
        )}
        {subs?.subscriptions.map((sub) => (
          <button key={sub.id} className="subscription-summary" onClick={() => open('subscriptions')}>
            <b>{sub.tag || '未命名订阅'}</b>
            <span className="console-badge">{({ fetched: '更新成功', fetch_error: '更新失败', fetching: '更新中', pending: '待更新' } as Record<string, string>)[sub.status] || (sub.status ? '状态未知' : '未更新')}</span>
            <small>
              最近更新 {sub.updatedAt ? new Date(sub.updatedAt).toLocaleDateString() : '—'} · {sub.nodes.edges.length}{' '}
              节点
            </small>
          </button>
        ))}
        {subError && <p className="console-empty">订阅读取失败</p>}
        <div className="summary-footer-links">
          <button onClick={() => open('nodes')}>
            节点
            <ChevronRight size={14} />
          </button>
          <button onClick={() => open('subscriptions')}>
            订阅
            <ChevronRight size={14} />
          </button>
        </div>
      </SummaryCard>
    </div>
    {editingGroup && <GroupResourceEditor key={editingGroup.id} {...editingGroup} onClose={() => setEditingGroup(null)} />}
    </>
  )
}

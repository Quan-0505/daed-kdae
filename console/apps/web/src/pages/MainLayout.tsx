import { useStore } from '@nanostores/react'
import { Activity, FileText, Settings, Globe, Route, Layers, Cloud, Radio, ChevronRight, Menu, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Outlet, useNavigate, useSearchParams } from 'react-router-dom'
import {
  useConfigsQuery,
  useDNSsQuery,
  useGeneralQuery,
  useGroupsQuery,
  useNodesQuery,
  useRoutingsQuery,
  useSubscriptionsQuery,
} from '~/apis'
import { HeaderWithActions } from '~/components/Header'
import { useInitialize } from '~/initialize'
import { isMockMode } from '~/mocks'
import { endpointURLAtom, tokenAtom } from '~/store'

export const consoleSections = [
  { key: 'overview', label: '总览', icon: Activity },
  { key: 'logs', label: '日志', icon: FileText },
  { key: 'config', label: '配置', icon: Settings },
  { key: 'dns', label: 'DNS', icon: Globe },
  { key: 'routing', label: '路由', icon: Route },
  { key: 'groups', label: '群组', icon: Layers },
  { key: 'nodes', label: '节点', icon: Cloud },
  { key: 'subscriptions', label: '订阅', icon: Radio },
]
export function MainLayout() {
  const navigate = useNavigate()
  const token = useStore(tokenAtom)
  const endpointURL = useStore(endpointURLAtom)
  const initialize = useInitialize()
  const [params, setParams] = useSearchParams()
  const [menuOpen, setMenuOpen] = useState(false)
  const active = consoleSections.find((s) => s.key === params.get('panel')) ?? consoleSections[0]
  const { data: general } = useGeneralQuery()
  const { data: configs } = useConfigsQuery()
  const { data: dnss } = useDNSsQuery()
  const { data: routings } = useRoutingsQuery()
  const { data: groups } = useGroupsQuery()
  const { data: nodes } = useNodesQuery()
  const { data: subscriptions } = useSubscriptionsQuery()
  const counts: Record<string, number | undefined> = {
    config: configs?.configs.length,
    dns: dnss?.dnss.length,
    routing: routings?.routings.length,
    groups: groups?.groups.length,
    nodes: nodes?.nodes.edges.length,
    subscriptions: subscriptions?.subscriptions.length,
  }
  useEffect(() => {
    if (isMockMode() || (endpointURL && token))
      void initialize().catch((error) => toast.error(`初始化失败：${String(error)}`))
  }, [endpointURL, token])
  useEffect(() => {
    if (!isMockMode() && (!endpointURL || !token)) navigate('/setup')
  }, [endpointURL, navigate, token])
  function openSection(key: string) {
    setParams(key === 'overview' ? {} : { panel: key })
    setMenuOpen(false)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
  return (
    <div className="kdae-shell">
      {menuOpen && <button className="console-scrim" aria-label="关闭导航" onClick={() => setMenuOpen(false)} />}
      <aside className={`console-sidebar ${menuOpen ? 'is-open' : ''}`} aria-label="主导航">
        <a
          className="console-brand"
          href="#"
          onClick={(e) => {
            e.preventDefault()
            openSection('overview')
          }}
        >
          <img src="./logo.webp" alt="" />
          <span>
            Daed<span className="brand-accent">Kdae</span>
            <small>NETWORK CONSOLE</small>
          </span>
        </a>
        <button className="console-close" aria-label="关闭导航" onClick={() => setMenuOpen(false)}>
          <X size={18} />
        </button>
        <nav>
          {consoleSections.map(({ key, label, icon: Icon }, index) => (
            <div key={key}>
              {(index === 0 || index === 2) && <p className="nav-heading">{index === 0 ? '监控' : '资源'}</p>}
              <button
                className={`nav-link ${active.key === key ? 'is-active' : ''}`}
                onClick={() => openSection(key)}
                aria-current={active.key === key ? 'page' : undefined}
              >
                <Icon size={17} />
                <span>{label}</span>
                {counts[key] !== undefined && <small>{String(counts[key]).padStart(2, '0')}</small>}
                {active.key === key && <ChevronRight size={14} />}
              </button>
            </div>
          ))}
        </nav>
        <div className="console-sidebar-footer">
          <p>
            <i className={general?.general.dae.running ? 'status-dot running' : 'status-dot'} />
            {general ? (general.general.dae.running ? '运行中' : '已停止') : '连接中'}
          </p>
          <div>
            <span>{general?.general.dae.version ?? '—'}</span>
            <span>GO / KDAE</span>
          </div>
        </div>
      </aside>
      <div className="console-workspace">
        <div className="console-topbar">
          <button className="console-menu" aria-label="打开导航" onClick={() => setMenuOpen(true)}>
            <Menu size={20} />
          </button>
          <HeaderWithActions />
        </div>
        <main className="console-main">
          <div className="console-page-heading">
            <div>
              <p>
                01 <span>/</span> 网络工作区
              </p>
              <h1>{active.label}</h1>
            </div>
            {active.key === 'overview' && (
              <button className="console-outline" onClick={() => openSection('routing')}>
                查看路由 <ChevronRight size={15} />
              </button>
            )}
          </div>
          {isMockMode() && (
            <div className="console-preview-note">布局预览 · 当前使用演示数据，操作不会写入 kdae 后端</div>
          )}
          <Outlet />
        </main>
      </div>
    </div>
  )
}

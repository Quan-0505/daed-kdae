import { Activity, ArrowDownLeft, ArrowUpRight } from 'lucide-react'
import { Line, LineChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { useConfigsQuery, useGeneralQuery, useNodesQuery, useSubscriptionsQuery, useTrafficOverviewQuery } from '~/apis'
import { useRuntimeMetrics } from '~/apis/native'

export function formatBytes(value: number | string | undefined) {
  if (value === undefined || !Number.isFinite(Number(value))) return '—'
  const n = Number(value)
  const unit = n === 0 ? 0 : Math.min(4, Math.max(0, Math.floor(Math.log(n) / Math.log(1024))))
  return `${(n / 1024 ** unit).toFixed(unit === 0 ? 0 : 1)} ${['B', 'KB', 'MB', 'GB', 'TB'][unit]}`
}
export function TrafficOverview() {
  const query = useTrafficOverviewQuery(60, 120)
  const metrics = useRuntimeMetrics()
  const { data: general } = useGeneralQuery()
  const { data: configs } = useConfigsQuery()
  const { data: nodes } = useNodesQuery()
  const { data: subs } = useSubscriptionsQuery()
  const config = configs?.configs.find((c) => c.selected)
  // A stopped Go engine returns a zero-value snapshot with year 0001.
  const data =
    query.data && Date.parse(query.data.updatedAt) > 0 && general?.general.dae.running !== false
      ? query.data
      : undefined
  const unavailable = query.isError
  const unsupported = unavailable && /Cannot query field|Unknown argument/.test(String(query.error))
  const stats = [
    ['累计上传', formatBytes(data?.uploadTotal)],
    ['累计下载', formatBytes(data?.downloadTotal)],
    ['活跃连接', data?.activeConnections?.toString() ?? '—'],
    ['UDP 会话', data?.udpSessions?.toString() ?? '—'],
    ['RSS 内存', formatBytes(metrics.data?.rssBytes ?? undefined)],
    ['堆内存', formatBytes(metrics.data?.heapBytes)],
    ['协程数', metrics.data?.goroutines?.toString() ?? '—'],
    ['CPU 占用率', metrics.data?.cpuPercent == null ? '—' : `${metrics.data.cpuPercent.toFixed(1)} %`],
  ]
  return (
    <section className="runtime-card" aria-label="运行状态">
      <div className="runtime-heading">
        <div className="runtime-heading-row">
          <div className="card-title">
            <span className="card-icon">
              <Activity size={18} />
            </span>
            <h2>运行状态</h2>
            <span className="console-badge">
              {general ? (general.general.dae.running ? '运行中' : '已停止') : '连接中'}
            </span>
          </div>
          <small>
            最近采样
            <br />
            <strong>{data ? new Date(data.updatedAt).toLocaleTimeString() : '—'}</strong>
          </small>
        </div>
        <div className="runtime-badges">
          <span>
            引擎 <b>kdae</b>
          </span>
          {metrics.data && <span>服务运行 <b>{Math.floor(metrics.data.processUptimeSeconds / 3600)} 小时 {Math.floor(metrics.data.processUptimeSeconds % 3600 / 60)} 分钟</b></span>}
          <span>
            拨号 <b>{config?.global.dialMode ?? '—'}</b>
          </span>
          <span>
            订阅 <b>{subs?.subscriptions.length ?? '—'}</b> · 节点 <b>{nodes?.nodes.edges.length ?? '—'}</b>
          </span>
          <span>
            流量{' '}
            <b>
              {data
                ? unavailable
                  ? '采样中断'
                  : '实时'
                : unsupported
                  ? '暂不支持'
                  : unavailable
                    ? '连接失败'
                    : query.isPending
                      ? '连接中'
                      : general?.general.dae.running
                        ? '等待采样'
                        : '已停止'}
            </b>
          </span>
          {general?.general.dae.modified && <span>配置待应用</span>}
        </div>
      </div>
      <div className="runtime-body">
        <div className="runtime-chart">
          <div className="traffic-legends">
            <span>
              <ArrowDownLeft size={14} />
              下载{' '}
              <b>
                {formatBytes(data?.downloadRate)}
                {data ? '/s' : ''}
              </b>
            </span>
            <span>
              <ArrowUpRight size={14} />
              上传{' '}
              <b>
                {formatBytes(data?.uploadRate)}
                {data ? '/s' : ''}
              </b>
            </span>
            <small>60 s</small>
          </div>
          <div className="traffic-plot">
            {data?.samples.length ? (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={data.samples} margin={{ top: 15, right: 8, bottom: 0, left: 0 }}>
                  <CartesianGrid vertical={false} stroke="var(--border)" strokeDasharray="3 4" />
                  <XAxis
                    dataKey="timestamp"
                    tickFormatter={(v) => new Date(v).toLocaleTimeString()}
                    minTickGap={75}
                    tickLine={false}
                    axisLine={false}
                    tick={{ fontSize: 10, fill: 'var(--muted-foreground)' }}
                  />
                  <YAxis
                    tickFormatter={(v) => formatBytes(v)}
                    width={58}
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 10, fill: 'var(--muted-foreground)' }}
                  />
                  <Tooltip
                    labelFormatter={(v) => new Date(String(v)).toLocaleTimeString()}
                    formatter={(v) => `${formatBytes(Number(v))}/s`}
                    contentStyle={{ background: 'var(--card)', border: '1px solid var(--border)', borderRadius: 12 }}
                  />
                  <Line
                    name="下载"
                    dataKey="downloadRate"
                    type="monotone"
                    stroke="var(--chart-2)"
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                  />
                  <Line
                    name="上传"
                    dataKey="uploadRate"
                    type="monotone"
                    stroke="var(--primary)"
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                  />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="chart-empty">
                {unsupported
                  ? '当前后端未提供流量接口，请升级支持 runtimeOverview 的 dae-wing。'
                  : unavailable
                    ? '无法获取流量数据，请检查后端连接。'
                    : query.isPending
                      ? '正在连接监控数据…'
                      : '暂无流量采样'}
              </div>
            )}
          </div>
          {data && unavailable && <p className="metric-note">连接中断 · 图表保留上次采样</p>}
        </div>
        <div className="runtime-stats">
          {stats.map(([label, value]) => (
            <div key={label}>
              <small>{label}</small>
              <strong>{value}</strong>
            </div>
          ))}
          <p className="metric-note">{metrics.isError ? (metrics.data ? '指标连接中断 · 保留上次采样' : '服务进程指标暂不可用') : metrics.data ? '服务进程指标 · 部分内核直连流量不计入统计' : '正在读取服务进程指标…'}</p>
        </div>
      </div>
    </section>
  )
}

# DaedKdae 控制台源码

在 Go dae-wing / kdae 后端上实现 DaeNext 风格的侧栏、顶栏、浅粉主题、实时流量面板和三栏摘要。资源管理继续使用 GraphQL，日志、系统指标、备份与 geodata 使用新增的 Go 原生鉴权接口。配置、DNS、路由、群组、节点、订阅、账户、主题、配置方案与启动/重载操作沿用 daed 组件。

## 来源与许可

- GraphQL 前端、编辑器、节点解析器与语言服务：MIT 许可的 [ksong008/daed](https://github.com/ksong008/daed)，固定提交 `4a519fcbaa7004e50e80a309c68d6811596468cf`，保留 [LICENSE](LICENSE) 及各源码版权声明。
- 布局参考：[ksong008/DaeNext](https://github.com/ksong008/DaeNext) 和其中使用的 DaedNext 控制台。借鉴导航结构、颜色、流量与资源卡片排布；没有引入 Rust 后端；新增接口由 Go 实现。
- 自定义界面：`apps/web/src/pages/MainLayout.tsx`、`pages/Orchestrate/DashboardSummary.tsx`、`pages/Orchestrate/TrafficOverview.tsx`、`src/index.css`。

## 开发与构建

需要 Node.js 22.12+（验证使用 24.12.0）和 pnpm 10.24.0。在仓库根目录执行：

```sh
pnpm --dir console install --frozen-lockfile
npm run build
npm run check-types
npm run dev:mock
```

`dev:mock` 是明确标注的演示模式，只用于布局预览。默认 `npm run dev` 和 `npm run build` 使用真实 GraphQL；生产构建强制关闭演示数据。

构建结果：仓库根目录 `dist/`。新版控制台是唯一生产构建入口，旧前端压缩包和历史 REST 演示服务已移除。

独立运行生产前端（PowerShell）：

```powershell
$env:DAED_BACKEND_URL='http://你的测试设备:2023/graphql'
npm start
```

浏览器打开 `http://127.0.0.1:3000`，通过登录页连接 `/graphql`，代理转发到指定 dae-wing 端点并保留用户 JWT。服务默认仅监听回环地址。开发服务器默认连接当前主机的 `:2023/graphql`，可在登录页改成实际后端 URL（跨域时后端需要允许 CORS）。

## 嵌入现有 kdae 二进制

将生产 `dist/` 放到 Linux 构建机的 `/root/daed/daed/dist/`，使用现有、已修复热重载的 wing 与 kdae 源码：

```sh
cd /root/daed/daed/wing
make OUTPUT=../daed APPNAME=daed WEB_DIST=../dist VERSION=<新版本号> bundle
```

构建前运行 `python3 backend/apply-console-features.py /absolute/path/to/wing`，它会核对源码、应用需要的热重载修复并安装日志等功能。详见 [后端说明](../backend/README.md)。已在两台 Debian 设备验证；v2.2.4-kdae 安装包内嵌新版控制台。

## 功能边界

- 总览显示选中的配置、接口地址、群组策略和可用节点；GraphQL 没有当前实际选中出口的字段，不把可用节点冒充当前出口。
- 流量来自 `general.runtimeOverview(windowSec:60,maxPoints:120)`；旧后端缺少该字段时显示升级提示并停止轮询。连接失败显示错误；中断时标注保留上次采样。
- RSS、Go 堆内存、协程、CPU 和服务进程时长由 `/api/runtime/metrics` 提供；进程时长不等于内核运行时长。缺少原生接口时显示不可用。
- 日志支持带鉴权的 SSE 实时流、断线重连、级别/关键词筛选、自动滚动、刷新、清空、保留数量和文件大小设置。运行日志级别立即生效并保留到重启后；默认保留 500 条 / 5 MB，设置范围 500–50000 条 / 5–200 MB。
- 顶栏支持 dae 导出/静态快照导入，以及 Go JSON 备份导出/新增资源导入。备份包含节点和订阅凭据，不包含账号密码；不自动替换运行配置，导入订阅的定时任务默认关闭。Rust 数据库迁移使用单独的迁移脚本，Rust 备份格式不能直接当 Go JSON 备份导入。
- dae 导入拒绝动态订阅、复杂群组筛选和群组独立检查参数，避免丢失语义。导出的订阅节点以静态节点快照保存；数字开头或带空格的节点名称使用整条列表项引号，导出结果会重新解析验证。
- Geosite/Geoip 每天北京时间 04:00 自动更新并热重载，显示日程和最近成功/失败结果；默认使用 Loyalsoldier 的 jsDelivr 地址，各自支持「使用代理」设置。两份全部验证后合并应用，失败保留旧数据。显示实际文件版本、分类及条目数；可保存 HTTPS 更新源，更新前验证格式和大小，运行引擎拒绝时恢复文件。`DAE_LOCATION_ASSET` 指向其他目录时拒绝网页更新，避免更新到未被引擎使用的路径。
- 迁移的 Rust 账号支持使用原密码首次登录，验证成功后自动转成 Go 哈希，保留 JWT secret；不需要重置密码。
- 首次访问默认浅色 Rose；已有主题偏好保留，语言与主题按钮继续使用原有功能。

## 验证

```sh
npm run build
npm run check-types
pnpm --dir console exec playwright install chromium
npm run test:console
```

Windows 如已安装 Edge，可省去 Chromium 下载：`$env:PLAYWRIGHT_CHANNEL='msedge'`，再执行测试。

浏览器检查使用**生产构建和受控 GraphQL 响应**，覆盖鉴权头、导航、URL 刷新、群组拖拽来源、缺失监控接口、采样中断、390px 手机导航与未登录跳转；截图保存到 `console/test-results/`。实机验证结果见 [后端说明](../backend/README.md)。设备配置、日志、迁移报告和实机截图不随源码发布。

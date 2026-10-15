# daed-kdae

Go dae-wing + kdae eBPF 引擎，内嵌新版网络管理控制台。

[简体中文](README.md) | [English](README.en.md) | [最新安装包](https://github.com/Quan-0505/daed-kdae/releases/latest)

## v2.2.6-kdae

已同步最新 Go 上游：kdae `eb510872`（SNI 路由学习、DNS/UDP 修正）与 dae-wing `b089b568`（数据层和延迟测试修正）。保留连接的热重载及日志等功能经过隔离网络实测；2026-10-16 复核这些提交仍是上游最新版本；v2.2.6 刷新公开规则数据并重新构建。详见 [更新记录](backend/UPSTREAM-20261016.md)。

新版控制台是唯一生产界面；旧皮肤压缩包、历史 REST 演示服务和过期安装资料已移除。首次访问默认浅粉 Rose，保留已有主题与语言偏好。

- 左侧导航、顶栏、流量曲线、配置 / 群组 / 节点与订阅摘要；放大首页字号和卡片，支持手机布局。
- 原生实时日志、级别与关键词筛选、自动滚动、持久化容量设置、运行时日志级别和真实进程指标。
- Geosite / Geoip 每天北京时间 04:00 成批验证、更新并热重载；支持独立更新源和「使用代理」，失败保留旧数据。
- 定时订阅成功更新后自动合并重载，无需手动点击；已停止的引擎不会被定时任务启动。
- 首页群组编辑按钮，支持节点 / 订阅选择、Go RE2 正则校验、匹配预览和事务保存；订阅状态显示中文。
- 配置、DNS、路由、群组、节点、订阅、账户、备份导入导出及启动 / 重载管理。
- Rust 账户密码兼容与独立数据库迁移工具；浏览器旧 `/api` 端点自动迁移到 `/graphql`。

源码和安装包不包含运行设备的订阅、节点、密码、令牌、数据库、备份或设备配置。预览模式使用虚构数据，生产构建强制关闭预览数据。

## 安装包

从 [v2.2.6-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.6-kdae) 下载。

| 系统 / 设备 | 文件 |
|---|---|
| Debian / Ubuntu x86-64-v2 | `daed-kdae_2.2.6-linux-x86_64_v2_sse.deb` |
| Debian / Ubuntu x86-64-v3 / AVX2 | `daed-kdae_2.2.6-linux-x86_64_v3_avx2.deb` |
| OpenWrt 25.12 R2S / R3S / R4S | `daed-kdae-r2s.apk` / `daed-kdae-r3s.apk` / `daed-kdae-r4s.apk` |
| OpenWrt 25.12 x86_64 | `daed-kdae-x86.apk` |
| 旧 APK 工具链 | 对应的 `*-v2.apk` 归档；先核对固件包格式 |

```sh
sudo apt install ./daed-kdae_2.2.6-linux-x86_64_v2_sse.deb
sudo systemctl enable --now daed
# OpenWrt 25.12：先上传到 /tmp，以下为 R4S 示例
apk add --allow-untrusted /tmp/daed-kdae-r4s.apk
/etc/init.d/daed enable
/etc/init.d/daed start
```

访问 `http://<设备IP>:2023`。全新安装创建管理员后配置接口、DNS、路由和订阅。群组资源编辑保存后按顶栏提示应用；定时更新自动重载。

默认 Go 数据目录 `/etc/daed/`，现有 systemd 数据目录覆盖继续有效。Rust 数据库不能直接用 Go 打开，从 Rust 切换前先备份并迁移到独立目录，见 [部署指引](README-部署指引.md)。安装包不预设账户或密码。

## 系统与验证

需要 Linux x86_64 / arm64、root、BTF 和匹配的数据面内核功能（veth、clsact / TC BPF；Netkit 路径需 Netkit）。不能仅凭内核版本号判断兼容性。

v2.2.4 及之前已在两台 Debian 13 / Linux 6.12 设备验证安装、登录、新界面、三次重载、DNS/联网、重启和日志；隔离网络还验证 TCP/UDP 代理转发、长连接和错误候选拒绝后旧引擎继续运行。v2.2.6 在构建机的隔离网络中验证三次真实 eBPF 热重载、持续 TCP/UDP 流量和实时日志，并通过全新源码构建与 race 测试。APK 格式、静态链接、架构及内层二进制检查不代表 OpenWrt 真机验证。

流量图来自代理会话统计，不等于网卡全部流量；进程时长不等于引擎时长。现有引擎退出时可能出现 conn_state_map 清理日志，见 [后端说明](backend/README.md)。

## 源码构建

```sh
pnpm --dir console install --frozen-lockfile
npm run build
npm run check-types
python3 backend/prepare-wing.py /absolute/path/to/wing --clone
bash packaging/build-release.sh /absolute/path/to/wing "$PWD/dist" /absolute/path/to/output 2.2.6
```

需要 Node.js 24、pnpm 10.24、Go 1.26、clang/LLVM、make、Python 3、dpkg-deb、curl。生产输出 `dist/`；`npm run dev:mock` 仅供虚构数据预览。详见 [前端说明](console/README.md) 和 [后端说明](backend/README.md)。APK v3 使用 `repack-apk-v3.yml` 工作流生成。

## 上游与许可

- 前端：[ksong008/daed](https://github.com/ksong008/daed)，固定 `4a519fcbaa7004e50e80a309c68d6811596468cf`，保留 MIT [许可](console/LICENSE)。布局参考 [DaeNext](https://github.com/ksong008/DaeNext)。
- 控制器：[dae-wing](https://github.com/daeuniverse/dae-wing)，固定 `b089b568649f03f909477adb6c7db25535a58782`。
- 引擎：[kdae](https://github.com/olicesx/dae/tree/kdae)，固定 `eb510872ae9474b7fa55a7f1d252ffd79a937854`。

引擎延续已验证的分阶段 TC hook / routing epoch 热重载适配；不自动跟随上游更新。后端遵循 [AGPL-3.0](LICENSE)，各组件保留原有版权及许可。

# Go 后端与构建

本仓库只构建新版 Go 控制台。前端在 `console/`，后端以固定上游源码加可审核补丁组成，旧 UI 压缩包和 REST 演示服务已经移除。

## 固定源码

- dae-wing：`dc503088945812c11235b35362d2bfa1a4c3bdf0`
- kdae：`f7c1cb6b04364903a55dbf9790d6e5eca7d5a628`
- GraphQL 前端来源与 MIT 许可见 [console/README.md](../console/README.md)。后端保持各上游许可。

`patches/bootstrap-manifest.json` 固定 kdae API、依赖和生命周期适配；`console-manifest.json` 固定原生日志、流量、群组编辑、密码兼容与订阅维护补丁。安装器核对 SHA256，拒绝覆盖未知修改。

## 从干净源码构建

Linux 构建机需要 Go 1.26、Git、Python 3、clang/LLVM、make，以及 Node.js 24 / pnpm 10.24。

```sh
pnpm --dir console install --frozen-lockfile
npm run build
npm run check-types
python3 backend/prepare-wing.py /absolute/path/to/wing --clone
export PATH=/usr/local/go/bin:$PATH
cd /absolute/path/to/wing
make deps
go test -race ./maintenance ./observability ./consoleapi ./graphql ./graphql/service/subscription
make OUTPUT=/absolute/path/to/daed APPNAME=daed WEB_DIST=/absolute/path/to/daed-kdae/dist VERSION=v2.2.4-kdae bundle
```

`OUTPUT` 是二进制文件路径。`prepare-wing.py` 可以重复用于已准备的固定源码目录，不能用于任意上游版本。嵌入生产 `dist/` 后，使用 `packaging/build-release.sh` 生成静态 arm64、amd64-v2/v3 二进制及 DEB/APK v2；OpenWrt 25.12 APK v3 由 SDK 工作流转换。

## 功能与验证

- 原生鉴权 SSE 日志、筛选、持久化容量设置、运行时日志级别、进程指标。
- 两份 geodata 每天北京时间 04:00 验证后成批应用并热重载；支持独立更新地址和代理开关；失败保留旧文件。
- 定时订阅更新成功并提交数据库后，合并触发重载；引擎停止时不自动启动。
- 群组节点/订阅编辑、Go RE2 正则校验及匹配预览、事务保存和冲突检测。
- Rust Argon2 账户首次成功登录后转换 Go 哈希；迁移保留账户 ID 和 JWT secret。
- dae / Go JSON 资源备份。备份含节点和订阅凭据，应自行妥善保管，不提交到 Git。

已在 Debian 13 / Linux 6.12 的两台设备实测安装、登录、浏览器界面、三次热重载、DNS/联网、重启和日志；隔离网络还验证了真实 TCP/UDP 转发、长连接和错误候选配置被拒绝后旧引擎继续运行。APK 包结构与架构验证不等于 OpenWrt 真机安装验证。现有 kdae 在退出阶段可能出现 conn_state_map 清理日志，运行与重载验证未因此失败。

## Rust 数据迁移

先停止服务，备份程序、数据库、配置、服务单元和 geodata，再在独立目录执行：

```sh
python3 backend/migrate-rust-state.py --help
python3 -m unittest discover -s backend/tests -p 'test_*.py'
```

迁移脚本不替换原 Rust 数据目录；实际安装应单独配置 Go 数据目录并设置回滚。不要将运行设备数据库、迁移报告、日志、截图、令牌或备份提交到仓库。


## Native DNS listener reload handoff

A configured `dns.bind` owns separate UDP/TCP sockets. The staged tproxy listener clone does not release that bind. Reload now stops the previous native DNS listener immediately before activating the candidate; rejection closes the candidate and restarts the previous DNS listener. The previous routing plane and established sessions remain alive during preparation. This prevents valid geodata updates from being rolled back solely because a native DNS port is occupied. A real-socket regression covers the collision, handoff and rollback. Backend CI also uploads a source-identified ARM64 binary for device validation.

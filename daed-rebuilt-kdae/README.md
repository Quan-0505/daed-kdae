# daed 重建版（内嵌 kdae 引擎）— x86_64

| 项目 | 值 |
|---|---|
| 构建日期 | 2026-09-01（Debian 13 机器 192.168.5.44 实机构建） |
| 版本 | **daed v1.28.0-kdae**（`--version` 输出） |
| Web 面板 | **ksong008/daed main 分支**（v1.28.0，GraphQL 架构，最新） |
| dae 引擎 | **olicesx/dae `kdae` 分支 @ `3ffde84`**（2026-09-01 更新，`fix(cmd,trace): close reload races and bound trace skb tracking`；已验证与 olicesx/kdae 分支 head 完全一致） |
| 后端 | daeuniverse/dae-wing main（dc50308894）+ kdae API 适配补丁 |
| 架构 | linux/amd64（x86_64），59.7 MB（含内嵌 web UI + eBPF 程序） |

## ⚠️ 重要：为什么是 v1.28.0（main 分支 web）而不是 v2.2.2-daed-test

部署测试发现：**ksong008/daed 的 v2.2.2-daed-test（2026-05-29）是过渡实验版**（tag 后缀 -daed-test），它的 web 走 **REST API**（`/api/auth/*`、`/api/configs`…），配套的是新架构 wing（040d8de，含 `engine/` 抽象层），而该 wing 依赖 ksong 血统 dae（bfe19ae）的控制面内部 API（`RuntimeDeps`、`NewDaeNetns`、`FlushReloadScopedResources` 等 7+ 个符号）。

**kdae 分支重构掉了这些控制面 API**（373 commits 的分叉），导致：
- v2.2.2 web（REST）↔ 新架构 wing ↔ bfe19ae 控制面 = 自洽，但引擎不是 kdae
- kdae ↔ 旧架构 wing（GraphQL）= 自洽（本次构建），但 v2.2.2 web（REST）无法认证 → **打开面板无登录提示框**

**解决方案（本构建采用）**：daed **main 分支的 web（v1.28.0，GraphQL 架构）** 与旧架构 wing + kdae 完全兼容——51 个 GraphQL 操作全部实测通过。即：**最新版 ksong daed web + kdae 引擎**。

## 运行时验证（已在 192.168.50.5 完整实测，2026-09-01）

- `daed --version` → `daed version v1.28.0-kdae`
- systemd `daed.service` **active (running)**，开机自启
- **Web UI 登录正常**：http://192.168.50.5:2023 → 登录框出现 → 账户 `REDACTED` / 密码 `REDACTED` 登录成功
- **完整 setup 流程通过**：createConfig（WAN=auto→enp1s0）/createDns/createRouting/createGroup/importNodes/selectConfig 全部成功
- **透明代理实测通过（含关键修复后）**：
  - baidu **HTTP 200**（国内直连，~0.07s）
  - google **HTTP 200**（走代理节点，~1.1s）
  - youtube **HTTP 200**（走代理节点，~1.9s）
  - 6 个 eBPF 过滤器全部挂载（wan/lan ingress+egress + dae0 + dae0peer）

## ⚠️ 关键修复记录（v1.28.0-kdae 含 3 组 kdae 适配补丁，缺一不可）

1. **`PublishActiveDebugState()` 发布**（dae/run.go）：kdae 的 handleConn 需要"活跃控制面"作为 routing-epoch 执行者；wing 旧式 Run 流程没调用 → 所有 TCP 连接报 `routing epoch execution owner is unavailable`。
2. **`IsReload: bpf != nil`**（dae/run.go）：reload 代控制面必须用 flip=1 的不同 TC handle；否则新旧代用同一 handle，旧控制面 Close 清理时**删掉新控制面的过滤器**，dae0/dae0peer 重定向路径断裂 → 流量完全不被代理（症状：面板/节点健康正常，但机器无外网）。
3. API 适配：`NewControlPlaneWithContextOptions`、`SnapshotRuntimeStats`、异步延迟探测、`NewFromLinkContext`、`Marshaller.Bytes()`。

**排查"没外网"的快速方法**：`tc filter show dev <WAN> ingress/egress` 应有 4 个过滤器；`ip netns exec daens tc filter show dev dae0peer ingress` 应有 `daed_dae0peer_ingress`。缺 dae0peer 过滤器 = 版本未含修复 2。

## 部署到新机器（推荐 .deb 一键安装）

```bash
# Debian/Ubuntu (x86_64)：
sudo dpkg -i daed_1.28.0-kdae_amd64.deb
# 依赖：iproute2 >= 6.7.0（apt install iproute2），内核 >= 5.8 且支持 BTF
# 安装后自动启服务：systemctl status daed
# 浏览器访问 http://<新机器IP>:2023，首次初始化账户
```

手动方式：

```bash
sudo install -m755 daed-linux-x86_64-kdae /usr/bin/daed
sudo mkdir -p /usr/share/daed /etc/daed
# geoip.dat / geosite.dat 放 /usr/share/daed/（本包 dae-kdae-20260831/ 内有）
sudo systemctl daemon-reload && sudo systemctl enable --now daed
# 浏览器访问 http://<IP>:2023，首次初始化账户
```

## 说明

- 构建源码与补丁位于构建机 `/root/daed/`（daed main + wing + dae-core@kdae + 4 组 API 适配补丁）。
- dae 内核要求：Linux ≥ 5.8 + BTF + iproute2 ≥ 6.7.0（本机 6.12 内核实测通过）。
- 独立 dae 二进制（dae-kdae-20260831/）与面板内嵌引擎同源（kdae 93a9957e）。

# dae（kdae 分支）最新构建 — 2026-09-13

| 项目 | 值 |
|---|---|
| 来源 | **olicesx/dae** 仓库 **`kdae` 分支**（本地构建） |
| 版本 | **unstable-20260913.r1.ee4ce27** |
| 对应提交 | `ee4ce27`（`test(control): satisfy errcheck in the janitor boundary test`，**与 kdae 分支 head 完全一致**） |
| 上一版 | 2026-09-01 / `3ffde84`（见 `dae-kdae-20260901/`） |
| 构建环境 | Debian 13（构建机 192.168.5.44），Go 1.26.3 |
| 架构 | x86_64（`dae-linux-x86_64`）、arm64（`dae-linux-arm64`） |

## 本版相对 2026-09-01 的主要更新（`3ffde84` → `ee4ce27`）

kdae 分支本区间含 500+ 提交，重点修复：

- **`fix(control): bound datapath name resolution and survive its failure`** —— 域名解析失败不再导致整体不可用（对 DNS 受限环境尤其重要）
- **`fix(cmd): bound the startup network wait and cancel it cleanly`** —— 启动等待网络有界、可取消（避免网络未就绪时卡住）
- **`fix(outbound): reset the compiled-filter regexp cache on reload`** —— reload 相关竞态
- **`fix(routing): reject unknown l4proto and ipversion operands`** —— 路由规则操作数校验
- **`fix(control): drop the bind-state entry instead of resetting it`** / **`report the real bpf pin directory failure`** —— 绑定状态与 bpf pin 处理
- **`fix(dns): enforce ipversion_prefer on every delivery path`**、**IPv4-mapped UDP 源归一化**
- 依赖更新：outbound fork → `olicesx/outbound v0.0.0-sticky-ip.0.20260912070929-ebd5cd55cbda`，新增 `olicesx/quic-go` replace

## 校验

- 二进制版本字符串（双架构一致）：`unstable-20260913.r1.ee4ce27`
- SHA256：

```
13afea60c5f0fa0859f9d28553fca61e40d53402ad9b4bdfe8f86547648d396f  dae-linux-x86_64
82e0dd2a1e40a4499684343e8e1b0b6b115e34167dac4275d4f973a7538abf9a  dae-linux-arm64
```

## 构建复现（本地）

```sh
# 源码（kdae 分支）
git clone --depth 1 --branch kdae https://github.com/olicesx/dae.git
cd dae
export PATH=/usr/local/go/bin:$PATH   # 需要 Go >= 1.26
make OUTPUT=./dae-linux-x86_64 VERSION=unstable-20260913.r1.ee4ce27
GOOS=linux GOARCH=arm64 make OUTPUT=./dae-linux-arm64 VERSION=unstable-20260913.r1.ee4ce27
```

## 文件

- `dae-linux-x86_64` / `dae-linux-arm64` — dae 可执行文件（静态 ELF）
- `geoip.dat` / `geosite.dat` — 规则数据
- `dae.service` / `example.dae` — systemd 部署与配置示例

## 说明

- daed 重建版（`daed-rebuilt-kdae/`）内嵌的引擎与本文档同一提交（`ee4ce27`）。
- 独立 dae 二进制与面板内嵌引擎同源同版本。

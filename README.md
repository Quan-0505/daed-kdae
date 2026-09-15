<div align="center">

# daed-kdae

**daed（Go 版，kdae 引擎）一体式透明代理安装包**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v2.2.2--kdae-orange.svg)](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.2-kdae)

基于 [daed](https://github.com/ksong008/daed)（GraphQL Web）+ [dae-wing](https://github.com/daeuniverse/dae-wing) + [kdae](https://github.com/olicesx/dae/tree/kdae)（olicesx/dae 的 `kdae` 分支 eBPF 引擎）。

</div>

**[简体中文](./README.md)** &nbsp;|&nbsp; **[English](./README.en.md)**

---

## ✨ 特性

- 🖥 **daed Web 面板** v2.2.2（GraphQL，内嵌二进制）
- 📌 **kdae 深度定制 eBPF 引擎**（基于 [olicesx/dae `kdae`](https://github.com/olicesx/dae/tree/kdae) 分支）
- 🔄 **reload 双代平滑切换**：IsReload flip handle 修复（透明代理无感重载、网络不断流）
- ⚡ **BBRv3 拥塞控制**：出站引入并默认集成 experimental BBRv3，提升抗丢包与长肥管道吞吐
- 🌐 **DNS RFC 规范强化**：完整支持 RFC 2308 否定缓存、严格 `ipversion_prefer`、DoQ 优雅流回收
- 🛡️ **数据路径加固**：eBPF 热路径单包 map 查找开销削减、活跃流动态重路由（re-route）、tproxy 限速防丢

## ⚡ kdae 引擎最新特性与演进

相比上游原始 dae，本项目内嵌的 [olicesx/dae `kdae` 分支](https://github.com/olicesx/dae/tree/kdae) 包含大量专为高性能网关和软路由定制的深度优化：

| 模块 | 关键演进与优化特性 |
|---|---|
| **内核与 eBPF 热路径** | • 削减 eBPF 热路径中的单包 map 查找冗余，显著降低软路由 CPU 负载并提升小包转发吞吐<br>• 数据面解析去除 bitfields 位域依赖，采用单写者（single-writer）应答绑定，丢包状态完全可观测<br>• tproxy 丢弃事件削峰限速与单调时间戳保障，彻底消除高并发下的 `fast_sock` 竞态丢包 |
| **拥塞控制 (BBRv3)** | • 引入并默认集成 **BBRv3** 拥塞控制算法（支持通过 `cc_override` 灵活覆盖）<br>• 显著提升 QUIC / TUIC / Hysteria 等现代出站协议在越洋高延迟、弱网高丢包环境下的吞吐效率 |
| **DNS 引擎强化** | • 严格在所有分发路径执行 `ipversion_prefer`（IPv4 / IPv6 偏好策略）<br>• 完整实现 **RFC 2308 否定缓存**（Negative Caching）规范、全消息 TTL 与 CNAME-only NODATA 规范分类<br>• 完善 DoQ（DNS-over-QUIC）多路复用连接回收，规避 `CancelRead` 资源残留；支持截断响应（TC=1）原样传递与 TCP 自动降级重试 |
| **动态路由与重载** | • 路由代际（routing generation）变更时，自动对现有活跃连接（live flows）执行优雅平滑重新路由<br>• 节点组（Group availability）健康探测排序解耦，消除并发回调死锁<br>• 配置热重载（reload）时自动清理并重置 compiled-filter 正则缓存，杜绝热更状态污染 |
| **协议与边缘适配** | • Shadowsocks 2022 过期时间戳容错降级为 soft endpoint error，抵御时钟轻微漂移引发的异常断连<br>• REALITY spider 爬虫调度加固；启动网络等待超时优雅退出 |

## 📦 安装包（[v2.2.2-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.2-kdae)，deb + apk 统一发布）

| 平台 / 设备 | OpenWrt 25.12+ (apk v3) | OpenWrt 24.x/23.x/Alpine (apk v2) | 架构 |
|---|---|---|---|
| Debian/Ubuntu x86_64 (SSE4.2/v2) | - | `daed-kdae_2.2.2-linux-x86_64_v2_sse.deb` | x86_64 |
| Debian/Ubuntu x86_64 (AVX2/v3) | - | `daed-kdae_2.2.2-linux-x86_64_v3_avx2.deb` | x86_64 |
| OpenWrt X86 软路由 | `daed-kdae_2.2.2-x86_64-v3.apk` | `daed-kdae_2.2.2-x86_64-v2.apk` | x86_64 |
| NanoPi R4S | `daed-kdae_2.2.2-R4S-v3.apk` | `daed-kdae_2.2.2-R4S-v2.apk` | aarch64_generic |
| NanoPi R3S | `daed-kdae_2.2.2-R3S-v3.apk` | `daed-kdae_2.2.2-R3S-v2.apk` | aarch64_generic |
| NanoPi R2S | `daed-kdae_2.2.2-R2S-v3.apk` | `daed-kdae_2.2.2-R2S-v2.apk` | aarch64_generic |

## 🚀 快速开始

```sh
# Debian / Ubuntu (标准 x86_64 SSE4.2，老旧 CPU 通用)
sudo dpkg -i daed-kdae_2.2.2-linux-x86_64_v2_sse.deb

# Debian / Ubuntu (现代 x86_64 AVX2，推荐 Intel 4代+ / AMD Zen+)
sudo dpkg -i daed-kdae_2.2.2-linux-x86_64_v3_avx2.deb

# OpenWrt 25.12（apk v3 / apk-tools 3）
apk add --allow-untrusted ./daed-kdae_2.2.2-<设备>-v3.apk

# OpenWrt 24.x / 23.x / Alpine（apk v2）
apk add --allow-untrusted ./daed-kdae_2.2.2-<设备>-v2.apk

# 启用并启动服务
/etc/init.d/daed enable && /etc/init.d/daed start
# Web 面板: http://<机器IP>:2023
```

## 📋 系统要求

x86_64 / aarch64 Linux，内核 ≥ 5.8 且启用 **BTF**；iproute2 ≥ 6.7；root 权限。

dae 的 eBPF 数据面还要求内核开启 **veth**、**clsact**（`NET_SCH_INGRESS` / `NET_CLS_ACT` / `NET_CLS_BPF`），
并需要宿主工具 `tc` / `bpftool` / `ipset`（OpenWrt 上：`apk add tc-full bpftool-minimal ip-full ipset`）。

⚠️ **BTF 是两者共同前提**：Go 版（kdae）与 Rust 版（[rust-daed](https://github.com/Quan-0505/rust-daed)）都依赖内核 BTF 与 veth/clsact。
OpenWrt 官方固件多默认未开 `CONFIG_DEBUG_INFO_BTF`，这类设备两者都无法启动——需要自编译内核，或直接使用已启用的固件
[Quan-0505/OpenWrt](https://github.com/Quan-0505/OpenWrt)（该固件已内置 BTF / veth / clsact 与宿主工具）。

### ⚠️ 包格式（OpenWrt 25.12 用 apk v3，旧环境用 apk v2）

OpenWrt **25.12 起使用 apk-tools 3.x**，包格式为 **apk v3**（ADB 容器：文件头为 `ADB`；既不是 tar 也不是 gzip，用 `tar`/apk2 工具打不开是正常的）。旧版系统（24.x、23.x、Alpine 等）则使用传统 tar 格式的 **apk v2**。

本 Release 为避免混淆，严格按版本后缀区分提供：

```sh
# OpenWrt 25.12+ (apk v3)
scp daed-kdae_2.2.2-R4S-v3.apk root@192.168.2.1:/tmp/
ssh root@192.168.2.1 'apk add --allow-untrusted /tmp/daed-kdae_2.2.2-R4S-v3.apk'

# OpenWrt 24.x / 23.x / Alpine (apk v2)
scp daed-kdae_2.2.2-R4S-v2.apk root@192.168.2.1:/tmp/
ssh root@192.168.2.1 'apk add --allow-untrusted /tmp/daed-kdae_2.2.2-R4S-v2.apk'
```

| 目标环境 | 包管理 | 推荐安装包 |
|---|---|---|
| OpenWrt 25.12+（apk-tools 3） | `apk` | ✅ `daed-kdae_2.2.2-<device>-v3.apk`（apk v3，ADB 容器格式） |
| OpenWrt 24.x / 23.x / Alpine | `apk` | ✅ `daed-kdae_2.2.2-<device>-v2.apk`（apk v2，传统 tar 格式） |
| Debian / Ubuntu (标准/老旧 x86_64) | `dpkg` | ✅ `daed-kdae_2.2.2-linux-x86_64_v2_sse.deb` |
| Debian / Ubuntu (现代 x86_64 AVX2) | `dpkg` | ✅ `daed-kdae_2.2.2-linux-x86_64_v3_avx2.deb` |

已在真机验证（NanoPi R4S / OpenWrt 25.12.5 / apk-tools 3.0.5 / `aarch64_generic`）：

```text
(1/1) Upgrading daed (3.1.0-r2 -> 3.1.1-r1)
  Executing daed-3.1.1-r1.post-upgrade
OK: 144.5 MiB in 305 packages
```

重打包由仓库内 `workflow_dispatch` 流程 **Repack apk as OpenWrt 25.12 (apk v3)** 完成（用 OpenWrt 25.12 SDK 的 apk-tools 3 重新打包，v2 的脚本钩子改写成 OpenWrt 的 `postinst`/`prerm`）——升级上游二进制时重跑一次即可。

> 注意：本包与 [rust-daed](https://github.com/Quan-0505/rust-daed)（Rust 引擎版）提供相同的 `/usr/bin/daed` 与 `/etc/init.d/daed`，**两者只能装一个**，后装的会覆盖先装的。

## 📂 仓库内容

| 目录 / 文件 | 说明 |
|---|---|
| `daed-rebuilt-kdae/` | ★ 重建版 daed（内嵌 kdae，x86_64）：二进制 + deb + README（修复记录） |
| `dae-kdae-20260901/` | 独立 kdae 二进制（x86_64 + arm64）+ geoip/geosite + dae.service/example.dae |
| `daed-official/` | ksong 官方 installer（v2.2.2-daed-test 参考） |
| `README-部署指引.md` | 完整部署文档 |
| `checksums.txt` | 全部文件 SHA256 校验和 |

## 📄 许可

[GNU Affero General Public License v3.0](LICENSE)。上游 daed-wing / dae 生态为 AGPL-3.0。

---
*Rust 版（DaedNext 引擎）见 [rust-daed](https://github.com/Quan-0505/rust-daed)。*

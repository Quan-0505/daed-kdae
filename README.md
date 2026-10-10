<div align="center">

# daed-kdae

**daed（Go 版，kdae 引擎）一体式透明代理安装包**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v2.2.3--kdae-orange.svg)](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.3-kdae)

基于 [daed](https://github.com/ksong008/daed)（GraphQL Web）+ [dae-wing](https://github.com/daeuniverse/dae-wing) + [kdae](https://github.com/olicesx/dae/tree/kdae)（olicesx/dae 的 `kdae` 分支 eBPF 引擎）。

</div>

**[简体中文](./README.md)** &nbsp;|&nbsp; **[English](./README.en.md)**

---

## 当前版本：v2.2.3-kdae

- kdae 从 `ee4ce27` 升级到 [`f7c1cb6`](https://github.com/olicesx/dae/commit/f7c1cb6b04364903a55dbf9790d6e5eca7d5a628)（2026-10-08），包含 157 个提交及上游热重载架构重构。
- dae-wing 适配 `Prepare → Commit → Adopt → Finalize` 分阶段 TC hook 接管、候选世代延迟提交数据面与新旧 routing epoch peer 交接。
- 修复 `attach tcx link: file exists`、`has no routing epoch peer`、第二次重载 `TC HookSet transaction already active`，以及旧世代退出通知导致守护进程结束的问题。

**验证范围：**在 Debian 13 / Linux 6.12 上使用 dummy WAN/LAN 接口，冷启动、GraphQL 鉴权及**连续三次配置热重载**全部成功，进程持续存活，日志无 error/fatal，`healthCheck=1`。APK 已检查包结构、架构、静态链接与符号；DEB 已核对内层二进制。**本版本尚未完成 APK 真机安装/启动及真实网卡转发验证，也尚未部署到生产设备。**上述检查不代表业务流量零中断。

下列特性以本版本固定的引擎提交为准；上游后续更新不会自动进入已发布安装包。

## ✨ 特性

- 🖥 **daed Web 面板**（GraphQL，内嵌二进制）
- 📌 **kdae 深度定制 eBPF 引擎**（基于 [olicesx/dae `kdae`](https://github.com/olicesx/dae/tree/kdae) 分支）
- 🔄 **分阶段配置热重载**：TC hook 接管与新旧 routing epoch peer 交接
- ⚡ **BBRv3 拥塞控制**：出站引入并默认集成 experimental BBRv3，提升抗丢包与长肥管道吞吐
- 🌐 **DNS RFC 规范强化**：完整支持 RFC 2308 否定缓存、严格 `ipversion_prefer`、DoQ 优雅流回收
- 🛡️ **数据路径加固**：eBPF 热路径单包 map 查找开销削减、活跃流动态重路由（re-route）、tproxy 限速防丢

## ⚡ kdae 引擎最新特性与演进

相比上游原始 dae，本项目内嵌的 [olicesx/dae `kdae` 分支](https://github.com/olicesx/dae/tree/kdae) 包含大量专为高性能网关和软路由定制的深度优化：

| 模块 | 关键演进与优化特性 |
|---|---|
| **内核与 eBPF 热路径** | • 削减 eBPF 热路径中的单包 map 查找冗余，旨在减少数据面开销<br>• 数据面解析去除 bitfields 位域依赖，采用单写者（single-writer）应答绑定，丢包状态完全可观测<br>• tproxy 丢弃事件削峰限速与单调时间戳保障，改进高并发下的 `fast_sock` 竞态处理 |
| **拥塞控制 (BBRv3)** | • 引入并默认集成 **BBRv3** 拥塞控制算法（支持通过 `cc_override` 灵活覆盖）<br>• 面向 QUIC / TUIC / Hysteria 等出站协议的拥塞控制改进，实际吞吐取决于链路与配置 |
| **DNS 引擎强化** | • 严格在所有分发路径执行 `ipversion_prefer`（IPv4 / IPv6 偏好策略）<br>• 完整实现 **RFC 2308 否定缓存**（Negative Caching）规范、全消息 TTL 与 CNAME-only NODATA 规范分类<br>• 完善 DoQ（DNS-over-QUIC）多路复用连接回收，规避 `CancelRead` 资源残留；支持截断响应（TC=1）原样传递与 TCP 自动降级重试 |
| **动态路由与重载** | • 路由代际（routing generation）变更时，自动对现有活跃连接（live flows）执行优雅平滑重新路由<br>• 节点组（Group availability）健康探测排序解耦，消除并发回调死锁<br>• 配置热重载（reload）时自动清理并重置 compiled-filter 正则缓存，杜绝热更状态污染 |
| **协议与边缘适配** | • Shadowsocks 2022 过期时间戳容错降级为 soft endpoint error，抵御时钟轻微漂移引发的异常断连<br>• REALITY spider 爬虫调度加固；启动网络等待超时优雅退出 |

## 📦 安装包（[v2.2.3-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.3-kdae)，deb + apk 统一发布）

| 平台 / 设备 | 安装包 | 格式 |
|---|---|---|
| OpenWrt NanoPi R2S | [daed-kdae-r2s-v2.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-r2s-v2.apk) | APK v2 |
| OpenWrt NanoPi R2S | [daed-kdae-r2s.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-r2s.apk) | APK v3 (ADB) |
| OpenWrt NanoPi R3S | [daed-kdae-r3s-v2.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-r3s-v2.apk) | APK v2 |
| OpenWrt NanoPi R3S | [daed-kdae-r3s.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-r3s.apk) | APK v3 (ADB) |
| OpenWrt NanoPi R4S | [daed-kdae-r4s-v2.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-r4s-v2.apk) | APK v2 |
| OpenWrt NanoPi R4S | [daed-kdae-r4s.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-r4s.apk) | APK v3 (ADB) |
| OpenWrt x86_64 | [daed-kdae-x86-v2.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-x86-v2.apk) | APK v2 |
| OpenWrt x86_64 | [daed-kdae-x86.apk](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae-x86.apk) | APK v3 (ADB) |
| Debian / Ubuntu — x86-64-v2 | [daed-kdae_2.2.3-linux-x86_64_v2_sse.deb](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae_2.2.3-linux-x86_64_v2_sse.deb) | DEB |
| Debian / Ubuntu — x86-64-v3（含 AVX2） | [daed-kdae_2.2.3-linux-x86_64_v3_avx2.deb](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae_2.2.3-linux-x86_64_v3_avx2.deb) | DEB |

## 🚀 快速开始

```sh
# Debian / Ubuntu：按 CPU 选择一个 DEB
sudo apt install ./daed-kdae_2.2.3-linux-x86_64_v2_sse.deb
# 或：sudo apt install ./daed-kdae_2.2.3-linux-x86_64_v3_avx2.deb
sudo systemctl enable --now daed
sudo systemctl status daed --no-pager
```

```sh
# OpenWrt 25.12 / APK v3：R4S 示例
apk add --allow-untrusted /tmp/daed-kdae-r4s.apk
/etc/init.d/daed enable
/etc/init.d/daed start
```

从上表下载适合设备的安装包；OpenWrt 示例先将包上传到 `/tmp`。打开 `http://<设备IP>:2023`，首次运行创建管理员，再配置接口、订阅、DNS 与路由。WebUI 保存配置会触发热重载。

## 📋 系统要求

Linux x86_64 / arm64、root 权限，内核启用 BTF（检查 `/sys/kernel/btf/vmlinux`），并具备所选数据面的功能，包括 veth、clsact / TC BPF。Netkit 路径还需要内核支持 Netkit。不能只凭“内核 ≥ 5.8”判断当前引擎可用；当前实测环境为 Linux 6.12。

- `/sys/fs/bpf` 必须挂载为 bpffs，可用 `findmnt -T /sys/fs/bpf` 检查。
- DEB 声明依赖 `iproute2 >= 6.7.0`。系统需提供 `ip`、`tc` 等工具；OpenWrt 还需与内核匹配的模块。
- DEB 需满足完整 x86-64-v2 或 x86-64-v3 指令集要求；仅有 AVX2 不足以证明满足 v3。

### 包格式与兼容性

OpenWrt 25.12 使用 APK，其 **APK v3（ADB 格式）**资产保留原名，例如 `daed-kdae-r4s.apk`，**没有 `-v3` 后缀**。原始 **APK v2** 包归档为 `daed-kdae-r4s-v2.apk`。APK 包格式与 DEB 的 x86-64-v2/v3 CPU 指令集是不同含义。

[OpenWrt 24.10 及更早官方版本使用 opkg](https://openwrt.org/docs/guide-user/additional-software/managing_packages)，不能直接安装这些 APK，需要匹配的 IPK 或迁移到适用固件。APK v2 格式也不等于 Alpine 兼容：本项目包内包含 OpenWrt 服务配置及安装脚本，尚未验证 Alpine 安装。安装前请核对目标包管理器、架构与依赖。

重打包由 [repack-apk-v3.yml](.github/workflows/repack-apk-v3.yml) 使用 OpenWrt SDK 完成：v3 覆盖原资产名，v2 归档为 `-v2.apk`。本版本共四个 v3 APK、四个 v2 APK、两个 DEB。

### LAN 网关参数

检查 `all`、`default` 与实际 LAN 接口的 `rp_filter`。仅调整 `all` 可能仍被接口值覆盖，造成本机正常、下游断网。

```sh
sudo sysctl -w net.ipv4.conf.all.rp_filter=0
sudo sysctl -w net.ipv4.conf.default.rp_filter=0
sudo sysctl -w net.ipv4.conf.<lan_iface>.rp_filter=0
```

将 `<lan_iface>` 替换为实际 LAN 接口，并写入系统的 sysctl 持久化配置；Debian / Ubuntu 可使用 `/etc/sysctl.d/99-daed.conf`。

### 切换与回滚

本项目与 [rust-daed](https://github.com/Quan-0505/rust-daed) 共用 `/usr/bin/daed`、服务名及 `dae0` 数据面，**同一设备只能运行其中一个**。切换前停止现有服务，备份二进制、服务配置、`/etc/daed/` 与配置数据库 `wing.db`，并保留旧安装包；不能假定两条产品线的数据库兼容。

升级后检查启动日志、WebUI 鉴权、LAN 客户端联网及连续热重载。验证失败时停止新服务，恢复旧安装包和对应配置，再启动旧服务。首次切换先在测试设备或单台设备上演练。

Debian / Ubuntu 若存在历史 `/etc/systemd/system/daed.service`，会覆盖包提供的 unit。使用 `systemctl cat daed` 与 `systemctl show daed -p FragmentPath -p CanReload` 检查实际生效配置后，再处理覆盖文件。

## 📂 仓库内容

| 目录 / 文件 | 说明 |
|---|---|
| `daed-rebuilt-kdae/` | 历史重建二进制、DEB 与修复记录；当前安装包以 Release 为准 |
| `dae-kdae-20260901/` | 独立 kdae 二进制（x86_64 + arm64）+ geoip/geosite + dae.service/example.dae |
| `daed-official/` | ksong 官方 installer（v2.2.2-daed-test 参考） |
| `README-部署指引.md` | 历史部署文档；版本与路径以本文和当前 Release 为准 |
| `checksums.txt` | 仓库文件 SHA256 清单，不是当前 Release 资产校验清单 |

## 📄 许可

[GNU Affero General Public License v3.0](LICENSE)。上游 daed-wing / dae 生态为 AGPL-3.0。

---
*Rust 版（DaedNext 引擎）见 [rust-daed](https://github.com/Quan-0505/rust-daed)。*

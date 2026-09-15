<div align="center">

# daed-kdae

**daed（Go 版，kdae 引擎）一体式透明代理安装包**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v2.2.2--kdae-orange.svg)](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.2-kdae)

基于 [daed](https://github.com/ksong008/daed)（GraphQL Web）+ [dae-wing](https://github.com/daeuniverse/dae-wing) + [dae](https://github.com/daeuniverse/dae) `kdae` 分支（Go eBPF 引擎）。

</div>

---

## ✨ 特性

- 🖥 **daed Web 面板** v2.2.2（GraphQL，内嵌二进制）
- 📌 **kdae 引擎** @ `3ffde84`（sticky-ip / 异步延迟探测 / routing-epoch 竞态修复）
- 🔄 **reload 双代切换**：IsReload flip handle 修复（透明代理不断流）

## 📦 安装包（[v2.2.2-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.2-kdae)，deb + apk 统一发布）

| 平台 / 设备 | OpenWrt 25.12+ (apk v3) | OpenWrt 24.x/23.x/Alpine (apk v2) | 架构 |
|---|---|---|---|
| Debian/Ubuntu x86_64 | - | `daed_2.2.2-kdae_amd64.deb` (dpkg) | amd64 |
| OpenWrt X86 软路由 | `daed-kdae-x86-v3.apk` | `daed-kdae-x86-v2.apk` | x86_64 |
| NanoPi R4S | `daed-kdae-r4s-v3.apk` | `daed-kdae-r4s-v2.apk` | aarch64_generic |
| NanoPi R3S | `daed-kdae-r3s-v3.apk` | `daed-kdae-r3s-v2.apk` | aarch64_generic |
| NanoPi R2S | `daed-kdae-r2s-v3.apk` | `daed-kdae-r2s-v2.apk` | aarch64_generic |

## 🚀 快速开始

```sh
# Debian / Ubuntu
sudo dpkg -i daed_2.2.2-kdae_amd64.deb

# OpenWrt 25.12（apk v3 / apk-tools 3）
apk add --allow-untrusted ./daed-kdae-<设备>-v3.apk

# OpenWrt 24.x / 23.x / Alpine（apk v2）
apk add --allow-untrusted ./daed-kdae-<设备>-v2.apk

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
scp daed-kdae-r4s-v3.apk root@192.168.2.1:/tmp/
ssh root@192.168.2.1 'apk add --allow-untrusted /tmp/daed-kdae-r4s-v3.apk'

# OpenWrt 24.x / 23.x / Alpine (apk v2)
scp daed-kdae-r4s-v2.apk root@192.168.2.1:/tmp/
ssh root@192.168.2.1 'apk add --allow-untrusted /tmp/daed-kdae-r4s-v2.apk'
```

| 目标环境 | 包管理 | 用哪个文件 |
|---|---|---|
| OpenWrt 25.12+（apk-tools 3） | `apk` | ✅ `daed-kdae-<device>-v3.apk`（apk v3，ADB 容器） |
| OpenWrt 24.x / 23.x / Alpine | `apk` | ✅ `daed-kdae-<device>-v2.apk`（apk v2，传统 tar 格式） |
| Debian / Ubuntu | `dpkg` | ✅ `daed_2.2.2-kdae_amd64.deb` |

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

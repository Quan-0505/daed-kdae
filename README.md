<div align="center">

# daed-kdae

**daed（Go 版，kdae 引擎）一体式透明代理安装包**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v1.28.0--kdae-orange.svg)](https://github.com/Quan-0505/daed-kdae/releases/tag/v1.28.0-kdae)

基于 [daed](https://github.com/ksong008/daed)（GraphQL Web）+ [dae-wing](https://github.com/daeuniverse/dae-wing) + [dae](https://github.com/daeuniverse/dae) `kdae` 分支（Go eBPF 引擎）。

</div>

---

## ✨ 特性

- 🖥 **daed Web 面板** v1.28.0（GraphQL，内嵌二进制）
- 📌 **kdae 引擎** @ `3ffde84`（sticky-ip / 异步延迟探测 / routing-epoch 竞态修复）
- 🔄 **reload 双代切换**：IsReload flip handle 修复（透明代理不断流）

## 📦 安装包（[v1.28.0-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v1.28.0-kdae)，deb + apk 统一发布）

| 平台 / 设备 | 文件 | 架构 |
|---|---|---|
| Debian/Ubuntu x86_64 | `daed_1.28.0-kdae_amd64.deb` | amd64 |
| OpenWrt X86 软路由 | `daed-kdae-x86.apk` | x86_64 |
| NanoPi R4S | `daed-kdae-r4s.apk` | aarch64_cortex-a72 |
| NanoPi R3S | `daed-kdae-r3s.apk` | aarch64_cortex-a53 |
| NanoPi R2S | `daed-kdae-r2s.apk` | aarch64_cortex-a53 |

## 🚀 快速开始

```sh
# Debian / Ubuntu
sudo dpkg -i daed_1.28.0-kdae_amd64.deb
# OpenWrt 25.12（apk v3 / apk-tools 3）
apk add --allow-untrusted ./daed-kdae-<设备>.apk
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

### ⚠️ 包格式兼容性（重要）

OpenWrt **25.12 起使用 apk-tools 3.x**，包格式为 **apk v3**（ADB 容器）。本 Release 的 `daed-kdae-*.apk` 目前是 **apk v2 格式**，
在 apk-tools 3 上安装会报 `v2 package format error`；v3 重打包完成前，OpenWrt 25.12 请使用上面的内置固件，或 Debian/Ubuntu 用 `.deb`。

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
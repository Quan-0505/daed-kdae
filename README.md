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
# OpenWrt 25.12（apk v2）
apk add --allow-untrusted ./daed-kdae-<设备>.apk
/etc/init.d/daed enable && /etc/init.d/daed start
# Web 面板: http://<机器IP>:2023
```

## 📋 系统要求

x86_64 / aarch64 Linux，内核 ≥ 5.8 且启用 **BTF**；iproute2 ≥ 6.7；root 权限。
⚠️ OpenWrt 官方固件多默认未开 CONFIG_DEBUG_INFO_BTF——此类设备推荐用 [rust-daed](https://github.com/Quan-0505/rust-daed)。

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
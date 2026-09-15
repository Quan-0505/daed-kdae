<div align="center">

# daed-kdae

**daed (Go edition, kdae engine) all-in-one transparent proxy installer package**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v2.2.2--kdae-orange.svg)](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.2-kdae)

Built on [daed](https://github.com/ksong008/daed) (GraphQL Web) + [dae-wing](https://github.com/daeuniverse/dae-wing) + [kdae](https://github.com/olicesx/dae/tree/kdae) (the eBPF engine from the `kdae` branch of olicesx/dae).

</div>

**English** &nbsp;|&nbsp; **[简体中文](./README.md)**

---

## ✨ Features

- 🖥 **daed Web panel** v2.2.2 (GraphQL, embedded binary)
- 📌 **Deeply customized kdae eBPF engine** (based on the [olicesx/dae `kdae`](https://github.com/olicesx/dae/tree/kdae) branch)
- 🔄 **Seamless dual-generation reload switchover**: IsReload flip handle fix (transparent-proxy reloads with no perceptible interruption and no dropped flows)
- ⚡ **BBRv3 congestion control**: introduced into the outbound path and integrated by default as experimental BBRv3, improving loss resilience and long-fat-pipe throughput
- 🌐 **DNS RFC hardening**: full support for RFC 2308 negative caching, strict `ipversion_prefer`, graceful DoQ stream reclamation
- 🛡️ **Data path hardening**: reduced per-packet map lookup overhead on the eBPF hot path, dynamic re-routing of live flows (re-route), tproxy rate limiting against drops

## ⚡ Latest kdae engine features and evolution

Compared with the original upstream dae, the [olicesx/dae `kdae` branch](https://github.com/olicesx/dae/tree/kdae) embedded in this project contains a large number of deep optimizations tailored for high-performance gateways and soft routers:

| Module | Key evolution and optimizations |
|---|---|
| **Kernel and eBPF hot path** | • Eliminated redundant per-packet map lookups on the eBPF hot path, significantly reducing soft-router CPU load and improving small-packet forwarding throughput<br>• The data plane no longer relies on bitfields for parsing and uses single-writer reply binding, making packet-drop state fully observable<br>• tproxy drop event peak shaving and rate limiting plus monotonic timestamps, completely eliminating `fast_sock` race-induced packet loss under high concurrency |
| **Congestion control (BBRv3)** | • Introduced and integrated the **BBRv3** congestion control algorithm by default (flexibly overridable via `cc_override`)<br>• Significantly improves the throughput efficiency of modern outbound protocols such as QUIC / TUIC / Hysteria in transoceanic high-latency, lossy-network conditions |
| **DNS engine hardening** | • Strictly enforces `ipversion_prefer` (IPv4 / IPv6 preference policy) on all distribution paths<br>• Fully implements the **RFC 2308 Negative Caching** specification, whole-message TTL, and CNAME-only NODATA classification<br>• Improved DoQ (DNS-over-QUIC) multiplexed connection reclamation, avoiding `CancelRead` resource leftovers; supports passing truncated responses (TC=1) through as-is with automatic TCP fallback and retry |
| **Dynamic routing and reload** | • When the routing generation changes, live flows are automatically and gracefully re-routed<br>• Node group (Group availability) health probing and ordering are decoupled, eliminating concurrent callback deadlocks<br>• On config hot reload, the compiled-filter regex cache is automatically cleaned up and reset, preventing hot-reload state pollution |
| **Protocol and edge adaptation** | • Shadowsocks 2022 expired timestamps degrade gracefully to soft endpoint errors, defending against abnormal disconnects triggered by slight clock drift<br>• Hardened REALITY spider scheduling; graceful exit on startup network wait timeout |

## 📦 Installer packages ([v2.2.2-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.2-kdae), deb + apk released together)

| Platform / Device | OpenWrt 25.12+ (apk v3) | OpenWrt 24.x/23.x/Alpine (apk v2) | Architecture |
|---|---|---|---|
| Debian/Ubuntu x86_64 (SSE4.2/v2) | - | `daed-kdae_2.2.2-linux-x86_64_v2_sse.deb` | x86_64 |
| Debian/Ubuntu x86_64 (AVX2/v3) | - | `daed-kdae_2.2.2-linux-x86_64_v3_avx2.deb` | x86_64 |
| OpenWrt X86 soft router | `daed-kdae_2.2.2-x86_64-v3.apk` | `daed-kdae_2.2.2-x86_64-v2.apk` | x86_64 |
| NanoPi R4S | `daed-kdae_2.2.2-R4S-v3.apk` | `daed-kdae_2.2.2-R4S-v2.apk` | aarch64_generic |
| NanoPi R3S | `daed-kdae_2.2.2-R3S-v3.apk` | `daed-kdae_2.2.2-R3S-v2.apk` | aarch64_generic |
| NanoPi R2S | `daed-kdae_2.2.2-R2S-v3.apk` | `daed-kdae_2.2.2-R2S-v2.apk` | aarch64_generic |

## 🚀 Quick start

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

## 📋 System requirements

x86_64 / aarch64 Linux, kernel ≥ 5.8 with **BTF** enabled; iproute2 ≥ 6.7; root privileges.

dae's eBPF data plane additionally requires the kernel to enable **veth** and **clsact** (`NET_SCH_INGRESS` / `NET_CLS_ACT` / `NET_CLS_BPF`),
as well as the host tools `tc` / `bpftool` / `ipset` (on OpenWrt: `apk add tc-full bpftool-minimal ip-full ipset`).

⚠️ **BTF is a shared prerequisite for both**: the Go edition (kdae) and the Rust edition ([rust-daed](https://github.com/Quan-0505/rust-daed)) both depend on kernel BTF and veth/clsact.
Official OpenWrt firmware mostly does not enable `CONFIG_DEBUG_INFO_BTF` by default, and neither can start on such devices — you need to build your own kernel, or directly use firmware that has it enabled:
[Quan-0505/OpenWrt](https://github.com/Quan-0505/OpenWrt) (that firmware already bundles BTF / veth / clsact and the host tools).

### ⚠️ Package format (OpenWrt 25.12 uses apk v3, older environments use apk v2)

Starting with OpenWrt **25.12, apk-tools 3.x** is used, with the package format **apk v3** (ADB container: the file header is `ADB`; it is neither tar nor gzip, so it is normal that `tar`/apk2 tools cannot open it). Older systems (24.x, 23.x, Alpine, etc.) instead use the traditional tar format, **apk v2**.

To avoid confusion, this Release provides packages strictly distinguished by version suffix:

```sh
# OpenWrt 25.12+ (apk v3)
scp daed-kdae_2.2.2-R4S-v3.apk root@192.168.2.1:/tmp/
ssh root@192.168.2.1 'apk add --allow-untrusted /tmp/daed-kdae_2.2.2-R4S-v3.apk'

# OpenWrt 24.x / 23.x / Alpine (apk v2)
scp daed-kdae_2.2.2-R4S-v2.apk root@192.168.2.1:/tmp/
ssh root@192.168.2.1 'apk add --allow-untrusted /tmp/daed-kdae_2.2.2-R4S-v2.apk'
```

| Target environment | Package manager | Recommended package |
|---|---|---|
| OpenWrt 25.12+ (apk-tools 3) | `apk` | ✅ `daed-kdae_2.2.2-<device>-v3.apk` (apk v3, ADB container format) |
| OpenWrt 24.x / 23.x / Alpine | `apk` | ✅ `daed-kdae_2.2.2-<device>-v2.apk` (apk v2, traditional tar format) |
| Debian / Ubuntu (standard/older x86_64) | `dpkg` | ✅ `daed-kdae_2.2.2-linux-x86_64_v2_sse.deb` |
| Debian / Ubuntu (modern x86_64 AVX2) | `dpkg` | ✅ `daed-kdae_2.2.2-linux-x86_64_v3_avx2.deb` |

Verified on real hardware (NanoPi R4S / OpenWrt 25.12.5 / apk-tools 3.0.5 / `aarch64_generic`):

```text
(1/1) Upgrading daed (3.1.0-r2 -> 3.1.1-r1)
  Executing daed-3.1.1-r1.post-upgrade
OK: 144.5 MiB in 305 packages
```

Repackaging is performed by the in-repo `workflow_dispatch` workflow **Repack apk as OpenWrt 25.12 (apk v3)** (repackaged with apk-tools 3 from the OpenWrt 25.12 SDK, with the v2 script hooks rewritten as OpenWrt `postinst`/`prerm`) — just re-run it once when updating the upstream binary.

> Note: this package and [rust-daed](https://github.com/Quan-0505/rust-daed) (the Rust engine edition) provide the same `/usr/bin/daed` and `/etc/init.d/daed`, so **only one of the two can be installed**; whichever is installed later overwrites the earlier one.

## 📂 Repository contents

| Directory / File | Description |
|---|---|
| `daed-rebuilt-kdae/` | ★ Rebuilt daed (kdae embedded, x86_64): binary + deb + README (fix log) |
| `dae-kdae-20260901/` | Standalone kdae binary (x86_64 + arm64) + geoip/geosite + dae.service/example.dae |
| `daed-official/` | ksong's official installer (v2.2.2-daed-test reference) |
| `README-部署指引.md` | Full deployment documentation |
| `checksums.txt` | SHA256 checksums for all files |

## 📄 License

[GNU Affero General Public License v3.0](LICENSE). The upstream daed-wing / dae ecosystem is AGPL-3.0.

---
*For the Rust edition (DaedNext engine), see [rust-daed](https://github.com/Quan-0505/rust-daed).*

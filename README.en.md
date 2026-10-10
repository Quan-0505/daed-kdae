<div align="center">

# daed-kdae

**daed (Go edition, kdae engine) all-in-one transparent proxy installer package**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v2.2.3--kdae-orange.svg)](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.3-kdae)

Built on [daed](https://github.com/ksong008/daed) (GraphQL Web) + [dae-wing](https://github.com/daeuniverse/dae-wing) + [kdae](https://github.com/olicesx/dae/tree/kdae) (the eBPF engine from the `kdae` branch of olicesx/dae).

</div>

**English** &nbsp;|&nbsp; **[简体中文](./README.md)**

---

## Current release: v2.2.3-kdae

- Updated kdae from `ee4ce27` to [`f7c1cb6`](https://github.com/olicesx/dae/commit/f7c1cb6b04364903a55dbf9790d6e5eca7d5a628) (2026-10-08), covering 157 commits and the upstream reload architecture changes.
- Adapted dae-wing to `Prepare → Commit → Adopt → Finalize` TC hook handoff, delayed candidate data-plane commit, and routing epoch peer transfer.
- Fixed reload failures including `attach tcx link: file exists`, `has no routing epoch peer`, `TC HookSet transaction already active` on the second reload, and daemon termination caused by the retiring generation's exit notification.

**Validation scope:** Debian 13 / Linux 6.12 with dummy WAN/LAN interfaces: cold startup, GraphQL authentication, and **three consecutive configuration reloads** succeeded. The daemon remained alive, logs contained no error/fatal messages, and `healthCheck=1`. APK structure, architecture, static linking, and symbols were checked; embedded DEB binaries were also checked. **APK installation/startup on actual devices and forwarding through real interfaces remain unverified for this release. It has not been deployed to production devices.** These checks do not establish uninterrupted application traffic.

The features below describe the pinned engine revision; later upstream changes do not automatically enter released packages.

## ✨ Features

- 🖥 **daed Web panel** (GraphQL, embedded binary)
- 📌 **Deeply customized kdae eBPF engine** (based on the [olicesx/dae `kdae`](https://github.com/olicesx/dae/tree/kdae) branch)
- 🔄 **Staged configuration reload**: TC hook handoff and routing epoch peer transfer between generations
- ⚡ **BBRv3 congestion control**: introduced into the outbound path and integrated by default as experimental BBRv3, improving loss resilience and long-fat-pipe throughput
- 🌐 **DNS RFC hardening**: full support for RFC 2308 negative caching, strict `ipversion_prefer`, graceful DoQ stream reclamation
- 🛡️ **Data path hardening**: reduced per-packet map lookup overhead on the eBPF hot path, dynamic re-routing of live flows (re-route), tproxy rate limiting against drops

## ⚡ Latest kdae engine features and evolution

Compared with the original upstream dae, the [olicesx/dae `kdae` branch](https://github.com/olicesx/dae/tree/kdae) embedded in this project contains a large number of deep optimizations tailored for high-performance gateways and soft routers:

| Module | Key evolution and optimizations |
|---|---|
| **Kernel and eBPF hot path** | • Reduced redundant per-packet map lookups on the eBPF hot path, intended to reduce data-plane overhead<br>• The data plane no longer relies on bitfields for parsing and uses single-writer reply binding, making packet-drop state fully observable<br>• tproxy drop event peak shaving and rate limiting plus monotonic timestamps, improving `fast_sock` race handling under high concurrency |
| **Congestion control (BBRv3)** | • Introduced and integrated the **BBRv3** congestion control algorithm by default (flexibly overridable via `cc_override`)<br>• Congestion control improvements for outbound protocols such as QUIC / TUIC / Hysteria; throughput depends on the link and configuration |
| **DNS engine hardening** | • Strictly enforces `ipversion_prefer` (IPv4 / IPv6 preference policy) on all distribution paths<br>• Fully implements the **RFC 2308 Negative Caching** specification, whole-message TTL, and CNAME-only NODATA classification<br>• Improved DoQ (DNS-over-QUIC) multiplexed connection reclamation, avoiding `CancelRead` resource leftovers; supports passing truncated responses (TC=1) through as-is with automatic TCP fallback and retry |
| **Dynamic routing and reload** | • When the routing generation changes, live flows are automatically and gracefully re-routed<br>• Node group (Group availability) health probing and ordering are decoupled, eliminating concurrent callback deadlocks<br>• On config hot reload, the compiled-filter regex cache is automatically cleaned up and reset, preventing hot-reload state pollution |
| **Protocol and edge adaptation** | • Shadowsocks 2022 expired timestamps degrade gracefully to soft endpoint errors, defending against abnormal disconnects triggered by slight clock drift<br>• Hardened REALITY spider scheduling; graceful exit on startup network wait timeout |

## 📦 Installer packages ([v2.2.3-kdae Release](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.3-kdae), deb + apk released together)

| Platform / Device | Package | Format |
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
| Debian / Ubuntu — x86-64-v3 (including AVX2) | [daed-kdae_2.2.3-linux-x86_64_v3_avx2.deb](https://github.com/Quan-0505/daed-kdae/releases/download/v2.2.3-kdae/daed-kdae_2.2.3-linux-x86_64_v3_avx2.deb) | DEB |

## 🚀 Quick start

```sh
# Debian / Ubuntu: choose ONE DEB matching your CPU
sudo apt install ./daed-kdae_2.2.3-linux-x86_64_v2_sse.deb
# Or: sudo apt install ./daed-kdae_2.2.3-linux-x86_64_v3_avx2.deb
sudo systemctl enable --now daed
sudo systemctl status daed --no-pager
```

```sh
# OpenWrt 25.12 / APK v3: R4S example
apk add --allow-untrusted /tmp/daed-kdae-r4s.apk
/etc/init.d/daed enable
/etc/init.d/daed start
```

Download a matching package from the table; upload the OpenWrt example package to `/tmp`. Open `http://<device-IP>:2023`, create an administrator on first startup, then configure interfaces, subscriptions, DNS, and routing. Saving configuration in the WebUI triggers a reload.

## 📋 System requirements

Linux x86_64 / arm64, root privileges, kernel BTF (`/sys/kernel/btf/vmlinux`), and support for the selected data plane (including veth and clsact / TC BPF). The Netkit path also requires kernel Netkit support. Kernel ≥ 5.8 alone does not establish compatibility with this engine; Linux 6.12 is the tested environment.

- `/sys/fs/bpf` must be mounted as bpffs; inspect with `findmnt -T /sys/fs/bpf`.
- DEB declares `iproute2 >= 6.7.0`. Provide the required `ip`, `tc`, and related tools, plus matching OpenWrt kernel modules where applicable.
- Select DEBs by the full x86-64-v2 or x86-64-v3 instruction set; AVX2 alone is insufficient to establish v3 compatibility.

### Package format and compatibility

OpenWrt 25.12 uses APK. Its **APK v3 (ADB)** assets retain the original names, such as `daed-kdae-r4s.apk`, with **no `-v3` suffix**. Original **APK v2** assets are archived as `daed-kdae-r4s-v2.apk`. These package formats are separate from the x86-64-v2/v3 CPU requirements of the DEBs.

[Official OpenWrt 24.10 and older use opkg](https://openwrt.org/docs/guide-user/additional-software/managing_packages), which cannot install these APK files directly. Such systems need a matching IPK or suitable firmware migration. APK v2 does not establish Alpine compatibility: these packages contain OpenWrt service configuration and installation hooks; Alpine installation is unverified. Check the target package manager, architecture, and dependencies before installation.

Repacking uses [repack-apk-v3.yml](.github/workflows/repack-apk-v3.yml) and the OpenWrt SDK: v3 replaces the original asset, and v2 is archived with a `-v2.apk` suffix. This release contains four v3 APKs, four v2 APKs, and two DEBs.

### LAN gateway settings

Check `rp_filter` on `all`, `default`, and the actual LAN interface. Changing only `all` may leave an interface value that blocks downstream forwarding while the host remains connected.

```sh
sudo sysctl -w net.ipv4.conf.all.rp_filter=0
sudo sysctl -w net.ipv4.conf.default.rp_filter=0
sudo sysctl -w net.ipv4.conf.<lan_iface>.rp_filter=0
```

Replace `<lan_iface>` with the actual LAN interface. Persist the settings using the system's sysctl configuration; Debian / Ubuntu can use `/etc/sysctl.d/99-daed.conf`.

### Switching and rollback

This package and [rust-daed](https://github.com/Quan-0505/rust-daed) share `/usr/bin/daed`, the service name, and the `dae0` data plane. **Run only one on a device.** Before switching, stop the existing service and back up its binary, service configuration, `/etc/daed/`, and the `wing.db` configuration database. Keep the old installer; do not assume database compatibility between the products.

After upgrading, check startup logs, WebUI authentication, LAN client connectivity, and consecutive reloads. If validation fails, stop the new service, restore the previous package and its configuration, and start the previous service. Rehearse the first switch on a test device or a single device.

On Debian / Ubuntu, a historical `/etc/systemd/system/daed.service` overrides the packaged unit. Inspect `systemctl cat daed` and `systemctl show daed -p FragmentPath -p CanReload` before handling such an override.

## 📂 Repository contents

| Directory / File | Description |
|---|---|
| `daed-rebuilt-kdae/` | Historical rebuilt binary, DEB, and fix records; use Release assets for current installation |
| `dae-kdae-20260901/` | Standalone kdae binary (x86_64 + arm64) + geoip/geosite + dae.service/example.dae |
| `daed-official/` | ksong's official installer (v2.2.2-daed-test reference) |
| `README-部署指引.md` | Historical deployment guide; use this README and current Release for versions and paths |
| `checksums.txt` | SHA256 list for repository files, not current Release assets |

## 📄 License

[GNU Affero General Public License v3.0](LICENSE). The upstream daed-wing / dae ecosystem is AGPL-3.0.

---
*For the Rust edition (DaedNext engine), see [rust-daed](https://github.com/Quan-0505/rust-daed).*

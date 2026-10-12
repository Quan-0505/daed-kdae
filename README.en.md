# daed-kdae

Go dae-wing and the pinned kdae eBPF engine with an embedded network console.

[English](README.en.md) | [简体中文](README.md) | [Latest release](https://github.com/Quan-0505/daed-kdae/releases/latest)

## v2.2.4-kdae

The new console is the sole production UI. Legacy UI archives, REST demo server and obsolete distribution files have been removed. Existing language and theme preferences are retained.

Features include a responsive dashboard with larger typography, authenticated live logs and filters, persistent logging limits, process metrics, node/subscription group editors with Go RE2 previews, Chinese subscription status, backup import/export, Rust password compatibility and a separate database migration tool.

Geosite and Geoip update together at **04:00 Asia/Shanghai daily**, validate before applying, and automatically reload the engine. Each source supports a proxy switch. Successful scheduled subscription updates coalesce into an automatic reload; a stopped engine stays stopped. Defaults use Loyalsoldier's jsDelivr geodata URLs.

Source and packages contain no device subscriptions, nodes, passwords, tokens, databases, backups or personal configuration. Explicit preview mode uses fictional fixtures; production builds disable it.

## Installation

[Release v2.2.4-kdae](https://github.com/Quan-0505/daed-kdae/releases/tag/v2.2.4-kdae):

- Debian/Ubuntu: `daed-kdae_2.2.4-linux-x86_64_v2_sse.deb` or `daed-kdae_2.2.4-linux-x86_64_v3_avx2.deb` (requires v3/AVX2 CPU support).
- OpenWrt 25.12 APK v3: `daed-kdae-{r2s,r3s,r4s,x86}.apk`.
- Legacy APK v2: corresponding `*-v2.apk` archives; match your firmware's package format.

Open `http://<device-IP>:2023` after installing. Fresh installations require creating an administrator; there is no default account/password. Default Go state is `/etc/daed/`; existing systemd data-directory overrides remain effective. Rust databases require backup and explicit migration into a separate directory; see the [deployment guide](README-部署指引.md).

## Build and validation

```sh
pnpm --dir console install --frozen-lockfile
npm run build
npm run check-types
python3 backend/prepare-wing.py /absolute/path/to/wing --clone
bash packaging/build-release.sh /absolute/path/to/wing "$PWD/dist" /absolute/path/to/output 2.2.4
```

Requires Node.js 24, pnpm 10.24, Go 1.26, Python 3, clang/LLVM, make, curl and dpkg-deb. Production output is `dist/`. `npm run dev:mock` uses fictional fixtures. The SDK workflow converts APK payloads into OpenWrt 25.12 APK v3.

Two Debian 13/Linux 6.12 installations passed login, UI, three reloads, DNS/connectivity, restart and logging checks. Isolated tests covered proxy TCP/UDP, long transfers and rejected candidate configurations preserving the running engine. Package format/architecture/hash checks do not establish OpenWrt hardware compatibility. Traffic metrics measure proxy sessions, not total interface traffic. Existing engine shutdown may log conn_state_map cleanup errors; see [backend details](backend/README.md).

Requires root, BTF and suitable TC/eBPF kernel capabilities. Netkit mode requires Netkit support.

## Sources and licenses

Frontend: [ksong008/daed](https://github.com/ksong008/daed), pinned `4a519fcbaa7004e50e80a309c68d6811596468cf`, [MIT](console/LICENSE). Layout reference: [DaeNext](https://github.com/ksong008/DaeNext).

Controller: [dae-wing](https://github.com/daeuniverse/dae-wing), pinned `dc503088945812c11235b35362d2bfa1a4c3bdf0`. Engine: [kdae](https://github.com/olicesx/dae/tree/kdae), pinned `f7c1cb6b04364903a55dbf9790d6e5eca7d5a628`. Backend follows [AGPL-3.0](LICENSE); component copyrights/licenses are retained. Upstream changes are not automatically included.

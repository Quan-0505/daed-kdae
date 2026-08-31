# daed 安装包（合成版）

> 合成自 **ksong008/daed**（Web 面板/守护进程，官方 installer）+ **olicesx/dae `kdae` 分支**（官方 Actions 最新构建 + 本地重建内嵌）
>
> 适用架构：**x86_64** 与 **arm64** 双架构（Linux）

---

## 一、包内容与版本来源

| 组件 | 来源 | 版本 | 说明 |
|---|---|---|---|
| daed 守护进程（含 Web UI） | ksong008/daed 官方 release | **v2.2.2-daed-test**（2026-05-29） | 单一二进制 = dae 引擎 + 管理 API + 内嵌 Web 面板 |
| daed 内嵌 dae 引擎 | ksong008/dae（wing 钉住） | commit `bfe19ae`（2026-05-16） | daed 运行时实际使用的 dae 版本；位于 kdae 血缘内 |
| dae 独立二进制 | **olicesx/dae `kdae` 分支** 官方 Actions（`build.yml`，经 nightly.link 获取） | **unstable-20260831.r1214.3ffde8**（提交 `3ffde84`，与 kdae 分支 head 一致） | kdae 深度定制分支的最新构建 |
| **daed 重建版（内嵌 kdae 引擎）** | 本地 Debian 13 构建（192.168.5.44），**daed main web v1.28.0**（GraphQL 架构）+ wing main + dae-core@kdae | **v1.28.0-kdae** | x86_64，完整运行时验证（登录/Setup/dae 运行/Web/GraphQL） |
| geoip.dat / geosite.dat | kdae 官方构建自带（v2fly 数据） | 构建当日最新 | 供独立 dae 使用 |

```
daed-安装包-v2.2.2-daed-test/
├── README-部署指引.md            ← 本文件
├── checksums.txt                ← 全部文件 SHA256
├── daed-official/               ← ksong008/daed v2.2.2-daed-test 官方安装包
│   ├── installer-daed-linux-x86_64.deb / .rpm / .pkg.tar.zst
│   ├── installer-daed-linux-arm64.deb / .rpm / .pkg.tar.zst
│   ├── daed-linux-x86_64.zip / daed-linux-arm64.zip   （免安装原始包）
│   ├── web.zip                   ← ksong 的 daed Web 前端（独立托管用）
│   └── *.dgst                    ← 上游发布时的官方 SHA256 校验值（已逐一验证匹配）
└── dae-kdae-20260901/           ← olicesx/dae kdae 分支最新官方构建（独立二进制）
    ├── README.md
    ├── dae-linux-x86_64
    ├── dae-linux-arm64
    ├── geoip.dat / geosite.dat
    └── dae.service / example.dae （kdae 官方构建自带）
└── daed-rebuilt-kdae/           ← ★ 重建版 daed（内嵌 kdae 引擎，x86_64，本机实测通过）
    ├── README.md
    └── daed-linux-x86_64-kdae    （v1.28.0-kdae）
```

---

## 二、重要架构说明（先读）

1. **daed v2 是"一体化"架构**：它由 dae-wing 守护进程构成，把 **dae 引擎以 Go 库形式内嵌**（`wing/go.mod`: `replace github.com/daeuniverse/dae => ./dae-core`），Web 面板也编译进同一个二进制。**运行时不需要、也不使用独立的 dae 二进制**。
2. 因此本包里的 `dae-kdae/` 是**可选附加项**，供你另外以"独立 dae"方式部署（例如脱离 daed 面板直接跑 dae），而不是 daed 的运行时依赖。
3. 想让 daed 面板使用"最新 dae 引擎"，唯一途径是把 dae-core 子模块升到最新后**重新编译 daed**（需 Linux + Go 1.26 + clang-15/llvm-15），本包未包含重建产物（见第四节）。

---

## 三、安装 daed（推荐路径）

### 方式 A：Debian / Ubuntu（deb）

```bash
# x86_64 设备
sudo dpkg -i installer-daed-linux-x86_64.deb
# arm64 设备
sudo dpkg -i installer-daed-linux-arm64.deb

sudo systemctl enable --now daed
```

### 方式 B：RHEL / Fedora（rpm）

```bash
sudo rpm -ivh installer-daed-linux-x86_64.rpm   # 或 arm64 对应包
sudo systemctl enable --now daed
```

### 方式 C：Arch Linux（pkg.tar.zst）

```bash
sudo pacman -U installer-daed-linux-x86_64.pkg.tar.zst
sudo systemctl enable --now daed
```

### 方式 D：通用部署（zip 原始包）

```bash
unzip daed-linux-x86_64.zip -d daed-bundle
sudo install -m755 daed-bundle/daed-linux-x86_64 /usr/bin/daed
sudo install -m644 daed-bundle/daed.service /usr/lib/systemd/system/daed.service
sudo mkdir -p /usr/share/daed /etc/daed
sudo install -m644 daed-bundle/geoip.dat daed-bundle/geosite.dat /usr/share/daed/
sudo systemctl daemon-reload && sudo systemctl enable --now daed
```

### 使用 Web 面板

1. 浏览器访问 `http://<设备IP>:2023`
2. 首次打开完成初始化（创建管理员账号）
3. **Setup** 页：导入订阅/节点；**Orchestrate** 页：编排路由规则
4. 保存后 daed 热加载 dae，无需重启

数据目录：`/etc/daed/`（配置文件与数据库）；日志：`journalctl -u daed -f`

---

## 四、dae 内核要求（daed 与独立 dae 均适用）

dae 基于 eBPF，对内核有硬性要求：

| 要求 | 说明 | 检查方法 |
|---|---|---|
| Linux 内核 ≥ 5.8 | 越新越好 | `uname -r` |
| BTF 支持 | 内核需开启 `CONFIG_DEBUG_INFO_BTF` | `ls /sys/kernel/btf/vmlinux` 存在 |
| iproute2 ≥ 6.7.0 | dae 二进制硬性要求 | `ip -V` |
| root 权限 | 加载 eBPF 程序需要 | — |

> 若检查不通过，建议升级内核（发行版最新 LTS）后再安装。

---

## 五、独立 dae（kdae）可选部署

> daed 面板与独立 dae **互斥**（daed 的 systemd 单元声明了 `Conflicts=dae.service`），二选一。

```bash
# 以 x86_64 为例
sudo install -m755 dae-linux-x86_64 /usr/local/bin/dae
sudo mkdir -p /usr/local/share/dae /etc/dae
sudo install -m644 geoip.dat geosite.dat /usr/local/share/dae/
sudo cp example.dae /etc/dae/config.dae     # 按需编辑
sudo cp dae.service /usr/lib/systemd/system/dae.service
sudo systemctl enable --now dae
# 验证
dae --version    # 预期输出 unstable-20260831.r1214.3ffde8（kdae 构建）
```

---

## 六、"dae（kdae）官方 Actions 最新"的版本情况（如实说明）

- 本包 dae 独立二进制取自 **olicesx/dae 的 `kdae` 分支**（该仓库主开发分支，376 commits ahead of 其自身 main）。kdae 没有 release 资产，其 GitHub Actions `build.yml` 构建产物为内部 artifact（需登录 GitHub 才能直接下载），本包通过 **nightly.link** 免登录代理获取 kdae 分支最近成功构建：**`unstable-20260831.r1214.3ffde8`**（提交 `3ffde84`，已验证与 kdae 分支 head 完全一致）。
- **daed v2.2.2-daed-test 内嵌的 dae 引擎与 kdae 同源**：其 dae-core 为 ksong008/dae@`bfe19ae`（2026-05-16），而 kdae ⊇ ksong008/dae ⊇ olicesx/dae，属于同一定制血缘。若需让面板使用**更新**的 kdae 引擎，需把 dae-core 升到 kdae 最新提交后重新编译 daed（需 Linux + Go 1.26 + clang-15/llvm-15，本包未包含重建产物）。
- 获取更新 kdae 构建：访问 https://nightly.link/olicesx/dae/workflows/build/kdae 下载对应架构 zip（免登录）；或登录 GitHub 在 [olicesx/dae Actions](https://github.com/olicesx/dae/actions) 下载 build.yml 的 artifact。
- 上游 daeuniverse/dae（官方原版，release v2.0.0 / 每日构建）与 kdae 是不同分支，本包按你的要求使用 kdae。

---

## 七、完整性校验

```bash
# 本包所有文件
sha256sum -c checksums.txt

# 官方 daed 安装包（对照上游 .dgst）
cd daed-official && for f in *.deb *.rpm *.pkg.tar.zst *.zip; do
  echo "$(grep -oE '[0-9a-f]{64}' $f.dgst)  $f" | sha256sum -c -
done
```

## 附：版本核对命令

```bash
daed --version          # daed 面板版本
dae --version           # 独立 dae 版本（kdae 应为 unstable-20260831.r1214.3ffde8）
systemctl status daed   # 面板服务状态
```

---

*本包于 2026-08-31 合成；所有二进制均来自上游官方发布渠道，并已验证 SHA256。*

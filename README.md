# daed kdae

**daed v1.28.0（ksong daed web，GraphQL）+ olicesx/dae kdae 引擎重建安装包**，完整安装文件已解压入库，可直接下载使用。

## 仓库内容

| 目录/文件 | 说明 |
|---|---|
| `daed-rebuilt-kdae/` | ★ 重建版 daed（内嵌 kdae 引擎，x86_64）：`daed-linux-x86_64-kdae`（59.7MB 裸二进制）+ `daed_1.28.0-kdae_amd64.deb`（22.3MB 一键安装包）+ README（含修复记录） |
| `dae-kdae-20260901/` | 独立 kdae 二进制 `dae-linux-x86_64` + `dae-linux-arm64`（unstable-20260831.r1214.3ffde8）+ geoip.dat/geosite.dat 规则数据 + dae.service/example.dae |
| `daed-official/` | ksong 官方 installer（v2.2.2-daed-test，参考用）：x86_64 + arm64 的 deb/rpm/pkg.tar.zst/zip + *.dgst 校验文件 |
| `README-部署指引.md` | 完整部署文档（架构说明、两种部署方式、验证方法） |
| `checksums.txt` | 全部文件 SHA256 校验和（`sha256sum -c checksums.txt` 验证） |

> **deb 直链**（Release）：https://github.com/Quan-0505/daed-kdae-installer/releases/download/v1.28.0-kdae/daed_1.28.0-kdae_amd64.deb

## 快速安装（Debian/Ubuntu x86_64）

```bash
# 方式一：Release deb 一键安装
sudo dpkg -i daed_1.28.0-kdae_amd64.deb

# 方式二：仓库内 deb
sudo dpkg -i daed-rebuilt-kdae/daed_1.28.0-kdae_amd64.deb

# 安装后自动启动服务；浏览器访问 http://<机器IP>:2023
# 首次访问初始化账户 → 登录 → 导入订阅
# 卸载：sudo dpkg -P daed
```

## 系统要求

- x86_64 Linux（Debian/Ubuntu）
- 内核 ≥ 5.8 且支持 BTF（`ls /sys/kernel/btf/vmlinux` 存在）
- iproute2 ≥ 6.7.0
- root 权限（eBPF 必需）

## 版本信息

- Web 面板：ksong008/daed main 分支 v1.28.0（GraphQL 架构）
- dae 引擎：olicesx/dae `kdae` 分支 @ `3ffde84`（已验证与分支 head 一致）
- 含 3 组 kdae API 适配补丁（routing-epoch 发布 / IsReload flip handle / 构造器与运行时 API），透明代理实测通过（baidu 直连 0.07s、google/youtube 走代理 1-2s）

## License

本项目基于 [GNU General Public License v3.0](LICENSE) 发布。

> 说明：本仓库为 daed（ksong daed web）+ dae（kdae 分支）重建安装包的发布仓库；上游组件
> [dae](https://github.com/daeuniverse/dae) 为 GPL-3.0，[dae-wing](https://github.com/daeuniverse/dae-wing) 为 AGPL-3.0，
> [daed](https://github.com/ksong008/daed) 为 GPL-3.0，使用与分发请遵循对应许可证。

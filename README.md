# daed-kdae-installer

**daed v1.28.0（ksong daed web，GraphQL）+ olicesx/dae kdae 引擎重建安装包**

- Web 面板：ksong008/daed main 分支（v1.28.0）
- dae 引擎：olicesx/dae `kdae` 分支 @ `3ffde84`（已验证与分支 head 一致）
- 架构：x86_64（重建版）+ arm64（官方独立二进制）

## 快速安装（Debian/Ubuntu x86_64）

```bash
sudo dpkg -i daed_1.28.0-kdae_amd64.deb
# 浏览器访问 http://<机器IP>:2023 → 初始化账户 → 导入订阅
```

## 系统要求

- x86_64 Linux，内核 ≥ 5.8 且支持 BTF（`ls /sys/kernel/btf/vmlinux` 存在）
- iproute2 ≥ 6.7.0，root 权限

完整安装包见 **Releases**。
## License

本项目基于 [GNU General Public License v3.0](LICENSE) 发布。

> 说明：本仓库为 daed（ksong daed web）+ dae（kdae 分支）重建安装包的发布仓库；上游组件
> [dae](https://github.com/daeuniverse/dae) 为 GPL-3.0，[dae-wing](https://github.com/daeuniverse/dae-wing) 为 AGPL-3.0，
> [daed](https://github.com/ksong008/daed) 为 GPL-3.0，使用与分发请遵循对应许可证。

# dae（kdae 分支）最新官方构建 — 2026-09-01

| 项目 | 值 |
|---|---|
| 来源 | **olicesx/dae** 仓库 **`kdae` 分支**（GitHub Actions `build.yml` 构建产物，经 nightly.link 免登录获取） |
| 版本 | **unstable-20260831.r1214.3ffde8** |
| 对应提交 | `3ffde84`（`fix(cmd,trace): close reload races and bound trace skb tracking`，**与 kdae 分支 head 完全一致**） |
| 架构 | x86_64（`dae-linux-x86_64`）、arm64（`dae-linux-arm64`） |

## kdae 是什么

`kdae` 是 [olicesx/dae](https://github.com/olicesx/dae/tree/kdae) 仓库的主开发分支，是 **dae 的深度定制分支**，包含大量上游没有的补丁（TCP relay half-close 超时控制、路由 epoch 槽位、fast_sock 内核队列排水、ECH 配置支持、quic-go/outbound 协议加固等）。本目录二进制为**独立 dae** 用途（daed 面板内嵌引擎与此同源同版本）。

## 文件

- `dae-linux-x86_64` / `dae-linux-arm64` — dae 可执行文件（静态 ELF）
- `geoip.dat` / `geosite.dat` — 规则数据（构建时最新）
- `dae.service` / `example.dae` — 官方构建自带（systemd 部署与配置示例）

## 验证

- 二进制版本字符串（双架构一致）：`unstable-20260831.r1214.3ffde8`
- 独立部署：`sudo install -m755 dae-linux-x86_64 /usr/local/bin/dae` + 数据放 `/usr/local/share/dae/` + `dae.service`（详见上层 README 第五节）。

## 说明

- kdae 分支持续更新；更新构建：访问 https://nightly.link/olicesx/dae/workflows/build/kdae 下载对应架构 zip。
- daed 重建版（`daed-rebuilt-kdae/`）内嵌的引擎与本文档同一提交（3ffde84）。

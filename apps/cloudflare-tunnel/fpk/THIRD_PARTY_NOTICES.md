# Third-party notices — Cloudflare Tunnel for fnOS

本包（`cloudflare-tunnel` fpk）内嵌下列第三方组件。各组件的许可证原文一并随包分发。

## 1. cloudflared

- 项目：`cloudflare/cloudflared`
- 版本：**2026.10.7**（`linux-amd64`，随包放在 `app/bin/cloudflared`）
- 来源：<https://github.com/cloudflare/cloudflared/releases/tag/2026.10.7>
- 许可证：**Apache License 2.0**
- 许可证全文：`app/third_party/cloudflared/LICENSE`（仓库内为 `fpk/third_party/cloudflared/LICENSE`）
- 说明：cloudflared 是独立可执行文件，由本应用作为子进程启动（`cloudflared tunnel --no-autoupdate run`），
  未做任何修改，也未静态链接进本应用的 Go 二进制。
- 版权：Copyright (c) Cloudflare, Inc.

## 2. 本应用自身代码

本应用自身代码（Go 后端 + Vue 前端）在撰写本文件时**尚未选定开源许可证**，
因此包内不包含本应用自己的 `LICENSE`。仓库 owner 决定后应补上，并在此处登记。

## 3. 前端依赖（随 `web/dist` 一起被嵌入 Go 二进制）

前端产物由本仓库 `web/` 构建，包含以下直接依赖：

- **Vue 3** — MIT License，<https://github.com/vuejs/core>
- **Vite** — MIT License，<https://github.com/vitejs/vite>（仅构建期）
- **@fontsource/manrope** — Manrope 字体，SIL Open Font License 1.1，<https://github.com/sharanda/manrope>

### 3.1 内置字体（离线打包，NAS 无外网）

- **Maple Mono NF CN** — 等宽字体，SIL Open Font License 1.1，
  来源 <https://github.com/subframe7536/maple-font>。
  本应用只取 UI 所需的 Latin/符号子集并转为 WOFF2，属于 OFL 允许的修改；
  因 OFL 保留字体名 “Maple Mono”，子集在 CSS 中改名为 **CT Mono**。许可证全文：
  `app/third_party/fonts/LICENSE-MapleMono.txt`（仓库内 `web/public/fonts/LICENSE-MapleMono.txt`）。
- **MiSans** — 中文字体，小米 MiSans 字体知识产权许可协议，
  来源 <https://hyperos.mi.com/font/zh/>。按许可要求以**未修改的原始 TTF**
  （Regular/Semibold/Bold）随应用嵌入，仅用于本应用界面，不单独售卖或再分发；
  不用于商标注册。声明全文：
  `app/third_party/fonts/NOTICE-MiSans.txt`（仓库内 `web/public/fonts/NOTICE-MiSans.txt`）。

完整依赖树与各组件许可证随 `web/pnpm-lock.yaml` 固定。其余 MIT 依赖如需在包内附带
许可证全文，请从各依赖的 `node_modules/<pkg>/LICENSE` 复制到本文件同级目录。

## 4. Go 标准库

Go 标准库以 BSD-3-Clause 发布（<https://golang.org/LICENSE>），其代码已编译进
`app/cloudflare-tunnel-server`。

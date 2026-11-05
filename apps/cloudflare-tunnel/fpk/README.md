# cloudflare-tunnel fpk — 打包骨架

fnOS 原生应用打包目录（复制自 `fndepot/app-template` 并适配）。
本目录只负责**打包**：程序本体来自 `../server`（Go），前端来自 `../web`（Vue），两者都由 `build.sh` 收进来。

- appname：`cloudflare-tunnel`
- 显示名：`Cloudflare Tunnel`
- 版本：`2026.10.7-10`（上游 `cloudflare/cloudflared` 2026.10.7 + 本地版本号 10）
- 服务端口：`18960`（`manifest service_port`；运行时以 `$TRIM_SERVICE_PORT` 为准）
- 平台：`x86`（= amd64，`GOARCH=amd64`）

## 结构

```
fpk/
├── manifest                         INI 元信息（版本、端口、maintainer/distributor、changelog）
├── config/
│   ├── privilege                    run-as: package（官方唯一推荐值）
│   └── resource                     data-share: cloudflare-tunnel ← 数据落 share 的唯一入口
├── ui/
│   ├── config                       桌面入口 iframe（cloudflare-tunnel.panel，protocol ""，port 18960）
│   └── images/icon-{64,256,1024}.png
├── ICON.PNG / ICON_256.PNG          商店图标（256×256）
├── icon.svg                         图标源（矢量，与 tools/make_icons.py 同一套几何）
├── wizard/
│   ├── install                      安装向导：Token 权限说明 + wizard_port（默认 18960）
│   └── uninstall                    卸载向导：保留 / 清除数据（wizard_delete_data）
├── cmd/                             9 个生命周期脚本（缺任一 → fnpack 直接报错）
│   ├── main                         start/stop/restart/status + PID 文件 + 日志
│   ├── install_init / install_callback
│   ├── upgrade_init / upgrade_callback
│   ├── uninstall_init / uninstall_callback
│   └── config_init / config_callback
├── third_party/
│   ├── cloudflared/LICENSE          cloudflared（Apache-2.0）全文
│   ├── cloudflared.sha256           下载校验（已填入 2026.10.7 官方 sha256）
│   └── cloudflared-linux-amd64      （可选）本地预置的上游二进制，build.sh 优先用它
├── THIRD_PARTY_NOTICES.md
├── tools/make_icons.py              图标生成（Pillow；无 SVG 栅格化器时用它）
├── tools/fix_fpk_modes.py           Windows fnpack 的 0666 修正（重写 mode + 同步 checksum）
├── build.sh                         全流程构建
└── app/                             构建产物（由 build.sh 生成，不进版本库）
```

## 构建

在 `fpk/` 下：

```bash
# 默认产物：<app>/dist/cloudflare-tunnel-2026.10.7-10-x86.fpk
GO=/path/to/go ./build.sh

# 指定产物路径 / 本地 cloudflared / 其它 pnpm
./build.sh /tmp/out.fpk
CLOUDFLARED_BIN=/path/to/cloudflared-linux-amd64 ./build.sh
PNPM=/path/to/pnpm GO=/path/to/go ./build.sh
```

`build.sh` 做的事（顺序）：

1. 前置检查：9 个 cmd 脚本、config/、manifest 版本、图标、**备份文件闸门**（`.bak*` / `.orig` / `~` 一律 FATAL）。
2. `../web`：`pnpm install --frozen-lockfile`（失败回退 `pnpm install`）→ `pnpm build`。
3. `web/dist` → `../server/internal/assets/dist`，再 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o <可写构建目录>/cloudflare-tunnel-server ./cmd/cloudflare-tunnel`，最后拷进 `app/`（`CT_BUILD_TMP` 可指定可写目录）。
4. cloudflared：`$CLOUDFLARED_BIN` → `third_party/cloudflared-linux-amd64` → 从 GitHub Release 下载；与 `third_party/cloudflared.sha256` 比对（占位符时跳过并打印实测值）。
5. 组装 `app/`（server、bin/cloudflared、ui/config、ui/images、THIRD_PARTY_NOTICES、cloudflared LICENSE）。
6. `fnpack build -d $HERE`（在系统临时目录里调用，产物不落回源码目录）。
   Windows（MINGW/MSYS）上自动执行 `tools/fix_fpk_modes.py`：外层 `cmd/*` 0755、其他 0644，
   `app.tgz` 内二进制 0755、其余 0644，并把 manifest 的 `checksum` 同步为新的 app.tgz MD5。
7. 拷到 OUT 后自检：fpk 清单、9 脚本、`app.tgz` 内容、包内版本、`resolve_share_root`、无备份、
   `ui/config` 的 `protocol: ""`、`cmd/main` 执行位、manifest checksum 与 app.tgz 一致；
   打印大小 + sha256。**永不自动安装。**

## 运行时布局

| 位置 | 内容 |
| --- | --- |
| `/var/apps/cloudflare-tunnel/shares/cloudflare-tunnel/config/settings.json` | 账号 ID、API Token、偏好（0600） |
| `…/config/apikeys.json` | API Key 元数据 + SHA-256 哈希（0600） |
| `…/data/state.json` | 上次隧道 / 状态（开机重连用） |
| `…/data/activity.json` | 最近 100 条活动 |
| `$TRIM_PKGVAR/logs/app.log` | server stdout/stderr（易失） |
| `$TRIM_PKGVAR/logs/cloudflared.log` | cloudflared 日志镜像（JSON lines） |
| `$TRIM_PKGVAR/app.pid` | server PID |
| `$TRIM_PKGVAR/{install,upgrade,config,uninstall}.log` | 生命周期脚本日志 |

`cmd/main` 给 server 的环境变量：

```
CT_LISTEN=0.0.0.0:${TRIM_SERVICE_PORT:-18960}
CT_CONFIG_DIR=<share>/config
CT_DATA_DIR=<share>/data
CT_LOG_DIR=$TRIM_PKGVAR/logs
CT_CLOUDFLARED=$APP_DIR/bin/cloudflared
CT_METRICS_PORT=20241
```

share 根目录解析：`/var/apps/cloudflare-tunnel/shares/cloudflare-tunnel`（fnOS 维护的软链，优先）→
`${TRIM_DATA_SHARE_PATHS%%:*}`（首段），拒绝 `/vol0/*` 占位路径。**四个脚本各带一份 `resolve_share_root`**，不 `source` 兄弟文件（卸载阶段 `cmd/` 可能已被清）。

进程模型：`cloudflare-tunnel-server` 是唯一被 PID 管理的进程；cloudflared 由 server 自己 spawn / 监控 / 回收。
`cmd/main stop` 对 server 发 SIGTERM 并最多等 10 秒，再 SIGKILL；最后按 `$APP_DIR/bin/cloudflared` 路径兜杀残留孤儿。

## 发布前必须做的事

- [ ] **许可证（本仓库 owner 待定）**：本应用自身代码的许可证**尚未选定**，这里刻意没有放 `LICENSE`。
      `TODO(owner)`：确定后把 `LICENSE` 放到仓库根，并同步 `THIRD_PARTY_NOTICES.md`。
      第三方部分已经就位：cloudflared 是 Apache-2.0，全文在 `third_party/cloudflared/LICENSE`。
- [x] `third_party/cloudflared.sha256` 已填入官方 2026.10.7 sha256。
- [x] Windows 打包权限：`build.sh` 在 fnpack 之后自动调用 `tools/fix_fpk_modes.py`，把 `cmd/*` 改成 0755、
      其它文件 0644，并同步 manifest 的 app.tgz MD5。自检会验证执行位与 checksum。
      若本机没有 Python，退回 Linux/WSL 用原生 `fnpack` 重打。
- [ ] 装机验证：`tar tzf` / `app.tgz` 内容、`ui/config` 的 `protocol: ""`、DB 里 `app_open` = `(app_id,'iframe','cloudflare-tunnel.panel')`。

## 已知风险

- **fpk 根目录的开发期文件**：`third_party/`、`tools/`、`icon.svg`、`README.md`、`THIRD_PARTY_NOTICES.md`、`.gitignore`
  都不是 fnOS 认识的打包项。fnpack 1.2.3 通常忽略未知条目，但若 `fnpack build` 报根目录非法，
  先把它们移到 `fpk/` 之外（`third_party/cloudflared/LICENSE` 与 `THIRD_PARTY_NOTICES.md` 会被
  `build.sh` 拷进 `app/`，移走不影响包内容）。
- **执行位**：Windows fnpack 固定输出 0666（Git Bash 的 `chmod +x` 它读不到），fnOS 会报 10111。
  `build.sh` 已自动后处理；不要手工跳过 `tools/fix_fpk_modes.py`。
- **`chmod 700` 在本机无效**：Windows 文件系统不保留 POSIX mode，`config/` 的 0700 只在真机 ext4 上生效；
  `settings.json` / `apikeys.json` 的 0600 由服务端自己写。

## 打包后自检（人工复核用）

```bash
tar tzf dist/cloudflare-tunnel-2026.10.7-10-x86.fpk | head -30
tar xzOf dist/cloudflare-tunnel-2026.10.7-10-x86.fpk app.tgz | tar tzf - | grep -E 'cloudflare-tunnel-server|bin/cloudflared|ui/'
tar xOf dist/cloudflare-tunnel-2026.10.7-10-x86.fpk manifest | grep -E '^(appname|version|maintainer|distributor|service_port)'
```

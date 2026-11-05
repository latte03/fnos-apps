#!/bin/bash
# build.sh — cloudflare-tunnel 全流程构建（web SPA → Go server → cloudflared → fnpack）
#
# 用法（在 fpk/ 目录下）:
#   ./build.sh                                   # 产物 dist/cloudflare-tunnel-2026.10.7-10-x86.fpk
#   ./build.sh /path/to/out.fpk                  # 指定产物路径
#   GO=/path/to/go ./build.sh
#   PNPM=/path/to/pnpm ./build.sh
#   CLOUDFLARED_BIN=/path/to/cloudflared-linux-amd64 ./build.sh
#
# 铁律：本脚本**永不安装**到 NAS，只产出 fpk，装哪由用户决定。
#
# 流程：
#   1. 前置检查（模板 9 脚本、config、manifest 版本、备份文件闸门）
#   2. web: pnpm install --frozen-lockfile（失败回退 pnpm install）+ pnpm build → dist
#   3. server: web/dist → server/internal/assets/dist，GOOS=linux GOARCH=amd64 交叉编译进 app/
#   4. cloudflared: $CLOUDFLARED_BIN / third_party/cloudflared-linux-amd64 / 下载 → app/bin/
#   5. 组装 app/（ui/config、图标、THIRD_PARTY_NOTICES、Apache-2.0 全文）
#   6. fnpack build（在临时目录里调用，避免产物落在源码目录）
#   7. 自检（tar 清单、版本、resolve_share_root、无备份）+ 打印 size/sha256
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
PRODUCT_DIR="$(cd "$HERE/.." && pwd)"
SERVER_DIR="$PRODUCT_DIR/server"
WEB_DIR="$PRODUCT_DIR/web"

APP_NAME="cloudflare-tunnel"
DISPLAY_NAME="Cloudflare Tunnel"
VERSION="2026.10.7-10"
ARCH="x86"
UPSTREAM_CLOUDFLARED_VERSION="2026.10.7"
SERVER_BIN_NAME="cloudflare-tunnel-server"

OUT="${1:-${OUT:-$PRODUCT_DIR/dist/${APP_NAME}-${VERSION}-${ARCH}.fpk}}"

GO="${GO:-go}"
PNPM="${PNPM:-pnpm}"
CLOUDFLARED_URL="${CLOUDFLARED_URL:-https://github.com/cloudflare/cloudflared/releases/download/${UPSTREAM_CLOUDFLARED_VERSION}/cloudflared-linux-amd64}"

APP_OUT="$HERE/app"
# fnpack 把产物写到调用时的 cwd —— 用系统临时目录，避免 .fpk 落回源码目录被下次打包带上
BUILD_TMP="${TMPDIR:-/tmp}/${APP_NAME}-fnpack"
# Go 在某些受控环境下不能直接写工作区；允许用 CT_BUILD_TMP 指定一个可写的构建目录。
BUILD_BIN_DIR="${CT_BUILD_TMP:-$HERE/.build-tmp}"
SHA_FILE="$HERE/third_party/cloudflared.sha256"
PLACEHOLDER_SHA="0000000000000000000000000000000000000000000000000000000000000000"

say() { printf '%s\n' "$*"; }
warn() { printf '⚠️  %s\n' "$*" >&2; }
fatal() { printf 'FATAL: %s\n' "$*" >&2; exit 1; }

file_size() {
  if stat -c '%s' "$1" >/dev/null 2>&1; then stat -c '%s' "$1"
  else wc -c < "$1" | tr -d ' \n'; fi
}
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | cut -d' ' -f1
  else echo "unavailable"; fi
}
# 注意：不要用 `| grep -q`——grep 命中即退管道会让左侧收 SIGPIPE(141)，
# 在 set -o pipefail 下整条被判失败（误报“缺文件”）。先取进变量，再用本函数判。
has() { printf '%s\n' "$1" | grep -E -- "$2" >/dev/null 2>&1; }

cd "$HERE"

# ---------------------------------------------------------------- 0. 前置检查
say "== 0. 前置检查 =="

for f in cmd/main cmd/install_init cmd/install_callback cmd/upgrade_init cmd/upgrade_callback \
         cmd/uninstall_init cmd/uninstall_callback cmd/config_init cmd/config_callback \
         config/privilege config/resource manifest wizard/install wizard/uninstall \
         ui/config ui/images/icon-64.png ui/images/icon-256.png \
         ICON.PNG ICON_256.PNG \
         third_party/cloudflared/LICENSE THIRD_PARTY_NOTICES.md; do
  [ -e "$f" ] || fatal "缺必需文件 $f"
done

# 9 个 cmd 脚本必须可执行（缺 exec 位 → 应用中心 10111 / 启动即失败）
chmod +x cmd/* 2>/dev/null || true
for f in cmd/*; do
  [ -x "$f" ] || warn "$f 没有执行位（Windows 上目录可能不保留 mode，打包前请在 WSL/Linux 上 chmod +x cmd/*）"
done

# manifest 版本必须与 VERSION 一致（不就地改 manifest：它已入库，避免构建改源码）
MANIFEST_VER="$(grep -E '^version[[:space:]]*=' manifest | head -1 | sed 's/.*=[[:space:]]*//' | tr -d '\r' | tr -d '[:space:]' || true)"
[ "$MANIFEST_VER" = "$VERSION" ] || fatal "manifest version=${MANIFEST_VER:-<空>} 与 VERSION=$VERSION 不一致"
say "manifest 版本 = $VERSION"

# 闸门 1 —— 备份文件污染：fnpack 会把 app/ 与 cmd/ 整个塞进包，.bak 会跟着上机
BAK="$(find "$HERE" -type f \( -name '*.bak*' -o -name '*.orig' -o -name '*~' \) 2>/dev/null | head -20 || true)"
if [ -n "$BAK" ]; then
  printf 'FATAL: 目录里有备份文件，先移到仓库外再打包：\n%s\n' "$BAK" >&2
  exit 1
fi
say "无备份文件污染"

command -v fnpack >/dev/null 2>&1 || fatal "fnpack 不在 PATH（fnOS fnpack，v1.2+）"

# ---------------------------------------------------------------- 1. web SPA
say "== 1. web 构建（$WEB_DIR）=="
[ -d "$WEB_DIR" ] || fatal "$WEB_DIR 不存在（前端尚未落地）"
[ -f "$WEB_DIR/package.json" ] || fatal "$WEB_DIR/package.json 不存在"

cd "$WEB_DIR"
LOCK_ARGS=""
if [ -f pnpm-lock.yaml ]; then LOCK_ARGS="--frozen-lockfile"; fi
# pnpm 10+ 会把「依赖的 postinstall 未批准」当成 install 失败（ERR_PNPM_IGNORED_BUILDS），
# 但 esbuild 这类包的实际产物走 optionalDependencies，不需要跑脚本 → 先按宽松模式装。
run_install() {
  "$PNPM" install $LOCK_ARGS --config.strict-dep-builds=false && return 0
  warn "pnpm install $LOCK_ARGS --config.strict-dep-builds=false 失败，换参数重试"
  "$PNPM" install $LOCK_ARGS && return 0
  warn "pnpm install $LOCK_ARGS 失败，回退无锁模式"
  "$PNPM" install && return 0
  return 1
}
if ! run_install; then
  if [ -d node_modules ]; then
    warn "pnpm install 返回非 0，但 node_modules 已存在，继续尝试构建"
  else
    fatal "pnpm install 失败（$WEB_DIR），前端依赖装不上"
  fi
fi
"$PNPM" build || fatal "pnpm build 失败（$WEB_DIR），见上方 vite 报错"
[ -f "$WEB_DIR/dist/index.html" ] || fatal "web 构建未产出 dist/index.html"
say "web dist 就绪：$WEB_DIR/dist"

# ---------------------------------------------------------------- 2. server
cd "$HERE"   # 上一步 cd 到了 web/，后面全用 fpk/ 的相对路径
say "== 2. server 交叉编译（GO=$GO）=="
[ -d "$SERVER_DIR" ] || fatal "$SERVER_DIR 不存在（后端尚未落地）"
[ -d "$SERVER_DIR/cmd/$APP_NAME" ] || fatal "$SERVER_DIR/cmd/$APP_NAME 不存在（Go main 包路径）"

ASSETS_DST="$SERVER_DIR/internal/assets/dist"
rm -rf "$ASSETS_DST"
mkdir -p "$ASSETS_DST"
cp -a "$WEB_DIR/dist/." "$ASSETS_DST/"
say "web/dist → server/internal/assets/dist（$(find "$ASSETS_DST" -type f | wc -l | tr -d ' ') 个文件）"

rm -rf "$APP_OUT"
mkdir -p "$APP_OUT/bin" "$APP_OUT/ui/images" "$APP_OUT/third_party/cloudflared" "$BUILD_BIN_DIR"
(
  cd "$SERVER_DIR"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$GO" build \
    -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o "$BUILD_BIN_DIR/$SERVER_BIN_NAME" \
    ./cmd/"$APP_NAME"
) || fatal "go build 失败（GO=$GO，GOOS=linux GOARCH=amd64）—— 检查 GO 路径是否可执行、./cmd/$APP_NAME 是否存在"
[ -f "$BUILD_BIN_DIR/$SERVER_BIN_NAME" ] || fatal "go build 未产出 $BUILD_BIN_DIR/$SERVER_BIN_NAME"
cp "$BUILD_BIN_DIR/$SERVER_BIN_NAME" "$APP_OUT/$SERVER_BIN_NAME"

# 闸门 2 —— 主程序必须在 app/ 里且像真二进制：漏编译 = 几十 KB 空包，装上直接瘫
SERVER_SIZE="$(file_size "$APP_OUT/$SERVER_BIN_NAME")"
[ "$SERVER_SIZE" -gt 1000000 ] || fatal "$SERVER_BIN_NAME 只有 ${SERVER_SIZE} 字节（<1MB），不像真二进制"
say "$SERVER_BIN_NAME 大小 = $SERVER_SIZE 字节"

# ---------------------------------------------------------------- 3. cloudflared
say "== 3. cloudflared（upstream $UPSTREAM_CLOUDFLARED_VERSION, linux-amd64）=="
CF_SRC=""
if [ -n "${CLOUDFLARED_BIN:-}" ]; then
  [ -f "$CLOUDFLARED_BIN" ] || fatal "CLOUDFLARED_BIN=$CLOUDFLARED_BIN 不存在"
  CF_SRC="$CLOUDFLARED_BIN"
  say "使用 CLOUDFLARED_BIN=$CF_SRC"
elif [ -f "$HERE/third_party/cloudflared-linux-amd64" ]; then
  CF_SRC="$HERE/third_party/cloudflared-linux-amd64"
  say "使用 third_party/cloudflared-linux-amd64"
else
  CF_SRC="$HERE/third_party/cloudflared-linux-amd64"
  say "本地无 cloudflared，下载：$CLOUDFLARED_URL"
  command -v curl >/dev/null 2>&1 || fatal "没有 curl，无法下载 cloudflared（请设置 CLOUDFLARED_BIN）"
  curl -fL --retry 3 --connect-timeout 20 -o "$CF_SRC.part" "$CLOUDFLARED_URL" \
    || fatal "cloudflared 下载失败（可先用 CLOUDFLARED_BIN 指定本地文件）"
  mv "$CF_SRC.part" "$CF_SRC"
fi

CF_COMPUTED="$(sha256_of "$CF_SRC")"
CF_PLACEHOLDER=0
CF_EXPECTED=""
if [ -f "$SHA_FILE" ]; then
  CF_EXPECTED="$(grep -viE '^[[:space:]]*(#|$)' "$SHA_FILE" | awk '{print tolower($1)}' | head -1 || true)"
fi
if [ -z "$CF_EXPECTED" ]; then CF_PLACEHOLDER=1; fi
if [ "$CF_EXPECTED" = "$PLACEHOLDER_SHA" ]; then CF_PLACEHOLDER=1; fi
if [ "${#CF_EXPECTED}" -ne 64 ]; then CF_PLACEHOLDER=1; fi
case "$CF_EXPECTED" in *[!0-9a-f]*) CF_PLACEHOLDER=1 ;; esac

if [ "$CF_PLACEHOLDER" -eq 1 ]; then
  warn "third_party/cloudflared.sha256 还是占位符/无效值，跳过校验"
  say "   实测 sha256 = $CF_COMPUTED"
  say "   发布前请把该值写进 third_party/cloudflared.sha256（64 位十六进制）"
else
  [ "$CF_EXPECTED" = "$CF_COMPUTED" ] || fatal "cloudflared sha256 不匹配：期望 $CF_EXPECTED 实际 $CF_COMPUTED"
  say "cloudflared sha256 校验通过：$CF_COMPUTED"
fi

cp "$CF_SRC" "$APP_OUT/bin/cloudflared"
chmod +x "$APP_OUT/bin/cloudflared" "$APP_OUT/$SERVER_BIN_NAME"
say "app/bin/cloudflared 就绪（$(file_size "$APP_OUT/bin/cloudflared") 字节）"

# ---------------------------------------------------------------- 4. 组装 app/
say "== 4. 组装 app/ =="
cp ui/config "$APP_OUT/ui/config"
cp ui/images/icon-64.png "$APP_OUT/ui/images/icon-64.png"
cp ui/images/icon-256.png "$APP_OUT/ui/images/icon-256.png"
cp THIRD_PARTY_NOTICES.md "$APP_OUT/THIRD_PARTY_NOTICES.md"
cp third_party/cloudflared/LICENSE "$APP_OUT/third_party/cloudflared/LICENSE"
cp THIRD_PARTY_NOTICES.md "$APP_OUT/third_party/cloudflared/THIRD_PARTY_NOTICES.md"
mkdir -p "$APP_OUT/third_party/fonts"
cp "$WEB_DIR/public/fonts/LICENSE-MapleMono.txt" "$APP_OUT/third_party/fonts/LICENSE-MapleMono.txt"
cp "$WEB_DIR/public/fonts/NOTICE-MiSans.txt" "$APP_OUT/third_party/fonts/NOTICE-MiSans.txt"
say "app/ = server + bin/cloudflared + ui/{config,images} + third_party/{cloudflared,fonts}"

# ---------------------------------------------------------------- 5. fnpack
say "== 5. fnpack build =="
rm -rf "$BUILD_TMP"
mkdir -p "$BUILD_TMP"
( cd "$BUILD_TMP" && fnpack build -d "$HERE" ) || {
  rm -rf "$BUILD_TMP"
  fatal "fnpack build 失败。若报根目录有非法项，注意 fpk 根只该有 manifest/config/cmd/wizard/ui/app/ICON*.PNG；third_party/、tools/、icon.svg、README.md 是开发期文件，必要时先移出 fpk/ 再打包"
}

PRODUCED="$(find "$BUILD_TMP" -maxdepth 1 -type f -name '*.fpk' | head -1 || true)"
[ -n "$PRODUCED" ] || fatal "fnpack 未产出 ${APP_NAME}-*.fpk"

# Windows fnpack 读不到 POSIX mode：cmd/* 会打成 0666，fnOS 报 10111。
# 重写外层与 app.tgz 内层 mode，并同步 manifest 里的 app.tgz MD5。
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*)
    PY=python
    command -v "$PY" >/dev/null 2>&1 || PY=python3
    command -v "$PY" >/dev/null 2>&1 || {
      rm -rf "$BUILD_TMP"
      fatal "Windows 上需要 python 修正 fpk 权限（或改用 Linux/WSL 原生 fnpack）"
    }
    say "== 5b. 修正 Windows fnpack 的 0666 权限 =="
    "$PY" "$HERE/tools/fix_fpk_modes.py" "$PRODUCED" "$PRODUCED.fixed" || {
      rm -rf "$BUILD_TMP"
      fatal "fpk 权限修正失败"
    }
    mv "$PRODUCED.fixed" "$PRODUCED"
    ;;
esac

mkdir -p "$(dirname "$OUT")"
cp "$PRODUCED" "$OUT"
say "产物：$OUT"

# ---------------------------------------------------------------- 6. 自检
say "== 6. 自检 =="
# Windows 上 GNU tar 会把 E:/... 当成远程主机；--force-local 让 drive letter 按本地路径处理。
TAR="tar"
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*) TAR="tar --force-local" ;;
esac
FPK_LIST="$($TAR -tzf "$OUT" 2>/dev/null || true)"
INNER_LIST="$($TAR -xzOf "$OUT" app.tgz 2>/dev/null | tar -tzf - 2>/dev/null || true)"
MANIFEST_IN="$($TAR -xf "$OUT" -O manifest 2>/dev/null || true)"
MAIN_IN="$($TAR -xf "$OUT" -O cmd/main 2>/dev/null || true)"

FAILED=0
ok() { say "✅ $*"; }
bad() { say "❌ $*"; FAILED=1; }

if [ -n "$FPK_LIST" ]; then ok "fpk 是合法 gzip+tar"; else bad "fpk 不是合法 gzip+tar"; fi
if has "$FPK_LIST" '(^|/)cmd/main$'; then ok "cmd/main 在包内"; else bad "cmd/main 不在包内"; fi
# Windows 上已在 5b 自动修正；这里保留为回归闸门。
CMD_MODE="$($TAR -tzvf "$OUT" cmd/main 2>/dev/null | awk '{print $1}' | head -1 || true)"
case "$CMD_MODE" in
  -rwx*) ok "cmd/main 有执行位（$CMD_MODE）" ;;
  *) bad "cmd/main 没有执行位（$CMD_MODE）—— 检查 tools/fix_fpk_modes.py 是否执行，或改用 Linux/WSL 原生 fnpack" ;;
esac
APP_MD5="$($TAR -xzOf "$OUT" app.tgz 2>/dev/null | md5sum | awk '{print $1}' || true)"
MANIFEST_SUM="$(printf '%s\n' "$MANIFEST_IN" | tr -d '\r' | awk -F'[ =]+' '/^checksum[[:space:]]*=/{print $2; exit}' || true)"
if [ -n "$APP_MD5" ] && [ "$APP_MD5" = "$MANIFEST_SUM" ]; then
  ok "manifest checksum 与 app.tgz 一致（$APP_MD5）"
else
  bad "manifest checksum 失配（manifest=${MANIFEST_SUM:-missing} app.tgz=${APP_MD5:-missing}）"
fi
if has "$FPK_LIST" '(^|/)ui/config$' || has "$FPK_LIST" '(^|/)app\.tgz$'; then
  ok "桌面入口路径（ui/config 或 app.tgz）在位"
else
  bad "桌面入口路径缺失（桌面入口打不开）"
fi

for n in cmd/install_init cmd/install_callback cmd/upgrade_init cmd/upgrade_callback \
         cmd/uninstall_init cmd/uninstall_callback cmd/config_init cmd/config_callback; do
  if has "$FPK_LIST" '(^|/)'"${n}"'$'; then ok "$n 在包内"; else bad "$n 不在包内"; fi
done

if has "$INNER_LIST" '(^|/)'"${SERVER_BIN_NAME}"'$'; then ok "app.tgz 内含 $SERVER_BIN_NAME"; else bad "app.tgz 缺 $SERVER_BIN_NAME"; fi
if has "$INNER_LIST" '(^|/)bin/cloudflared$'; then ok "app.tgz 内含 bin/cloudflared"; else bad "app.tgz 缺 bin/cloudflared"; fi
if has "$INNER_LIST" '(^|/)ui/config$'; then ok "app.tgz 内含 ui/config"; else bad "app.tgz 缺 ui/config"; fi
if has "$INNER_LIST" '(^|/)ui/images/icon-64\.png$'; then ok "app.tgz 内含 ui/images/icon-64.png"; else bad "app.tgz 缺 icon-64.png"; fi
if has "$INNER_LIST" '(^|/)ui/images/icon-256\.png$'; then ok "app.tgz 内含 ui/images/icon-256.png"; else bad "app.tgz 缺 icon-256.png"; fi
if has "$INNER_LIST" '(^|/)third_party/cloudflared/LICENSE$'; then ok "app.tgz 内含 third_party/cloudflared/LICENSE"; else bad "app.tgz 缺 cloudflared LICENSE"; fi
if has "$INNER_LIST" '(^|/)third_party/fonts/LICENSE-MapleMono\.txt$'; then ok "app.tgz 内含 Maple Mono OFL 全文"; else bad "app.tgz 缺 Maple Mono OFL 全文"; fi
if has "$INNER_LIST" '(^|/)third_party/fonts/NOTICE-MiSans\.txt$'; then ok "app.tgz 内含 MiSans 声明"; else bad "app.tgz 缺 MiSans 声明"; fi

if has "$INNER_LIST" '\.bak|~$|\.orig$'; then bad "app.tgz 内含备份文件"; else ok "app.tgz 无备份污染"; fi

if has "$MAIN_IN" 'resolve_share_root'; then ok "cmd/main 用 resolve_share_root（share 软链优先）"; else bad "cmd/main 缺 resolve_share_root"; fi
if has "$MAIN_IN" '/var/apps/'; then ok "cmd/main 优先用 /var/apps/<app>/shares/<share> 软链"; else bad "cmd/main 没用 /var/apps/<app>/shares/<share> 软链"; fi
# 只认「赋值语句里出现 /vol0」为硬编码（注释与 case 里的 /vol0/* 拒绝分支是允许的）
if has "$MAIN_IN" '^[A-Za-z_][A-Za-z0-9_]*=[^#]*/vol0/'; then bad "cmd/main 把 /vol0 路径写进了变量（安装期占位路径）"; else ok "cmd/main 未把 /vol0 写进任何变量"; fi

MSV="$(printf '%s\n' "$MANIFEST_IN" | grep -E '^version[[:space:]]*=' | head -1 | sed 's/.*=[[:space:]]*//' | tr -d '\r' | tr -d '[:space:]' || true)"
if [ "$MSV" = "$VERSION" ]; then ok "包内 manifest version = $VERSION"; else bad "包内 manifest version = ${MSV:-<空>}，应为 $VERSION"; fi
if has "$MANIFEST_IN" '^appname[[:space:]]*=[[:space:]]*cloudflare-tunnel$'; then ok "包内 appname = cloudflare-tunnel"; else bad "包内 appname 不对"; fi
if has "$MANIFEST_IN" 'maintainer_url'; then ok "包内 manifest 带 maintainer_url"; else bad "包内 manifest 缺 maintainer_url"; fi
if has "$MANIFEST_IN" 'distributor_url'; then ok "包内 manifest 带 distributor_url"; else bad "包内 manifest 缺 distributor_url"; fi

if has "$(cat ui/config)" '"protocol": ""'; then ok "ui/config protocol 为空（禁硬编码 http）"; else bad "ui/config protocol 不为空"; fi
if has "$(cat ui/config)" '"port":[[:space:]]*"?18960"?'; then ok "ui/config port = 18960"; else bad "ui/config port 不是 18960"; fi

OUT_SIZE="$(file_size "$OUT")"
OUT_SHA="$(sha256_of "$OUT")"
say ""
say "产物：$OUT"
say "大小：${OUT_SIZE} 字节（$((OUT_SIZE / 1024 / 1024)) MB）"
say "sha256：$OUT_SHA"

rm -rf "$BUILD_TMP"
if [ "$FAILED" -ne 0 ]; then
  say ""
  say "❌ 自检有失败项，产物不可发布。"
  exit 1
fi

say ""
say "完成。包已生成，**未安装** —— 装哪台/装不装先问用户。"

# Agan 的飞牛应用源

这是一个面向 [FnDepot](https://github.com/EWEDLCM/FnDepot) 的第三方 fnOS 应用源，使用 FnDepot V2 源格式。清单文件是仓库根目录的 [`fnpack.json`](./fnpack.json)。

## 添加源

在 FnDepot 的“应用源”页面添加以下任一地址：

- GitHub 仓库地址（推荐）：`https://github.com/latte03/fnos-apps`
- 清单直链：`https://raw.githubusercontent.com/latte03/fnos-apps/main/fnpack.json`

仓库必须保持公开，FnDepot 才能无鉴权读取。GitHub 源会优先读取默认分支根目录的 `fnpack.json`；文件名大小写必须完全一致，客户端不会探测 `fndepot.json`。

## 收录应用

| 应用 | appname | 架构 | 版本 | 上游 |
| --- | --- | --- | --- | --- |
| 磁盘瞭望 DiskLens | `DiskLens` | x86 / arm | 0.3.3 | [JoySec123/DiskLens](https://github.com/JoySec123/DiskLens) |

DiskLens 的安装包直接引用上游 GitHub Release，并用 `sha256` 与 `size` 锁定；本仓库只保存清单、图标和说明。上游发布新版本后，定时工作流会更新清单。

## 源格式速查（V2）

### 顶层结构

```json
{
  "schema_version": "2",
  "source_info": {
    "name": "源名称",
    "author": "源作者",
    "homepage": "https://example.com",
    "description": "源简介"
  },
  "apps": {
    "sample.app": {
      "display_name": "示例应用",
      "desc": "应用简介。",
      "platform": ["all"],
      "categories": ["系统工具"],
      "icon_url": "./apps/sample.app/icon.png",
      "run_as": "package",
      "install_type": "",
      "is_docker": false,
      "releases": {
        "1.0.0": {
          "packages": {
            "all": {
              "download_url": "https://example.com/sample.app-1.0.0.fpk",
              "sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
              "size": 12345678
            }
          }
        }
      }
    }
  }
}
```

`schema_version` 必须是字符串 `"2"`；数字 `2`、`"2.0"`、`"2.1"` 都会被拒绝。`source_info.name` 与 `source_info.author` 必填。

### 应用字段

`apps` 的键就是 FPK manifest 里的 `appname`，大小写敏感，只能使用字母、数字、点、下划线和短横线，且必须以字母或数字开头。

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `display_name` | 是 | 应用显示名称 |
| `desc` | 是 | 应用简介，支持 HTML 文本 |
| `platform` | 是 | 字符串或数组，只允许 `all`、`x86`、`arm` |
| `categories` | 是 | 固定分类，最多两个，第一个为主分类 |
| `icon_url` | 是 | 图标地址，PNG 或 WebP，建议小于 500 KB |
| `run_as` | 是 | 只允许 `package` 或 `root` |
| `install_type` | 是 | 空字符串表示存储空间，`root` 表示系统空间 |
| `is_docker` | 是 | 必须是布尔值 |
| `releases` | 是 | 版本映射，单文件模式必填 |
| `preview_urls` | 否 | 预览图数组，最多读取前 8 张，单张小于 2 MB |
| `readme_url` | 否 | 详情页 README 地址，小于 1 MB |
| `bug_report_url` | 否 | 问题反馈地址 |
| `maintainer` / `maintainer_url` | 否 | 上游维护者与链接 |
| `distributor` / `distributor_url` | 否 | 分发者与链接 |
| `service_port` | 否 | 服务端口，没有 TCP 端口时留空字符串 |
| `details_url` / `details_updated_at` | 否 | 拆分详情文件模式，本仓库未使用 |
| `i18n` | 否 | 名称与简介的本地化 |

固定分类只有九个：`影音娱乐`、`系统工具`、`编程开发`、`AI赋能`、`生活服务`、`智能智控`、`教育学习`、`游戏地带`、`硬件驱动`。不要填写“全部”。

### 版本与安装包

`releases` 的键是可比较的版本号，推荐 SemVer，不能使用 `latest` 或日期文本。每个版本的结构：

```json
{
  "changelog": "本次更新内容。",
  "updated_at": "2026-10-20T05:24:31Z",
  "os_min_version": "1.1.31",
  "os_max_version": "",
  "packages": {
    "x86": { "download_url": "...", "sha256": "...", "size": 12345678 },
    "arm": { "download_url": "...", "sha256": "...", "size": 12345678 }
  }
}
```

- `packages` 的键只能是 `all`、`x86`、`arm`；其他键会让版本校验失败。
- 每个架构分支必须有 `download_url`；`sha256` 为 64 位十六进制，`size` 为字节数。
- x86 设备按 `packages.x86` → `packages.all` 回退；arm 设备按 `packages.arm` → `packages.all` 回退。
- 字段合并顺序为：应用级 → 版本级 → `packages.all` → `packages.[当前架构]`。
- `download_url` 不会从其他架构分支继承。

### 限制与错误处理

- 主清单约 2 MB，拆分详情文件约 512 KB，单源最多 5,000 个应用、单应用最多 100 个版本。
- 源级错误（JSON 语法、顶层结构、`schema_version`、缺少 `source_info`）会中止整次同步，但保留旧缓存。
- 应用级错误只跳过该应用；版本或安装包级错误只跳过该分支，并回退到 `all`。
- 相对 URL 按清单文件所在目录解析，也可以直接写绝对 URL。

完整规则以上游 [FnDepot README](https://github.com/EWEDLCM/FnDepot) 为准。

## 仓库结构

```text
.
├─ fnpack.json                 # FnDepot V2 源清单
├─ apps/
│  └─ DiskLens/
│     ├─ icon.png              # 256 × 256 PNG 图标
│     └─ README.md             # 详情页说明
├─ scripts/
│  ├─ sync-disklens.mjs        # 跟随上游 Release 更新清单
│  └─ validate.mjs             # 按 V2 规则校验
└─ .github/workflows/
   ├─ sync-disklens.yml        # 每日检查上游新版本
   └─ validate.yml             # push / PR 时校验清单
```

## 自动同步

`scripts/sync-disklens.mjs` 会读取 DiskLens 的最新 Release，完成以下工作：

1. 找到 `DiskLens-<version>-amd64.fpk` 与 `DiskLens-<version>-arm64.fpk`。
2. 读取 GitHub 提供的 `sha256` 摘要，缺失时下载文件自行计算。
3. 读取对应 tag 的 `fnos/manifest`，核对 `appname`、`version`、`display_name`、`desc`、`changelog`。
4. 把新版本写入 `fnpack.json`，保留已有版本，按发布时间倒序排列。

```sh
node scripts/sync-disklens.mjs            # 同步最新 Release
node scripts/sync-disklens.mjs --dry-run  # 只打印结果，不写文件
node scripts/sync-disklens.mjs --tag v0.3.3
node scripts/sync-disklens.mjs --force    # 允许覆盖已存在的同版本记录
```

已存在的版本如果摘要不一致，脚本默认报错退出，不会静默替换。GitHub Actions 每天 03:17 UTC 运行一次，也可以在 Actions 页面手动触发；工作流会在提交前下载两个安装包并核对 `sha256` 与 `size`。GitHub 可能在仓库连续 60 天无活动后暂停定时工作流，届时在 Actions 页面手动运行 `Sync DiskLens release` 即可。

## 新增应用

1. 新建 `apps/<appname>/`，放入 `icon.png`（PNG/WebP，小于 500 KB）和可选的 `README.md`。
2. 在 `fnpack.json` 的 `apps` 下增加条目，键必须与 FPK manifest 的 `appname` 完全一致。
3. 至少写入一个版本；每个架构分支填写 `download_url`、`sha256`、`size`。
4. 运行 `node scripts/validate.mjs`，再提交。

## 本地校验

```sh
jq empty fnpack.json
node scripts/validate.mjs
node scripts/validate.mjs --remote   # 额外下载安装包核对摘要，较慢
git diff --check
```

`validate.mjs` 会检查顶层结构、必填字段、分类与架构取值、版本号、安装包分支、本地图标/说明文件的类型与大小，以及 `sha256` 格式和 `size` 是否为非负整数。

## 说明

- 本仓库只提供索引与元数据。应用版权归各自上游所有；DiskLens 以 MIT 许可证发布。
- 安装包引用上游 Release 地址。如果上游删除或替换了资产，对应版本会校验失败；此时请在上游恢复资产，或改为把 FPK 镜像到本仓库。
- 源名称、简介与字段规则可能随 FnDepot 更新；升级客户端后建议重新检查上游 README。

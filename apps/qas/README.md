# QAS 控制台 / 夸克自动转存

> 上游项目：[latte03/quark-auto-save](https://github.com/latte03/quark-auto-save)
>
> 当前版本：`0.3.7`
>
> 许可证：AGPL-3.0

QAS 是自托管的夸克网盘自动转存控制台：Go 服务端 + React 控制台，打包为飞牛 fnOS 应用。一个任务 = 一个分享链接 + 保存路径 + 魔法正则，按设定的运行时刻自动转存，并把结果交给后续插件处理。

## 功能

- **任务追更**：按任务设置多个运行时刻（精确到分钟、按星期过滤），也可设为「仅执行一次」。
- **魔法正则**：稳定 `id` 的命名规则，改名不影响已引用它的任务。
- **插件**：Emby / Plex / 飞牛影视 v1·v2 / Alist 四个变体 / Aria2 / 自动云解压 / SmartStrm，按任务开关。
- **本地下载**：任务级开关、目录与同名覆盖，双增量（网盘新增 + 本地缺失补齐）、大文件分段并发、Range 续传、暂停/继续/取消、实时进度与速度、并发上限、最小剩余空间。
- **通知**：Console / Bark / Telegram / SMTP / Server 酱 / 钉钉 / 飞书等渠道，签到与任务结果可推送。
- **账号**：扫码登录直接写入主账号 Cookie；多账号时第一个账号执行转存、全部账号执行签到。
- **运行记录**：落盘日志，可配置保留条数并自动清理；支持试运行与整轮重跑。
- **MCP**：`qas mcp` 以 stdio 方式提供工具，供 Claude / Cursor 等客户端调用。

## 系统要求

- fnOS `1.2.0` 及以上。
- 仅 x86（amd64）设备：上游目前只发布 `qas_<版本>_x86.fpk`。
- 服务端口：`8710`。
- 需要能访问夸克网盘；不依赖 Docker。

## 安装与使用

1. 安装时设置 WebUI 登录密码（用户名固定为 `admin`，密码需 8–64 位字母、数字与 `._@#%+-`）。
2. 装好后打开应用，在控制台里扫码登录夸克账号并创建任务。
3. 任务里配置分享链接、保存路径与命名规则；需要时开启插件与通知。

## 安全提示

- 配置文件在应用私有目录里，包含夸克 Cookie、WebUI 密码与推送渠道密钥，不要外泄。
- WebUI 使用 HttpOnly Cookie 会话；接口与 SSE 在会话失效时统一返回 401。
- 不要把 `8710` 端口直接暴露到公网；需要远程访问时用反向代理并加 TLS。

## 上游资料

- [项目主页与源码](https://github.com/latte03/quark-auto-save)
- [0.3.7 发布说明](https://github.com/latte03/quark-auto-save/releases/tag/v0.3.7)
- [产品简报](https://github.com/latte03/quark-auto-save/blob/v0.3.7/docs/archive/PRODUCT.md)
- [问题反馈](https://github.com/latte03/quark-auto-save/issues)

本页由源维护者整理；应用图标取自上游安装包，遵循 AGPL-3.0 许可。

# NAS 管家

> 上游项目：[kaf931/fnos-manager](https://github.com/kaf931/fnos-manager)
>
> 当前版本：`0.3.6`
>
> 许可证：上游未提供开源许可证，默认保留所有权利

NAS 管家把常用的 NAS 运维工具收进一个入口：桌面图标、计划任务、SSH 终端、存储管理与网络唤醒。既可以统一使用，也可以把单个功能单独加到飞牛桌面。界面同时适配 PC 与移动端。

## 主要功能

- **桌面图标**：给 Docker 容器、自定义链接和管家菜单创建桌面入口。
- **计划任务**：定时启停程序、控制容器、运行脚本，并可联动网络设备。
- **SSH 终端**：多标签连接、文件管理、常用命令库、资源监控，以及结合当前主机命令输出的 AI 助手（支持 OpenAI、Claude、Ollama、Gemini、GLM、MiniMax、DeepSeek、MiMo 等协议）与 Skills 管理。
- **存储管理**：空间分析、重复文件检查、批量改名，以及聊天附件的记录与清理。
- **网络唤醒**：扫描局域网、批量添加设备、唤醒与 SSH 电源控制。

## 系统要求

- fnOS `1.2.0800` 及以上。
- CPU 架构：`x86`（`NAS.-<版本>-x86_64.fpk`）或 `arm`（`NAS.-<版本>-arm64.fpk`）。
- 依赖 Node.js v22：FPK manifest 已声明 `install_dep_apps = nodejs_v22`，安装时由 fnOS 处理。
- 没有固定 TCP 端口：应用通过 `/app/deskicons/` 页面和内部 Socket 提供服务，`service_port` 为空。

## 权限说明

上游在 `config/privilege` 中声明以 `root` 运行，用于管理容器、计划任务、SSH 连接和存储文件等系统级操作。安装前请确认你接受这一点，并只在可信的局域网环境中使用。

## 安装

1. 在 FnDepot 或 fnOS 应用中心安装本应用。
2. 按飞牛提示选择安装位置，等待依赖与应用文件就绪。
3. 打开「NAS 管家」，可按需把桌面图标、SSH 终端等功能单独添加到飞牛桌面。

## 安全提示

SSH 连接、文件管理、AI 助手都能执行命令或读写文件，本应用没有面向公网的鉴权设计。请仅在内网使用，不要把管理入口暴露到公网；AI 的建议命令首次使用建议保持逐条人工审核。

## 图标与截图

应用图标取自上游安装包内的 `ICON_256.PNG`（256 × 256），与安装后在 NAS 上的 `/vol1/@appcenter/deskicons/ui/images/icon_256.png` 完全一致。上游仓库未提供 LICENSE 文件，如上游后续调整图标或授权条款，本源会同步更新。详情页预览图直接链接上游仓库 `docs/images/readme/` 下的截图。

## 上游资料

- [项目主页与源码](https://github.com/kaf931/fnos-manager)
- [0.3.6 发布说明](https://github.com/kaf931/fnos-manager/releases/tag/0.3.6)
- [使用文档](https://github.com/kaf931/fnos-manager/tree/0.3.6/docs)
- [问题反馈](https://github.com/kaf931/fnos-manager/issues)

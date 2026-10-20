# DiskLens / 磁盘瞭望

> 上游项目：[JoySec123/DiskLens](https://github.com/JoySec123/DiskLens)  
> 当前版本：`0.3.3`（开源测试版）  
> 许可证：MIT（第三方组件遵循各自许可证）

DiskLens 是面向 fnOS 管理员的本地只读磁盘空间分析器。它把目录扫描、历史比较、空间账单、增长提醒和重复文件确认放在同一个界面中，帮助定位“空间被什么占用”和“哪个目录持续增长”。

应用不会删除、移动、重命名或自动清理文件，也不提供对应 API。

## 功能

- 多个授权目录统一扫描，自动合并重复及父子重叠范围。
- 概览、目录下钻、文件类型、大文件、完整索引筛选与 CSV 导出。
- 保留最近 20 次完整扫描、同范围历史比较，以及 365 天 / 2,000 条空间账单。
- 只读重复文件检测：完整 SHA-256 后再逐字节确认，跳过链接、特殊文件和变化中的文件。
- 每日定时巡检、扫描速率和最长时间设置，以及增长、容量和覆盖提醒。
- 查询、导出、比较和重复结果都会重新核对当前 fnOS 目录授权；任一扫描根撤权后隐藏整个快照。

## 安全与数据边界

普通扫描只读取文件元数据。只有用户主动启动重复检测时才读取候选文件内容。结果保存在 NAS 本机。

文件大小为逻辑大小，不等于真实磁盘占用；快照、稀疏文件、压缩、reflink 和硬链接都会造成差异。重复结果只表示内容相同，不是自动删除建议。

## 安装与升级

在 FnDepot 中选择对应架构的安装包即可。升级旧版时直接安装新包，不要先卸载；升级后关闭旧窗口并从桌面重新打开，确认左下角显示 `v0.3.3`。

- Intel / AMD 设备使用 `x86` 包（上游文件名 `DiskLens-0.3.3-amd64.fpk`）。
- ARM 设备使用 `arm` 包（上游文件名 `DiskLens-0.3.3-arm64.fpk`）。

## 上游资料

- [项目主页与源码](https://github.com/JoySec123/DiskLens)
- [0.3.3 发布说明与校验值](https://github.com/JoySec123/DiskLens/blob/v0.3.3/doc/RELEASE-0.3.3.md)
- [问题反馈](https://github.com/JoySec123/DiskLens/issues)
- [安全策略](https://github.com/JoySec123/DiskLens/blob/v0.3.3/SECURITY.md)
- [第三方组件许可](https://github.com/JoySec123/DiskLens/blob/v0.3.3/THIRD_PARTY_NOTICES.md)

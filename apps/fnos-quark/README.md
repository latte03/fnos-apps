# 飞牛夸克下载器

> 上游项目：[ElyDemiurge/fnOS-quark](https://github.com/ElyDemiurge/fnOS-quark)
>
> 当前版本：`0.7.12`
>
> 许可证：专有软件，仅允许个人使用作者发布的未修改 FPK

飞牛夸克下载器是面向 fnOS 的第三方夸克网盘下载工具。它提供网页界面，可浏览网盘文件、选择下载目录，并把文件直接保存到 NAS。应用与夸克网盘及其运营方没有隶属、合作或授权关系。

## 主要功能

- 浏览、排序网盘文件和文件夹，下载时保留目录结构。
- 扫码或 Cookie 登录夸克账号，支持暂停、恢复、重试、优先下载和断点续传。
- 多目录增量同步、下载并发数、单文件连接数、硬盘预留空间和总限速设置。
- 任务进度、剩余时间、速度曲线、日志筛选和诊断导出。
- 桌面、平板和手机自适应界面。

## 安装与首次使用

1. 在 FnDepot 或 fnOS 应用中心安装对应架构的 FPK。
2. 从 `应用文件/fnos-quark/config/access_token_settings.json` 复制 `token`。
3. 打开应用，同意使用协议并输入访问令牌。
4. 在“账号管理”中使用夸克网盘 App 扫码登录，或选择 Cookie 登录。

升级时保留 `应用文件/fnos-quark` 即可保留配置和任务。卸载不会删除该目录。

## 数据目录

```text
应用文件/fnos-quark/
├── config/       访问令牌、登录信息和下载/同步设置
├── data/         任务、通知及运行状态
├── downloads/    默认下载文件
└── logs/         日志及崩溃报告
```

## 安全提示

访问令牌、Cookie、配置和诊断包都属于敏感信息。不要把应用端口直接暴露到公网；仅在可信内网关闭令牌验证。只下载自己有权使用的文件，并遵守夸克网盘的服务条款。

## 上游资料

- [项目主页](https://github.com/ElyDemiurge/fnOS-quark)
- [最新 Release](https://github.com/ElyDemiurge/fnOS-quark/releases)
- [问题反馈](https://github.com/ElyDemiurge/fnOS-quark/issues)
- [最终用户许可协议](https://github.com/ElyDemiurge/fnOS-quark/blob/v0.7.12/LICENSE)

本页由源维护者整理，安装包和官方素材的版权归上游作者 Cyberangel 所有。应用图标为本源使用的通用云下载占位图，不是上游官方图标。

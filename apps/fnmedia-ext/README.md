# 飞牛影视在线扩展

> 上游项目：[yujian0815/fnmedia-ext](https://github.com/yujian0815/fnmedia-ext)
>
> 当前版本：`0.2.0`
>
> 许可证：MIT

飞牛影视在线扩展是飞牛影视（trim.media）的第三方在线资源扩展。它通过 Unix Socket 可逆接管官方影视入口，在原生搜索中聚合苹果 CMS 资源站结果，并注入豆瓣热门榜单与分类媒体库。

## 主要功能

- 原生搜索聚合在线结果，支持多线路切换和噪音过滤。
- 豆瓣热门榜单四库，以及电影、剧集、动漫、综艺分类四库，支持无限滚动、点开即播。
- m3u8 / mp4 代理播放、播放记录和智能续播。
- 应用内管理页，可配置资源站、刷新榜单和查看播放记录。
- 停用或卸载扩展后自动把 `/var/run/trim_media.sock` 还原给官方影视服务。

## 系统要求

- fnOS `1.2.0` 及以上。
- 官方「影视」应用（`trim.media`，实测 `0.9.8-1`）必须保持安装并启用。
- CPU 架构：`all`（纯 Python，安装时创建独立 venv）。

## 安装

1. 在 FnDepot 或 fnOS 应用中心安装本扩展。
2. 选择应用安装所在的数据卷，例如 `/vol1`、`/vol2`。
3. 保持官方「影视」应用为运行状态，启用「飞牛影视在线扩展」。
4. 打开官方「影视」，搜索中文片名即可看到带 `【在线】` 标记的结果。
5. 管理入口在应用内 iframe：`/app/fnmedia-ext/`。

本扩展的安装包不内置任何资源站点、订阅地址、爬虫脚本或账号凭证，在线内容需要用户在管理页自行添加并核实来源与授权。

## 精简版与完整版

上游 Release 同时提供两个包：

- `fnmedia-ext-0.2.0-full.fpk`：完整版，约 7.4 MB，额外包含 TVBox type=3 爬虫运行时；本源默认使用这个包。
- `fnmedia-ext-0.2.0.fpk`：公开精简版，约 0.8 MB；需要更小的安装包时从 [上游 Release](https://github.com/yujian0815/fnmedia-ext/releases/tag/v0.2.0) 下载。

两个版本都不内置第三方源或凭证，完整版的执行组件需要用户自行准备。

## 上游资料

- [项目主页与源码](https://github.com/yujian0815/fnmedia-ext)
- [0.2.0 发布说明](https://github.com/yujian0815/fnmedia-ext/releases/tag/v0.2.0)
- [更新日志](https://github.com/yujian0815/fnmedia-ext/blob/v0.2.0/CHANGELOG.md)
- [问题反馈](https://github.com/yujian0815/fnmedia-ext/issues)
- [MIT 许可证](https://github.com/yujian0815/fnmedia-ext/blob/v0.2.0/LICENSE)

本页由源维护者整理；应用图标来自上游 MIT 仓库。

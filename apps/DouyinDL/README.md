# 抖音下载

> 上游项目：[Zyc009234/fnos-douyin-dl](https://github.com/Zyc009234/fnos-douyin-dl)
>
> 当前版本：`1.0.5`
>
> 许可证：上游未提供开源许可证，默认保留所有权利

抖音下载是飞牛 fnOS 的原生应用，用于把抖音网页版正在播放的视频（无水印）保存到 NAS。它通过浏览器书签接收视频地址，也支持在手机抖音里分享链接给快捷指令后下载。

## 主要功能

- 浏览器书签：在抖音网页版播放视频，点一下书签即可保存到 NAS。
- 分享链接下载：管理页粘贴分享链接，或通过 iPhone 快捷指令「存到NAS」发送。
- 自定义保存位置，支持飞牛的文件夹选择和授权流程。
- 在应用页面播放、下载或删除已保存的视频。
- 默认保存到共享目录 `DouyinDL/videos`。

## 系统要求

- fnOS `1.0.0` 及以上。
- Node.js v22：FPK manifest 已声明 `install_dep_apps = nodejs_v22`，安装时由 fnOS 处理。
- 分享链接下载需要在 NAS 上安装 Chromium：`sudo apt install chromium`。
- 服务端口：`8866`。

## 安装与使用

1. 在 FnDepot 或 fnOS 应用中心安装本应用。
2. 打开「抖音下载」，把「存到 NAS」按钮拖到浏览器书签栏。
3. 在 `www.douyin.com` 播放视频，点书签；首次使用时允许抖音页面打开弹窗。
4. 默认保存到 `DouyinDL/videos`，也可以在页面底部「保存位置」选择其他文件夹。

手机分享链接下载需要手机和 NAS 在同一局域网，具体快捷指令配置见上游 README。

## 安全提示

服务没有登录鉴权，只应在局域网内使用，不要把 `8866` 端口映射到公网。请只下载自己有权保存的内容，尊重创作者版权。

## 上游资料

- [项目主页与源码](https://github.com/Zyc009234/fnos-douyin-dl)
- [1.0.5 发布说明](https://github.com/Zyc009234/fnos-douyin-dl/releases/tag/v1.0.5)
- [问题反馈](https://github.com/Zyc009234/fnos-douyin-dl/issues)

本页由源维护者整理。上游未提供 LICENSE，本源不复制其官方图标或素材；应用图标为本源使用的通用视频下载占位图。

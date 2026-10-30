# RustFS / 飞牛 fnOS 应用包

> 上游项目：[rustfs/rustfs](https://github.com/rustfs/rustfs)（本应用来自社区打包仓库：[a10463981/rustfs-fnos](https://github.com/a10463981/rustfs-fnos)）
>
> 当前版本：`1.0.4-2`
>
> 许可证：Apache-2.0

RustFS 是兼容 Amazon S3 的对象存储。本应用把 RustFS 做成飞牛应用中心里的一个应用：容器由应用中心托管，装好后桌面上多一个图标，点开就在 fnOS 窗口内打开控制台，不需要手写 `docker-compose.yml`，也不需要登录 SSH。

## 主要用途

- 备份目标：支持「S3 兼容存储」的备份软件可以把备份写到自家 NAS，数据不出门。
- 媒体库 / 图床后端：给媒体服务器、相册、图床类应用提供对象存储。
- 给其它自托管应用提供本地 S3 端点，附件、导出包、日志归档集中存放。

对象存储本身不是备份方案：里面的数据如果有价值，仍然需要另做备份。

## 系统要求

- Docker：必需。设备上要有可用的 Docker 服务，容器由应用中心托管。
- 存储空间：至少一个已创建的存储空间；建议给 RustFS 单独一块盘或单独一个卷，不要放在系统分区。
- 网络：安装与升级时需要能访问容器镜像仓库，用来拉取 `rustfs/rustfs` 与网关用的 `nginx` 镜像。
- 架构：`all`，包里只有配置与脚本，x86_64 与 aarch64 共用同一个安装包。ARM 上游未做真机验证，装完请按上游清单逐条自查。
- 端口：默认 `9200`（S3 API）与 `9201`（Web 控制台），安装时可以改。

## 安装与使用

1. 在 FnDepot 或 fnOS 应用中心安装。
2. 向导里设置管理员账号密码、S3 API 端口、控制台端口，以及允许访问控制台的来源（例如 `http://<你的NAS>:5666`）。
3. 装完点桌面图标，在 fnOS 窗口内打开控制台；也可以直接访问 `http://<你的NAS>:9201`。

安装向导不再询问数据目录，首次一律使用默认位置。想改到自己的存储空间，按两步走：先到「应用设置 → 访问权限」把该目录授权给应用，再到「运行设置 → 数据存放位置」选「改到下面填写的目录」并填同一路径。顺序反了会报「数据目录不可写」。

## 升级须知

- 从 `1.0.4-1` 或 `1.0.4` 升级：先卸载（数据选「保留」）再安装本版，凭据需要重设；自定义数据目录里的数据不受影响。
- 升级会重建容器，S3 API 与控制台会中断几秒到几十秒。
- 安装或升级停在「应用初始化」阶段时，fnOS 的失败回滚会删除应用的数据目录与配置目录。升级前先备份 `/vol*/@appdata/rustfs-fnos/data`，见上游 [升级说明](https://github.com/a10463981/rustfs-fnos/blob/v1.0.4-2/docs/upgrade-safety.md)。

## 上游资料

- [打包仓库](https://github.com/a10463981/rustfs-fnos)
- [1.0.4-2 发布说明](https://github.com/a10463981/rustfs-fnos/releases/tag/v1.0.4-2)
- [安装与 ARM 自测清单](https://github.com/a10463981/rustfs-fnos/blob/v1.0.4-2/docs/install-fnos.md)
- [常见问题](https://github.com/a10463981/rustfs-fnos/blob/v1.0.4-2/docs/faq.md)
- [问题反馈](https://github.com/a10463981/rustfs-fnos/issues)

本页由源维护者整理；应用图标取自上游安装包，遵循 Apache-2.0 许可。

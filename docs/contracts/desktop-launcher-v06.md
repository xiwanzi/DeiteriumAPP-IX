# 桌面启动器 0.6 / Web 2.0.19

启动器独立仓库为 `Deuterium-IX/DeLauncher`。APP 仓库保留社区网页和管理/API；Windows 程序、Qt WebEngine 容器、更新器与签名构建脚本在启动器仓库维护。

## 社区内嵌

Windows 启动器直接加载同源 `/information?launcher=1`，用户代理包含 `DLauncher/版本`。网页据此使用适合启动器视口的布局；该标识仅控制样式，不授予权限。普通浏览器保留原布局。

内嵌模式默认深色，保持用户已选主题；网页填满右侧区域，使用原有账号、会话和业务 API。Minecraft 正版身份不会代替 Deuterium ID 身份。

## 程序发布 API

| 路由 | 作用 |
| --- | --- |
| `GET /api/v1/launcher/application` | 公开签名版本，`data.release` 为信封或 `null`；不缓存 |
| `GET /api/v1/admin/desktop-launcher/application` | 当前版本、发布修订和上传状态 |
| `POST .../application/upload` | 验证签名清单，返回当前 OSS 的预签名 ZIP 上传地址 |
| `POST .../application/publish` | 校验 OSS 文件后发布；输入 `release`、`expectedRevision`、`clientRequestId` |

管理端沿用 `platform.admin`、Web Cookie、Origin、CSRF。发布事务重新验证管理员资格。重复请求复用结果，过期修订返回冲突，版本编码必须高于已发布版本。上传或校验失败不改变当前公开版本。

签名信封为 `{payload,signature}`，均为标准 Base64。payload 的原始字节由发布者使用 Ed25519 签名，服务与启动器内置相同公钥。私钥不在服务器、浏览器或 Git 中。字段协议见启动器仓库 `docs/SELF-UPDATE.md`。

只接受 `DLauncher/windows-x64`，ZIP 最大 1 GiB，包地址必须对应当前配置的 OSS 前缀 `launcher/application/windows-x64/<sha256>.zip`。服务不请求清单中的任意外部地址；按校验出的对象键从配置好的 S3 客户端读取文件并验证大小和摘要。

SQL `031_desktop_application.sql` 只新增独立单行发布表与初始记录。游戏资源草稿、McPatch 索引、整包版本、账号和交易数据不受此次程序发布影响。

## 发布与回退

部署前备份数据库和当前 Go/Web 路径，运行新增迁移再切换程序。迁移不修改旧表，回退 Go/Web 时保留新表和迁移记录。Windows 0.6 开始依赖新版本接口，服务回退需考虑已交付客户端的接口兼容性。

0.5.x 玩家需要手动覆盖一次启动器引导包，之后才可使用自更新。本次不更新 Android App、不重置白名单、不重置 McPatch 基线、不创建完整游戏 ZIP。

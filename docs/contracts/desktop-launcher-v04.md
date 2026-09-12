# 桌面启动器服务契约（0.5.0）

适用：2026-09-12 的 Go / Web 2.0.14 启动器管理功能。Android App 沿用 2.0.12。桌面端固定 Minecraft 1.21.1、NeoForge 21.1.248；账户认证继续由桌面端直接使用微软正版链路。

## 公共读取

| 路径 | 返回 |
| --- | --- |
| `GET /api/v1/launcher/bootstrap` | 已删除，404。客户端不再请求安装配方 |
| `GET /api/v1/launcher/content` | 标准 envelope：`data.version` 和 `data.content`；只读已发布内容，未发布返回 404 |
| `GET /api/v1/launcher/updates/index.json` | McPatch 原始 JSON 数组，保持客户端协议兼容；仅返回后台已核对的同步索引 |
| `GET /api/v1/launcher/updates/{file}` | 仅允许已发布索引中的文件，307 跳转到 OSS，保留 Range 下载；未知文件返回 404 |

`bootstrap` 仅作为历史存储兼容键保留，内容为版本标识；写入时移除 java、installer、enabled 等旧下载字段。完整游戏由管理员发布，客户端靠随包 `bundle.json` 记录出厂资源版本，仅请求后续 McPatch 索引。

## 管理写入

所有管理路由都要求现有 `platform.admin` 权限。Web 会话写入继续校验 Origin 与 CSRF。

| 路径 | 输入与行为 |
| --- | --- |
| `GET /api/v1/admin/desktop-launcher` | 返回草稿/发布版本、最近同步任务、原生 McPatch URL、连接状态及已打包版本 |
| `PUT /api/v1/admin/desktop-launcher` | `clientRequestId`、`expectedVersion`、`document:{bootstrap,content}`、`publish`；上限 1 MiB，完整内容另有限制。幂等、乐观版本控制；发布前要求初始基线已同步 |
| `POST .../sync` | `clientRequestId`，可选当前浏览器 `nativeToken`。原生令牌不持久化到业务请求。返回持久化任务，异步执行原生上传；同一时刻只运行一个任务 |
| `POST .../access` | 返回原生管理 URL、用户名、密码，仅平台管理员使用；秘密来自服务器环境，不进入公开配置 |
| `POST .../media` | `contentType`、`size`、`sha256`，签发 15 分钟 OSS PUT 地址；仅 PNG/JPEG/WebP/MP4，最多 128 MiB |
| `POST .../media/complete` | `key`；服务端读取并核对 SHA-256 后返回公开 URL、大小和摘要 |

同步状态为 `RUNNING`、`SUCCEEDED`、`FAILED`。APP 调用原生 McPatch `/api/task/upload` 并等待，读取原生 `/public/index.json`，核对 OSS 索引内容和 TAR 大小后才原子发布索引。失败保留上一份索引。服务重启后核对遗留任务，不能确认完整时标记可重试失败。

内容 schemaVersion=2，保留 arknights、deuterium_ix、popucom 三个配置项，通过 visible=false 暂时隐藏两个非服务器入口。静态图、视频、宣传图和公告正文配图上传后加入 `mediaAssets` 的 size/SHA-256 元数据；客户端缓存校验完成后切换，失败保留原缓存。

新增数据库迁移 `027_desktop_launcher.sql` 只增加设置和同步记录表，不修改原业务记录。运行服务沿用原有数据库权限；迁移需数据库管理员权限。

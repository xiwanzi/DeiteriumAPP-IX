# Deuterium IX

网站、原生 Android App、Go 后端与 Deuterium Core 的源码和运维入口。当前发布：2026-09-09 App 2.0.5（20500）/Web 2.0.4、配套 Go、Core 1.0.1 与 XConomy .2 已上线；Mail/Sync 保持既有版本。本轮源码已获用户授权公开推送。

| 组件 | 源码位置 | 当前发布基准 |
| --- | --- | --- |
| Android App | [android-app/ui-lab](android-app/ui-lab/README.md) | 2.0.5（20500），源码 `bcfe87e398e75` |
| 玩家、商家和平台管理网站 | [web-app](web-app/README.md) | 2.0.4，业务源码 `812f1d1a5056`，静态版本配置 `5581566` |
| Go 后端 | [backend-next](backend-next/README.md) | `812f1d1a5056`，统一账本、SMTP 提醒与完整交易审计 |
| 游戏 Core | [deuterium-core](deuterium-core/README.md) | 1.0.1，四服已换包，三个常驻服已恢复 |
| XConomy 扩展 | [adapters/xconomy](adapters/xconomy/README.md) | 2.26.3-deuterium.2，包含原生/App 已提交账本查询 |
| 独立 Mail | [Deuterium-IX/Deuterium-Mail](https://github.com/Deuterium-IX/Deuterium-Mail) | 插件 0.6.1 / API 2，Bridge、UI 0.6.0 |
| 独立 Sync | [Deuterium-IX/YouerModSync-Patches](https://github.com/Deuterium-IX/YouerModSync-Patches) | 0.7-deuterium.3，Core 保存屏障 |

从 [项目状态](docs/project-status.md)、[架构与业务边界](CONTEXT.md)、[构建指南](docs/build.md) 和 [部署手册](docs/deployment/README.md) 开始。固定外部版本、来源摘要和本次验证见 [源码归档说明](docs/repository-handoff.md) 与 [组件锁定清单](components.lock.json)。完整契约、ADR、历史 QA 见 [文档导航](docs/README.md)。

现有服务入口为 [Deuterium IX](https://47.103.99.34)，App 下载及摘要见 [2.0.5 发布记录](docs/deployment/deuterium-2.0.5-live.md)。Android 包名仍为 `com.deuterium.app.uilab`；目录名 ui-lab 不代表当前仍是演示版。SMTP 配置页已上线，管理员填写实际邮箱参数后启用提醒。

本仓库保留源码、素材、示例配置、迁移和测试。服务器凭据、签名私钥、玩家数据、数据库备份与 APK/JAR 不进入 Git。已有测试签名须由维护者受限保管，新机器自行生成的 debug key 不能覆盖当前已安装 App。

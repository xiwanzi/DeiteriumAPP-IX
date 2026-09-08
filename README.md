# Deuterium IX

网站、原生 Android App、Go 后端与 Deuterium Core 的私有源码和运维入口。整理基准：2026-09-08 已发布版本；本次仓库整理不更新线上服务、不迁移业务数据。

| 组件 | 源码位置 | 当前发布基准 |
| --- | --- | --- |
| Android App | [android-app/ui-lab](android-app/ui-lab/README.md) | 2.0.2 (20200)，原提交 `4299ce37d0bf` |
| 玩家、商家和平台管理网站 | [web-app](web-app/README.md) | 2.0.0，发布源码 `4de1ee7`，QA `2455abb` |
| Go 后端 | [backend-next](backend-next/README.md) | 图片生命周期 `1b5d9d1c63f8`；集成记录 `f07f69e` |
| 游戏 Core | [deuterium-core](deuterium-core/README.md) | 1.0.0，原开发基准 `5ae6706`，发布集成源码见清单 |
| XConomy 扩展 | [adapters/xconomy](adapters/xconomy/README.md) | 2.26.3-deuterium.1，指定原版 JAR 的补丁源码 |
| 独立 Mail | [Deuterium-IX/Deuterium-Mail](https://github.com/Deuterium-IX/Deuterium-Mail) | 插件 0.6.1 / API 2，Bridge、UI 0.6.0 |
| 独立 Sync | [Deuterium-IX/YouerModSync-Patches](https://github.com/Deuterium-IX/YouerModSync-Patches) | 0.7-deuterium.3，Core 保存屏障 |

从 [项目状态](docs/project-status.md)、[架构与业务边界](CONTEXT.md)、[构建指南](docs/build.md) 和 [部署手册](docs/deployment/README.md) 开始。固定外部版本、来源摘要和本次验证见 [源码归档说明](docs/repository-handoff.md) 与 [组件锁定清单](components.lock.json)。完整契约、ADR、历史 QA 见 [文档导航](docs/README.md)。

现有服务入口为 [Deuterium IX](https://47.103.99.34)，App 下载及摘要见 [2.0.2 发布记录](docs/deployment/deuterium-2.0.2-live.md)。Android 包名仍为 `com.deuterium.app.uilab`；目录名 ui-lab 不代表当前仍是演示版。

本仓库保留源码、素材、示例配置、迁移和测试。服务器凭据、签名私钥、玩家数据、数据库备份与 APK/JAR 不进入 Git。已有测试签名须由维护者受限保管，新机器自行生成的 debug key 不能覆盖当前已安装 App。

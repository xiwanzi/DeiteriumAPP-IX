# 模型管理 Web 2.0.35 上线

2026-10-10（UTC+8），现有模型管理入口已切换为 Cloud 模型库、版本、玩家外观、Citizens NPC、节点及审计页面。保留 Web 2.0.34 聊天材质改动；Android 仍为已发布的 2.0.25。

Web/Go 构建源码 `4fec6907cc58d5fcc3f6fddc119a007d8c20c817`，对应 [PR #93](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/93)。Go 位于 `/opt/deuterium/releases/spm-cloud-4fec6907cc58`，Web 位于 `/var/www/deuterium/releases/web-2.0.35-4fec6907cc58/dist`；备份 `/var/backups/deuterium/spm-cloud-4fec6907cc58`。

83 项 Web 测试、生产构建、Go YSM 定向测试与 Linux 构建通过。生产 Go → Velocity → Cloud 鉴权链路读取全部 283 个模型、三个在线节点和 3 个 NPC；匿名管理请求拒绝，管理令牌不返回浏览器。HTTPS IP 与 chat 域名的版本、首页、JS/CSS 均匹配；临时验收会话已撤销。

无业务数据库迁移，私有配置不变。备份及核对保留 58 个账号、49 个交易资源、9,328 个目录记录、12 个工单、2 条注销记录、61 条私聊记录的既有 ID 集合。回退只恢复原 Go/Web 链接，不覆盖新增业务数据。

配套客户端 0.0.32-dev、Cloud、CitizensYSM 3 与代理已部署。生产跨服联验发现旧状态交接残留，代理已补修并通过回归测试、启动及后台连通检查。用户接手补丁后的跨服、重登及 NPC 游戏体验验收，尚未宣称这些生产操作全部通过。游戏部署与回退记录见 [模型仓库](https://github.com/Deuterium-IX/DeuteriumYSM/blob/xiwanzi/spm-cloud-migration/docs/SPARKLE_DEPLOYMENT_20261010.md)。

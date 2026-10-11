# 游戏聊天 0.0.47-dev：离线头像、快捷键与中文输入

2026-10-11 **23:27（UTC+8）**完成公网核验。客户端 **0.0.47-dev**（聊天模组 **0.1.3**）及配套 Go 已上线。

- 头像与快捷键源码 `50f918e`，[PR #101](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/101)，合并 `2ee1957b054dc189b22d703d07b09245ab3a45a8`。
- IMBlocker 豁免源码 `836c5a3`，[PR #102](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/102)，合并 `bfd72cdb019023bbd51f847d36a40a73cc7edfaa`。
- 0.0.46 为本轮已打包中间步骤，收到中文输入要求后与 0.0.47 一起同步，玩家可以从 0.0.45 直接升级最终版本。

## 修复

玩家离线不再依赖在线玩家列表：当前客户端保留已见皮肤，服务器按权威 UUID 持久保存 8×8 PNG 合成头像，重进游戏可异步取回。正式接口验证 xiwanzi 的头像为 **235 字节**，数据库与重复响应哈希一致。图片上游不可用时保留旧头像；日刷新有最多一天换肤缓存延迟，当前会话看到的在线皮肤仍优先。身份范围、来源限制和注销清理见[头像契约](../contracts/game-chat-avatars.md)。

`keybindings-v2` 一次迁移明确将 `key.tacz.open_config.desc` 设置为未绑定，T 快捷聊天、Alt+T 表情轮盘和 Enter 完整页面保持。实际 TAC 1.1.8-hotfix-r6 的配置入口经映射匹配触发，解绑会使该入口不再匹配 T；之后玩家仍可自行修改键位。

使用可选 Mixin 为 IMBlocker 的 `isScreenInWhitelist` 增加仅针对 `ChatScreenFull` 的中文输入豁免；不改全局 IME 设置或用户白名单，不豁免其他 LDLib2 页面。T 快捷聊天沿用原版输入框的既有兼容。验收实际加载 IMBlocker 5.5.4，确认完整聊天被豁免、其他对象未被放行；本轮未单独人工操作微软拼音候选窗，不将自动化白名单检查当作真实输入法候选交互证明。

## 验证和产物

- Go 全量单元测试通过；头像持久化、离线查询、旧缓存读取、上游失败保留头像及游戏会话隔离集成测试通过。头像合成层与纹理 URL 主机限制有单元检查。
- 客户端构建通过；实际游戏 + IMBlocker 5.5.4 场景 `9596_eccaf1b`：108 步、6 项检查通过，含离线头像恢复与豁免范围。
- 实际 DLauncher **0.0.45 → 0.0.47** 两步更新链通过，0.1.1 / 0.1.2 旧 JAR 清理、0.1.3 新 JAR 校验、个人设置和存档保留、二次检查无操作。
- OSS 同步成功；0.0.46 与 0.0.47 两个公网 TAR 完整 SHA-256 核对，最终索引与 Range 元数据核对，原有客户端条目和其他模组保留。

| 产物 | 字节 | SHA-256 |
| --- | ---: | --- |
| 聊天 0.1.3 JAR | 18,892,678 | `43b53b5c7aa0fe5e322ad9d766ab81242e741a71d4f1571efe2ef75300edddf0` |
| 0.0.47-dev.tar | 18,896,896 | `82dc50d6fb95c9e6d870f8096977ff697b33da738d9ea459977ac27cecfd1b92` |
| Go Linux amd64 | — | `7c24bd37127c63523aada1499c2577f143730f928dd431d6f0491f7114c38f43` |

## 保留项和回退

新增数据库迁移 `039_game_chat_avatars.sql`，旧迁移校验和与账号、交易、目录、附件、工单 ID 集合保留。仍为 **61 账号、47 订单、5 委托、3,085 在售商品、12 工单、2 注销记录**。Core **1.0.4**、App **2.0.25**、Web **2.0.35** 不变；三常驻节点在线且存储健康、MEK 停服，本轮未重启游戏服。

Go `/opt/deuterium/releases/game-chat-avatar-50f918e`，备份 `/var/backups/deuterium/game-chat-0.0.46-50f918e`，含数据库一致性快照和旧程序/配置；回退仅切旧 Go 链接，保留新表和新增业务数据。客户端备份 `/var/backups/deuterium/game-chat-client-0.0.46-50f918e`、`/var/backups/deuterium/game-chat-client-0.0.47-836c5a3`；客户端下载后需前向修正版回退。

维护 `.worktrees/game-chat-ldlib` / `xiwanzi/game-chat-ldlib`。发布与验证临时会话均撤销，无聊天、邮件或测试订单投递。

[Go 部署](artifacts/game-chat-0047/go-deployment.json) · [头像持久化](artifacts/game-chat-0047/avatar-api-verification.json) · [游戏 UI](artifacts/game-chat-0047/ui-verification.json) · [跨版本升级](artifacts/game-chat-0047/client-updater-verification.json) · [公网验证](artifacts/game-chat-0047/public-verification.json) · [OSS 同步](artifacts/game-chat-0047/sync-result.json)

# 游戏原生聊天 0.0.44-dev 上线

2026-10-11 **22:33（UTC+8）**完成公网核验。启动器客户端内容 **0.0.44-dev**（聊天模组 0.1.0）、**Core 1.0.4** 与配套 Go 已发布。App **2.0.25（22500）**、Web **2.0.35**、官网与客服中心保持。

源码 [PR #96](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/96)：`67837e3`，合并 `2f544a3934b2261308f2cb4a2821e9b2a6b80dee`。Core 版本声明修正 [PR #97](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/97)：`d172e75`，合并 `d11e29e1eca158587e8cd85277a274934cc72705`。维护工作区 `.worktrees/game-chat-ldlib` / `xiwanzi/game-chat-ldlib`。

## 更新内容

- LDLib2 原生完整聊天、HUD、私聊、附近 50 米及现有团队路由；图片、个人贴纸、表情、图片查看与提示音。
- `Enter` 打开完整聊天，`T` / `/` 保留薄输入栏和指令补全。游戏内动态图片与贴纸当前显示首帧；随包图片编解码组件为 Windows x64。
- Core 签发游戏聊天专用会话；Go 复用现有媒体与贴纸接口，隔离私聊、附近及团队附件访问权限。迁移新增 `037_game_chat_origin.sql`、`038_scoped_chat_media.sql`。
- 保留主线 YSM 身份能力及家具 registry 同步修复。服务器其他插件、模组及现有配置哈希均未变化。

## 产物和发布验证

| 产物 | SHA-256 |
| --- | --- |
| 聊天客户端 0.1.0（18,884,623 字节） | `9e05029cb8f6130aa51c165cdb68c64d0b8ff91db43d9ab63ae6e9dc09332a77` |
| McPatch 0.0.44-dev.tar（18,889,216 字节） | `1d962493d87399e4da1e047f9379d4432d1e63ebf8e39fb5798ce4ff3aa53daf` |
| Core 1.0.4 | `4138df8e7ac761361bbbcad8c43ff8844833d17560a79a8d1a0a7b03e09e9c26` |
| Go Linux amd64 | `c8d64b388574bed2bbc54692fb586d3612819728a7905e92664ebe57f48cf893` |

- 客户端单元测试、Core 35 项测试、Go 全量测试以及游戏会话、媒体隔离和 YSM 身份集成测试通过；版本声明修正后重新执行 Core `verify`，35 项通过。
- GUI 2/3/4 各 105 步及 4 项断言通过；UI 验收范围见[本机记录](../qa/game-chat-ldlib-20261011.md)。不将本机媒体夹具视为生产用户实发验收。
- 实际 DLauncher McPatch 更新测试通过：0.0.43-dev → 0.0.44-dev 仅新增聊天 JAR、更新开屏显示版本；个人设置、存档和无关模组保留，二次检查无重复更新。
- OSS 同步成功，公网 HTTPS 索引与本机发布索引一致，全部 58 个旧条目保留；完整 TAR SHA-256 和 Range 元数据校验通过。
- Login / Amiya / Odyssey 在确认无人在线后逐服更新并正常启动；Go 核对三节点在线且 Core 存储健康。MEK 文件已更新，保持停服。
- 首次 Login 启动检查发现 JAR 描述仍为 1.0.3，发布脚本自动恢复原 1.0.2 后重新构建、发布正确的 1.0.4。最终四份插件文件均为上表哈希；未向公网发布该中间包。

## 数据和回退

原有 **61 条账号、47 笔订单、5 个委托、3,085 件在售商品、12 个工单、2 条注销记录**保留。切换前后核对账号、交易、目录、附件、工单 ID 集合；全部旧记录仍在。数据库仅新增迁移，原迁移校验和一致。后端配置、App 更新清单、启动器配置及三个静态站点链接保持。

- Go：`/opt/deuterium/releases/game-chat-67837e3`；备份 `/var/backups/deuterium/game-chat-0.0.44-67837e3`，含一致性数据库备份及旧二进制/配置。失败回退旧 Go 链接时保留新增表和新增业务数据，不整库覆盖。
- 客户端：`/var/backups/deuterium/game-chat-client-0.0.44-67837e3`，保存旧索引、开屏配置、变更清单及打包/同步记录。已下载的客户端应通过新的 McPatch 版本进行前向回退，不只回退索引。
- Core：`C:/Users/ovo/CodexStaging/game-chat-0.0.44-d172e75/{login,amiya,odyssey,mek}` 保存各节点原 JAR 和配置；停服后恢复该节点原包，恢复原在线状态。

生产核验未发送聊天、图片、邮件或测试订单；游戏真实用户的图片/私聊/团队交互仍待上线后实际使用反馈。

[Go 切换证据](artifacts/game-chat-0044/go-deployment.json) · [启动器更新测试](artifacts/game-chat-0044/client-updater-verification.json) · [公网与保留项核验](artifacts/game-chat-0044/public-verification.json) · [OSS 同步](artifacts/game-chat-0044/sync-result.json)

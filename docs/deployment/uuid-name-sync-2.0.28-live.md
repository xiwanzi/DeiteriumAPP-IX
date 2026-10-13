# 正版改名、事件同步与聊天交互联合上线

2026-10-14（UTC+8），前一晚授权的更新已全部推送、合并并发布。App/Go/Web 于 **00:09** 完成切换，三个游戏节点于 **00:12** 恢复就绪，启动器于 **00:31** 发布，游戏资源于 **00:32** 同步完成、**00:39** 公网完整校验通过。

## 正式版本及源码

| 组件 | 当前版本 | 合并记录 |
| --- | --- | --- |
| App / Web / Go / Core / 聊天 | App **2.0.28（22800）**、Web **2.0.39**、Core **1.0.7**、聊天 **0.1.11** | [PR #126](https://github.com/xiwanzi/DeiteriumAPP-IX/pull/126)，`62f4da02fb3a` |
| Teams | **0.3.4** | [PR #1](https://github.com/Deuterium-IX/DeuteriumTeams/pull/1)，`b3f846642f35` |
| Cloud / Sparkle | Cloud **2.0.1**；Sparkle 兼容标识保持 **2.0.0+d9.1**，构件已更新 | [PR #11](https://github.com/Deuterium-IX/DeuteriumYSM/pull/11)，`f9dee452c55e` |
| Mail 插件 | **0.7.4**；Bridge、协议及原客户端 UI 保持 | [PR #5](https://github.com/Deuterium-IX/Deuterium-Mail/pull/5)，`39eacc5c72cd` |
| DLauncher | **0.6.8**，程序发布 revision **7** | [PR #8](https://github.com/Deuterium-IX/DeLauncher/pull/8)，`eef9f5959c99` |
| 游戏资源 | **0.0.60-dev** | OSS 同步 `ls_fde043ade30267ee81effc86a0e603a6` 成功 |

维护工作树：`C:/DeiteriumAPP-IX/.worktrees/uuid-name-sync`，分支 `xiwanzi/uuid-name-sync`。配套仓库工作树和前一候选基线见[改名验收](../qa/uuid-name-sync-20261013.md)。Teams 远端缺失的历史提交已与当前已部署 0.3.2 服务端 / 0.3.3 客户端哈希核对后补齐，未回退现有驿站功能。

## 已生效内容

- 同 UUID 正版改名继续使用原账号和密码；更新结构化名称，保留资产、昵称、头衔、有效会话和历史原文。覆盖 App/Web、游戏聊天、公会、订单、Cloud、启动器及邮件名单。
- 名称同步由已认证会话变化触发；移除固定名字扫描和 App/Web 四秒全量会话刷新。公网实时连接按用户/频道分发，失败独立退避，重连补齐。
- 正版查询不占密码计算名额；公会同步使用独立进度，不阻塞 Core 和聊天。在线目录只写变化，通用 HTTP/WebSocket 编码不再遍历整份 JSON 改名字。
- 游戏详细聊天页在加号右侧增加离开底部时显示的小倒三角，点击立即回到最新消息；世界、附近、团队和私聊统一适用。
- 回车在输入框未聚焦时先聚焦，已聚焦且有内容时发送；长按不误发草稿。
- TrChat 聊天、告示牌、铁砧屏蔽词和云词库此前已关闭；本轮重启后四服配置再次确认保持关闭。

## 数据与运行状态

App 切换前后 **61 账号、47 订单、5 委托、3,085 件在售商品、12 工单、2 注销记录**保持，原业务 ID 集合均包含于切换后数据。只新增迁移 `041_player_name_sync.sql` 的名称状态与日志表，原 36 项迁移校验未改；新迁移使用与 Git 一致的 LF 字节，SHA-256 `844630451d88bac4efb6c07e71c886aa469e4adece8b8d52f0c6d9b9991e7110`。

Cloud 原 **7 账号、6 身份、283 模型、7 条玩家状态、3 个 NPC**及模型 ACL/版本记录保留，原环境配置哈希不变。Cloud 程序现位于 `E:/Deuterium_IX/services/SPMCloud/releases/uuid-name-f9dee452c55e/spm-cloud.exe`，数据继续使用原目录。

Login、Amiya、Odyssey 正常停启后就绪，公会 revision **23**；Amiya 报告 **684** 项资产。四服仅替换 Core、Mail 插件及 Teams JAR，其他插件/模组与关键配置哈希核对通过。Velocity 未重启，MEK 只更换文件、继续停止。三个运行节点的 Core 存储健康；Core 状态接口的既有 `version` 字段仍硬编码为 `1.0.0`，实际 **1.0.7** 通过加载日志、插件描述及文件哈希独立核验。

官网、客服站、独立图片审核服务及其 PID 保持。原 **28** 条 App 发布记录与 **74** 条游戏资源记录保留。

## 构件核验

| 产物 | 字节 | SHA-256 |
| --- | ---: | --- |
| App 22800 | 9,088,574 | `f9302c4a8a78b5c32820221ca79ad3bdc75b88da54d237ae7e77b4b1acfe9cbc` |
| Go Linux amd64 | 22,098,080 | `5d5a763a1edd22c07cb180a3daa18b307f9fe6096580bfd3b98c398600669180` |
| Core 1.0.7 | 15,821,898 | `48481c25762a81cb602a478cd7e15e72d443e482ac27c3b3180fc81acaa7b7da` |
| 聊天 0.1.11 | 18,937,572 | `f309a3930d019adab2d80c0c2894fa4af4b6df645c0997f8cb5a77df8210db09` |
| Teams 0.3.4 | 17,906,461 | `72ccb0fb62f4a25fe44e0d7753db79dcd9fbe0d4fd831a0fa5db3a149b98435c` |
| Mail 0.7.4 | 1,221,674 | `b13105ee7b6e571474079527639747f7a1db109e3d93f628b4e8317e368bd911` |
| Cloud 2.0.1 | 13,648,896 | `859ac0621db049c38fb7dd5fdbd18a3642862c75d956fcd6da8c8d1a3c724f39` |
| Sparkle 客户端 | 7,097,784 | `c8a4c4d1bb204e0fadee723f017ab91ecd6a91fd4ead41a3ae495c7f948409b0` |
| 启动器更新 ZIP | 284,944,401 | `75f34bf208fb23778a43adce1d8257f29c21c4cc552ab50e651fe034a2c8acb5` |
| 资源 0.0.60-dev TAR | 43,948,544 | `4f49d7f4db71327ad36ba9674ee695d542c4b38f5244ad0f2024a8cd963d69c6` |

- App performance 包经过 R8/lint，非 debuggable，签名证书与旧版相同：`04a213066a78a968c2a1328a153a0967f99d117e0a826285e3f4af705db6f2f3`。隔离 Android 35 模拟器 22700→22800 覆盖安装与桌面入口启动成功。
- App 公网完整下载哈希一致；22800/22801 不重复更新或降级。两个 Web 入口的 2.0.39 配置及首页 JS/CSS 与本机构件一致。
- 公网 WSS 认证、同步能力提示、名称变更提示及 ping/pong 通过，没有发送实际聊天消息；临时验收和发布会话全部撤销。
- 服务器只读访问官方 Minecraft 名字查询成功，并与一个已有账号的绑定 UUID 相符；没有改写玩家身份。
- 启动器 102 项测试与 849 文件审计通过；实际签名安装及打包 EXE 启动后，已有新版账号/加密凭据、游戏、Java、配置和社区哨兵均保留。公开签名信封与完整 ZIP 校验通过。
- 资源包恰好六项变化：聊天、Teams、Sparkle、开屏显示版本，以及移除两个旧聊天/Teams JAR。真实更新器 59→60、逐文件校验、个人设置/存档/其他模组保留、重复更新无操作通过；公网 TAR 完整哈希与 Range 206 元数据一致。
- 候选功能测试、原生三种 GUI 缩放和浏览器请求量证据见[架构验收](../qa/uuid-name-event-architecture-20261013.md)与[聊天交互验收](../qa/game-chat-composer-20261013.md)。

## 切换中的处理与备份

首次后端切换因未先执行迁移而触发就绪失败，已自动恢复旧程序。随后确认运行账号仅有业务读写权限，使用本机数据库维护账号完成新增表迁移，没有扩大运行账号权限。更新清单按后端启动时加载的既有规则，在最终启动前原子替换；未通过更新接口核验的尝试均回退程序和清单。最终上线以 00:09 的成功回执为准。

Cloud 的计划任务停止后，旧子进程仍独立运行；完成 SQLite 一致性备份并核对 PID/程序路径后，停止该旧进程并启动新程序，数据与 ID 核验通过。

- 最终 App/Go/Web 备份：`/var/backups/deuterium/uuid-name-v228-62f4da02fb3a-r1-retry3`；先前尝试的备份与回退证据保留。
- 游戏插件及三个 MySQL 库备份：`E:/Deuterium_IX/servers/_deployment-backups/uuid-name-20261013`。
- Cloud：`E:/Deuterium_IX/services/SPMCloud/backups/uuid-name-20261014-retry2`。
- 游戏资源：`/var/backups/deuterium/uuid-name-client-0.0.60-20261014`。
- 启动器：`/var/backups/deuterium/launcher-0.6.8-20261014`。
- 本机回执：工作树 `output/night-release/`；App 交付：`C:/DeuteriumAPP/delivery/Deuterium-2.0.28/`。

回退仅恢复相应程序、更新清单和被替换文件；不得用旧整库覆盖发布后的新业务。真实正版账号实际改名仍待条件具备后验收，本次不宣称已完成真实改名。

玩家需在 App 检查更新；退出游戏后通过启动器接收 **0.6.8 / 0.0.60-dev**。本轮未强行关闭玩家本机正在运行的游戏。

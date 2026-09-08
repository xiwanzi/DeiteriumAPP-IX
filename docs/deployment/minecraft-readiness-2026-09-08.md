# Minecraft Core / Mail 部署准备核查

核查日期：2026-09-08。对象为当前用户授权的 App 2.0.0、网页、后端与 Minecraft 联通发布。本文是部署前只读快照，**不是已部署或业务验收完成的记录**。

本次未启停任何 Minecraft 实例，未修改生产数据库、账号或余额。准备核查阶段仅只读；后续按主任务明确授权上传官方账号导出工具到受限 staging，尝试只读账号导出，结果见下文。Core 工作区仍由原任务开发，本次只核对其代码与契约；没有改动该工作区。

## 1. 主机与节点身份

新 App/网页后端主机为用户指定的 Linux `47.103.99.34`。它不是 Minecraft 主机；域名 `lnyozu.cn` 的备案尚未完成，测试访问地址按本次发布配置确定。

Minecraft 主机为已有 SSH 别名 `ovo-server-codex` / `ovo-server` 指向的 Windows `orgna.net:3944`。本次成功只读连接并核对正式根目录 `E:/Deuterium_IX/servers`。

| 节点 ID | 目录 | 游戏监听 | 本次监听状态 | YouerModSync playerdata | 邮箱领取策略 |
| --- | --- | --- | --- | --- | --- |
| login | Deuterium_Login | 127.0.0.1:25565 | 运行 | false | 禁领 |
| amiya | Deuterium_Amiya | 127.0.0.1:25566 | 运行 | true | 仅在真实同步屏障和物品兼容验证通过后开放 |
| odyssey | Deuterium_Odyssey | 127.0.0.1:25567 | 运行 | true | 同上 |
| mek | Deuterium_MEK | 127.0.0.1:25572 | 本次未监听 | true | 默认禁领；未证明与主服属于同一背包域 |

以上端口来自本次读取的各服 `server.properties`，状态来自 `Get-NetTCPConnection`。MEK 的旧日志有启动完成记录，不能据此认定本次仍运行。测试、镜像、备份目录不属于这四个正式节点，不能替代目标服。

当前正式路线为 **Youer / Minecraft 1.21.1 / Java 21**；本次运行中的三个 Youer JVM 使用 Java 21.0.12.1。MEK 既有日志明确显示 NeoForge 21.1.248、Minecraft 1.21.1。原根文档的 Mohist 1.20.1 是旧系统背景，不能用于选择本次 Mail/Core 产物。

## 2. 本次实际安装清单

四服均发现 TrChat 2.4.9、Vault、XConomy Bukkit 2.26.3、YouerModSync 0.7-deuterium.2，以及 LDLib2 2.2.38.a。

四服均**未发现 DeuteriumCore、DeuteriumMail、DeuteriumMailBridge 或 Mail UI**。独立 `DeuteriumItems-1.21.1-1.0.0.jar` 是已有 MOD，不能把它当作本次 Core 自带的版本化物品库。

四服 TrChat 已有 `Normal`、`Global`、`Private`、`Staff` 频道文件。Core 默认公共频道 Normal/Global 与目录匹配；仍需真实最终事件验证过滤后的正文、禁言和去重。私聊、Staff、验证码不得转发进公共频道。

YouerModSync 配置均指向本机 MySQL/MariaDB 的 `minecraft` 库；本次没有读取其数据或凭据。主机本次有 MariaDB 3306 和回环 Redis 6379 监听。Core/Mail 应使用各自独立 schema，不复用该同步库的业务表。

## 3. Mail 0.6.0 产物与依赖

源码位置：`C:/Mod/DeuteriumIX/work/mail/Deuterium-Mail`，开发分支 `codex/mailbox-integration-v1`，存在该任务尚未提交的工作。已核对本地产物 SHA-256 与该任务验收记录一致：

| 产物 | 安装位置 | SHA-256 |
| --- | --- | --- |
| deuterium-mail-0.6.0.jar | 服务端 plugins | F9AAD8202E27F30AE021DC3D09F45B45AD89C6B063245C5D4B2459B3F1FA2641 |
| deuterium-mail-bridge-0.6.0.jar | 服务端 plugins | 3F855586AE229D35E4A71D587BE22A21B22969D1387E952AEB0532B743729D1C |
| deuterium-mail-ui-neoforge-1.21.1-0.6.0.jar | 服务端和客户端 mods | 5634AB812ED3E8B2EAE19B2C5F72952429208A656E4C3E3DDFD333C49104D966 |

Mail UI 的实际声明要求 Minecraft 1.21.1、NeoForge >=21.1.221；客户端另需 LDLib2 >=2.2.37 和 modId `deuterium_ui` >=0.1.11。仓库已包含 `vendor/ldlib2-uikit/0.1.11/ldlib2-uikit-neoforge-1.21.1-0.1.11.jar`，但发客户端前仍应按实际 JAR 元数据及客户端现装版本核对。共享契约由 Mail MOD 唯一携带，不另放第四份 contract JAR。

旧邮箱任务记录 243 项不同测试通过，新增共享数据库用例另在隔离 MariaDB 11.8.8 复测；这是此前任务证据，本次仅核对产物哈希，没有重跑或声称完成真实游戏验收。

## 4. 功能就绪与明确缺口

后续版本核对更正：原始 0.7 的 `onQuit` 清理 capability 行为已由线上 `.1/.2` 补丁规避，不能将原版反汇编当作当前 `.2` 的运行结论。Core 工作线已按实际 `.2` SHA 基线保留此修复，继续补明确的加载/保存成功回执与会话屏障；同时核对 `modules.vault=false`，防止独立 XConomy 共享余额被旧同步值恢复。最终以新产物与实服验证为准。

| 链路 | 当前证据 | 部署放行前仍需完成 |
| --- | --- | --- |
| 游戏公共聊天 → 后端 → App/网页 | Core 开发代码已有 TrChat 最终公共发送事件适配、持久 outbox；Foundation 后端有公共消息与节点 ACK | 等 Core 定稿，实际三/四服过滤、双向、断线重连、去重验证 |
| App/网页 → 游戏公共聊天 | 后端有按目标节点保存的投递记录，Core 直接显示避免 TrChat 再传播 | 验证每个在线服只显示一次；离线目标恢复后补投且不形成环路 |
| UUID/验证码 | Core 在开发玩家解析、私密验证码能力 | 与真实认证会话及后端注册/重置完整联调 |
| 钱包与转账 | 现服有 XConomy/Vault；Core 有余额和转账受控调用、RPC journal 路线，默认经济关闭 | 统一 authority-node；后端业务路由、未知结果恢复、对账以及专用测试身份的小额验证 |
| 邮件创建/查询/严格撤回 | Mail 0.6.0 提供版本化 Bukkit 服务、共享存储、幂等交付和撤回互斥 | Core/Go 最终 DTO 与事件映射、独立共享库配置、真实 Youer 类加载验证 |
| 商城付费附件领取 | Mail 要求真实 MailPlayerDataBarrier；Core 提供 PlayerDataService 适配接口 | 现有 YouerModSync 的成功加载、连接代次、排斥切服/覆盖、同步保存回执和 MOD 兼容适配仍缺实服证据，不能填假 true 开放领取 |
| 客户端邮箱界面 | Mail UI 与依赖产物存在 | 向实际测试客户端交付匹配包，真实连服、背包满、禁领、跨服领取验证 |

本次在 Core 开发中代码发现的具体对齐点已报告主任务：

1. Core 默认 `mail.consumer-id=core-deuterium`；Mail 0.6.0 的 `DeliveryEvents` 只接受 `deuterium-backend`，否则拉取和 ACK 均拒绝。须在 Core 默认、校验及部署示例统一。
2. Core 默认 `node.cluster=deuterium`；Mail 默认 `integration.cluster-id=deuterium-production`。实际发布必须显式配置一致的测试集群身份和独立节点 ID，不复制默认节点配置到四服。
3. 当时 `MailboxAdapter.pumpEvents` 直接序列化含嵌套 receipt 的 Java Event；邮箱契约要求明确映射 deliveryId/orderId/recipientUuid/mailRevision/occurredAt 等字段。最终 Core/Go 需共同固定信封、类型、时间格式与 committed ACK；本次快照的 Foundation Go 尚未处理邮箱事件。

这些是有时间边界的开发中发现；Core 任务修复后应以最终测试结果更新，不能把它们当成永久状态。

## 5. 旧 App 账号迁移来源

Minecraft Windows 主机当前仍运行旧 App 1.0.4 的 `SupervisorMainKt` 与 `ApplicationKt`。可通过该进程的 `ExecutablePath` 反推发布目录，正式配置位于其 `backend/config/application.conf`，避免按相似目录名选错旧包。

本次该配置的 JDBC 目标为 `127.0.0.1:3306/deuterium_app`，只读 SQL 结果：**app_users 70 行，70 行状态均为 active**。未输出账号清单、QQ、UUID、密码哈希或连接凭据。

新后端官方 `export-legacy` 从 app_users 导出 id/server_uuid/current_game_id/qq/password_hash/status/created_at/updated_at；后续通过 `import-legacy` 预检再导入新的独立库。正式迁移产物必须留在受保护的临时位置，不进入源码、日志或交付压缩包。

旧库还存在 transfers、wallet_records、wallet_balances、chat_messages、sessions、AI 等表。仅导入 app_users 不等于账单、聊天、AI、会话等全部迁移。余额仍以 XConomy/Vault 权威查询为准，不将旧缓存表复制成新的资金权威。

初始核查未发现被授权的专用测试身份；随后用户明确授权：付款测试身份 xiwanzi、收款 luoyinwuchen1、网页管理员 xiwanzi；只有 xiwanzi 余额不足时才允许通过游戏控制台补款。

本次只读确认 xiwanzi 已注册旧 App 且 active，绑定 UUID 与 Login/Amiya/Odyssey 三服缓存一致。luoyinwuchen1 三服缓存存在一致 UUID，XConomy 也有账号，但旧 app_users 未注册。稳定 ID、UUID 与余额快照已保存到本机当前用户专属 ACL 文件，未写入本文。

XConomy 共享库持久余额快照为 xiwanzi 99,974,286.00、luoyinwuchen1 799.00；这是数据库读数，执行测试前仍须经 Core/Vault 重新查询。MCSManager 的实时实例详情确认 Login/Amiya/Odyssey 为 1.21.1、RUNNING，三个节点在线人数均为 0；本次快照两名测试玩家均未在这些节点在线。没有执行转账、补款或聊天消息测试。

### 官方导出兼容性问题

按新增授权使用已交付 Foundation Windows 工具 `delivery/Deuterium-Backend-Foundation-0.1.0/binaries/windows-amd64/deuterium.exe`，SHA-256 为 `341BC9BD007B71EFB1986D449757DC41898E55B08CB1B1CB036DAF86F6F48B5C`。上传到当前远端用户独占的 `CodexStaging/DeuteriumLegacyExport-20260908-mc-audit` 后校验一致，凭据只在导出进程环境中构造和使用。

第一次 `export-legacy` 安全失败并删除不完整文件：70 行的身份、UUID、游戏名、时间、状态及 Argon2i 参数符合新工具校验，但有 **1 行旧 QQ 不符合新导入器的纯数字规则**。没有修改旧库，也没有绕过整批校验导出部分账号。主任务需协调正式迁移工具处理该遗留记录及登录别名策略，再重试官方导出；在成功导出前不提供伪造的文件哈希或“70 账号已迁移”结论。

该遗留记录不是两个授权测试身份；其 QQ 非空、包含英文字母及标点、无空白、长度未越界。对应游戏名格式有效，且与其他游戏名和有效 QQ 别名无冲突；全库游戏名大小写折叠后的重复组为 0。主任务确定迁移兼容策略：保留旧账号原 ID/UUID/密码与原记录，不创建无效 QQ 的登录别名，仍允许原游戏名登录。

后续主任务已使用修复后的官方独立 identity 工具成功导出、预检并导入 70 个账号和 139 条登录别名，再次预检全部 unchanged。本文维护者另只读核对本机受限导出文件为 70 行，SHA-256 `F0665A45D9EC5AF6B19E7F31F3AA52FED12B8F217214368CE474CCBC97F4F485`；导入与复查结果来自本轮主任务记录。旧库、密码与会话没有因只读导出而改变，最终切换仍需核对增量。本段覆盖上文第一次安全失败的阶段状态，不把该失败写成持续阻塞。

### 同步源码定位结果

已对本机 `C:/Mod/DeuteriumIX/work` 及相关源码文件名做有界搜索，并复核既有同步修补工程与历史检索。当前找到的是 `work/plan/plugin-builds/YouerModSync-0.7-deuterium.1` / `.2` 的精确二进制补丁、`work/your/YourFix` 字节码修复、以及 `work/mail/review-checks/YouerModSync-SyncListener.bytecode.txt` 的反汇编，未找到完整 `YouerModSync.java` / `SyncListener.java` 上游工程。

`.1` 的补丁源码明确说明当时工作区没有原二进制的上游源码；`.2` 仅修复两处 Player→ServerPlayer 字节码校验并保持同步行为。它们不提供可审计的完整加载成功、排斥切服/保存、持久保存回执接口。真实屏障需在完整同步实现的成功与失败路径接入；不能用这些补丁“能加载”或集合为空来证明发放安全。

## 6. 受控部署、备份与回滚步骤

1. 等 Core 最终产物与后端契约定稿，生成 App/网页/Go/Core/Mail/客户端依赖的统一版本与哈希清单；先在隔离环境完成相关联调。
2. 在 Linux 准备独立后端库和服务配置，在 MC 主机准备独立 Core/Mail schema、逐节点凭据与配置。凭据通过受保护配置或环境提供，文档只列键名；游戏桥仅由 MC 主动连后端。
3. 使用本次已核实的 MCSManager：实际控制面为 `E:/Deuterium_VIII_mcdr/mcsmanager`，daemon 回环端口 24444；通过官方 Socket.IO `auth` 和 `instance/detail` 只读验证了三个正式实例。`E:/r39Mcsm/mcsmanager` 只发现旧数据，不能用作本次控制面。既有 `C:/Users/ovo/CodexStaging/McsInstanceControl/Control-McsManagerInstances.ps1` 含历史一次性请求授权，不能直接重放；正式变更需按当前授权生成新请求并核实配置。各目录 start.bat 与当前 JVM 参数不完全一致，不应直接按旧脚本重启。不得关闭另一个仍运行的旧 1.20.1 服或旧 APP 后端。
4. 为正式目标记录在线玩家及当前状态，使用已核实管理器受控停服并等世界/同步保存完成、进程退出、JAR 解除占用；不热替换正在执行的插件，不 force kill。
5. 按节点备份待替换插件/MOD、相关配置、Core/Mail 数据与共享库一致性快照；核实备份绝对路径位于指定备份根。旧库/存档必须保留，不以清理为前置条件。
6. 上传到 staging 后先校验哈希，再安装确定清单。四服分别设置身份与策略；Login/MEK 禁领，Amiya/Odyssey 也必须以真实 barrier 能力决定。当前缺失插件属于新增安装，回滚不应误删其他业务插件。
7. 按原运行顺序受控恢复目标节点，逐项验证加载、节点鉴权、TrChat 模式、Vault 提供者、共享库、邮箱能力与 schema；核实三端真实调用。未验证成功保存屏障时保持相关发放入口关闭，并把限制作为未完成项处理。
8. 使用明确授权的专用测试账号进行双向聊天、同 ID 重试、断网恢复与小额转账；核对借贷双方和实际经济账、原操作结果。邮件需覆盖领取/撤回竞争、禁领服、满背包及切服保存。
9. 出现加载或协议故障时先停止新业务入口，再受控停目标节点、恢复本轮备份或移除本轮新增 JAR，按原配置恢复。若已经产生新资金/邮件业务，不可直接回滚数据库覆盖新事实；先按原 operationId 对账，保留 journal/outbox/UNKNOWN 记录。

## 7. 证据入口

- 当前 Core 计划：`C:/DeuteriumAPP/.worktrees/backend-rewrite/docs/plans/deuterium-core-v1.md`。
- 后端基础版与生产只读发现：`C:/DeuteriumAPP/.worktrees/backend-rewrite/docs/plans/deuterium-backend-rewrite.md`。
- Mail 对接：`C:/Mod/DeuteriumIX/work/mail/Deuterium-Mail/docs/mailbox-integration-v1.md`。
- Mail 历史本机验收：`C:/Mod/DeuteriumIX/work/mail/Deuterium-Mail/docs/qa/mailbox-integration-2026-09-08.md`。
- 本次新增证据：SSH 只读目录/配置/监听核对、现运行 JVM 版本、Mail 三个 JAR SHA-256、旧 app_users 的只读行数统计。

下一步由主任务协调 Core 定稿、真实同步适配、后端补全和测试身份，然后执行受控部署并另记实际结果。本文件不替代部署结果记录。

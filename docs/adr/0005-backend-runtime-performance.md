# ADR 0005: 后端运行时性能与实时消息处理

## Status

Accepted

## Date

2026-05-01

## Context

DeuteriumAPP 已实现账号、钱包、聊天、玩家目录、关心玩家、后台尽量保活接收和 Android 通知等第一阶段能力。后端运行时出现过明显延迟：玩家目录和关心/取消关心操作响应慢，聊天历史同步会周期性放大数据库压力，插件桥事件处理可能阻塞插件请求回复。

当前产品约束不变：

- 不修改 Android 前端接口和现有 API 返回结构。
- 不修改 Minecraft 桥接插件，也不要求用户更新插件。
- 后台接收仍采用登录态 App 级 WebSocket 尽量保活，不使用前台服务常驻通知。
- 系统杀掉 App 进程后不承诺继续接收消息。
- App 不直连数据库，不直接修改服务器经济数据。

## Decision

后端运行时性能优化采用“缓存优先、批量读取、读写分离、桥回复优先”的实现策略。

具体决策如下：

- 高频读取接口不得在读取列表时顺手创建或更新 `player_refs`。
- 聊天历史、钱包流水、玩家目录等列表读取必须批量解析玩家资料，避免按条目重复查库。
- 玩家目录和关心列表默认使用后端缓存的服务器在线快照，不直接实时请求插件。
- 在线状态接口使用缓存优先；缓存缺失或过旧时再触发节流刷新。
- 插件桥 WebSocket 收到 `replyTo` 响应时必须优先完成 pending request；插件主动事件进入后端事件队列异步处理。
- App WebSocket 广播不得被单个慢连接阻塞；发送失败或超时的连接应从 hub 清理。
- 数据库性能优化通过 Flyway 迁移新增索引，不改已有表字段，不删除数据。

## Implementation

### 2026-05-03 兼容性性能修复补充

本 ADR 接受后，后端继续在“不改 Android、不改插件、不改旧 API 语义”的边界内补充了 Repository 层性能和并发修复：

- 高频写路径优先使用数据库原子能力，余额缓存写入使用 MySQL 兼容 `INSERT ... ON DUPLICATE KEY UPDATE`，且不使用 MySQL 8.0.20+ 已弃用的 `VALUES(column)` 函数。
- App presence 首次写入和已有记录更新都使用单条 upsert，不再使用 `UPDATE -> insertIgnore -> UPDATE` 兜底。
- 验证码尝试次数改为数据库原子自增，避免并发丢计数。
- 钱包流水和转账创建写入后直接构造返回值，不再读回刚插入记录。
- 幂等钱包流水写入改为直接插入并捕获重复主键。
- 玩家引用创建增加固定 stripe lock，降低同一后端实例内重复活跃引用风险，同时避免按玩家生成无界锁对象。
- 新增 6 小时间隔过期数据清理任务，清理 session、验证码、登录失败、聊天历史和服务器事件中的过期数据。
- 新增 `V4__maintenance_indexes.sql`，只增加维护和清理路径索引，不改字段、不删现有数据。
- 登录失败计数改为数据库原子 upsert 递增，避免并发登录失败时 attempts 丢更新。
- 维护清理循环跟随后端进程生命周期启动，不再绑定到 bridge Ktor module。

### 数据读取

`repository/Repositories.kt` 当前实现要求：

- `AccountRepository.findByServerUuids` 批量读取 App 用户。
- `PlayerRefRepository.findActiveByServerUuids` 和 `findByPlayerRefs` 批量读取玩家引用。
- `ChatRepository.listMessages` 读取最近聊天历史时，先批量加载发送者账号和玩家引用，再生成 `ChatMessage`。
- `WalletRepository.listRecords` 直接使用流水表里的对方玩家快照作为兜底，不再在读取流水时写 `player_refs`。
- `ChatRepository.listPlayerDirectory` 一次性读取 App presence、关心关系、已注册用户和玩家引用，再排序生成目录。
- `followPlayer` 使用数据库幂等插入，重复关心不报错；`unfollowPlayer` 重复取消不报错。

`ensurePlayerRef` 仍然保留在写入或身份确认路径中使用，例如：

- 账号资料生成。
- 收款玩家搜索确认。
- 新聊天消息入库。
- 插件在线快照落库。
- 钱包转账和钱包事件写流水。

### 在线状态

`routes/Routes.kt` 当前实现要求：

- `/chat/player-directory` 和 `/chat/follows` 只读取 `presence_snapshots` 缓存。
- `/chat/presence` 和 `/chat/online-players` 缓存存在时立即返回。
- 在线快照超过 15 秒时，后端在后台触发刷新，但接口仍先返回缓存。
- 缓存缺失时，接口允许同步刷新一次。
- 后端最多每 5 秒启动一次非强制在线刷新，避免并发请求打爆插件桥。
- 插件不可用时返回最近缓存，不因为在线刷新失败影响聊天历史、玩家目录或关心列表读取。

### 插件桥

`bridge/WebSocketPluginBridge.kt` 当前实现要求：

- 后端请求插件时，通过 `messageId -> CompletableDeferred` 关联响应。
- 收到带 `replyTo` 的插件消息时，立即完成对应 pending request 并返回，不进入事件处理队列。
- `chat.serverMessage.event`、`server.event`、`presence.snapshot.event`、`wallet.pay.event`、`wallet.balanceChange.event` 进入后端事件队列顺序处理。
- 事件处理失败只记录日志，不阻塞后续插件回复。
- `presence.snapshot.event` 处理时批量读取注册账号，并在一个数据库事务内更新玩家引用和在线快照。

该设计保证：插件桥上的慢事件处理不会拖慢 App 发聊天、刷新余额、转账、验证码投递等需要插件回复的请求。

### App WebSocket Hub

`chat/AppChatHub.kt` 当前实现要求：

- 广播时先复制当前连接快照，再离开锁执行发送。
- 群发和单用户发送均并发发送。
- 单个连接发送超时为 1500 毫秒。
- 发送失败或超时的连接从 hub 中移除。
- `broadcastMessage` 仍然面向所有 App 聊天连接广播普通聊天消息。
- `sendWalletRecord` 仍然只发给对应用户的连接。

该设计保证：后台常驻连接数量增加后，单个异常连接不会拖慢其他用户接收最新消息。

### 数据库迁移

`V3__performance_indexes.sql` 只新增索引：

- `player_refs(server_uuid, registered, expires_at)`
- `app_users(status, current_game_id)`
- `chat_messages(sent_at, id)`
- `wallet_records(user_id, occurred_at, id)`
- `app_presence(last_seen_at)`

该迁移不改字段、不删数据、不改变旧接口语义。生产部署时运行 `migrate-db.bat` 或启动后端即可执行 Flyway 迁移。

`V4__maintenance_indexes.sql` 只新增维护索引：

- `sessions(expires_at, revoked_at)`
- `verification_requests(expires_at)`
- `login_failures(updated_at, locked_until)`
- `server_events(occurred_at, id)`

该迁移用于支撑过期数据清理任务，不改变 App API、插件桥协议或任何表字段。

## Compatibility

本次后端性能实现保持以下兼容性：

- Android App 不需要更新。
- Minecraft 桥接插件不需要更新。
- 插件桥消息类型和 payload 不变。
- App HTTP API 路径、请求字段、响应字段不变。
- 已有账号、会话、聊天历史、钱包流水、关心关系和在线快照数据保留。
- 生产交付包可以继续包含旧插件 jar；只要 bridge token 与后端配置一致即可。

## Verification

后端验证要求：

- `backend-api\gradlew.bat test prepareWindowsRuntime` 必须通过。
- 测试覆盖聊天历史读取不会创建或更新 `player_refs`。
- 测试覆盖钱包流水读取不会创建或更新 `player_refs`。
- 测试覆盖玩家目录仍保持服务器在线玩家优先，并按 App 状态排序。
- 测试覆盖关心和取消关心幂等。
- 交付前校验生产包内插件 jar SHA256 与当前指定插件一致。

手动验收建议：

- 后台常驻 App WebSocket 时，服务器普通聊天仍能实时到达 App。
- 被关心玩家普通聊天在 App 后台触发通知，非关心玩家不触发。
- 连续点击关心/取消关心，状态快速切换且不明显卡顿。
- 聊天页停留 1 分钟以上，15 秒同步不会导致明显卡顿。
- 玩家目录仍显示服务器在线玩家优先。

## Consequences

后端读路径复杂度下降，列表接口不再因为每条记录写 `player_refs` 而放大数据库压力。插件桥回复优先后，慢事件处理不会拖慢需要即时响应的 App 操作。

代价是在线玩家列表采用缓存优先，可能存在很短延迟。该延迟接受，因为插件主动上报在线快照和节流刷新会继续保持列表接近实时。

后续如果用户规模显著增长，应继续补充：

- 增量聊天历史接口和 cursor 分页。
- 钱包流水 cursor 分页。
- 插件事件队列容量和丢弃策略指标。
- 后端运行指标和慢查询日志。

## Non-Goals

本 ADR 不做以下事情：

- 不新增 App API v2。
- 不修改桥接插件协议。
- 不引入 Redis、消息队列或新的运行时基础设施。
- 不把后台接收升级为 Android 前台服务。
- 不承诺系统杀掉 App 进程后仍能接收消息。

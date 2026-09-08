# 后端运行时性能实现说明

本文记录 2026-05-01 至 2026-05-03 后端性能增强后的实际技术实现。架构决策见 `docs/adr/0005-backend-runtime-performance.md`，生产交付仍以 `docs/production-delivery.md` 为准。

## 1. 兼容边界

本轮性能增强只修改后端：

- 不修改 Android 前端工程。
- 不修改 Minecraft 桥接插件工程。
- 不修改 App HTTP API 路径、请求字段或响应字段。
- 不修改插件桥消息类型和 payload。
- 不要求更换现有桥接插件 jar。

后台接收语义保持不变：App 登录后维持 App 级 WebSocket 尽量保活，App 退后台不主动断开；但 Android 系统杀掉 App 进程后，不承诺继续接收。

## 2. 读取路径优化

高频读取路径已经避免“读接口顺手写 `player_refs`”：

- `GET /chat/messages` 批量读取最近聊天记录的发送者资料，不再每条消息调用 `ensurePlayerRef()`。
- `GET /wallet/records` 使用流水表中的对方玩家快照作为兜底，不再读取流水时创建或更新玩家引用。
- `GET /chat/player-directory` 一次性读取 App presence、关心关系、已注册用户和玩家引用，再生成玩家目录。
- `POST /chat/follows` 和 `DELETE /chat/follows/{playerRef}` 只构造目标玩家的返回状态，不重建整份目录。
- App presence 不再无条件全表读取；玩家目录按当前活跃用户集合读取，单个玩家返回只读取目标用户相关 presence。
- 验证码冷却查询使用 `ORDER BY created_at DESC LIMIT 1`，不再把全部匹配记录加载到 JVM 后做 `maxByOrNull`。
- `findActiveByServerUuids` 保持接口语义不变，但改为单次遍历选出每个 UUID 的最佳玩家引用，避免 `groupBy + 每组排序` 的额外内存开销。

`ensurePlayerRef()` 仍用于身份确认和写入路径，例如聊天新消息入库、在线快照落库、收款人搜索确认、转账流水写入等。

## 3. 在线状态与玩家目录

玩家目录和关心列表使用 `presence_snapshots` 缓存，不直接请求插件。

在线状态接口采用缓存优先：

- 缓存存在时，`/chat/presence` 和 `/chat/online-players` 立即返回缓存。
- 缓存超过 15 秒时，后端后台触发一次刷新，但当前请求仍先返回缓存。
- 缓存缺失时，接口允许同步刷新一次。
- 非强制刷新最短间隔为 5 秒，避免多个 App 同时打开页面时并发请求插件。
- 插件不可用时返回最近缓存，避免在线刷新失败拖慢聊天和玩家目录。

## 4. 插件桥事件处理

`WebSocketPluginBridge` 当前以“回复优先、事件排队”的方式处理消息：

- 带 `replyTo` 的插件响应立即完成对应 pending request。
- 插件主动事件进入后端事件队列顺序处理。
- 事件处理失败只记录日志，不阻塞后续插件响应。
- `presence.snapshot.event` 会批量解析注册账号，并在一个数据库事务内更新在线快照。

这样可以避免服务器聊天、在线快照或钱包事件处理过慢时，拖慢 App 发聊天、刷新余额、转账或验证码投递等需要插件响应的请求。

## 5. App WebSocket 广播

`AppChatHub` 当前广播策略：

- 广播前复制连接快照，避免持锁发送。
- 群发和单用户发送都并发执行。
- 单连接发送超时为 1500 毫秒。
- 发送失败或超时的连接会从 hub 移除。
- 后端使用标准 WebSocket ping/pong 探活，默认 15 秒 ping、35 秒超时，用于快速清理半开 App 连接。
- 普通聊天仍广播给所有聊天 WebSocket 连接。
- 钱包变动事件仍只发送给对应 App 用户。

该策略保证后台常驻连接增多后，单个慢连接或移动网络半开连接不会拖慢其他用户接收最新消息；探活是协议控制帧，不访问数据库，也不进入聊天广播路径。

## 6. 写入路径与并发修复

2026-05-03 的后端兼容性性能修复只改变 Repository 内部实现，不改变 App API、插件桥协议或响应字段：

- `WalletRepository.upsertBalance` 使用 MySQL 兼容 `INSERT ... ON DUPLICATE KEY UPDATE`，不使用 MySQL 8.0.20+ 已弃用的 `VALUES(column)` 函数；余额刷新写入从“先查、写入、再查”变为单条数据库语句，返回值由已知参数构造。
- `ChatRepository.updateAppPresence` 使用单条 upsert；App presence 首次写入和已有记录更新都只走一次数据库往返。
- `VerificationRepository.incrementAttempts` 使用数据库原子自增，避免并发验证码尝试丢计数。
- `WalletRepository.createTransfer` 和钱包流水 `insertRecord` 写入后直接构造返回对象，不再立即 `SELECT` 刚写入的记录。
- `WalletRepository.createRecordIfAbsent` 去掉先查 `any()`，改为直接插入并捕获重复主键，保证插件重复上报同一流水时保持幂等。
- `PlayerRefRepository.ensurePlayerRef` 增加固定 64 条 stripe lock，避免同一后端实例内并发创建重复活跃玩家引用，同时避免按玩家生成无界锁对象。
- `ChatRepository.updatePresence` 使用单条 upsert 保持在线快照幂等更新。
- `LoginFailureRepository.recordFailure` 使用 MySQL 兼容 upsert 原子递增登录失败次数，再读取结果返回给登录流程；不再读出 attempts 后在 JVM 中加一。
- `isDuplicateKey()` 不再把 MySQL 通用 SQLState `23000` 全部当作重复键，只识别 H2/PostgreSQL 的 `23505` 和 MySQL duplicate entry 错误码 `1062`。

## 7. 数据生命周期清理

后端启动后会在进程生命周期内启动轻量维护任务，默认每 6 小时执行一次。该任务只清理已经过期或超出保留期的数据，不影响当前业务数据：

- 删除已过期 session，以及长期前已撤销的 session。
- 删除过期超过 1 天的验证码上下文。
- 删除超过 1 天未更新或已解锁的登录失败记录。
- 按 `chat.historyRetentionDays` 删除过期聊天记录。
- 按同一保留期删除过期服务器事件。

清理任务失败只记录 warning，不会阻塞 App API、插件桥或后端启动。

## 8. 数据库迁移

本轮新增 Flyway 迁移：

```text
V3__performance_indexes.sql
V4__maintenance_indexes.sql
```

这些迁移只新增索引，不改字段、不删数据。V3 用于支撑玩家引用批量读取、玩家目录排序、聊天历史排序和钱包流水排序。V4 用于支撑过期 session、验证码、登录失败记录和服务器事件清理。

生产部署时应先运行：

```bat
backend\migrate-db.bat
```

再运行：

```bat
backend\start-backend.bat
```

只运行启动脚本通常也会自动执行迁移；单独先运行迁移脚本更容易提前发现数据库问题。不要为了执行 V3/V4 迁移运行 `reset-database.bat`。

## 9. 验证要求

后端交付前必须运行：

```powershell
cd backend-api
.\gradlew.bat test prepareWindowsRuntime
```

当前测试覆盖：

- 余额 upsert 每次写入只使用一条数据库语句。
- 已存在的 App presence 更新不再预查询。
- App presence 首次写入和更新都只使用一条数据库语句。
- presence snapshot 首次写入和更新都只使用一条数据库语句。
- 登录失败计数第 5 次失败时进入锁定状态。
- 钱包流水和转账创建写入后不再读回刚插入的记录。
- 聊天历史读取不会创建或更新 `player_refs`。
- 钱包流水读取不会创建或更新 `player_refs`。
- 增量钱包流水查询仍按旧到新返回。
- 玩家目录仍保持服务器在线玩家优先，并按 App 在线状态排序。
- 关心和取消关心重复点击保持幂等。
- 维护任务只清理过期运营数据，保留未过期 session。

交付包还必须校验插件 jar 未被意外更新。当前指定插件 SHA256：

```text
5E19DCEE84C5CA98A84E52DFC9906B72250C71D09BEDA5F6D62DCC2028C22170
```

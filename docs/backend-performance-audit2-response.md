# 后端性能审计第二轮回应

日期：2026-05-03

本文回应 `backend-performance-audit2.md`。目标不是为代码辩护，而是把成立的问题、已经修复的问题、暂不修的问题和后续边界说清楚。当前产品规模是 Deuterium VIII 玩家服务器，日活约 20-40 人；后端优化必须服从兼容性要求：不改 Android 前端、不改 Minecraft 插件、不改旧 API 语义。

## 已采纳并修复

### 登录失败计数竞态

审计指出 `LoginFailureRepository.recordFailure()` 仍是“读 attempts -> JVM 加一 -> 写回”。这个问题成立。

已修复为数据库原子 upsert 递增：

- 新 key 插入 `attempts = 1`。
- 已存在 key 通过 `ON DUPLICATE KEY UPDATE attempts = attempts + 1` 原子递增。
- 第 5 次及之后失败写入 `locked_until`。

新增测试覆盖第 5 次失败进入锁定状态。

### upsert SQL 的 `VALUES(column)` 风险

审计指出 MySQL 8.0.20+ 已弃用 `ON DUPLICATE KEY UPDATE` 中的 `VALUES(column)` 形式。这个问题成立。MySQL 官方文档也说明该用法已 deprecated，未来版本可能移除。

已修复为不使用 `VALUES(column)` 的写法，例如直接使用新的参数值：

```sql
ON DUPLICATE KEY UPDATE
  amount = ?,
  currency = ?,
  fresh = ?,
  refreshed_at = ?
```

余额 upsert 仍保持单条数据库语句。

### App presence 和在线快照三步兜底

审计指出 `UPDATE -> insertIgnore -> UPDATE` 过于复杂。这个问题成立。

已修复：

- `updateAppPresence()` 首次写入和已有记录更新都使用单条 upsert。
- `updatePresence()` 首次写入和已有记录更新都使用单条 upsert。
- 测试已覆盖这两个路径的 SQL 语句数量。

### 重复键判断过宽

审计指出 `sqlState == "23000"` 会把 MySQL 的通用约束违反都当成重复键。这个问题成立。

已修复为只识别：

- `23505`：H2/PostgreSQL 的唯一约束违反。
- `1062`：MySQL duplicate entry 错误码。

不再把所有 `23000` 都吞掉。

### 玩家引用锁对象无界增长

审计指出 `ConcurrentHashMap<String, Any>` 按玩家永久保存锁对象。这个问题成立，虽然当前规模下不是高风险内存问题。

已修复为固定 64 条 stripe lock：

- 锁对象数量固定。
- 避免按玩家增长。
- 仍然降低同一后端实例内并发创建重复活跃玩家引用的风险。

### 维护任务绑定 bridge module

审计指出清理任务只在 `bridgeModule()` 启动。当前实际生产进程同时启动 public 和 bridge，所以不是当前故障风险，但把维护任务绑定到某个 Ktor module 不够清晰。

已调整为跟随后端进程生命周期启动，不再绑定到 bridge module。

## 暂不采纳为本轮修复

### 全面替换 `selectAll()`

审计说所有 `selectAll()` 都应该改成精确投影。方向正确，但本轮不全面替换。

原因：

- 很多仓库方法目前返回完整领域对象，例如 `RegisteredUserRecord`、`TransferRecord`、`PlayerRefRecord`，不是所有调用点都只需要少数字段。
- 强行局部投影容易引入“某个调用点后来需要字段但对象里为空”的维护风险。
- 当前用户规模下，`selectAll()` 的主要实际风险集中在高频列表和包含 `password_hash` 的用户查询，不是所有查询同等严重。

后续更合理的做法是按用途拆专用投影模型，例如：

- `AccountIdentityRecord`
- `PlayerDirectoryUserRow`
- `SessionAuthRow`

再逐步替换高频路径，而不是机械把所有 `selectAll()` 改成散落的字段列表。

### `listActiveUsers()` 分页

审计指出 `listActiveUsers()` 每次玩家目录加载所有 active 用户。这个方向成立，但本轮不做分页。

原因：

- 当前玩家目录的产品语义是“列出所有已注册活跃玩家 + 当前服务器在线玩家”，分页会改变前端可见结果和排序语义。
- Android 端当前没有目录分页协议；后端单独分页会破坏旧客户端体验。
- 当前规模下几十到几百 active 用户仍可接受，真正应优先做的是投影裁剪和缓存，而不是分页。

后续可做兼容优化：

- 保留旧接口全量返回。
- 新增可选分页参数或新 endpoint。
- 旧客户端不传参数时行为不变。

### `findActiveByServerUuids()` 改 SQL 窗口函数

审计建议使用 `ROW_NUMBER() OVER (PARTITION BY ...)`。方向正确，但本轮不做。

原因：

- 这会引入更强的数据库版本假设。
- 当前方法已从 `groupBy + 每组排序` 降为单次遍历，内存复杂度和 CPU 已明显下降。
- 该方法输入本身是调用方提供的 serverUuid 集合，当前不会无边界扫描全表。

后续如果确认生产 MySQL 版本稳定支持窗口函数，可单独做一轮 MySQL 专项 SQL 重构和 explain 验证。

### `synchronized` 改协程 `Mutex`

审计指出 `synchronized` 在协程环境里不理想。这个判断有技术依据，但本轮不改为 `Mutex`。

原因：

- 当前 Repository API 是同步函数，运行在 Exposed 事务和 `Dispatchers.IO` 中。
- 改成 `Mutex.withLock` 需要把 `ensurePlayerRef()` 及大量调用链改成 `suspend`，影响面明显扩大。
- 该锁只保护同一个玩家引用的短写入路径，且已改为固定 stripe lock，当前风险可接受。

后续如果要系统性协程化 Repository，可以再统一处理锁策略；不建议为了一个局部点把同步仓库接口半改成挂起接口。

## 结论

第二轮审计中关于并发计数、upsert 写法、重复键判断、三步 fallback 和锁对象增长的批评成立，已修复。

关于 `selectAll()`、玩家目录分页、窗口函数和 `synchronized` 的批评有方向价值，但本轮不作为 backend-only 热修处理。它们更适合进入后续 Repository 分层重构：先定义投影模型和 MySQL explain 验证，再逐步改高频读路径。

当前交付目标仍然是：只替换后端即可提升稳定性和性能，不要求用户更新 Android App 或 Minecraft 插件。

# DeuteriumAPP 后端性能审计报告（第二轮）

> 审计范围：`backend-api/` 全部 Kotlin 源码、SQL 迁移文件、测试文件
> 第一轮审计日期：2026-05-03
> 第二轮审计日期：2026-05-03
> 结论：修了一半，另一半换了种姿势继续错。

> 当前状态：2026-05-03 已完成第二轮后端修复。已采纳的问题包括登录失败计数竞态、MySQL `VALUES(column)` 弃用风险、App presence / presence snapshot 三步兜底、重复键误判和无界锁对象。未采纳为本轮热修的点已单独回应，见 `docs/backend-performance-audit2-response.md`。

---

## 第一轮修复验收

以下问题已被正确修复：

| # | 问题 | 修复方式 | 评价 |
|---|------|----------|------|
| 2.1 | `upsertBalance()` 三次查询 | 原生 SQL UPSERT | 正确，但见下方新问题 |
| 2.3 | `incrementAttempts()` 竞态 | `attempts = attempts + 1` 原子更新 | 正确 |
| 2.4 | `insertRecord()` INSERT 后 SELECT | 直接构造返回对象 | 正确 |
| 2.5 | `updateTransferStatus()` UPDATE 后 SELECT | `record.copy()` 模式 | 正确 |
| 1.1 | `appPresenceByUserId()` 全表扫描 | `WHERE userId IN (...)` 过滤 | 正确 |
| 3.1 | `findActiveByServerUuids()` groupBy + sort | `fold()` 单遍扫描 | 改善，但仍非最优 |
| 3.2 | `latestActiveCooldown()` maxByOrNull | `ORDER BY ... DESC LIMIT 1` | 正确 |
| 6.2 | `createRecordIfAbsent()` CHECK-THEN-ACT | 捕获 `ExposedSQLException` | 正确 |
| 7 | 无数据过期清理 | `MaintenanceRepository` + 定时任务 | 正确 |
| 9 | 无性能回归测试 | `CountingSqlLogger` + 5 个测试 | 正确 |

---

## 未修复的问题

### 1. `selectAll()` 仍然是默认行为

**文件**：`Repositories.kt` 全文

整个仓库层没有任何一处使用 `select(...)` 指定投影列。所有查询都是 `selectAll()`：

```kotlin
Users.selectAll().where { Users.qq eq qq }.singleOrNull()
Sessions.innerJoin(Users).selectAll().where { ... }
AppPresence.selectAll().where { AppPresence.userId inList uniqueUserIds }
```

`Users` 表包含 `password_hash` 字段。每次 `findByQq()`、`findByServerUuid()` 等查询都会把密码哈希一起拉出来，即使调用方只需要 `userId` 和 `serverUuid`。`selectAll()` 是代码生成工具的懒惰写法，不是合理的技术选择。

### 2. `listActiveUsers()` 仍然全量加载

**文件**：`Repositories.kt:80-84`

```kotlin
fun listActiveUsers(): List<RegisteredUserRecord> =
    Users.selectAll()
        .where { Users.status eq "active" }
        .orderBy(Users.currentGameId to SortOrder.ASC)
        .map { it.toRegisteredUser() }
```

这个方法被 `listPlayerDirectory()` 每次调用。它加载所有活跃用户的全部字段（包括密码哈希），然后在内存中映射。没有投影，没有分页，没有利用覆盖索引。

### 3. `LoginFailureRepository.recordFailure()` 仍然有竞态

**文件**：`Repositories.kt:286-306`

```kotlin
fun recordFailure(key: String): Pair<Int, Instant?> {
    val row = LoginFailures.selectAll().where { ... }.singleOrNull()
    val attempts = (row?.get(LoginFailures.attempts) ?: 0) + 1
    if (row == null) {
        LoginFailures.insert { ... }
    } else {
        LoginFailures.update({ ... }) { ... }
    }
}
```

`incrementAttempts()` 被修复了（同一文件第 242 行），但 `recordFailure()` 用的是完全相同的读-加一-写回模式，却没有被修复。两个并发的登录失败请求可以同时读到 `attempts = 4`，都写回 `5`，然后都触发锁定——或者更糟，一个写回 5 触发锁定，另一个写回 5 但没有触发锁定，因为 `lockedUntil` 的计算也基于读到的旧值。

这说明修复是逐个方法看的，不是系统性审查的。

### 4. `findActiveByServerUuids()` 仍然在应用层做分组排序

**文件**：`Repositories.kt:409-426`

```kotlin
return PlayerRefs.selectAll()
    .where { ... }
    .fold(mutableMapOf<String, PlayerRefRecord>()) { bestByServerUuid, row ->
        val candidate = row.toPlayerRefRecord()
        val current = bestByServerUuid[candidate.serverUuid]
        if (current == null || playerRefPriority.compare(candidate, current) < 0) {
            bestByServerUuid[candidate.serverUuid] = candidate
        }
        bestByServerUuid
    }
```

`fold()` 替换了 `groupBy() + sortedWith() + first()`，从 O(n log n) 降到了 O(n)，这是改善。但根本问题没变：把所有匹配记录从数据库拉到 JVM 堆里做筛选。SQL 的 `ROW_NUMBER() OVER (PARTITION BY server_uuid ORDER BY registered DESC, expires_at IS NULL DESC, confirmed_at DESC)` 可以在数据库层直接返回每组最优记录，传输量从 N 条降到 K 条（K = 去重后的 server_uuid 数量）。

---

## 修复引入的新问题

### 5. `upsertWalletBalance()` 绕过 Exposed 类型安全

**文件**：`Repositories.kt:491-522`

```kotlin
private fun upsertWalletBalance(userId: String, amount: BigDecimal, refreshedAt: Instant) {
    val transaction = TransactionManager.current()
    val dialect = transaction.db.dialect.name.lowercase()
    val sql = if ("mysql" in dialect || "mariadb" in dialect) {
        """INSERT INTO wallet_balances ... ON DUPLICATE KEY UPDATE ...""".trimIndent()
    } else {
        """MERGE INTO wallet_balances ... KEY (user_id) ...""".trimIndent()
    }
    transaction.exec(sql, listOf(...), StatementType.UPDATE)
}
```

问题：

1. **方言检测用字符串匹配**：`"mysql" in dialect` 是脆弱的。如果 Exposed 升级后 `dialect.name` 返回 `"MySQL8"` 或 `"MariaDB10"`，逻辑会错误地走到 H2 分支。应该用 Exposed 的 `VendorDialect` 类型体系判断。

2. **MySQL 的 `VALUES()` 函数在 MySQL 8.0.20+ 已被弃用**：应该用 `ON DUPLICATE KEY UPDATE amount = VALUES(amount)` 的新写法 `AS new SET amount = new.amount`。当前写法在 MySQL 8.0.20+ 会产生 deprecation warning，在未来版本会直接报错。

3. **两套 SQL 语义不同**：MySQL 的 `ON DUPLICATE KEY UPDATE` 基于唯一键冲突触发，H2 的 `MERGE INTO ... KEY` 基于指定键匹配触发。如果 `wallet_balances` 表有多个唯一键，两者行为不一致。

4. **绕过了 Exposed 的表定义**：SQL 语句中的列名是硬编码字符串，与 `WalletBalances` 表定义解耦。如果表结构变更（列重命名），这段代码不会编译报错，只会运行时失败。

正确做法是使用 Exposed 的 `replace()` 或 `upsert()` 扩展（Exposed 0.40.0+ 支持），或者至少用 `WalletBalances.columns` 动态生成列名。

### 6. `updateAppPresence()` 三步 fallback 比原来的两步更差

**文件**：`Repositories.kt:764-788`

```kotlin
fun updateAppPresence(userId: String, foreground: Boolean, at: Instant = Instant.now()) {
    val updated = AppPresence.update({ AppPresence.userId eq userId }) { ... }  // 尝试 1：UPDATE
    if (updated == 0) {
        val inserted = AppPresence.insertIgnore { ... }.insertedCount  // 尝试 2：INSERT IGNORE
        if (inserted == 0) {
            AppPresence.update({ AppPresence.userId eq userId }) { ... }  // 尝试 3：再 UPDATE 一次
        }
    }
}
```

原来的代码是 SELECT → INSERT/UPDATE（两步）。现在是 UPDATE → INSERT IGNORE → UPDATE（三步）。对于**已存在**的行（最常见的场景），第一次 UPDATE 就成功了，确实比原来少一次查询。但对于**不存在**的行（首次写入），变成了三次操作：

1. UPDATE 返回 0（浪费一次数据库往返）
2. INSERT IGNORE（可能成功）
3. 如果 INSERT IGNORE 因并发冲突返回 0，再 UPDATE 一次

第三步的存在是为了处理两个并发请求同时到达的场景：请求 A 的 UPDATE 返回 0，请求 B 的 INSERT IGNORE 抢先成功，请求 A 的 INSERT IGNORE 返回 0，所以需要再 UPDATE。但这个场景本身就很罕见，而且如果真的发生，第三步的 UPDATE 也可能和请求 B 的 UPDATE 产生竞态。

对比一下理想方案：

```kotlin
// Exposed 0.40.0+ 的原生 upsert
AppPresence.upsert(
    onUpdate = listOf(
        AppPresence.foreground to foregroundParam,
        AppPresence.lastSeenAt to atParam,
        ...
    )
) {
    it[userId] = userIdValue
    it[foreground] = foregroundValue
    ...
}
```

一条 SQL，原子操作，没有竞态。代码量从 12 行降到 8 行。

### 7. `updatePresence()` 同样的三步模式

**文件**：`Repositories.kt:970-992`

```kotlin
fun updatePresence(players: List<OnlinePlayer>, updatedAt: Instant) {
    val updated = PresenceSnapshots.update({ PresenceSnapshots.id eq 1 }) { ... }
    if (updated == 0) {
        val inserted = PresenceSnapshots.insertIgnore { ... }.insertedCount
        if (inserted == 0) {
            PresenceSnapshots.update({ PresenceSnapshots.id eq 1 }) { ... }
        }
    }
}
```

`PresenceSnapshots` 表只有一行（`id = 1`）。这是一个单行配置表的 upsert，完全可以用一条 SQL 搞定。三步模式在这里尤其荒谬：第一次 UPDATE 返回 0 意味着行不存在，INSERT IGNORE 失败意味着行被并发插入了，再 UPDATE 一次——对于一个只有一行的表。

### 8. `ensurePlayerRef()` 的 `ConcurrentHashMap` 锁无界增长

**文件**：`Repositories.kt:314-316`

```kotlin
companion object {
    private val playerRefLocks = ConcurrentHashMap<String, Any>()
    ...
}
```

`playerRefLocks` 的 key 是 `"$serverUuid|$registered"`。每个不同的 `(serverUuid, registered)` 组合都会在 map 中永久保留一个条目。如果服务器有 1000 个不同玩家注册过，map 中就有 1000 个条目。每个条目持有一个 `Any()` 对象的强引用。

更关键的是，这个 map **永远不会清理**。即使某个玩家注销了，他的锁对象仍然在 map 中。在长期运行的服务器上，这是一块持续增长的堆内存。虽然增长速度慢（每个玩家一个条目），但这是一个设计缺陷，不应该出现在生产代码中。

正确做法是用 `Striped<Lock>`（Guava）或 `Caffeine` 的 `evictionLock` 模式，或者干脆用数据库的 `SELECT ... FOR UPDATE` 行锁。

### 9. `synchronized` 阻塞协程调度

**文件**：`Repositories.kt:339`

```kotlin
return synchronized(lock) {
    ensurePlayerRefLocked(serverUuid, gameId, qq, registered, online, source, expiresAt)
}
```

`synchronized` 是 JVM 级别的阻塞锁。在协程环境中使用 `synchronized` 会阻塞当前线程，阻止调度器在该线程上调度其他协程。如果 `ensurePlayerRefLocked()` 内部的数据库操作耗时较长（网络延迟、锁等待），整个线程会被阻塞。

Ktor 的默认调度器线程数有限（通常等于 CPU 核心数）。如果多个请求同时进入 `synchronized` 块等待同一个锁，会耗尽调度器线程，导致整个应用挂起。

应该使用 `kotlinx.coroutines.sync.Mutex`：

```kotlin
private val playerRefMutex = Mutex()

suspend fun ensurePlayerRef(...): PlayerRefRecord {
    return playerRefMutex.withLock {
        ensurePlayerRefLocked(...)
    }
}
```

但 `Mutex` 是挂起函数，不能在非挂起的 `ensurePlayerRef()` 中使用。这意味着需要把 `ensurePlayerRef()` 改为 `suspend fun`，或者重新设计锁策略。

### 10. `playerDirectoryItem()` 单条重载仍然多次查询

**文件**：`Repositories.kt:879-917`

两个单条 `playerDirectoryItem()` 重载各自独立调用：
- `accounts.findByServerUuids(listOf(ref.serverUuid))` — 1 次查询
- `appPresenceByUserId(...)` — 1 次查询
- `followedServerUuids(currentUser.userId)` — 1 次查询

每次 follow/unfollow 操作（Routes.kt:536, 549）都会触发这 3 次查询。而 `listPlayerDirectory()` 批量版本已经加载过这些数据。follow/unfollow 路由没有复用任何缓存。

### 11. 维护清理只在 bridge 模块运行

**文件**：`Application.kt:114-138`

```kotlin
fun Application.bridgeModule(services: ApplicationServices) {
    ...
    environment.monitor.subscribe(ApplicationStarted) {
        launch {
            while (true) {
                ...
                dbQuery { services.maintenance.cleanupExpiredData() }
                ...
                delay(MaintenanceCleanupIntervalMillis)
            }
        }
    }
}
```

维护清理任务只在 `bridgeModule()` 中启动。如果部署架构是 public 和 bridge 分离运行（Application.kt:68-82 确实创建了两个独立的 `embeddedServer`），那么只有 bridge 模块会执行清理。如果 bridge 模块因某种原因重启或不可用，过期数据会持续积累。

两个模块共享同一个 `ApplicationServices` 实例和数据库连接，所以清理任务放在哪个模块都能工作。但应该在两个模块中都启动，或者放在独立的调度进程中。

### 12. `isDuplicateKey()` 检测逻辑脆弱

**文件**：`Repositories.kt:1165-1166`

```kotlin
private fun ExposedSQLException.isDuplicateKey(): Boolean =
    sqlState == "23505" || sqlState == "23000" || errorCode == 1062
```

- `sqlState == "23505"` 是 PostgreSQL/H2 的唯一约束违反
- `sqlState == "23000"` 是 MySQL 的通用约束违反（太宽泛，可能误判外键违反等）
- `errorCode == 1062` 是 MySQL 的重复键错误

问题：
1. `"23000"` 是整个约束违反类的 SQL state，包含外键违反、非空违反等。用它判断重复键会把所有约束违反都当成重复键处理。
2. 如果未来支持 PostgreSQL，需要同时检查 `sqlState` 和 `errorCode`，但当前逻辑只用 `||` 连接，不同数据库的错误码可能冲突。
3. 没有注释说明每个条件对应哪个数据库，维护者无法判断是否需要添加新的数据库支持。

---

## 测试覆盖评估

### 新增测试的评价

`CountingSqlLogger` 是一个巧妙的设计，通过 Exposed 的 `SqlLogger` 接口拦截 SQL 语句。5 个新测试验证了关键路径的 SQL 语句数量，这对防止性能回归有价值。

**问题**：

1. **`dmlStatements()` 把 SELECT 算作 DML**（第 387-395 行）：SELECT 是 DQL（Data Query Language），不是 DML（Data Manipulation Language）。虽然测试逻辑不影响正确性，但命名误导。应该叫 `sqlStatements()` 或 `allStatements()`。

2. **只测了 happy path**：`appPresenceUpdate` 测试只测了已有行的更新（第一次 UPDATE 返回 >0），没有测首次插入（UPDATE 返回 0 → INSERT IGNORE）的路径，也没有测并发竞态（INSERT IGNORE 返回 0 → 第三次 UPDATE）的路径。

3. **没有并发测试**：`ensurePlayerRef()` 的锁机制、`recordFailure()` 的竞态、`createRecordIfAbsent()` 的异常捕获——这些修复的正确性在并发场景下没有被验证。H2 内存数据库的并发行为与 MySQL 不同（H2 默认用表锁，MySQL 用行锁），测试结果不能直接推断到生产环境。

4. **H2 与 MySQL 的 SQL 方言差异未覆盖**：`upsertWalletBalance()` 有两套 SQL（MySQL 和 H2），但测试只在 H2 上运行，MySQL 分支从未被执行。

---

## 总结

第一轮修复解决了一些明显的性能热点（UPSERT、原子更新、全表扫描过滤），但修复方式是逐个方法打补丁，不是系统性重构。

**根本问题没有变**：

1. **`selectAll()` 仍然无处不在**。这是最大的遗留问题，影响所有查询的 I/O 和内存开销。
2. **CHECK-THEN-ACT 模式在 `ensurePlayerRef()` 和 `recordFailure()` 中仍然存在**。一个用了 `synchronized`（引入了协程阻塞问题），一个完全没有防护。
3. **应用层做数据库该做的事**。`findActiveByServerUuids()` 虽然从 O(n log n) 优化到 O(n)，但仍然是把所有行拉到内存再筛选。
4. **修复引入了新的技术债**：raw SQL 绕过 ORM、三步 fallback 比原方案更复杂、`ConcurrentHashMap` 无界增长、`synchronized` 阻塞协程。

第一轮修复证明了"修一个方法"的能力，但没有证明"理解系统设计"的能力。代码仍然像是逐个方法生成的，而不是作为整体设计的。

# DeuteriumAPP 后端稳定性与性能 Review

## 文档状态

- 评审对象：`DeuteriumAPP-backend-only-1.0.4-ai-sse-fix1-20260616`
- 评审日期：2026-07-15
- 评审结论：完成；存在必须先处理的 P0 依赖与运行时风险，另有两项用户明确接受的私有部署取舍
- 实施回填：已在隔离分支完成兼容性门禁、首批后端硬化、本地 Web 启动台和便携候选包；生产切换与长稳验收仍未执行
- 兼容性原则：后续后端修改不得破坏当前 1.0.4 App 的 REST、聊天 WebSocket、AI SSE、鉴权、错误码和数据结构

## 1. 执行摘要

本次没有证据表明“后端重启后一定恢复”，也没有在 2026-07-15 的低频公开探测中复现长时间连接。公开健康、就绪和版本检查共 24 次请求全部返回 HTTP 200，p95 总耗时分别约为 425 ms、192 ms 和 164 ms。

但目标后端存在数个能够解释“运行一段时间后整体连接变慢”的机制：

| 排名 | 机制 | 证据结论 | 与症状的关系 |
| --- | --- | --- | --- |
| 1 | AI Provider SSE 每个请求占用一个 `Dispatchers.IO` 工作线程和一个专用读取线程，客户端取消不能立即中断阻塞读取 | 已确认代码行为；生产触发量待验证 | 较可能造成短时线程饱和，随后 REST、握手和 SSE 一起变慢 |
| 2 | Hikari 池只有 `maximumPoolSize=10` 被显式设置，借连接最长默认等待 30 秒，且没有池指标 | 已确认配置；生产池饱和待验证 | 数据库或网络抖动时，所有需要鉴权/查库的连接可能一起长时间等待 |
| 3 | 插件主动事件进入无界、串行消费队列 | 已确认代码行为；生产积压待验证 | 会累积内存和业务延迟，严重时影响整个 JVM；更直接影响聊天、钱包和在线状态 |
| 4 | 当前 App 固定 1.8 秒重连，成功后立即触发多项同步请求 | 已确认 App 行为 | 后端、隧道或网络恢复时会形成同步重连波峰 |
| 5 | PassNAT、DNS、TCP 状态或 Windows 进程资源异常 | 当前内部证据不可取得 | 仍是待验证的外部链路原因，不能只从应用代码排除 |

此外，公网 Netty 4.1.111.Final 落入多项已修复的 HTTP 请求走私和拒绝服务漏洞范围，属于 P0 补丁缺口。生产配置随包交付、当前 APK 继续使用明文 HTTP/WS，则是用户在 2026-07-15 基于私有环境和“复制单文件夹、双击即用”目标明确接受的取舍，不再作为发布阻断项。继续优化前，仍应先建立不破坏当前 App 的兼容性门禁。

## 2. 证据等级

- **已确认**：有目标二进制、目标配置、同源源码、测试结果或本次观测直接支持。
- **较可能**：代码机制与症状吻合，但缺少生产线程、连接池、TCP 或日志数据确认其在事故时实际触发。
- **待验证**：现有权限或环境不足，不能得出结论。

“根因置信度”只表示该问题解释本次连接症状的能力，不代表问题本身是否存在。

## 3. 版本、制品与源码基线

### 3.1 正式基线

| 制品 | 大小 | SHA-256 |
| --- | ---: | --- |
| `DeuteriumAPP-backend-only-1.0.4-ai-sse-fix1-20260616.zip` | 53,728,727 bytes | `21E6CA9D54059BFEB1E775905BC3F7E7705E0F50ACECE65FEFB6FEF6B833FB34` |
| `backend/lib/deuterium-backend-api-0.1.0.jar` | 1,853,906 bytes | `239BE52CEBC3CB4D5F75C0BC8409E3EF42773E30034999F3B9234A997410951C` |
| `backend/start-backend.bat` | — | `A389113946C3AC55F10E39FE48B15F24CC3E250DB79F352F1964F077ADBCA86B` |
| `backend/migrate-db.bat` | — | `DF657CEDD3B7E56473AFD89E9B035A037E1508C6C09F2C05A7D2BD9E13FF98D1` |
| `backend/reset-database.bat` | — | `B797A1DB0BF7AADDC5DC817525152C9601C07D1F065FD2A5781D9250D6BB0BB6` |
| `backend/config/application.conf` | — | `2E0FAB9729E92C163F98C35DC42B4B7495CE2D03985F3F3C4B71A61093BFD372` |
| `backend/config/application.example.conf` | — | `D548B4A01217B4548B001ABB146FCFA1EC6053EA4601E756C857FB61D3CB4E57` |

目标包包含 67 个 JAR、11 个 Flyway SQL 资源和随包 Zulu OpenJDK 17.0.18。`start-backend.bat` 已使用 `lib\*` 通配 classpath，并在启动前检查关键依赖；历史上的 Windows 展开超长 classpath 问题在 `fix1` 中已关闭。

### 3.2 源码映射可信度

目标 JAR 与 Git checkpoint `20f080d` 的隔离重建结果：

- 双方均有 873 个 ZIP entry。
- 所有 `.class` entry 逐字节一致。
- 原始字节不同的 11 个迁移 SQL 和 `logback.xml`，在 CRLF/LF 归一化后 SHA-256 全部一致。
- 因此，本报告以 `20f080d` 源码定位目标二进制行为，执行代码映射可信度为高。

目标 JAR 与 2026-06-17 `pony-ultra` checkpoint `35d2d29` 比较时，仅 678 个 entry 相同，90 个同名 entry 内容不同，目标独有 105 个，pony 独有 112 个。`pony-ultra` 只能作为补丁来源，不能直接替换当前生产基线。

### 3.3 当前 App 基线

| 制品 | SHA-256 | 结论 |
| --- | --- | --- |
| `DeuteriumAPP-1.0.4-ai-sse-20260616.apk` | `E8ED1789808D8FF07F8C1B187CBF747F53D81C3288B5B3D802C774B71C8F778C` | 当前兼容性基线，`versionCode=6`、`versionName=1.0.4` |

以 `20f080d` 隔离重建 Debug APK 后：

- 142 个 APK entry 中 140 个逐字节一致。
- 差异仅在 `classes5.dex`、`classes6.dex`；结构比较只显示 UI 编译产物大小变化，网络/API/WebSocket/SSE 类型没有结构差异。
- Android `testDebugUnitTest` 16 个测试全部通过，Debug APK 可成功构建。

据此，当前 App 的冻结网络面为：27 个 Retrofit REST 接口、1 个 `/api/v1/chat/ws` WebSocket、1 个 `/api/v1/ai/chat/stream` SSE。

## 4. 生产只读观测

### 4.1 本次可取得证据

2026-07-15 对以下端点执行 8 轮、每轮间隔 5 秒的顺序低频采样：

| 端点 | 样本 | HTTP 200 | 总耗时 p50 | 总耗时 p95/最大值 |
| --- | ---: | ---: | ---: | ---: |
| `/health/live` | 8 | 8 | 122.1 ms | 424.8 ms |
| `/health/ready` | 8 | 8 | 128.1 ms | 191.6 ms |
| `/api/v1/app/update-check?versionCode=6&versionName=1.0.4` | 8 | 8 | 122.8 ms | 163.8 ms |

随后单次读取确认：`live.alive=true`、`ready.ready=true`。本次样本只能说明观测窗口内轻量请求正常，不能证明长稳、并发、AI、数据库或隧道状态正常。

### 4.2 无法取得的证据

SSH alias `ovo-server-codex` 两次低频只读连接均在密钥交换前被远端关闭：`kex_exchange_identification: Connection closed by remote host`。因此未取得以下生产内部证据：

- JVM uptime、CPU、RSS、线程、句柄和 GC；
- TCP 状态、监听端口和重启历史；
- Hikari 活跃/空闲/等待连接；
- MySQL/MariaDB 版本、会话、等待和超时；
- PassNAT 进程、隧道重连和错误日志；
- 后端持久日志和事故时线程栈。

这些项目均标记为待验证。本报告不把历史会话中出现过的 PassNAT 异常当作当前生产事实。

## 5. 本地验证

### 5.1 已完成

- 在系统临时目录通过 `git archive 20f080d` 建立隔离源码，不切换当前分支、不覆盖未跟踪工程。
- 基线后端执行 `clean test jar` 成功：7 个 suite、35 个测试、0 failure、0 error。
- 硬化候选后端执行 `clean test` 成功：11 个 suite、45 个测试、0 failure、0 error。
- Android 候选执行 `testDebugUnitTest` 成功：4 个 suite、18 个测试、0 failure、0 error。
- 后端测试覆盖 SSE delta/done、重复 `clientMessageId`、部分流停顿、提前 EOF、隐藏活动总超时、迟到成功不得完成失败 exchange，以及 Repository SQL 往返约束。

### 5.2 未完成且不得表述为通过

没有执行原计划中的 15 分钟预热、60 分钟混合负载和 6 小时低频长稳，原因是：

- 本机没有 MySQL/MariaDB/Docker；
- 生产数据库主版本因 SSH 不可用而无法只读确认；
- 目标包内 `application.conf` 是生产配置，出于安全边界绝不用于本地启动或连接生产数据库；
- 不能用不同主版本数据库冒充“与生产同主版本”的验收。

因此，40/100 并发和 6 小时资源回落指标当前状态为**待验证、发布阻断**。具体执行方式已写入 `docs/plans/backend-hardening-console.md`。

## 6. 当前 App 兼容性冻结面

后续改后端时，至少冻结以下行为：

| 协议面 | 当前 App 依赖 |
| --- | --- |
| REST | `/api/v1` 下 27 个方法的 method/path/query/body、HTTP 状态、`requestId/data/page` 成功 envelope、`requestId/error` 失败 envelope |
| 鉴权 | `Authorization: Bearer <token>`；401 会清除本地会话并要求重新登录 |
| WebSocket | URL `/api/v1/chat/ws?includeServerEvents=...`；客户端发送 `app.state`、`chat.send` |
| WebSocket 下行 | `chat.message`、`chat.mention.event`、`server.event`、`presence.update`、`chat.send.result`、`error`、`wallet.record.event` |
| SSE | `meta`、`status`、`sources`、`delta`、`done`、`error`；EOF 未收到 terminal event 时视为失败；心跳为 SSE comment |
| 超时 | 普通 HTTP connect/read/write 为 12/20/20 秒；AI SSE read 为 45 秒 |
| 重连 | 聊天 WebSocket 失败后固定等待 1.8 秒；连接成功立即同步历史、在线状态/目录和关心列表 |

冻结不代表永不改进。新增字段、新本地管理端点和内部指标必须是 additive，并通过旧 APK 契约回归；删除、重命名、改类型、改既有状态码或改 SSE terminal 语义均为阻断项。

## 7. 用户接受的部署取舍

### A-01 生产凭据随便携交付包提供

- **状态/优先级**：已确认；Accepted Risk，不是 P0，也不阻断零配置交付。
- **证据**：根 README 明确说明 `application.conf` “沿用上一版生产配置”；安全扫描确认数据库密码、session pepper、verification pepper 和 plugin bridge token 均为非占位值。报告未输出任何值。
- **影响**：获得 ZIP 的人可取得数据库和桥接凭据；pepper 泄露会削弱会话/验证码哈希隔离。ZIP 复制、聊天传输、备份或网盘都会扩大暴露面。
- **复现条件**：读取交付 ZIP 即可，不需要启动服务。
- **根因置信度**：对连接变慢为低；对安全风险为确定。
- **用户决策**：最终交付必须是已经填好当前生产连接参数的单一文件夹；复制到服务器后不得要求编辑 `application.conf`、设置环境变量或重新录入 token。为实现这一目标，实际配置可以随专用生产包交付。
- **保留措施**：配置使用相对路径并由启动台只读加载；不得在 UI、日志、进程命令行和诊断包中再次显示配置值；不因为本次硬化主动轮换 session pepper 或 plugin token，以免当前 App 会话和插件连接失效。
- **验证方式**：将完整文件夹复制到包含空格和非 ASCII 字符的新路径，双击 `launch-console.bat`，无需输入或编辑即可连接现有数据库并启动；当前 1.0.4 App 既有会话继续可用；日志和诊断包不重复泄露值。

### A-02 当前对应 Debug APK 使用明文 HTTP/WS

- **状态/优先级**：已确认；Accepted Risk，不是本轮整改或发布阻断项。
- **证据**：交付 README 指定对应 Android Debug APK；其 BuildConfig 使用 `http://deuterium.s.odn.cc/api/v1/` 和 `ws://deuterium.s.odn.cc/api/v1/chat/ws`，且允许 cleartext。Bearer token、登录密码、聊天和钱包请求都经过该链路。
- **影响**：不可信网络可窃听或篡改账号、会话及业务数据；WebSocket 也没有传输层保护。
- **复现条件**：使用当前 Debug APK 访问公网服务。
- **根因置信度**：对连接变慢为低，但中间网络设备干预明文长连接的可能性高于 TLS；对安全风险为确定。
- **用户决策**：该环境由用户私有管理，当前优先级是兼容和免配置，不要求新增 TLS、证书部署、重定向或域名调整。
- **实施约束**：后端继续监听当前 `28657` 并保持原 HTTP/WS URL；本轮不得用 HTTPS/WSS 改造迫使 App 更新。若未来用户主动要求 TLS，应另立兼容迁移，不与稳定性硬化混发。
- **验证方式**：原 APK 通过原 URL 完成功能回归，覆盖 SSE 45 秒读超时及 WebSocket 断线重连；启动台不提示用户配置证书。

## 8. Critical

### C-03 公网 Netty 4.1.111.Final 落入多项已修复漏洞范围

- **状态**：受影响版本已确认；具体攻击路径可达性待结合 PassNAT/代理拓扑验证。
- **证据**：目标包包含 `netty-codec-http`、`netty-codec-http2`、`netty-common` 等 4.1.111.Final。OSV 批量扫描识别出 28 个 Maven coordinate，其中 11 个 coordinate 有 advisory；Netty HTTP 包包含多项请求走私和 DoS 修复。官方 4.1.135.Final release 汇总了安全修复；NVD 明确 `CVE-2026-33870` 在 `<4.1.132`、`CVE-2026-42581` 在 `<4.1.133` 受影响。
- **影响**：公网请求可触发协议解析差异、请求走私或资源耗尽；目标后端位于隧道/代理之后时，解析差异更值得关注。
- **复现条件**：取决于 HTTP/1.x、HTTP/2、代理转发和相关 handler 是否启用。不能仅凭版本声称所有 CVE 都可利用。
- **根因置信度**：对日常长稳变慢为中低；对补丁缺口为确定。
- **修复方向**：先在 Ktor 2.3.12 上以受控 dependency constraint/BOM 将 Netty 4.1.x 提升到至少 4.1.135.Final，并独立跑契约、协议和负载测试；若二进制兼容失败，再单独规划 Ktor 3.x 迁移，不能把框架大版本迁移和业务硬化混成一次发布。
- **验证方式**：SBOM/OSV 复扫；HTTP 请求走私 corpus、畸形 chunk、HTTP/2 frame flood 防护测试；27 REST、WS、SSE 的旧 APK 回归；40/100 并发指标通过。

参考：[Netty 4.1.135.Final 安全修复](https://github.com/netty/netty/releases/tag/netty-4.1.135.Final)、[NVD CVE-2026-33870](https://nvd.nist.gov/vuln/detail/CVE-2026-33870)、[NVD CVE-2026-42581](https://nvd.nist.gov/vuln/detail/CVE-2026-42581)。

## 9. Important

### I-01 AI Provider SSE 的阻塞读取不能随客户端取消立即释放

- **状态**：已确认代码机制；生产饱和待验证。
- **证据**：`AiService.streamAttempt` 在 `Dispatchers.IO` 中调用阻塞的 `HttpClient.send`；每个流创建 `Executors.newSingleThreadExecutor()`，用 `Future.get(timeout)` 等待 `BufferedReader.readLine()`。关闭 App/取消 Ktor coroutine 没有取消 Java HTTP request 或当前 blocking future 的显式通路。
- **影响**：一个失去下游客户端或卡住的 Provider 流，在首 token/idle/总超时前仍可占一个 IO worker 和一个专用线程。连续断流可把多类请求一起拖慢，最坏持续到当前总超时 120 秒。
- **复现条件**：Provider 停顿、移动网络断开、App 切页取消或隧道半开，且同时存在多条 AI 流。
- **根因置信度**：较可能。
- **修复方向**：改为单个可关闭的异步/suspending HTTP client 和 `ByteReadChannel` 类读取；取消 call 时立即关闭上游 body；增加全局及单用户 active stream 上限、排队上限、输出字符/字节上限和指标；保留现有 SSE event 与 terminal 语义。
- **验证方式**：真实 Netty 端到端断开测试；取消后 1 秒内 active stream、专用线程和上游连接回落；重复 100 次断流后线程不增长；旧 APK 仍收到相同 SSE event 顺序和错误码。

### I-02 数据库池耗尽时可能让鉴权和握手等待约 30 秒

- **状态**：配置行为已确认；生产池耗尽待验证。
- **证据**：目标仅显式设置 JDBC URL、用户、密码和 `maximumPoolSize=10`，未设置 `connectionTimeout`、`validationTimeout`、`keepaliveTime`、`maxLifetime`、pool name、JMX/metrics。Hikari 官方文档说明借连接达到上限后最多等待 `connectionTimeout`，默认 30 秒。
- **影响**：DB 慢查询、网络抖动或连接失效时，登录、REST、WebSocket 鉴权和 presence 写入都可能排队，表现为“前端很久才能连接”。
- **复现条件**：10 个连接均被占用或失效，且新的请求需要 `dbQuery`。
- **根因置信度**：较可能。
- **修复方向**：先采集 active/idle/pending/acquire time；根据数据库连接上限设池大小；显式设置较短且经压测验证的 connection/validation timeout；`maxLifetime` 小于数据库/网络连接寿命，按环境设置 keepalive；超时统一返回既有兼容 error envelope。
- **验证方式**：数据库暂停、kill connection、网络黑洞和慢 SQL 实验；池恢复后 pending 回零；当前 App 不出现解析错误或无限重试。

参考：[HikariCP 配置和快速恢复说明](https://github.com/brettwooldridge/HikariCP)。

### I-03 插件主动事件使用无界队列且单协程串行消费

- **状态**：已确认代码机制；生产积压待验证。
- **证据**：`WebSocketPluginBridge` 使用 `Channel.UNLIMITED`，事件串行执行数据库写入和 App 广播。replyTo 回复被优先处理是正确的，但事件数量和处理延迟没有容量、深度、年龄或拒绝指标。
- **影响**：插件事件突发或数据库变慢时，JVM 内存和事件延迟可持续增长；钱包/聊天/在线状态最终变得陈旧，极端时 OOM。
- **复现条件**：事件生产速度长期高于串行消费速度。
- **根因置信度**：较可能解释运行时间越长越差；是否在线触发待验证。
- **修复方向**：先加 queue depth/oldest age/process latency；再设计有界队列。钱包事件不可静默丢弃，应补 event id、ack、幂等和重放，或在溢出时通过可恢复协议反压。不能只照搬 pony 的 `capacity=1024` 后继续 `trySend` 失败日志，否则会制造资金/状态丢失。
- **验证方式**：持续事件洪峰、DB 短暂不可用、桥重连；队列有界且恢复后归零；每个钱包事件恰好产生一次结果或明确告警/重放。

### I-04 插件桥重连存在旧 session 清空新请求的竞态

- **状态**：已确认。
- **证据**：新 session 接入时先关闭旧 session；任何 `accept` 的 `finally` 都会对全局 `pending.values` complete exceptionally 并 `clear()`，即使退出的是已经被替换的旧 session。与此同时，新 session 已可能发送新的 pending request。
- **影响**：插件重连窗口内，原本经新连接发送的验证码、聊天、余额刷新、转账或 AI 购买请求可被旧连接的 finally 错误判失败。
- **复现条件**：A session 被 B 替换，B 创建 pending 后 A 的 finally 执行。
- **根因置信度**：对桥相关间歇失败为高；对最初 WebSocket 握手慢为低。
- **修复方向**：引入 connection generation；pending 记录所属 generation；仅当前 generation 断开时清理其请求；`session/lastSeenAt` 使用单一原子快照或全部在同一锁内读取。
- **验证方式**：可重复双 session 交错测试至少 1,000 次；B 上的请求不得被 A 关闭；旧回复不得完成新 generation 的请求。

### I-05 WebSocket 帧和消息并发缺少硬上限

- **状态**：已确认。
- **证据**：Ktor WebSockets 配置 `maxFrameSize=Long.MAX_VALUE`；公共聊天没有每连接/每用户消息速率限制和连接数上限；OneBot 每收到一个文本帧就 `launch` 一个子 coroutine。
- **影响**：超大帧可导致分配压力；恶意或失控客户端可制造无界任务、DB 写和桥请求。它也放大重连洪峰。
- **复现条件**：已认证 App、持有本地桥 token 的进程或错误的 OneBot 客户端发送大帧/高频帧。
- **根因置信度**：较可能造成局部或全局停顿；是否在线触发待验证。
- **修复方向**：直接回收 pony 的 64 KiB frame cap；增加连接数、每连接入站速率和 OneBot 有界并发。64 KiB 远高于当前 App 的 256 字符聊天和现有 envelope，不改变正常契约。
- **验证方式**：当前 APK 全功能；64 KiB 以下正常，超限以明确 close code 关闭；100 并发突发不出现 5 秒全局停顿。

### I-06 健康语义和日志无法定位退化状态

- **状态**：已确认。
- **证据**：`/health/live` 永远返回 alive；`/health/ready` 只检查 plugin bridge freshness，且 ready=false 仍为 HTTP 200。CallLogging 仅有 method/path/status，无耗时；`logback.xml` 只有 console appender，无文件轮转。没有 REST/WS/SSE 数量、Hikari、队列、Provider 或 restart 指标。
- **影响**：DB 已离线、池已堵塞或 AI/队列已饱和时，健康探测仍可能看起来正常；事故后无持久证据，无法区分后端、数据库、插件或隧道。
- **复现条件**：任何部分退化但进程仍活着。
- **根因置信度**：不是性能根因，但确定阻碍诊断和自动恢复。
- **修复方向**：保持现有公网 health 响应兼容，新增 28658 loopback supervisor status；记录 component 状态、连接池、队列、active streams、请求延迟和最近错误；由启动台持久化 stdout/stderr 并轮转。
- **验证方式**：DB/插件/Provider 分别离线，internal status 精确 degraded；公网旧响应仍可被现有消费者解析；诊断包不含敏感值。

### I-07 当前关闭流程不能保证优雅排空

- **状态**：已确认。
- **证据**：shutdown hook 只取消 maintenance scope 并关闭 datasource；没有先停止接收、排空 public/bridge engine、关闭 bridge event scope、取消 AI client 或等待活跃请求。两个 Ktor engine 的生命周期也没有统一 owner。
- **影响**：关闭时请求可能仍访问已关闭数据源；SSE/WS 被硬断；启动台若直接 kill，会放大 App 同步重连波峰和 exchange 收尾风险。
- **复现条件**：人工停止、进程退出、自动重启或系统关机。
- **根因置信度**：对日常慢为低；对重启可靠性为高。
- **修复方向**：单一 runtime owner；进入 draining、停止新请求、调用 engine graceful stop、等待最多 15 秒、取消 scopes、关闭 HTTP client 和 datasource；仅超时后强制终止。
- **验证方式**：活跃 REST/WS/SSE 下停止；不接受新业务写，已开始交易不重复，15 秒内退出或被 supervisor 明确强杀；旧 App 自动重连恢复。

### I-08 当前 App 的固定重连会在恢复时形成同步波峰

- **状态**：已确认，属于邻接检查。
- **证据**：`ChatRepository.scheduleReconnect()` 固定 `delay(1800)`，无指数退避、jitter 或网络可用性门控；`onOpen` 随即同步历史、presence/directory、follows，并触发 socket-connected 回调。
- **影响**：后端、隧道或数据库恢复时，20–40 个客户端可能在相近时间完成 WS 鉴权和多次 REST/DB 请求；自动重启策略若过激会加剧抖动。
- **复现条件**：批量掉线或后端重启。
- **根因置信度**：较可能解释恢复阶段连接慢，但无法解释没有掉线事件的持续退化。
- **修复方向**：后端先按当前行为承载 40/100 客户端并限制重启频率；未来 App 以 1.8 秒为起点加入指数退避、全抖动和网络恢复触发。不得要求当前 App 更新才能部署后端硬化。
- **验证方式**：40 个旧客户端同时断开/恢复，握手 p95 <1.5 秒、p99 <3 秒、错误率 <0.5%。

### I-09 生产运行包携带可直接清库的入口

- **状态**：已确认。
- **证据**：生产 ZIP 包含 `reset-database.bat`；主 `ApplicationKt` 在正常启动入口中只要环境变量 `DEUTERIUM_RESET_DATABASE=true` 就执行清表。重置覆盖账号、会话、钱包记录、转账、聊天、事件和在线状态。
- **影响**：误操作或继承了错误环境变量会造成灾难性业务数据删除；普通 `start-backend.bat` 未显式清除此变量。
- **复现条件**：本地运行脚本并输入 RESET，或启动进程继承 reset 环境变量。
- **根因置信度**：与连接慢无关；交付风险确定。
- **修复方向**：生产包移除 reset 工具；破坏性工具拆成独立 test-only artifact，要求数据库名 allowlist、二次确认和备份证明；正常主入口不解释 destructive env flag。启动台明确不提供重置。
- **验证方式**：生产 ZIP 不含 reset；设置该环境变量启动正式后端也不得删数据；测试工具只能对临时数据库工作。

### I-10 AI 管理 token 在 URL、表单和重定向间传播

- **状态**：已确认。
- **证据**：`/admin/ai?token=...` 接受 query token，页面在每个 form 中嵌入 token，并在 redirect URL 再带回。bridge 端口当前绑定 127.0.0.1，降低但不消除浏览器历史、日志、截图和本机软件泄露风险。
- **影响**：管理员 token 更容易被浏览器和诊断材料持久化。
- **复现条件**：使用当前管理页面。
- **根因置信度**：与连接慢无关。
- **修复方向**：可直接回收 pony 的 HttpOnly、SameSite=Lax cookie 登录方案，同时补 constant-time compare、短期过期和退出；若管理面未来仍仅 loopback，cookie 不设 Secure 是明确限制，不得把它暴露公网。
- **验证方式**：URL、HTML、redirect 和 access log 均不出现 token；伪造 Origin/Host 请求被拒绝。

## 10. Suggestion

### S-01 限制剩余无界状态和内容

- **状态/证据**：已确认。`aiPurchaseLocks`、公共聊天限频 key 等 map 缺少淘汰；AI reply `StringBuilder` 没有独立最终体积上限；REST 也没有统一 request body 上限。
- **影响**：键或内容基数长期增长会缓慢占用内存。当前 20–40 DAU 下优先级低于无界 plugin queue。
- **复现条件**：持续制造新 user/plan/rate-limit key，或 Provider 不遵守 `max_tokens` 并持续输出。
- **根因置信度**：解释当前连接症状为低。
- **修复方向**：锁改为固定 stripe 或安全 TTL cache；AI 增加最终 chars/bytes cap；REST 增加高于当前合法 payload 的统一体积上限，并保留现有字段级错误语义。
- **验证方式**：高基数/超长输出实验后 map 和内存有界；当前 APK 所有合法请求不触发新上限。

### S-02 维护清理改为可观测批处理

- **状态/证据**：已确认。维护任务启动后立即执行，之后每 6 小时对 5 张表执行无 LIMIT delete。
- **影响**：数据量大时可能形成长事务、锁等待和周期性 IO 峰值；当前已有索引且规模较小，不能据此认定已在线触发。
- **复现条件**：过期 session/chat/event 大量积累，单次 delete 扫描或删除很多行。
- **根因置信度**：待验证。
- **修复方向**：先记录每表删除数、耗时和 lock wait；超过阈值后按主键/时间窗口分批并在批间 yield。
- **验证方式**：用生产量级合成过期数据运行清理，普通 API p99 不出现超过目标的停顿，事务时间和每批行数受控。

### S-03 收紧迁移和数据库权限

- **状态/证据**：已确认。Flyway 全局 `ignoreMigrationPatterns("versioned:missing")`；启动前还会用 runtime credential 执行 `CREATE DATABASE IF NOT EXISTS`。
- **影响**：缺失 migration 可能被静默忽略；运行账户需要超出日常业务的建库权限，扩大凭据泄露影响。
- **复现条件**：数据库 history 中存在制品缺少的 versioned migration，或 runtime credential 被滥用。
- **根因置信度**：与连接症状无直接关系。
- **修复方向**：恢复完整 migration 或明确限定允许缺失版本；由单独 migration identity 建库/迁移，runtime identity 只保留业务最小权限。
- **验证方式**：故意删除一个非允许 migration 时部署必须失败；runtime user 执行 CREATE/DROP DATABASE 必须被拒绝，但旧 App 业务读写继续通过。

### S-04 框架大版本升级单独立项

- **状态/证据**：已确认目标依赖为 Ktor 2.3.12、Kotlin 1.9.24、Exposed 0.53.0、Flyway 10.17.0；Ktor 官方 release 文档已有 3.x 系列。
- **影响**：长期停留旧栈增加安全补丁和维护成本；直接跨大版本又会扩大当前稳定性修复的回归面。
- **复现条件**：需要新安全修复、JDK 支持或框架修复而旧分支不再提供时。
- **根因置信度**：不能仅凭版本差认定为当前性能根因。
- **修复方向**：先完成 Netty/Logback 等可隔离安全补丁，再单独迁移 Ktor 3.x；不要利用本次硬化顺手重写路由或数据层。
- **验证方式**：大版本候选必须与旧版本跑同一 compatibility/load suite，并单独生成迁移和回滚报告。参考：[Ktor releases](https://ktor.io/docs/releases.html)。

## 11. 优化路线

### P0：阻断安全与全局资源风险

1. 建立当前 APK 的 29 个网络面契约测试和 golden fixtures。
2. 生产专用包保留已填好的配置，但移除 reset 工具并确保日志、UI、诊断不回显值。
3. 补 Netty 至安全 4.1.x，Logback 至至少 1.5.37；复扫 Jackson/Protobuf 等 transitive advisory 的实际可达性。
4. WebSocket frame cap 64 KiB；OneBot 和公共 WS 设有界并发/速率。
5. 为 AI streams、Hikari、插件事件队列和 WS 增加最小指标；先观测再定容量。
6. 修复 plugin session generation 竞态。
7. 增加便携包 smoke test：复制文件夹后只双击 `launch-console.bat` 即可完成预检、兼容迁移和启动。

预期收益：消除已知公网补丁缺口和无界大帧；让下一次“连接很慢”能在不重启的情况下定位到线程、DB、桥或隧道。兼容风险主要来自依赖补丁和限额阈值，必须由旧 APK 契约及 40/100 并发测试阻断。

### P1：取消、背压、数据库恢复与优雅生命周期

1. AI Provider 改为真正可取消的异步流，增加 active/queue/output 上限。
2. 插件事件补可恢复的有界背压语义；钱包事件必须 ack/幂等/重放。
3. Hikari 显式 timeout/lifetime/keepalive 和 pool metrics；做 DB 短暂不可用恢复测试。
4. 增加 28658 loopback runtime status 和 authenticated graceful shutdown。
5. 实现 supervisor console、日志轮转、诊断包和有限自动恢复。

预期收益：断流、DB 抖动和插件洪峰不再占满全局资源；启动台能安全停止/恢复自己启动的进程。

### P2：基础设施与长期演进

1. Ktor 3.x/Kotlin 2.x 独立迁移。
2. App WebSocket 指数退避+jitter，但必须继续兼容当前固定 1.8 秒重连客户端。
3. HTTPS/WSS 仅在用户未来另行要求时立项，不属于本轮完成条件。
4. 按实际数据扩展慢查询、batch cleanup、容量告警和趋势图。

## 12. 建议回收 `pony-ultra` 的改动

| 改动 | 建议 | 原因 |
| --- | --- | --- |
| `maxFrameSize = 64 * 1024L` | 直接回收 | 与当前 App 最大消息体有大幅余量，兼容风险低 |
| `clientRequestId`、`playerRef` 长度校验 | 直接回收 | 阻止异常大幂等 key/引用进入 DB；需用当前 APK fixtures 验证正常值 |
| admin token 从 URL/form 移到 HttpOnly cookie | 直接回收并补测试 | 降低本地泄露风险，不涉及 App API |
| 路由拆分 | 仅在行为 golden tests 后回收 | 主要改善维护性，不应与 P0 运行时行为修改同批 |
| plugin event channel 容量 1024 | 不可原样直接回收 | 当前 `trySend` 失败只日志，会静默丢钱包/聊天/在线事件；应先补指标和可恢复语义 |
| 整个 pony JAR | 禁止直接替换 | 与目标有大量类/资源差异，无法视为小补丁 |

## 13. 验收状态

| 验收项 | 当前状态 |
| --- | --- |
| 高优先级问题有代码/二进制/配置/权威漏洞库证据 | 通过 |
| 目标 JAR 与源码 checkpoint 可验证映射 | 通过 |
| 基线后端 35 tests、Android 16 tests | 通过 |
| 候选后端 45 tests、Android 18 tests | 通过 |
| 生产只读公开轻量探测 | 通过，但样本窗口有限 |
| 生产 JVM/DB/隧道内部观测 | 未通过：SSH 在 key exchange 前关闭 |
| 40 并发 p95/p99 和错误率 | 未执行，发布阻断 |
| 100 并发无 5 秒全局停顿 | 未执行，发布阻断 |
| 6 小时资源回落与内存增长 <20% | 未执行，发布阻断 |
| 当前 App 契约自动回归门禁 | 已实现并通过：27 REST method/path、envelope、WS/SSE 邻接行为 |
| 零配置便携包、随包 JRE、manifest、SBOM、中文/空格路径 smoke | 通过；尚未连接生产数据库启动业务进程 |

## 14. 结论

目标后端的 SSE 收尾修复和历史 classpath 修复均真实存在，已有测试也覆盖了多个关键 AI 流异常；问题不应被简单归结为“之前修复无效”。当前最需要的是：先建立旧 APK 契约与便携包双击启动门禁，处理依赖补丁和 reset 入口，同时补齐能区分 AI 线程、数据库池、插件队列和隧道的观测；随后改造取消、背压和生命周期。随包生产配置和明文 HTTP/WS 是已记录的用户取舍，不再作为本轮未完成项。

在取得事故时 JVM/DB/隧道证据，以及完成 40/100 并发和 6 小时长稳前，任何“已经找到唯一根因”或“重启一定能恢复”的表述都不成立。

## 15. 实施回填（2026-07-15）

候选实现位于隔离工作树分支 `codex/backend-hardening-console`，没有覆盖当前 `main` 的未跟踪工程，也没有连接、停止、重启或迁移生产服务。已完成：

- Netty 全族对齐 `4.1.135.Final`，Logback 对齐 `1.5.38`；
- 公共和桥 WebSocket 帧上限 64 KiB，OneBot 帧改为顺序反压；
- AI Provider 请求改为可取消的异步请求，阻塞 body read 在取消时主动关闭，并设置默认 16 条全局活跃流上限；专门实验确认取消后 Provider 流在 2 秒窗口内关闭；
- Hikari 显式设置 10 秒借连接、3 秒校验、idle/maxLifetime/keepalive，并向内部状态页暴露 active/idle/pending/max；
- 修复 plugin bridge 旧 session 清理新 generation pending request 的竞态，增加 generation、queue depth 和状态快照；
- 移除正式主入口的清库环境变量和便携包中的 reset 脚本；
- 新增只在 `28658` loopback 注册的聚合状态与临时 token 优雅关闭接口；
- 新增 `28659` 本地 Web 启动台、owned-process 控制、external 只读模式、2/10/30 秒退避、3 次/10 分钟熔断、日志轮转和脱敏诊断包；
- 生成约 79 MB 的完整 Windows 文件夹，含 67 个 JAR、随包 JRE 17、预填生产配置、265 项 SHA-256 manifest、`PACKAGE-INFO.json` 和 CycloneDX 1.5 SBOM（66 个组件）；Protobuf 已固定到修复版本 3.25.5。

仍未关闭或未验收：

- plugin 主动事件仍需跨插件协议的 ack/幂等/重放后才能安全改成有界队列；当前只增加深度告警，不能靠静默丢弃换取“有界”；
- AI admin query token、统一 REST body limit、剩余高基数 map 和维护批处理尚未处理；
- 没有生产数据库同主版本、隧道和 fake plugin 完整环境，因此 DB pause/恢复、40/100 并发、6 小时长稳、实机旧 APK 全功能和灰度回滚仍是上线前门禁；
- 本机黑盒只以 `--no-autostart` 验证了随包 JRE、Web UI、安全响应头、诊断、中文/空格路径和端口释放，没有使用随包生产配置连接数据库。

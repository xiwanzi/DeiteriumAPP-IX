# DeuteriumAPP 后端硬化与本地启动台实施计划

## 1. 状态

- 计划状态：Implemented candidate / production validation pending
- 基线：`DeuteriumAPP-backend-only-1.0.4-ai-sse-fix1-20260616`
- 依据：`docs/backend-review-1.0.4-ai-sse-fix1.md`、ADR 0005、ADR 0006、ADR 0007
- 当前阶段：候选代码和便携目录已实现；生产数据库/隧道接入、负载长稳、实机灰度与回滚仍未执行

## 2. 目标

在不破坏现有 APP（当前 1.0.4 APK）使用的前提下：

1. 消除已知依赖漏洞、破坏性 reset 入口、无界资源和桥重连竞态；
2. 让 AI SSE、聊天 WebSocket、数据库和插件桥在超时、取消、断线和恢复时可控；
3. 建立能够解释“为什么连接慢”的指标、日志和诊断证据；
4. 实现 ADR 0007 的 Windows 本地 Web supervisor console，并交付复制后只需双击的完整生产文件夹；
5. 自动使用随包配置连接现有数据库，完成安全增量迁移，无需用户安装依赖、编辑配置或执行命令；
6. 用当前 APK 的契约、40/100 客户端负载和 6 小时长稳测试阻断不兼容发布。

## 3. 不做什么

- 不在本计划中重做 Android UI、账号、钱包、聊天或 AI 产品规则；
- 不把后端拆成微服务，不引入 Redis/Kafka；
- 不把 App 改为直连数据库或插件；
- 不在启动台提供配置编辑、手工选择 migration、数据库 reset/repair/clean、备份恢复或任意命令；
- 不把 supervisor 管理面暴露到公网；
- 不把 Ktor 3.x 大版本迁移和第一批 P0 补丁放进同一发布；
- 不用 `pony-ultra` 整包替换生产 JAR。

## 4. 兼容性发布门禁

### 4.1 冻结制品

兼容基线固定为：

- APK SHA-256：`E8ED1789808D8FF07F8C1B187CBF747F53D81C3288B5B3D802C774B71C8F778C`
- 后端 JAR SHA-256：`239BE52CEBC3CB4D5F75C0BC8409E3EF42773E30034999F3B9234A997410951C`
- 源码 checkpoint：`20f080d`

每个候选后端必须记录源码 commit、JAR hash、依赖 lock/SBOM 和 migration hash。无法映射源码的候选禁止发布。

### 4.2 冻结契约

建立 `backend-api` 下的 compatibility test suite，输入来自当前 App 源码和 APK 行为，覆盖：

- 27 个 Retrofit REST method/path/query/body；
- 成功 envelope `requestId/data/page`；
- 失败 envelope `requestId/error.code/error.message/error.details/error.retryAfterSeconds`；
- 401/403/404/409/422/429/503 等当前使用状态；
- Bearer token、登录、logout 和旧 session；
- `/api/v1/chat/ws` 鉴权、query 和双向 frame types；
- `/api/v1/ai/chat/stream` 的 `meta/status/sources/delta/done/error`、heartbeat comment、提前 EOF 和取消；
- 钱包转账/AI 购买的 `clientRequestId` 幂等；
- 当前 App 的 12/20 秒普通请求和 45 秒 SSE 读超时。

每个接口保存 canonical request、response 和 error golden fixture。新增字段必须验证 Gson 旧客户端可忽略；删除、重命名、改类型、改既有状态码或 terminal 事件均直接失败。

### 4.3 当前 App 实机门禁

自动测试通过后，仍必须用原 APK 在至少 Android 8/主流当前版本上验收：

- 注册、登录、旧 session 恢复、改密；
- 余额、刷新、收款人搜索、转账、流水；
- 聊天历史、聊天 WS、关注、mention、在线状态；
- AI 历史、SSE、取消/切页、重试、购买；
- 后端 restart 后固定 1.8 秒重连能够恢复。

“后端修改不要求 App 更新”是发布条件，不是事后说明。

### 4.4 数据库兼容

- 迁移只做 expand：新增表/列/索引先允许 null/default，旧 binary 可继续运行；
- 不在同一发布删除/重命名旧列；
- 写新读旧经过至少一个兼容版本后再收缩；
- 回滚只切旧 JAR，不回滚已经成功的 expand migration；
- migration 在生产副本和与生产同主版本的临时数据库验证。

### 4.5 网络兼容

- 28657 现有路径和 HTTP/WS 行为保持；
- 28658 只 additive 新增 `/supervisor/v1/*`；
- 私有环境继续使用当前明文 HTTP/WS，不新增证书或 TLS 配置前置步骤；
- 不做 redirect，不改变当前 App 固化的 URL。

### 4.6 零配置交付门禁

正式候选必须产出一个专用生产文件夹，其中已经包含随包 JRE、supervisor、backend、migration 和当前生产 `application.conf`。在一台未设置项目环境变量、未安装 Java/Node/Python、路径包含空格和非 ASCII 字符的 Windows 验收环境中：

1. 复制完整文件夹；
2. 双击 `launch-console.bat`；
3. 本地页面自动打开并显示 `preflight -> migration -> backend start -> health`；
4. 若数据库 schema 已是最新版本，validate 后直接启动；若落后，只自动执行 checksum 固定的 expand migration；
5. 不输入密码、token、数据库地址、JVM 参数或 migration 命令；
6. 当前 APK 无需更新即可正常使用。

任一步需要用户现场补配置，或使用系统 Java/PowerShell Web 服务，均判定交付失败。

## 5. 工作流总览

| 阶段 | 交付 | 主要风险 | 发布条件 |
| --- | --- | --- | --- |
| 0 | 兼容性 harness、SBOM、负载 harness | golden 不完整 | 当前基线自己对自己全绿 |
| 1 | P0 依赖/边界补丁与便携包规则 | 依赖二进制兼容 | 旧 APK + 双击启动 smoke 全绿 |
| 2 | 可观测性和 DB 恢复 | 指标本身引入开销 | 负载开销可接受、故障可定位 |
| 3 | AI 取消、桥背压和竞态修复 | 状态/资金语义 | 断流、重放、幂等测试全绿 |
| 4 | 后端 runtime owner 和 loopback API | 停止时数据一致性 | drain/force/回滚测试全绿 |
| 5 | supervisor console | 误杀、CSRF、日志泄密 | ADR 0007 全部场景通过 |
| 6 | 完整长稳、安全和兼容验收 | 环境差异 | 40/100/6h 指标通过 |
| 7 | 灰度、切换和回滚 | 隧道/重连波峰 | 回滚演练通过后上线 |

## 6. 阶段 0（Phase 0）：建立可重复证据和兼容 harness

### 6.1 后端契约测试

新增独立测试组，例如：

```text
backend-api/src/test/kotlin/.../compat/v104/
  RestContractV104Test.kt
  ChatWebSocketContractV104Test.kt
  AiSseContractV104Test.kt
  ErrorEnvelopeV104Test.kt
```

要求：

- fixture 来自当前 App model，而不是根据新后端反向生成；
- 每个 REST method 至少一条成功和一条关键失败；
- WebSocket frame JSON 做双向字段精确检查；
- SSE 检查事件顺序、terminal exactly once、EOF 和取消；
- 在 baseline `20f080d` 上先运行，证明 harness 不会误报。

### 6.2 依赖和制品

- 生成 CycloneDX 或等价 SBOM；
- 锁定直接/传递依赖版本；
- 对每个 JAR 记录 Maven coordinate 和 SHA-256；
- CI 调用 OSV/GitHub Advisory Database 扫描；
- unexpected-secret scanner 同时扫描 Git diff、运行目录和最终 ZIP；专用生产包只允许实际值出现在固定的 `backend/config/application.conf`，不得复制进脚本、日志或 README；
- ZIP 内容 manifest 记录路径、大小、hash，以及该配置是用户明确允许的生产专用文件。

### 6.3 负载 harness

使用 JVM/Kotlin 或 k6 等可版本化工具，但运行时不得成为生产依赖。场景需要能模拟：

- 40 个旧 App 行为的稳定客户端；
- 100 个同时恢复的 WebSocket 客户端；
- REST 混合：登录态 me/余额/聊天/目录/版本检查；
- WS `app.state`、聊天、断线重连；
- SSE 正常、首 token 慢、idle stall、提前 EOF、客户端取消；
- fake plugin、fake OneBot、fake AI Provider；
- DB pause/kill connection/恢复。

输出统一 JSON/CSV：p50/p95/p99、错误率、CPU、RSS、线程、句柄、Hikari、WS、AI、event queue、TCP 和恢复时间。

### 6.4 完成标准

- baseline 对 compatibility suite 全绿；
- baseline 负载结果被保存为对照，但不要求它先满足新目标；
- 所有工具只使用测试凭据和隔离数据库；
- 当前工作树不需要切换分支才能运行 harness。

## 7. 阶段 1：P0 依赖、便携交付和输入边界

建议拆成彼此可回滚的小 PR，不合并发布风险。

### 7.1 专用生产配置与便携包

后端/交付：

- 专用生产包保留当前有效的 `backend/config/application.conf`，不要求服务器端再次填写；
- 保持现有 DB 凭据、session/verification pepper 和 plugin token，避免当前 App 会话、验证码与插件桥因无必要轮换而失效；
- 所有运行路径从 `%~dp0` 相对解析，不依赖解压目录名或盘符；
- `launch-console.bat` 只使用随包 JRE，不读取 `JAVA_HOME`，也不回退系统 Java；
- 配置、JRE、manifest 或 migration 缺失时明确判定包损坏，不启动业务端口，不进入配置向导；
- 生产包移除 `reset-database.bat`；
- 主 `ApplicationKt` 不再解释 destructive reset 环境变量；
- 配置值不进入命令行、UI、普通日志或诊断包；配置来源摘要只显示 key/source；
- 专用包生成时校验四项必需生产配置非空，并记录配置文件 hash，但不在 manifest 中展开值。

便携 smoke：将候选复制到新的长路径、含空格路径和含中文路径，分别双击启动；除真正的 DB/端口故障外，均不得要求输入或修改文件。

### 7.2 Netty 安全补丁

- 保持 Ktor 2.3.12，先用 dependency constraint/BOM 将所有 Netty module 对齐到至少 4.1.135.Final；
- 禁止不同 Netty module 混用版本；
- 运行 Ktor 自身启动、HTTP/1、HTTP/2（若启用）、WebSocket 和 SSE 测试；
- 用 request smuggling/畸形 chunk corpus 验证；
- 若存在 binary incompatibility，回滚该 PR，另开 Ktor 迁移阶段。

### 7.3 其他依赖

- Logback 升级到至少 1.5.37；当前时间优先验证 1.5.38；
- Protobuf 升至修复 CVE-2024-7254 的兼容版本；
- Jackson advisory 按 Flyway 实际调用面判断，但版本必须通过 SBOM 门禁；
- 删除 Windows 不使用且可安全排除的 native epoll/kqueue JAR，减少扫描噪音和制品面；
- 每次只升级一个依赖族并保存性能对比。

参考：[Logback 1.5.37/1.5.38 release notes](https://logback.qos.ch/news.html)、[Netty releases](https://github.com/netty/netty/releases)。

### 7.4 输入和并发边界

直接回收并验证：

- WebSocket `maxFrameSize=64 KiB`；
- `clientRequestId`、`playerRef` 长度/格式校验；
- AI admin token cookie 登录。

新增：

- 每用户 App WS 最大连接数，保留合理多设备余量；
- 每连接 chat/app.state 入站速率限制；
- OneBot 全局有界 worker/semaphore；
- AI input/output/request body 上限；
- 超限沿用现有 JSON error envelope 或明确 WS close code。

### 7.5 完成标准

- 旧 APK compatibility suite 和实机全绿；
- SBOM 中 Netty/Logback P0 advisory 清零或有书面 VEX；
- 64 KiB 以下正常，超限不会分配无界内存；
- 专用生产包只在固定 `application.conf` 中包含预期实际配置，其他文件 unexpected-secret scan 为零，且无 reset 工具；
- rollback 到旧 JAR 仍可读取数据库。

## 8. 阶段 2：可观测性和数据库恢复

### 8.1 指标

后端内存中维护低基数 metrics snapshot，由 28658 internal status 读取：

- REST request count/in-flight/duration/status；
- App WS total/foreground/slow send/close reason；
- SSE active/queued/cancelled/timeout/first token/total duration；
- Hikari active/idle/pending/max/acquire time/timeout；
- plugin connected/last seen/pending requests/event queue depth/oldest age；
- OneBot connected/in-flight/rejected；
- maintenance duration/deleted rows；
- JVM uptime/heap/thread/GC。

`requestId` 应回写响应 header 并保留当前 JSON 字段。日志只记录 endpoint template，不把 token、query secret 或业务正文写入。

### 8.2 Hikari 配置

新增可配置且有范围校验的：

- `connectionTimeoutMillis`；
- `validationTimeoutMillis`；
- `maxLifetimeMillis`；
- `keepaliveTimeMillis`；
- `poolName`；
- `registerMbeans` 或等价内部 metrics。

数值不能拍脑袋固定。先读取生产 DB `wait_timeout`、连接上限和隧道/NAT 行为，再保证 `keepalive < maxLifetime < infrastructure timeout`。池大小用实测 DB 并发和查询耗时确定，而不是简单随客户端数线性增加。

### 8.3 health 兼容

- 保留 `/health/live` 和 `/health/ready` 当前响应结构，避免未知消费者破坏；
- internal status 提供 DB、bridge、public connector、queue 和 AI 分组件状态；
- console 使用 internal status，不用公网 ready 替代完整诊断；
- 后续如需把 ready=false 改为 503，必须先审计隧道/监控消费者并另立兼容变更。

### 8.4 故障实验

- DB 停 10/30/60 秒后恢复；
- kill idle/in-use connection；
- DNS/TCP 黑洞和连接拒绝；
- pool 全占用；
- 慢查询和 lock wait；
- plugin 断线但 DB 正常。

验收：新请求在明确 timeout 内失败，不能挂 30 秒无反馈；DB 恢复后池和错误率回落，不需要重启 JVM。

## 9. 阶段 3：AI 取消、插件背压和桥竞态

### 9.1 AI Provider client

实现方向：

- 一个长生命周期、可关闭的异步 HTTP client；
- response body 使用 suspending channel/flow 读取；
- Ktor call 取消立即取消上游 request、关闭 body 和 parser；
- 不为每个 stream 创建 executor；
- first-token、idle、total deadline 仍独立；
- heartbeat 不能延长 provider deadline；
- active stream 和 waiting queue 有配置上限；
- reply bytes/chars 有上限；
- exchange 最终状态在 `NonCancellable` 短事务中 exactly once 收尾；
- 保持当前 SSE event、错误码、迟到完成保护和 quota 语义。

测试：

- 客户端在 meta 前、delta 后、done 前断开；
- provider 卡在 headers、首 token、hidden activity、正文中间；
- 上游取消后 1 秒内连接/线程/active metric 回落；
- 100 次取消没有 exchange 留在 pending/streaming；
- 同 clientMessageId 重放仍返回既有结果或既有错误。

### 9.2 plugin connection generation

- session state 改为 immutable snapshot：generation/session/lastSeen；
- pending value 包含 generation；
- old session finally 只清理自己的 pending；
- reply 必须 generation 匹配；
- 双 session 交错 property/stress test 至少 1,000 次。

### 9.3 plugin event 背压

分两步：

1. 只加 depth/age/latency/reject 指标，保留现状以取得真实容量；
2. 插件协议增加 event id、ack、幂等和有限重放后，再切有界 channel。

事件分类：

- 钱包/余额：不得静默丢弃，必须 ack + idempotency + replay；
- 聊天/在线：允许按明确规则 coalesce 或丢旧快照，但必须计数和告警；
- replyTo：继续直接优先完成，不经过慢事件队列。

如果暂时不能改插件协议，不得直接采用 pony 的 `Channel(1024)+trySend` 作为最终方案。

## 10. 阶段 4：后端 runtime owner 和优雅关闭

### 10.1 Runtime 结构

引入单一 `BackendRuntime`，持有：

- public Ktor engine；
- bridge Ktor engine；
- Hikari datasource；
- maintenance scope；
- plugin event scope/channel；
- AI HTTP client/stream registry；
- runtime metrics；
- draining/stop state。

`start()`、`stop(grace)`、`close()` 必须幂等。shutdown hook 和 supervisor endpoint 只调用 runtime，不直接各自关闭资源。

### 10.2 Loopback API

实现 ADR 0007：

- `GET /supervisor/v1/status`；
- `POST /supervisor/v1/shutdown`；
- 只注册在 127.0.0.1:28658；
- ephemeral token 来自 supervisor 子进程环境；
- loopback、Host、token 三重校验；
- 不返回配置值和业务内容。

### 10.3 Drain 语义

- status 标记 draining；
- 新 mutating REST 返回兼容的 503 error envelope；
- 新 WS/SSE handshake 不再接受；
- 已进入 DB transaction 的钱包/购买完成或回滚；
- 允许短期完成普通读取；
- 15 秒到期关闭 WS/SSE、取消 scopes、stop engine、close datasource；
- 记录未排空数量供 console 显示。

### 10.4 完成标准

- 两个 engine 不再由不同 wait/hook 隐式拥有；
- 活跃请求停止实验无重复交易/额度；
- datasource 最后关闭；
- 无 supervisor 时传统 Ctrl+C 也走相同 graceful path；
- 旧 App restart 后自动恢复。

## 11. 阶段 5：Supervisor console

### 11.1 工程

建议新增独立 Gradle module：

```text
backend-supervisor/
  src/main/kotlin/...
  src/main/resources/web/index.html
  src/test/kotlin/...
```

仅依赖 Kotlin/JVM、JDK 17 和必要 JSON 库；HTTP server 使用 `jdk.httpserver`。静态页面随 JAR 内嵌，不从公网加载资源。

### 11.2 模块切片

1. `SupervisorStateMachine`：串行 action、固定状态转换；
2. `PackageLayout`：从 BAT 所在目录解析固定 JRE、配置、JAR、日志和 manifest；
3. `StartupPipeline`：串行执行 preflight、migration、backend start、health wait；
4. `MigrationRunner`：使用固定 classpath/配置执行 Flyway validate/migrate，禁止任意参数与 repair/clean/reset；
5. `BackendProcessOwner`：ProcessBuilder、PID/start time/launch id、exit watcher；
6. `BackendProbe`：public health + internal status；
7. `RestartPolicy`：2/10/30 秒和 10 分钟 3 次熔断；
8. `RotatingLogStore`：双 stream drain、32 MiB/14 天/256 MiB；
9. `DiagnosticsService`：受控内容、二次脱敏、SHA-256；
10. `ConsoleHttpServer`：Host/Origin/CORS/content-type/header 防护；
11. `WebAssets`：启动阶段、状态、日志、actions UI；
12. `launch-console.bat`：只定位并启动随包 supervisor，第二实例时打开已有页面。

### 11.3 BAT 规则

- 路径全部使用 `%~dp0` 和双引号；
- 不启用不需要的 delayed expansion，避免路径中的 `!` 被破坏；
- 不接受用户拼接 command；
- 可接受的 supervisor JVM options 写在固定配置/白名单中；
- 固定使用 `%~dp0backend\jre\bin\java.exe`；缺失即报告包损坏，不读取 `PATH`/`JAVA_HOME`；
- supervisor 监听后立即打开 `http://127.0.0.1:28659`，页面实时显示自动预检、迁移和后端启动进度；
- 第二次启动检测现有 console，打开页面后退出，不创建第二实例。

### 11.4 API/UI

实现 ADR 固定接口：

- `GET /api/v1/status`
- `GET /api/v1/logs?cursor=&limit=`
- `POST /api/v1/actions/start`
- `POST /api/v1/actions/stop`
- `POST /api/v1/actions/restart`
- `POST /api/v1/diagnostics`

所有 mutation body 第一版固定 `{}` 或固定枚举，不接受 path/command/env/JVM options。

首次双击不需要点击 start。supervisor 没有发现 external backend 时自动触发一次与 `POST /api/v1/actions/start` 相同的启动流程；人工 stop 后在同一 supervisor 生命周期内不自动拉起。

### 11.5 单元和集成测试

- state transition/property tests；
- PID reuse/start-time mismatch；
- external port owner；
- concurrent action serialization；
- auto restart time-window/fuse；
- stdout/stderr 大量输出不死锁；
- rotation/retention/total cap；
- graceful 15 秒与 force fallback；
- missing config/JRE/log permission；
- package manifest mismatch、migration checksum mismatch、migration failed；
- 已是最新 schema 和需要增量 migration 两条自动启动路径；
- 文件夹移动、长路径、空格、中文和 `!` 路径；
- 无 Java/Node/Python/项目环境变量的干净 Windows 环境；
- fake backend internal status；
- diagnostics redaction corpus。

## 12. 阶段 6：完整验证

### 12.1 环境

- 先只读确认生产 MySQL/MariaDB 主版本；
- 使用同主版本、临时独立数据目录和测试凭据；
- fake plugin/OneBot/AI Provider 全部 loopback；
- 不复制生产业务数据；如需分布特征，只生成匿名合成数据；
- 候选后端使用备用端口，不占当前生产 28657/28658。

### 12.2 负载阶段

1. 15 分钟预热；
2. 60 分钟混合负载；
3. 6 小时低频长稳；
4. 长稳中插入 DB 30 秒不可用、plugin 重连、Provider stall 和 100 客户端恢复波峰；
5. 客户端退出后继续观察至少 15 分钟资源回落。

### 12.3 指标门槛

- 40 个并发客户端的连接/握手 p95 <1.5 秒；
- p99 <3 秒；
- 错误率 <0.5%；
- 100 并发突发没有 >5 秒全局停顿；
- 6 小时后线程、Hikari、TCP 在客户端退出后回落；
- 预热后内存不持续单调增长 >20%；
- AI 取消后 1 秒内 active stream 回落；
- event queue 有界且故障恢复后归零；
- DB 恢复不需要 JVM restart；
- 所有旧 APK 契约仍通过。

### 12.4 安全测试

- 非 loopback 访问 28659/28658 supervisor path；
- Host header DNS rebinding；
- forged/missing Origin；
- cross-site form 和 CORS preflight；
- path traversal/encoded traversal；
- JVM/command/env 参数注入；
- oversized JSON/frame；
- request smuggling/畸形 chunk；
- diagnostic/log secret corpus；
- production ZIP unexpected-secret scan（允许固定 `application.conf`）和 SBOM scan。

## 13. 阶段 7：灰度、切换和回滚

### 13.1 上线前

- 取得生产进程/DB/隧道只读快照；
- 备份数据库并验证可恢复，不由 console 执行；
- 候选在备用端口运行并完成 loopback smoke；
- 使用旧 APK 对候选执行全功能验收；
- 确认 PassNAT/反向代理能否无中断切换；
- 旧 JAR、旧配置结构和旧隧道配置随时可回切。

### 13.2 无影响切换原则

如果当前隧道支持稳定前端到两个 backend：

1. 候选加入但不接业务；
2. health/contract 通过；
3. 小流量或只读流量灰度；
4. 全量切换；
5. 旧 backend 保持可回滚但不接新写；
6. 观察至少一个高峰窗口。

如果当前拓扑只能切单端口，则无法承诺严格零断线。必须先增加稳定的本地反向代理/双 backend 切换层，或由用户批准短维护窗口。不能把 App 的 1.8 秒自动重连等同于“绝对无影响”。

### 13.3 自动回滚触发

- 旧 APK 契约失败；
- 5 分钟窗口错误率 >0.5%；
- handshake p95 >1.5 秒或 p99 >3 秒；
- 钱包/购买出现重复、unknown 增长或 bridge event 丢失；
- Hikari pending 持续增长；
- active AI streams/threads 取消后不回落；
- supervisor 错误控制 external process。

回滚切旧 JAR/旧 backend，不执行 destructive DB rollback；保留候选日志、thread dump 和指标用于分析。

## 14. 交付结构

未来最终交付是一个可直接复制的生产专用文件夹：

```text
DeuteriumAPP-backend-<version>/
  launch-console.bat
  backend/
    lib/
    jre/                 # 固定 Windows x64 JRE 17
    config/
      application.conf  # 已填好的当前生产配置
      application.example.conf
  supervisor/
    lib/
    web/                 # 如未内嵌，只允许固定只读资源
  logs/                  # 首次运行创建
  diagnostics/           # 首次运行创建
  MANIFEST.sha256
  PACKAGE-INFO.json      # 版本、目标 OS/JRE、schema/migration 摘要
  SBOM.cdx.json
  README.md              # 只需说明“复制后双击 launch-console.bat”
```

允许实际生产值只存在于 `backend/config/application.conf`。不得包含 reset 工具、业务数据库、真实日志、heap dump，也不得在 BAT、README、manifest 或其他文件中复制 token/password/pepper/key。

## 15. 工作分解和依赖

| 工作包 | 依赖 | 可独立验收 |
| --- | --- | --- |
| Compatibility v1.0.4 suite | 无 | 是，必须最先完成 |
| SBOM/unexpected-secret/package manifest | 无 | 是 |
| Netty/Logback patch | compatibility suite | 是 |
| Frame/input limits | compatibility suite | 是 |
| Metrics snapshot | compatibility suite | 是 |
| Hikari recovery | metrics | 是 |
| Plugin generation race | bridge stress harness | 是 |
| AI cancellable client | SSE harness + metrics | 是 |
| Plugin durable backpressure | plugin protocol + fake plugin | 否，跨后端/插件 |
| BackendRuntime/graceful stop | metrics + AI/bridge close interface | 是 |
| Internal supervisor API | BackendRuntime | 是 |
| Supervisor state/process/log | ADR 0007 | 是，使用 fake backend |
| Supervisor UI/diagnostics/BAT | supervisor core | 是 |
| 6h soak/security | 所有候选功能 | 最终发布门禁 |

## 16. 需要用户/环境确认的事项

以下不是当前文档工作的阻塞，但在实现或上线前必须取得：

- 生产数据库产品和主版本、连接上限、`wait_timeout`；
- Windows Server 版本、CPU/内存、当前 Java 进程启动方式；
- PassNAT/反向代理的实际拓扑、健康检查和切换能力；
- 当前 App 安装量中该 Debug APK 的占比；
- 是否允许为严格无中断切换新增稳定本地反向代理层。

## 17. Definition of Done

只有同时满足以下条件，后端硬化和启动台才算完成：

- 当前 1.0.4 APK 不更新也能完成全部现有功能；
- 27 REST + WS + SSE 契约自动门禁全绿；
- Netty、reset、frame 和其他未接受的 P0 风险关闭；随包配置与明文 HTTP/WS 作为用户接受项保留；
- AI 取消、DB 恢复、bridge reconnect/backpressure 有可重复实验；
- 40/100 并发和 6 小时长稳达到指标；
- supervisor 只控制 owned process，external 模式不误杀；
- graceful stop、重启熔断、日志轮转和诊断脱敏通过；
- 在无系统 Java/Node/Python、无项目环境变量、含空格和非 ASCII 路径的 Windows 环境中，复制文件夹并只双击一次即可自动 validate/migrate、启动和打开页面；
- 新包包含已填好的唯一生产配置、随包 JRE、hash manifest 和 SBOM，其他文件无配置值副本；
- 灰度与回滚演练完成；
- Review 中所有 Critical/Important 已关闭，或有用户明确接受并记录的残余风险。

## 18. 当前实施检查点（2026-07-15）

已完成的工作包：Compatibility v1.0.4 suite、Netty/Logback patch、64 KiB frame limit、Hikari 显式恢复参数与池快照、plugin generation race、AI cancellable client/全局流上限、BackendRuntime/graceful stop、internal supervisor API、supervisor state/process/log/UI/diagnostics/BAT、随包 JRE、生产配置复制、manifest、package info 和 CycloneDX SBOM。

自动验证结果：后端 45/45、Android 18/18；最终目录 265 条 manifest 全部匹配；中文与空格路径黑盒启动台 smoke 通过。候选 JAR SHA-256 为 `253A1CBA8343E4B7569648895216CF504BD5DA58575AF6EE143E9C6852483D57`。

未完成且继续阻断生产替换：plugin durable backpressure、同主版本临时数据库故障恢复、40/100 并发、6 小时 soak、旧 APK 实机完整业务、生产只读快照、灰度和回滚演练。当前候选可以交给用户复制到服务器做受控验收，但不能把本机 `--no-autostart` smoke 表述为生产业务启动已通过。

# 后端与 Minecraft 插件维护手册

本文档记录当前阶段 `backend-api/` 与 `minecraft-plugin/` 的实现状态、运行方式、数据流和维护注意事项。它面向后续接手 DeuteriumAPP 后端、服务器插件和生产交付的开发者。

本文档不替代接口合同。HTTP 和 WebSocket 的字段级合同仍以 `docs/contracts/app-backend-api-v1.md` 和 `docs/contracts/openapi-v1.yaml` 为准。

生产交付包的目录结构、配置、打包步骤和启动验收，以 `docs/production-delivery.md` 为准。后续交付不要再临时拼装运行包。

## 1. 当前架构总览

当前实现由两个独立工程组成：

- `backend-api/`：Kotlin/JVM + Ktor + Exposed + Flyway + MySQL/MariaDB，Java 17。
- `minecraft-plugin/`：Java + Bukkit/Spigot API + Vault Economy + Java-WebSocket，面向 Mohist 1.20.1，Java 17。

运行拓扑：

```text
Android App
  |
  | HTTPS / WSS
  v
Backend public server :28657
  |
  | local WebSocket, Bearer bridge token
  v
Backend bridge server :28658  <--- Minecraft plugin 主动连接
  |
  | Vault Economy
  v
XConomy / 服务器经济系统
```

端口约定：

- `28657`：App HTTP API 与 App 聊天 WebSocket，允许通过 HTTPS 内网穿透暴露公网。
- `28658`：后端插件桥本地端口，只允许 Minecraft 插件在同机访问，不应暴露公网。

生产 Base URL：

```text
https://deuterium.s.odn.cc/api/v1
```

生产聊天 WebSocket：

```text
wss://deuterium.s.odn.cc/api/v1/chat/ws
```

## 2. 工程结构

### 后端

关键入口：

- `Application.kt`：进程启动、Ktor server、迁移模式、清库模式、公共端口和桥端口启动。
- `ApplicationServices.kt`：仓库、桥、聊天 hub 等依赖装配。
- `routes/Routes.kt`：HTTP API、App WebSocket、插件桥 WebSocket 路由。
- `bridge/WebSocketPluginBridge.kt`：插件桥连接管理、请求响应关联、插件事件处理。
- `repository/Repositories.kt`：账号、会话、验证码、玩家引用、钱包、聊天仓库。
- `db/Tables.kt`：Exposed 表定义。
- `db/migration/V1__initial_schema.sql`：Flyway 初始 schema。
- `config/AppConfig.kt`：本地配置文件和环境变量加载。

### 插件

关键入口：

- `DeuteriumBridgePlugin.java`：插件生命周期、Vault 检查、事件监听、余额监控、玩家解析。
- `BridgeClient.java`：插件到后端的 WebSocket 客户端、桥消息分发、后端请求处理。
- `src/main/resources/plugin.yml`：Bukkit 插件描述。
- `src/main/resources/config.yml`：默认插件配置模板。
- `src/main/local-resources/config.yml`：交付脚本临时生成的本地配置，已在 `.gitignore` 中忽略。

## 3. 配置与密钥

后端运行配置来自：

1. `DEUTERIUM_CONFIG` 环境变量指向的 properties 文件。
2. 默认 `config/application.conf`。
3. 同名环境变量覆盖配置项，例如 `DEUTERIUM_DATABASE_JDBCURL`。

关键配置项：

```properties
public.host=0.0.0.0
public.port=28657

bridge.host=127.0.0.1
bridge.port=28658

database.jdbcUrl=jdbc:mysql://127.0.0.1:3306/deuterium_app?useSSL=false&serverTimezone=UTC&allowPublicKeyRetrieval=true
database.user=...
database.password=...

security.sessionTokenPepper=...
security.verificationPepper=...
security.sessionDays=30

pluginBridge.token=...
pluginBridge.requestTimeoutMillis=10000
pluginBridge.heartbeatIntervalMillis=30000
pluginBridge.staleAfterMillis=90000

chat.websocketPingIntervalMillis=15000
chat.websocketTimeoutMillis=35000
```

安全规则：

- 不要提交真实 `application.conf`。
- 不要提交数据库密码、bridge token、pepper 或任何生产密钥。
- 交付脚本会生成本地配置和插件内置配置，生成结果在 `delivery/` 和 `src/main/local-resources/`，均不应提交。

插件配置：

```yaml
backend:
  ws-url: "ws://127.0.0.1:28658/bridge/plugin/ws"
  token: "CHANGE_ME"
  reconnect-seconds: 5

chat:
  broadcast-format: "§x§b§1§f§7§f§f%player% §7: §f%message%"

wallet-monitor:
  enabled: true
  online-poll-seconds: 15
  offline-poll-seconds: 300
  max-offline-players-per-cycle: 100
```

当前插件运行时优先读取 jar 内置 `config.yml` 中的 bridge 地址和 token。这样即使服务器里已有旧的 `plugins/DeuteriumBridge/config.yml`，也不会因为旧 token 导致桥接反复断开。

## 4. 构建与交付

后端测试：

```powershell
cd backend-api
.\gradlew.bat test
```

插件编译：

```powershell
cd minecraft-plugin
.\gradlew.bat build
```

生成生产交付目录：

```powershell
.\scripts\build-production-delivery.ps1 -DbUser root -DbPassword "实际密码"
```

生成结果：

```text
delivery/DeuteriumAPP-production/
  backend/
    start-backend.bat
    migrate-db.bat
    reset-database.bat
    config/application.conf
    jre/
    lib/
  minecraft-plugin/
    deuterium-minecraft-plugin-0.1.0.jar
  README.md
```

交付包压缩文件：

```text
delivery/DeuteriumAPP-production.zip
```

部署顺序：

1. 解压交付包到 Windows 服务器。
2. 确认 MySQL 已启动。
3. 运行 `backend/migrate-db.bat`。
4. 运行 `backend/start-backend.bat`，窗口需要保持打开。
5. 将 `minecraft-plugin/deuterium-minecraft-plugin-0.1.0.jar` 放入 Minecraft 服务器 `plugins/`。
6. 重启 Minecraft 服务器，后端日志应出现 `Plugin bridge connected`。

测试期清库：

```text
backend/reset-database.bat
```

该脚本会要求输入 `RESET`。它保留表结构，清空账号、会话、验证码、钱包、转账、聊天、事件和在线状态等业务数据。

## 5. 后端启动模式

`Application.kt` 当前支持三种模式：

- 正常模式：迁移数据库后启动公共 server 和插件桥 server。
- 迁移模式：设置 `DEUTERIUM_MIGRATE_ONLY=true`，执行 Flyway 后退出。
- 清库模式：设置 `DEUTERIUM_RESET_DATABASE=true`，先执行 Flyway，再清空业务表后退出。

注意：

- 清库模式会删除账号，所以测试后需要重新注册。
- 清库模式不会 drop 表，不会删除 Flyway schema history。
- 后端启动时公共 server 与桥 server 是两个 Ktor Netty 实例。

## 6. HTTP API 概览

所有 App API 位于 `/api/v1` 下，统一返回：

```json
{
  "requestId": "req_...",
  "data": {},
  "error": null
}
```

健康检查不在 `/api/v1` 下：

- `GET /health/live`：进程存活。
- `GET /health/ready`：插件桥可用性。

账号：

- `POST /account/registration-code`
- `POST /account/register`
- `POST /account/login`
- `POST /account/password-reset-code`
- `POST /account/password-reset`
- `POST /account/logout`
- `GET /account/me`

钱包：

- `GET /wallet/balance`
- `POST /wallet/balance/refresh`
- `GET /wallet/recipients/search`
- `POST /wallet/transfers`
- `GET /wallet/transfers/{transferId}`
- `GET /wallet/records`

聊天：

- `GET /chat/messages`
- `GET /chat/presence`
- `GET /chat/online-players`
- `GET /chat/ws`

## 7. 鉴权与安全

App 会话：

- 使用 Bearer token。
- token 为 opaque 随机值。
- 数据库只保存 `SHA-256(token + pepper)` 结果。
- 默认有效期 30 天。
- 当前没有 refresh token。

密码：

- 使用 Argon2id。
- 明文密码不入库。

验证码：

- 后端生成 6 位数字验证码。
- 验证码明文只通过插件私聊发给在线玩家。
- 数据库保存验证码 hash。
- 验证码默认 10 分钟有效。
- 同一用途和玩家存在冷却与尝试次数限制。

登录失败限制：

- 以“标准化账号输入 + 来源 IP”作为失败 key。
- 连续 5 次失败后锁定 15 分钟。

插件桥：

- 插件主动连接后端。
- 使用 `Authorization: Bearer <pluginBridge.token>`。
- 鉴权失败时后端关闭 WebSocket，关闭原因包含 `invalid bridge token`。
- 默认请求超时 10 秒。
- 后端每 30 秒发送心跳。
- 90 秒无有效响应视为不可用。

## 8. 插件桥协议

所有插件桥消息使用统一 envelope：

```json
{
  "type": "wallet.balance.request",
  "messageId": "bridge_msg_...",
  "replyTo": "bridge_msg_...",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {}
}
```

约定：

- 后端请求插件时带 `messageId`。
- 插件响应时带 `replyTo`，值为原请求的 `messageId`。
- 插件主动上报事件时不带 `replyTo`。
- 后端用 `messageId -> CompletableDeferred` 做请求响应关联。

后端请求插件：

- `verification.deliver.request`
- `player.resolve.request`
- `wallet.balance.request`
- `wallet.transfer.request`
- `chat.appMessage.request`
- `presence.list.request`
- `bridge.ping`

插件响应后端：

- `verification.deliver.result`
- `player.resolve.result`
- `wallet.balance.result`
- `wallet.transfer.result`
- `chat.appMessage.result`
- `presence.list.result`
- `bridge.pong`

插件主动上报后端：

- `chat.serverMessage.event`
- `server.event`
- `presence.snapshot.event`
- `wallet.pay.event`
- `wallet.balanceChange.event`

## 9. 数据模型概览

核心表：

- `app_users`：App 账号，绑定服务器 UUID、当前游戏 ID、QQ 和密码 hash。
- `sessions`：登录 token hash、过期时间、撤销时间。
- `verification_requests`：注册和改密验证码上下文。
- `login_failures`：登录失败计数和锁定时间。
- `player_refs`：前端使用的 opaque 玩家引用。
- `wallet_balances`：最近余额缓存。
- `transfers`：App 发起的转账请求与状态。
- `wallet_records`：App 展示的统一钱包流水。
- `chat_messages`：公共聊天历史。
- `server_events`：服务器事件提示。
- `presence_snapshots`：最近在线玩家快照。

重要约束：

- 服务器经济权威来源是 Vault/XConomy，不是后端数据库。
- `wallet_balances` 是缓存。
- `wallet_records` 是 App 展示流水，不是经济系统账本的唯一权威。
- `playerRef` 是前端 opaque 引用，前端不得提交可信 UUID。

## 10. 账号数据流

注册验证码：

1. App 提交 gameId、QQ、密码到 `/account/registration-code`。
2. 后端检查 QQ 是否重复和验证码冷却。
3. 后端通过插件桥要求插件向在线玩家私聊验证码。
4. 插件确认玩家在线，返回服务器 UUID 和当前 gameId。
5. 后端保存验证码 hash 和 verification token hash。

提交注册：

1. App 提交 verification token、验证码、密码。
2. 后端校验验证码、过期、次数和唯一性。
3. 后端写入 `app_users`。
4. 后端签发 session token。

改密：

1. App 请求 `/account/password-reset-code`。
2. 后端通过游戏内私聊投递验证码。
3. App 提交新密码和验证码。
4. 后端更新密码 hash，并撤销该用户全部 session。

## 11. 钱包与流水数据流

余额读取：

1. App 请求 `/wallet/balance` 返回缓存。
2. App 请求 `/wallet/balance/refresh` 时，后端通过插件桥读取 Vault Economy 余额。
3. 插件返回余额后，后端更新 `wallet_balances`。

App 转账：

1. App 搜索收款玩家，后端通过 QQ 或插件桥解析玩家。
2. App 提交 `/wallet/transfers`。
3. 后端用 `clientRequestId` 和请求 fingerprint 做业务幂等。
4. 后端创建 `transfers`，再通过插件桥请求 Vault withdraw/deposit。
5. 插件执行转账。
6. 后端更新 transfer 状态，并写付款方 `wallet_records` 支出流水。

服务器 `/pay`：

1. 插件监听玩家执行 `/pay <玩家> <金额>`。
2. 命令执行后延迟 1 tick 比较付款方和收款方余额变化。
3. 只有扣款和入账都与金额匹配时，插件上报 `wallet.pay.event`。
4. 后端对已注册 App 的付款方写 `expense`，对已注册 App 的收款方写 `income`。
5. 同一 `payEventId` 的同一方向使用确定性 recordId，避免重复写入。

通用经济变化：

1. 插件定期扫描玩家 Vault 余额。
2. 首次看到玩家余额时只建立基线，不写流水。
3. 后续余额增加则上报 `wallet.balanceChange.event` + `income`。
4. 后续余额减少则上报 `wallet.balanceChange.event` + `expense`。
5. 后端写入 `wallet_records`，备注默认 `服务器经济变动`。

限制：

- 余额扫描只能发现“变多/变少”，不能自动知道来源。
- 部署前已经发生的交易不会自动补回。
- 如果多个经济变动发生在同一轮扫描之间，可能被合并为一条未知来源变化。
- `/pay` 若存在手续费、税率或非精确到账，可能不会被识别为 `/pay` 成功流水，而会由余额扫描记为普通经济变化。

## 12. 聊天数据流

服务器到 App：

1. 插件监听 `AsyncPlayerChatEvent`。
2. 插件上报 `chat.serverMessage.event`。
3. 后端写入 `chat_messages`。
4. 后端通过 `AppChatHub` 广播给 `/api/v1/chat/ws` 连接。

App 到服务器：

1. App 连接 `/api/v1/chat/ws`，使用 Bearer token 鉴权。
2. App 发送 `chat.send`。
3. 后端从当前 session 获取发送者服务器 UUID 和 gameId。
4. 后端通过插件桥发送 `chat.appMessage.request`。
5. 插件验证发送者存在后，用 Bukkit 广播普通聊天样式。
6. 后端写入 `chat_messages` 并广播给 App。

当前 App 转发服务器聊天样式：

```text
§x§b§1§f§7§f§f%player% §7: §f%message%
```

服务器事件：

- 插件监听 `death`、`server_say`、`join`、`quit`。
- 事件上报为 `server.event`。
- App 是否展示服务器事件由 App 聊天连接参数和 UI 设置控制。

在线状态：

- 插件在连接、玩家进服、玩家退服时上报 `presence.snapshot.event`。
- App 请求在线列表时，后端也可通过 `presence.list.request` 拉取。

## 13. 插件实现注意事项

Vault/XConomy：

- 插件启动时检查 Vault。
- 如果没有 Economy provider，钱包能力禁用，但聊天和部分事件仍可运行。
- 当前经济读写都走 Vault Economy，不直接访问 XConomy 数据库。

玩家解析：

- 在线玩家优先使用 Bukkit 在线玩家 UUID。
- 离线玩家必须满足 `hasPlayedBefore` 或 Vault/XConomy 存在账户。
- 避免因为任意文本创建不存在玩家。

App 转账：

- 扣款使用 `withdrawPlayer`。
- 入账使用 `depositPlayer`。
- 若扣款后入账失败，插件尝试回滚。
- 回滚失败时返回 `unknown`，后端不会自动重试资产操作。

余额监控：

- 在线玩家默认每 15 秒扫描。
- 离线玩家默认每 300 秒分批扫描，每轮最多 100 人。
- 扫描依赖内存基线，插件重启后第一轮只建立基线。
- App 转账和 `/pay` 成功后会更新基线，减少重复流水。

App 聊天：

- 插件使用固定模板，不允许旧配置覆盖。
- 插件不会额外标记 App 来源。
- 插件使用 `Bukkit.broadcastMessage` 广播。

## 14. 生产排障

### 后端窗口一闪而过

用命令行进入 `backend/` 后运行脚本：

```bat
migrate-db.bat
start-backend.bat
```

常见原因：

- MySQL 未启动。
- MySQL 密码错误。
- 端口 `28657` 或 `28658` 被占用。
- `config/application.conf` 缺失或含 `CHANGE_ME`。

### 插件反复 Connected / Backend bridge closed

后端日志如果只有 `/bridge/plugin/ws 101`，但没有 `Plugin bridge connected`，通常是 token 不一致。

检查：

- 交付包里的后端和插件 jar 是否来自同一次构建。
- 是否替换了服务器 `plugins/` 中的旧 jar。
- 插件日志是否显示 `invalid bridge token`。

当前 jar 内置 token 优先级高于旧配置，一般只要后端和插件来自同一交付包即可。

### `/health/ready` 为不可用

含义是后端进程正常，但插件桥不可用。

检查：

- Minecraft 服务器是否启动。
- 插件是否加载成功。
- Vault 和 XConomy 是否存在。
- 后端桥端口 `28658` 是否监听在 `127.0.0.1`。

### 余额刷新失败

常见原因：

- Vault 缺失。
- XConomy 未通过 Vault 暴露 Economy provider。
- 玩家没有经济账户。
- 插件桥断开。

### 流水没有出现

区分情况：

- App 转账流水：应由后端在 `/wallet/transfers` 后写入。
- `/pay` 流水：需要插件确认付款方和收款方余额变化都等于金额。
- 其他经济流水：首次扫描只建立基线，后续变化才会记录。
- 未注册 App 的玩家不会看到自己的流水，只有已注册账号能通过 `/wallet/records` 查看。

## 15. 测试建议

### 后端性能维护状态

2026-05-03 后端已补充兼容性性能修复，部署时只需要替换后端并执行数据库迁移，不要求更新 Android App 或 Minecraft 插件。当前修复包括：

- 余额缓存写入改为数据库原子 upsert。
- App presence 首次写入和已有记录更新都改为单条 upsert。
- 验证码尝试次数使用数据库原子自增。
- 登录失败次数使用数据库原子 upsert 递增。
- 钱包流水和转账创建不再写后读回。
- 幂等钱包流水改为直接插入并捕获重复主键。
- 后端每 6 小时清理过期 session、验证码、登录失败记录、聊天记录和服务器事件。
- 新增 `V4__maintenance_indexes.sql` 支撑维护清理路径。

生产部署仍按 `docs/production-delivery.md` 执行，先运行 `backend\migrate-db.bat`，再运行 `backend\start-backend.bat`。

自动测试：

```powershell
cd backend-api
.\gradlew.bat test

cd minecraft-plugin
.\gradlew.bat build
```

手动验收：

1. 运行 `migrate-db.bat`。
2. 运行 `start-backend.bat`。
3. 启动 Minecraft 服务器并确认 `Plugin bridge connected`。
4. 请求 `GET /health/live`。
5. 请求 `GET /health/ready`。
6. 用 App 或接口工具完成注册、登录、余额刷新。
7. 在游戏内执行 `/pay`，确认双方 App 流水。
8. 做一次箱子商店或其他经济变动，等待余额扫描后确认未知来源流水。
9. App 发送聊天，确认服务器内普通聊天样式。
10. 运行 `reset-database.bat` 验证测试清理流程。

## 16. 后续扩展建议

箱子商店精确来源：

- 如果确认具体 ChestShop/QuickShop 插件，可接入其交易事件。
- 插件应在事件里拿到买家、卖家、金额、物品和店铺信息。
- 后端仍复用 `wallet.balanceChange.event` 或新增更具体事件，但 App 接口不变。

钱包流水字段：

- 当前 `WalletRecord` 没有 `source` 字段。
- 为保持前端兼容，来源信息暂时放在 `note` 和 `otherPlayer` 中。
- 若未来要筛选来源，可在 API v2 或兼容新增可选字段。

分页：

- 当前 `/wallet/records` 返回 `nextCursor = null`。
- 若流水增多，需要实现 cursor 分页。

聊天历史：

- 当前文档要求保留默认 30 天，但清理任务尚未实现。
- 后续可增加定时清理。

监控：

- 当前只有日志和健康检查。
- 后续可增加桥连接状态、余额扫描计数、上报失败计数等指标。

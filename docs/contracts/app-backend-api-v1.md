# DeuteriumAPP 统一接口规范 v1

## 1. 文档定位

本文档定义 DeuteriumAPP 第一阶段 Android App、后端 API 与 Minecraft 插件桥之间的统一接口契约，便于 Android 端与后端并行开发。

本文档遵循：

- `AGENTS.md`
- `CONTEXT.md`
- `docs/prd/account.md`
- `docs/prd/wallet-transfer.md`
- `docs/prd/chat.md`
- `docs/adr/0002-backend-plugin-bridge.md`
- `docs/adr/0003-account-identity-authority.md`
- `docs/adr/0004-backend-plugin-tech-stack.md`

配套 HTTP OpenAPI 文件为 `docs/contracts/openapi-v1.yaml`。

本文档覆盖：

- Android App 调用后端的公开 HTTP API。
- Android App 与后端之间的 Chat WebSocket 消息约定。
- 后端与 Minecraft 插件桥之间的内部 WebSocket 消息约定。
- 共享响应、错误码、身份、安全和幂等规则。

本文档不覆盖：

- 数据库表结构。
- 后端、Android 或插件代码实现。
- Wiki.js OIDC Provider；该契约见 `docs/contracts/oidc-provider-v1.md`。
- 生产密钥、数据库密码、插件桥密钥或 token。
- XConomy 具体 API 调用代码。
- Minecraft 插件命令、权限节点或事件监听代码。

## 2. 全局约定

### 2.1 公开 HTTP API

- Base path：`/api/v1`
- 数据格式：`application/json; charset=utf-8`
- 时间格式：UTC ISO-8601 字符串，例如 `2026-04-29T12:30:00Z`
- 金额格式：decimal string，例如 `"1820.50"`，不使用 JSON number 表达信用点金额
- 公开接口默认只面向 Android App，不面向 Minecraft 插件或第三方客户端
- 除注册、登录、验证码请求、改密验证码请求和检查更新外，接口都需要登录态

检查更新公开轻量接口：

`GET /api/v1/app/update-check?versionCode=5&versionName=1.0.3`

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "latest": true,
    "message": "当前已是最新版本",
    "latestVersionCode": 5,
    "latestVersionName": "1.0.3"
  }
}
```

规则：

- 不需要登录态。
- 后端只读取配置 `app.latestVersionCode` 和 `app.latestVersionName`，不访问数据库。
- 当 `versionCode >= latestVersionCode` 时返回 `latest=true` 和“当前已是最新版本”。
- 当 `versionCode < latestVersionCode` 时返回 `latest=false` 和“版本已过时，请更新最新版本”。

### 2.2 会话

第一版采用简单 opaque Bearer token。

登录或注册成功后，后端返回：

```json
{
  "requestId": "req_01H...",
  "data": {
    "token": "opaque-session-token",
    "user": {}
  }
}
```

Android 后续请求使用：

```http
Authorization: Bearer <token>
```

规则：

- token 对 Android 不透明，Android 不解析 token 内容。
- token 不在 URL 中传递。
- 退出登录使当前 token 失效。
- 密码重设成功后，该账号既有 token 全部失效。
- 第一版不定义 access token + refresh token 双 token 机制。

### 2.3 请求标识

客户端可以在 HTTP 请求头携带：

```http
X-Request-Id: req_client_...
```

后端响应必须返回 `requestId`。如果客户端未提供，后端生成一个。

`requestId` 用于日志关联和排障，不用于业务幂等。

转账提交必须单独使用 `clientRequestId` 作为业务幂等键。

### 2.4 统一成功响应

HTTP 成功响应统一使用：

```json
{
  "requestId": "req_01H...",
  "data": {}
}
```

列表响应可以增加 `page`：

```json
{
  "requestId": "req_01H...",
  "data": [],
  "page": {
    "nextCursor": "opaque-cursor-or-null"
  }
}
```

### 2.5 统一错误响应

错误响应统一使用：

```json
{
  "requestId": "req_01H...",
  "error": {
    "code": "PLAYER_NOT_ONLINE",
    "message": "玩家当前不在线，请先进入 Deuterium VIII 服务器。",
    "details": {},
    "retryAfterSeconds": 60
  }
}
```

规则：

- `code` 是前后端判断分支的稳定标识。
- `message` 是可直接展示或轻度改写的中文提示。
- `details` 只放非敏感调试信息，不放密码、验证码、token、数据库信息或插件桥密钥。
- `retryAfterSeconds` 只在冷却、限流或临时锁定类错误中返回。

### 2.6 身份与 opaque 引用

玩家身份权威来自服务器解析 UUID，但公开 App API 不要求 Android 读写 UUID。

规则：

- App 不得提交自称可信的 UUID。
- App 可以提交玩家输入标识，例如游戏内 ID、QQ 号或搜索关键词。
- 后端返回给 App 的收款方、在线玩家和基础资料引用使用 opaque `playerRef`。
- `playerRef` 只代表后端已经确认过的玩家引用，Android 不解析其内容。
- 任何钱包转账和聊天发送都以当前登录账号绑定身份为准，App 不允许指定付款方或聊天发送者。

### 2.7 验证码 opaque token

验证码流程使用 `verificationToken` 绑定用途、玩家身份上下文、冷却、过期和尝试次数。

规则：

- 验证码本身只通过 Minecraft 服务器私聊投递。
- 后端响应不得返回验证码明文。
- 注册验证码与密码重设验证码不能跨用途复用。
- 新验证码生成后，旧验证码立即失效。

### 2.8 错误码

通用错误码：

| code | HTTP | 含义 |
| --- | --- | --- |
| `INVALID_REQUEST` | 400 | 请求格式、字段或参数不合法 |
| `UNAUTHORIZED` | 401 | 未登录、token 缺失或 token 无效 |
| `FORBIDDEN` | 403 | 当前账号无权执行该操作 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `CONFLICT` | 409 | 资源状态冲突 |
| `RATE_LIMITED` | 429 | 请求过于频繁 |
| `SERVER_UNAVAILABLE` | 503 | 后端依赖不可用 |
| `PLUGIN_BRIDGE_UNAVAILABLE` | 503 | 插件桥不可用 |

账号错误码：

| code | HTTP | 含义 |
| --- | --- | --- |
| `PLAYER_NOT_ONLINE` | 409 | 玩家不在线，不能接收验证码或完成依赖在线状态的操作 |
| `PLAYER_NOT_FOUND` | 404 | 游戏内 ID 无法识别 |
| `PLAYER_IDENTITY_CONFLICT` | 409 | 游戏内 ID 与服务器身份解析冲突 |
| `QQ_ALREADY_USED` | 409 | QQ 号已被绑定 |
| `UUID_ALREADY_REGISTERED` | 409 | 服务器解析 UUID 已注册 |
| `VERIFICATION_COOLDOWN` | 429 | 验证码重发冷却中 |
| `VERIFICATION_INVALID` | 422 | 验证码错误或 token 无效 |
| `VERIFICATION_EXPIRED` | 422 | 验证码过期 |
| `VERIFICATION_ATTEMPTS_EXCEEDED` | 422 | 验证码尝试次数耗尽 |
| `PASSWORD_INVALID` | 422 | 密码不满足 8-64 位 |
| `ACCOUNT_PASSWORD_INVALID` | 401 | 账号或密码错误 |
| `LOGIN_LOCKED` | 429 | 连续登录失败后临时限制 |

钱包转账错误码：

| code | HTTP | 含义 |
| --- | --- | --- |
| `AMOUNT_INVALID` | 422 | 金额为空、非数字、不大于 0 或超过两位小数 |
| `RECIPIENT_NOT_FOUND` | 404 | 未找到收款玩家 |
| `RECIPIENT_IDENTITY_UNCONFIRMED` | 409 | 收款玩家身份无法确认 |
| `BALANCE_INSUFFICIENT` | 409 | 余额不足 |
| `TRANSFER_DUPLICATE` | 409 | `clientRequestId` 重复且请求内容不一致 |
| `TRANSFER_FAILED` | 409 | 转账明确失败 |
| `TRANSFER_RESULT_UNKNOWN` | 202 | 转账结果未知，需要稍后查余额和流水 |

聊天错误码：

| code | HTTP/WebSocket | 含义 |
| --- | --- | --- |
| `CHAT_CONNECTION_UNAVAILABLE` | 503/ws | 实时连接不可用 |
| `CHAT_MESSAGE_EMPTY` | 422/ws | 消息为空或只包含空白字符 |
| `CHAT_MESSAGE_TOO_LONG` | 422/ws | 消息超过 256 字符 |
| `CHAT_SEND_FAILED` | 409/ws | 聊天发送失败 |
| `SENDER_IDENTITY_UNCONFIRMED` | 409/ws | 发送者身份不可确认 |

AI 错误码：

| code | HTTP/SSE | 含义 |
| --- | --- | --- |
| `AI_DISABLED` | 503/sse | AI 功能未启用或模型密钥未配置 |
| `AI_MESSAGE_EMPTY` | 422/sse | AI 消息为空 |
| `AI_MESSAGE_TOO_LONG` | 422/sse | AI 消息超过长度限制 |
| `AI_QUOTA_EXCEEDED` | 429/sse | AI 请求额度已用完 |
| `AI_PROVIDER_UNAVAILABLE` | 503/sse | 模型供应商不可用 |
| `AI_PROVIDER_TIMEOUT` | 503/sse | 模型供应商响应超时 |
| `AI_PROMPT_RISK_BLOCKED` | 422/sse | 请求触发提示词攻击或安全风险拦截 |
| `AI_PLAN_NOT_FOUND` | 404 | AI 套餐不存在或已下架 |
| `AI_PURCHASE_DUPLICATE` | 409 | 购买幂等键重复且内容不一致 |
| `AI_PURCHASE_FAILED` | 409 | AI 套餐购买失败 |
| `AI_PURCHASE_RESULT_UNKNOWN` | 202 | AI 套餐购买结果未知，需要后台审计 |

## 3. 共享类型

### 3.1 `UserProfile`

```json
{
  "userId": "usr_opaque",
  "playerRef": "player_opaque",
  "gameId": "StarDust",
  "qq": "24681012",
  "identityStatus": "bound"
}
```

字段规则：

- `userId`：DeuteriumAPP 后端内部用户引用，Android 不解析。
- `playerRef`：当前绑定 Minecraft 玩家引用，Android 不解析。
- `gameId`：当前展示用游戏内 ID。
- `qq`：账号绑定 QQ 号。
- `identityStatus`：第一版固定为 `bound`；如果后续支持异常状态，再扩展。

### 3.2 `PlayerSummary`

```json
{
  "playerRef": "player_opaque",
  "gameId": "EchoMint",
  "qq": "10002",
  "online": true,
  "registered": true,
  "source": "qq"
}
```

`source` 可选值：

- `game_id`
- `qq`
- `search`
- `online_list`

### 3.3 `ResolvedPlayerRef`

```json
{
  "playerRef": "player_opaque",
  "gameId": "Lantern",
  "qq": "10003",
  "online": true,
  "registered": true,
  "source": "game_id",
  "confirmedAt": "2026-04-29T12:30:00Z",
  "expiresAt": "2026-04-29T12:40:00Z"
}
```

字段规则：

- `ResolvedPlayerRef` 表示后端已经确认到服务器玩家身份的引用。
- `playerRef` 是 Android 后续提交转账时唯一允许使用的收款方引用。
- `expiresAt` 可以为空；如果后端设置过期时间，Android 过期后应重新搜索或确认收款方。
- Android 不得从 `playerRef` 推断 UUID，也不得自行构造 `playerRef`。

### 3.4 `WalletBalance`

```json
{
  "currency": "CREDIT",
  "amount": "1820.50",
  "fresh": true,
  "refreshedAt": "2026-04-29T12:30:00Z"
}
```

### 3.5 `WalletRecord`

```json
{
  "recordId": "wrec_opaque",
  "direction": "expense",
  "otherPlayer": {},
  "amount": "68.50",
  "currency": "CREDIT",
  "status": "success",
  "note": "矿镐维修",
  "occurredAt": "2026-04-29T12:30:00Z"
}
```

`direction`：

- `income`
- `expense`

`status`：

- `processing`
- `success`
- `failed`
- `unknown`

### 3.6 `Transfer`

```json
{
  "transferId": "tr_opaque",
  "clientRequestId": "android-generated-id",
  "recipient": {},
  "amount": "68.50",
  "currency": "CREDIT",
  "note": "矿镐维修",
  "status": "success",
  "createdAt": "2026-04-29T12:30:00Z",
  "updatedAt": "2026-04-29T12:30:03Z"
}
```

`status`：

- `processing`
- `success`
- `failed`
- `unknown`

### 3.7 `ChatMessage`

```json
{
  "messageId": "msg_opaque",
  "sender": {},
  "content": "晚上一起去下界吗？",
  "kind": "public_chat",
  "sentAt": "2026-04-29T12:30:00Z"
}
```

### 3.8 `ServerEvent`

```json
{
  "eventId": "evt_opaque",
  "eventType": "death",
  "content": "Lantern 摔得太狠了",
  "occurredAt": "2026-04-29T12:30:00Z"
}
```

### 3.9 `OnlinePlayer`

```json
{
  "playerRef": "player_opaque",
  "gameId": "Lantern",
  "qq": "10003",
  "registered": true,
  "onlineSince": "2026-04-29T12:00:00Z"
}
```

### 3.10 `AiQuota`

```json
{
  "used": 1,
  "limit": 20,
  "remaining": 19,
  "windowHours": 5,
  "resetsAt": "2026-06-15T12:00:00Z"
}
```

### 3.11 `AiPlan`

```json
{
  "planId": "plan_pro",
  "code": "pro",
  "name": "AI Pro",
  "description": "免费额度 2 倍的限时套餐",
  "price": "500.00",
  "currency": "CREDIT",
  "quotaPerWindow": 40,
  "windowHours": 5,
  "durationDays": 30,
  "modelTier": "flash",
  "active": true
}
```

### 3.12 `AiMessage`

```json
{
  "messageId": "aim_opaque",
  "conversationId": "aic_opaque",
  "role": "assistant",
  "content": "你好，我是 xxxAI。",
  "createdAt": "2026-06-15T12:00:00Z"
}
```

`role` 只允许：

- `user`
- `assistant`

## 4. Account HTTP API

### 4.1 请求注册验证码

`POST /api/v1/account/registration-code`

请求：

```json
{
  "gameId": "StarDust",
  "qq": "24681012"
}
```

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "verificationToken": "ver_opaque",
    "expiresAt": "2026-04-29T12:40:00Z",
    "resendAfterSeconds": 60
  }
}
```

规则：

- 后端通过插件桥解析 `gameId` 并确认玩家在线。
- 后端检查游戏 ID、QQ 格式和绑定占用；获取验证码不需要密码。旧客户端附带的 `password` 字段被忽略，密码在创建账号时校验。
- 后端生成 6 位数字验证码，通过服务器私聊发送。
- 响应不得返回验证码明文。

主要错误：

- `GAME_ID_INVALID`
- `QQ_INVALID`
- `ACCOUNT_ALREADY_EXISTS`
- `PLAYER_OFFLINE`
- `RATE_LIMITED`
- `PLUGIN_BRIDGE_UNAVAILABLE`

### 4.2 提交注册

`POST /api/v1/account/register`

请求：

```json
{
  "verificationToken": "ver_opaque",
  "code": "123456",
  "password": "password-8-to-64"
}
```

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "token": "opaque-session-token",
    "user": {}
  }
}
```

规则：

- `verificationToken` 已绑定游戏内 ID、QQ、用途和解析身份。
- 后端再次确认 UUID 未注册、QQ 未绑定。
- 注册成功后签发当前设备 token。

主要错误：

- `VERIFICATION_INVALID`
- `VERIFICATION_EXPIRED`
- `VERIFICATION_ATTEMPTS_EXCEEDED`
- `QQ_ALREADY_USED`
- `UUID_ALREADY_REGISTERED`
- `PASSWORD_INVALID`

### 4.3 登录

`POST /api/v1/account/login`

请求：

```json
{
  "account": "StarDust-or-24681012",
  "password": "password-8-to-64"
}
```

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "token": "opaque-session-token",
    "user": {}
  }
}
```

规则：

- `account` 是单一登录输入框内容，可以是游戏内 ID 或 QQ。
- 登录失败提示使用 `ACCOUNT_PASSWORD_INVALID`，避免泄露账号枚举信息。
- 连续 5 次失败后返回 `LOGIN_LOCKED`，`retryAfterSeconds` 为剩余限制时间。

### 4.4 请求密码重设验证码

`POST /api/v1/account/password-reset-code`

请求：

```json
{
  "account": "StarDust-or-24681012"
}
```

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "verificationToken": "ver_opaque",
    "expiresAt": "2026-04-29T12:40:00Z",
    "resendAfterSeconds": 60
  }
}
```

规则：

- 后端根据账号找到已绑定 Minecraft 身份。
- 玩家必须在线才能接收改密验证码。
- 密码重设验证码不能用于注册。

主要错误：

- `ACCOUNT_PASSWORD_INVALID`
- `PLAYER_NOT_ONLINE`
- `VERIFICATION_COOLDOWN`
- `PLUGIN_BRIDGE_UNAVAILABLE`

### 4.5 提交新密码

`POST /api/v1/account/password-reset`

请求：

```json
{
  "verificationToken": "ver_opaque",
  "code": "123456",
  "newPassword": "new-password-8-to-64"
}
```

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "passwordReset": true
  }
}
```

规则：

- 成功后该账号既有会话全部失效。
- Android 应引导玩家重新登录。

### 4.6 退出登录

`POST /api/v1/account/logout`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "loggedOut": true
  }
}
```

规则：

- 只退出当前 token。
- 不影响其他设备，除非后续另有会话管理设计。

### 4.7 获取当前用户资料

`GET /api/v1/account/me`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "user": {}
  }
}
```

## 5. Wallet HTTP API

### 5.1 获取当前余额

`GET /api/v1/wallet/balance`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "balance": {}
  }
}
```

规则：

- 返回后端最近一次已知余额。
- 如果从未成功刷新过，后端可以返回 `fresh: false` 与可理解状态，或返回业务错误。
- Android 不得把失败状态展示为余额 `0`。

### 5.2 刷新余额

`POST /api/v1/wallet/balance/refresh`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "balance": {}
  }
}
```

规则：

- 后端通过当前登录账号绑定身份请求插件桥读取 XConomy 余额。
- 插件桥不可用时直接失败，不返回伪成功余额。

### 5.3 搜索收款玩家

`GET /api/v1/wallet/recipients/search?query=Lantern&type=auto`

需要登录态。

参数：

- `query`：游戏内 ID、QQ 号或搜索关键词。
- `type`：`auto`、`game_id`、`qq`、`search`，默认 `auto`。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "candidates": []
  }
}
```

规则：

- QQ 搜索只命中已绑定 App 账号的玩家。
- 游戏内 ID 和搜索入口可以命中未注册 App 但服务器可确认的玩家。
- 后端返回 `playerRef` 后，Android 后续转账只能提交 `playerRef`，不能提交自称 UUID。

主要错误：

- `RECIPIENT_NOT_FOUND`
- `RECIPIENT_IDENTITY_UNCONFIRMED`
- `PLUGIN_BRIDGE_UNAVAILABLE`

### 5.4 提交转账

`POST /api/v1/wallet/transfers`

需要登录态。

请求：

```json
{
  "clientRequestId": "android-generated-unique-id",
  "recipientPlayerRef": "player_opaque",
  "amount": "68.50",
  "note": "矿镐维修"
}
```

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "transfer": {}
  }
}
```

规则：

- `clientRequestId` 是 Android 为一次用户确认提交生成的唯一值。
- 如果同一 `clientRequestId` 重复提交且内容一致，后端返回同一转账结果，不重复执行。
- 如果同一 `clientRequestId` 重复提交但内容不一致，返回 `TRANSFER_DUPLICATE`。
- `amount` 必须大于 0，最多两位小数。
- `note` 可选，最大 80 字符。
- 转账失败或结果未知时后端不自动重试。
- 插件桥不可用时不排队转账。

主要错误：

- `AMOUNT_INVALID`
- `RECIPIENT_IDENTITY_UNCONFIRMED`
- `BALANCE_INSUFFICIENT`
- `TRANSFER_DUPLICATE`
- `TRANSFER_FAILED`
- `TRANSFER_RESULT_UNKNOWN`
- `PLUGIN_BRIDGE_UNAVAILABLE`

### 5.5 查询转账结果

`GET /api/v1/wallet/transfers/{transferId}`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "transfer": {}
  }
}
```

规则：

- 用于转账结果页、结果未知后的查账和前端恢复。
- 后端只允许查询当前登录用户相关转账。

### 5.6 获取基础流水

`GET /api/v1/wallet/records?limit=20&cursor=opaque`

增量同步：

`GET /api/v1/wallet/records?limit=100&afterRecordId=wrec_opaque`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "records": []
  },
  "page": {
    "nextCursor": "opaque-or-null"
  }
}
```

规则：

- 第一版默认按时间倒序。
- 不传 `afterRecordId` 时保持旧客户端语义不变，返回最近流水。
- 传 `afterRecordId` 时返回该流水之后的新流水，按时间正序返回，供新客户端上线后补齐离线期间记录。
- 流水展示方向、对方玩家、金额、时间、状态和可选备注。
- 第一版不要求筛选、搜索、导出或复杂对账。

## 6. Chat HTTP API

### 6.1 获取最近公共聊天

`GET /api/v1/chat/messages?limit=100&before=opaque`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "messages": []
  },
  "page": {
    "nextCursor": "opaque-or-null"
  }
}
```

规则：

- `limit` 默认 100，最大 100。
- 默认只返回公共聊天，不包含服务器事件提示。
- 服务器事件提示是否展示由 Chat WebSocket 或后续历史接口扩展控制；第一版历史加载不默认混入事件。

### 6.2 获取在线概览

`GET /api/v1/chat/presence`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "onlineCount": 12,
    "available": true,
    "updatedAt": "2026-04-29T12:30:00Z"
  }
}
```

### 6.3 获取在线玩家列表

`GET /api/v1/chat/online-players`

需要登录态。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "players": []
  }
}
```

规则：

- 在线玩家列表表示 Deuterium VIII 服务器在线玩家，不是 App 在线用户。
- 每个玩家返回 `playerRef`，可作为转账候选入口，但转账仍需走 `wallet-transfer` 的二次确认。

### 6.4 获取玩家目录

`GET /api/v1/chat/player-directory`

需要登录态。该接口兼容新增，不改变 `/chat/online-players`。

响应：

```json
{
  "requestId": "req_01H...",
  "data": {
    "players": [
      {
        "playerRef": "player_ref_opaque",
        "gameId": "Lantern",
        "qq": "10000",
        "registered": true,
        "serverOnline": true,
        "appStatus": "online",
        "appConnected": true,
        "appForeground": true,
        "appLastSeenAt": "2026-04-30T12:30:00Z",
        "onlineSince": "2026-04-30T12:00:00Z",
        "followed": false,
        "self": false
      }
    ]
  }
}
```

规则：

- 排序优先为服务器当前在线玩家，其后为已注册 App 玩家。
- `appStatus` 只允许 `online`、`just_online`、`recent_online`、`recently_online`、`long_offline`。
- `appConnected` 表示目标用户当前存在 App WebSocket 连接；`appForeground` 表示这些连接中至少一个在前台。
- 当前登录用户也可以出现在列表中，`self=true`。

### 6.5 关心玩家

`GET /api/v1/chat/follows`

返回当前账号已关心玩家列表。

`POST /api/v1/chat/follows`

请求：

```json
{
  "playerRef": "player_ref_opaque"
}
```

`DELETE /api/v1/chat/follows/{playerRef}`

规则：

- 关心关系按当前账号和目标玩家服务器身份保存。
- 重复关心保持幂等。
- 可关心服务器在线但尚未注册 App 的玩家。

## 7. App Chat WebSocket

### 7.1 连接

路径：

```text
GET /api/v1/chat/ws
```

鉴权：

```http
Authorization: Bearer <token>
```

如果 Android WebSocket 客户端无法发送自定义 Header，可以使用一次性 WebSocket ticket 作为后续扩展；v1 默认使用 Header。

连接参数：

- `includeServerEvents=true|false`：是否接收服务器事件提示，默认 `false`。

连接保活：

- 后端对 App Chat WebSocket 启用标准 WebSocket ping/pong 探活。
- 默认配置为 `chat.websocketPingIntervalMillis=15000`、`chat.websocketTimeoutMillis=35000`。
- 该探活不新增业务消息类型，不改变 `chat.send`、`chat.message`、`presence.update` 等现有 payload。
- 旧 Android 客户端无需处理自定义心跳；底层 WebSocket 客户端响应标准 pong 即可保持兼容。

### 7.2 消息信封

App 与后端 WebSocket 消息统一格式：

```json
{
  "type": "chat.send",
  "requestId": "req_ws_opaque",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {}
}
```

规则：

- `type` 是消息类型。
- `requestId` 由发送方生成，用于请求响应关联；服务端推送事件可以省略。
- `sentAt` 使用 UTC。
- `payload` 按类型定义。

### 7.3 Client -> Server：发送公共聊天

`type = chat.send`

```json
{
  "type": "chat.send",
  "requestId": "req_ws_01",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "clientMessageId": "android-generated-message-id",
    "content": "晚上一起去下界吗？",
    "mentionedPlayerRefs": ["player_ref_opaque"]
  }
}
```

规则：

- `content` trim 后不能为空。
- `content` 最大 256 字符。
- 发送者身份来自当前 WebSocket 会话绑定账号。
- App 不得提交 sender、gameId 或 UUID。
- `mentionedPlayerRefs` 为可选字段；不传或空数组时按旧聊天发送逻辑处理。
- 只有用户从 @ 列表点选玩家时才应写入 `mentionedPlayerRefs`。手动输入 `@玩家名` 不触发通知。

### 7.4 Server -> Client：发送结果

`type = chat.send.result`

```json
{
  "type": "chat.send.result",
  "requestId": "req_ws_01",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "clientMessageId": "android-generated-message-id",
    "status": "accepted",
    "messageId": "msg_opaque"
  }
}
```

`status`：

- `accepted`
- `failed`

失败时返回：

```json
{
  "type": "chat.send.result",
  "requestId": "req_ws_01",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "clientMessageId": "android-generated-message-id",
    "status": "failed",
    "error": {
      "code": "PLUGIN_BRIDGE_UNAVAILABLE",
      "message": "服务器连接暂不可用，请稍后再试。"
    }
  }
}
```

### 7.5 Server -> Client：公共聊天消息

`type = chat.message`

```json
{
  "type": "chat.message",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "message": {}
  }
}
```

规则：

- App 内普通聊天不区分 App 来源或游戏内来源。
- 发送者展示由后端基于服务器身份提供。

### 7.6 Server -> Client：服务器事件提示

`type = server.event`

```json
{
  "type": "server.event",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "event": {}
  }
}
```

规则：

- 仅当连接开启 `includeServerEvents=true` 时推送。
- 事件提示不能伪装成玩家发言。

### 7.7 Server -> Client：在线状态变化

`type = presence.update`

```json
{
  "type": "presence.update",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "onlineCount": 12,
    "players": []
  }
}
```

规则：

- `players` 可以省略；省略时 Android 只更新在线人数。
- 在线玩家列表完整刷新可以继续调用 HTTP `GET /api/v1/chat/online-players`。

### 7.8 Server -> Client：错误

`type = error`

```json
{
  "type": "error",
  "requestId": "req_ws_01",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "code": "CHAT_MESSAGE_TOO_LONG",
    "message": "消息不能超过 256 字符。"
  }
}
```

### 7.9 Client -> Server：App 前后台状态

`type = app.state`

```json
{
  "type": "app.state",
  "sentAt": "2026-04-30T12:30:00Z",
  "payload": {
    "foreground": true
  }
}
```

规则：

- App 进入前台时发送 `foreground=true`。
- App 退后台时发送 `foreground=false`，但不主动断开 WebSocket。
- 后端用该状态维护玩家目录中的 App 在线状态。

### 7.10 Server -> Client：钱包流水事件

`type = wallet.record.event`

```json
{
  "type": "wallet.record.event",
  "sentAt": "2026-04-30T12:30:00Z",
  "payload": {
    "record": {}
  }
}
```

规则：

- 后端在成功写入当前用户可见 `wallet_records` 后推送。
- 事件只推给相关已登录 App 用户，不广播给所有聊天连接。
- 旧 App 忽略未知 WebSocket 类型即可保持兼容。

### 7.11 Server -> Client：@ 提及事件

`type = chat.mention.event`

```json
{
  "type": "chat.mention.event",
  "sentAt": "2026-05-01T12:30:00Z",
  "payload": {
    "message": {}
  }
}
```

规则：

- payload 复用 `ChatMessage`。
- 后端只推给当前有 App WebSocket 连接的被提及用户。
- 不推给发送者自己。
- 旧 App 忽略未知 WebSocket 类型即可保持兼容。

## 8. AI HTTP API

AI HTTP API 复用 `/api/v1`、opaque Bearer token、统一成功/错误响应和 `X-Request-Id`。

### 8.1 获取 AI 状态

`GET /api/v1/ai/me`

响应：

```json
{
  "requestId": "req_opaque",
  "data": {
    "assistantName": "xxxAI",
    "plan": {},
    "quota": {},
    "conversation": {
      "conversationId": "aic_opaque",
      "active": true,
      "startedAt": "2026-06-15T12:00:00Z",
      "updatedAt": "2026-06-15T12:00:00Z"
    }
  }
}
```

### 8.2 获取套餐列表

`GET /api/v1/ai/plans`

响应：

```json
{
  "requestId": "req_opaque",
  "data": {
    "plans": []
  }
}
```

规则：

- App 购买页必须以该接口返回为准，不写死套餐额度、价格或有效期。

### 8.3 获取当前会话消息

`GET /api/v1/ai/messages?limit=50&before=opaque`

响应：

```json
{
  "requestId": "req_opaque",
  "data": {
    "messages": []
  }
}
```

规则：

- 只返回当前活跃会话消息。
- `/new` 或 reset 后旧会话不在 App 恢复。

### 8.4 流式发送 AI 消息

`POST /api/v1/ai/chat/stream`

请求：

```json
{
  "clientMessageId": "android-ai-msg-opaque",
  "content": "怎么绑定账号？"
}
```

响应类型：

```http
Content-Type: text/event-stream
Cache-Control: no-cache
X-Accel-Buffering: no
```

SSE `meta`：

```text
event: meta
data: {"conversationId":"aic_opaque","userMessageId":"aim_user","assistantMessageId":"aim_assistant","quota":{}}
```

SSE `delta`：

```text
event: delta
data: {"content":"你好"}
```

SSE `status`：

```text
event: status
data: {"status":"searching_knowledge"}
```

`status` 可选值：

- `thinking`：后端已进入 AI 生成准备阶段。
- `searching_knowledge`：后端正在执行受控知识库检索。

SSE `sources`：

```text
event: sources
data: [{"query":"创造权限申请","title":"服务器权限申请详细规则","category":"wiki","headingPath":"服务器权限申请详细规则","sourceUrl":"https://wiki.deuterium.cafe/zh/GetOP","score":12}]
```

规则：

- `sources` 只用于 AI 私聊参考来源展示和排障；公共聊天不展示 Markdown 和来源卡片。
- 后端可以在等待模型或查库期间发送 SSE comment heartbeat，例如 `: keep-alive`；App 必须忽略 comment，但用它保持连接活跃。
- App 流式阶段建议先按纯文本显示；收到 `done` 后再做 Markdown 渲染。

SSE `done`：

```text
event: done
data: {"message":{},"quota":{}}
```

SSE `error`：

```text
event: error
data: {"error":{"code":"AI_QUOTA_EXCEEDED","message":"AI 请求次数已用完。","retryAfterSeconds":300}}
```

规则：

- `content` trim 后不能为空。
- `content == "/new"` 等同新建会话，不消耗额度。
- `clientMessageId` 是 AI 私聊发送幂等键，trim 后不能为空且最长 128 字符。后端用请求交换状态记录 `pending/streaming/completed/failed` 生命周期。
- 同一登录用户重复提交同一 `clientMessageId` 时，后端不得重复写用户消息或重复扣额度；若已有完整回复，应返回既有回复；若仍在 `pending/streaming`，返回 `AI_PROVIDER_UNAVAILABLE` 并提示稍后刷新；若处理中状态超过后端超时窗口或已失败，提示重新发送新消息。
- 同一用户同一时间只允许一个 AI 私聊请求处于 `pending/streaming`。新的不同 `clientMessageId` 请求遇到旧请求仍在运行时，返回 `AI_PROVIDER_UNAVAILABLE`，防止并发绕过额度和 provider 成本控制。
- 如果客户端已收到部分 `delta` 后主动断开，后端可消耗一次额度并把该请求标记失败，用于覆盖 provider 成本。
- 后端必须区分 provider 首个正文 `delta` 超时、连续正文 `delta` 空闲超时、HTTP 错误和 App 普通网络错误，并把 provider 状态码、首 token 延迟、总耗时、retry 次数和是否已发送 delta 写入审计。Provider keep-alive/comment 不得无限延长首个正文 delta 等待窗口或正文 delta 空闲窗口。
- App 应按 `delta` 增量展示，并以 `done.message` 作为最终消息。
- AI 私聊可渲染安全 Markdown；公共聊天触发的 AI 回复仍为纯文本、非流式。

### 8.5 重置当前会话

`POST /api/v1/ai/conversation/reset`

响应：

```json
{
  "requestId": "req_opaque",
  "data": {
    "conversation": {}
  }
}
```

### 8.6 购买 AI 套餐

`POST /api/v1/ai/purchases`

请求：

```json
{
  "clientRequestId": "android-ai-purchase-opaque",
  "planId": "plan_pro"
}
```

响应：

```json
{
  "requestId": "req_opaque",
  "data": {
    "purchase": {},
    "quota": {}
  }
}
```

规则：

- `clientRequestId` 是购买幂等键。
- 同一 `clientRequestId` 重复且套餐一致返回同一购买结果。
- 套餐购买通过插件桥扣当前玩家信用点。
- 购买结果未知时返回 `202`，不授予权益。

### 8.7 查询购买结果

`GET /api/v1/ai/purchases/{purchaseId}`

响应：

```json
{
  "requestId": "req_opaque",
  "data": {
    "purchase": {},
    "quota": {}
  }
}
```

## 9. Plugin Bridge WebSocket

### 9.1 连接

路径：

```text
GET /bridge/plugin/ws
```

鉴权：

```http
Authorization: Bearer <plugin-bridge-token>
```

规则：

- 插件启动后主动连接后端。
- 插件桥 token 只能放在生产本地配置或环境变量中，不得提交到仓库。
- 鉴权失败时后端拒绝连接并记录安全日志。
- 插件桥不可用时，后端对依赖服务器能力的公开 API 返回 `PLUGIN_BRIDGE_UNAVAILABLE`。

### 9.2 消息信封

```json
{
  "type": "player.resolve.request",
  "messageId": "bridge_msg_opaque",
  "replyTo": null,
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {}
}
```

规则：

- `messageId` 由发送方生成。
- 响应消息使用 `replyTo` 指向请求的 `messageId`。
- 请求必须有超时，超时后后端按失败处理。
- 插件桥不提供通用远程命令执行消息。

### 9.3 心跳

`bridge.ping`

```json
{
  "type": "bridge.ping",
  "messageId": "bridge_msg_ping",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {}
}
```

`bridge.pong`

```json
{
  "type": "bridge.pong",
  "messageId": "bridge_msg_pong",
  "replyTo": "bridge_msg_ping",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {}
}
```

### 9.4 后端 -> 插件：投递验证码

`type = verification.deliver.request`

```json
{
  "type": "verification.deliver.request",
  "messageId": "bridge_msg_01",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "verificationId": "ver_internal",
    "purpose": "registration",
    "gameId": "StarDust",
    "code": "123456",
    "expiresAt": "2026-04-29T12:40:00Z"
  }
}
```

响应 `verification.deliver.result`：

```json
{
  "type": "verification.deliver.result",
  "messageId": "bridge_msg_02",
  "replyTo": "bridge_msg_01",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "status": "delivered",
    "serverUuid": "internal-server-uuid",
    "currentGameId": "StarDust"
  }
}
```

`purpose`：

- `registration`
- `password_reset`

`status`：

- `delivered`
- `player_offline`
- `player_not_found`
- `identity_conflict`
- `failed`

### 9.5 后端 -> 插件：解析玩家身份

`type = player.resolve.request`

```json
{
  "type": "player.resolve.request",
  "messageId": "bridge_msg_03",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "gameId": "Lantern",
    "purpose": "transfer_recipient"
  }
}
```

响应 `player.resolve.result`：

```json
{
  "type": "player.resolve.result",
  "messageId": "bridge_msg_04",
  "replyTo": "bridge_msg_03",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "status": "resolved",
    "serverUuid": "internal-server-uuid",
    "currentGameId": "Lantern",
    "online": true
  }
}
```

`purpose`：

- `registration`
- `password_reset`
- `transfer_recipient`
- `wallet_owner`
- `chat_sender`
- `online_list`

`status`：

- `resolved`
- `player_not_found`
- `identity_conflict`
- `failed`

### 9.6 后端 -> 插件：查询余额

`type = wallet.balance.request`

```json
{
  "type": "wallet.balance.request",
  "messageId": "bridge_msg_05",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "serverUuid": "internal-server-uuid"
  }
}
```

响应 `wallet.balance.result`：

```json
{
  "type": "wallet.balance.result",
  "messageId": "bridge_msg_06",
  "replyTo": "bridge_msg_05",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "status": "success",
    "amount": "1820.50",
    "currency": "CREDIT",
    "currentGameId": "StarDust"
  }
}
```

`status`：

- `success`
- `player_not_found`
- `economy_unavailable`
- `failed`

### 9.7 后端 -> 插件：执行转账

`type = wallet.transfer.request`

```json
{
  "type": "wallet.transfer.request",
  "messageId": "bridge_msg_07",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "transferId": "tr_internal",
    "idempotencyKey": "android-client-request-id",
    "fromServerUuid": "internal-from-uuid",
    "toServerUuid": "internal-to-uuid",
    "amount": "68.50",
    "currency": "CREDIT",
    "note": "矿镐维修"
  }
}
```

响应 `wallet.transfer.result`：

```json
{
  "type": "wallet.transfer.result",
  "messageId": "bridge_msg_08",
  "replyTo": "bridge_msg_07",
  "sentAt": "2026-04-29T12:30:02Z",
  "payload": {
    "status": "success",
    "reason": null
  }
}
```

`status`：

- `success`
- `balance_insufficient`
- `recipient_not_found`
- `economy_unavailable`
- `failed`
- `unknown`

规则：

- 插件必须只执行受控转账能力。
- 插件不得接受任意命令字符串。
- `unknown` 表示插件无法确认最终结果，后端对 App 返回结果未知并提示查账。

### 9.8 后端 -> 插件：受控扣款

`type = wallet.debit.request`

```json
{
  "type": "wallet.debit.request",
  "messageId": "bridge_msg_debit_01",
  "sentAt": "2026-06-15T12:00:00Z",
  "payload": {
    "debitId": "aip_opaque",
    "idempotencyKey": "android-ai-purchase-opaque",
    "serverUuid": "player-server-uuid",
    "amount": "500.00",
    "currency": "CREDIT",
    "note": "购买 AI Pro"
  }
}
```

响应 `wallet.debit.result`：

```json
{
  "type": "wallet.debit.result",
  "messageId": "bridge_msg_debit_02",
  "replyTo": "bridge_msg_debit_01",
  "sentAt": "2026-06-15T12:00:02Z",
  "payload": {
    "status": "success",
    "reason": null,
    "balanceAfter": "1200.00"
  }
}
```

`status`：

- `success`
- `balance_insufficient`
- `player_not_found`
- `economy_unavailable`
- `failed`
- `unknown`

规则：

- 该能力只允许扣当前指定玩家信用点，不允许执行任意命令。
- `unknown` 时后端不得授予 AI 套餐权益。

### 9.9 插件 -> 后端：服务器公共聊天

`type = chat.serverMessage.event`

```json
{
  "type": "chat.serverMessage.event",
  "messageId": "bridge_msg_09",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "serverMessageId": "mc_msg_opaque",
    "serverUuid": "internal-server-uuid",
    "currentGameId": "EchoMint",
    "content": "有人要去末地补货吗？",
    "occurredAt": "2026-04-29T12:30:00Z"
  }
}
```

规则：

- 插件上报服务器侧身份，后端负责转换成 App 可展示的 `PlayerSummary`。
- 后端可记录最近公共聊天，用于 App 进入聊天页时加载最近 100 条。

### 9.10 插件 -> 后端：服务器事件提示

`type = server.event`

```json
{
  "type": "server.event",
  "messageId": "bridge_msg_10",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "eventId": "mc_evt_opaque",
    "eventType": "death",
    "content": "Lantern 摔得太狠了",
    "occurredAt": "2026-04-29T12:30:00Z"
  }
}
```

规则：

- 只允许白名单事件类型。
- 第一版白名单建议：`death`、`server_say`、`join`、`quit`。
- App 默认不展示服务器事件提示，用户开启后才展示。

### 9.10.1 插件 -> 后端：服务器 Pay 流水事件

`type = wallet.pay.event`

```json
{
  "type": "wallet.pay.event",
  "messageId": "bridge_msg_pay_01",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "payEventId": "mc_pay_opaque",
    "fromServerUuid": "payer-server-uuid",
    "fromGameId": "StarDust",
    "toServerUuid": "receiver-server-uuid",
    "toGameId": "Lantern",
    "amount": "100.00",
    "currency": "CREDIT",
    "status": "success",
    "note": "服务器内转账",
    "fromBalanceAfter": "900.00",
    "toBalanceAfter": "300.00",
    "occurredAt": "2026-04-29T12:30:00Z"
  }
}
```

规则：

- 只上报插件已确认成功的服务器内 `/pay`。
- 后端将该事件写入现有 `wallet_records`，并通过 `GET /wallet/records` 返回兼容 `WalletRecord`。
- 同一 `payEventId` 重复上报不得产生重复流水。
- 若付款方或收款方未注册 App，只给已注册一方写可见流水。

### 9.10.2 插件 -> 后端：服务器经济余额变化事件

`type = wallet.balanceChange.event`

```json
{
  "type": "wallet.balanceChange.event",
  "messageId": "bridge_msg_balance_01",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "eventId": "mc_balance_opaque",
    "serverUuid": "player-server-uuid",
    "gameId": "StarDust",
    "direction": "income",
    "amount": "12.50",
    "currency": "CREDIT",
    "status": "success",
    "source": "server_economy",
    "note": "服务器经济变动",
    "balanceAfter": "912.50",
    "occurredAt": "2026-04-29T12:30:00Z"
  }
}
```

规则：

- 用于无法可靠识别对方玩家或业务来源的经济变动，例如箱子商店、奖励、扣款或其他插件导致的余额变化。
- `direction` 只允许 `income` 或 `expense`。
- 后端仍写入现有 `wallet_records`，并通过 `GET /wallet/records` 返回兼容 `WalletRecord`。
- 同一 `eventId` 重复上报不得产生重复流水。
- 如果后续插件能明确识别来源，可把 `source` 和 `note` 设置为更具体的来源，例如 `chest_shop` / `箱子商店`。

### 9.11 后端 -> 插件：转发 App 公共聊天

`type = chat.appMessage.request`

```json
{
  "type": "chat.appMessage.request",
  "messageId": "bridge_msg_11",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "appMessageId": "msg_internal",
    "senderServerUuid": "internal-server-uuid",
    "senderGameId": "StarDust",
    "content": "晚上一起去下界吗？"
  }
}
```

响应 `chat.appMessage.result`：

```json
{
  "type": "chat.appMessage.result",
  "messageId": "bridge_msg_12",
  "replyTo": "bridge_msg_11",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "status": "sent"
  }
}
```

`status`：

- `sent`
- `sender_not_found`
- `chat_unavailable`
- `failed`

规则：

- App 消息在服务器内显示为普通游戏内聊天，不额外标记 App 来源。
- 服务器内广播格式固定为 `§x§b§1§f§7§f§f%player% §7: §f%message%`。
- 插件不得允许后端传入任意伪造身份；如果 `senderServerUuid` 无法匹配已绑定身份或当前服务器身份，应返回失败。

### 9.12 后端 -> 插件：在线玩家列表

`type = presence.list.request`

```json
{
  "type": "presence.list.request",
  "messageId": "bridge_msg_13",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {}
}
```

响应 `presence.list.result`：

```json
{
  "type": "presence.list.result",
  "messageId": "bridge_msg_14",
  "replyTo": "bridge_msg_13",
  "sentAt": "2026-04-29T12:30:01Z",
  "payload": {
    "status": "success",
    "onlineCount": 2,
    "players": [
      {
        "serverUuid": "internal-server-uuid",
        "currentGameId": "Lantern"
      }
    ]
  }
}
```

### 9.13 插件 -> 后端：在线状态快照

`type = presence.snapshot.event`

```json
{
  "type": "presence.snapshot.event",
  "messageId": "bridge_msg_15",
  "sentAt": "2026-04-29T12:30:00Z",
  "payload": {
    "onlineCount": 2,
    "players": [
      {
        "serverUuid": "internal-server-uuid",
        "currentGameId": "Lantern"
      }
    ]
  }
}
```

规则：

- 插件可以在连接建立后或在线状态变化时上报快照。
- 后端可以用快照更新 App Chat WebSocket 的 `presence.update`。

## 10. 安全与实现边界

- App 只连接后端 API 和 App Chat WebSocket。
- App 不连接数据库。
- App 不连接 Minecraft 插件桥。
- App 不提交可信 UUID。
- 后端不得把数据库凭据、插件桥 token、验证码明文或会话 token 写入日志。
- 插件桥只实现本文件列出的受控能力。
- 插件桥不得实现通用远程命令执行。
- 插件桥断开时，注册验证码、改密验证码、余额刷新、收款方身份确认、转账执行、App 聊天发送和在线状态获取都应明确失败。
- 第一版不排队钱包转账和 App 聊天消息。
- 新增玩家目录、关心和钱包流水推送必须兼容旧接口；旧 App 不更新也不应受影响。

## 11. 覆盖检查

账号产品级操作覆盖：

- 请求注册验证码：`POST /account/registration-code` + `verification.deliver.request`
- 提交注册：`POST /account/register`
- 登录：`POST /account/login`
- 请求密码重设验证码：`POST /account/password-reset-code` + `verification.deliver.request`
- 提交新密码：`POST /account/password-reset`
- 退出登录：`POST /account/logout`

钱包产品级操作覆盖：

- 查看信用点余额：`GET /wallet/balance`
- 刷新信用点余额：`POST /wallet/balance/refresh` + `wallet.balance.request`
- 查找收款玩家：`GET /wallet/recipients/search` + `player.resolve.request`
- 发起信用点转账：`POST /wallet/transfers` + `wallet.transfer.request`
- 查询转账结果：`GET /wallet/transfers/{transferId}`
- 查看基础流水：`GET /wallet/records`
- 离线后增量补齐流水：`GET /wallet/records?afterRecordId=...`
- 后台接收钱包变动：`wallet.record.event`

聊天产品级操作覆盖：

- 查看最近公共聊天：`GET /chat/messages`
- 实时接收公共聊天：`GET /chat/ws` + `chat.message`
- 发送公共聊天：`chat.send` + `chat.appMessage.request`
- 点选 @ 玩家并发送提及通知：`chat.send.mentionedPlayerRefs` + `chat.mention.event`
- 服务器事件提示：`server.event`
- 查看在线人数：`GET /chat/presence` + `presence.update`
- 查看在线玩家列表：`GET /chat/online-players` + `presence.list.request`
- 查看玩家目录：`GET /chat/player-directory`
- 关心玩家：`GET/POST/DELETE /chat/follows`
- 上报 App 前后台状态：`app.state`
- 在线玩家进入转账：在线玩家返回 `playerRef`，转账仍走 `wallet-transfer` 二次确认

App 基础操作覆盖：

- 检查更新：`GET /app/update-check`

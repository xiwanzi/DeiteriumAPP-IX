# Deuterium OIDC Provider v1

## 1. 文档定位

本文档定义 Deuterium 后端作为最小 OIDC Provider 时，对 Wiki.js 暴露的接口、配置、安全边界和当前已确认的 Wiki.js 接入方式。

该能力用于让 Wiki.js 使用 Deuterium APP 账号登录，不替代 Android APP 现有 `/api/v1/*` opaque Bearer session，也不要求 Android APP 或 Minecraft 插件桥更新。

## 2. Origin 与端口

后端 OIDC endpoints 与公开 App API 共用 `28657` 端口。插件桥端口 `28658` 不参与 OIDC，仍不得暴露到公网或内网穿透。

推荐生产 issuer：

```text
https://auth.deuterium.cafe
```

当前已确认的内部 HTTP-only 入口：

```text
http://authdeuterium.s.odn.cc -> 127.0.0.1:28657
```

公网密码登录必须优先使用 HTTPS。仅在内部或受控 HTTP-only 环境中，才允许同时配置：

```properties
oidc.issuer=http://authdeuterium.s.odn.cc
oidc.allowInsecureHttp=true
```

## 3. Endpoints

```text
GET  /.well-known/openid-configuration
GET  /.well-known/jwks.json
GET  /oauth/authorize
POST /oauth/login
POST /oauth/token
GET  /oauth/userinfo
GET  /oauth/logout
```

支持的 flow：

```text
Authorization Code Flow
```

支持的 scopes：

```text
openid profile email
```

v1 不签发 refresh token。

Discovery 文档包含：

- `issuer`
- `authorization_endpoint`
- `token_endpoint`
- `userinfo_endpoint`
- `jwks_uri`
- `response_types_supported=["code"]`
- `subject_types_supported=["public"]`
- `id_token_signing_alg_values_supported=["RS256"]`
- `scopes_supported=["openid","profile","email"]`
- `token_endpoint_auth_methods_supported=["client_secret_basic","client_secret_post"]`
- `claims_supported=["sub","email","displayName","preferred_username","name"]`

所有 discovery、JWKS、token 和 userinfo 响应都应带 `Cache-Control: no-store` / `Pragma: no-cache`。

## 4. Wiki.js Client

当前 Wiki.js 使用内置 Generic OIDC strategy，不修改 Wiki.js 源码。

推荐 HTTPS 配置：

```text
Client ID: wikijs
Issuer: https://auth.deuterium.cafe
Authorization URL: https://auth.deuterium.cafe/oauth/authorize
Token URL: https://auth.deuterium.cafe/oauth/token
UserInfo URL: https://auth.deuterium.cafe/oauth/userinfo
JWKS URL: https://auth.deuterium.cafe/.well-known/jwks.json
Logout URL: https://auth.deuterium.cafe/oauth/logout
Email claim: email
Display name claim: displayName
Scopes: openid profile email
```

当前已确认的内部 HTTP-only 配置：

```text
Strategy key: oidc
Display name: Deuterium 账号登录
Enabled: true
Self registration: true
Client ID: wikijs
Client Secret: <redacted>
Issuer: http://authdeuterium.s.odn.cc
Authorization URL: http://authdeuterium.s.odn.cc/oauth/authorize
Token URL: http://authdeuterium.s.odn.cc/oauth/token
UserInfo URL: http://authdeuterium.s.odn.cc/oauth/userinfo
Logout URL: http://authdeuterium.s.odn.cc/oauth/logout
Email claim: email
Display name claim: displayName
Group mapping: false
autoEnrollGroups: [4]
```

当前已确认的 Wiki.js 自动入组目标：

```text
Group ID: 4
Group Name: Deuterium Users
Permissions:
- read:pages
- read:assets
- read:comments
- write:comments
```

`redirect_uri` 必须与 `backend/config/application.conf` 中的 `oidc.redirectUri` 精确一致，不做前缀匹配或域名级匹配。

当前 callback：

```text
https://wiki.deuterium.cafe/login/oidc/callback
```

## 5. Authorization 与登录

`GET /oauth/authorize` 支持参数：

- `response_type=code`
- `client_id`
- `redirect_uri`
- `scope`，必须包含 `openid`
- `state`，可选，原样带回
- `nonce`，可选，写入 ID Token

行为：

- 后端自动 upsert 当前配置中的 OIDC client。
- `client_id` 必须存在且启用。
- `redirect_uri` 必须精确匹配登记值。
- `response_type` 只接受 `code`。
- 已有有效 OIDC 浏览器 session 时直接签发 authorization code 并重定向回 Wiki.js callback。
- 没有 session 时返回后端内置的最小 HTML 登录页。

`POST /oauth/login` 使用 `application/x-www-form-urlencoded`，字段包括：

- OIDC 上下文字段：`response_type`、`client_id`、`redirect_uri`、`scope`、`state`、`nonce`
- 账号字段：`account`
- 密码字段：`password`

登录规则：

- 复用 Deuterium 账号表和密码校验逻辑。
- `account` 可以是游戏内 ID 或 QQ 号，与 APP 登录输入语义一致。
- 只允许 `status=active` 的用户完成 OIDC 登录。
- OIDC 登录失败计数使用独立 key：`oidc|account|remoteHost`，避免和 Android APP 登录失败计数完全混在一起。

登录成功后创建独立浏览器 session，cookie 名：

```text
deuterium_oidc_session
```

cookie 属性：

- `HttpOnly`
- `SameSite=Lax`
- `path=/`
- `Secure` 仅当 `oidc.issuer` 为 HTTPS 时启用

## 6. Token 与 UserInfo

`POST /oauth/token` 使用 `application/x-www-form-urlencoded`。

支持：

- `grant_type=authorization_code`
- `client_secret_basic`
- `client_secret_post`，即表单字段 `client_id` / `client_secret`

校验：

- client 必须存在且启用。
- `redirect_uri` 必须精确匹配登记值。
- `client_secret` 只和配置中的 SHA-256 hex hash 做常量时间比较。
- authorization code 必须未过期、未消费，且 client 与 redirect URI 匹配。
- authorization code 消费使用 `consumed_at` 标记，重复消费返回 `invalid_grant`。

成功响应字段：

- `access_token`
- `token_type=Bearer`
- `expires_in`
- `id_token`
- `scope`

`GET /oauth/userinfo` 使用：

```http
Authorization: Bearer <oidc-access-token>
```

返回 claim：

```json
{
  "sub": "usr_...",
  "email": "usr_...@users.deuterium.local",
  "displayName": "CurrentGameId",
  "preferred_username": "CurrentGameId",
  "name": "CurrentGameId"
}
```

`sub` 是稳定的 Deuterium 用户 ID。当前游戏名可能变化，不得作为 Wiki.js 侧稳定身份 key。

`GET /oauth/logout` 撤销当前 OIDC 浏览器 session，并清空 `deuterium_oidc_session` cookie。OIDC logout 不会登出 Android APP。

## 7. 后端配置

OIDC 默认关闭：

```properties
oidc.enabled=false
oidc.issuer=http://127.0.0.1:28657
oidc.clientId=wikijs
oidc.clientSecretHash=
oidc.redirectUri=https://wiki.deuterium.cafe/login/oidc/callback
oidc.signingKeyPath=config/oidc-signing-key.json
oidc.allowInsecureHttp=false
oidc.authorizationCodeMinutes=5
oidc.accessTokenMinutes=10
oidc.idTokenMinutes=10
oidc.webSessionHours=12
```

内部 HTTP-only 环境示例：

```properties
oidc.enabled=true
oidc.issuer=http://authdeuterium.s.odn.cc
oidc.clientId=wikijs
oidc.clientSecretHash=<sha256-hex-of-client-secret>
oidc.redirectUri=https://wiki.deuterium.cafe/login/oidc/callback
oidc.signingKeyPath=config/oidc-signing-key.json
oidc.allowInsecureHttp=true
oidc.authorizationCodeMinutes=5
oidc.accessTokenMinutes=10
oidc.idTokenMinutes=10
oidc.webSessionHours=12
```

`backend/config/oidc-signing-key.json` 保存 RS256 签名私钥，必须在后端包升级时保留或复制。否则 Wiki.js 已缓存的旧 `kid` / 签名可能短期内校验失败。

明文 Wiki.js client secret 不得写入 Git，不得写入本文档；`application.conf` 只保存 raw SHA-256 hex hash。

## 8. 数据库迁移

OIDC Provider 使用 Flyway migration：

```text
backend-api/src/main/resources/db/migration/V5__oidc_provider.sql
```

新增表：

- `oidc_clients`
- `oidc_authorization_codes`
- `oidc_access_tokens`
- `oidc_web_sessions`

新增索引：

- `idx_oidc_codes_client_expiry(client_id, expires_at, consumed_at)`
- `idx_oidc_access_tokens_expiry(expires_at, revoked_at)`
- `idx_oidc_web_sessions_expiry(expires_at, revoked_at)`

这些表是增量新增表，不修改旧 APP 表结构或插件桥表结构。旧后端回滚后可以忽略这些表。

## 9. 安全与兼容边界

- OIDC client secret 不入库明文；后端只保存 SHA-256 hex。
- Authorization code、OIDC access token、OIDC browser session 都只保存 pepper hash。
- OIDC browser session 与 Android APP session 分表隔离。
- `/api/v1/*` 不接受 OIDC access token。
- `/api/v1/account/login` 不改成 OIDC。
- `/api/v1/account/me` 仍只接受 Android APP opaque Bearer session。
- 插件桥 `/bridge/plugin/ws` 不开放给 OIDC。
- ID Token 使用 RS256。
- v1 不实现 refresh token。
- v1 不做 Wiki.js group claim mapping；权限由 Wiki.js `autoEnrollGroups` 控制。

## 10. 验证与回滚

已覆盖的后端测试：

```powershell
cd C:\DeuteriumAPP\backend-api
.\gradlew.bat test --tests com.deuterium.backend.OidcRoutesTest --tests com.deuterium.backend.AppConfigTest
```

覆盖点：

- discovery endpoint 输出配置中的 issuer 和 endpoint。
- token endpoint 接受 Wiki.js 风格的 authorization code exchange。
- 未登记 redirect URI 被拒绝。
- authorization code 重复消费被拒绝。
- OIDC access token 不能访问 `/api/v1/account/me`。
- 非 localhost HTTP issuer 默认被拒绝。
- 显式 `oidc.allowInsecureHttp=true` 时允许内部 HTTP issuer。

回滚方式：

- 只回滚 Wiki.js 登录入口：禁用 Wiki.js `oidc` authentication strategy；`Deuterium Users` 组可保留。
- 只回滚 staged backend 配置：恢复同目录 `application.conf.bak-oidc-*` 或把 `oidc.enabled=false`。
- 回滚旧后端：停止新 backend，启动旧 backend；新增 OIDC 表可保留，旧后端会忽略。
- 如未来重新启用 OIDC，应优先复用 `backend/config/oidc-signing-key.json`。

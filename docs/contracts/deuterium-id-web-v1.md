# Deuterium ID 与浏览器会话增量 v1

日期：2026-09-08。此文是新后端增量，不将原 v2 契约的全部端点标成实现。当前支持从旧库导入后登录，注册和改密待游戏验证码适配。

## 共用账号

`userId` 是 Deuterium ID 稳定主体。旧账号沿用 `app_users.id`；游戏 UUID 不下发为客户端可修改标识；QQ 与游戏名统一作为登录别名。同一别名不能同时指向两人。`playerRef` 仍为游戏业务引用，不能替代 userId 作为其他应用主键。

旧密码兼容 PHC Argon2i/Argon2id v19。本机旧代码使用 Argon2Factory.create()、m=65536、t=3、p=2，导入时保留哈希；成功登录后将旧类型升级为 Argon2id。迁移不携带会话、验证码和管理权限，禁用账号保持禁用。导出文件是敏感本机文件，不提交 Git，不通过网页上传。

2026-09-08 迁移核实补充：旧库允许存在非数字 QQ 历史数据。导入保留 1–20 个可打印 ASCII、无空白/控制字符的原 QQ 显示值；不符合数字 QQ 规则的值**不创建 QQ 登录别名**，账号仍通过原游戏名登录，ImportResult 的 `skippedQqAliases` 明确计数。ID、UUID、密码、状态与审计指纹不变，不修改旧库；新注册仍严格要求数字 QQ，不能把这条历史兼容规则用于新账号。

## App

- `POST /api/v1/account/login`：原有 `{account,password}`；成功 `data={token,user}`。
- `GET /api/v1/account/me`：`Authorization: Bearer <token>`；返回 `data.user`。
- `POST /api/v1/account/logout`：撤销当前会话；返回 `data.loggedOut=true`。
- 不透明随机 256 bit token；数据库仅保存 SHA-256。有效期 7 天，首版无 refresh token 接口，过期重新登录。未来会话续期另加明确协议，不假装现有 token 永久有效。

## 玩家网页与官方管理网页

同域部署，例如 `https://<community>` 的网页、`/api/v1` 和 `/api/v1/chat/ws`。实际域名尚未指定，配置占位域名不可用于上线。

| 操作 | 请求 | 成功数据 |
| --- | --- | --- |
| 登录 | `POST /api/v1/web/session`，JSON `{account,password}`，Origin 必须精确匹配配置 | `{user,csrfToken,expiresAt}`；Set-Cookie，不返回 Bearer token |
| 恢复会话 | `GET /api/v1/web/session`，浏览器 Cookie | `{user,csrfToken,expiresAt}` |
| 当前用户 | 原有 `GET /api/v1/account/me` 支持 Cookie | `{user}` |
| 退出 | `DELETE /api/v1/web/session` 或原有 POST logout，Cookie＋Origin＋X-CSRF-Token | `{loggedOut:true}`，删除 Cookie |
| 实时连接 | 原有 `GET /api/v1/chat/ws`，浏览器自动携带 Cookie，Origin 精确检查 | 使用原有聊天信封；不在 URL 放 token |

生产 Cookie：`__Host-deuterium_session`，Secure、HttpOnly、Path=/、SameSite=Lax，不设置 Domain。仅本地回环开发模式使用不同名字且允许 HTTP。浏览器将 csrfToken 保存在内存；刷新后 GET 会话重新获取。所有 Cookie 认证的修改请求必须同时通过 Origin 和 CSRF；不能仅靠 SameSite。

登录不要求旧 CSRF，但强制 JSON 和精确 Origin，重新登录轮换旧 Cookie 会话。拒绝同一请求同时携带 Bearer 和会话 Cookie；App token 不能当作网页 Cookie 使用，反之亦然。网页不使用 localStorage 保存登录令牌。

服务端不启用宽泛 CORS。反向代理必须覆盖 X-Real-IP，应用只从显式 trustedProxyCidrs 接受它，否则用实际连接 IP。登录按账号和 IP 持久限次，数据库不可用时拒绝认证；哈希验证并发最多 2。

注销后的 WebSocket 写操作每次查会话；空闲连接在唤醒或 5 秒内复查并关闭。重连需重新查询历史，不能把 WebSocket 当成无期限授权。

## 后续统一 SSO

新的独立应用接入 Deuterium ID 的 OIDC Provider，使用 Authorization Code + PKCE（S256）、精确 redirect URI、state/nonce、受众与 scope 校验。OIDC 不增加第二套账密，也不让每个应用接触用户密码。不使用 password grant，不共享 App Bearer token 给第三个应用。

当前服务**未实现 OIDC discovery／authorize／token／JWKS**，不发布假的 discovery 数据。原 OIDC client 与旧 issuer/subject 的兼容迁移需单独盘点；保留旧 userId 不自动代表原第三方登录无需配置变更。依据：[RFC 9700](https://www.rfc-editor.org/rfc/rfc9700)。

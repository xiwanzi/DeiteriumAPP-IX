# 管理员永久注销与客户端清理

2026-09-12。已随 App 2.0.13 / Web 2.0.15 / 配套 Go 上线。补充[账号管理](saki-admin-v206.md)；[OpenAPI](openapi-account-deletion.yaml)；[部署记录](../deployment/deuterium-2.0.13-live.md)。

## 永久注销

`POST /api/v1/admin/accounts/{userId}/delete`。仅 `platform.admin` 的有效会话可调用。工作台使用现有 HttpOnly Cookie、同源 Origin 和 `X-CSRF-Token`。

```json
{"clientRequestId":"a-stable-request-id","expectedVersion":3,"password":"当前登录管理员的密码"}
```

成功 `200`：

```json
{"data":{"userId":"user_target","deleted":true,"version":4}}
```

- 密码根据认证会话的用户 ID 验证，不根据目标 userId 或表单账户名。复用现有 Argon2 算法和两个并发哈希槽。每位管理员 15 分钟内最多 8 次连续失败；每个来源 IP 15 分钟最多 60 次确认尝试。成功清除管理员失败预算，不清除 IP 预算。
- 密码不进入幂等摘要、请求持久记录、审计或响应。事务再次检查当前会话、密码摘要和管理权限；禁止注销自己，保留有效管理员。
- `expectedVersion` 来自账号与权限列表。注销为 `deleted` 终态，不可解封或恢复权限，默认列表排除已注销账号。
- 同一目标/版本/请求键成功重试返回原结果，仍须验证当前管理员密码。网络中断保留原请求键、清空密码输入，重新输入密码确认原操作结果。
- 未完成订单/委托、待处理退款/介入、处理中或结果未知的转账会阻止注销。持有商店所有权时先处理归属。平台分类、优惠券等公共管理资源不随创建者注销而清空。

| 状态/代码 | 含义 |
| --- | --- |
| 400 `ADMIN_PASSWORD_INVALID` | 当前管理员密码错误，目标不变，工作台会话不退出 |
| 401 `UNAUTHORIZED` | 会话失效或账号不可用 |
| 403 `FORBIDDEN` | 非管理员或 Origin/CSRF 拒绝 |
| 409 `ADMIN_CREDENTIAL_CHANGED` | 验证后管理员改过密码，须重新确认 |
| 409 `SELF_ACCOUNT_CHANGE` / `LAST_ADMIN` | 本人/最后有效管理员保护 |
| 409 `ACCOUNT_HAS_OPEN_TRANSACTIONS` | 订单、委托、退款或介入未完成 |
| 409 `ACCOUNT_HAS_PENDING_TRANSFER` | 转账仍在处理或结果未知 |
| 409 `ACCOUNT_OWNS_STORE` | 仍持有商店所有权 |
| 409 `ACCOUNT_DELETED` | 已永久注销，不能恢复 |
| 429 `RATE_LIMITED` | 密码确认限次，沿用 Retry-After |

格式、版本、资源不可见、幂等冲突及服务不可用沿用现有错误封装。

## 数据与展示

同一事务移除会话、密码/姓名/QQ、登录别名、权限、资料、关注、相关私聊及公开作者消息、个人市场商品展示和绑定、旧验证请求、AI 私有记录、购物袋和个人请求缓存；清理复制的消息引用，写入注销引用及管理员审计。

保留内部账号占位、财务记录和原始订单/争议快照。对外结构化当事人显示“已注销用户”，QQ/头像隐藏；快照摘要仍指向服务器留存的原始证据，不是身份隐藏后响应 JSON 的摘要。其他用户自行撰写的非结构化文本、外部截图、受限备份不属于可保证撤回的数据。

原游戏 UUID/QQ 可通过新的游戏验证码重新注册，新账号获得新的 userId/playerRef，不继承旧会话、私聊或商品。游戏账号、背包、余额不改。旧 App 引用始终不可用；未重新注册的旧身份不会被游戏目录或聊天事件重新加入 App。

资料/商品图片移除当前绑定，未绑定私有图片标记移除并进入现有生命周期；仍属于他人订单凭据或其他有效业务的图片保留。

## 注销增量

`GET /api/v1/account/deletions?after=0`，需要有效账号会话。每页最多 100 项，按 sequence 递增。

```json
{"data":{"items":[{"sequence":7,"playerRef":"player_old_opaque_reference"}],"cursor":7,"hasMore":false}}
```

不返回原姓名、QQ、UUID、聊天内容或私聊会话 ID。无新记录返回空 items 和传入游标。首次从 0 读取全部分页。

App 持久保存游标及注销引用，清理联系人、关注、消息、商品、私聊草稿、头像缓存和关联系统通知，过滤延迟返回的数据。完整联系人同步移除已消失的会话，同名新账号更换会话 ID 时清理旧消息；注销后重取关注关系。旧主页、会话退出或显示不可用，不生成虚构的空白资料。

网页在登录后同步引用，清理对应主页、会话、市场详情和缓存，保留正在编辑的其他账号/业务表单。刷新后重新同步。

离线或旧版客户端需更新并联网后清理。部署先迁移/后端/Web，再 App。`028_account_deletion.sql` 仅建立门控、记录和索引，本身不删除账号；实际注销只能通过管理员确认执行。回退不能恢复已注销账号。

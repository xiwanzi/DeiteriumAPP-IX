# 通行申请与代理准入接口

2026-09-12。申请站独立部署；当前目标 `https://47.103.99.34:9443`，管理端继续使用现有网站。所有业务响应沿用 `{requestId,data,serverTime}`，错误沿用 `{error:{code,message}}`。

## 公开申请

| 方法与路径 | 用途 |
|---|---|
| GET `/api/v1/admission/config` | 公约版本、群号/邀请链接、默认服及申请站地址 |
| POST `/api/v1/admission/applications` | 提交申请；无需 App 会话 |
| POST `/api/v1/admission/status` | 使用查询凭证读取结果 |

提交字段：`receiptToken`（客户端安全随机 32 字节、43 位 Base64URL，无填充）、`gameId`（3–16 位 Java 玩家名）、`qq`（5–11 位数字）、`interests`（0–4 个枚举值）、`message`（最多 200 字）、`covenantVersion`（当前 `2026-09-12-v1`）、`covenantAccepted=true`、`website`（反自动填写字段，必须为空）。

同一查询凭证也是提交幂等键：相同内容重试返回同一申请，内容变化返回冲突。凭证只存哈希，不能通过 UUID/QQ/玩家名读取申请。不同凭证不能给同一 UUID 同时创建多份待审申请；每 UUID 每日最多 3 份。IP/QQ 使用持久化限次，Nginx 另外限流。公开请求只接受配置的浏览器来源或无浏览器来源的原生请求；浏览器不携带账号 Cookie。

查询请求为 `{receiptToken}`。结果包含 `applicationId,gameId,status,createdAt,reviewedAt,reason,accessAllowed,accessStatus`。不暴露 QQ、UUID、审核人或其他申请。`reason` 只用于公开拒绝原因；通过时的内部审核备注不公开。申请状态 `PENDING/APPROVED/REJECTED` 与当前资格 `NONE/ACTIVE/REVOKED` 分离。新申请待审不被过去的撤销记录遮盖。

断网或 5xx 后保留原凭证和原请求，不能换键盲目重发。前端暂存当前标签页待确认内容，并保存查询凭证；刷新后先查询原结果。资料展示使用文本节点，不把留言/原因作为 HTML。

## 管理接口

全部要求当前有效管理员 `platform.admin`；Web 写操作要求原有 Cookie + Origin + CSRF。修改在数据库事务内再次验证管理员状态，使用持久幂等请求及版本冲突校验。

| 方法与路径 | 用途 |
|---|---|
| GET `/api/v1/admin/whitelist/summary` | 待审/有效/撤销数量、代理心跳与申请 URL |
| GET `/api/v1/admin/whitelist/applications` | `q,status,offset,limit` 筛选分页申请 |
| GET `/api/v1/admin/whitelist/entries` | 同上，读取当前白名单与断开执行状态 |
| GET `/api/v1/admin/whitelist/resolve?gameId=...` | 查询正版 UUID/名称；不等于账号归属证明 |
| POST `/api/v1/admin/whitelist/applications/{id}/review` | 通过/拒绝 |
| POST `/api/v1/admin/whitelist/entries` | 手动添加/重新添加 |
| POST `/api/v1/admin/whitelist/entries/{uuid}/revoke` | 移除资格，可同时断开在线玩家 |
| GET `/api/v1/admin/whitelist/entries/{uuid}/history` | 最近 100 条管理操作记录 |

审批输入：`clientRequestId,expectedVersion,decision(APPROVE/REJECT),reason,qqMemberConfirmed,identityConfirmed`。通过必须确认后两项；拒绝必须说明原因。第一版QQ群成员和账号使用权由管理员人工核对，不把申请者自己勾选当作核验结果。

手动添加输入：`clientRequestId,gameId,expectedUUID,qq,reason,qqMemberConfirmed,identityConfirmed`。后端重新向 Minecraft 官方解析名称并核对预览 UUID，避免确认期间名字归属变化。无需创建 App 账号。

移除输入：`clientRequestId,expectedVersion,reason,kickOnline`。当前资格立即撤销；不删除申请、玩家账号或存档。在线断开通过持久指令处理，状态为 `PENDING/DISCONNECTED/NOT_ONLINE/CANCELLED`。重新授予会取消未执行旧指令；代理执行前再次核对资格版本。

## 代理专用接口

POST `/bridge/v1/admission/check`，请求 `{uuid}`，响应 `{allowed,status,version,message}`。只有正版认证后的 UUID 可作为输入；缺项/异常不放行。

POST `/bridge/v1/admission/poll`，请求 `{instanceId,pluginVersion,onlinePlayers,acks:[{commandId,status}]}`，响应 `{commands:[{commandId,uuid,version,reason}]}`。每 5 秒轮询，客户端每批最多确认 25 条。超过 25 秒未有心跳，后台显示未连接。

凭据为独立随机 Bearer token。Go 配置 `admissionGatewayTokenSha256` 保存哈希，不能与 Core 节点共用；拒绝浏览器 Origin，不支持任意命令、不直接连接数据库。主 Nginx 需额外代理 `/bridge/v1/admission/` 的普通 HTTP POST，不能当作 WebSocket 路径。

## 部署与故障

代理在认证后异步查询资格，查询失败拒绝本次新连接；轮询故障不会主动断开已有连接。通过后才选择上次子服；首次进 Amiya，失败按 Amiya/Login 备用顺序尝试。成功切服原子保存，自动回退不覆盖原目的地，迟到的旧会话回调不覆盖新会话。

`029_admission.sql` 只新增申请、资格、事件、断开指令、心跳和锁表。`deuterium import-whitelist --file ...` 默认为预览；加 `--apply` 才导入，已有条目一律保留，不能通过重跑迁移重新启用已移除玩家。旧 App 账号的注册/注销与游戏通行权限独立管理。

网站换独立域名时，更新 Nginx 的站点/证书、Go 的 `admissionPublicOrigin`、代理的 `applicationUrl` 并验证来源校验。申请站和现有网站继续各自使用自己的站点根目录。

# 2.0.4 管理、提醒与统一流水

2026-09-09。基于 2.0.3 的增量实现，当前为候选版本，未部署。原资金执行、担保和订单隐藏规则继续适用。

## 权限与接口

网页使用既有 HttpOnly Cookie、Origin/CSRF；App 使用 Bearer。每次请求检查当前权限。游标不是授权凭据，不允许跨账号或跨列表复用。

| 接口（前缀 `/api/v1`） | 权限 | 行为 |
| --- | --- | --- |
| `POST /admin/announcements/{id}/delete` | `announcements.manage` / `platform.admin` | `{clientRequestId,expectedVersion}`；永久删除公告，过期版本返回 409 |
| `GET/PUT /admin/email-settings` | `platform.admin` | 读取/保存 SMTP 配置；GET 还返回待发数量与最近 20 条送达记录 |
| `POST /admin/email-settings/test` | `platform.admin` | `{clientRequestId}`；发送测试邮件入队，每分钟最多一条新测试，重试原请求不重复入队 |
| `GET /admin/interventions/summary` | `intervention.manage` / `platform.admin` | 未结案数量 `pending`、待受理数量 `submitted` |
| `GET /admin/players` | `audit.read` / `platform.admin` | `q,cursor`；注册账号与 Core 已知游戏玩家，25 条分页，可按游戏名/QQ/UUID/引用查找 |
| `GET /admin/transactions` | 同上 | 全服或 `playerRef` 指定玩家的已提交经济流水 |
| `GET /admin/orders` | 同上 | 所有订单，支持 `playerRef,channel,status,q,limit,cursor` |
| `GET /admin/orders/{id}` | 同上 | 含成交商品、交付和退款的只读详情，`availableActions=[]` |
| `GET /admin/products` | 同上 | 市场和官方商品，支持 `playerRef,kind,status,q,limit,cursor` |
| `GET /admin/products/{id}` | 同上 | 草稿/已发布商品详情，不能借审计接口修改商品 |
| `GET /wallet/records` | 当前会话 | 统一账本，个人 UUID 由会话固定；时间倒序 |
| `GET /wallet/records/econ_{sequence}` | 当前会话 | 本人的账本明细，不接受他人的账户查询参数 |

订单/商品审计不读取个人隐藏偏好进行过滤。玩家订单包含本人买入、卖出及其有权限管理的店铺订单；商品包含本人市场发布、创建的官方商品和其管理的店铺商品。筛选另一玩家不会使用该玩家的登录会话或授予其管理权限。全服资金列表只返回玩家账户的收支腿，不把 DIMA/DaoYu 托管内部账户混为玩家；玩家间转账会有一条支出和一条收入。

订单/商品按 `createdAt DESC,sequence_id DESC` 翻页；流水额外固定首查最大账本序号和时间区间。新条件重新查询；分页请求只带原 `cursor` 与 `limit`。流水默认最近一年，单次区间最多 366 天，每页最多 25 条以保持 Core 32 KiB 帧限制（客户端可请求 1–100，服务端最多返回 25）。订单/商品每页默认 25、最大 50。

## 永久删除公告

同一事务删除公告行（草稿和发布正文）、对应公告通知及图片业务绑定，把历史公告创建/编辑/发布幂等回执中的正文替换为 `{announcementId,deleted:true}`。保留仅含操作者、动作、编号、时间的审计及幂等指纹；重放旧创建请求不能恢复已删除正文。图片仍遵守共享用途和现有 GC 生命周期，不能删除其他业务正在使用的文件。未修改订单、财务账本或其他人的交易记录。

## SMTP 与送达

配置字段：`enabled,host,port,security,username,from,recipients`；保存还需 `clientRequestId,expectedVersion`。`security` 为 `TLS`（通常 465）或 `STARTTLS`（通常 587）；证书验证开启，不降级为明文认证。最多 10 个不同接收邮箱。`password` 只写入，省略表示保留，GET 仅返回 `passwordConfigured`。密码用 AES-256-GCM 加密，密钥来自服务器环境变量 `DEUTERIUM_SMTP_KEY`（32 个随机字节的 Base64）；密钥及密码不进入审计或响应。

案件创建、受理、资料更新、等待资料、裁决、资金处理完成/失败与撤回按案件编号、版本生成唯一队列事件，与案件状态同事务提交。没有配置或配置停用时保留队列。工作线程每 5 秒检查，发送超时 20 秒，领取租约 60 秒；失败从 30 秒退避到最多 1 小时，重启后继续。保存的错误仅为简短诊断，不含供应商回显的认证信息。

邮件只包含案件编号、状态、更新时间和管理页链接，不包含聊天或证据正文。SMTP 接受 DATA 后标为 `SENT`，不因 QUIT 失败重复发送；网络中断导致接收确认未知时会用固定 Message-ID 重试。SMTP 无法保证邮箱端严格恰好一次，队列保证同一业务事件只入队一次。`SENT` 表示邮件服务器已接受，最终是否进入收件箱仍由邮箱服务决定。

## 游戏经济只读协议

XConomy `2.26.3-deuterium.2` 保留原 `apiVersion()=1`，新增 `ledgerApiVersion()=1` 和 `records(Map<String,Object>)`。Core `1.0.1` 新增只读命令 `wallet.records`，只允许配置的经济权威节点执行。参数全部为字符串：

```json
{"playerUuid":"权威玩家 UUID；管理全服查询为空","from":"2026-09-08T00:00:00Z","to":"2026-09-09T00:00:00Z","direction":"","businessType":"","beforeTime":"","beforeSequence":"0","snapshot":"0","limit":"25","recordId":""}
```

返回 `{records,snapshot}`。每条含十进制字符串 `sequence`、`playerUuid,gameId,otherUuid,operationId,businessRef,businessType,source,direction,amount,beforeBalance,afterBalance,occurredAt`。余额与金额为精确两位小数字符串；数据来自经济库的已提交账本，零变化不产生可见收支，失败操作没有成功流水。查询不执行存取款，不通过余额差推断交易，不重新执行历史操作。

`source=GAME` 根据既有 `native.*` 操作或 `NATIVE_*` 账本分类识别，覆盖游戏 `/pay`、Vault、原生 API/管理命令；App/Web 统一显示“游戏内收入/游戏内支出”。其他业务归为 `TRANSFER/OFFICIAL_STORE/MARKET_ORDER/COMMISSION`，游戏内归为 `GAME`。经济库保留的历史原生流水无需重新导入；安装旧持久化扩展之前不存在于该账本的历史不能凭余额补造。

`wallet.balance` 增加 `today={income,expense,date,timeZone:'+08:00'}`，以北京时间自然日汇总同一账本。App 最近流水只取第一页；历史页按所选日期请求和加载更多，避免一年的大量游戏收支阻塞最近列表或被当日统计误截断。

## 升级与回退

后端新增 `020_admin_email_v204.sql` 两张配置/队列表，不更改旧迁移；XConomy 初始化按名称检测后新增 `(created_at,sequence_id)` 与 `(player_uuid,created_at,sequence_id)` 查询索引，不改旧经济记录。上线前备份业务库、经济库、服务环境与插件，保存邮件密钥。按既有经济插件流程统一停服替换 XConomy/Core，再部署迁移后的 Go、Web 和 App。

旧 Core 不提供新只读命令，因此不能先发布依赖它的新 App/Go。回退可保留新表与索引并切旧程序；不得恢复旧数据库覆盖新增真实业务。公告永久删除不能靠代码回退恢复。

[管理 OpenAPI](openapi-admin-v204.yaml)、[App 钱包契约](openapi-app-v2.yaml)、[实施与验收](../plans/admin-ledger-v204.md)。

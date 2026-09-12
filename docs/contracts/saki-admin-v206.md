# Saki、账号与头像增量接口

2026-09-12 增量候选：账号管理新增[管理员密码确认的永久注销](account-permanent-deletion.md)。已注销账号从本页账号列表排除，不能通过解封恢复；原有授权/封禁接口保持原格式。尚未部署。

2026-09-09，本机候选；与 [App OpenAPI](openapi-app-v2.yaml) 配合使用。所有路径前缀 `/api/v1`，沿用 App Bearer / Web Cookie、Origin 与 CSRF 校验。

| 接口 | 权限与内容 |
| --- | --- |
| `GET /admin/accounts?q=&status=&offset=0&limit=30` | 仅 `platform.admin`。返回 `data.items/total/offset/limit`；每项为 userId、playerRef、gameId、qq、status、admin、banReason、version、createdAt。状态筛选为空或 active/disabled/locked；limit 1–100。 |
| `POST /admin/accounts/{userId}` | 仅 `platform.admin`。输入 clientRequestId、expectedVersion、action、reason。action 为 grant-admin/revoke-admin/ban/unban；封禁须有原因。拒绝操作本人、过期版本及移除最后一个可用管理员。封禁、撤销管理权限都会撤销该账号现有会话；解封不恢复旧会话。 |
| `GET /admin/ai-settings` | 仅 `platform.admin`。返回 settings 与 providerConfigured；不返回服务密钥。 |
| `PUT /admin/ai-settings` | 输入 clientRequestId、expectedVersion、完整 settings；原子保存，冲突返回 409。保存结果为新 version。已存在套餐只能停用，不能删除其标识或改变 code。 |
| `GET /ai/plans` | 增加 version、purchasable。只有服务和全局购买开放、套餐启用时才允许购买。 |
| `GET /ai/me` | plan、quota 根据有效权益计算；增加 expiresAt（免费为 null）。maxInputChars 与 webSearchAvailable 来自当前设置。 |
| `POST /ai/purchases` | 输入 clientRequestId、planId、expectedPlanVersion。价格、有效期和额度全部从服务器当前套餐取值；版本变化须重新确认。返回商城创建结果 `data.operation/order`，201 为新订单，200 为已有结果，202 为处理中。 |
| `GET /ai/purchases/{purchaseId}` | purchaseId 就是 orderId；仅本人可查，返回 `data` 为 OrderView。原操作恢复使用 `/operations/{id}` 或 by-client-request 的 kind=AI_PURCHASE。关闭售卖不影响已受理交易的查询与恢复。 |

settings 包括 version、enabled、paidPlansEnabled、adminQuotaExempt、assistantName、model、systemPrompt、webSearch、reasoningEffort（none/low/high）、temperature（0–2）、maxInputChars（1–2000）、maxContextMessages（2–40）、maxOutputTokens（256–16384）、timeoutSeconds（5–300）、maxConcurrent（1–16）、knowledge、plans。提示词最多 32000 字；知识库最多 50 条，每条 id/title/content/enabled，单条内容最多 12000 字，总内容不超过 120000 UTF-8 字节。每个新回复持有独立设置快照，已有回复不中途改参数。上游地址和凭据仍由服务器私有配置提供。

套餐沿用 AiPlan 字段：planId、code、name、description、price（信用点十进制字符串）、quotaPerWindow、windowHours、durationDays、active；version 为只读。保留唯一 plan_free/code=free、价格 0.00、durationDays=0 且启用；付费有效期 1–3650 天，额度 1–100000 次/1–168 小时。价格不能超过既有经济接口的单笔限额。

Saki 订单 channel=OFFICIAL_STORE，orderType=AI_SUBSCRIPTION，seller.displayName=Saki AI，seller.storeId=saki-ai，delivery.method=DIGITAL。aiPlan 保存成交套餐，aiExpiresAt 保存该次开通后的到期时间。无邮件、领取、退款或用户结算动作。两段经济步骤仍是既有托管与结算；完整结算凭据确认后，权益与订单状态同事务落库。未知结果只查询原步骤；明确结算失败后退回托管资金，不开通权益。自动退回若明确失败，会保留资金记录并提醒联系平台处理。

同套餐续购从当前到期时间累计天数，不自动续费；有效期内跨套餐购买返回 AI_PLAN_ACTIVE，避免覆盖剩余权益。到期自动使用免费计划。旧订单的价格和权益快照不受后台编辑影响。管理员额度豁免仍由后端权限决定。

商店头像继续使用 logoAssetId。创建前上传使用 `POST /assets/uploads`，purpose=STORE_MEDIA、businessType=STORE、businessRef 为空；仅平台管理员允许。已有店铺提供实际 storeId，沿用店铺管理权限。前端裁切为 512×512 PNG 后上传，再将 assetId 随创建/编辑商店保存；取消保存的临时图片沿用已有孤立素材生命周期。订单的 seller.avatar 返回当前商店头像，已有订单同样能显示；财务和成交约定不变。

通知增加 systemPush；false 仅用于站内列表。官方订单只保留到货、退款或异常等有用结果，普通到货与退款不触发系统提醒，领取不重复提醒。内部步骤仍留在交易审计，旧冗余通知改为不显示，不删除账务数据。App 转账通知与成功回执同事务；游戏 `/pay` 通过已提交经济流水分页补取，持久游标和事件键保证重启、重叠查询不重复提醒，只通知已注册收款账号。系统显示仍沿用现有 App 通知权限与进程内收件箱同步，不引入新的离线推送服务。

迁移 `021_saki_admin.sql` 新增管理版本/原因、设置、权益、通知显示策略和流水游标。上线前备份数据库；已经有 Saki 订单后，不能直接退回不识别数字订单的旧后端。回退应保留数字订单的退款拒绝及原操作查询，不能恢复旧 SQL 覆盖新增业务。

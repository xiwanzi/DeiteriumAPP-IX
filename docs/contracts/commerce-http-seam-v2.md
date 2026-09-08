# Commerce HTTP 装配接缝

`internal/httpapi/commerce_v2_routes.go` 提供 `registerCommerceV2(mux)`；`store/commerce_v2*.go` 保存事务状态，`httpapi/commerce_v2_runner.go` 恢复执行及到期任务。2026-09-08 已完成本机 MariaDB、并发与 HTTP 契约验收，见 [验证记录](../qa/commerce-v2-2026-09-08.md)。013 迁移已冻结；本文不代表已经部署或实服交易验收完成。

所有路径加 `/api/v1`。先 `authenticate`，再解析和校验正文；写 Cookie 会话由 authenticate 校验 Origin/CSRF。`catalogReadV2` 可复用（512 KiB、32 层、重复键拒绝）。成功用 v2Success/v2List，错误优先 `catalogFailV2`（它映射 CatalogErrorV2 和资产错误），其余走既有 failError；不得把错误包装成成功订单。

## Runner 与通用输出

- `RunCommerceOperationV2(ctx, operationID string, execute bool) (store.CommerceOperationV2,error)`：POST 的 Prepare 返回 OperationID 后用 execute=true；GET 原操作用 false，只查询/吸收已认证 Mail 事实，不发送新步骤。runner 会保留原步骤 key/actor/Core ID，校验证明后才落业务结果。
- `StartCommerceV2()`：main 在配置执行器后调用一次，服务关闭时跟随 Server.ctx 退出。恢复原操作、吸收邮箱回执并补算到期确认。
- 所有 Prepare/Action 返回 `CommerceMutationV2{ResourceID,Kind,OperationID,Replayed}`。Kind 为 ORDER/COMMISSION；介入方法的 Kind=INTERVENTION、ResourceID=caseId，而其 OperationID 所属的底层交易仍从 operation.resourceId 读取。
- `Store.CommerceViewV2(ctx,viewerID,resourceID,public bool)` 返回最终 OrderView/CommissionView；`CommerceSnapshotV2` 返回不可变快照；`CommerceRefundViewV2(ctx,viewerID,resourceID,refundID)` 返回嵌套退款详情。
- 购买/委托发布，以及订单确认、委托确认/取消返回 `{operation,order}` 或 `{operation,commission}`。创建 COMPLETED 用 201，PROCESSING/UNKNOWN 用 202；其他操作的包装响应使用 200。即使 HTTP 成功也必须检查 operation.status；资金未知不可显示付款成功。
- `CommerceOperationViewV2(op,resourceKind)` 返回 OperationLookup；`Store.CommerceVisibleOperationV2(ctx,viewer,id,key,kind)` 先检索并校验参与者权限，返回 operation+resource。按 ID 时 key/kind 传空；按 key 时 id 传空。
- View 新增 `interventionCaseId`、`pendingOperationId`（string|null）；UNPAID 仅用于已证明预付失败、没有扣款，不能写成 REFUNDED。COMMISSION_ACCEPT 是一次受益人绑定操作种类。
- Order/Commission 返回 `completionDescription`、`completionAssetIds`，以及经已鉴权业务绑定签名的 `completionAssets`。InterventionView 同时返回 `evidenceAssets`、`evidenceEntries`、冻结交易对应的 `completionAssets`、当前 `decision` 和 `pendingOperationId`；临时图片地址不参与冻结证据摘要。

## 路由、正文和 Store 入口

以下 `actor` 为当前 session.User.ID，`kind` 为 ORDER 或 COMMISSION，`expected` 是正文对应版本。时间传 `time.Now().UTC()`，自动动作由 runner 调度。

| 路由 | 正文与入口 |
| --- | --- |
| POST `/store/orders`、`/market/orders` | `CommerceCreateOrderInputV2(input)` 得 key/quoteId/expectedQuoteVersion。调用 `PrepareOrderV2(ctx,actor,key,quoteID,expected,channel,available,nodes)`，channel 固定 OFFICIAL_STORE/PLAYER_MARKET；nodes 从 Config.Nodes 映射 `CatalogNodePolicyV2{InventoryDomain,ClaimEnabled}`。官方需要 reserve + mailbox.create + mailbox.revoke 能力，市场需要 reserve。 |
| POST `/commissions` | 复用 `CatalogMutationInputV2(input,true,false,"")` 取 key/content，再 `ValidateCommissionContentV2`。`PrepareCommissionV2(ctx,actor,key,content,availableReserve)`。 |
| POST `/commissions/{id}/accept` | MutationRequest；`PrepareCommissionAcceptV2(ctx,actor,id,key,expected,availableBind)`；直接 CommissionView。bind 未确认时 ACTIVE + UNKNOWN + pendingOperationId，不能开放完成或重新接取。 |
| POST `/orders/{id}/ship`、`/start-work` | MutationRequest；`CommerceFulfillmentV2(ctx,actor,id,"ORDER",key,action,expected,input)`，action=ship/start-work。直接 OrderView。 |
| POST `/orders/{id}/complete-work`、`/commissions/{id}/complete` | MutationRequest + description(2–500)、evidenceAssetIds(0–5)；同 CommerceFulfillmentV2，action=complete-work/complete。直接 View。 |
| POST `/{orders|commissions}/{id}/confirm` | MutationRequest；`PrepareCommerceSettlementV2(ctx,actor,id,kind,key,expected,availableSettle,false,now)`，包装 `{operation,order|commission}`。 |
| POST `/commissions/{id}/cancel` | MutationRequest；`PrepareCommissionCancelV2(ctx,actor,id,key,expected,availableRefund)`，包装返回。 |
| POST `/{orders|commissions}/{id}/refunds` | RefundRequest：clientRequestId、**父交易 expectedVersion**、reasonCode、description(2–500)、evidenceAssetIds(0–5)。`PrepareCommerceRefundV2(ctx,actor,id,kind,key,expected,input,available)`；available 为 refund（官方另须 revoke），普通已发货退款请求没有立即发钱，不因资金服务离线阻止提交。直接 View。 |
| POST 同退款 `/{refundId}/withdraw` | MutationRequest 的 expectedVersion 为 **refund.version**。`WithdrawCommerceRefundV2(ctx,actor,id,kind,refundID,key,expected)`；直接 View。 |
| POST 同退款 `/{refundId}/resolve` | RefundResolutionRequest：clientRequestId、refund.version、decision(APPROVE/REJECT)、reason（拒绝至少 2 字）。`ResolveCommerceRefundV2(ctx,actor,id,kind,refundID,key,expected,decision,reason,availableRefund)`；直接 View。 |
| GET `/orders/{id}/mailbox` | 先取得有权查看的 OrderView；调用 runner 邮箱事实刷新（不重发创建），再返回 OrderView。未来统一 `RefreshCommerceMailboxV2(ctx,resourceID)` 入口。 |
| POST `/merchant/orders/{id}/delivery-retry` | ReasonedMutationRequest；`PrepareDeliveryRetryV2(ctx,actor,id,key,expected,reason,availableMailCreate)`；包装 `{operation,order}`。同 deliveryId/原快照，仅重试交付，绝不再 reserve。 |
| GET `/operations/{operationId}`、`/operations/by-client-request` | 后者 clientRequestId + kind；先 VisibleOperation，再 runner execute=false，最终 OperationLookup。 |

普通动作校验器 `CommerceActionInputV2(input,action)` 返回 key/expected/error；支持 ship/start-work/confirm/cancel/accept/withdraw（基础 MutationRequest）、complete/complete-work、refund、resolve-refund。delivery-retry 复用 ReasonedMutation 校验（reason 不能仅空白）。所有字段按 v2 明确校验，不忽略额外字段。

## 列表

`Store.CommerceListV2(ctx,viewer,CommerceFilterV2)` 返回记录；先取 limit+1，截断后逐条调用 CommerceViewV2。limit 默认 20、范围 1–100。

- `/orders`：Kind=ORDER、Public=false，channel、role(BUYER/SELLER；缺省两者)、status、hasRefund（可选 bool 指针）均需应用。
- `/commissions`：Kind=COMMISSION、Public=true，只允许 status=OPEN，urgency(NORMAL/SOON/URGENT)、q(<=80)；增量支持 sort=NEWEST/REWARD_DESC/REWARD_ASC，缺省 NEWEST。
- `/commissions/me`：Kind=COMMISSION、Public=false；role=PUBLISHER 映射 OWNER，WORKER 原样；status 按 CommissionView 枚举。缺省本人发布或接取。
- `/merchant/orders`：Kind=ORDER、StoreID 为 query.storeId，先做店铺 ORDER_MANAGE 权限；建议必填 storeId，避免误查询个人订单。status 应用。
- 游标使用 `CommerceCursorV2(scope,record,sort)` / `CommerceParseCursorV2(scope,cursor,sort)`，结果填 Filter.Before / BeforeAmount；scope 含 viewer、path、全部筛选。价格排序以金额+sequence 保持稳定，不用序号游标假装价格分页。
- Filter 字段：Kind/Channel/Role/State/Query/StoreID/Urgency/Sort string，Public bool，HasRefund *bool，Before int64，BeforeAmount string，Limit int。

## 介入

创建 `POST /orders/{id}/interventions` 或 `/commissions/{id}/interventions`：`CommerceInterventionInputV2(input,"create")` 校验 key/父 expectedVersion，调用 `CreateCommerceInterventionV2(ctx,actor,id,kind,key,expected,input)`；201 data InterventionView。普通介入只针对市场/委托已拒绝退款，每笔交易一个案件，已有案件从 interventionCaseId 恢复。

`GET /interventions/{caseId}` / `/admin/interventions/{caseId}`：`CommerceCaseViewV2(ctx,viewer,caseID,admin)`。

动作统一 `CommerceCaseActionV2(ctx,actor,caseID,key,action,expected,input,available)`，action 为 evidence/withdraw/assign/request-evidence/resolve；用 `CommerceInterventionInputV2(input,action)` 校验。admin 路由额外要求 `admin(r,"intervention.manage")`，store 再做角色/案件校验；resolve 的 available 为相关 refund/settle 命令能力。financial resolve 会返回 OperationID，先跑原 operation 后再读 CaseView；CaseView 保持 RESOLVING 直到全部资金步骤有证明。

`GET /admin/interventions`：`CommerceCaseListV2(ctx,viewer,CommerceCaseFilterV2)`，字段 State/TransactionID string、AssignedToMe bool、Before int64、Limit int；先 admin 权限。assignedToMe=true 限本人，false/缺省不限制；status、transactionId 不忽略。

资产权限桥 `CommerceCanUploadEvidenceV2(ctx,user,businessType,businessRef)` 已存在：ORDER/COMMISSION/INTERVENTION；在 013 应用后用于 DISPUTE_EVIDENCE。成交图通过受保护的快照绑定读取，不能把上传者身份交给客户端冒用。

`REQUIRE_MANUAL_RECOVERY` 只适用于已无平台在管冻结款的人工追回场景；仍 held 时返回 `HELD_FUNDS_REQUIRE_DISPOSITION`，必须选择退款、结算或继续履约。官方 Immediate 退款若撤回明确失败、随后已认证 CLAIMED 回执到达，在无进行中操作时将该退款标为 REJECTED，并允许原冻结款向 DIMA 结算；不得让失败退款永久阻塞结算。

`013_commerce_v2.sql` 使用 LF，冻结 SHA-256：`556e39d842517f2a36d90b744e534c8e94f41a91b9b627180ea078c8525da310`。后续结构变化使用新迁移，不修改已应用文件。

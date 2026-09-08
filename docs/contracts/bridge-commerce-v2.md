# 担保、商城邮箱与通知桥接增量契约

日期：2026-09-08。配套 `app-api-complete-v2.md`。**这些是正式接入所需的新能力，当前插件和后端并未因为本轮交付而具备这些能力。** 原账号、玩家身份、余额、转账、公共聊天的既有消息继续遵守 `app-backend-api-v1.md`。

## 1. 通道和权威

插件主动连接 `GET /bridge/plugin/ws`，使用独立插件 Bearer 凭据。该凭据不进入 Android、商家网页、管理网页或公开 API。公开层使用 playerRef，后端解析成已绑定的服务器 UUID 后才调用插件。

消息保留 v1 信封：

```json
{
  "type": "wallet.escrow.reserve.request",
  "messageId": "bridge_req_001",
  "replyTo": null,
  "sentAt": "2026-09-08T02:00:00Z",
  "payload": {
    "operationId": "operation_001",
    "idempotencyKey": "escrow-reserve-order-001",
    "businessType": "MARKET_ORDER",
    "businessRef": "order_001",
    "payerServerUuid": "00000000-0000-0000-0000-000000000001",
    "amount": "1800.00",
    "currency": "CREDIT"
  }
}
```

响应 type 将 request 改为 result，replyTo 指向原 messageId。operationId 是业务操作身份，messageId 只是某次网络消息身份；重连/重发可以换 messageId，但不能换 operationId 或 idempotencyKey。

## 2. 必需能力

| 消息类型 | payload 字段 | 规则 |
| --- | --- | --- |
| `wallet.escrow.reserve.request` | operationId、idempotencyKey、businessType、businessRef、payerServerUuid、amount、currency | 原子校验可用余额并预付。businessType 为 MARKET_ORDER 或 COMMISSION；同键不同主体/金额拒绝。 |
| `wallet.escrow.settle.request` | operationId、idempotencyKey、escrowRef、payeeServerUuid、amount、businessRef | 从实际剩余担保款结算，不能凭请求任意加钱。收款人必须与已锁定交易参与方一致。 |
| `wallet.escrow.refund.request` | operationId、idempotencyKey、escrowRef、amount、businessRef | 原路退回 reserve 时的付款人；请求没有可任意指定的新退款账户。 |
| `wallet.escrow.query.request` | operationId 或 escrowRef，至少其一 | 返回权威执行/资金状态，支持断线后对账。 |
| `mailbox.create.request` | operationId、deliveryId、orderId、recipientServerUuid、templateRef、quantity、snapshotSha256 | 只允许受控物品模板；同 deliveryId 不能重复发放，不能接受任意服务器命令。 |
| `mailbox.revoke.request` | operationId、deliveryId、orderId | 与领取互斥；已领取返回 ALREADY_CLAIMED，不得继续退款并凭空删除玩家物品。 |
| `mailbox.query.request` | deliveryId、orderId | 查询 CREATED、CLAIMED、REVOKED、FAILED、UNKNOWN。 |
| `mailbox.claimed.event` | eventId、deliveryId、orderId、recipientServerUuid、occurredAt | 游戏端产生的权威领取事件；后端去重并更新订单。 |

所有金额均为 decimal string、信用点 CREDIT。UUID 只存在于受控内部桥接，客户端不能直接填写或选择。quantity 为正整数，受批准模板和商品限购约束。

建议返回结构：

```json
{
  "type": "wallet.escrow.reserve.result",
  "messageId": "bridge_result_001",
  "replyTo": "bridge_req_001",
  "sentAt": "2026-09-08T02:00:01Z",
  "payload": {
    "operationId": "operation_001",
    "status": "COMPLETED",
    "escrowRef": "escrow_001",
    "reservedAmount": "1800.00",
    "remainingHeldAmount": "1800.00",
    "availableBalance": "8300.00",
    "refreshedAt": "2026-09-08T02:00:01Z",
    "error": null
  }
}
```

status 为 PROCESSING、COMPLETED、FAILED 或 UNKNOWN。失败 error 包含 code/message；不得回传数据库连接信息、密码或插件凭据。查询结果需包含实际已退、已结算和剩余担保额，金额总和必须与原预付一致。

## 3. 并发、超时和恢复

1. 后端在发送资金指令前持久化业务操作、付款人/收款人和金额，且校验账号、商品/委托版本、库存与动作权限。
2. 插件保存可恢复的业务执行身份，并与实际经济系统核对。必须先确认经济 API 能提供何种原子或可追溯能力，再实现上述保证；不能只加一个内存去重集合就称为防重复扣款。
3. 网络超时不证明经济操作失败。若不能证明是否执行，状态为 UNKNOWN，查原 operationId；禁止新建一次扣款或凭空补款。无法自动核实时进入人工对账。
4. 库存预留、接取/取消、手动确认/自动确认、退款/平台受理在后端按交易版本互斥处理。并发请求只有一个胜者，其他返回冲突或原结果。
5. 待处理退款和平台资金介入阻止自动结算。恢复时使用剩余时间，不能重置为完整 72 小时。
6. 正式自动确认由服务端定时任务执行，即使所有 App 都关闭也应可靠运行。本机体验版的退出后补算不代表已实现这一后台服务。
7. 库存和账本更新与通知采用可恢复的事务/投递机制，不能先通知成功再尝试落账。事件按 eventId 去重，允许重复投递但不重复产生业务效果。
8. 桥不可用时拒绝新的资金指令，不排队等待恢复后偷偷执行。已发出的未知操作恢复连接后先查询原执行结果，不能当作新支付重发。

## 4. 向 App 推送的增量事件

使用现有 `/api/v1/chat/ws` 登录连接，保留 type/requestId/sentAt/payload 信封。旧客户端忽略未知 type；网页管理端可先使用 HTTP 查询，不把 Bearer token 放入 WebSocket URL。

| type | payload | 接收范围 |
| --- | --- | --- |
| `order.updated` | eventId、resourceId（orderId）、resourceVersion、status、occurredAt | 仅买卖参与方；官方商城订单另限对应店铺有权商家，玩家市场订单不向普通商家广播。 |
| `commission.updated` | eventId、resourceId（commissionId）、resourceVersion、status、occurredAt | 仅发布者、接取者；授权后台另按案件/资源范围审计访问。大厅通过重新读取公开列表移除已接取委托，不向大厅广播私有履约事件。 |
| `notification.created` | eventId、notification（NotificationView） | notification 所属用户。 |
| `catalog.changed` | eventId、storeId、productIds、resourceVersion、occurredAt | 已登录商品浏览会话，不包含草稿或私密数据。 |
| `appearance.updated` | eventId、packageName、channel、scope、recordVersion | 全局变更仅对应包名/频道会话，玩家覆盖仅目标玩家；提示重新 GET /app/appearance，不广播玩家配置内容。 |
| `app.update.available` | eventId、packageName、channel、releaseIds、occurredAt | 与包名/频道匹配的会话；下载前仍 GET 更新清单。 |

App 收到事件后查询权威详情，不能直接根据通知摘要改余额。通知开关影响推送，不删除账单和交易记录。私聊只面向会话双方，不得经 Minecraft 公共频道泄漏；游戏公共聊天双向桥接继续沿用原协议。

## 5. 上线前需决策/配置

- 经济系统的预付、退款、担保释放、执行记录和人工对账方案。
- 游戏邮箱插件或受控发放模板的实现和领取/撤回互斥能力。
- 正式后台任务、通知适配器、服务域名和权限配置。

这份增量契约规定需要达到的业务保证，不替代生产实现 ADR、数据库设计和插件联调验收。

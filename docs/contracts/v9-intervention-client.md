# 体验版 09：平台介入客户端对接说明

日期：2026-09-08。用户最新决定覆盖 08 的“不在 App 新增介入表单”。本文件补充既有 v2 契约，不新增或替换后端接口。

市场订单和委托的退款被拒后，付款方可点击“申请平台介入”，填写原因、10–3000 字事实说明、诉求、部分退款金额及最多 10 张凭证图。已有案件显示进度或平台判决，不重新创建案件。提交前展示确认页和不可编辑的原交易、退款及拒绝记录；申请诉求不等于平台判决。

| 客户端动作 | 既有接口 |
| --- | --- |
| 订单申请 | `POST /api/v1/orders/{orderId}/interventions` |
| 委托申请 | `POST /api/v1/commissions/{commissionId}/interventions` |
| 查询本人参与的案件 | `GET /api/v1/interventions/{caseId}` |
| 管理员受理、要求补证、裁决 | `admin/interventions` 下的 `assign`、`request-evidence`、`resolve`，仅管理端授权调用 |

`InterventionRequest` 映射：`reasonCode` 为 `NOT_DELIVERED / NOT_AS_DESCRIBED / REFUND_DISAGREEMENT / OTHER`；`description` 为事实说明；`desiredResolution` 为 `FULL_REFUND / PARTIAL_REFUND / CONTINUE_FULFILLMENT / OTHER`；`requestedRefundAmount` 对应部分退款金额；`evidenceAssetIds` 对应上传完成后的资源 ID。正式请求以 `clientRequestId` 传递 UUID 幂等键，并提交最近读取的 `expectedVersion`；金额遵循 `CreditAmount` 十进制字符串（最多两位小数），本机最小单位整数必须精确转换，例如 `3500` → `"35.00"`，不经浮点数。09 本机图片路径不能直接当作服务端资源 ID，原交易、参与者、担保金额及快照须由服务端权威读取。

仍冻结的交易受理后暂停原剩余计时，禁止双方自行确认、退款或修改履约状态。平台全额退款、部分退款并结算剩余、全额结算三种资金结果互斥，执行一次后关闭担保；部分退款和结算之和等于案件担保金额。平台退款不自动推断实物已退回，不自动补库存。继续履约或不采取资金动作恢复原剩余时间。案件提交时已经结算的交易仅支持人工协商处理或不采取资金动作，不能重复退款或强制扣收款方余额。

09 已实现本机表单、申请材料保存、案件状态、暂停/恢复和裁决资金演示，与钱包、交易记录一起保存到版本 8 的原子本机快照，兼容旧版本 6–7。详情末尾“体验平台处理”仅用于模拟管理员动作，正式玩家端必须去除该工具，以服务端通知或查询驱动结果；不能让玩家调用 `resolve`。

本轮检查了 `backend-rewrite/backend-next` 和 `web-player-v1/web-app`：当时新增资金/案件业务仍在开发，网页演示不代表真实争议处理已接入。本 APK 不向真实管理端发送案件。正式鉴权、并发控制、资金编排、证据上传、补证/撤回及跨设备通知仍需按完整契约接通并联调；不把本机体验标记为已部署。

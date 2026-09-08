# Core / 后端的严格取消接入

2026-09-08。依赖独立 Mail 插件 `0.6.1`、`MailboxIntegration` API 2。Bridge/UI 的玩家协议保持不变，Core 对 Mail 仅使用 provided 公共 API，不打包或直连其存储。

`mailbox.revoke` 保留原 source / deliveryId / orderId / expectedSnapshotSha256 / reasonCode。交易服务新增可选 `snapshotJson`：必须是创建时保存的原始 JSON 字节，不能重排后复用原摘要。有此字段时调用 `revokeOrCancel(Cancel)`；无此字段时保持原 `revoke` 语义。

Core 先核对原字节摘要、订单、UUID、库存域与服务器范围；取消不解析物品库载荷，也不会为了退款创建一封可领取的邮件。返回 `CancellationReceipt`，含原 operationId、来源、交付/订单、摘要、收件 UUID、范围、`proofKind`、`mailReceipt`、`cancelledAt`（UNIX epoch 秒）和 replayed。

- `REVOKED_MAIL`：mailReceipt 必须是同一实际邮件的 REVOKED 回执，包含真实 mailId 与正 revision。
- `CANCELLED_BEFORE_CREATE`：不存在普通邮件回执；mailReceipt 为空（Core 的 JSON 序列化可能省略空字段），没有虚构 mailId。该证明表示 Mail 已持久关闭该交付键，迟到 create 不能再投递。

Core 本地日志在 Mail 提交后丢失回复时，`operation.query` 先以固定来源和原 operationId 调用只读 `queryCancellation`，包装为原操作的提交证明；不会重发取消或资金动作。没有已提交证明则仍返回原 UNKNOWN/NOT_FOUND，不能推断可退款。

Go 对实时回复与恢复回复使用相同验证：取消证明必须匹配原操作、原请求摘要、订单、收件人、范围与邮箱集群；只有真实 REVOKED_MAIL 会写入普通 `core_mail_receipts`，缺失取消仅保存在原 Core 操作回执中。官方新付款要求 `apiVersion >= 2`、`missingDeliveryCancellation=true` 和完整邮箱能力。

同批加强普通转账回执：COMPLETED 的 data 也必须匹配 operationId、fromUuid、toUuid、amount、CREDIT 币种、提交状态与 committedAt。网络传输成功不能替代这些字段。

验证涵盖快照改动/身份错配/额外字段拒绝、无需物品库即可构造取消请求、Core 崩溃后只读恢复、Go 恢复的错配证明拒绝、缺失取消不伪造普通邮件，以及 API 1 不开放新取消能力。独立 Mail 的真实数据库竞争与回滚验证见其 `docs/mailbox-cancellation-v2.md` 与相邻 QA。

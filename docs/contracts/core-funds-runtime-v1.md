# Core 持久化资金命令 v1

使用 `core.command` 信封的 `command`、`operationId`、`expiresAt`、`payload`。Go 业务端使用 `CoreCall` 持久化原操作，`QueryCoreOperation` 找回已知 ID，`CoreResult` 解码 RPC 终态。同 actor+clientRequestId 必须始终对应相同节点、动作和正文；即使节点离线，已持久结果也可查回。未发送命令可以在恢复连接后发送；UNKNOWN 不会自动换键执行。

资金执行要求配置单一 `economy` 权威节点，以及 XConomy `ControlledEconomyAPI` v1。普通 Vault 的同步返回不是资金持久化证明。XConomy 数据库须为 InnoDB、`innodb_flush_log_at_trx_commit=1`、连接池及两位小数模式。

| command | payload 字段 | 语义 |
| --- | --- | --- |
| wallet.balance | playerUuid | 已提交可用余额、heldAmount、heldBreakdown 与 refreshedAt |
| wallet.transfer | fromUuid、toUuid、amount | 普通玩家间两边余额及流水同事务提交 |
| wallet.escrow.reserve | escrowRef、businessRef、businessType、payerUuid、payeeUuid（可选）、amount、currency | 付款人 → DaoYu，并冻结交易快照 |
| wallet.escrow.bind | escrowRef、businessRef、payeeUuid | 仅委托可后置绑定一次；重复同一收款人可核实，不可改为别人 |
| wallet.escrow.settle | escrowRef、businessRef、amount、currency | DaoYu → 冻结收款人，不接受新收款人参数 |
| wallet.escrow.refund | escrowRef、businessRef、amount、currency | DaoYu → 原付款人 |
| wallet.escrow.query | escrowRef、businessRef | 只读当前担保记录 |
| operation.query | operationId | 原执行记录；Core 本地结果未知时读取 XConomy 的提交回执 |

`currency` 固定 CREDIT。金额是最多两位小数的十进制字符串，正数，单笔与单账号余额最大 1,000,000,000,000，同时遵守更低的 XConomy 配置上限。

`businessType` 为 OFFICIAL_STORE、MARKET_ORDER 或 COMMISSION。官方商店 payee 固定 DIMA；市场必须传入被冻结卖家 UUID；委托发布时可不传收款人，接取后 bind 一次。客户端不提交 UUID：由 Go 根据已绑定身份、业务冻结参与者或服务器已解析目录取得。

成功的资金 Data 包含：operationId、businessRef、escrowRef（担保动作）、businessType、payerUuid、payeeUuid（冻结收款人）、fromUuid/toUuid（本动作真实余额流向）、amount、currency、reservedAmount/settledAmount/refundedAmount/heldAmount、escrowStatus、status=COMPLETED、committedAt。普通转账没有担保累计字段。业务端必须核对引用、主体、金额和动作证据后再推进订单，不能只看外层 RPC COMPLETED。

refund 的冻结 payeeUuid 仍可为原商家/接取人，真实退款目标使用 toUuid，必须等于 payerUuid。query 的 amount 为 0.00，仅表明当前余额状态，不能充当某一次结算的提交证明；确认具体动作应查询原 operationId。

部分结算、退款分别有独立且稳定的业务动作键；累计之和不得超出预付金额。资金查询发现 UNKNOWN 时不能直接认定失败并退款，必须先找回原提交记录。明确失败的操作重放仍失败，不能把原错误请求换成一次隐式付款。

Core 管理命令 `/dc economy init-system-accounts` 仅在权威节点的可信 Console/RCON 执行，新建零余额 DIMA、DaoYu 并持久化 UUID；名称冲突时拒绝接管。RCON 先得到受理消息，最终身份信息写到服务器日志。它们没有 App/QQ/密码身份，不能通过普通游戏登录、Vault 存取、支付或删除路径控制。

常见拒绝：INSUFFICIENT_BALANCE、MAX_BALANCE_EXCEEDED、SYSTEM_ACCOUNT_PROTECTED、SYSTEM_ACCOUNTS_NOT_INITIALIZED、ESCROW_CONFLICT、ESCROW_ALREADY_RESERVED、PAYEE_LOCKED、PAYEE_NOT_BOUND、ESCROW_AMOUNT_EXCEEDED、ESCROW_LEDGER_MISMATCH、IDEMPOTENCY_CONFLICT。基础设施不确定使用 RESULT_UNKNOWN/STORAGE_UNAVAILABLE/ECONOMY_UNAVAILABLE；不得据此推断未扣款。

独立邮箱命令继续使用 Mail API v1 的 source/deliveryId/orderId/冻结 snapshotJson 和 snapshotSha256。邮箱 RPC 的 Data.code 也必须为成功并有匹配 receipt；传输成功不等于创建或撤销成功。

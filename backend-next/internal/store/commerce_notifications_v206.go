package store

import (
	"context"
	"database/sql"
	"strings"
)

// Only customer milestones belong in the inbox. Execution details remain in
// commerce_events_v2 for audit and recovery.
func commerceNoticeV206(d CommerceRecordV2, kind, actor, viewer string) (title, body, key string, push bool) {
	if d.Channel == "OFFICIAL_STORE" {
		if viewer != d.OwnerID || d.PendingOperationID != "" {
			return
		}
		switch {
		case IsAIOrderV206(d) && d.FundsState == "SETTLED":
			title = "Saki AI 已开通"
			body = "套餐已生效，现在就可以与小祥聊天。"
			key = "activated"
		case d.State == "AWAITING_CLAIM":
			title = "商品已送达"
			body = "请在游戏邮箱领取商品。领取前可在订单中申请退款。"
			key = "ready"
		case d.FundsState == "REFUNDED":
			title = "信用点已退回"
			body = "这笔订单的信用点已退回钱包。"
			key = "refunded"
		case d.FundsState == "UNPAID":
			title = "购买未完成"
			body = "本次未扣款，可以稍后重新购买。"
			key = "unpaid"
		case d.State == "PAYMENT_PROCESSING" && d.FundsState == "HELD":
			title = "订单需要处理"
			body = "购买暂未完成，请联系平台客服协助处理。"
			key = "help"
			push = true
		}
		return
	}
	if strings.HasPrefix(kind, "operation.") {
		if d.PendingOperationID != "" {
			return
		}
		switch {
		case d.FundsState == "REFUNDED":
			title = "信用点已退回"
			body = "退款已完成，可在钱包查看。"
			key = "refunded"
			push = viewer == d.OwnerID
		case d.FundsState == "SETTLED":
			title = "交易已完成"
			body = "信用点已结算，可在钱包查看。"
			key = "settled"
			push = viewer == d.PayeeID
		case d.Kind == "ORDER" && d.State == "AWAITING_SHIPMENT" && viewer == d.PayeeID:
			title = "收到新订单"
			body = "买家已付款，请及时安排交付。"
			key = "paid"
			push = true
		case d.Kind == "COMMISSION" && d.State == "ACTIVE" && viewer == d.OwnerID:
			title = "委托已被接取"
			body = "已有玩家接取你的委托，可在详情中联系对方。"
			key = "accepted"
			push = true
		}
		return
	}
	if actor == viewer {
		return
	}
	switch kind {
	case "ORDER.ship":
		title = "卖家已发货"
		body = "请确认商品是否收到，并在订单中确认收货。"
	case "ORDER.start-work":
		title = "工程已开始"
		body = "卖家已开始施工，可在订单中查看进展。"
	case "ORDER.complete-work":
		title = "工程等待验收"
		body = "卖家已提交完成，请及时查看并验收。"
	case "COMMISSION.complete":
		title = "委托等待验收"
		body = "对方已提交完成，请在 72 小时内查看并确认。"
	case "refund.requested":
		title = "收到退款申请"
		body = "对方申请退款，请查看原因并处理。"
	case "refund.resolved":
		title = "退款申请有新进展"
		body = "对方已处理退款申请，请查看结果。"
	case "refund.withdrawn":
		title = "退款申请已撤回"
		body = "对方已撤回退款申请，交易继续进行。"
	case "intervention.request-evidence":
		title = "请补充说明"
		body = "平台需要你补充相关材料，请查看案件详情。"
	case "intervention.resolve":
		title = "平台处理结果已出"
		body = "请查看平台给出的处理结果。"
	}
	if title != "" {
		key = kind
		push = true
	}
	return
}

func customerCommerceTextV206(text string) string {
	return strings.NewReplacer(
		"接取名额已锁定，正在确认唯一受益人。", "正在接取委托，请稍候。",
		"原资金或交付步骤已由服务端证明确认。", "交易进度已更新，可查看详情。",
		"原操作结果仍待核实，不会换标识重复执行。", "交易仍在处理中，请稍后查看。",
		"原操作明确未完成，请查看交易当前状态后处理。", "操作未完成，请查看订单详情。",
		"游戏邮箱状态已按原订单回执同步。", "游戏邮箱中的商品状态已更新。",
		"平台介入已提交，原案件引用已同步到交易。", "平台已收到你的申请，请等待处理。",
		"平台案件状态已更新，请查看原案件详情。", "平台处理有新进展，请查看详情。",
		"退款申请已撤回，继续原剩余期限，退款次数不恢复。", "退款申请已撤回，交易继续进行。",
	).Replace(text)
}

func walletTransferNoticeV206(ctx context.Context, tx *sql.Tx, operation string) error {
	var id, user, sender, amount string
	err := tx.QueryRowContext(ctx, `SELECT t.transfer_id,receiver.id,sender.game_id,t.amount FROM wallet_transfers_next t JOIN identities receiver ON receiver.server_uuid=t.to_uuid AND receiver.status='active' JOIN identities sender ON sender.id=t.actor_id WHERE t.operation_id=?`, operation).Scan(&id, &user, &sender, &amount)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return AddNotificationV2(ctx, tx, user, "transfer:"+id, "WALLET", "转账到账", sender+" 向你转账 "+amount+" 信用点，已到账。", SocialTarget{Kind: "WALLET", ReferenceID: id, StateVersion: 1})
}

package store

import (
	"context"
	"database/sql"
	"time"
)

func commerceRefundStepsV2(d CommerceRecordV2, amount string) ([]CommerceStepV2, error) {
	steps := []CommerceStepV2{}
	if d.Channel == "OFFICIAL_STORE" {
		plan, ok := catalogObject(d.Body["mailboxPlan"])
		if !ok {
			return nil, catalogError(503, "DELIVERY_SNAPSHOT_MISSING", "原交付快照需要核对。")
		}
		steps = append(steps, CommerceStepV2{Command: "mailbox.revoke", Payload: map[string]any{"source": "deuterium-commerce", "deliveryId": plan["deliveryId"], "orderId": d.ID, "expectedSnapshotSha256": plan["snapshotSha256"], "snapshotJson": plan["snapshotJson"], "reasonCode": "CUSTOMER_REFUND"}})
	}
	steps = append(steps, commerceMoneyStepV2(d, "wallet.escrow.refund", amount))
	return steps, nil
}
func (s *Store) PrepareCommerceRefundV2(ctx context.Context, actor, id, kind, key string, expected int64, input CatalogObjectV2, available bool) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, kind+":refund:"+id, input, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		d, e := commerceRecordTxV2(ctx, tx, id, kind, true)
		if e != nil {
			return result, e
		}
		if IsAIOrderV206(d) {
			return result, catalogError(409, "AI_NON_REFUNDABLE", "Saki AI 套餐为即时开通服务，不支持退款。")
		}
		if d.OwnerID != actor {
			return result, catalogDenied()
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if e = commerceGuardHeldV2(d); e != nil {
			return result, e
		}
		active, e := commerceCaseActiveTxV2(ctx, tx, d)
		if e != nil {
			return result, e
		}
		if active {
			return result, catalogError(409, "INTERVENTION_ACTIVE", "请在原平台案件中继续处理。")
		}
		remaining, e := commerceRemainingV2(d)
		if e != nil {
			return result, e
		}
		if remaining.Sign() <= 0 {
			return result, catalogError(409, "NO_HELD_FUNDS", "交易已经完成结算。")
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		amount := catalogMoneyString(remaining)
		immediate := d.Kind == "ORDER" && (d.State == "AWAITING_SHIPMENT" || d.Channel == "OFFICIAL_STORE" && (d.State == "AWAITING_CLAIM" || d.State == "PAYMENT_PROCESSING"))
		eligible := immediate || d.Kind == "ORDER" && (d.State == "SHIPPED" || d.State == "WORK_COMPLETED") || d.Kind == "COMMISSION" && (d.State == "ACTIVE" || d.State == "COMPLETED")
		if !eligible {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "当前交易不能申请退款。")
		}
		var refund CommerceRefundV2
		retry := false
		if d.RefundID != "" {
			refund, e = commerceRefundTxV2(ctx, tx, d.RefundID, true)
			if e != nil {
				return result, e
			}
			retry = refund.Immediate && refund.State == "PROCESSING" && immediate
		}
		if d.RefundAttempts != 0 && !retry {
			return result, catalogError(409, "REFUND_ATTEMPT_USED", "这笔交易的退款机会已使用，拒绝或撤回不会恢复。")
		}
		if immediate {
			if e = commerceAvailableV2(available); e != nil {
				return result, e
			}
		}
		if !retry {
			refund = CommerceRefundV2{ID: ID("refund_"), ResourceID: d.ID, Version: 1, State: "REQUESTED", ReasonCode: catalogString(input, "reasonCode"), Description: catalogString(input, "description"), Evidence: catalogIDs(input, "evidenceAssetIds"), Amount: amount, Immediate: immediate, RequestedAt: now}
			if immediate {
				refund.State = "PROCESSING"
			}
			if e = s.commerceBindEvidenceV2(ctx, tx, actor, "REFUND", refund.ID, refund.Evidence); e != nil {
				return result, e
			}
			if e = commerceSaveRefundV2(ctx, tx, refund, true); e != nil {
				return result, e
			}
			d.RefundID = refund.ID
			d.RefundAttempts = 1
			commercePauseV2(&d, now)
		} else {
			refund.Version++
			if e = commerceSaveRefundV2(ctx, tx, refund, false); e != nil {
				return result, e
			}
		}
		var operationID string
		if immediate {
			steps, e := commerceRefundStepsV2(d, amount)
			if e != nil {
				return result, e
			}
			op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "REFUND", "refund", amount, steps, false, now)
			if e != nil {
				return result, e
			}
			operationID = op.ID
		}
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		summary := "已提交唯一一次退款申请，确认期限暂停。"
		if immediate {
			summary = "已提交原路退款，游戏邮箱交付会先核实严格撤回。"
		}
		if e = commerceEventV2(ctx, tx, d, actor, "refund.requested", summary, input); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: operationID}, nil
	})
}
func (s *Store) ResolveCommerceRefundV2(ctx context.Context, actor, id, kind, refundID, key string, expected int64, decision, reason string, available bool) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, "refund.resolve:"+refundID, map[string]any{"expectedVersion": expected, "decision": decision, "reason": reason}, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		d, e := commerceRecordTxV2(ctx, tx, id, kind, true)
		if e != nil {
			return result, e
		}
		if actor != d.PayeeID || d.Channel == "OFFICIAL_STORE" {
			return result, catalogDenied()
		}
		if d.PendingOperationID != "" {
			return result, catalogError(409, "OPERATION_IN_PROGRESS", "原资金操作尚未确认。")
		}
		if e = commerceGuardHeldV2(d); e != nil {
			return result, e
		}
		active, e := commerceCaseActiveTxV2(ctx, tx, d)
		if e != nil {
			return result, e
		}
		if active {
			return result, catalogError(409, "INTERVENTION_ACTIVE", "案件处理期间不能自行退款。")
		}
		if d.RefundID != refundID {
			return result, catalogNotFound()
		}
		refund, e := commerceRefundTxV2(ctx, tx, refundID, true)
		if e != nil {
			return result, e
		}
		if refund.ResourceID != d.ID || refund.Version != expected || refund.Version >= 2147483647 {
			return result, catalogVersion()
		}
		if refund.State != "REQUESTED" {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "退款申请当前不能处理。")
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		var opID string
		if decision == "REJECT" {
			if !catalogText(reason, 2, 500) {
				return result, catalogInvalid()
			}
			refund.State = "REJECTED"
			refund.RejectionReason = reason
			refund.ResolvedAt = &now
			commerceResumeV2(&d, now)
		} else if decision == "APPROVE" {
			if e = commerceAvailableV2(available); e != nil {
				return result, e
			}
			remaining, e := commerceRemainingV2(d)
			if e != nil {
				return result, e
			}
			amount := catalogMoneyString(remaining)
			if remaining.Sign() <= 0 {
				return result, catalogError(409, "NO_HELD_FUNDS", "交易已无可退款资金。")
			}
			refund.State = "PROCESSING"
			steps, e := commerceRefundStepsV2(d, amount)
			if e != nil {
				return result, e
			}
			op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "REFUND", "refund", amount, steps, false, now)
			if e != nil {
				return result, e
			}
			opID = op.ID
		} else {
			return result, catalogInvalid()
		}
		refund.Version++
		commerceTouchV2(&d, now)
		if e = commerceSaveRefundV2(ctx, tx, refund, false); e != nil {
			return result, e
		}
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		summary := "退款已同意，正在核对原路返款。"
		if decision == "REJECT" {
			summary = "退款被拒绝：" + reason
		}
		if e = commerceEventV2(ctx, tx, d, actor, "refund.resolved", summary, map[string]string{"decision": decision, "reason": reason}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: opID}, nil
	})
}
func (s *Store) WithdrawCommerceRefundV2(ctx context.Context, actor, id, kind, refundID, key string, expected int64) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, "refund.withdraw:"+refundID, map[string]any{"expectedVersion": expected}, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		d, e := commerceRecordTxV2(ctx, tx, id, kind, true)
		if e != nil {
			return result, e
		}
		if d.OwnerID != actor {
			return result, catalogDenied()
		}
		if d.PendingOperationID != "" || d.RefundID != refundID {
			return result, catalogError(409, "OPERATION_IN_PROGRESS", "原退款正在处理。")
		}
		if e = commerceGuardHeldV2(d); e != nil {
			return result, e
		}
		active, e := commerceCaseActiveTxV2(ctx, tx, d)
		if e != nil {
			return result, e
		}
		if active {
			return result, catalogError(409, "INTERVENTION_ACTIVE", "案件处理期间不能自行撤回退款。")
		}
		refund, e := commerceRefundTxV2(ctx, tx, refundID, true)
		if e != nil {
			return result, e
		}
		if refund.Version != expected || refund.ResourceID != d.ID || refund.Version >= 2147483647 {
			return result, catalogVersion()
		}
		if refund.State != "REQUESTED" {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "只有待处理的退款可以撤回。")
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		refund.State = "WITHDRAWN"
		refund.Version++
		refund.ResolvedAt = &now
		commerceResumeV2(&d, now)
		commerceTouchV2(&d, now)
		if e = commerceSaveRefundV2(ctx, tx, refund, false); e != nil {
			return result, e
		}
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "refund.withdrawn", "退款申请已撤回，继续原剩余期限，退款次数不恢复。", map[string]string{"refundId": refundID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind}, nil
	})
}

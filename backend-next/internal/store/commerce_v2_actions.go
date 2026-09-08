package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) commerceBindEvidenceV2(ctx context.Context, tx *sql.Tx, actor, businessType, reference string, ids []string) error {
	for _, id := range ids {
		var purpose string
		if e := tx.QueryRowContext(ctx, "SELECT purpose FROM asset_uploads_v2 WHERE asset_id=?", id).Scan(&purpose); e != nil {
			return e
		}
		if purpose != "DISPUTE_EVIDENCE" {
			return ErrAssetUnavailable
		}
	}
	return s.SetAssetBindingsV2(ctx, tx, actor, businessType, reference, ids)
}
func (s *Store) CommerceFulfillmentV2(ctx context.Context, actor, id, kind, key, action string, expected int64, input CatalogObjectV2) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, kind+":"+action+":"+id, input, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		d, e := commerceRecordTxV2(ctx, tx, id, kind, true)
		if e != nil {
			return result, e
		}
		if e = commercePermissionTxV2(ctx, tx, actor, d, false); e != nil {
			return result, e
		}
		if actor != d.PayeeID {
			return result, catalogDenied()
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if e = commerceRequireFulfillmentV2(ctx, tx, d); e != nil {
			return result, e
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		construction, _ := d.Body["construction"].(bool)
		summary := ""
		switch action {
		case "ship", "start-work":
			if d.Kind != "ORDER" || d.Channel != "PLAYER_MARKET" || d.State != "AWAITING_SHIPMENT" || (action == "ship" && construction) || (action == "start-work" && !construction) {
				return result, catalogError(409, "INVALID_STATE_TRANSITION", "当前订单不能执行此交付动作。")
			}
			d.State = "SHIPPED"
			d.Body["shippedAt"] = now
			hours := int64(72)
			summary = "卖家已确认发货，开始 72 小时确认期。"
			if construction {
				hours = catalogNumber(d.Body, "confirmationHours")
				summary = "卖家已开始施工，总工期包含验收预留。"
			}
			commerceSetDeadlineV2(&d, "ORDER_CONFIRM", now.Add(time.Duration(hours)*time.Hour))
			if e = commerceFinalizeStockV2(ctx, tx, d.ID); e != nil {
				return result, e
			}
		case "complete-work":
			if d.Kind != "ORDER" || !construction || d.State != "SHIPPED" {
				return result, catalogError(409, "INVALID_STATE_TRANSITION", "工程尚未开始或已经提交完成。")
			}
			d.State = "WORK_COMPLETED"
			d.Body["workCompletedAt"] = now
			d.Body["completionDescription"] = input["description"]
			d.Body["completionAssetIds"] = catalogIDs(input, "evidenceAssetIds")
			if e = s.commerceBindEvidenceV2(ctx, tx, actor, "ORDER_COMPLETION", d.ID, catalogIDs(input, "evidenceAssetIds")); e != nil {
				return result, e
			}
			summary = "工程已提交完成，原总工期不会重新计算。"
		case "complete":
			if d.Kind != "COMMISSION" || d.State != "ACTIVE" {
				return result, catalogError(409, "INVALID_STATE_TRANSITION", "委托当前不能提交完成。")
			}
			d.State = "COMPLETED"
			d.Body["completedAt"] = now
			d.Body["completionDescription"] = input["description"]
			d.Body["completionAssetIds"] = catalogIDs(input, "evidenceAssetIds")
			commerceSetDeadlineV2(&d, "ACCEPTANCE", now.Add(72*time.Hour))
			if e = s.commerceBindEvidenceV2(ctx, tx, actor, "COMMISSION_COMPLETION", d.ID, catalogIDs(input, "evidenceAssetIds")); e != nil {
				return result, e
			}
			summary = "接取者已提交完成，开始独立 72 小时验收期。"
		default:
			return result, catalogInvalid()
		}
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, kind+"."+action, summary, input); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind}, nil
	})
}
func (s *Store) PrepareCommerceSettlementV2(ctx context.Context, actor, id, kind, key string, expected int64, available, automatic bool, now time.Time) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, kind+":confirm:"+id, map[string]any{"expectedVersion": expected, "automatic": automatic}, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		if e := commerceAvailableV2(available); e != nil {
			return result, e
		}
		d, e := commerceRecordTxV2(ctx, tx, id, kind, true)
		if e != nil {
			return result, e
		}
		if actor != d.OwnerID {
			return result, catalogDenied()
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if e = commerceRequireFulfillmentV2(ctx, tx, d); e != nil {
			return result, e
		}
		construction, _ := d.Body["construction"].(bool)
		eligible := d.Kind == "COMMISSION" && d.State == "COMPLETED" || d.Kind == "ORDER" && d.Channel == "PLAYER_MARKET" && ((!construction && d.State == "SHIPPED") || (construction && d.State == "WORK_COMPLETED")) || d.Kind == "ORDER" && d.Channel == "OFFICIAL_STORE" && d.State == "CLAIMED"
		if !eligible {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "尚未完成交付，不能确认结算。")
		}
		if d.PayeeUUID == "" {
			return result, catalogError(409, "BENEFICIARY_UNCONFIRMED", "原受益人尚未确认。")
		}
		if automatic && d.Channel != "OFFICIAL_STORE" && (d.Deadline == nil || d.Deadline.After(now)) {
			return result, catalogError(409, "DEADLINE_NOT_REACHED", "尚未到自动确认时间。")
		}
		remaining, e := commerceRemainingV2(d)
		if e != nil {
			return result, e
		}
		if remaining.Sign() <= 0 {
			return result, catalogError(409, "NO_HELD_FUNDS", "交易没有可结算的冻结款。")
		}
		amount := catalogMoneyString(remaining)
		op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "SETTLEMENT", "settle", amount, []CommerceStepV2{commerceMoneyStepV2(d, "wallet.escrow.settle", amount)}, automatic, now)
		if e != nil {
			return result, e
		}
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		summary := "已记录确认，正在核对原冻结款结算。"
		if automatic {
			summary = "已到确认期限，正在按原交易结算。"
		}
		if d.Channel == "OFFICIAL_STORE" {
			summary = "游戏邮箱已确认领取，正在结算至 DIMA 官方收入专户。"
		}
		if e = commerceEventV2(ctx, tx, d, actor, "settlement.started", summary, map[string]string{"operationId": op.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: op.ID}, nil
	})
}
func (s *Store) PrepareCommissionCancelV2(ctx context.Context, actor, id, key string, expected int64, available bool) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, "commission.cancel:"+id, map[string]any{"expectedVersion": expected}, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		if e := commerceAvailableV2(available); e != nil {
			return result, e
		}
		d, e := commerceRecordTxV2(ctx, tx, id, "COMMISSION", true)
		if e != nil {
			return result, e
		}
		if d.OwnerID != actor {
			return result, catalogDenied()
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if e = commerceRequireFulfillmentV2(ctx, tx, d); e != nil {
			return result, e
		}
		if d.State != "OPEN" || d.PayeeID != "" {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "已接取委托需通过双方退款协商处理。")
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "REFUND", "cancel", d.Amount, []CommerceStepV2{commerceMoneyStepV2(d, "wallet.escrow.refund", d.Amount)}, false, now)
		if e != nil {
			return result, e
		}
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "commission.cancel.started", "已申请取消未接取委托，正在原路退回预付。", map[string]string{"operationId": op.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: op.ID}, nil
	})
}
func (s *Store) PrepareDeliveryRetryV2(ctx context.Context, actor, id, key string, expected int64, reason string, available bool) (CommerceMutationV2, error) {
	if !catalogText(reason, 2, 500) {
		return CommerceMutationV2{}, catalogInvalid()
	}
	return s.commerceMutateV2(ctx, actor, key, "order.delivery-retry:"+id, map[string]any{"expectedVersion": expected, "reason": reason}, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		if e := commerceAvailableV2(available); e != nil {
			return result, e
		}
		d, e := commerceRecordTxV2(ctx, tx, id, "ORDER", true)
		if e != nil {
			return result, e
		}
		if d.Channel != "OFFICIAL_STORE" {
			return result, catalogNotFound()
		}
		if e = catalogPermissionTx(ctx, tx, actor, d.StoreID, "ORDER_MANAGE"); e != nil {
			return result, e
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if e = commerceRequireFulfillmentV2(ctx, tx, d); e != nil {
			return result, e
		}
		if d.State != "PAYMENT_PROCESSING" {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "此订单不需要重试创建交付。")
		}
		plan, ok := catalogObject(d.Body["mailboxPlan"])
		if !ok {
			return result, catalogError(503, "DELIVERY_SNAPSHOT_MISSING", "原交付快照需要核对。")
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "STORE_PURCHASE", "deliver", d.Amount, []CommerceStepV2{{Command: "mailbox.create", Payload: map[string]any(plan)}}, false, now)
		if e != nil {
			return result, e
		}
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "delivery.retry.started", "正在重试原邮箱交付，未重新扣款。", map[string]string{"reason": reason, "operationId": op.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: d.Kind, OperationID: op.ID}, nil
	})
}

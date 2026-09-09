package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func IsAIOrderV206(d CommerceRecordV2) bool {
	return d.Kind == "ORDER" && catalogString(d.Body, "orderType") == "AI_SUBSCRIPTION"
}

func aiEffectivePlanV206(ctx context.Context, tx *sql.Tx, user string, policy AIPolicyV2, now time.Time) (AIPlanV2, *time.Time, error) {
	var p AIPlanV2
	var raw string
	var expiry time.Time
	err := tx.QueryRowContext(ctx, "SELECT plan_snapshot,expires_at FROM ai_entitlements_v206 WHERE user_id=? AND expires_at>?", user, now).Scan(&raw, &expiry)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &p)
		return p, &expiry, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return p, nil, err
	}
	err = tx.QueryRowContext(ctx, "SELECT plan_id,code,name,description,CAST(price AS CHAR),quota_limit,window_hours,duration_days,model_tier,active,version FROM ai_plans_v2 WHERE code='free'").Scan(&p.PlanID, &p.Code, &p.Name, &p.Description, &p.Price, &p.QuotaPerWindow, &p.WindowHours, &p.DurationDays, &p.ModelTier, &p.Active, &p.Version)
	p.Currency = "CREDIT"
	if !policy.Configured {
		p.QuotaPerWindow = policy.FreeQuota
		p.WindowHours = policy.WindowHours
	}
	return p, nil, err
}

type AIPurchaseInputV206 struct {
	ClientRequestID     string `json:"clientRequestId"`
	PlanID              string `json:"planId"`
	ExpectedPlanVersion int64  `json:"expectedPlanVersion"`
}

func (s *Store) PrepareAIPurchaseV206(ctx context.Context, actor string, input AIPurchaseInputV206, enabled, available bool) (CommerceMutationV2, error) {
	if !CatalogReferenceV2(input.PlanID) || input.ExpectedPlanVersion < 1 {
		return CommerceMutationV2{}, catalogInvalid()
	}
	return s.commerceMutateV2(ctx, actor, input.ClientRequestID, "ai.purchase", input, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		// Lock configuration before the payer; disabling sales and checkout are atomic.
		var settingsRaw string
		err := tx.QueryRowContext(ctx, "SELECT settings_json FROM ai_settings_v206 WHERE id=1 LOCK IN SHARE MODE").Scan(&settingsRaw)
		if errors.Is(err, sql.ErrNoRows) || !enabled {
			return result, catalogError(503, "AI_PURCHASE_UNAVAILABLE", "套餐购买暂未开放。")
		}
		if err != nil {
			return result, err
		}
		var settings AISettingsV206
		if err = json.Unmarshal([]byte(settingsRaw), &settings); err != nil {
			return result, err
		}
		if !settings.Enabled || !settings.PaidEnabled {
			return result, catalogError(503, "AI_PURCHASE_UNAVAILABLE", "套餐购买暂未开放。")
		}
		if err = commerceAvailableV2(available); err != nil {
			return result, err
		}
		var identityStatus string
		if err = tx.QueryRowContext(ctx, "SELECT status FROM identities WHERE id=? FOR UPDATE", actor).Scan(&identityStatus); err != nil {
			return result, err
		}
		if identityStatus != "active" {
			return result, ErrUnauthorized
		}
		owner, err := commerceUserTxV2(ctx, tx, actor, true)
		if err != nil {
			return result, err
		}
		var p AIPlanV2
		err = tx.QueryRowContext(ctx, "SELECT plan_id,code,name,description,CAST(price AS CHAR),quota_limit,window_hours,duration_days,model_tier,active,version FROM ai_plans_v2 WHERE plan_id=? LOCK IN SHARE MODE", input.PlanID).Scan(&p.PlanID, &p.Code, &p.Name, &p.Description, &p.Price, &p.QuotaPerWindow, &p.WindowHours, &p.DurationDays, &p.ModelTier, &p.Active, &p.Version)
		if err != nil {
			return result, socialMissing(err)
		}
		if !p.Active || p.Code == "free" {
			return result, catalogError(409, "AI_PLAN_UNAVAILABLE", "这个套餐暂时无法购买，请选择其他套餐。")
		}
		if p.Version != input.ExpectedPlanVersion {
			return result, catalogError(409, "AI_PLAN_CHANGED", "套餐信息已更新，请查看最新价格和权益后重新确认。")
		}
		if err = commerceWithinExecutionLimitV2(p.Price); err != nil {
			return result, err
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		var activePlan string
		err = tx.QueryRowContext(ctx, "SELECT plan_id FROM ai_entitlements_v206 WHERE user_id=? AND expires_at>? LOCK IN SHARE MODE", actor, now).Scan(&activePlan)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
		if activePlan != "" && activePlan != p.PlanID {
			return result, catalogError(409, "AI_PLAN_ACTIVE", "当前套餐仍在有效期内，到期后可更换套餐。")
		}
		var pending string
		if err = tx.QueryRowContext(ctx, `SELECT resource_id FROM commerce_resources_v2 WHERE owner_id=? AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.orderType'))='AI_SUBSCRIPTION' AND funds_state NOT IN ('SETTLED','UNPAID','REFUNDED') LIMIT 1 FOR UPDATE`, actor).Scan(&pending); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
		if pending != "" {
			return result, catalogError(409, "AI_PURCHASE_PENDING", "上一笔套餐购买仍在处理中，请稍后查看。")
		}
		p.Currency = "CREDIT"
		p.Purchasable = true
		d := commerceNewResourceV2("ORDER", "OFFICIAL_STORE", owner, p.Price, now)
		seller := CatalogObjectV2{"kind": "OFFICIAL_STORE", "playerRef": nil, "storeId": "saki-ai", "displayName": "Saki AI", "contactQq": nil}
		items := []any{CatalogObjectV2{"productId": p.PlanID, "productVersion": p.Version, "title": p.Name, "subtitle": p.Description, "description": p.Description, "unitPrice": p.Price, "quantity": 1, "photoAssetIds": []string{}, "categoryName": "AI 套餐", "includedItems": []any{}, "contentBlocks": []any{}}}
		delivery := CatalogObjectV2{"method": "DIGITAL", "note": "付款成功后自动开通，无需领取"}
		d.Body = CatalogObjectV2{"orderType": "AI_SUBSCRIPTION", "orderNo": "S" + now.Format("20060102") + d.ID[6:], "construction": false, "buyer": commercePartyV2(owner), "seller": seller, "items": items, "delivery": delivery, "confirmationHours": 0, "aiPlan": p}
		snapshot := CatalogObjectV2{"snapshotId": d.SnapshotID, "orderId": d.ID, "buyer": d.Body["buyer"], "seller": seller, "items": items, "totalAmount": p.Price, "delivery": delivery, "confirmationHours": 0, "orderType": "AI_SUBSCRIPTION", "aiPlan": p, "capturedAt": now}
		d.SnapshotSHA256 = commerceSnapshotHashV2(snapshot)
		snapshot["sha256"] = d.SnapshotSHA256
		if err = commerceSaveV2(ctx, tx, &d, true); err != nil {
			return result, err
		}
		if err = commerceInsertSnapshotV2(ctx, tx, d, snapshot, now); err != nil {
			return result, err
		}
		steps := []CommerceStepV2{commerceReserveStepV2(d), commerceMoneyStepV2(d, "wallet.escrow.settle", d.Amount)}
		op, err := commerceNewOperationV2(ctx, tx, &d, actor, input.ClientRequestID, "AI_PURCHASE", "ai-purchase", d.Amount, steps, false, now)
		if err != nil {
			return result, err
		}
		if err = commerceSaveV2(ctx, tx, &d, false); err != nil {
			return result, err
		}
		if err = commerceEventV2(ctx, tx, d, actor, "ai.purchase", "正在开通 Saki AI 套餐。", nil); err != nil {
			return result, err
		}
		return CommerceMutationV2{ResourceID: d.ID, Kind: "ORDER", OperationID: op.ID}, nil
	})
}

func activateAIOrderV206(ctx context.Context, tx *sql.Tx, d *CommerceRecordV2, now time.Time) error {
	var p AIPlanV2
	if err := json.Unmarshal([]byte(catalogJSON(d.Body["aiPlan"])), &p); err != nil {
		return err
	}
	if p.DurationDays < 1 {
		return catalogInvalid()
	}
	var oldExpiry time.Time
	err := tx.QueryRowContext(ctx, "SELECT expires_at FROM ai_entitlements_v206 WHERE user_id=? FOR UPDATE", d.OwnerID).Scan(&oldExpiry)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	start := now
	if oldExpiry.After(start) {
		start = oldExpiry
	}
	expires := start.AddDate(0, 0, p.DurationDays)
	_, err = tx.ExecContext(ctx, `INSERT INTO ai_entitlements_v206 VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE plan_id=VALUES(plan_id),plan_snapshot=VALUES(plan_snapshot),expires_at=VALUES(expires_at),updated_at=VALUES(updated_at)`, d.OwnerID, p.PlanID, catalogJSON(p), expires, now)
	if err != nil {
		return err
	}
	d.State = "CONFIRMED"
	d.FundsState = "SETTLED"
	d.Body["confirmedAt"] = now
	d.Body["aiExpiresAt"] = expires
	return nil
}

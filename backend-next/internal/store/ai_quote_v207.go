package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"time"
)

// A short-lived, immutable price decision. The entitlement fingerprint stays
// inside the stored snapshot and is never accepted from a client.
type AIQuoteV207 struct {
	QuoteID                string     `json:"quoteId"`
	Plan                   AIPlanV2   `json:"plan"`
	Kind                   string     `json:"kind"`
	TotalAmount            string     `json:"totalAmount"`
	Currency               string     `json:"currency"`
	PreviousPrice          string     `json:"previousPrice,omitempty"`
	PriceDifference        string     `json:"priceDifference,omitempty"`
	RemainingDays          int64      `json:"remainingDays"`
	ChargedDays            int64      `json:"chargedDays,omitempty"`
	BillingCycleDays       int        `json:"billingCycleDays,omitempty"`
	EntitlementExpiresAt   *time.Time `json:"entitlementExpiresAt"`
	ExpiresAt              time.Time  `json:"expiresAt"`
	EntitlementFingerprint string     `json:"entitlementFingerprint"`
}

func (q AIQuoteV207) PublicV207() CatalogObjectV2 {
	var result CatalogObjectV2
	_ = json.Unmarshal([]byte(catalogJSON(q)), &result)
	delete(result, "entitlementFingerprint")
	return result
}

func aiUpgradeAmountV207(oldPrice, newPrice string, remaining time.Duration, cycleDays int) (string, int64, error) {
	if cycleDays < 1 {
		return "", 0, catalogInvalid()
	}
	old, err := commerceAmountV2(oldPrice)
	if err != nil {
		return "", 0, err
	}
	next, err := commerceAmountV2(newPrice)
	if err != nil {
		return "", 0, err
	}
	delta := new(big.Int).Sub(next, old)
	if remaining <= 0 || delta.Sign() <= 0 {
		return "", 0, catalogError(409, "AI_UPGRADE_UNAVAILABLE", "当前套餐无法补差价升级，请查看最新套餐信息。")
	}
	days := int64(remaining / (24 * time.Hour))
	if remaining%(24*time.Hour) != 0 {
		days++
	}
	chargedDays := max(days, int64(min(5, cycleDays)))
	cents := new(big.Int).Mul(delta, big.NewInt(chargedDays))
	// The purchased cycle is immutable. Keep five days' difference as the floor
	// (the whole cycle for cycles shorter than five days), rounding cents half up.
	cents.Add(cents, big.NewInt(int64(cycleDays/2))).Quo(cents, big.NewInt(int64(cycleDays)))
	amount := catalogMoneyString(cents)
	return amount, days, commerceWithinExecutionLimitV2(amount)
}

// Call after locking settings and the payer, matching checkout's lock order.
func aiPurchaseTermsV207(ctx context.Context, tx *sql.Tx, actor string, input AIPurchaseInputV206, now time.Time) (AIQuoteV207, error) {
	q := AIQuoteV207{Kind: "PURCHASE", Currency: "CREDIT", ExpiresAt: now.Add(90 * time.Second)}
	p := &q.Plan
	var rank int
	err := tx.QueryRowContext(ctx, "SELECT plan_id,code,name,description,CAST(price AS CHAR),quota_limit,window_hours,duration_days,model_tier,active,version,sort_order FROM ai_plans_v2 WHERE plan_id=? LOCK IN SHARE MODE", input.PlanID).Scan(&p.PlanID, &p.Code, &p.Name, &p.Description, &p.Price, &p.QuotaPerWindow, &p.WindowHours, &p.DurationDays, &p.ModelTier, &p.Active, &p.Version, &rank)
	if err != nil {
		return q, socialMissing(err)
	}
	if !p.Active || p.Code == "free" {
		return q, catalogError(409, "AI_PLAN_UNAVAILABLE", "这个套餐暂时无法购买，请选择其他套餐。")
	}
	if p.Version != input.ExpectedPlanVersion {
		return q, catalogError(409, "AI_PLAN_CHANGED", "套餐信息已更新，请查看最新价格和权益后重新确认。")
	}
	p.Currency = "CREDIT"
	p.Purchasable = true
	q.TotalAmount = p.Price
	var raw string
	var expiry time.Time
	err = tx.QueryRowContext(ctx, "SELECT plan_snapshot,expires_at FROM ai_entitlements_v206 WHERE user_id=? AND expires_at>? LOCK IN SHARE MODE", actor, now).Scan(&raw, &expiry)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return q, err
	}
	if err == nil {
		q.EntitlementFingerprint = Digest([]byte(raw + expiry.UTC().Format(time.RFC3339Nano)))
		q.EntitlementExpiresAt = &expiry
		if expiry.Before(q.ExpiresAt) {
			q.ExpiresAt = expiry
		}
		var old AIPlanV2
		if err = json.Unmarshal([]byte(raw), &old); err != nil {
			return q, err
		}
		if old.PlanID == p.PlanID {
			q.Kind = "RENEWAL"
		} else {
			var oldRank int
			if err = tx.QueryRowContext(ctx, "SELECT sort_order FROM ai_plans_v2 WHERE plan_id=? LOCK IN SHARE MODE", old.PlanID).Scan(&oldRank); err != nil {
				return q, err
			}
			if rank <= oldRank {
				return q, catalogError(409, "AI_DOWNGRADE_NOT_ALLOWED", "有效期内不支持降级，可继续使用当前套餐。")
			}
			q.Kind = "UPGRADE"
			q.PreviousPrice = old.Price
			q.BillingCycleDays = old.DurationDays
			q.TotalAmount, q.RemainingDays, err = aiUpgradeAmountV207(old.Price, p.Price, expiry.Sub(now), q.BillingCycleDays)
			if err != nil {
				return q, err
			}
			q.ChargedDays = max(q.RemainingDays, int64(min(5, q.BillingCycleDays)))
			// A quoted price must not survive the instant its billable day drops.
			if q.RemainingDays > int64(min(5, q.BillingCycleDays)) {
				nextDay := expiry.Add(-time.Duration(q.RemainingDays-1) * 24 * time.Hour)
				if nextDay.Before(q.ExpiresAt) {
					q.ExpiresAt = nextDay
				}
			}
			oldCents, _ := commerceAmountV2(old.Price)
			newCents, _ := commerceAmountV2(p.Price)
			q.PriceDifference = catalogMoneyString(new(big.Int).Sub(newCents, oldCents))
		}
	}
	return q, commerceWithinExecutionLimitV2(q.TotalAmount)
}

func aiSalesLockV207(ctx context.Context, tx *sql.Tx, actor string, enabled, available bool) error {
	var raw string
	err := tx.QueryRowContext(ctx, "SELECT settings_json FROM ai_settings_v206 WHERE id=1 LOCK IN SHARE MODE").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) || !enabled {
		return catalogError(503, "AI_PURCHASE_UNAVAILABLE", "套餐购买暂未开放。")
	}
	if err != nil {
		return err
	}
	var settings AISettingsV206
	if err = json.Unmarshal([]byte(raw), &settings); err != nil {
		return err
	}
	if !settings.Enabled || !settings.PaidEnabled {
		return catalogError(503, "AI_PURCHASE_UNAVAILABLE", "套餐购买暂未开放。")
	}
	if err = commerceAvailableV2(available); err != nil {
		return err
	}
	var status string
	if err = tx.QueryRowContext(ctx, "SELECT status FROM identities WHERE id=? FOR UPDATE", actor).Scan(&status); err != nil {
		return err
	}
	if status != "active" {
		return ErrUnauthorized
	}
	return nil
}

func (s *Store) AIQuoteV207(ctx context.Context, actor string, input AIPurchaseInputV206, enabled, available bool) (AIQuoteV207, error) {
	var q AIQuoteV207
	if !CatalogReferenceV2(input.PlanID) || input.ExpectedPlanVersion < 1 || input.QuoteID != "" {
		return q, catalogInvalid()
	}
	raw, err := s.catalogMutation(ctx, actor, input.ClientRequestID, "ai.quote", input, func(tx *sql.Tx) (any, error) {
		if err := aiSalesLockV207(ctx, tx, actor, enabled, available); err != nil {
			return nil, err
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		value, err := aiPurchaseTermsV207(ctx, tx, actor, input, now)
		if err != nil {
			return nil, err
		}
		value.QuoteID = ID("quote_")
		_, err = tx.ExecContext(ctx, "INSERT INTO catalog_quotes_v2 VALUES(?,?,?,?,?,?)", value.QuoteID, actor, "AI_SUBSCRIPTION", catalogJSON(value), now, value.ExpiresAt)
		return value, err
	})
	if err == nil {
		err = json.Unmarshal(raw, &q)
	}
	return q, err
}

func aiConfirmQuoteV207(ctx context.Context, tx *sql.Tx, actor, quoteID string, current AIQuoteV207, now time.Time) (AIQuoteV207, error) {
	var q AIQuoteV207
	var raw string
	var expires time.Time
	err := tx.QueryRowContext(ctx, "SELECT snapshot,expires_at FROM catalog_quotes_v2 WHERE quote_id=? AND user_id=? AND channel='AI_SUBSCRIPTION' FOR UPDATE", quoteID, actor).Scan(&raw, &expires)
	if err != nil {
		return q, socialMissing(err)
	}
	if !expires.After(now) {
		return q, catalogError(409, "QUOTE_EXPIRED", "价格确认已过期，请重新查看后确认购买。")
	}
	if err = json.Unmarshal([]byte(raw), &q); err != nil {
		return q, err
	}
	if q.Plan.PlanID != current.Plan.PlanID || q.Plan.Version != current.Plan.Version || q.EntitlementFingerprint != current.EntitlementFingerprint || q.Kind != current.Kind {
		return q, catalogError(409, "AI_QUOTE_CHANGED", "套餐或有效期已变化，请重新查看后确认购买。")
	}
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT resource_id FROM commerce_resources_v2 WHERE quote_id=? FOR UPDATE", quoteID).Scan(&existing)
	if err == nil {
		return q, catalogError(409, "QUOTE_ALREADY_USED", "此次购买已创建订单，请查看原订单。")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return q, err
	}
	return q, nil
}

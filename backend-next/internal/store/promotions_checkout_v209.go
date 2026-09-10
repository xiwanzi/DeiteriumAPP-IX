package store

import (
	"context"
	"database/sql"
	"time"
)

var promotionZoneV209 = time.FixedZone("Asia/Shanghai", 8*60*60)

func promotionPeriodStartV209(period string, limits CatalogObjectV2, now time.Time) time.Time {
	local := now.In(promotionZoneV209)
	clock, _ := time.Parse("15:04", catalogString(limits, period+"Time"))
	anchor := time.Date(local.Year(), local.Month(), local.Day(), clock.Hour(), clock.Minute(), 0, 0, promotionZoneV209)
	switch period {
	case "daily":
		if local.Before(anchor) {
			anchor = anchor.AddDate(0, 0, -1)
		}
	case "weekly":
		day := int(catalogNumber(limits, "weeklyDay"))
		if day == 0 {
			day = 1
		}
		delta := (int(local.Weekday()) - day%7 + 7) % 7
		anchor = anchor.AddDate(0, 0, -delta)
		if local.Before(anchor) {
			anchor = anchor.AddDate(0, 0, -7)
		}
	case "monthly":
		day := int(catalogNumber(limits, "monthlyDay"))
		if day == 0 {
			day = 1
		}
		inMonth := func(year int, month time.Month) time.Time {
			last := time.Date(year, month+1, 0, 0, 0, 0, 0, promotionZoneV209).Day()
			d := day
			if d > last {
				d = last
			}
			return time.Date(year, month, d, clock.Hour(), clock.Minute(), 0, 0, promotionZoneV209)
		}
		anchor = inMonth(local.Year(), local.Month())
		if local.Before(anchor) {
			anchor = inMonth(local.Year(), local.Month()-1)
		}
	default:
		return time.Time{}
	}
	return anchor.UTC()
}

// Checkout holds the identity and product rows before counting reservations.
// Thus simultaneous orders cannot both spend the same remaining allowance.
func promotionCheckLimitsV209(ctx context.Context, tx *sql.Tx, uuid string, product CatalogRecordV2, quantity int64, now time.Time) error {
	limits, ok := catalogObject(product.Published["purchaseLimits"])
	if !ok {
		return nil
	}
	for _, period := range []string{"lifetime", "daily", "weekly", "monthly"} {
		max := catalogNumber(limits, period)
		if max == 0 {
			continue
		}
		q := `SELECT COALESCE(SUM(h.quantity),0) FROM commerce_stock_holds_v2 h JOIN commerce_resources_v2 r ON r.resource_id=h.resource_id WHERE h.product_id=? AND r.owner_uuid=? AND r.channel='OFFICIAL_STORE' AND h.state<>'RELEASED' AND r.state NOT IN ('CANCELLED','REFUNDED')`
		args := []any{product.ID, uuid}
		if start := promotionPeriodStartV209(period, limits, now); !start.IsZero() {
			q += " AND r.created_at>=?"
			args = append(args, start)
		}
		var used int64
		if e := tx.QueryRowContext(ctx, q, args...).Scan(&used); e != nil {
			return e
		}
		if used+quantity > max {
			label := map[string]string{"lifetime": "累计", "daily": "每日", "weekly": "每周", "monthly": "每月"}[period]
			return catalogError(422, "PLAYER_PURCHASE_LIMIT", catalogString(product.Published, "title")+"已超出每位玩家的"+label+"限购，请调整购买数量。")
		}
	}
	return nil
}

func promotionCartSnapshotV209(ctx context.Context, tx *sql.Tx, user string, quantities map[string]int64) ([]any, error) {
	cart, e := catalogCartTx(ctx, tx, user, false)
	if e != nil {
		return nil, e
	}
	result := []any{}
	for _, item := range cart.Items {
		q := quantities[item.ProductID]
		if q > item.Quantity {
			q = item.Quantity
		}
		if q > 0 {
			result = append(result, CatalogObjectV2{"productId": item.ProductID, "quantity": q})
		}
	}
	return result, nil
}

func promotionCompletePurchaseV209(ctx context.Context, tx *sql.Tx, d *CommerceRecordV2, now time.Time) error {
	if d.Channel != "OFFICIAL_STORE" || IsAIOrderV206(*d) {
		return nil
	}
	if _, e := tx.ExecContext(ctx, "UPDATE promotion_redemptions_v209 SET redeemed_at=COALESCE(redeemed_at,?) WHERE resource_id=?", now, d.ID); e != nil {
		return e
	}
	if consumed, _ := d.Body["cartConsumed"].(bool); consumed {
		return nil
	}
	items, ok := d.Body["cartItems"].([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	if _, e := catalogCartTx(ctx, tx, d.OwnerID, true); e != nil {
		return e
	}
	for _, v := range items {
		item, ok := catalogObject(v)
		if !ok {
			return catalogInvalid()
		}
		id, q := catalogString(item, "productId"), catalogNumber(item, "quantity")
		// Subtract only purchased units: a later increase leaves its extra units.
		if _, e := tx.ExecContext(ctx, "UPDATE catalog_cart_items_v2 SET quantity=GREATEST(0,quantity-?) WHERE user_id=? AND product_id=?", q, d.OwnerID, id); e != nil {
			return e
		}
		if _, e := tx.ExecContext(ctx, "DELETE FROM catalog_cart_items_v2 WHERE user_id=? AND product_id=? AND quantity=0", d.OwnerID, id); e != nil {
			return e
		}
	}
	if _, e := tx.ExecContext(ctx, "UPDATE catalog_carts_v2 SET version=version+1,updated_at=? WHERE user_id=?", now, d.OwnerID); e != nil {
		return e
	}
	d.Body["cartConsumed"] = true
	return nil
}

func promotionReleaseV209(ctx context.Context, tx *sql.Tx, order CommerceRecordV2) error {
	// This runs inside the order-completion transaction, after proof validation.
	// Match the original order so later reuse survives a replay of this refund.
	fullRefund := order.Channel == "OFFICIAL_STORE" && order.State == "REFUNDED" && order.FundsState == "REFUNDED" && order.PendingOperationID == ""
	if fullRefund {
		paid, e := commerceAmountV2(order.Amount)
		if e != nil {
			return e
		}
		refunded, e := commerceAmountV2(order.RefundedAmount)
		if e != nil {
			return e
		}
		settled, e := commerceAmountV2(order.SettledAmount)
		if e != nil {
			return e
		}
		fullRefund = paid.Cmp(refunded) == 0 && settled.Sign() == 0
	}
	query := "DELETE FROM promotion_redemptions_v209 WHERE resource_id=?"
	if !fullRefund {
		query += " AND redeemed_at IS NULL"
	}
	_, e := tx.ExecContext(ctx, query, order.ID)
	return e
}

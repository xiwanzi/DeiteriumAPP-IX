package store

import (
	"context"
	"database/sql"
)

// Display preferences never change the business record or another participant's view.
func commerceCanHideV203(d CommerceRecordV2, viewer string, refund *CommerceRefundV2, activeCase bool) bool {
	participant := viewer == d.OwnerID || (d.PayeeID != "" && viewer == d.PayeeID)
	terminal := d.State == "CONFIRMED" || d.State == "CLAIMED" || d.State == "REFUNDED" || d.State == "CANCELLED"
	finalFunds := d.FundsState == "SETTLED" || d.FundsState == "REFUNDED" || d.FundsState == "UNPAID"
	refundPending := refund != nil && (refund.State == "REQUESTED" || refund.State == "PROCESSING")
	return participant && terminal && finalFunds && d.PendingOperationID == "" && !activeCase && !refundPending
}

// Reopened records become visible even if they were hidden when previously final.
const commerceFinalVisibilityV203 = `state IN ('CONFIRMED','CLAIMED','REFUNDED','CANCELLED')
 AND funds_state IN ('SETTLED','REFUNDED','UNPAID') AND pending_operation_id IS NULL
 AND NOT EXISTS (SELECT 1 FROM commerce_refunds_v2 vr WHERE vr.refund_id=commerce_resources_v2.refund_id AND vr.state IN ('REQUESTED','PROCESSING'))
 AND NOT EXISTS (SELECT 1 FROM commerce_interventions_v2 vi WHERE vi.case_id=commerce_resources_v2.intervention_case_id AND vi.state NOT IN ('RESOLVED','WITHDRAWN'))`

func listingCanHideV203(d CatalogRecordV2, viewer string) bool {
	return d.Kind == "listing" && d.OwnerID == viewer && (d.State != "ACTIVE" || (d.Stock != nil && *d.Stock == 0))
}

type visibilityQueryV203 interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func hiddenRecordV203(ctx context.Context, db visibilityQueryV203, user, kind, id string) (bool, error) {
	var hidden bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM personal_record_visibility_v203 WHERE user_id=? AND resource_kind=? AND resource_id=?)`, user, kind, id).Scan(&hidden)
	return hidden, err
}

func (s *Store) HideRecordV203(ctx context.Context, user, kind, id, key string, expected int64) error {
	if !CatalogReferenceV2(id) || !CatalogReferenceV2(key) || expected < 1 || (kind != "ORDER" && kind != "COMMISSION" && kind != "LISTING") {
		return catalogInvalid()
	}
	_, err := s.catalogMutation(ctx, user, key, "record-hide:"+kind+":"+id, map[string]any{"expectedVersion": expected}, func(tx *sql.Tx) (any, error) {
		if kind == "LISTING" {
			d, err := catalogRecordTx(ctx, tx, id, "listing", true)
			if err != nil {
				return nil, err
			}
			if d.OwnerID != user {
				return nil, catalogNotFound()
			}
			if d.Version != expected {
				return nil, catalogVersion()
			}
			if !listingCanHideV203(d, user) {
				return nil, catalogError(409, "RECORD_ACTIVE", "请先下架商品，再删除记录。")
			}
		} else {
			d, err := commerceRecordTxV2(ctx, tx, id, kind, true)
			if err != nil {
				return nil, err
			}
			if d.OwnerID != user && d.PayeeID != user {
				return nil, catalogNotFound()
			}
			if d.Version != expected {
				return nil, catalogVersion()
			}
			var refund *CommerceRefundV2
			if d.RefundID != "" {
				value, err := commerceRefundTxV2(ctx, tx, d.RefundID, false)
				if err != nil {
					return nil, err
				}
				refund = &value
			}
			active, err := commerceCaseActiveTxV2(ctx, tx, d)
			if err != nil {
				return nil, err
			}
			if !commerceCanHideV203(d, user, refund, active) {
				return nil, catalogError(409, "RECORD_ACTIVE", "交易仍需处理，结束后才能删除记录。")
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO personal_record_visibility_v203(user_id,resource_kind,resource_id,hidden_at) VALUES(?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE hidden_at=hidden_at`, user, kind, id)
		return map[string]bool{"hidden": true}, err
	})
	return err
}

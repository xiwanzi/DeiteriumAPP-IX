package store

import (
	"context"
	"strings"
	"time"
)

// Attention is separate from entitlement: reading a list never consumes a coupon
// or its arrival reminder. Existing coupons also apply to later registrations.
func (s *Store) CouponAttentionV209(ctx context.Context, actor, cursor string, limit int) ([]any, string, bool, error) {
	records, next, more, e := s.couponsFilteredV209(ctx, actor, false, "", cursor, limit, "", true)
	if e != nil {
		return nil, "", false, e
	}
	records, e = s.expandCouponBatchAttentionV209(ctx, actor, records)
	if e != nil {
		return nil, "", false, e
	}
	views, e := s.CouponViewsV209(ctx, records, false)
	if e != nil || len(records) == 0 {
		return views, next, more, e
	}
	args, marks := []any{actor}, []string{}
	for _, record := range records {
		args = append(args, record.ID)
		marks = append(marks, "?")
	}
	batchArgs := append([]any(nil), args...)
	args = append(args, batchArgs...)
	rows, e := s.DB.QueryContext(ctx, `SELECT a.coupon_id FROM promotion_attention_v209 a
 JOIN identities i ON i.server_uuid=a.owner_uuid WHERE i.id=? AND a.coupon_id IN (`+strings.Join(marks, ",")+`)
 UNION SELECT target.coupon_id FROM promotion_coupon_batches_v209 target
 JOIN promotion_coupon_batches_v209 sibling ON sibling.batch_id=target.batch_id
 JOIN promotion_attention_v209 a ON a.coupon_id=sibling.coupon_id
 JOIN identities i ON i.server_uuid=a.owner_uuid WHERE i.id=? AND target.coupon_id IN (`+strings.Join(marks, ",")+`)`, args...)
	if e != nil {
		return nil, "", false, e
	}
	defer rows.Close()
	announced := map[string]bool{}
	for rows.Next() {
		var id string
		if e := rows.Scan(&id); e != nil {
			return nil, "", false, e
		}
		announced[id] = true
	}
	if e := rows.Err(); e != nil {
		return nil, "", false, e
	}
	for _, raw := range views {
		view := raw.(CatalogObjectV2)
		view["announced"] = announced[catalogString(view, "couponId")]
	}
	return views, next, more, nil
}

func (s *Store) AcknowledgeCouponAttentionV209(ctx context.Context, actor string, ids []string, viewed bool) error {
	if _, ok := catalogRefs(ids, 1, 100); !ok {
		return catalogInvalid()
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Serialize acknowledgements for one identity, including two devices marking
	// the same batch at once. A viewed receipt must never regress to announced.
	var lockedID string
	if e := tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? FOR UPDATE", actor).Scan(&lockedID); e != nil {
		return e
	}
	user, e := commerceUserTxV2(ctx, tx, actor, true)
	if e != nil {
		return e
	}
	now := time.Now().UTC()
	var readAt any
	if viewed {
		readAt = now
	}
	args := []any{user.ServerUUID, now, readAt, now.Format(time.RFC3339), catalogJSON(user.PlayerRef)}
	marks := []string{}
	for _, id := range ids {
		args = append(args, id)
		marks = append(marks, "?")
	}
	// A coupon can expire, be revoked or be spent between display and ack. Preserve
	// the receipt in all three cases; inaccessible and future coupons are ignored.
	_, e = tx.ExecContext(ctx, `INSERT INTO promotion_attention_v209(coupon_id,owner_uuid,announced_at,viewed_at)
 SELECT resource_id,?,?,? FROM catalog_records_v2 WHERE kind='coupon'
 AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.startsAt'))<=?
 AND (JSON_UNQUOTE(JSON_EXTRACT(body,'$.audience'))='ALL' OR JSON_CONTAINS(JSON_EXTRACT(body,'$.playerRefs'),?))
 AND resource_id IN (`+strings.Join(marks, ",")+`) ORDER BY resource_id
 ON DUPLICATE KEY UPDATE viewed_at=COALESCE(viewed_at,VALUES(viewed_at))`, args...)
	if e != nil {
		return e
	}
	return tx.Commit()
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type CouponDraftSelectionV209 struct {
	CouponID        string `json:"couponId"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

type CouponPublicationV209 struct {
	BatchID    string            `json:"releaseBatchId"`
	ReleasedAt time.Time         `json:"releasedAt"`
	Coupons    []CatalogRecordV2 `json:"coupons"`
}

func validateCouponReferencesV209(ctx context.Context, tx *sql.Tx, contents []CatalogObjectV2) error {
	for _, field := range []string{"storeIds", "productIds", "playerRefs"} {
		refs := map[string]bool{}
		for _, content := range contents {
			ids, _ := catalogRefs(content[field], 0, 1000)
			for _, id := range ids {
				refs[id] = true
			}
		}
		if len(refs) == 0 {
			continue
		}
		// A batch can target many players. Keep each query within a small parameter budget.
		ids := make([]string, 0, len(refs))
		for id := range refs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for offset := 0; offset < len(ids); offset += 1000 {
			end := offset + 1000
			if end > len(ids) {
				end = len(ids)
			}
			args, marks := []any{}, []string{}
			q := "SELECT COUNT(*) FROM catalog_records_v2 WHERE kind=? AND resource_id IN ("
			if field == "playerRefs" {
				q = "SELECT COUNT(*) FROM identities WHERE status='active' AND player_ref IN ("
			} else {
				args = append(args, strings.TrimSuffix(field, "Ids"))
			}
			for _, id := range ids[offset:end] {
				args = append(args, id)
				marks = append(marks, "?")
			}
			var count int
			if e := tx.QueryRowContext(ctx, q+strings.Join(marks, ",")+")", args...).Scan(&count); e != nil {
				return e
			}
			if count != end-offset {
				if field == "playerRefs" {
					return catalogError(422, "COUPON_PLAYER_UNAVAILABLE", "指定玩家不存在或账号不可用。")
				}
				return catalogError(422, "COUPON_SCOPE_UNAVAILABLE", "指定店铺或商品已不可用，请修改草稿。")
			}
		}
	}
	return nil
}

// All selected drafts become available at one commit. Request replay returns the
// same batch; a stale/invalid draft rolls back every selected coupon.
func (s *Store) PublishCouponDraftsV209(ctx context.Context, actor, key string, selected []CouponDraftSelectionV209) (CouponPublicationV209, error) {
	var result CouponPublicationV209
	if len(selected) < 1 || len(selected) > 100 {
		return result, catalogInvalid()
	}
	selected = append([]CouponDraftSelectionV209(nil), selected...)
	sort.Slice(selected, func(i, j int) bool { return selected[i].CouponID < selected[j].CouponID })
	for i, c := range selected {
		if !CatalogReferenceV2(c.CouponID) || c.ExpectedVersion < 1 || (i > 0 && c.CouponID == selected[i-1].CouponID) {
			return result, catalogInvalid()
		}
	}
	for attempt := 0; ; attempt++ {
		result, e := s.publishCouponDraftsAttemptV209(ctx, actor, key, selected)
		var conflict *mysql.MySQLError
		if attempt >= 3 || !errors.As(e, &conflict) || (conflict.Number != 1020 && conflict.Number != 1213 && conflict.Number != 1205) {
			return result, e
		}
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
	}
}

func (s *Store) publishCouponDraftsAttemptV209(ctx context.Context, actor, key string, selected []CouponDraftSelectionV209) (CouponPublicationV209, error) {
	var result CouponPublicationV209
	raw, e := s.catalogMutation(ctx, actor, key, "coupon-publish", selected, func(tx *sql.Tx) (any, error) {
		if e := requirePlatformAdminTxV206(ctx, tx, actor); e != nil {
			return nil, e
		}
		now := time.Now().UTC()
		batch := CouponPublicationV209{BatchID: ID("batch_"), ReleasedAt: now, Coupons: []CatalogRecordV2{}}
		contents := []CatalogObjectV2{}
		for _, selection := range selected {
			coupon, e := catalogRecordTx(ctx, tx, selection.CouponID, "coupon", true)
			if e != nil {
				return nil, e
			}
			if coupon.State != "DRAFT" || coupon.Version != selection.ExpectedVersion {
				return nil, catalogError(409, "COUPON_DRAFT_CHANGED", "「"+catalogString(coupon.Body, "name")+"」已变更或已发放，请刷新草稿后重新选择。")
			}
			if e := validateCouponV209(coupon.Body); e != nil {
				return nil, e
			}
			ends, _ := time.Parse(time.RFC3339, catalogString(coupon.Body, "endsAt"))
			if !ends.After(now) {
				return nil, catalogError(422, "COUPON_DRAFT_EXPIRED", "「"+catalogString(coupon.Body, "name")+"」的有效期已结束，请先修改草稿。")
			}
			contents = append(contents, coupon.Body)
			batch.Coupons = append(batch.Coupons, coupon)
		}
		if e := validateCouponReferencesV209(ctx, tx, contents); e != nil {
			return nil, e
		}
		for i := range batch.Coupons {
			coupon := &batch.Coupons[i]
			coupon.State, coupon.Body["active"], coupon.UpdatedAt = "ACTIVE", true, now
			coupon.Version++
			if e := catalogSave(ctx, tx, coupon, false); e != nil {
				return nil, e
			}
			if _, e := tx.ExecContext(ctx, "INSERT INTO promotion_coupon_batches_v209(coupon_id,batch_id,released_at) VALUES(?,?,?)", coupon.ID, batch.BatchID, now); e != nil {
				return nil, e
			}
		}
		return batch, nil
	})
	if e != nil {
		return result, e
	}
	e = json.Unmarshal(raw, &result)
	return result, e
}

type couponReleaseV209 struct {
	BatchID    string
	ReleasedAt time.Time
}

func (s *Store) couponReleaseInfoV209(ctx context.Context, records []CatalogRecordV2) (map[string]couponReleaseV209, error) {
	result := map[string]couponReleaseV209{}
	if len(records) == 0 {
		return result, nil
	}
	args, marks := []any{}, []string{}
	for _, record := range records {
		args = append(args, record.ID)
		marks = append(marks, "?")
	}
	rows, e := s.DB.QueryContext(ctx, "SELECT coupon_id,batch_id,released_at FROM promotion_coupon_batches_v209 WHERE coupon_id IN ("+strings.Join(marks, ",")+")", args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var release couponReleaseV209
		if e := rows.Scan(&id, &release.BatchID, &release.ReleasedAt); e != nil {
			return nil, e
		}
		result[id] = release
	}
	return result, rows.Err()
}

// Keep an eligible publication together even when its draft IDs fall on opposite
// sides of a paging cursor or the batch was published during pagination.
func (s *Store) expandCouponBatchAttentionV209(ctx context.Context, actor string, records []CatalogRecordV2) ([]CatalogRecordV2, error) {
	if len(records) == 0 {
		return records, nil
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	user, e := commerceUserTxV2(ctx, tx, actor, false)
	if e != nil {
		return nil, e
	}
	filter, args := promotionEligibilitySQLV209(user, time.Now().UTC())
	filter += " AND NOT EXISTS(SELECT 1 FROM promotion_attention_v209 a WHERE a.coupon_id=catalog_records_v2.resource_id AND a.owner_uuid=? AND a.viewed_at IS NOT NULL)"
	args = append(args, user.ServerUUID)
	marks := []string{}
	for _, record := range records {
		args = append(args, record.ID)
		marks = append(marks, "?")
	}
	q := catalogSelect + " WHERE kind='coupon'" + filter + " AND resource_id IN (SELECT sibling.coupon_id FROM promotion_coupon_batches_v209 sibling JOIN promotion_coupon_batches_v209 selected ON selected.batch_id=sibling.batch_id WHERE selected.coupon_id IN (" + strings.Join(marks, ",") + "))"
	rows, e := tx.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	merged := map[string]CatalogRecordV2{}
	for _, record := range records {
		merged[record.ID] = record
	}
	for rows.Next() {
		record, e := catalogScan(rows)
		if e != nil {
			return nil, e
		}
		merged[record.ID] = record
	}
	if e := rows.Err(); e != nil {
		return nil, e
	}
	result := make([]CatalogRecordV2, 0, len(merged))
	for _, record := range merged {
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence > result[j].Sequence })
	return result, nil
}

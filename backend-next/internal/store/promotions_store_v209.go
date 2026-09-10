package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func validateCouponV209(c CatalogObjectV2) error {
	if !catalogFields(c, "name type benefit amountOff discountRate minimumSpend maxDiscount stackWithProductDiscount audience playerRefs storeIds productIds startsAt endsAt active", "") || !catalogText(c["name"], 1, 80) || !catalogEnum(c["type"], "ORDER", "ITEM") || !catalogEnum(c["benefit"], "FIXED", "PERCENT") || !catalogEnum(c["audience"], "ALL", "PLAYERS") {
		return catalogInvalid()
	}
	if _, ok := c["active"].(bool); !ok {
		return catalogInvalid()
	}
	if _, ok := c["stackWithProductDiscount"].(bool); !ok {
		return catalogInvalid()
	}
	for _, key := range []string{"amountOff", "minimumSpend", "maxDiscount"} {
		v, e := commerceAmountV2(catalogString(c, key))
		if e != nil || !v.IsInt64() || v.Int64() > 100000000000000 {
			return catalogInvalid()
		}
	}
	if !catalogRange(c["discountRate"], 1, 10000) {
		return catalogInvalid()
	}
	if c["type"] == "ITEM" && c["benefit"] != "PERCENT" {
		return catalogInvalid()
	}
	if c["benefit"] == "PERCENT" && catalogNumber(c, "discountRate") == 10000 {
		return catalogInvalid()
	}
	if c["benefit"] == "FIXED" {
		if n, _ := commerceAmountV2(catalogString(c, "amountOff")); n.Sign() == 0 {
			return catalogInvalid()
		}
	}
	refs, ok := catalogRefs(c["playerRefs"], 0, 1000)
	if !ok || (c["audience"] == "PLAYERS" && len(refs) == 0) || (c["audience"] == "ALL" && len(refs) != 0) {
		return catalogInvalid()
	}
	for _, key := range []string{"storeIds", "productIds"} {
		if _, ok := catalogRefs(c[key], 0, 100); !ok {
			return catalogInvalid()
		}
	}
	start, e := time.Parse(time.RFC3339, catalogString(c, "startsAt"))
	if e != nil {
		return catalogInvalid()
	}
	end, e := time.Parse(time.RFC3339, catalogString(c, "endsAt"))
	if e != nil || !end.Truncate(time.Second).After(start.Truncate(time.Second)) {
		return catalogInvalid()
	}
	c["startsAt"], c["endsAt"] = start.UTC().Truncate(time.Second).Format(time.RFC3339), end.UTC().Truncate(time.Second).Format(time.RFC3339)
	return nil
}

func promotionAdminV209(ctx context.Context, tx *sql.Tx, actor string) error {
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND permission='platform.admin'", actor).Scan(&n); e != nil {
		return e
	}
	if n == 0 {
		return catalogDenied()
	}
	return nil
}

func (s *Store) SaveCouponV209(ctx context.Context, actor, id, key string, expected int64, content CatalogObjectV2) (CatalogRecordV2, error) {
	if e := validateCouponV209(content); e != nil {
		return CatalogRecordV2{}, e
	}
	content = promotionCopyV209(content)
	raw, e := s.catalogMutation(ctx, actor, key, "coupon:"+id, CatalogObjectV2{"expectedVersion": expected, "content": content}, func(tx *sql.Tx) (any, error) {
		if e := requirePlatformAdminTxV206(ctx, tx, actor); e != nil {
			return nil, e
		}
		if e := validateCouponReferencesV209(ctx, tx, []CatalogObjectV2{content}); e != nil {
			return nil, e
		}
		now := time.Now().UTC()
		var d CatalogRecordV2
		if id == "" {
			d = CatalogRecordV2{ID: ID("coupon_"), Kind: "coupon", OwnerID: actor, Version: 1, CreatedAt: now}
		} else {
			var e error
			d, e = catalogRecordTx(ctx, tx, id, "coupon", true)
			if e != nil {
				return nil, e
			}
			if d.Version != expected {
				return nil, catalogVersion()
			}
			d.Version++
		}
		draft := id == "" || d.State == "DRAFT"
		d.Body, d.UpdatedAt, d.State = content, now, "INACTIVE"
		if draft {
			d.State, d.Body["active"] = "DRAFT", false
		} else if active, _ := content["active"].(bool); active {
			d.State = "ACTIVE"
		}
		if e := catalogSave(ctx, tx, &d, id == ""); e != nil {
			return nil, e
		}
		return d, nil
	})
	return catalogDecodeRecord(raw, e)
}

func CouponViewV209(d CatalogRecordV2, admin bool) CatalogObjectV2 {
	publication := "PUBLISHED"
	if d.State == "DRAFT" {
		publication = "DRAFT"
	}
	view := CatalogObjectV2{"couponId": d.ID, "version": d.Version, "createdAt": d.CreatedAt, "publicationState": publication, "releaseBatchId": nil, "releasedAt": nil}
	for k, v := range d.Body {
		if admin || (k != "playerRefs" && k != "audience") {
			view[k] = v
		}
	}
	return view
}

func (s *Store) CouponViewsV209(ctx context.Context, records []CatalogRecordV2, admin bool) ([]any, error) {
	releases, e := s.couponReleaseInfoV209(ctx, records)
	if e != nil {
		return nil, e
	}
	names := map[string]string{}
	for _, record := range records {
		for _, field := range []string{"storeIds", "productIds"} {
			for _, id := range catalogIDs(record.Body, field) {
				names[id] = ""
			}
		}
	}
	if len(names) > 0 {
		args, marks := []any{}, []string{}
		for id := range names {
			args = append(args, id)
			marks = append(marks, "?")
		}
		rows, e := s.DB.QueryContext(ctx, "SELECT resource_id,title FROM catalog_records_v2 WHERE resource_id IN ("+strings.Join(marks, ",")+")", args...)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var id, name string
			if e := rows.Scan(&id, &name); e != nil {
				rows.Close()
				return nil, e
			}
			names[id] = name
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
	}
	result := []any{}
	for _, record := range records {
		view := CouponViewV209(record, admin)
		if release, ok := releases[record.ID]; ok {
			view["releaseBatchId"], view["releasedAt"] = release.BatchID, release.ReleasedAt
		}
		labels := []string{}
		for _, field := range []string{"storeIds", "productIds"} {
			ids := catalogIDs(record.Body, field)
			if len(ids) == 0 {
				if field == "storeIds" {
					labels = append(labels, "全部店铺")
				} else {
					labels = append(labels, "全部商品")
				}
				continue
			}
			selected := []string{}
			for _, id := range ids {
				if name := names[id]; name != "" {
					selected = append(selected, name)
				}
			}
			label := strings.Join(selected, "、")
			if len(selected) > 2 {
				label = strings.Join(selected[:2], "、") + "等 " + strconv.Itoa(len(ids)) + " 项"
			}
			if label == "" {
				label = "指定范围内的商品"
			}
			labels = append(labels, label)
		}
		view["scopeDescription"] = strings.Join(labels, " · ")
		result = append(result, view)
	}
	return result, nil
}

func promotionEligibilitySQLV209(user User, now time.Time) (string, []any) {
	instant := now.UTC().Format(time.RFC3339)
	q := ` AND state='ACTIVE' AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.startsAt'))<=? AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.endsAt'))>?
 AND (JSON_UNQUOTE(JSON_EXTRACT(body,'$.audience'))='ALL' OR JSON_CONTAINS(JSON_EXTRACT(body,'$.playerRefs'),?))
 AND NOT EXISTS(SELECT 1 FROM promotion_redemptions_v209 p WHERE p.coupon_id=catalog_records_v2.resource_id AND p.owner_uuid=?)`
	return q, []any{instant, instant, catalogJSON(user.PlayerRef), user.ServerUUID}
}

func promotionCouponsTxV209(ctx context.Context, tx *sql.Tx, user User, now time.Time) ([]CatalogRecordV2, error) {
	where, args := promotionEligibilitySQLV209(user, now)
	rows, e := tx.QueryContext(ctx, catalogSelect+" WHERE kind='coupon'"+where+" ORDER BY resource_id", args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []CatalogRecordV2{}
	for rows.Next() {
		d, e := catalogScan(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (s *Store) CouponsV209(ctx context.Context, actor string, admin bool, query, cursor string, limit int) ([]CatalogRecordV2, string, bool, error) {
	return s.CouponsFilteredV209(ctx, actor, admin, query, cursor, limit, "")
}

func (s *Store) CouponsFilteredV209(ctx context.Context, actor string, admin bool, query, cursor string, limit int, status string) ([]CatalogRecordV2, string, bool, error) {
	return s.couponsFilteredV209(ctx, actor, admin, query, cursor, limit, status, false)
}

func (s *Store) couponsFilteredV209(ctx context.Context, actor string, admin bool, query, cursor string, limit int, status string, attention bool) ([]CatalogRecordV2, string, bool, error) {
	if limit < 1 || limit > 100 || !catalogText(query, 0, 100) {
		return nil, "", false, catalogInvalid()
	}
	if !catalogEnum(status, "", "DRAFT", "ACTIVE", "SCHEDULED", "EXPIRED", "INACTIVE") || (!admin && status != "") {
		return nil, "", false, catalogInvalid()
	}
	after := int64(0)
	if cursor != "" {
		var e error
		after, e = strconv.ParseInt(cursor, 10, 64)
		if e != nil || after < 1 {
			return nil, "", false, catalogInvalid()
		}
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, "", false, e
	}
	defer tx.Rollback()
	where, args := " WHERE kind='coupon'", []any{}
	if admin {
		if e := promotionAdminV209(ctx, tx, actor); e != nil {
			return nil, "", false, e
		}
	} else {
		u, e := commerceUserTxV2(ctx, tx, actor, false)
		if e != nil {
			return nil, "", false, e
		}
		filter, values := promotionEligibilitySQLV209(u, time.Now().UTC())
		where += filter
		args = append(args, values...)
		if attention {
			where += " AND NOT EXISTS(SELECT 1 FROM promotion_attention_v209 a WHERE a.coupon_id=catalog_records_v2.resource_id AND a.owner_uuid=? AND a.viewed_at IS NOT NULL)"
			args = append(args, u.ServerUUID)
		}
	}
	if after > 0 {
		where += " AND sequence_id<?"
		args = append(args, after)
	}
	if admin && status != "" {
		now := time.Now().UTC().Format(time.RFC3339)
		switch status {
		case "DRAFT":
			where += " AND state='DRAFT'"
		case "ACTIVE":
			where += " AND state='ACTIVE' AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.startsAt'))<=? AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.endsAt'))>?"
			args = append(args, now, now)
		case "SCHEDULED":
			where += " AND state='ACTIVE' AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.startsAt'))>?"
			args = append(args, now)
		case "EXPIRED":
			where += " AND state='ACTIVE' AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.endsAt'))<=?"
			args = append(args, now)
		case "INACTIVE":
			where += " AND state='INACTIVE'"
		}
	}
	if strings.TrimSpace(query) != "" {
		where += " AND INSTR(LOWER(title),LOWER(?))>0"
		args = append(args, strings.TrimSpace(query))
	}
	args = append(args, limit+1)
	rows, e := tx.QueryContext(ctx, catalogSelect+where+" ORDER BY sequence_id DESC LIMIT ?", args...)
	if e != nil {
		return nil, "", false, e
	}
	defer rows.Close()
	result := []CatalogRecordV2{}
	for rows.Next() {
		d, e := catalogScan(rows)
		if e != nil {
			return nil, "", false, e
		}
		result = append(result, d)
	}
	if e := rows.Err(); e != nil {
		return nil, "", false, e
	}
	more, next := len(result) > limit, ""
	if more {
		result = result[:limit]
		next = strconv.FormatInt(result[len(result)-1].Sequence, 10)
	}
	return result, next, more, nil
}

func promotionReserveV209(ctx context.Context, tx *sql.Tx, owner User, order CommerceRecordV2, quote CatalogObjectV2, now time.Time) error {
	applied, ok := catalogObject(quote["coupon"])
	if !ok {
		return nil
	}
	id := catalogString(applied, "couponId")
	coupon, e := catalogRecordTx(ctx, tx, id, "coupon", true)
	if e != nil {
		return e
	}
	if coupon.Version != catalogNumber(applied, "version") || coupon.State != "ACTIVE" {
		return catalogError(409, "COUPON_CHANGED", "优惠券已变化，请重新确认结算。")
	}
	start, _ := time.Parse(time.RFC3339, catalogString(coupon.Body, "startsAt"))
	end, _ := time.Parse(time.RFC3339, catalogString(coupon.Body, "endsAt"))
	if now.Before(start) || !now.Before(end) {
		return catalogError(409, "COUPON_EXPIRED", "优惠券已失效，请重新确认结算。")
	}
	if coupon.Body["audience"] == "PLAYERS" {
		refs, _ := catalogRefs(coupon.Body["playerRefs"], 0, 1000)
		found := false
		for _, ref := range refs {
			if ref == owner.PlayerRef {
				found = true
			}
		}
		if !found {
			return catalogDenied()
		}
	}
	var existing string
	e = tx.QueryRowContext(ctx, "SELECT resource_id FROM promotion_redemptions_v209 WHERE coupon_id=? AND owner_uuid=? FOR UPDATE", id, owner.ServerUUID).Scan(&existing)
	if e == nil {
		return catalogError(409, "COUPON_ALREADY_USED", "优惠券已在另一笔订单中使用，请重新确认结算。")
	}
	if e != sql.ErrNoRows {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO promotion_redemptions_v209(coupon_id,owner_uuid,resource_id) VALUES(?,?,?)", id, owner.ServerUUID, order.ID)
	return e
}

func promotionCopyV209(value CatalogObjectV2) CatalogObjectV2 {
	var result CatalogObjectV2
	_ = json.Unmarshal([]byte(catalogJSON(value)), &result)
	return result
}

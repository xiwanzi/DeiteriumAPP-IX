package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CatalogRecordV2 struct {
	Sequence                   int64
	ID, Kind, StoreID, OwnerID string
	Version                    int64
	Body                       CatalogObjectV2
	Published                  CatalogObjectV2
	PublishedVersion           int64
	PublishedAt                *time.Time
	State                      string
	Stock                      *int64
	CreatedAt, UpdatedAt       time.Time
}

const catalogSelect = `SELECT sequence_id,resource_id,kind,store_id,owner_id,version,body,published_body,published_version,published_at,state,available_stock,created_at,updated_at FROM catalog_records_v2`

type catalogScanner interface{ Scan(...any) error }

func catalogScan(row catalogScanner) (d CatalogRecordV2, err error) {
	var body string
	var pub sql.NullString
	var version, stock sql.NullInt64
	var published sql.NullTime
	err = row.Scan(&d.Sequence, &d.ID, &d.Kind, &d.StoreID, &d.OwnerID, &d.Version, &body, &pub, &version, &published, &d.State, &stock, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return d, catalogNotFound()
	}
	if err != nil {
		return
	}
	if err = json.Unmarshal([]byte(body), &d.Body); err != nil {
		return
	}
	if pub.Valid {
		err = json.Unmarshal([]byte(pub.String), &d.Published)
	}
	d.PublishedVersion = version.Int64
	if published.Valid {
		d.PublishedAt = &published.Time
	}
	if stock.Valid {
		d.Stock = &stock.Int64
	}
	return
}
func catalogRecordTx(ctx context.Context, tx *sql.Tx, id, kind string, lock bool) (CatalogRecordV2, error) {
	q := catalogSelect + " WHERE resource_id=? AND kind=?"
	if lock {
		q += " FOR UPDATE"
	}
	return catalogScan(tx.QueryRowContext(ctx, q, id, kind))
}
func (s *Store) catalogRecord(ctx context.Context, id, kind string) (CatalogRecordV2, error) {
	return catalogScan(s.DB.QueryRowContext(ctx, catalogSelect+" WHERE resource_id=? AND kind=?", id, kind))
}
func catalogSave(ctx context.Context, tx *sql.Tx, d *CatalogRecordV2, create bool) error {
	var published any
	if d.Published != nil {
		published = catalogJSON(d.Published)
	}
	var pv any
	if d.PublishedVersion > 0 {
		pv = d.PublishedVersion
	}
	title := catalogString(d.Body, "title")
	if title == "" {
		title = catalogString(d.Body, "name")
	}
	category := catalogString(d.Body, "categoryId")
	if category == "" {
		category = catalogString(d.Body, "categoryCode")
	}
	brand := catalogString(d.Body, "brandId")
	// Public search indexes always describe the published product, never a draft.
	if d.Kind == "product" && d.Published != nil {
		title = catalogString(d.Published, "title")
		category = catalogString(d.Published, "categoryId")
		brand = catalogString(d.Published, "brandId")
	}
	if create {
		r, e := tx.ExecContext(ctx, `INSERT INTO catalog_records_v2(resource_id,kind,store_id,owner_id,version,body,published_body,published_version,published_at,state,available_stock,title,category_ref,brand_ref,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, d.ID, d.Kind, d.StoreID, d.OwnerID, d.Version, catalogJSON(d.Body), published, pv, d.PublishedAt, d.State, d.Stock, title, category, brand, d.CreatedAt, d.UpdatedAt)
		if e != nil {
			return e
		}
		d.Sequence, e = r.LastInsertId()
		return e
	}
	_, e := tx.ExecContext(ctx, `UPDATE catalog_records_v2 SET owner_id=?,version=?,body=?,published_body=?,published_version=?,published_at=?,state=?,available_stock=?,title=?,category_ref=?,brand_ref=?,updated_at=? WHERE resource_id=?`, d.OwnerID, d.Version, catalogJSON(d.Body), published, pv, d.PublishedAt, d.State, d.Stock, title, category, brand, d.UpdatedAt, d.ID)
	return e
}
func catalogPermissionTx(ctx context.Context, tx *sql.Tx, user, storeID, permission string) error {
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND permission IN ('platform.admin','store.manage')", user).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	var permissions string
	err := tx.QueryRowContext(ctx, "SELECT permissions FROM catalog_members_v2 WHERE store_id=? AND user_id=? AND active=TRUE", storeID, user).Scan(&permissions)
	if errors.Is(err, sql.ErrNoRows) {
		return catalogDenied()
	}
	if err != nil {
		return err
	}
	var values []string
	if json.Unmarshal([]byte(permissions), &values) != nil {
		return catalogDenied()
	}
	for _, v := range values {
		if v == permission || permission == "READ" {
			return nil
		}
	}
	return catalogDenied()
}
func (s *Store) CatalogCanManageV2(ctx context.Context, user, storeID, permission string) error {
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	return catalogPermissionTx(ctx, tx, user, storeID, permission)
}
func (s *Store) catalogMutation(ctx context.Context, user, key, scope string, input any, fn func(*sql.Tx) (any, error)) (json.RawMessage, error) {
	if !CatalogReferenceV2(key) {
		return nil, catalogInvalid()
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	fingerprint := Digest([]byte(scope + ":" + catalogJSON(input)))
	_, err = tx.ExecContext(ctx, `INSERT INTO catalog_mutations_v2(user_id,request_id,scope,fingerprint,result,created_at) VALUES(?,?,?,?,'',UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE request_id=VALUES(request_id)`, user, key, scope, fingerprint)
	if err != nil {
		return nil, err
	}
	var existingScope, existingFingerprint, result string
	if err = tx.QueryRowContext(ctx, "SELECT scope,fingerprint,result FROM catalog_mutations_v2 WHERE user_id=? AND request_id=? FOR UPDATE", user, key).Scan(&existingScope, &existingFingerprint, &result); err != nil {
		return nil, err
	}
	if existingScope != scope || existingFingerprint != fingerprint {
		return nil, catalogError(409, "IDEMPOTENCY_CONFLICT", "此请求标识已用于不同内容。")
	}
	if result != "" {
		return json.RawMessage(result), nil
	}
	value, err := fn(tx)
	if err != nil {
		return nil, err
	}
	result = catalogJSON(value)
	if _, err = tx.ExecContext(ctx, "UPDATE catalog_mutations_v2 SET result=? WHERE user_id=? AND request_id=?", result, user, key); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", user, "catalog.mutate", scope); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return json.RawMessage(result), nil
}
func catalogDecodeRecord(raw json.RawMessage, err error) (CatalogRecordV2, error) {
	var d CatalogRecordV2
	if err == nil {
		err = json.Unmarshal(raw, &d)
	}
	return d, err
}
func catalogPurpose(kind string) string {
	if kind == "listing" {
		return "MARKET_PHOTO"
	}
	return "STORE_MEDIA"
}
func catalogBinding(kind string) string {
	switch kind {
	case "listing":
		return "MARKET_LISTING"
	case "product":
		return "PRODUCT_DRAFT"
	case "store":
		return "STORE_PROFILE"
	case "homepage":
		return "STORE_HOMEPAGE"
	case "brand":
		return "STORE_BRAND"
	}
	return "CATALOG_CATEGORY"
}
func (s *Store) catalogBind(ctx context.Context, tx *sql.Tx, d CatalogRecordV2, businessType string, content CatalogObjectV2) error {
	ids := CatalogAssetsV2(d.Kind, content)
	for _, id := range ids {
		var purpose string
		if e := tx.QueryRowContext(ctx, "SELECT purpose FROM asset_uploads_v2 WHERE asset_id=?", id).Scan(&purpose); e != nil {
			return e
		}
		if purpose != catalogPurpose(d.Kind) {
			return ErrAssetUnavailable
		}
	}
	return s.SetAssetBindingsV2(ctx, tx, d.OwnerID, businessType, d.ID, ids)
}
func catalogReferences(ctx context.Context, tx *sql.Tx, kind, storeID string, body CatalogObjectV2) error {
	refs := map[string][]string{}
	if kind == "product" {
		refs["brand"] = []string{catalogString(body, "brandId")}
		refs["category"] = []string{catalogString(body, "categoryId")}
	}
	if kind == "homepage" {
		refs["brand"] = catalogIDs(body, "brandIds")
		refs["category"] = catalogIDs(body, "categoryIds")
		for _, sec := range body["sections"].([]any) {
			o, _ := catalogObject(sec)
			refs["product"] = append(refs["product"], catalogIDs(o, "productIds")...)
		}
	}
	for k, ids := range refs {
		for _, id := range ids {
			d, e := catalogRecordTx(ctx, tx, id, k, false)
			if e != nil {
				return e
			}
			if d.StoreID != storeID || d.State != "ACTIVE" {
				return catalogError(422, "CATALOG_REFERENCE_UNAVAILABLE", "分类、品牌或引用商品不可用。")
			}
		}
	}
	return nil
}

func (s *Store) CatalogCreateV2(ctx context.Context, user, kind, storeID, key string, content CatalogObjectV2) (CatalogRecordV2, error) {
	if e := ValidateCatalogContentV2(kind, content); e != nil {
		return CatalogRecordV2{}, e
	}
	var replay int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_mutations_v2 WHERE user_id=? AND request_id=?", user, key).Scan(&replay); err != nil {
		return CatalogRecordV2{}, err
	}
	if replay == 0 {
		if err := s.EnsureAssetsActiveV2(ctx, user, "", "", CatalogAssetsV2(kind, content)); err != nil {
			return CatalogRecordV2{}, err
		}
	}
	if kind != "listing" {
		p := "STORE_EDIT"
		if kind == "product" {
			p = "PRODUCT_EDIT"
		}
		if e := s.CatalogCanManageV2(ctx, user, storeID, p); e != nil {
			return CatalogRecordV2{}, e
		}
	}
	return catalogDecodeRecord(s.catalogMutation(ctx, user, key, "create:"+kind+":"+storeID, content, func(tx *sql.Tx) (any, error) {
		if kind != "listing" {
			perm := "STORE_EDIT"
			if kind == "product" {
				perm = "PRODUCT_EDIT"
			}
			if e := catalogPermissionTx(ctx, tx, user, storeID, perm); e != nil {
				return nil, e
			}
			if kind != "store" {
				if _, e := catalogRecordTx(ctx, tx, storeID, "store", kind == "homepage"); e != nil {
					return nil, e
				}
			}
			if kind == "homepage" {
				var count int
				if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_records_v2 WHERE kind='homepage' AND store_id=?", storeID).Scan(&count); e != nil {
					return nil, e
				}
				if count != 0 {
					return nil, catalogVersion()
				}
			}
		}
		if e := catalogReferences(ctx, tx, kind, storeID, content); e != nil {
			return nil, e
		}
		now := time.Now().UTC()
		d := CatalogRecordV2{ID: ID(kind + "_"), Kind: kind, StoreID: storeID, OwnerID: user, Version: 1, Body: content, State: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		if kind == "product" {
			d.State = "DRAFT"
		}
		if kind == "listing" {
			v := catalogNumber(content, "stock")
			d.Stock = &v
		}
		if v, ok := content["active"].(bool); ok && !v {
			d.State = "INACTIVE"
		}
		if e := catalogSave(ctx, tx, &d, true); e != nil {
			return nil, e
		}
		if e := s.catalogBind(ctx, tx, d, catalogBinding(kind), content); e != nil {
			return nil, e
		}
		return d, nil
	}))
}
func (s *Store) CatalogEditV2(ctx context.Context, user, id, kind, key string, expected int64, content CatalogObjectV2, republish bool) (CatalogRecordV2, error) {
	if e := ValidateCatalogContentV2(kind, content); e != nil {
		return CatalogRecordV2{}, e
	}
	current, e := s.CatalogGetRecordV2(ctx, user, id, kind, true)
	if e != nil {
		return current, e
	}
	if kind != "listing" {
		p := "STORE_EDIT"
		if kind == "product" {
			p = "PRODUCT_EDIT"
		}
		storeID := current.StoreID
		if kind == "store" {
			storeID = current.ID
		}
		if e = s.CatalogCanManageV2(ctx, user, storeID, p); e != nil {
			return current, e
		}
	}
	if current.Version == expected {
		if err := s.EnsureAssetsActiveV2(ctx, user, catalogBinding(kind), id, CatalogAssetsV2(kind, content)); err != nil {
			return current, err
		}
	}
	return catalogDecodeRecord(s.catalogMutation(ctx, user, key, fmt.Sprintf("edit:%s:%s:%t", kind, id, republish), map[string]any{"content": content, "expectedVersion": expected}, func(tx *sql.Tx) (any, error) {
		d, e := catalogRecordTx(ctx, tx, id, kind, true)
		if e != nil {
			return nil, e
		}
		if kind == "listing" {
			if d.OwnerID != user {
				return nil, catalogDenied()
			}
		} else {
			p := "STORE_EDIT"
			if kind == "product" {
				p = "PRODUCT_EDIT"
			}
			storeID := d.StoreID
			if kind == "store" {
				storeID = d.ID
			}
			if e = catalogPermissionTx(ctx, tx, user, storeID, p); e != nil {
				return nil, e
			}
		}
		if d.Version != expected || d.Version >= 2147483647 {
			return nil, catalogVersion()
		}
		if d.State == "ARCHIVED" {
			return nil, catalogError(409, "INVALID_STATE_TRANSITION", "已归档商品不能编辑。")
		}
		if republish && d.State != "UNLISTED" {
			return nil, catalogError(409, "INVALID_STATE_TRANSITION", "仅下架商品可以重新发布。")
		}
		if e = catalogReferences(ctx, tx, kind, d.StoreID, content); e != nil {
			return nil, e
		}
		d.Body = content
		d.Version++
		d.UpdatedAt = time.Now().UTC()
		d.OwnerID = user
		if kind == "listing" {
			v := catalogNumber(content, "stock")
			held, holdError := catalogHeldStockV2(ctx, tx, d.ID)
			if holdError != nil {
				return nil, holdError
			}
			if v+held > 999 {
				return nil, catalogError(422, "STOCK_RESERVED", "库存加上仍可退款的订单占用不能超过 999。")
			}
			d.Stock = &v
			if republish {
				d.State = "ACTIVE"
			}
		}
		if v, ok := content["active"].(bool); ok {
			d.State = "INACTIVE"
			if v {
				d.State = "ACTIVE"
			}
		}
		if e = s.catalogBind(ctx, tx, d, catalogBinding(kind), content); e != nil {
			return nil, e
		}
		if e = catalogSave(ctx, tx, &d, false); e != nil {
			return nil, e
		}
		return d, nil
	}))
}
func (s *Store) CatalogActionV2(ctx context.Context, user, id, kind, key string, expected int64, action, reason string, delta int64) (CatalogRecordV2, error) {
	current, e := s.CatalogGetRecordV2(ctx, user, id, kind, true)
	if e != nil {
		return current, e
	}
	permission := "PRODUCT_PUBLISH"
	if action == "stock-adjustments" {
		permission = "PRODUCT_EDIT"
	}
	if kind != "listing" {
		if e = s.CatalogCanManageV2(ctx, user, current.StoreID, permission); e != nil {
			return current, e
		}
	}
	if action == "publish" && current.Version == expected {
		if err := s.EnsureAssetsActiveV2(ctx, user, "PRODUCT_DRAFT", id, CatalogAssetsV2(kind, current.Body)); err != nil {
			return current, err
		}
	}
	return catalogDecodeRecord(s.catalogMutation(ctx, user, key, action+":"+id, map[string]any{"expectedVersion": expected, "reason": reason, "delta": delta}, func(tx *sql.Tx) (any, error) {
		d, e := catalogRecordTx(ctx, tx, id, kind, true)
		if e != nil {
			return nil, e
		}
		if kind == "listing" {
			if d.OwnerID != user {
				return nil, catalogDenied()
			}
		} else if e = catalogPermissionTx(ctx, tx, user, d.StoreID, permission); e != nil {
			return nil, e
		}
		if d.Version != expected {
			return nil, catalogVersion()
		}
		if d.Version >= 2147483647 {
			return nil, catalogVersion()
		}
		now := time.Now().UTC()
		switch action {
		case "publish":
			if kind != "product" || d.State == "ARCHIVED" {
				return nil, catalogError(409, "INVALID_STATE_TRANSITION", "此商品无法发布。")
			}
			if e = catalogReferences(ctx, tx, "product", d.StoreID, d.Body); e != nil {
				return nil, e
			}
			approvedTemplate, templateError := catalogDeliveryTemplateTxV2(ctx, tx, d.StoreID, catalogString(d.Body, "deliveryTemplateRef"), true)
			if templateError != nil {
				return nil, templateError
			}
			if e = catalogFreezePublicationTemplateV2(ctx, tx, d, approvedTemplate, now); e != nil {
				return nil, e
			}
			d.Published = d.Body
			d.PublishedVersion = d.Version + 1
			d.PublishedAt = &now
			d.State = "ACTIVE"
			if d.Body["inventoryPolicy"] == "UNLIMITED" {
				d.Stock = nil
			} else if d.Stock == nil {
				v := catalogNumber(d.Body, "stock")
				held, holdError := catalogHeldStockV2(ctx, tx, d.ID)
				if holdError != nil {
					return nil, holdError
				}
				if v+held > 999999 {
					return nil, catalogError(422, "STOCK_RESERVED", "当前订单仍占用部分库存。")
				}
				d.Stock = &v
			}
			if e = s.PromoteAssetBindingsV2(ctx, tx, "PRODUCT_DRAFT", "PRODUCT_PUBLISHED", d.ID, CatalogAssetsV2("product", d.Published)); e != nil {
				return nil, e
			}
		case "unlist":
			if d.State != "ACTIVE" {
				return nil, catalogError(409, "INVALID_STATE_TRANSITION", "仅在售商品可以下架。")
			}
			d.State = "UNLISTED"
		case "archive":
			if kind != "product" || d.State == "ACTIVE" || d.State == "ARCHIVED" {
				return nil, catalogError(409, "INVALID_STATE_TRANSITION", "请先下架商品后再归档。")
			}
			d.State = "ARCHIVED"
		case "stock-adjustments":
			if kind != "product" || d.State == "ARCHIVED" || d.Stock == nil || delta == 0 || delta < -999999 || delta > 999999 || *d.Stock+delta < 0 || *d.Stock+delta > 999999 {
				return nil, catalogError(422, "INVALID_STOCK", "库存调整超出范围或商品不限量。")
			}
			v := *d.Stock + delta
			held, holdError := catalogHeldStockV2(ctx, tx, d.ID)
			if holdError != nil {
				return nil, holdError
			}
			if v+held > 999999 {
				return nil, catalogError(422, "STOCK_RESERVED", "可用库存加上订单占用不能超过库存上限。")
			}
			d.Stock = &v
			d.Body["stock"] = v
		default:
			return nil, catalogInvalid()
		}
		d.Version++
		d.UpdatedAt = now
		if e = catalogSave(ctx, tx, &d, false); e != nil {
			return nil, e
		}
		return d, nil
	}))
}

func (s *Store) CatalogGetRecordV2(ctx context.Context, user, id, kind string, management bool) (CatalogRecordV2, error) {
	d, e := s.catalogRecord(ctx, id, kind)
	if e != nil {
		return d, e
	}
	if management {
		if kind == "listing" {
			if d.OwnerID != user {
				return d, catalogNotFound()
			}
		} else {
			storeID := d.StoreID
			if kind == "store" {
				storeID = d.ID
			}
			if e = s.CatalogCanManageV2(ctx, user, storeID, "READ"); e != nil {
				return d, e
			}
		}
	} else if (kind == "product" && (d.State != "ACTIVE" || d.Published == nil)) || (kind == "listing" && (d.State != "ACTIVE" || d.Stock == nil || *d.Stock <= 0) && d.OwnerID != user) || ((kind == "brand" || kind == "category") && d.State != "ACTIVE") {
		return d, catalogNotFound()
	}
	return d, nil
}

type CatalogFilterV2 struct {
	Kind, StoreID, BrandID, CategoryID, SellerRef, Query, State string
	Management                                                  bool
	Limit                                                       int
	Before                                                      int64
}

func (s *Store) CatalogListV2(ctx context.Context, user string, f CatalogFilterV2) ([]CatalogRecordV2, error) {
	if f.Limit < 1 || f.Limit > 101 {
		return nil, catalogInvalid()
	}
	if f.Management && f.Kind != "listing" {
		if e := s.CatalogCanManageV2(ctx, user, f.StoreID, "READ"); e != nil {
			return nil, e
		}
	}
	q := catalogSelect + " WHERE kind=?"
	args := []any{f.Kind}
	if f.StoreID != "" {
		q += " AND store_id=?"
		args = append(args, f.StoreID)
	}
	if f.Before > 0 {
		q += " AND sequence_id<?"
		args = append(args, f.Before)
	}
	if f.Management {
		if f.Kind == "listing" {
			q += " AND owner_id=?"
			args = append(args, user)
		}
		if f.State != "" {
			q += " AND state=?"
			args = append(args, f.State)
		}
	} else if f.Kind != "store" && f.Kind != "homepage" {
		q += " AND state='ACTIVE'"
		if f.Kind == "product" {
			q += " AND published_body IS NOT NULL"
		}
		if f.Kind == "listing" {
			q += " AND available_stock>0"
		}
	}
	if f.BrandID != "" {
		q += " AND brand_ref=?"
		args = append(args, f.BrandID)
	}
	if f.CategoryID != "" {
		q += " AND category_ref=?"
		args = append(args, f.CategoryID)
	}
	if f.SellerRef != "" {
		q += " AND owner_id IN (SELECT id FROM identities WHERE player_ref=?)"
		args = append(args, f.SellerRef)
	}
	if f.Query != "" {
		contentColumn := "body"
		if f.Kind == "product" && !f.Management {
			contentColumn = "published_body"
		}
		q += " AND (LOCATE(?,LOWER(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(" + contentColumn + ",'$.title')),title)))>0 OR LOCATE(?,LOWER(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(" + contentColumn + ",'$.subtitle')),'')))>0 OR (kind='product' AND brand_ref IN (SELECT b.resource_id FROM catalog_records_v2 b WHERE b.kind='brand' AND LOCATE(?,LOWER(b.title))>0)) OR (kind='listing' AND owner_id IN (SELECT id FROM identities WHERE LOCATE(?,LOWER(game_id))>0)))"
		args = append(args, strings.ToLower(f.Query), strings.ToLower(f.Query), strings.ToLower(f.Query), strings.ToLower(f.Query))
	}
	q += " ORDER BY sequence_id DESC LIMIT ?"
	args = append(args, f.Limit)
	rows, e := s.DB.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []CatalogRecordV2{}
	for rows.Next() {
		d, e := catalogScan(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func CatalogCursorV2(scope string, seq int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(Digest([]byte(scope))[:16] + ":" + strconv.FormatInt(seq, 10)))
}
func CatalogParseCursorV2(scope, cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	if len(cursor) > 512 {
		return 0, catalogInvalid()
	}
	raw, e := base64.RawURLEncoding.DecodeString(cursor)
	if e != nil {
		return 0, catalogInvalid()
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 2 || parts[0] != Digest([]byte(scope))[:16] {
		return 0, catalogInvalid()
	}
	n, e := strconv.ParseInt(parts[1], 10, 64)
	if e != nil || n < 1 {
		return 0, catalogInvalid()
	}
	return n, nil
}

// Call only after checking visibility of the containing catalog record. The
// uploader is obtained from the DB; clients never supply a privileged viewer ID.
func (s *Store) catalogAsset(ctx context.Context, binding, ref, id string) (any, error) {
	if id == "" {
		return nil, nil
	}
	return s.AssetForBindingV2(ctx, binding, ref, id)
}
func (s *Store) CatalogViewV2(ctx context.Context, viewer string, d CatalogRecordV2, management bool) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range d.Body {
		out[k] = v
	}
	out["version"] = d.Version
	switch d.Kind {
	case "store":
		out["storeId"] = d.ID
		for _, k := range []string{"logo", "cover"} {
			asset, e := s.catalogAsset(ctx, "STORE_PROFILE", d.ID, catalogString(d.Body, k+"AssetId"))
			if e != nil {
				return nil, e
			}
			out[k] = asset
			delete(out, k+"AssetId")
		}
	case "brand":
		out["brandId"] = d.ID
	case "category":
		out["categoryId"] = d.ID
	case "homepage":
		out["storeId"] = d.StoreID
	case "product":
		var published any
		if d.Published != nil {
			images := []any{}
			for _, id := range catalogIDs(d.Published, "galleryAssetIds") {
				a, e := s.catalogAsset(ctx, "PRODUCT_PUBLISHED", d.ID, id)
				if e != nil {
					return nil, e
				}
				images = append(images, a)
			}
			state := d.State
			if state == "DRAFT" {
				state = "UNLISTED"
			}
			published = map[string]any{"productId": d.ID, "storeId": d.StoreID, "version": d.PublishedVersion, "content": d.Published, "images": images, "visibility": state, "availableStock": d.Stock, "publishedAt": d.PublishedAt, "updatedAt": d.UpdatedAt}
		}
		if !management {
			if published == nil {
				return nil, catalogNotFound()
			}
			return published.(map[string]any), nil
		}
		draftImages := []any{}
		for _, id := range catalogIDs(d.Body, "galleryAssetIds") {
			asset, e := s.catalogAsset(ctx, "PRODUCT_DRAFT", d.ID, id)
			if e != nil {
				return nil, e
			}
			draftImages = append(draftImages, asset)
		}
		return map[string]any{"productId": d.ID, "storeId": d.StoreID, "version": d.Version, "draft": d.Body, "draftImages": draftImages, "published": published, "visibility": d.State, "updatedAt": d.UpdatedAt}, nil
	case "listing":
		var sellerRef string
		if e := s.DB.QueryRowContext(ctx, "SELECT player_ref FROM identities WHERE id=?", d.OwnerID).Scan(&sellerRef); e != nil {
			return nil, e
		}
		seller, e := s.PublicProfileV2(ctx, viewer, sellerRef)
		if e != nil {
			return nil, e
		}
		photos := []any{}
		for _, id := range catalogIDs(d.Body, "photoAssetIds") {
			a, e := s.catalogAsset(ctx, "MARKET_LISTING", d.ID, id)
			if e != nil {
				return nil, e
			}
			photos = append(photos, a)
		}
		out["listingId"] = d.ID
		out["seller"] = seller
		out["photos"] = photos
		out["active"] = d.State == "ACTIVE"
		out["stock"] = d.Stock
		out["createdAt"] = d.CreatedAt
		out["updatedAt"] = d.UpdatedAt
	default:
		return nil, catalogInvalid()
	}
	return out, nil
}

type CatalogCartItemV2 struct {
	ProductID      string `json:"productId"`
	Quantity       int64  `json:"quantity"`
	ProductVersion int64  `json:"productVersion"`
}
type CatalogCartV2 struct {
	Version   int64               `json:"version"`
	Items     []CatalogCartItemV2 `json:"items"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

func catalogCartTx(ctx context.Context, tx *sql.Tx, user string, lock bool) (c CatalogCartV2, err error) {
	c.Items = []CatalogCartItemV2{}
	q := "SELECT version,updated_at FROM catalog_carts_v2 WHERE user_id=?"
	if lock {
		q += " FOR UPDATE"
	}
	err = tx.QueryRowContext(ctx, q, user).Scan(&c.Version, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.Version = 1
		c.UpdatedAt = time.Unix(0, 0).UTC()
		return c, nil
	}
	if err != nil {
		return
	}
	rows, e := tx.QueryContext(ctx, "SELECT product_id,quantity,product_version FROM catalog_cart_items_v2 WHERE user_id=? ORDER BY product_id", user)
	if e != nil {
		return c, e
	}
	defer rows.Close()
	for rows.Next() {
		var i CatalogCartItemV2
		if e = rows.Scan(&i.ProductID, &i.Quantity, &i.ProductVersion); e != nil {
			return c, e
		}
		c.Items = append(c.Items, i)
	}
	return c, rows.Err()
}
func (s *Store) CatalogCartV2(ctx context.Context, user string) (CatalogCartV2, error) {
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return CatalogCartV2{}, e
	}
	defer tx.Rollback()
	return catalogCartTx(ctx, tx, user, false)
}
func (s *Store) CatalogCartChangeV2(ctx context.Context, user, product, key string, expected, quantity int64, remove bool) (CatalogCartV2, error) {
	var result CatalogCartV2
	if expected < 1 || (!remove && (quantity < 1 || quantity > 999)) {
		return result, catalogInvalid()
	}
	raw, e := s.catalogMutation(ctx, user, key, "cart:"+product, map[string]any{"expectedVersion": expected, "quantity": quantity, "remove": remove}, func(tx *sql.Tx) (any, error) {
		if _, e := tx.ExecContext(ctx, "INSERT IGNORE INTO catalog_carts_v2 VALUES(?,1,UTC_TIMESTAMP(6))", user); e != nil {
			return nil, e
		}
		cart, e := catalogCartTx(ctx, tx, user, true)
		if e != nil {
			return nil, e
		}
		if cart.Version != expected {
			return nil, catalogVersion()
		}
		if cart.Version >= 2147483647 {
			return nil, catalogVersion()
		}
		if remove {
			if _, e = tx.ExecContext(ctx, "DELETE FROM catalog_cart_items_v2 WHERE user_id=? AND product_id=?", user, product); e != nil {
				return nil, e
			}
		} else {
			d, e := catalogRecordTx(ctx, tx, product, "product", true)
			if e != nil {
				return nil, e
			}
			if d.State != "ACTIVE" || d.Published == nil {
				return nil, catalogNotFound()
			}
			if quantity > catalogNumber(d.Published, "limitPerOrder") || (d.Stock != nil && quantity > *d.Stock) {
				return nil, catalogError(422, "INSUFFICIENT_STOCK", "数量超过库存或单笔限购。")
			}
			found := false
			for _, i := range cart.Items {
				if i.ProductID == product {
					found = true
				}
			}
			if !found && len(cart.Items) >= 100 {
				return nil, catalogError(422, "CART_LIMIT", "购物袋最多容纳 100 种商品。")
			}
			if _, e = tx.ExecContext(ctx, "INSERT INTO catalog_cart_items_v2 VALUES(?,?,?,?) ON DUPLICATE KEY UPDATE quantity=VALUES(quantity),product_version=VALUES(product_version)", user, product, quantity, d.PublishedVersion); e != nil {
				return nil, e
			}
		}
		if _, e = tx.ExecContext(ctx, "UPDATE catalog_carts_v2 SET version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE user_id=?", user); e != nil {
			return nil, e
		}
		return catalogCartTx(ctx, tx, user, false)
	})
	if e == nil {
		e = json.Unmarshal(raw, &result)
	}
	return result, e
}

func (s *Store) CatalogQuoteV2(ctx context.Context, user, key string, input CatalogObjectV2) (CatalogObjectV2, error) {
	if e := CatalogQuoteInputV2(input); e != nil {
		return nil, e
	}
	now := time.Now().UTC()
	if key == "" {
		key = "quote:" + Digest([]byte(catalogJSON(input) + strconv.FormatInt(now.Unix()/300, 10)))[:48]
	}
	raw, e := s.catalogMutation(ctx, user, key, "quote", input, func(tx *sql.Tx) (any, error) {
		items := input["items"].([]any)
		sorted := append([]any(nil), items...)
		sort.Slice(sorted, func(i, j int) bool {
			a, _ := catalogObject(sorted[i])
			b, _ := catalogObject(sorted[j])
			return catalogString(a, "productId") < catalogString(b, "productId")
		})
		docs := map[string]CatalogRecordV2{}
		quantities := map[string]int64{}
		channel := catalogString(input, "channel")
		kind := "product"
		if channel == "PLAYER_MARKET" {
			kind = "listing"
		}
		for _, v := range sorted {
			o, _ := catalogObject(v)
			id := catalogString(o, "productId")
			d, e := catalogRecordTx(ctx, tx, id, kind, true)
			if e != nil {
				return nil, e
			}
			if d.State != "ACTIVE" {
				return nil, catalogNotFound()
			}
			if kind == "listing" && d.OwnerID == user {
				return nil, catalogError(422, "SELF_PURCHASE", "不能购买自己发布的商品。")
			}
			docs[id] = d
			quantities[id] = catalogNumber(o, "quantity")
		}
		lines := []any{}
		total := new(big.Int)
		delivery, _ := catalogObject(input["delivery"])
		for _, v := range items {
			o, _ := catalogObject(v)
			d := docs[catalogString(o, "productId")]
			content, version := d.Body, d.Version
			if kind == "product" {
				content = d.Published
				version = d.PublishedVersion
			}
			if content == nil {
				return nil, catalogNotFound()
			}
			if version != catalogNumber(o, "expectedProductVersion") {
				return nil, catalogVersion()
			}
			quantity := catalogNumber(o, "quantity")
			if d.Stock != nil && quantity > *d.Stock {
				return nil, catalogError(422, "INSUFFICIENT_STOCK", "商品库存不足。")
			}
			if kind == "product" && quantity > catalogNumber(content, "limitPerOrder") {
				return nil, catalogError(422, "PURCHASE_LIMIT", "数量超过单笔限购。")
			}
			if kind == "listing" {
				methods, _ := catalogStrings(content["deliveryMethods"], 1, 2, 1, 16, true)
				matched := false
				for _, m := range methods {
					if m == delivery["method"] {
						matched = true
					}
				}
				if !matched {
					return nil, catalogError(422, "DELIVERY_UNAVAILABLE", "此商品不支持选择的交付方式。")
				}
			}
			price, ok := catalogMoney(content["price"])
			if !ok {
				return nil, catalogError(503, "INVALID_CATALOG_PRICE", "商品价格需要管理员修正。")
			}
			subtotal := new(big.Int).Mul(price, big.NewInt(quantity))
			total.Add(total, subtotal)
			if len(catalogMoneyString(total)) > 20 {
				return nil, catalogError(422, "AMOUNT_LIMIT", "报价总金额超出限制。")
			}
			lines = append(lines, map[string]any{"productId": d.ID, "title": content["title"], "unitPrice": catalogMoneyString(price), "quantity": quantity, "subtotal": catalogMoneyString(subtotal), "productVersion": version})
		}
		if e := commerceWithinExecutionLimitV2(catalogMoneyString(total)); e != nil {
			return nil, e
		}
		if kind == "product" {
			if _, e := commerceDeliveryContentsV2(ctx, tx, docs, quantities, nil); e != nil {
				return nil, e
			}
		}
		quote := CatalogObjectV2{"quoteId": ID("quote_"), "channel": channel, "items": lines, "totalAmount": catalogMoneyString(total), "currency": "CREDIT", "expiresAt": now.Add(5 * time.Minute), "delivery": delivery, "version": 1, "warnings": []string{"报价不预留库存，付款时将重新校验库存及交付条件。"}}
		if _, e := tx.ExecContext(ctx, "INSERT INTO catalog_quotes_v2 VALUES(?,?,?,?,?,?)", quote["quoteId"], user, channel, catalogJSON(quote), now, now.Add(5*time.Minute)); e != nil {
			return nil, e
		}
		return quote, nil
	})
	var result CatalogObjectV2
	if e == nil {
		e = json.Unmarshal(raw, &result)
	}
	return result, e
}

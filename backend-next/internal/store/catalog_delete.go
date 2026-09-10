package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Keep the catalog record for frozen orders, stock restoration and audit, while
// making deletion terminal for all catalog management and checkout entry points.
func (s *Store) DeleteCatalogEntry(ctx context.Context, actor, id, kind, key string, expected int64) (json.RawMessage, error) {
	if kind != "coupon" && kind != "product" || expected < 1 {
		return nil, catalogInvalid()
	}
	return s.catalogMutation(ctx, actor, key, "delete:"+kind+":"+id, CatalogObjectV2{"expectedVersion": expected}, func(tx *sql.Tx) (any, error) {
		d, e := catalogRecordTx(ctx, tx, id, kind, true)
		if e != nil {
			return nil, e
		}
		if kind == "coupon" {
			e = requirePlatformAdminTxV206(ctx, tx, actor)
		} else {
			e = catalogPermissionTx(ctx, tx, actor, d.StoreID, "PRODUCT_PUBLISH")
		}
		if e != nil {
			return nil, e
		}
		if d.Version != expected || d.Version >= 2147483647 {
			return nil, catalogVersion()
		}
		if d.State == "DELETED" {
			return nil, catalogNotFound()
		}
		if kind == "coupon" && d.State != "DRAFT" && d.State != "INACTIVE" {
			return nil, catalogError(409, "COUPON_DISABLE_REQUIRED", "请先停用优惠券，再删除。")
		}
		d.State, d.UpdatedAt = "DELETED", time.Now().UTC()
		d.Version++
		if kind == "coupon" {
			d.Body["active"] = false
		}
		if e = catalogSave(ctx, tx, &d, false); e != nil {
			return nil, e
		}
		return CatalogObjectV2{kind + "Id": id, "deleted": true, "version": d.Version}, nil
	})
}

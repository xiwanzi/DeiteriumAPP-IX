package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func (s *Store) CatalogWriteHomepageV2(ctx context.Context, user, storeID, key string, expected int64, content CatalogObjectV2) (CatalogRecordV2, error) {
	if e := ValidateCatalogContentV2("homepage", content); e != nil {
		return CatalogRecordV2{}, e
	}
	if e := s.CatalogCanManageV2(ctx, user, storeID, "STORE_EDIT"); e != nil {
		return CatalogRecordV2{}, e
	}
	return catalogDecodeRecord(s.catalogMutation(ctx, user, key, "homepage:"+storeID, map[string]any{"expectedVersion": expected, "content": content}, func(tx *sql.Tx) (any, error) {
		if e := catalogPermissionTx(ctx, tx, user, storeID, "STORE_EDIT"); e != nil {
			return nil, e
		}
		if _, e := catalogRecordTx(ctx, tx, storeID, "store", true); e != nil {
			return nil, e
		}
		d, e := catalogScan(tx.QueryRowContext(ctx, catalogSelect+" WHERE kind='homepage' AND store_id=? ORDER BY sequence_id DESC LIMIT 1 FOR UPDATE", storeID))
		var problem *CatalogErrorV2
		create := errors.As(e, &problem) && problem.Code == "NOT_FOUND"
		if e != nil && !create {
			return nil, e
		}
		now := time.Now().UTC()
		if create {
			if expected != 1 {
				return nil, catalogVersion()
			}
			d = CatalogRecordV2{ID: ID("home_"), Kind: "homepage", StoreID: storeID, OwnerID: user, Version: 1, State: "ACTIVE", CreatedAt: now}
		} else {
			if d.Version != expected || d.Version >= 2147483647 {
				return nil, catalogVersion()
			}
			d.Version++
		}
		d.OwnerID = user
		d.Body = content
		d.UpdatedAt = now
		if e = catalogReferences(ctx, tx, "homepage", storeID, content); e != nil {
			return nil, e
		}
		if e = s.catalogBind(ctx, tx, d, "STORE_HOMEPAGE", content); e != nil {
			return nil, e
		}
		if e = catalogSave(ctx, tx, &d, create); e != nil {
			return nil, e
		}
		return d, nil
	}))
}

func (s *Store) CatalogHomepageV2(ctx context.Context, user, storeID string, management bool) (CatalogRecordV2, error) {
	if _, e := s.CatalogGetRecordV2(ctx, user, storeID, "store", management); e != nil {
		return CatalogRecordV2{}, e
	}
	return catalogScan(s.DB.QueryRowContext(ctx, catalogSelect+" WHERE kind='homepage' AND store_id=? ORDER BY sequence_id DESC LIMIT 1", storeID))
}
func (s *Store) CatalogMerchantV2(ctx context.Context, u User) (map[string]any, error) {
	global, e := s.HasPermission(ctx, u.ID, "store.manage")
	if e != nil {
		return nil, e
	}
	q := "SELECT resource_id FROM catalog_records_v2 WHERE kind='store'"
	args := []any{}
	if !global {
		q += " AND resource_id IN (SELECT store_id FROM catalog_members_v2 WHERE user_id=? AND active=TRUE)"
		args = append(args, u.ID)
	}
	q += " ORDER BY sequence_id DESC LIMIT 20"
	rows, e := s.DB.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if len(ids) == 0 {
		return nil, catalogError(403, "STORE_NOT_CONFIGURED", "尚未分配店铺；平台管理员可先创建正式店铺。")
	}
	permissions := []string{"STORE_EDIT", "PRODUCT_EDIT", "PRODUCT_PUBLISH", "ORDER_MANAGE"}
	if !global {
		permissions = []string{"STORE_READ"}
	}
	return map[string]any{"playerRef": u.PlayerRef, "storeIds": ids, "permissions": permissions}, nil
}
func (s *Store) CatalogTemplatesV2(ctx context.Context, user, storeID string, limit int) ([]map[string]any, error) {
	rows, e := s.CatalogListV2(ctx, user, CatalogFilterV2{Kind: "delivery_template", StoreID: storeID, Management: true, Limit: limit})
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for _, d := range rows {
		out = append(out, CatalogTemplateViewV2(d, false))
	}
	return out, nil
}

type CatalogMemberV2 struct {
	PlayerRef   string   `json:"playerRef"`
	Permissions []string `json:"permissions"`
	Version     int64    `json:"version"`
}

func (s *Store) CatalogMembersV2(ctx context.Context, storeID string) ([]CatalogMemberV2, error) {
	rows, e := s.DB.QueryContext(ctx, "SELECT i.player_ref,m.permissions,m.version FROM catalog_members_v2 m JOIN identities i ON i.id=m.user_id WHERE m.store_id=? AND m.active=TRUE ORDER BY i.player_ref LIMIT 100", storeID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []CatalogMemberV2{}
	for rows.Next() {
		var m CatalogMemberV2
		var raw string
		if e = rows.Scan(&m.PlayerRef, &raw, &m.Version); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(raw), &m.Permissions); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) CatalogSetMemberV2(ctx context.Context, actor, storeID, key, playerRef string, expected int64, permissions []string, revoke bool) (CatalogMemberV2, error) {
	var result CatalogMemberV2
	allowed, e := s.HasPermission(ctx, actor, "store.members.manage")
	if e != nil {
		return result, e
	}
	if !allowed {
		return result, catalogDenied()
	}
	if !CatalogReferenceV2(playerRef) || expected < 1 || len(permissions) > 4 || (!revoke && len(permissions) == 0) {
		return result, catalogInvalid()
	}
	seen := map[string]bool{}
	for _, p := range permissions {
		if !catalogEnum(p, "STORE_EDIT", "PRODUCT_EDIT", "PRODUCT_PUBLISH", "ORDER_MANAGE") || seen[p] {
			return result, catalogInvalid()
		}
		seen[p] = true
	}
	raw, e := s.catalogMutation(ctx, actor, key, "member:"+storeID+":"+playerRef, map[string]any{"expectedVersion": expected, "permissions": permissions, "revoke": revoke}, func(tx *sql.Tx) (any, error) {
		if _, e := catalogRecordTx(ctx, tx, storeID, "store", true); e != nil {
			return nil, e
		}
		var user string
		if e := tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE player_ref=? AND status='active'", playerRef).Scan(&user); errors.Is(e, sql.ErrNoRows) {
			return nil, catalogNotFound()
		} else if e != nil {
			return nil, e
		}
		var oldVersion int64
		e := tx.QueryRowContext(ctx, "SELECT version FROM catalog_members_v2 WHERE store_id=? AND user_id=? FOR UPDATE", storeID, user).Scan(&oldVersion)
		create := errors.Is(e, sql.ErrNoRows)
		if e != nil && !create {
			return nil, e
		}
		if create {
			if revoke || expected != 1 {
				return nil, catalogVersion()
			}
			oldVersion = 0
		} else if oldVersion != expected || oldVersion >= 2147483647 {
			return nil, catalogVersion()
		}
		next := oldVersion + 1
		if _, e = tx.ExecContext(ctx, "INSERT INTO catalog_members_v2 VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE permissions=VALUES(permissions),version=VALUES(version),active=VALUES(active)", storeID, user, catalogJSON(permissions), next, !revoke); e != nil {
			return nil, e
		}
		return CatalogMemberV2{playerRef, permissions, next}, nil
	})
	if e == nil {
		e = json.Unmarshal(raw, &result)
	}
	return result, e
}

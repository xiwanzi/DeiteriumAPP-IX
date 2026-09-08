package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"time"
)

func catalogPublicationTemplateIDV2(product string, version int64) string {
	return "pubtpl_" + Digest([]byte(product + ":" + strconv.FormatInt(version, 10)))[:40]
}
func catalogFreezePublicationTemplateV2(ctx context.Context, tx *sql.Tx, product, template CatalogRecordV2, now time.Time) error {
	body := CatalogObjectV2{"productId": product.ID, "productVersion": product.Version + 1, "templateRef": template.ID, "templateVersion": template.Version, "templateContent": template.Body}
	d := CatalogRecordV2{ID: catalogPublicationTemplateIDV2(product.ID, product.Version+1), Kind: "publication_template", StoreID: product.StoreID, OwnerID: product.OwnerID, Version: 1, Body: body, State: "FROZEN", CreatedAt: now, UpdatedAt: now}
	return catalogSave(ctx, tx, &d, true)
}
func catalogPublishedTemplateV2(ctx context.Context, tx *sql.Tx, product CatalogRecordV2, nodes map[string]CatalogNodePolicyV2) (CatalogRecordV2, error) {
	frozen, e := catalogRecordTx(ctx, tx, catalogPublicationTemplateIDV2(product.ID, product.PublishedVersion), "publication_template", false)
	if e != nil {
		return frozen, e
	}
	ref := catalogString(frozen.Body, "templateRef")
	if ref != catalogString(product.Published, "deliveryTemplateRef") {
		return frozen, catalogError(503, "PUBLICATION_SNAPSHOT_INVALID", "商品发布快照需要核对。")
	}
	current, e := catalogRecordTx(ctx, tx, ref, "delivery_template", true)
	if e != nil {
		return frozen, e
	}
	if current.StoreID != product.StoreID || current.State != "ACTIVE" {
		return frozen, catalogError(422, "DELIVERY_TEMPLATE_UNAVAILABLE", "商品交付模板已停用。")
	}
	content, ok := catalogObject(frozen.Body["templateContent"])
	if !ok {
		return frozen, catalogInvalid()
	}
	if e = catalogValidateTemplateCoreV2(ctx, tx, content, nodes); e != nil {
		return frozen, e
	}
	return CatalogRecordV2{ID: ref, Kind: "delivery_template", StoreID: product.StoreID, Version: catalogNumber(frozen.Body, "templateVersion"), Body: content, State: "ACTIVE"}, nil
}

type CatalogNodePolicyV2 struct {
	InventoryDomain string
	ClaimEnabled    bool
}

var catalogItemRefV2 = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}:[a-z0-9_./-]{1,63}$`)
var catalogSHA256V2 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidateDeliveryTemplateV2(content CatalogObjectV2) error {
	if !catalogFields(content, "name summary inventoryDomain allowedServerIds attachments active", "") || !catalogText(content["name"], 1, 80) || !catalogText(content["summary"], 1, 1000) || !CatalogReferenceV2(catalogString(content, "inventoryDomain")) {
		return catalogInvalid()
	}
	if _, ok := content["active"].(bool); !ok {
		return catalogInvalid()
	}
	if _, ok := catalogRefs(content["allowedServerIds"], 1, 32); !ok {
		return catalogInvalid()
	}
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) < 1 || len(attachments) > 32 {
		return catalogInvalid()
	}
	seen := map[string]bool{}
	for _, v := range attachments {
		a, ok := catalogObject(v)
		if !ok || !catalogFields(a, "itemRef revision quantity payloadSha256", "") || !catalogItemRefV2.MatchString(catalogString(a, "itemRef")) || !catalogRange(a["revision"], 1, 2147483647) || !catalogRange(a["quantity"], 1, 99999) || !catalogSHA256V2.MatchString(catalogString(a, "payloadSha256")) {
			return catalogInvalid()
		}
		key := catalogString(a, "itemRef")
		if seen[key] {
			return catalogInvalid()
		}
		seen[key] = true
	}
	return nil
}
func catalogValidateTemplateCoreV2(ctx context.Context, tx *sql.Tx, content CatalogObjectV2, nodes map[string]CatalogNodePolicyV2) error {
	if e := ValidateDeliveryTemplateV2(content); e != nil {
		return e
	}
	var tables int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='core_catalog_heads'").Scan(&tables); e != nil {
		return e
	}
	if tables != 1 {
		return catalogError(503, "CORE_CATALOG_UNAVAILABLE", "Core 物品目录尚未同步完成。")
	}
	domain := catalogString(content, "inventoryDomain")
	servers := catalogIDs(content, "allowedServerIds")
	if nodes != nil {
		for _, id := range servers {
			node, ok := nodes[id]
			if !ok || !node.ClaimEnabled || node.InventoryDomain != domain {
				return catalogError(422, "INVALID_DELIVERY_SCOPE", "选择的领取服务器未获准或不属于同一背包域。")
			}
		}
	}
	attachments := append([]any(nil), content["attachments"].([]any)...)
	sort.Slice(attachments, func(i, j int) bool {
		a, _ := catalogObject(attachments[i])
		b, _ := catalogObject(attachments[j])
		return catalogString(a, "itemRef") < catalogString(b, "itemRef")
	})
	for _, v := range attachments {
		a, _ := catalogObject(v)
		var raw, sha string
		var archived bool
		var latest int64
		e := tx.QueryRowContext(ctx, "SELECT v.metadata,v.payload_sha256,h.archived,h.latest_revision FROM item_versions v JOIN core_catalog_heads h ON h.item_ref=v.item_ref WHERE v.item_ref=? AND v.revision=? FOR UPDATE", catalogString(a, "itemRef"), catalogNumber(a, "revision")).Scan(&raw, &sha, &archived, &latest)
		if e == sql.ErrNoRows {
			return catalogError(422, "ITEM_VERSION_UNAVAILABLE", "物品版本尚未由 Core 登记。")
		}
		if e != nil {
			return e
		}
		if archived || latest < catalogNumber(a, "revision") {
			return catalogError(422, "ITEM_ARCHIVED", "物品已归档或目录状态尚未同步。")
		}
		var metadata CatalogObjectV2
		if e = json.Unmarshal([]byte(raw), &metadata); e != nil {
			return e
		}
		compatible, ok := catalogRefs(metadata["compatibleServerIds"], 1, 32)
		if !ok || sha != catalogString(a, "payloadSha256") || catalogNumber(a, "quantity") > catalogNumber(metadata, "maxQuantity") || catalogString(metadata, "inventoryDomain") != domain || catalogString(metadata, "codec") != "bukkit-bytes-v1" {
			return catalogError(422, "ITEM_SNAPSHOT_MISMATCH", "附件摘要、数量或背包域不符合 Core 的精确物品版本。")
		}
		for _, id := range servers {
			found := false
			for _, compatibleID := range compatible {
				if id == compatibleID {
					found = true
				}
			}
			if !found {
				return catalogError(422, "ITEM_INCOMPATIBLE", "附件不兼容选择的领取服务器。")
			}
		}
	}
	return nil
}
func (s *Store) CatalogWriteTemplateV2(ctx context.Context, user, storeID, id, key string, expected int64, content CatalogObjectV2, nodes map[string]CatalogNodePolicyV2) (CatalogRecordV2, error) {
	if e := ValidateDeliveryTemplateV2(content); e != nil {
		return CatalogRecordV2{}, e
	}
	if e := s.CatalogCanManageV2(ctx, user, storeID, "PRODUCT_EDIT"); e != nil {
		return CatalogRecordV2{}, e
	}
	scope := "template-create:" + storeID
	if id != "" {
		scope = "template-edit:" + id
	}
	return catalogDecodeRecord(s.catalogMutation(ctx, user, key, scope, map[string]any{"expectedVersion": expected, "content": content}, func(tx *sql.Tx) (any, error) {
		if e := catalogPermissionTx(ctx, tx, user, storeID, "PRODUCT_EDIT"); e != nil {
			return nil, e
		}
		if _, e := catalogRecordTx(ctx, tx, storeID, "store", false); e != nil {
			return nil, e
		}
		now := time.Now().UTC()
		d := CatalogRecordV2{ID: ID("template_"), Kind: "delivery_template", StoreID: storeID, OwnerID: user, Version: 1, Body: content, State: "ACTIVE", CreatedAt: now, UpdatedAt: now}
		create := id == ""
		if !create {
			var e error
			d, e = catalogRecordTx(ctx, tx, id, "delivery_template", true)
			if e != nil {
				return nil, e
			}
			if d.StoreID != storeID {
				return nil, catalogNotFound()
			}
			if d.Version != expected || d.Version >= 2147483647 {
				return nil, catalogVersion()
			}
			d.Version++
			d.OwnerID = user
			d.Body = content
			d.UpdatedAt = now
		}
		if e := catalogValidateTemplateCoreV2(ctx, tx, content, nodes); e != nil {
			return nil, e
		}
		d.State = "INACTIVE"
		if content["active"] == true {
			d.State = "ACTIVE"
		}
		if e := catalogSave(ctx, tx, &d, create); e != nil {
			return nil, e
		}
		return d, nil
	}))
}
func (s *Store) CatalogDisableTemplateV2(ctx context.Context, user, storeID, id, key string, expected int64) (CatalogRecordV2, error) {
	if e := s.CatalogCanManageV2(ctx, user, storeID, "PRODUCT_EDIT"); e != nil {
		return CatalogRecordV2{}, e
	}
	return catalogDecodeRecord(s.catalogMutation(ctx, user, key, "template-disable:"+id, map[string]any{"expectedVersion": expected}, func(tx *sql.Tx) (any, error) {
		if e := catalogPermissionTx(ctx, tx, user, storeID, "PRODUCT_EDIT"); e != nil {
			return nil, e
		}
		d, e := catalogRecordTx(ctx, tx, id, "delivery_template", true)
		if e != nil {
			return nil, e
		}
		if d.StoreID != storeID {
			return nil, catalogNotFound()
		}
		if d.Version != expected || d.Version >= 2147483647 {
			return nil, catalogVersion()
		}
		d.Version++
		d.State = "INACTIVE"
		d.Body["active"] = false
		d.UpdatedAt = time.Now().UTC()
		if e = catalogSave(ctx, tx, &d, false); e != nil {
			return nil, e
		}
		return d, nil
	}))
}
func CatalogTemplateViewV2(d CatalogRecordV2, detail bool) map[string]any {
	out := map[string]any{"templateRef": d.ID, "name": d.Body["name"], "summary": d.Body["summary"], "active": d.State == "ACTIVE"}
	if detail {
		out["storeId"] = d.StoreID
		out["version"] = d.Version
		out["inventoryDomain"] = d.Body["inventoryDomain"]
		out["allowedServerIds"] = d.Body["allowedServerIds"]
		out["attachments"] = d.Body["attachments"]
		out["createdAt"] = d.CreatedAt
		out["updatedAt"] = d.UpdatedAt
	}
	return out
}
func catalogDeliveryTemplateTxV2(ctx context.Context, tx *sql.Tx, storeID, id string, requireActive bool) (CatalogRecordV2, error) {
	d, e := catalogRecordTx(ctx, tx, id, "delivery_template", true)
	if e != nil {
		return d, e
	}
	if d.StoreID != storeID {
		return d, catalogNotFound()
	}
	if requireActive {
		if d.State != "ACTIVE" {
			return d, catalogError(422, "DELIVERY_TEMPLATE_UNAVAILABLE", "交付模板已停用。")
		}
		if e = catalogValidateTemplateCoreV2(ctx, tx, d.Body, nil); e != nil {
			return d, e
		}
	}
	return d, nil
}

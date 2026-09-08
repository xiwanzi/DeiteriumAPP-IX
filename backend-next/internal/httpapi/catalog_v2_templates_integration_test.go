//go:build integration

package httpapi

import (
	"context"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestCatalogDeliveryTemplatesRejectForgedVersionsAndArchivedHeads(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, content := catalogProductFixture(t, f)
	templateID := content["deliveryTemplateRef"].(string)
	template, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, templateID, "delivery_template", true)
	if e != nil {
		t.Fatal(e)
	}
	nodes := map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}, "login": {InventoryDomain: "login", ClaimEnabled: false}}
	if _, e = f.s.CatalogWriteTemplateV2(ctx, f.buyer.ID, product.StoreID, "", "steal-template", 0, template.Body, nodes); e == nil {
		t.Fatal("unauthorized merchant created template")
	}
	wrong := catalogObjectClone(template.Body)
	wrong["attachments"].([]any)[0].(map[string]any)["payloadSha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
	_, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, product.StoreID, "", "wrong-digest", 0, wrong, nodes)
	assertCatalogCode(t, e, "ITEM_SNAPSHOT_MISMATCH")
	wrong = catalogObjectClone(template.Body)
	wrong["attachments"].([]any)[0].(map[string]any)["quantity"] = 65
	_, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, product.StoreID, "", "wrong-quantity", 0, wrong, nodes)
	assertCatalogCode(t, e, "ITEM_SNAPSHOT_MISMATCH")
	wrong = catalogObjectClone(template.Body)
	wrong["allowedServerIds"] = []any{"login"}
	_, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, product.StoreID, "", "wrong-node", 0, wrong, nodes)
	assertCatalogCode(t, e, "INVALID_DELIVERY_SCOPE")
	if _, e = f.s.DB.Exec("UPDATE core_catalog_heads SET archived=TRUE WHERE item_ref='catalog:stone'"); e != nil {
		t.Fatal(e)
	}
	_, e = f.s.CatalogActionV2(ctx, f.admin.ID, product.ID, "product", "archive-publish", product.Version, "publish", "", 0)
	assertCatalogCode(t, e, "ITEM_ARCHIVED")
	quote := store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": product.ID, "quantity": 1, "expectedProductVersion": product.PublishedVersion}}, "delivery": map[string]any{"method": "MAILBOX"}}
	_, e = f.s.CatalogQuoteV2(ctx, f.buyer.ID, "archived-quote", quote)
	assertCatalogCode(t, e, "ITEM_ARCHIVED")
	disabled, e := f.s.CatalogDisableTemplateV2(ctx, f.admin.ID, product.StoreID, templateID, "disable", template.Version)
	if e != nil || disabled.State != "INACTIVE" {
		t.Fatal(disabled.State, e)
	}
	if _, e = f.s.DB.Exec("UPDATE core_catalog_heads SET archived=FALSE WHERE item_ref='catalog:stone'"); e != nil {
		t.Fatal(e)
	}
	_, e = f.s.CatalogQuoteV2(ctx, f.buyer.ID, "disabled-quote", quote)
	assertCatalogCode(t, e, "DELIVERY_TEMPLATE_UNAVAILABLE")
	original := catalogObjectClone(template.Body)
	original["active"] = true
	restored, e := f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, product.StoreID, templateID, "restore-template", disabled.Version, original, nodes)
	if e != nil || restored.State != "ACTIVE" || restored.Version != 3 {
		t.Fatal(restored.State, e)
	}
	list, e := f.s.CatalogTemplatesV2(ctx, f.admin.ID, product.StoreID, 20)
	if e != nil || len(list) != 1 {
		t.Fatal(list, e)
	}
	if _, leaked := list[0]["attachments"]; leaked {
		t.Fatal("legacy thin template response changed shape")
	}
}

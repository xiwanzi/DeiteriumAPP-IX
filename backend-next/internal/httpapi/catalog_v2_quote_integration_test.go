//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func quoteReviewProductV2(t *testing.T, f catalogFixture, content store.CatalogObjectV2, shop, key string) store.CatalogRecordV2 {
	t.Helper()
	ctx := context.Background()
	p, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "product", shop, key, content)
	if e != nil {
		t.Fatal(e)
	}
	p, e = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", key+"-publish", p.Version, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func quoteReviewAcceptedV2(t *testing.T, f catalogFixture, key string, products []store.CatalogRecordV2, quantities []int) store.CatalogObjectV2 {
	t.Helper()
	items := []any{}
	for i, p := range products {
		items = append(items, map[string]any{"productId": p.ID, "quantity": quantities[i], "expectedProductVersion": p.PublishedVersion})
	}
	q, e := f.s.CatalogQuoteV2(context.Background(), f.buyer.ID, key, store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": items, "delivery": map[string]any{"method": "MAILBOX"}})
	if e != nil {
		t.Fatal(e)
	}
	return q
}
func quoteReviewRejectedV2(t *testing.T, f catalogFixture, key string, products []store.CatalogRecordV2, quantities []int, code string) {
	t.Helper()
	items := []any{}
	for i, p := range products {
		items = append(items, map[string]any{"productId": p.ID, "quantity": quantities[i], "expectedProductVersion": p.PublishedVersion})
	}
	_, e := f.s.CatalogQuoteV2(context.Background(), f.buyer.ID, key, store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": items, "delivery": map[string]any{"method": "MAILBOX"}})
	assertCatalogCode(t, e, code)
}
func TestCatalogQuoteRejectsUndeliverableAggregate(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	template, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, fmt.Sprint(content["deliveryTemplateRef"]), "delivery_template", true)
	if e != nil {
		t.Fatal(e)
	}
	changed := catalogObjectClone(template.Body)
	changed["attachments"].([]any)[0].(map[string]any)["quantity"] = 40
	_, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, p.StoreID, template.ID, "review-template", template.Version, changed, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	a := quoteReviewProductV2(t, f, content, p.StoreID, "review-a")
	b := quoteReviewProductV2(t, f, content, p.StoreID, "review-b")
	for i, group := range []struct {
		products   []store.CatalogRecordV2
		quantities []int
	}{{[]store.CatalogRecordV2{a}, []int{2}}, {[]store.CatalogRecordV2{a, b}, []int{1, 1}}} {
		quoteReviewRejectedV2(t, f, fmt.Sprintf("review-quote-%d", i), group.products, group.quantities, "DELIVERY_QUANTITY_LIMIT")
		t.Log("Quote now rejects aggregate 80 units > item max64 before payment confirmation")
	}
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 0 || catalogTestCount(t, f.s, "commerce_operations_v2") != 0 || catalogTestCount(t, f.s, "commerce_stock_holds_v2") != 0 {
		t.Fatal("rejected aggregate wrote finance intent")
	}
}
func TestCatalogQuoteRejectsMixedStores(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a, content := catalogProductFixture(t, f)
	shop, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "store", "", "second-shop", store.CatalogObjectV2{"name": "另一个测试商店", "intro": "隔离复核", "logoAssetId": nil, "coverAssetId": nil, "contactQq": "123456789", "serviceHours": "", "notice": ""})
	if e != nil {
		t.Fatal(e)
	}
	brand, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "brand", shop.ID, "second-brand", store.CatalogObjectV2{"name": "第二品牌", "logoAssetId": nil, "sortOrder": 0, "active": true})
	if e != nil {
		t.Fatal(e)
	}
	category, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "category", shop.ID, "second-category", store.CatalogObjectV2{"name": "基础物品", "sortOrder": 0, "active": true})
	if e != nil {
		t.Fatal(e)
	}
	template, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, fmt.Sprint(content["deliveryTemplateRef"]), "delivery_template", true)
	if e != nil {
		t.Fatal(e)
	}
	template, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, shop.ID, "", "second-template", 0, template.Body, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	next := catalogObjectClone(content)
	next["brandId"] = brand.ID
	next["categoryId"] = category.ID
	next["deliveryTemplateRef"] = template.ID
	b := quoteReviewProductV2(t, f, next, shop.ID, "second-product")
	quoteReviewRejectedV2(t, f, "mixed-quote", []store.CatalogRecordV2{a, b}, []int{1, 1}, "MIXED_STORES")
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 0 || catalogTestCount(t, f.s, "commerce_operations_v2") != 0 {
		t.Fatal("mixed stores wrote payment intent")
	}
	t.Log("Quote now rejects products from different stores before payment confirmation")
}
func TestCatalogQuoteRejectsCrossDomainPreservesPublishedIdentity(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a, content := catalogProductFixture(t, f)
	_, e := f.s.DB.Exec("INSERT INTO core_catalog_heads VALUES('catalog:mekitem',1,1,FALSE,?)", strings.Repeat("c", 64))
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.DB.Exec("INSERT INTO item_versions VALUES('catalog:mekitem',1,'mek',?,?,?,UTC_TIMESTAMP(6))", strings.Repeat("d", 64), `{"displayName":"独立域物品","maxQuantity":64,"codec":"bukkit-bytes-v1","inventoryDomain":"mek","compatibleServerIds":["mek"]}`, strings.Repeat("c", 64))
	if e != nil {
		t.Fatal(e)
	}
	tpl, e := f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, a.StoreID, "", "mek-template", 0, store.CatalogObjectV2{"name": "独立域模板", "summary": "隔离复核", "inventoryDomain": "mek", "allowedServerIds": []any{"mek"}, "attachments": []any{map[string]any{"itemRef": "catalog:mekitem", "revision": 1, "quantity": 1, "payloadSha256": strings.Repeat("d", 64)}}, "active": true}, map[string]store.CatalogNodePolicyV2{"mek": {InventoryDomain: "mek", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	next := catalogObjectClone(content)
	next["deliveryTemplateRef"] = tpl.ID
	b := quoteReviewProductV2(t, f, next, a.StoreID, "mek-product")
	quoteReviewRejectedV2(t, f, "mixed-domain", []store.CatalogRecordV2{a, b}, []int{1, 1}, "DELIVERY_SCOPE_CONFLICT")
	q := quoteReviewAcceptedV2(t, f, "frozen-before-draft", []store.CatalogRecordV2{a}, []int{2})
	draft := catalogObjectClone(content)
	draft["price"] = "99.99"
	draft["title"] = "未发布新标题"
	_, e = f.s.CatalogEditV2(ctx, f.admin.ID, a.ID, "product", "save-only-draft", a.Version, draft, false)
	if e != nil {
		t.Fatal(e)
	}
	m, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "frozen-pay", fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	d, e := f.s.CommerceRecordV2(ctx, m.ResourceID)
	if e != nil {
		t.Fatal(e)
	}
	plan, _ := store.CommerceContentV2(d.Body, "mailboxPlan")
	var mail map[string]any
	if e = json.Unmarshal([]byte(fmt.Sprint(plan["snapshotJson"])), &mail); e != nil {
		t.Fatal(e)
	}
	if d.Amount != "22.20" || d.Body["items"].([]any)[0].(map[string]any)["title"] != "官方石材" || mail["attachments"].([]any)[0].(map[string]any)["quantity"] != float64(2) || store.Digest([]byte(fmt.Sprint(plan["snapshotJson"]))) != plan["snapshotSha256"] {
		t.Fatal("frozen quantity/price/hash changed")
	}
	current, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, a.ID, "product", true)
	if e != nil || *current.Stock != 8 {
		t.Fatal("stock and quantity diverged", e)
	}
	_, e = f.s.DB.Exec("UPDATE core_catalog_heads SET archived=TRUE WHERE item_ref='catalog:stone'")
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.CatalogQuoteV2(ctx, f.buyer.ID, "archived-quote", store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": a.ID, "quantity": 1, "expectedProductVersion": a.PublishedVersion}}, "delivery": map[string]any{"method": "MAILBOX"}})
	assertCatalogCode(t, e, "ITEM_ARCHIVED")
	t.Log("Verified cross-domain rejection before finance; published title/price/template survive draft save; stock -2, mail quantity2, raw SHA exact; archived item blocks new quote")
}
func TestCatalogQuoteStockAndRepublishMaintainContract(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	quoted := quoteReviewAcceptedV2(t, f, "before-stock", []store.CatalogRecordV2{p}, []int{2})
	adjusted, e := f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "stock-minus-one", p.Version, "stock-adjustments", "隔离数量复核", -1)
	if e != nil {
		t.Fatal(e)
	}
	m, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "after-stock-pay", fmt.Sprint(quoted["quoteId"]), 1, "OFFICIAL_STORE", true, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	before, e := f.s.CommerceSnapshotV2(ctx, f.buyer.ID, m.ResourceID, false)
	if e != nil {
		t.Fatal(e)
	}
	beforeBytes, _ := json.Marshal(before)
	current, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.ID, "product", true)
	if e != nil || *current.Stock != 7 || current.PublishedVersion != p.PublishedVersion || current.Version != adjusted.Version+1 {
		t.Fatal("stock reservation changed published version", e)
	}
	oldQuote := quoteReviewAcceptedV2(t, f, "before-republish", []store.CatalogRecordV2{p}, []int{1})
	next := catalogObjectClone(content)
	next["price"] = "55.00"
	next["title"] = "已发布新标题"
	next["stock"] = 99
	draft, e := f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "price-draft", current.Version, next, false)
	if e != nil {
		t.Fatal(e)
	}
	published, e := f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "price-publish", draft.Version, "publish", "", 0)
	if e != nil || *published.Stock != 7 {
		t.Fatal("republish silently replenished stock", e)
	}
	_, e = f.s.PrepareOrderV2(ctx, f.buyer.ID, "stale-price-pay", fmt.Sprint(oldQuote["quoteId"]), 1, "OFFICIAL_STORE", true, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	assertCatalogCode(t, e, "STATE_VERSION_CONFLICT")
	after, e := f.s.CommerceSnapshotV2(ctx, f.buyer.ID, m.ResourceID, false)
	if e != nil {
		t.Fatal(e)
	}
	afterBytes, _ := json.Marshal(after)
	if string(beforeBytes) != string(afterBytes) {
		t.Fatal("republish changed an existing order contract")
	}
	line := map[string]any{"productId": p.ID, "quantity": 1, "expectedProductVersion": published.PublishedVersion}
	_, e = f.s.CatalogQuoteV2(ctx, f.buyer.ID, "duplicate-products", store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{line, line}, "delivery": map[string]any{"method": "MAILBOX"}})
	assertCatalogCode(t, e, "INVALID_REQUEST")
	t.Log("Verified stock-only updates preserve publication/version while reserving exact amount; republish preserves availableStock; stale-price quote rejected; old contract unchanged; duplicate product rows rejected")
}
func TestCatalogQuoteRejectsDisjointClaimServers(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a, content := catalogProductFixture(t, f)
	existing, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, fmt.Sprint(content["deliveryTemplateRef"]), "delivery_template", true)
	if e != nil {
		t.Fatal(e)
	}
	allowed := catalogObjectClone(existing.Body)
	allowed["allowedServerIds"] = []any{"odyssey"}
	template, e := f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, a.StoreID, "", "odyssey-only-template", 0, allowed, map[string]store.CatalogNodePolicyV2{"odyssey": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	next := catalogObjectClone(content)
	next["deliveryTemplateRef"] = template.ID
	b := quoteReviewProductV2(t, f, next, a.StoreID, "odyssey-only-product")
	quoteReviewRejectedV2(t, f, "disjoint-servers", []store.CatalogRecordV2{a, b}, []int{1, 1}, "DELIVERY_SCOPE_CONFLICT")
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 0 || catalogTestCount(t, f.s, "commerce_operations_v2") != 0 || catalogTestCount(t, f.s, "catalog_quotes_v2") != 0 {
		t.Fatal("disjoint server combination created an intent or usable quote")
	}
	t.Log("Same store and survival domain, same item compatible with amiya/odyssey, but product claim sets {amiya} and {odyssey} are rejected at quote")
}

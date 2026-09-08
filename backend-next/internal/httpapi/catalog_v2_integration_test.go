//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type catalogFixture struct {
	s                      *store.Store
	server                 *Server
	admin, buyer, other    store.User
	adminToken, buyerToken string
}

func newCatalogFixture(t *testing.T) catalogFixture {
	t.Helper()
	s := testdb.New(t)
	ctx := context.Background()
	create := func(id, name, uuid string) store.User {
		u := store.User{ID: id, PlayerRef: "player_" + id, GameID: name, QQ: "123456789", ServerUUID: uuid, PasswordHash: "catalog-test-only-hash", Status: "active"}
		_, e := s.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,?,?,?,?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, u.ID, u.PlayerRef, u.ServerUUID, u.GameID, u.QQ, u.PasswordHash, strings.Repeat("0", 64))
		if e != nil {
			t.Fatal(e)
		}
		return u
	}
	f := catalogFixture{s: s, admin: create("catalog_admin", "CatalogAdmin", "10000000-0000-0000-0000-000000000001"), buyer: create("catalog_buyer", "CatalogBuyer", "10000000-0000-0000-0000-000000000002"), other: create("catalog_other", "CatalogOther", "10000000-0000-0000-0000-000000000003"), adminToken: strings.Repeat("A", 43), buyerToken: strings.Repeat("B", 43)}
	if e := s.Grant(ctx, f.admin.ID, "platform.admin"); e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct {
		user  store.User
		token string
	}{{f.admin, f.adminToken}, {f.buyer, f.buyerToken}} {
		if e := s.CreateSession(ctx, v.user, store.Digest([]byte(v.token)), "app", "", time.Now().Add(time.Hour), ""); e != nil {
			t.Fatal(e)
		}
	}
	f.server = New(s, config.Config{Development: true, PublicOrigin: "http://127.0.0.1"})
	t.Cleanup(f.server.Close)
	for k, v := range map[string]string{"DEUTERIUM_S3_ENDPOINT": "https://assets.example.invalid", "DEUTERIUM_S3_REGION": "test", "DEUTERIUM_S3_BUCKET": "catalog-test", "DEUTERIUM_S3_PREFIX": "catalog-test/", "DEUTERIUM_S3_ACCESS_KEY": "test-key", "DEUTERIUM_S3_SECRET_KEY": "test-secret"} {
		t.Setenv(k, v)
	}
	return f
}
func catalogTestAsset(t *testing.T, s *store.Store, user, purpose string) string {
	t.Helper()
	id := store.ID("asset_")
	_, e := s.CreateAssetUploadV2(context.Background(), store.AssetUploadV2{UploadID: store.ID("upload_"), AssetID: id, UserID: user, ClientRequestID: id, Fingerprint: store.Digest([]byte(id)), Purpose: purpose, BusinessType: "CATALOG_TEST", ObjectKey: "catalog-test/" + id + ".png", ContentType: "image/png", MD5: "1B2M2Y8AsgTpgAmY7PhCfg==", SizeBytes: 10})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec("UPDATE asset_uploads_v2 SET status='READY',width=10,height=10,sha256=? WHERE asset_id=?", strings.Repeat("a", 64), id); e != nil {
		t.Fatal(e)
	}
	return id
}
func catalogListingInput(asset string) store.CatalogObjectV2 {
	return store.CatalogObjectV2{"title": "试验石材", "subtitle": "仅隔离数据库测试", "description": "验证真实事务；本条不会进入实服。", "categoryCode": "MATERIALS", "price": "12.30", "stock": 1, "photoAssetIds": []any{asset}, "contactQq": "123456789", "deliveryMethods": []any{"PICKUP"}, "pickupLocation": "主城仓库", "workHours": 24}
}
func catalogObjectClone(v store.CatalogObjectV2) store.CatalogObjectV2 {
	b, _ := json.Marshal(v)
	var o store.CatalogObjectV2
	_ = json.Unmarshal(b, &o)
	return o
}
func assertCatalogCode(t *testing.T, e error, code string) {
	t.Helper()
	var p *store.CatalogErrorV2
	if !errors.As(e, &p) || p.Code != code {
		t.Fatalf("expected %s, got %v", code, e)
	}
}
func catalogTestCount(t *testing.T, s *store.Store, table string) int {
	t.Helper()
	var n int
	if e := s.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}

func TestCatalogMarketConcurrencyOwnershipAndAssetBinding(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	asset := catalogTestAsset(t, f.s, f.admin.ID, "MARKET_PHOTO")
	input := catalogListingInput(asset)
	const n = 12
	var wg sync.WaitGroup
	ids := make(chan string, n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "listing", "", "same-create", input)
			if e != nil {
				errs <- e
			} else {
				ids <- d.ID
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	close(ids)
	id := ""
	for v := range ids {
		if id != "" && v != id {
			t.Fatal("same request created multiple listings")
		}
		id = v
	}
	if catalogTestCount(t, f.s, "catalog_records_v2") != 1 {
		t.Fatal("duplicate rows")
	}
	bad := catalogObjectClone(input)
	bad["price"] = "99"
	_, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "listing", "", "same-create", bad)
	assertCatalogCode(t, e, "IDEMPOTENCY_CONFLICT")
	_, e = f.s.CatalogEditV2(ctx, f.buyer.ID, id, "listing", "steal-edit", 1, input, false)
	assertCatalogCode(t, e, "NOT_FOUND")
	if e = f.s.RemoveAssetV2(ctx, f.admin.ID, asset); !errors.Is(e, store.ErrAssetInUse) {
		t.Fatal("bound image removed", e)
	}
	var wins atomic.Int32
	errs = make(chan error, 2)
	for _, key := range []string{"unlist-a", "unlist-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, e := f.s.CatalogActionV2(ctx, f.admin.ID, id, "listing", key, 1, "unlist", "", 0)
			if e == nil {
				wins.Add(1)
			} else {
				errs <- e
			}
		}(key)
	}
	wg.Wait()
	close(errs)
	if wins.Load() != 1 {
		t.Fatalf("concurrent state transitions: %d", wins.Load())
	}
	for e := range errs {
		assertCatalogCode(t, e, "STATE_VERSION_CONFLICT")
	}
	public, e := f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "listing", Limit: 20})
	if e != nil || len(public) != 0 {
		t.Fatal("unlisted listing remained public", e)
	}
	own, e := f.s.CatalogListV2(ctx, f.admin.ID, store.CatalogFilterV2{Kind: "listing", Management: true, Limit: 20})
	if e != nil || len(own) != 1 || own[0].State != "UNLISTED" {
		t.Fatal("owner listing lost", e)
	}
	if _, e = f.s.CatalogEditV2(ctx, f.admin.ID, id, "listing", "republish", 2, input, true); e != nil {
		t.Fatal(e)
	}
	public, e = f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "listing", Limit: 20})
	if e != nil || len(public) != 1 {
		t.Fatal("republish not visible", e)
	}
	foreign := catalogListingInput(catalogTestAsset(t, f.s, f.buyer.ID, "MARKET_PHOTO"))
	if _, e = f.s.CatalogCreateV2(ctx, f.admin.ID, "listing", "", "foreign-photo", foreign); !errors.Is(e, store.ErrAssetUnavailable) {
		t.Fatal("accepted someone else's unbound image", e)
	}
	if catalogTestCount(t, f.s, "catalog_records_v2") != 1 {
		t.Fatal("failed image validation left partial listing")
	}
}

func catalogProductFixture(t *testing.T, f catalogFixture) (store.CatalogRecordV2, store.CatalogObjectV2) {
	t.Helper()
	ctx := context.Background()
	shop, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "store", "", "shop", store.CatalogObjectV2{"name": "测试商店", "intro": "隔离测试", "logoAssetId": nil, "coverAssetId": nil, "contactQq": "123456789", "serviceHours": "", "notice": ""})
	if e != nil {
		t.Fatal(e)
	}
	brand, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "brand", shop.ID, "brand", store.CatalogObjectV2{"name": "测试品牌", "logoAssetId": nil, "sortOrder": 0, "active": true})
	if e != nil {
		t.Fatal(e)
	}
	category, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "category", shop.ID, "category", store.CatalogObjectV2{"name": "基础物品", "sortOrder": 0, "active": true})
	if e != nil {
		t.Fatal(e)
	}
	asset := catalogTestAsset(t, f.s, f.admin.ID, "STORE_MEDIA")
	_, e = f.s.DB.Exec(`CREATE TABLE IF NOT EXISTS core_catalog_heads(item_ref VARCHAR(96) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,catalog_version BIGINT NOT NULL,latest_revision BIGINT NOT NULL,archived BOOLEAN NOT NULL,fingerprint CHAR(64) CHARACTER SET ascii NOT NULL)`)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.DB.Exec("INSERT INTO core_catalog_heads VALUES('catalog:stone',1,1,FALSE,?)", strings.Repeat("b", 64))
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.DB.Exec("INSERT INTO item_versions VALUES('catalog:stone',1,'amiya',?,?,?,UTC_TIMESTAMP(6))", strings.Repeat("a", 64), `{"displayName":"石材","description":"已发布模板","maxQuantity":64,"codec":"bukkit-bytes-v1","inventoryDomain":"survival","compatibleServerIds":["amiya","odyssey"]}`, strings.Repeat("b", 64))
	if e != nil {
		t.Fatal(e)
	}
	template, e := f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, shop.ID, "", "registered-template", 0, store.CatalogObjectV2{"name": "石材交付", "summary": "原样物品模板", "inventoryDomain": "survival", "allowedServerIds": []any{"amiya"}, "attachments": []any{map[string]any{"itemRef": "catalog:stone", "revision": 1, "quantity": 1, "payloadSha256": strings.Repeat("a", 64)}}, "active": true}, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	content := store.CatalogObjectV2{"title": "官方石材", "subtitle": "隔离事务测试", "description": "真实服务端价格和版本。", "brandId": brand.ID, "categoryId": category.ID, "price": "11.10", "coverAssetId": asset, "galleryAssetIds": []any{asset}, "galleryAltTexts": []any{"石材"}, "contentBlocks": []any{}, "includedItems": []any{"石材一组"}, "deliveryTemplateRef": "catalog:stone:v1", "deliverySummary": "游戏邮箱", "estimatedDelivery": "付款后投递", "inventoryPolicy": "FINITE", "stock": 10, "limitPerOrder": 5, "posterTone": "LIGHT", "accentColor": "#123456", "badges": []any{}, "sortOrder": 0}
	invalidProduct, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "product", shop.ID, "invalid-product", content)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.CatalogActionV2(ctx, f.admin.ID, invalidProduct.ID, "product", "missing-template", 1, "publish", "", 0); e == nil {
		t.Fatal("published an unknown delivery template")
	}
	content["deliveryTemplateRef"] = template.ID
	product, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "product", shop.ID, "product", content)
	if e != nil {
		t.Fatal(e)
	}
	product, e = f.s.CatalogActionV2(ctx, f.admin.ID, product.ID, "product", "publish", 1, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	return product, content
}
func TestCatalogPublishedSnapshotCartAndAuthoritativeQuote(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, content := catalogProductFixture(t, f)
	cart, e := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if e != nil || cart.Version != 1 || len(cart.Items) != 0 {
		t.Fatal(cart, e)
	}
	cart, e = f.s.CatalogCartChangeV2(ctx, f.buyer.ID, product.ID, "add-cart", 1, 2, false)
	if e != nil || cart.Version != 2 || len(cart.Items) != 1 {
		t.Fatal(cart, e)
	}
	replay, e := f.s.CatalogCartChangeV2(ctx, f.buyer.ID, product.ID, "add-cart", 1, 2, false)
	if e != nil || replay.Version != 2 {
		t.Fatal("cart retry failed", e)
	}
	_, e = f.s.CatalogCartChangeV2(ctx, f.buyer.ID, product.ID, "stale-cart", 1, 3, false)
	assertCatalogCode(t, e, "STATE_VERSION_CONFLICT")
	updated := catalogObjectClone(content)
	updated["price"] = "99.99"
	updated["title"] = "草稿标题"
	draft, e := f.s.CatalogEditV2(ctx, f.admin.ID, product.ID, "product", "draft-change", product.Version, updated, false)
	if e != nil || draft.Version != 3 {
		t.Fatal(e)
	}
	public, e := f.s.CatalogGetRecordV2(ctx, f.buyer.ID, product.ID, "product", false)
	if e != nil || public.Published["price"] != "11.10" || public.PublishedVersion != 2 {
		t.Fatal("draft leaked into publication", e)
	}
	input := store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": product.ID, "quantity": 2, "expectedProductVersion": 2}}, "delivery": map[string]any{"method": "MAILBOX", "location": "", "projectName": ""}}
	quote, e := f.s.CatalogQuoteV2(ctx, f.buyer.ID, "quote-test", input)
	if e != nil || quote["totalAmount"] != "22.20" {
		t.Fatal(quote, e)
	}
	again, e := f.s.CatalogQuoteV2(ctx, f.buyer.ID, "quote-test", input)
	if e != nil || again["quoteId"] != quote["quoteId"] || catalogTestCount(t, f.s, "catalog_quotes_v2") != 1 {
		t.Fatal("quote retry created another quote", e)
	}
	public, _ = f.s.CatalogGetRecordV2(ctx, f.buyer.ID, product.ID, "product", false)
	if *public.Stock != 10 {
		t.Fatal("quote reserved or spent stock")
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/store/orders", strings.NewReader(`{"clientRequestId":"offline-store-payment","quoteId":"`+quote["quoteId"].(string)+`","expectedQuoteVersion":1}`))
	r.Header.Set("Authorization", "Bearer "+f.buyerToken)
	r.Header.Set("Content-Type", "application/json")
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	f.server.registerCommerceV2(mux)
	mux.ServeHTTP(w, r)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "CAPABILITY_UNAVAILABLE") {
		t.Fatalf("payment reported success: %d %s", w.Code, w.Body.String())
	}
	_, e = f.s.CatalogActionV2(ctx, f.admin.ID, product.ID, "product", "empty-stock", draft.Version, "stock-adjustments", "测试扣减库存", -10)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.CatalogQuoteV2(ctx, f.buyer.ID, "out-of-stock", input)
	assertCatalogCode(t, e, "INSUFFICIENT_STOCK")
}

func TestCatalogStoreMembershipDoesNotGrantPlatformAuthority(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, content := catalogProductFixture(t, f)
	if _, e := f.s.CatalogSetMemberV2(ctx, f.buyer.ID, product.StoreID, "self-promote", f.buyer.PlayerRef, 1, []string{"PRODUCT_EDIT"}, false); e == nil {
		t.Fatal("player self-promoted")
	}
	member, e := f.s.CatalogSetMemberV2(ctx, f.admin.ID, product.StoreID, "grant-editor", f.buyer.PlayerRef, 1, []string{"PRODUCT_EDIT"}, false)
	if e != nil || member.Version != 1 {
		t.Fatal(e)
	}
	updated := catalogObjectClone(content)
	updated["title"] = "成员编辑"
	if _, e = f.s.CatalogEditV2(ctx, f.buyer.ID, product.ID, "product", "member-edit", product.Version, updated, false); e != nil {
		t.Fatal("authorized editor could not retain bound image", e)
	}
	if _, e = f.s.CatalogActionV2(ctx, f.buyer.ID, product.ID, "product", "member-publish", 3, "publish", "", 0); e == nil {
		t.Fatal("edit permission granted publish")
	}
	if _, e = f.s.CatalogSetMemberV2(ctx, f.admin.ID, product.StoreID, "revoke-editor", f.buyer.PlayerRef, member.Version, []string{}, true); e != nil {
		t.Fatal(e)
	}
	if e = f.s.CatalogCanManageV2(ctx, f.buyer.ID, product.StoreID, "READ"); e == nil {
		t.Fatal("revoked membership retained access")
	}
}

func TestCatalogHomepageFirstWriteReplayAndFilterPrivacy(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, _ := catalogProductFixture(t, f)
	content := store.CatalogObjectV2{"intro": "真实店铺首页", "sections": []any{map[string]any{"sectionId": "section_items", "title": "商品", "layout": "PRODUCT_GRID", "productIds": []any{product.ID}, "bannerAssetId": nil, "sortOrder": 0}}, "brandIds": []any{}, "categoryIds": []any{}}
	first, e := f.s.CatalogWriteHomepageV2(ctx, f.admin.ID, product.StoreID, "home-first", 1, content)
	if e != nil {
		t.Fatal(e)
	}
	again, e := f.s.CatalogWriteHomepageV2(ctx, f.admin.ID, product.StoreID, "home-first", 1, content)
	if e != nil || first.ID != again.ID || again.Version != 1 {
		t.Fatal("first homepage retry changed identity/version", e)
	}
	if _, e = f.s.CatalogWriteHomepageV2(ctx, f.buyer.ID, product.StoreID, "steal-home", 1, content); e == nil {
		t.Fatal("nonmember wrote official homepage")
	}
	for _, query := range []string{"官方石材", "隔离事务测试", "测试品牌"} {
		rows, e := f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "product", Query: query, Limit: 20})
		if e != nil || len(rows) != 1 {
			t.Fatal("title/subtitle/brand search failed", query, e)
		}
	}
	product.Body["subtitle"] = "不可公开的草稿文本"
	if _, e = f.s.CatalogEditV2(ctx, f.admin.ID, product.ID, "product", "new-subtitle", product.Version, product.Body, false); e != nil {
		t.Fatal(e)
	}
	rows, e := f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "product", Query: "不可公开的草稿", Limit: 20})
	if e != nil || len(rows) != 0 {
		t.Fatal("public search matched draft content", e)
	}
}

func TestCatalogHTTPPublicModelsAndCSRFBoundary(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, _ := catalogProductFixture(t, f)
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	request := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, "req_catalog_contract"))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	w := request("GET", "/api/v1/store/products", "", f.buyerToken)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var envelope struct {
		Data []map[string]any `json:"data"`
		Page map[string]any   `json:"page"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &envelope); e != nil || len(envelope.Data) != 1 || envelope.Data[0]["productId"] != product.ID || envelope.Data[0]["content"] == nil || envelope.Page["hasMore"] != false {
		t.Fatal("unexpected V2 catalog envelope", w.Body.String(), e)
	}
	samples := map[string]json.RawMessage{"V2StoreProductListResponse": append([]byte(nil), w.Body.Bytes()...)}
	for _, sample := range []struct{ schema, path string }{{"V2StoreProductResponse", "/api/v1/store/products/" + product.ID}, {"V2MerchantProductResponse", "/api/v1/merchant/products/" + product.ID}, {"V2ShoppingBagResponse", "/api/v1/store/cart"}, {"V2StoreViewResponse", "/api/v1/store/stores/" + product.StoreID}} {
		response := request("GET", sample.path, "", f.adminToken)
		if response.Code != 200 {
			t.Fatal(sample.schema, response.Code, response.Body.String())
		}
		samples[sample.schema] = append([]byte(nil), response.Body.Bytes()...)
	}
	quoteBody, _ := json.Marshal(map[string]any{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": product.ID, "quantity": 2, "expectedProductVersion": 2}}, "delivery": map[string]any{"method": "MAILBOX", "location": "", "projectName": ""}})
	quoteResponse := request("POST", "/api/v1/checkout/quotes", string(quoteBody), f.buyerToken)
	if quoteResponse.Code != 201 {
		t.Fatal(quoteResponse.Code, quoteResponse.Body.String())
	}
	samples["V2CheckoutQuoteResponse"] = append([]byte(nil), quoteResponse.Body.Bytes()...)
	asset := catalogTestAsset(t, f.s, f.admin.ID, "MARKET_PHOTO")
	body, _ := json.Marshal(map[string]any{"clientRequestId": "http-listing", "content": catalogListingInput(asset)})
	w = request("POST", "/api/v1/market/listings", string(body), f.adminToken)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	samples["V2MarketListingResponse"] = append([]byte(nil), w.Body.Bytes()...)
	var listingEnvelope struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listingEnvelope)
	for _, field := range []string{"listingId", "description", "seller", "photos", "stock", "active", "version"} {
		if _, ok := listingEnvelope.Data[field]; !ok {
			t.Fatalf("missing %s in market response", field)
		}
	}
	webToken := strings.Repeat("W", 43)
	if e := f.s.CreateSession(ctx, f.admin, store.Digest([]byte(webToken)), "web", "valid-csrf", time.Now().Add(time.Hour), ""); e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("POST", "/api/v1/market/listings", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://127.0.0.1")
	r.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: webToken})
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("cookie write missing CSRF accepted: %d", w.Code)
	}
	if dir := os.Getenv("DEUTERIUM_CATALOG_SAMPLES"); dir != "" {
		if !filepath.IsAbs(dir) {
			t.Fatal("catalog response output must be absolute")
		}
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		data, _ := json.MarshalIndent(samples, "", "  ")
		if e := os.WriteFile(filepath.Join(dir, "responses.json"), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
}

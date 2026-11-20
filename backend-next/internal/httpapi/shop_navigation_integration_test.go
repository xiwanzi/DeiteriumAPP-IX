//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestShopNavigationHierarchyBadgesAndMedia(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, content := catalogProductFixture(t, f)
	create := func(name, parent, key string) store.CatalogRecordV2 {
		t.Helper()
		body := store.CatalogObjectV2{"name": name, "sortOrder": 0, "active": true}
		if parent != "" {
			body["parentCategoryId"] = parent
		}
		d, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "category", product.StoreID, key, body)
		if e != nil {
			t.Fatal(e)
		}
		return d
	}
	root := create("家具家居", "", "root-furniture")
	child, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, content["categoryId"].(string), "category", true)
	if e != nil {
		t.Fatal(e)
	}
	body := catalogObjectClone(child.Body)
	body["parentCategoryId"] = root.ID
	child, e = f.s.CatalogEditV2(ctx, f.admin.ID, child.ID, "category", "move-child", child.Version, body, false)
	if e != nil {
		t.Fatal(e)
	}
	// Existing product references and published versions are retained by a category move.
	check := func(filter store.CatalogFilterV2, want int) {
		t.Helper()
		filter.Kind = "product"
		filter.Limit = 40
		rows, e := f.s.CatalogListV2(ctx, f.buyer.ID, filter)
		if e != nil || len(rows) != want {
			t.Fatalf("filter %#v: count=%d err=%v", filter, len(rows), e)
		}
	}
	check(store.CatalogFilterV2{CategoryID: root.ID}, 1)
	check(store.CatalogFilterV2{CategoryGroup: "家具家居"}, 1)
	check(store.CatalogFilterV2{CategoryGroup: "家具家居", Subcategory: child.Body["name"].(string)}, 1)
	check(store.CatalogFilterV2{CategoryGroup: "家具家居", Subcategory: "不属于此类"}, 0)
	check(store.CatalogFilterV2{Query: "家具家居"}, 1)
	check(store.CatalogFilterV2{Query: "基础物品"}, 1)
	check(store.CatalogFilterV2{Badge: "新品推荐"}, 0)
	edit := catalogObjectClone(content)
	edit["badges"] = []any{"新品推荐"}
	product, e = f.s.CatalogEditV2(ctx, f.admin.ID, product.ID, "product", "badge-draft", product.Version, edit, false)
	if e != nil {
		t.Fatal(e)
	}
	check(store.CatalogFilterV2{Badge: "新品推荐"}, 0)
	product, e = f.s.CatalogActionV2(ctx, f.admin.ID, product.ID, "product", "badge-publish", product.Version, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	check(store.CatalogFilterV2{Badge: "新品推荐"}, 1)
	check(store.CatalogFilterV2{Query: "新品推荐"}, 1)
	check(store.CatalogFilterV2{Badge: "新品"}, 0)
	for name, parent := range map[string]string{"self": root.ID, "cycle": child.ID} {
		b := catalogObjectClone(root.Body)
		b["parentCategoryId"] = parent
		if _, e = f.s.CatalogEditV2(ctx, f.admin.ID, root.ID, "category", "bad-parent-"+name, root.Version, b, false); e == nil {
			t.Fatal("accepted cyclic hierarchy", name)
		}
	}
	if _, e = f.s.CatalogCreateV2(ctx, f.admin.ID, "category", product.StoreID, "third-level", store.CatalogObjectV2{"name": "三级", "sortOrder": 0, "active": true, "parentCategoryId": child.ID}); e == nil {
		t.Fatal("accepted third level")
	}
	another, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "store", "", "second-store", store.CatalogObjectV2{"name": "另一店铺", "intro": "", "logoAssetId": nil, "coverAssetId": nil, "contactQq": "123456789", "serviceHours": "", "notice": ""})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.CatalogCreateV2(ctx, f.admin.ID, "category", another.ID, "cross-store", store.CatalogObjectV2{"name": "错误归属", "sortOrder": 0, "active": true, "parentCategoryId": root.ID}); e == nil {
		t.Fatal("accepted cross-store parent")
	}
	categories, e := f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "category", Limit: 40})
	if e != nil {
		t.Fatal(e)
	}
	views, e := f.s.CatalogPageViewsV2(ctx, f.buyer.ID, categories, false)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, raw := range views {
		v := raw.(map[string]any)
		if v["categoryId"] == child.ID {
			image, ok := v["image"].(map[string]any)
			if !ok || image["assetId"] != content["coverAssetId"] {
				t.Fatal("category cover missing", v)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("child missing")
	}
	banner := catalogTestAsset(t, f.s, f.admin.ID, "STORE_MEDIA")
	home, e := f.s.CatalogWriteHomepageV2(ctx, f.admin.ID, product.StoreID, "banner-home", 1, store.CatalogObjectV2{"intro": "", "brandIds": []any{}, "categoryIds": []any{}, "sections": []any{map[string]any{"sectionId": "activity", "title": "活动头图", "layout": "BANNER", "bannerAssetId": banner, "productIds": []any{}, "sortOrder": 0}}})
	if e != nil {
		t.Fatal(e)
	}
	homeView, e := f.s.CatalogViewV2(ctx, f.buyer.ID, home, false)
	if e != nil {
		t.Fatal(e)
	}
	banners := homeView["banners"].([]any)
	if len(banners) != 1 || banners[0].(map[string]any)["image"].(map[string]any)["assetId"] != banner {
		t.Fatal("missing bound banner image")
	}
	if _, e = f.s.CatalogWriteHomepageV2(ctx, f.buyer.ID, product.StoreID, "no-banner-permission", home.Version, home.Body); e == nil {
		t.Fatal("buyer changed banner")
	}
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	read := func(query string) []map[string]any {
		t.Helper()
		r := httptest.NewRequest("GET", "/api/v1/store/products?limit=40&"+query, nil)
		r.Header.Set("Authorization", "Bearer "+f.buyerToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
		}
		var data struct {
			Data []map[string]any `json:"data"`
		}
		if e = json.Unmarshal(w.Body.Bytes(), &data); e != nil {
			t.Fatal(e)
		}
		return data.Data
	}
	if len(read("badge="+url.QueryEscape("新品推荐")+"&categoryGroup="+url.QueryEscape("家具家居"))) != 1 {
		t.Fatal("HTTP filters missing")
	}
	// A category cover loses access when its exact product binding is removed.
	if _, e = f.s.DB.Exec("DELETE FROM asset_bindings_v2 WHERE business_type='PRODUCT_PUBLISHED' AND business_ref=?", product.ID); e != nil {
		t.Fatal(e)
	}
	views, e = f.s.CatalogPageViewsV2(ctx, f.buyer.ID, []store.CatalogRecordV2{child}, false)
	if e != nil {
		t.Fatal(e)
	}
	image := views[0].(map[string]any)["image"].(map[string]any)
	if image["url"] != "" {
		t.Fatal("unbound category cover exposed")
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestCatalogScalePagingAndBatchedMedia(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	base, content := catalogProductFixture(t, f)
	baseAsset := content["coverAssetId"].(string)
	tx, err := f.s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	productInsert, err := tx.Prepare(`INSERT INTO catalog_records_v2(resource_id,kind,store_id,owner_id,version,body,published_body,published_version,published_at,state,available_stock,title,category_ref,brand_ref,created_at,updated_at) VALUES(?,'product',?,?,2,?,?,2,UTC_TIMESTAMP(6),'ACTIVE',NULL,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`)
	if err != nil {
		t.Fatal(err)
	}
	defer productInsert.Close()
	assetInsert, err := tx.Prepare(`INSERT INTO asset_uploads_v2(upload_id,asset_id,user_id,client_request_id,fingerprint,purpose,business_type,business_ref,object_key,content_type,size_bytes,content_md5,alt_text,status,width,height,sha256,created_at,expires_at) SELECT ?,?,user_id,?,fingerprint,purpose,business_type,business_ref,?,content_type,size_bytes,content_md5,alt_text,status,width,height,sha256,created_at,expires_at FROM asset_uploads_v2 WHERE asset_id=?`)
	if err != nil {
		t.Fatal(err)
	}
	defer assetInsert.Close()
	bindInsert, err := tx.Prepare(`INSERT INTO asset_bindings_v2(asset_id,business_type,business_ref) VALUES(?,'PRODUCT_PUBLISHED',?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer bindInsert.Close()
	for i := 0; i < 3084; i++ {
		id, asset := fmt.Sprintf("scale_product_%04d", i), fmt.Sprintf("scale_asset_%04d", i)
		c := catalogObjectClone(content)
		c["title"], c["inventoryPolicy"], c["stock"] = fmt.Sprintf("家具%04d", i), "UNLIMITED", 0
		c["coverAssetId"], c["galleryAssetIds"] = asset, []any{asset}
		body, _ := json.Marshal(c)
		if _, err = productInsert.Exec(id, base.StoreID, f.admin.ID, body, body, c["title"], c["categoryId"], c["brandId"]); err != nil {
			t.Fatal(err)
		}
		if _, err = assetInsert.Exec("upload_"+asset, asset, asset, "catalog-test/"+asset+".png", baseAsset); err != nil {
			t.Fatal(err)
		}
		if _, err = bindInsert.Exec(asset, id); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	read := func(path string) ([]map[string]any, string) {
		t.Helper()
		r := httptest.NewRequest("GET", "/api/v1/store/products"+path, nil)
		r.Header.Set("Authorization", "Bearer "+f.buyerToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("catalog status %d: %s", w.Code, w.Body.String())
		}
		var response struct {
			Data []map[string]any `json:"data"`
			Page struct {
				NextCursor string `json:"nextCursor"`
			} `json:"page"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Data, response.Page.NextCursor
	}
	start := time.Now()
	first, cursor := read("?limit=40")
	firstTime := time.Since(start)
	if len(first) != 40 || cursor == "" {
		t.Fatalf("first page count=%d cursor=%q", len(first), cursor)
	}
	if first[0]["availableStock"] != nil {
		t.Fatal("unlimited stock must stay null")
	}
	ids := map[string]bool{}
	for _, p := range first {
		ids[p["productId"].(string)] = true
	}
	second, _ := read("?limit=40&cursor=" + url.QueryEscape(cursor))
	for _, p := range second {
		if ids[p["productId"].(string)] {
			t.Fatal("duplicate cursor page")
		}
	}
	search, _ := read("?limit=40&q=" + url.QueryEscape("家具0000"))
	if len(search) != 1 || search[0]["productId"] != "scale_product_0000" {
		t.Fatal("search missed product outside loaded pages")
	}
	category, _ := read("?limit=40&q=" + url.QueryEscape("基础物品"))
	if len(category) != 40 {
		t.Fatal("category name search lost full catalog coverage")
	}
	// Compare the existing per-record renderer with the page renderer using the
	// same 100 independently bound cover images and one DB connection.
	f.s.DB.SetMaxOpenConns(1)
	records, err := f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "product", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	questions := func() int64 {
		var name string
		var n int64
		if err := f.s.DB.QueryRow("SHOW SESSION STATUS LIKE 'Questions'").Scan(&name, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := questions()
	start = time.Now()
	old := []map[string]any{}
	for _, d := range records {
		v, e := f.s.CatalogViewV2(ctx, f.buyer.ID, d, false)
		if e != nil {
			t.Fatal(e)
		}
		old = append(old, v)
	}
	oldTime, oldQueries := time.Since(start), questions()-before-1
	before = questions()
	start = time.Now()
	batch, err := f.s.CatalogPageViewsV2(ctx, f.buyer.ID, records, false)
	newTime, newQueries := time.Since(start), questions()-before-1
	if err != nil {
		t.Fatal(err)
	}
	if oldQueries != 200 || newQueries != 2 {
		t.Fatalf("media queries old=%d new=%d", oldQueries, newQueries)
	}
	for i, v := range batch {
		p := v.(map[string]any)
		if p["productId"] != old[i]["productId"] || p["effectivePrice"] != old[i]["effectivePrice"] {
			t.Fatal("batch changed product identity or pricing")
		}
		a, b := p["images"].([]any)[0].(map[string]any), old[i]["images"].([]any)[0].(map[string]any)
		if a["assetId"] != b["assetId"] || a["sha256"] != b["sha256"] || a["status"] != "READY" || !strings.HasPrefix(a["url"].(string), "https://") {
			t.Fatal("batch changed authorized media")
		}
	}
	// Cross-product binding cannot authorize a cover for the wrong record.
	if _, err = f.s.DB.Exec("DELETE FROM asset_bindings_v2 WHERE business_ref=? AND business_type='PRODUCT_PUBLISHED'", records[0].ID); err != nil {
		t.Fatal(err)
	}
	batch, err = f.s.CatalogPageViewsV2(ctx, f.buyer.ID, records, false)
	if err != nil {
		t.Fatal(err)
	}
	missing := batch[0].(map[string]any)["images"].([]any)[0].(map[string]any)
	if missing["status"] != "UNAVAILABLE" || missing["url"] != "" {
		t.Fatal("missing binding leaked a signed URL")
	}
	if batch[1].(map[string]any)["images"].([]any)[0].(map[string]any)["status"] != "READY" {
		t.Fatal("missing image hid unrelated products")
	}
	t.Logf("3085 products; first 40 HTTP response=%s; 100-image render old=%s/%d SQL, batch=%s/%d SQL", firstTime, oldTime, oldQueries, newTime, newQueries)
}

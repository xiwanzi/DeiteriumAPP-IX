//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestCouponDeletionRequiresDisableAndCannotBeResurrected(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	c := promotionSaveTestV209(t, f, promotionCouponTestV209())
	_, e := f.s.DeleteCatalogEntry(ctx, f.buyer.ID, c.ID, "coupon", "forbidden", c.Version)
	if e == nil {
		t.Fatal("player deleted coupon")
	}
	_, e = f.s.DeleteCatalogEntry(ctx, f.admin.ID, c.ID, "coupon", "active-delete", c.Version)
	assertCatalogCode(t, e, "COUPON_DISABLE_REQUIRED")
	c.Body["active"] = false
	c, e = f.s.SaveCouponV209(ctx, f.admin.ID, c.ID, "disable", c.Version, c.Body)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.DeleteCatalogEntry(ctx, f.admin.ID, c.ID, "coupon", "stale-delete", c.Version-1)
	assertCatalogCode(t, e, "STATE_VERSION_CONFLICT")
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	var first string
	for i := 0; i < 2; i++ {
		body := fmt.Sprintf(`{"clientRequestId":"delete-coupon","expectedVersion":%d}`, c.Version)
		r := httptest.NewRequest("POST", "/api/v1/admin/coupons/"+c.ID+"/delete", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+f.adminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var response struct{ Data json.RawMessage }
		if e = json.Unmarshal(w.Body.Bytes(), &response); e != nil || w.Code != 200 || !strings.Contains(string(response.Data), `"deleted":true`) {
			t.Fatal(w.Code, w.Body.String(), e)
		}
		if i == 0 {
			first = string(response.Data)
		} else if first != string(response.Data) {
			t.Fatal("delete replay changed result")
		}
	}
	for _, admin := range []bool{true, false} {
		rows, _, _, e := f.s.CouponsV209(ctx, f.admin.ID, admin, "", "", 100)
		if e != nil || len(rows) != 0 {
			t.Fatal("deleted coupon remained visible", rows, e)
		}
	}
	c.Body["active"] = true
	if _, e = f.s.SaveCouponV209(ctx, f.admin.ID, c.ID, "resurrect", c.Version+1, c.Body); e == nil {
		t.Fatal("deleted coupon reenabled")
	}
	draft := couponDraftTestV209(t, f, promotionCouponTestV209())
	if _, e = f.s.DeleteCatalogEntry(ctx, f.admin.ID, draft.ID, "coupon", "delete-draft", draft.Version); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.PublishCouponDraftsV209(ctx, f.admin.ID, "publish-deleted", []store.CouponDraftSelectionV209{{CouponID: draft.ID, ExpectedVersion: draft.Version + 1}}); e == nil {
		t.Fatal("deleted draft published")
	}
}

func TestProductDeletionPreservesPaidOrderAndRefund(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	core := newCommerceTestCore()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	p, content := catalogProductFixture(t, f)
	d := commerceRunTest(t, f, promotionPrepareTestV209(t, f, promotionQuoteTestV209(t, f, p, "paid", 1, "DIRECT"), "pay"))
	staleQuote := promotionQuoteTestV209(t, f, p, "before-delete", 1, "DIRECT")
	p, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.ID, "product", true)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.DeleteCatalogEntry(ctx, f.buyer.ID, p.ID, "product", "forbidden", p.Version)
	if e == nil {
		t.Fatal("non-member deleted product")
	}
	if _, e = f.s.CatalogSetMemberV2(ctx, f.admin.ID, p.StoreID, "grant-editor", f.buyer.PlayerRef, 1, []string{"PRODUCT_EDIT"}, false); e != nil {
		t.Fatal(e)
	}
	_, e = f.s.DeleteCatalogEntry(ctx, f.buyer.ID, p.ID, "product", "editor-delete", p.Version)
	if e == nil {
		t.Fatal("editor bypassed publishing permission")
	}
	// An actual authorized merchant can delete, including an on-sale product.
	if _, e = f.s.CatalogSetMemberV2(ctx, f.admin.ID, p.StoreID, "grant-publisher", f.buyer.PlayerRef, 1, []string{"PRODUCT_PUBLISH"}, false); e != nil {
		t.Fatal(e)
	}
	deleted, e := f.s.DeleteCatalogEntry(ctx, f.buyer.ID, p.ID, "product", "delete-product", p.Version)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := f.s.DeleteCatalogEntry(ctx, f.buyer.ID, p.ID, "product", "delete-product", p.Version)
	if e != nil || string(replay) != string(deleted) {
		t.Fatal("delete replay", e)
	}
	for _, management := range []bool{true, false} {
		if _, e = f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.ID, "product", management); e == nil {
			t.Fatal("deleted detail visible")
		}
		rows, e := f.s.CatalogListV2(ctx, f.admin.ID, store.CatalogFilterV2{Kind: "product", StoreID: p.StoreID, Management: management, Limit: 100})
		if e != nil {
			t.Fatal(e)
		}
		for _, row := range rows {
			if row.ID == p.ID {
				t.Fatal("deleted product remained in list")
			}
		}
	}
	if _, e = f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "edit-deleted", p.Version+1, content, false); e == nil {
		t.Fatal("deleted product editable")
	}
	if _, e = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "republish-deleted", p.Version+1, "publish", "", 0); e == nil {
		t.Fatal("deleted product published")
	}
	if _, e = f.s.PrepareOrderV2(ctx, f.buyer.ID, "stale-payment", fmt.Sprint(staleQuote["quoteId"]), 1, "OFFICIAL_STORE", true, nil); e == nil {
		t.Fatal("deleted product purchased through stale quote")
	}
	view, e := f.s.CommerceViewV2(ctx, f.buyer.ID, d.ID, false)
	if e != nil || view["amount"] != d.Amount {
		t.Fatal("order snapshot lost", view, e)
	}
	m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund-deleted-product", d.Version, commerceRefundInput("", d.Version), true)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.State != "REFUNDED" {
		t.Fatal("deleted product blocked refund", d)
	}
	var state string
	if e = f.s.DB.QueryRow("SELECT state FROM catalog_records_v2 WHERE resource_id=?", p.ID).Scan(&state); e != nil || state != "DELETED" {
		t.Fatal("stock restoration resurrected product", state, e)
	}
}

func TestProductDeleteAndCheckoutSerialize(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	p, _ := catalogProductFixture(t, f)
	q := promotionQuoteTestV209(t, f, p, "race-quote", 1, "DIRECT")
	var wg sync.WaitGroup
	wg.Add(2)
	var deleted, ordered error
	go func() {
		defer wg.Done()
		_, deleted = f.s.DeleteCatalogEntry(ctx, f.admin.ID, p.ID, "product", "race-delete", p.Version)
	}()
	go func() {
		defer wg.Done()
		_, ordered = f.s.PrepareOrderV2(ctx, f.buyer.ID, "race-order", fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil)
	}()
	wg.Wait()
	if deleted == nil && ordered == nil {
		t.Fatal("delete and checkout both accepted same product version")
	}
	if deleted != nil && ordered != nil {
		t.Fatal(deleted, ordered)
	}
}

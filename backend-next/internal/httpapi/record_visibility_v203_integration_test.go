//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRecordVisibilityV203PersistsPerParticipantAndPreservesBusiness(t *testing.T) {
	f := newCatalogFixture(t)
	f.server.CommerceCore = newCommerceTestCore()
	ctx := context.Background()
	order, listing := commerceOrderTest(t, f, "history-order")
	request := func(token, path string, version int64, key string, expected int) {
		t.Helper()
		input, _ := json.Marshal(map[string]any{"clientRequestId": key, "expectedVersion": version})
		r := httptest.NewRequest("POST", "/api/v1/"+path+"/hide", strings.NewReader(string(input)))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		f.server.Handler().ServeHTTP(out, r)
		if out.Code != expected {
			t.Fatalf("hide %s: %d %s", path, out.Code, out.Body.String())
		}
		if out.Code == 200 {
			var response map[string]any
			if err := json.Unmarshal(out.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			writeV203ContractSample(t, "HideRecordResponseV203", response)
		}
	}
	request(f.buyerToken, "orders/"+order.ID, order.Version, "hide-active", 409)
	mutation, err := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, order.ID, "ORDER", "history-refund", order.Version, commerceRefundInput("history-refund", order.Version), true)
	if err != nil {
		t.Fatal(err)
	}
	order = commerceRunTest(t, f, mutation)
	before := order
	request(f.buyerToken, "orders/"+order.ID, order.Version-1, "stale-hide", 409)
	if err = f.s.HideRecordV203(ctx, f.other.ID, "ORDER", order.ID, "other-hide", order.Version); err == nil {
		t.Fatal("nonparticipant hid order")
	}
	request(f.buyerToken, "orders/"+order.ID, order.Version, "hide-final", 200)
	request(f.buyerToken, "orders/"+order.ID, order.Version, "hide-final", 200)
	// Retry races use one persistent display preference, not a business mutation.
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := f.s.HideRecordV203(ctx, f.buyer.ID, "ORDER", order.ID, "hide-final", order.Version); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	own, err := f.s.CommerceListV2(ctx, f.buyer.ID, store.CommerceFilterV2{Kind: "ORDER", Limit: 20})
	if err != nil || len(own) != 0 {
		t.Fatal("hidden order returned in owner history", len(own), err)
	}
	other, err := f.s.CommerceListV2(ctx, f.admin.ID, store.CommerceFilterV2{Kind: "ORDER", Limit: 20})
	if err != nil || len(other) != 1 {
		t.Fatal("seller history changed", len(other), err)
	}
	view, err := f.s.CommerceViewV2(ctx, f.buyer.ID, order.ID, false)
	if err != nil || view["hiddenFromHistory"] != true {
		t.Fatal("hidden marker not readable", err)
	}
	after := commerceRecordTest(t, f, order.ID)
	if before.Version != after.Version || before.FundsState != after.FundsState || before.SnapshotSHA256 != after.SnapshotSHA256 || before.RefundedAmount != after.RefundedAmount {
		t.Fatal("hiding changed financial record")
	}
	// An externally reopened record must become visible again.
	if _, err = f.s.DB.Exec("UPDATE commerce_resources_v2 SET funds_state='UNKNOWN' WHERE resource_id=?", order.ID); err != nil {
		t.Fatal(err)
	}
	own, err = f.s.CommerceListV2(ctx, f.buyer.ID, store.CommerceFilterV2{Kind: "ORDER", Limit: 20})
	if err != nil || len(own) != 1 {
		t.Fatal("unknown result hidden", err)
	}
	view, err = f.s.CommerceViewV2(ctx, f.buyer.ID, order.ID, false)
	if err != nil || view["canHideRecord"] != false || view["hiddenFromHistory"] != false {
		t.Fatal("active view not restored", err)
	}
	// Listings have separate ownership and lifecycle; hiding one does not hide orders.
	current, err := f.s.CatalogGetRecordV2(ctx, f.admin.ID, listing.ID, "listing", true)
	if err != nil {
		t.Fatal(err)
	}
	current, err = f.s.CatalogActionV2(ctx, f.admin.ID, current.ID, "listing", "unlist-history", current.Version, "unlist", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	request(f.buyerToken, "market/listings/"+current.ID, current.Version, "hide-foreign-listing", 404)
	request(f.adminToken, "market/listings/"+current.ID, current.Version, "hide-listing", 200)
	values, err := f.s.CatalogListV2(ctx, f.admin.ID, store.CatalogFilterV2{Kind: "listing", Management: true, Limit: 20})
	if err != nil || len(values) != 0 {
		t.Fatal("listing was not hidden", err)
	}
	var count int
	if err = f.s.DB.QueryRow("SELECT COUNT(*) FROM personal_record_visibility_v203").Scan(&count); err != nil || count != 2 {
		t.Fatal("duplicate visibility markers", count, err)
	}
	if _, err = f.s.CatalogGetRecordV2(ctx, f.admin.ID, current.ID, "listing", true); err != nil {
		t.Fatal("listing source was deleted", err)
	}
}

func writeV203ContractSample(t *testing.T, schema string, value any) {
	t.Helper()
	dir := os.Getenv("DEUTERIUM_CONTRACT_SAMPLES")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(map[string]any{schema: value}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, schema+".json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRecordVisibilityV203CommissionHallAndPersistentCancellation(t *testing.T) {
	f := newCatalogFixture(t)
	f.server.CommerceCore = newCommerceTestCore()
	ctx := context.Background()
	commission := commerceCommissionTest(t, f, "hide-commission")
	if e := f.s.HideRecordV203(ctx, f.buyer.ID, "COMMISSION", commission.ID, "active-commission", commission.Version); e == nil {
		t.Fatal("active commission hidden")
	}
	public, err := f.s.CommerceListV2(ctx, f.admin.ID, store.CommerceFilterV2{Kind: "COMMISSION", Public: true, Limit: 20})
	if err != nil || len(public) != 1 {
		t.Fatal("open commission absent", err)
	}
	mutation, err := f.s.PrepareCommissionCancelV2(ctx, f.buyer.ID, commission.ID, "cancel-history", commission.Version, true)
	if err != nil {
		t.Fatal(err)
	}
	commission = commerceRunTest(t, f, mutation)
	public, err = f.s.CommerceListV2(ctx, f.buyer.ID, store.CommerceFilterV2{Kind: "COMMISSION", Public: true, Limit: 20})
	if err != nil || len(public) != 0 {
		t.Fatal("cancelled commission in hall", err)
	}
	if err = f.s.HideRecordV203(ctx, f.buyer.ID, "COMMISSION", commission.ID, "hide-cancelled", commission.Version); err != nil {
		t.Fatal(err)
	}
	personal, err := f.s.CommerceListV2(ctx, f.buyer.ID, store.CommerceFilterV2{Kind: "COMMISSION", Limit: 20})
	if err != nil || len(personal) != 0 {
		t.Fatal("cancelled commission reappeared", err)
	}
	if d := commerceRecordTest(t, f, commission.ID); d.State != "CANCELLED" || d.FundsState != "REFUNDED" {
		t.Fatal("commission receipt was changed")
	}
	// The public HTTP handler remains protected even though display changes do not move money.
	w := httptest.NewRecorder()
	f.server.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/commissions/"+commission.ID+"/hide", strings.NewReader(`{}`)))
	if w.Code != 401 {
		t.Fatal("unauthenticated hide accepted", w.Code)
	}
}

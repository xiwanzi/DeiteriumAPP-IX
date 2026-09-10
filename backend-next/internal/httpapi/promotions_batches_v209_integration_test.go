//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func couponDraftTestV209(t *testing.T, f catalogFixture, content store.CatalogObjectV2) store.CatalogRecordV2 {
	t.Helper()
	d, e := f.s.SaveCouponV209(context.Background(), f.admin.ID, "", store.ID("draft_"), 0, content)
	if e != nil {
		t.Fatal(e)
	}
	if d.State != "DRAFT" || d.Body["active"] != false {
		t.Fatal("creation exposed draft", d)
	}
	return d
}

func couponSelectionsV209(drafts ...store.CatalogRecordV2) []store.CouponDraftSelectionV209 {
	items := []store.CouponDraftSelectionV209{}
	for _, d := range drafts {
		items = append(items, store.CouponDraftSelectionV209{CouponID: d.ID, ExpectedVersion: d.Version})
	}
	return items
}

func TestCouponDraftBatchPublicationIsAtomicReplayableAndCompleteAcrossPagesV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	one := promotionCouponTestV209()
	one["type"], one["benefit"], one["discountRate"] = "ITEM", "PERCENT", 8500
	two := promotionCouponTestV209()
	two["type"], two["benefit"], two["discountRate"] = "ITEM", "PERCENT", 7000
	a, b := couponDraftTestV209(t, f, one), couponDraftTestV209(t, f, two)
	if rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 100); e != nil || len(rows) != 0 {
		t.Fatal("draft visible in wallet", rows, e)
	}
	if rows, _, _, e := f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 100); e != nil || len(rows) != 0 {
		t.Fatal("draft triggered arrival", rows, e)
	}
	draftRows, _, _, e := f.s.CouponsFilteredV209(ctx, f.admin.ID, true, "", "", 100, "DRAFT")
	if e != nil || len(draftRows) != 2 {
		t.Fatal("draft filter", draftRows, e)
	}
	if _, e := f.s.PublishCouponDraftsV209(ctx, f.buyer.ID, "forbidden-publish", couponSelectionsV209(a, b)); e == nil {
		t.Fatal("player published a batch")
	}
	batch, e := f.s.PublishCouponDraftsV209(ctx, f.admin.ID, "publish-two", couponSelectionsV209(a, b))
	if e != nil {
		t.Fatal(e)
	}
	if batch.BatchID == "" || len(batch.Coupons) != 2 {
		t.Fatal(batch)
	}
	replayed, e := f.s.PublishCouponDraftsV209(ctx, f.admin.ID, "publish-two", couponSelectionsV209(b, a))
	if e != nil || replayed.BatchID != batch.BatchID {
		t.Fatal("replay duplicated batch", replayed, e)
	}
	views, _, _, e := f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 1)
	if e != nil || len(views) != 2 {
		t.Fatal("pagination split publication", views, e)
	}
	for _, raw := range views {
		v := raw.(store.CatalogObjectV2)
		if v["releaseBatchId"] != batch.BatchID || v["announced"] != false {
			t.Fatal(v)
		}
	}
	if e := f.s.AcknowledgeCouponAttentionV209(ctx, f.buyer.ID, []string{a.ID}, false); e != nil {
		t.Fatal(e)
	}
	views, _, _, e = f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 100)
	if e != nil || len(views) != 2 {
		t.Fatal(views, e)
	}
	for _, raw := range views {
		if raw.(store.CatalogObjectV2)["announced"] != true {
			t.Fatal("same batch prompted again", raw)
		}
	}
	if e := f.s.AcknowledgeCouponAttentionV209(ctx, f.buyer.ID, []string{a.ID}, true); e != nil {
		t.Fatal(e)
	}
	views, _, _, e = f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 100)
	if e != nil || len(views) != 1 || views[0].(store.CatalogObjectV2)["couponId"] != b.ID {
		t.Fatal("viewing one hid unviewed companion", views, e)
	}
}

func TestCouponBatchRejectsStaleExpiredOrConcurrentDuplicateWithoutPartialIssueV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a, b := couponDraftTestV209(t, f, promotionCouponTestV209()), couponDraftTestV209(t, f, promotionCouponTestV209())
	changed := b.Body
	changed["name"] = "修改后的草稿"
	changed["active"] = true
	b2, e := f.s.SaveCouponV209(ctx, f.admin.ID, b.ID, "edit-before-publish", b.Version, changed)
	if e != nil || b2.State != "DRAFT" || b2.Body["active"] != false {
		t.Fatal("editing published draft", b2, e)
	}
	if _, e := f.s.PublishCouponDraftsV209(ctx, f.admin.ID, "stale", couponSelectionsV209(a, b)); e == nil {
		t.Fatal("stale batch accepted")
	}
	if rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 100); e != nil || len(rows) != 0 {
		t.Fatal("stale batch partially issued", rows, e)
	}
	expired := promotionCouponTestV209()
	expired["endsAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	c := couponDraftTestV209(t, f, expired)
	if _, e := f.s.PublishCouponDraftsV209(ctx, f.admin.ID, "expired", couponSelectionsV209(a, c)); e == nil {
		t.Fatal("expired batch accepted")
	}
	if rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 100); e != nil || len(rows) != 0 {
		t.Fatal("expired batch partially issued", rows, e)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, key := range []string{"concurrent-one", "concurrent-two"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, e := f.s.PublishCouponDraftsV209(ctx, f.admin.ID, key, couponSelectionsV209(a, b2))
			results <- e
		}(key)
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else if ce, ok := e.(*store.CatalogErrorV2); !ok || ce.Code != "COUPON_DRAFT_CHANGED" {
			t.Fatal("unexpected concurrency failure", e)
		}
	}
	if success != 1 {
		t.Fatal("duplicate publication", success)
	}
	var coupons, batches int
	if e := f.s.DB.QueryRow("SELECT COUNT(*),COUNT(DISTINCT batch_id) FROM promotion_coupon_batches_v209").Scan(&coupons, &batches); e != nil || coupons != 2 || batches != 1 {
		t.Fatal(coupons, batches, e)
	}
}

func TestCouponBatchLaterMemberDoesNotRepeatAnExpiredMembersReminderV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a := couponDraftTestV209(t, f, promotionCouponTestV209())
	future := promotionCouponTestV209()
	future["startsAt"] = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	future["endsAt"] = time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	b := couponDraftTestV209(t, f, future)
	batch, e := f.s.PublishCouponDraftsV209(ctx, f.admin.ID, "scheduled-batch", couponSelectionsV209(a, b))
	if e != nil {
		t.Fatal(e)
	}
	if e := f.s.AcknowledgeCouponAttentionV209(ctx, f.buyer.ID, []string{a.ID}, false); e != nil {
		t.Fatal(e)
	}
	for _, coupon := range batch.Coupons {
		content := coupon.Body
		if coupon.ID == a.ID {
			content["endsAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
		} else {
			content["startsAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
		}
		if _, e := f.s.SaveCouponV209(ctx, f.admin.ID, coupon.ID, "advance-"+coupon.ID, coupon.Version, content); e != nil {
			t.Fatal(e)
		}
	}
	views, _, _, e := f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 100)
	if e != nil || len(views) != 1 || views[0].(store.CatalogObjectV2)["announced"] != true {
		t.Fatal("later member re-announced batch", views, e)
	}
	other, _, _, e := f.s.CouponAttentionV209(ctx, f.other.ID, "", 100)
	if e != nil || len(other) != 1 || other[0].(store.CatalogObjectV2)["announced"] != false {
		t.Fatal("another player's new arrival suppressed", other, e)
	}
}

func TestCouponDraftAndBatchHTTPContractV209(t *testing.T) {
	f := newCatalogFixture(t)
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	request := func(path, token string, body any) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", path, strings.NewReader(string(raw)))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	content := promotionCouponTestV209()
	content["clientRequestId"] = "http-draft"
	w := request("/api/v1/admin/coupons", f.buyerToken, content)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request("/api/v1/admin/coupons", f.adminToken, content)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			CouponID         string `json:"couponId"`
			Version          int64  `json:"version"`
			PublicationState string `json:"publicationState"`
			Active           bool   `json:"active"`
		}
	}
	if e := json.Unmarshal(w.Body.Bytes(), &response); e != nil || response.Data.PublicationState != "DRAFT" || response.Data.Active {
		t.Fatal(w.Body.String(), e)
	}
	body := map[string]any{"clientRequestId": "http-batch", "coupons": []store.CouponDraftSelectionV209{{CouponID: response.Data.CouponID, ExpectedVersion: response.Data.Version}}}
	if w := request("/api/v1/admin/coupons/publish", f.buyerToken, body); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request("/api/v1/admin/coupons/publish", f.adminToken, body)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if dir := os.Getenv("DEUTERIUM_COUPON_QA_OUTPUT"); dir != "" {
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, "CouponPublicationResponseV209.json"), w.Body.Bytes(), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if w := request("/api/v1/admin/coupons/publish", f.adminToken, map[string]any{"clientRequestId": "bad-batch", "coupons": []any{map[string]any{"couponId": "x", "expectedVersion": 1, "active": true}}}); w.Code != 400 {
		t.Fatal("extra fields accepted", w.Code)
	}
}

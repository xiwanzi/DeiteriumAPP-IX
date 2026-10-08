//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCatalogOffersV220MatchCheckoutAndRemainAccountScoped(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	content["discountRate"] = 8000
	p = promotionPublishTestV209(t, f, p, content)
	all := promotionCouponTestV209()
	all["amountOff"] = "1.00"
	public := promotionSaveTestV209(t, f, all)
	private := promotionCouponTestV209()
	private["amountOff"] = "2.00"
	private["audience"] = "PLAYERS"
	private["playerRefs"] = []any{f.buyer.PlayerRef}
	exclusive := promotionSaveTestV209(t, f, private)
	read := func(user string) store.CatalogObjectV2 {
		t.Helper()
		v, e := f.s.CatalogOfferPreviewsV220(ctx, user, []store.CatalogRecordV2{p})
		if e != nil {
			t.Fatal(e)
		}
		return v[p.ID]
	}
	preview := read(f.buyer.ID)
	if preview["couponId"] != exclusive.ID || preview["finalPrice"] != "6.88" {
		t.Fatal(preview)
	}
	if read(f.other.ID)["couponId"] != public.ID {
		t.Fatal("coupon leaked to another account")
	}
	q := promotionQuoteTestV209(t, f, p, "preview-quote", 1, "DIRECT")
	if q["totalAmount"] != preview["finalPrice"] {
		t.Fatal("preview and quote differ", q, preview)
	}
	if catalogTestCount(t, f.s, "promotion_redemptions_v209") != 0 {
		t.Fatal("preview consumed coupon")
	}
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	for _, path := range []string{"/api/v1/store/products?limit=40", "/api/v1/store/products/" + p.ID} {
		request := httptest.NewRequest("GET", path, nil)
		request.Header.Set("Authorization", "Bearer "+f.buyerToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "offerPreview") || strings.Contains(w.Body.String(), "playerRefs") || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(w.Code, w.Body.String())
		}
		var value map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &value); e != nil {
			t.Fatal(e)
		}
	}
	// Reservation by another checkout must remove the used coupon immediately.
	promotionPrepareTestV209(t, f, q, "preview-reserve")
	if read(f.buyer.ID)["couponId"] != public.ID {
		t.Fatal("reserved coupon still advertised")
	}
	// Expired, not-yet-started and wrong-scope records cannot be advertised.
	for _, change := range []string{"expired", "future", "scope"} {
		c := promotionCouponTestV209()
		c["amountOff"] = "9.00"
		switch change {
		case "expired":
			c["startsAt"] = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
			c["endsAt"] = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
		case "future":
			c["startsAt"] = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
			c["endsAt"] = time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
		case "scope":
			c["productIds"] = []any{store.ID("other_product_")}
		}
		// Fixture inserts use valid published coupon records, bypassing administration's scheduling UI.
		raw, _ := json.Marshal(c)
		_, e := f.s.DB.Exec("INSERT INTO catalog_records_v2(resource_id,kind,store_id,owner_id,version,body,state,title,created_at,updated_at) VALUES(?,'coupon','',?,1,?,'ACTIVE','fixture',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", store.ID("coupon_"), f.admin.ID, string(raw))
		if e != nil {
			t.Fatal(e)
		}
	}
	if read(f.other.ID)["couponId"] != public.ID {
		t.Fatal("invalid coupon became best")
	}
	if _, e := f.s.DB.Exec("UPDATE identities SET game_verified=FALSE WHERE id=?", f.other.ID); e != nil {
		t.Fatal(e)
	}
	if v := read(f.other.ID); v != nil {
		t.Fatal("unverified account advertised a game purchase coupon", fmt.Sprint(v))
	}
}

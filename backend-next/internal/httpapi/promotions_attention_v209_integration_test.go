//go:build integration

package httpapi

import (
	"context"
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

func TestCouponAttentionReceiptsAreMonotonicIsolatedAndDoNotConsumeCouponsV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a := promotionSaveTestV209(t, f, promotionCouponTestV209())
	b := promotionSaveTestV209(t, f, promotionCouponTestV209())
	private := promotionCouponTestV209()
	private["audience"], private["playerRefs"] = "PLAYERS", []any{f.other.PlayerRef}
	c := promotionSaveTestV209(t, f, private)
	read := func(actor string) map[string]bool {
		t.Helper()
		rows, _, _, e := f.s.CouponAttentionV209(ctx, actor, "", 100)
		if e != nil {
			t.Fatal(e)
		}
		result := map[string]bool{}
		for _, row := range rows {
			v := row.(store.CatalogObjectV2)
			result[v["couponId"].(string)] = v["announced"].(bool)
		}
		return result
	}
	for i := 0; i < 2; i++ {
		if rows := read(f.buyer.ID); len(rows) != 2 || rows[a.ID] || rows[b.ID] {
			t.Fatal("GET consumed or leaked attention", rows)
		}
	}
	if e := f.s.AcknowledgeCouponAttentionV209(ctx, f.buyer.ID, []string{a.ID, c.ID}, false); e != nil {
		t.Fatal(e)
	}
	if rows := read(f.buyer.ID); len(rows) != 2 || !rows[a.ID] || rows[b.ID] {
		t.Fatal(rows)
	}
	if rows := read(f.other.ID); len(rows) != 3 || rows[a.ID] || rows[c.ID] {
		t.Fatal("another player receipt changed", rows)
	}
	// A second device repeats the same acknowledgement while a wallet marks it viewed.
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(viewed bool) {
			defer wg.Done()
			if e := f.s.AcknowledgeCouponAttentionV209(ctx, f.buyer.ID, []string{a.ID}, viewed); e != nil {
				t.Error(e)
			}
		}(i%2 == 0)
	}
	wg.Wait()
	if rows := read(f.buyer.ID); len(rows) != 1 || rows[b.ID] {
		t.Fatal("receipt regressed or acknowledged unseen batch", rows)
	}
	rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 100)
	if e != nil || len(rows) != 2 {
		t.Fatal("ack consumed actual entitlement", rows, e)
	}
	var count int
	if e = f.s.DB.QueryRow(`SELECT COUNT(*) FROM promotion_attention_v209 WHERE coupon_id=?`, c.ID).Scan(&count); e != nil || count != 0 {
		t.Fatal("forged private receipt was accepted", count, e)
	}
}

func TestCouponAttentionHTTPPagingExpiryAndLaterRegistrationV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	a := promotionSaveTestV209(t, f, promotionCouponTestV209())
	b := promotionSaveTestV209(t, f, promotionCouponTestV209())
	rows, cursor, more, e := f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 1)
	if e != nil || len(rows) != 1 || !more || cursor == "" {
		t.Fatal(rows, cursor, more, e)
	}
	rows, _, more, e = f.s.CouponAttentionV209(ctx, f.buyer.ID, cursor, 1)
	if e != nil || len(rows) != 1 || more {
		t.Fatal("older available coupon lost", rows, more, e)
	}
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	request := func(method, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1/store/coupons/attention", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	capture := func(name string, w *httptest.ResponseRecorder) {
		t.Helper()
		if dir := os.Getenv("DEUTERIUM_COUPON_QA_OUTPUT"); dir != "" {
			if e := os.MkdirAll(dir, 0700); e != nil {
				t.Fatal(e)
			}
			if e := os.WriteFile(filepath.Join(dir, name+".json"), w.Body.Bytes(), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	if w := request("GET", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request("POST", `{"couponIds":["`+a.ID+`"],"viewed":true}`, f.buyerToken); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	} else {
		capture("CouponAttentionAckResponseV209", w)
	}
	if w := request("POST", `{"couponIds":[],"viewed":true}`, f.buyerToken); w.Code < 400 {
		t.Fatal("empty ack accepted")
	}
	if w := request("GET", "", f.buyerToken); w.Code != 200 || strings.Contains(w.Body.String(), a.ID) || !strings.Contains(w.Body.String(), b.ID) || strings.Contains(w.Body.String(), "playerRefs") {
		t.Fatal(w.Code, w.Body.String())
	} else {
		capture("CouponAttentionListResponseV209", w)
	}
	if _, e := f.s.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) SELECT 'notice_new','player_notice_new','10000000-0000-0000-0000-000000000098','NoticeNew','999998','test','active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),legacy_fingerprint FROM identities WHERE id=?`, f.other.ID); e != nil {
		t.Fatal(e)
	}
	if rows, _, _, e := f.s.CouponAttentionV209(ctx, "notice_new", "", 100); e != nil || len(rows) != 2 {
		t.Fatal("new player arrival missing", rows, e)
	}
	expired := b.Body
	expired["endsAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	if _, e = f.s.SaveCouponV209(ctx, f.admin.ID, b.ID, "expire-notice", b.Version, expired); e != nil {
		t.Fatal(e)
	}
	if rows, _, _, e := f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 100); e != nil || len(rows) != 0 {
		t.Fatal("expired reminder visible", rows, e)
	}
}

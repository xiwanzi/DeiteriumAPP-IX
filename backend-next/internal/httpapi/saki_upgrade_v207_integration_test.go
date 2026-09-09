//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func sakiUpgradeFixtureV207(t *testing.T) (*aiFixtureV2, *commerceTestCore, store.AISettingsV206, time.Time) {
	f, c, v := sakiFixtureV206(t)
	v.Plans[2].Active = true
	v.Plans[2].Price = "42.50"
	sakiSaveV206(t, f, v, 0, "open-upgrades")
	m, err := f.db.PrepareAIPurchaseV206(context.Background(), f.users["Alice"].ID, store.AIPurchaseInputV206{ClientRequestID: "base", PlanID: "plan_pro", ExpectedPlanVersion: 2}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	op, err := f.app.RunCommerceOperationV2(context.Background(), m.OperationID, true)
	if err != nil || op.State != "COMPLETED" {
		t.Fatal(op, err)
	}
	expiry := time.Now().UTC().Add(7*24*time.Hour + 12*time.Hour).Truncate(time.Microsecond)
	if _, err = f.db.DB.Exec("UPDATE ai_entitlements_v206 SET expires_at=? WHERE user_id=?", expiry, f.users["Alice"].ID); err != nil {
		t.Fatal(err)
	}
	return f, c, v, expiry
}
func sakiQuoteV207(t *testing.T, f *aiFixtureV2, key string) store.AIQuoteV207 {
	t.Helper()
	q, err := f.db.AIQuoteV207(context.Background(), f.users["Alice"].ID, store.AIPurchaseInputV206{ClientRequestID: key, PlanID: "plan_ultra", ExpectedPlanVersion: 2}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestSakiUpgradeQuoteExpiresWhenRoundedDayChangesV207(t *testing.T) {
	f, _, _, _ := sakiUpgradeFixtureV207(t)
	expiry := time.Now().UTC().Add(7*24*time.Hour + 20*time.Second).Truncate(time.Microsecond)
	if _, err := f.db.DB.Exec("UPDATE ai_entitlements_v206 SET expires_at=? WHERE user_id=?", expiry, f.users["Alice"].ID); err != nil {
		t.Fatal(err)
	}
	q := sakiQuoteV207(t, f, "day-boundary")
	if q.RemainingDays != 8 || !q.ExpiresAt.Equal(expiry.Add(-7*24*time.Hour)) {
		t.Fatalf("quote outlived its billable day: %+v", q)
	}
}
func TestSakiUpgradeSnapshotAmountExpiryAndConcurrentRecoveryV207(t *testing.T) {
	f, c, _, expiry := sakiUpgradeFixtureV207(t)
	ctx := context.Background()
	actor := f.users["Alice"].ID
	q := sakiQuoteV207(t, f, "quote")
	quoteHTTP := f.json(t, "Alice", "POST", "/api/v1/ai/purchase-quotes", store.AIPurchaseInputV206{ClientRequestID: "quote", PlanID: "plan_ultra", ExpectedPlanVersion: 2}, 200)
	if q.Kind != "UPGRADE" || q.TotalAmount != "8.00" || q.RemainingDays != 8 || q.ChargedDays != 8 || q.BillingCycleDays != 30 || q.PriceDifference != "30.00" || !q.EntitlementExpiresAt.Equal(expiry) {
		t.Fatalf("bad quote: %+v", q)
	}
	q2 := sakiQuoteV207(t, f, "quote")
	if q2.QuoteID != q.QuoteID {
		t.Fatal("quote replay changed")
	}
	input := store.AIPurchaseInputV206{ClientRequestID: "upgrade", PlanID: "plan_ultra", ExpectedPlanVersion: 2, QuoteID: q.QuoteID}
	var wg sync.WaitGroup
	ids := make(chan store.CommerceMutationV2, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := f.db.PrepareAIPurchaseV206(ctx, actor, input, true, true)
			if e != nil {
				errs <- e
			} else {
				ids <- m
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	var m store.CommerceMutationV2
	for got := range ids {
		if m.OperationID != "" && m.OperationID != got.OperationID {
			t.Fatal("duplicate upgrade")
		}
		m = got
	}
	c.unknownOnce = CommerceSettleV2
	op, err := f.app.RunCommerceOperationV2(ctx, m.OperationID, true)
	if err != nil || op.State != "UNKNOWN" {
		t.Fatal(op, err)
	}
	state, err := f.db.AIStateV2(ctx, actor, store.AIPolicyV2{Configured: true}, time.Now())
	if err != nil || state.Plan.Code != "pro" {
		t.Fatal("upgrade before settlement", err)
	}
	op, err = f.app.RunCommerceOperationV2(ctx, m.OperationID, false)
	if err != nil || op.State != "COMPLETED" {
		t.Fatal(op, err)
	}
	state, err = f.db.AIStateV2(ctx, actor, store.AIPolicyV2{Configured: true}, time.Now())
	if err != nil || state.Plan.Code != "ultra" || !state.ExpiresAt.Equal(expiry) || state.Plan.Price != "42.50" {
		t.Fatalf("upgrade changed expiry/price: %+v %v", state, err)
	}
	d, err := f.db.CommerceRecordV2(ctx, m.ResourceID)
	if err != nil || d.Amount != "8.00" || c.calls[CommerceReserveV2] != 2 || c.calls[CommerceSettleV2] != 2 {
		t.Fatalf("wrong money: %+v %v", d, err)
	}
	if _, err = f.db.PrepareAIPurchaseV206(ctx, actor, input, false, false); err != nil {
		t.Fatal("accepted payment replay lost", err)
	}
	input.ClientRequestID = "duplicate-quote"
	if _, err = f.db.PrepareAIPurchaseV206(ctx, actor, input, true, true); err == nil {
		t.Fatal("spent quote reused")
	}
	low := store.AIPurchaseInputV206{ClientRequestID: "downgrade", PlanID: "plan_pro", ExpectedPlanVersion: 2}
	if _, err = f.db.AIQuoteV207(ctx, actor, low, true, true); err == nil {
		t.Fatal("downgrade quote accepted")
	}
	if _, err = f.db.PrepareAIPurchaseV206(ctx, actor, low, true, true); err == nil {
		t.Fatal("legacy downgrade accepted")
	}
	view := f.json(t, "Alice", "GET", "/api/v1/orders/"+m.ResourceID, nil, 200)
	f.json(t, "Alice", "POST", "/api/v1/orders/"+m.ResourceID+"/refunds", map[string]any{"clientRequestId": "refund-upgrade", "expectedVersion": d.Version, "reasonCode": "OTHER", "description": "refund", "evidenceAssetIds": []string{}}, 409)
	if dir := os.Getenv("DEUTERIUM_CONTRACT_SAMPLES"); dir != "" {
		os.MkdirAll(dir, 0700)
		raw, _ := json.MarshalIndent(map[string]any{"V2OrderViewResponse": view, "AiPurchaseQuoteResponse": quoteHTTP}, "", "  ")
		if err = os.WriteFile(filepath.Join(dir, "saki-upgrade-v207.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSakiUpgradeQuoteRejectsDriftIsolationAndExpiryV207(t *testing.T) {
	for _, mode := range []string{"expired", "entitlement", "price", "owner", "legacy", "pending", "quote-used"} {
		t.Run(mode, func(t *testing.T) {
			f, _, v, _ := sakiUpgradeFixtureV207(t)
			ctx := context.Background()
			q := sakiQuoteV207(t, f, "quote")
			actor := f.users["Alice"].ID
			input := store.AIPurchaseInputV206{ClientRequestID: "upgrade", PlanID: "plan_ultra", ExpectedPlanVersion: 2, QuoteID: q.QuoteID}
			switch mode {
			case "expired":
				f.db.DB.Exec("UPDATE catalog_quotes_v2 SET expires_at=? WHERE quote_id=?", time.Now().Add(-time.Second), q.QuoteID)
			case "entitlement":
				f.db.DB.Exec("UPDATE ai_entitlements_v206 SET expires_at=DATE_ADD(expires_at,INTERVAL 1 DAY) WHERE user_id=?", actor)
			case "price":
				v.Plans[2].Price = "60.00"
				sakiSaveV206(t, f, v, 1, "reprice")
			case "owner":
				actor = f.users["Bob"].ID
			case "legacy":
				input.QuoteID = ""
			case "pending", "quote-used":
				first := input
				first.ClientRequestID = "first"
				if _, err := f.db.PrepareAIPurchaseV206(ctx, actor, first, true, true); err != nil {
					t.Fatal(err)
				}
				if mode == "pending" {
					input.QuoteID = sakiQuoteV207(t, f, "other-quote").QuoteID
				}
			}
			if _, err := f.db.PrepareAIPurchaseV206(ctx, actor, input, true, true); err == nil {
				t.Fatal("invalid quote accepted")
			}
			var count int
			f.db.DB.QueryRow("SELECT COUNT(*) FROM commerce_resources_v2").Scan(&count)
			want := 1
			if mode == "pending" || mode == "quote-used" {
				want = 2
			}
			if count != want {
				t.Fatal("rejection created order", count)
			}
		})
	}
}

func TestSakiUpgradeUsesPaidPriceAndFailedSettlementKeepsEntitlementV207(t *testing.T) {
	f, c, v, expiry := sakiUpgradeFixtureV207(t)
	ctx := context.Background()
	actor := f.users["Alice"].ID
	v.Plans[1].Price = "20.00"
	v.Plans[1].DurationDays = 15
	sakiSaveV206(t, f, v, 1, "new-price")
	input := store.AIPurchaseInputV206{ClientRequestID: "quote", PlanID: "plan_ultra", ExpectedPlanVersion: 3}
	q, err := f.db.AIQuoteV207(ctx, actor, input, true, true)
	if err != nil || q.TotalAmount != "8.00" || q.PreviousPrice != "12.50" || q.BillingCycleDays != 30 {
		t.Fatal("lost paid snapshot", q, err)
	}
	input.ClientRequestID = "upgrade"
	input.QuoteID = q.QuoteID
	m, err := f.db.PrepareAIPurchaseV206(ctx, actor, input, true, true)
	if err != nil {
		t.Fatal(err)
	}
	c.failOnce = CommerceSettleV2
	op, err := f.app.RunCommerceOperationV2(ctx, m.OperationID, true)
	if err != nil || op.State != "FAILED" {
		t.Fatal(op, err)
	}
	state, err := f.db.AIStateV2(ctx, actor, store.AIPolicyV2{Configured: true}, time.Now())
	if err != nil || state.Plan.Code != "pro" || !state.ExpiresAt.Equal(expiry) || c.calls[CommerceRefundV2] != 1 {
		t.Fatal("failed upgrade changed old entitlement", state, err)
	}
}

func TestSakiUpgradeLastDayKeepsFiveDayFloorAndActualExpiryV207(t *testing.T) {
	f, _, _, _ := sakiUpgradeFixtureV207(t)
	ctx := context.Background()
	actor := f.users["Alice"].ID
	expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	if _, err := f.db.DB.Exec("UPDATE ai_entitlements_v206 SET expires_at=? WHERE user_id=?", expiry, actor); err != nil {
		t.Fatal(err)
	}
	q := sakiQuoteV207(t, f, "last-day")
	if q.TotalAmount != "5.00" || q.RemainingDays != 1 || q.ChargedDays != 5 || q.BillingCycleDays != 30 {
		t.Fatalf("floor not retained: %+v", q)
	}
	input := store.AIPurchaseInputV206{ClientRequestID: "last-day-upgrade", PlanID: "plan_ultra", ExpectedPlanVersion: 2, QuoteID: q.QuoteID}
	m, err := f.db.PrepareAIPurchaseV206(ctx, actor, input, true, true)
	if err != nil {
		t.Fatal(err)
	}
	op, err := f.app.RunCommerceOperationV2(ctx, m.OperationID, true)
	if err != nil || op.State != "COMPLETED" {
		t.Fatal(op, err)
	}
	state, err := f.db.AIStateV2(ctx, actor, store.AIPolicyV2{Configured: true}, time.Now())
	if err != nil || !state.ExpiresAt.Equal(expiry) {
		t.Fatal("floor extended subscription", state, err)
	}
}

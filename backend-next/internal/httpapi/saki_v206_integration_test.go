//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sakiFixtureV206(t *testing.T) (*aiFixtureV2, *commerceTestCore, store.AISettingsV206) {
	var calls atomic.Int32
	f := newAIFixtureV2(t, aiFastTestProviderV2(&calls))
	c := newCommerceTestCore()
	f.app.CommerceCore = c
	mux := f.http.Config.Handler.(*http.ServeMux)
	f.app.registerCommerceV2(mux)
	f.app.registerAdminAccountsV206(mux)
	v := f.config.settingsV206()
	v.PaidEnabled = true
	v.Knowledge = []store.AIKnowledgeV206{{ID: "community", Title: "社区", Content: "社区名称 Deuterium IX", Enabled: true}}
	plans, err := f.db.AIPlansV2(context.Background(), f.config.policy())
	if err != nil {
		t.Fatal(err)
	}
	for i := range plans {
		if plans[i].Code == "pro" {
			plans[i].Active = true
			plans[i].Name = "小祥 Plus"
			plans[i].Price = "12.50"
			plans[i].QuotaPerWindow = 12
			plans[i].WindowHours = 2
			plans[i].DurationDays = 30
		}
	}
	v.Plans = plans
	return f, c, v
}
func sakiSaveV206(t *testing.T, f *aiFixtureV2, v store.AISettingsV206, expected int64, key string) {
	t.Helper()
	f.json(t, "Admin", "PUT", "/api/v1/admin/ai-settings", map[string]any{"clientRequestId": key, "expectedVersion": expected, "settings": v}, 200)
}
func TestSakiConfigurationAuthorityVersionAndRuntimeV206(t *testing.T) {
	f, _, v := sakiFixtureV206(t)
	f.json(t, "Alice", "GET", "/api/v1/admin/ai-settings", nil, 403)
	sakiSaveV206(t, f, v, 0, "settings-one")
	sakiSaveV206(t, f, v, 0, "settings-one")
	f.json(t, "Admin", "PUT", "/api/v1/admin/ai-settings", map[string]any{"clientRequestId": "stale", "expectedVersion": 0, "settings": v}, 409)
	r := f.json(t, "Alice", "GET", "/api/v1/ai/plans", nil, 200)
	raw, _ := json.Marshal(r)
	if !strings.Contains(string(raw), "小祥 Plus") || !strings.Contains(string(raw), `"purchasable":true`) {
		t.Fatalf("plans not live: %s", raw)
	}
	v.WebSearch = false
	v.Plans[0].QuotaPerWindow = 7
	v.Prompt = "Runtime prompt"
	sakiSaveV206(t, f, v, 1, "settings-two")
	r = f.json(t, "Alice", "GET", "/api/v1/ai/me", nil, 200)
	raw, _ = json.Marshal(r)
	if !strings.Contains(string(raw), `"limit":7`) || !strings.Contains(string(raw), `"webSearchAvailable":false`) {
		t.Fatalf("runtime settings not applied: %s", raw)
	}
	invalid := v
	invalid.MaxConcurrent = 0
	f.json(t, "Admin", "PUT", "/api/v1/admin/ai-settings", map[string]any{"clientRequestId": "bad-settings", "expectedVersion": 2, "settings": invalid}, 400)
}
func TestSakiPurchaseIdempotencyEntitlementSnapshotAndRefundV206(t *testing.T) {
	f, c, v := sakiFixtureV206(t)
	ctx := context.Background()
	sakiSaveV206(t, f, v, 0, "open-sales")
	input := store.AIPurchaseInputV206{ClientRequestID: "saki-one", PlanID: "plan_pro", ExpectedPlanVersion: 2}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := f.db.PrepareAIPurchaseV206(ctx, f.users["Alice"].ID, input, true, true)
			if e != nil {
				errs <- e
			} else {
				ids <- m.OperationID
			}
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	id := ""
	for got := range ids {
		if id != "" && id != got {
			t.Fatal("duplicate intent")
		}
		id = got
	}
	op, err := f.app.RunCommerceOperationV2(ctx, id, true)
	if err != nil || op.State != "COMPLETED" {
		t.Fatalf("purchase: %+v %v", op, err)
	}
	for i := 0; i < 3; i++ {
		if _, err = f.app.RunCommerceOperationV2(ctx, id, true); err != nil {
			t.Fatal(err)
		}
	}
	if c.calls[CommerceReserveV2] != 1 || c.calls[CommerceSettleV2] != 1 {
		t.Fatal("duplicate debit or settlement")
	}
	state, err := f.db.AIStateV2(ctx, f.users["Alice"].ID, store.AIPolicyV2{Configured: true}, time.Now().UTC())
	if err != nil || state.Plan.Name != "小祥 Plus" || state.Quota.Limit != 12 || state.Quota.WindowHours != 2 || state.ExpiresAt == nil {
		t.Fatalf("entitlement: %+v %v", state, err)
	}
	oldExpiry := *state.ExpiresAt
	order, err := f.db.CommerceViewV2(ctx, f.users["Alice"].ID, op.ResourceID, false)
	if err != nil {
		t.Fatal(err)
	}
	if order["orderType"] != "AI_SUBSCRIPTION" || order["status"] != "CONFIRMED" || len(order["availableActions"].([]string)) != 0 {
		t.Fatalf("not a digital order: %+v", order)
	}
	if dir := os.Getenv("DEUTERIUM_CONTRACT_SAMPLES"); dir != "" {
		samples := map[string]any{
			"V2OrderViewResponse":             f.json(t, "Alice", "GET", "/api/v1/orders/"+op.ResourceID, nil, 200),
			"V2OrderCreationResultResponse":   f.json(t, "Alice", "POST", "/api/v1/ai/purchases", input, 200),
			"V2OrderContractSnapshotResponse": f.json(t, "Alice", "GET", "/api/v1/orders/"+op.ResourceID+"/snapshot", nil, 200),
			"AiMeResponse":                    f.json(t, "Alice", "GET", "/api/v1/ai/me", nil, 200),
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(samples, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "saki-v206.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	f.json(t, "Alice", "POST", "/api/v1/orders/"+op.ResourceID+"/refunds", map[string]any{"clientRequestId": "try-refund", "expectedVersion": order["version"], "reasonCode": "OTHER", "description": "refund", "evidenceAssetIds": []string{}}, 409)
	f.json(t, "Bob", "GET", "/api/v1/ai/purchases/"+op.ResourceID, nil, 404)
	f.json(t, "Alice", "GET", "/api/v1/orders?channel=OFFICIAL_STORE", nil, 200)
	v.Plans[1].Name = "新名称"
	v.Plans[1].Price = "18.00"
	v.Plans[1].QuotaPerWindow = 18
	sakiSaveV206(t, f, v, 1, "change-plan")
	state, err = f.db.AIStateV2(ctx, f.users["Alice"].ID, store.AIPolicyV2{Configured: true}, time.Now().UTC())
	if err != nil || state.Plan.Name != "小祥 Plus" || !state.ExpiresAt.Equal(oldExpiry) {
		t.Fatal("old purchase changed")
	}
	input.ClientRequestID = "stale-price"
	if _, err = f.db.PrepareAIPurchaseV206(ctx, f.users["Alice"].ID, input, true, true); err == nil {
		t.Fatal("stale price accepted")
	}
	input.ClientRequestID = "renew"
	input.ExpectedPlanVersion = 3
	m, err := f.db.PrepareAIPurchaseV206(ctx, f.users["Alice"].ID, input, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.app.RunCommerceOperationV2(ctx, m.OperationID, true); err != nil {
		t.Fatal(err)
	}
	state, err = f.db.AIStateV2(ctx, f.users["Alice"].ID, store.AIPolicyV2{Configured: true}, time.Now().UTC())
	if err != nil || !state.ExpiresAt.Equal(oldExpiry.AddDate(0, 0, 30)) {
		t.Fatal("renewal did not extend once")
	}
	state, err = f.db.AIStateV2(ctx, f.users["Alice"].ID, store.AIPolicyV2{Configured: true}, state.ExpiresAt.Add(time.Second))
	if err != nil || state.Plan.Code != "free" {
		t.Fatal("expiry failed")
	}
	notices, err := f.db.NotificationsV2(ctx, f.users["Alice"].ID, 0, false, 100)
	if err != nil || len(notices) != 2 {
		t.Fatalf("noisy purchase notifications: %d %v", len(notices), err)
	}
	for _, n := range notices {
		if n.SystemPush {
			t.Fatal("purchase unnecessarily pushed")
		}
	}
}
func TestSakiUnknownRecoveryAndFailedSettlementCompensationV206(t *testing.T) {
	for _, mode := range []string{"unknown", "reserve-fail", "settle-fail"} {
		t.Run(mode, func(t *testing.T) {
			f, c, v := sakiFixtureV206(t)
			sakiSaveV206(t, f, v, 0, "open")
			ctx := context.Background()
			if mode == "unknown" {
				c.unknownOnce = CommerceSettleV2
			} else if mode == "reserve-fail" {
				c.failOnce = CommerceReserveV2
			} else {
				c.failOnce = CommerceSettleV2
			}
			m, err := f.db.PrepareAIPurchaseV206(ctx, f.users["Alice"].ID, store.AIPurchaseInputV206{ClientRequestID: "purchase", PlanID: "plan_pro", ExpectedPlanVersion: 2}, true, true)
			if err != nil {
				t.Fatal(err)
			}
			op, err := f.app.RunCommerceOperationV2(ctx, m.OperationID, true)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "unknown" {
				if op.State != "UNKNOWN" {
					t.Fatalf("not unknown: %s", op.State)
				}
				var count int
				f.db.DB.QueryRow("SELECT COUNT(*) FROM ai_entitlements_v206").Scan(&count)
				if count != 0 {
					t.Fatal("entitlement before proof")
				}
				v.PaidEnabled = false
				sakiSaveV206(t, f, v, 1, "close")
				op, err = f.app.RunCommerceOperationV2(ctx, m.OperationID, false)
				if err != nil || op.State != "COMPLETED" {
					t.Fatal("recovery lost after disabling sales", err)
				}
				if c.calls[CommerceSettleV2] != 1 {
					t.Fatal("recovery reran money")
				}
			} else {
				if op.State != "FAILED" {
					t.Fatalf("failure became success: %+v", op)
				}
				var count int
				f.db.DB.QueryRow("SELECT COUNT(*) FROM ai_entitlements_v206").Scan(&count)
				if count != 0 {
					t.Fatal("failed payment got entitlement")
				}
				if mode == "settle-fail" {
					d, _ := f.db.CommerceRecordV2(ctx, m.ResourceID)
					if d.FundsState != "REFUNDED" || c.calls[CommerceRefundV2] != 1 {
						t.Fatalf("hold not returned: %+v", d)
					}
				}
			}
		})
	}
}
func TestAccountManagementRevokesSessionsAndRejectsSelfAndStaleV206(t *testing.T) {
	f, _, _ := sakiFixtureV206(t)
	f.json(t, "Alice", "GET", "/api/v1/admin/accounts", nil, 403)
	f.json(t, "Admin", "GET", "/api/v1/admin/accounts?q=Alice", nil, 200)
	change := func(target, action, key string, version int64, status int) {
		f.json(t, "Admin", "POST", "/api/v1/admin/accounts/"+f.users[target].ID, store.AdminAccountChangeV206{ClientRequestID: key, ExpectedVersion: version, Action: action, Reason: "社区规则"}, status)
	}
	change("Admin", "ban", "self", 1, 409)
	change("Alice", "grant-admin", "grant", 1, 200)
	change("Alice", "ban", "stale", 1, 409)
	f.json(t, "Alice", "GET", "/api/v1/admin/accounts", nil, 200)
	change("Alice", "ban", "ban", 2, 200)
	f.json(t, "Alice", "GET", "/api/v1/admin/accounts", nil, 401)
	f.json(t, "Alice", "GET", "/api/v1/ai/me", nil, 401)
	change("Alice", "unban", "unban", 3, 200)
	f.json(t, "Alice", "GET", "/api/v1/ai/me", nil, 401)
}

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
	"sync"
	"testing"
	"time"
)

func promotionCouponTestV209() store.CatalogObjectV2 {
	return store.CatalogObjectV2{"name": "旗舰店礼遇", "type": "ORDER", "benefit": "FIXED", "amountOff": "3.00", "discountRate": 9000, "minimumSpend": "0.00", "maxDiscount": "0.00", "stackWithProductDiscount": true, "audience": "ALL", "playerRefs": []any{}, "storeIds": []any{}, "productIds": []any{}, "startsAt": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), "endsAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "active": true}
}
func promotionQuoteTestV209(t *testing.T, f catalogFixture, p store.CatalogRecordV2, key string, quantity int, source string) store.CatalogObjectV2 {
	t.Helper()
	q, e := f.s.CatalogQuoteV2(context.Background(), f.buyer.ID, key, store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "source": source, "items": []any{map[string]any{"productId": p.ID, "quantity": quantity, "expectedProductVersion": p.PublishedVersion}}, "delivery": map[string]any{"method": "MAILBOX"}})
	if e != nil {
		t.Fatal(e)
	}
	return q
}
func promotionPublishTestV209(t *testing.T, f catalogFixture, p store.CatalogRecordV2, content store.CatalogObjectV2) store.CatalogRecordV2 {
	t.Helper()
	ctx := context.Background()
	d, e := f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", store.ID("edit_"), p.Version, content, false)
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.s.CatalogActionV2(ctx, f.admin.ID, d.ID, "product", store.ID("publish_"), d.Version, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func promotionSaveTestV209(t *testing.T, f catalogFixture, c store.CatalogObjectV2) store.CatalogRecordV2 {
	t.Helper()
	d, e := f.s.SaveCouponV209(context.Background(), f.admin.ID, "", store.ID("coupon_create_"), 0, c)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func promotionPrepareTestV209(t *testing.T, f catalogFixture, q store.CatalogObjectV2, key string) store.CommerceMutationV2 {
	t.Helper()
	m, e := f.s.PrepareOrderV2(context.Background(), f.buyer.ID, key, fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil)
	if e != nil {
		t.Fatal(e)
	}
	return m
}

func TestPromotionAudienceExpiryNewPlayersAndHTTPIsolationV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	all := promotionSaveTestV209(t, f, promotionCouponTestV209())
	target := promotionCouponTestV209()
	target["audience"] = "PLAYERS"
	target["playerRefs"] = []any{f.buyer.PlayerRef, f.admin.PlayerRef}
	promotionSaveTestV209(t, f, target)
	if _, e := f.s.SaveCouponV209(ctx, f.buyer.ID, "", "forged-admin", 0, promotionCouponTestV209()); e == nil {
		t.Fatal("ordinary player issued coupons")
	}
	if _, e := f.s.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) SELECT 'new_player','player_new','10000000-0000-0000-0000-000000000099','NewPlayer','999999','test','active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),legacy_fingerprint FROM identities WHERE id=?`, f.other.ID); e != nil {
		t.Fatal(e)
	}
	rows, _, _, e := f.s.CouponsV209(ctx, "new_player", false, "", "", 30)
	if e != nil || len(rows) != 1 || rows[0].ID != all.ID {
		t.Fatal("new signup entitlement", rows, e)
	}
	mux := http.NewServeMux()
	f.server.registerCatalogV2(mux)
	r := httptest.NewRequest("GET", "/api/v1/store/coupons", nil)
	r.Header.Set("Authorization", "Bearer "+f.buyerToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), f.admin.PlayerRef) || strings.Contains(w.Body.String(), "playerRefs") {
		t.Fatal("recipient list exposed", w.Code, w.Body.String())
	}
	r = httptest.NewRequest("GET", "/api/v1/admin/coupons", nil)
	r.Header.Set("Authorization", "Bearer "+f.buyerToken)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	expired := all.Body
	expired["startsAt"] = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	expired["endsAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	if _, e = f.s.SaveCouponV209(ctx, f.admin.ID, all.ID, "expire", all.Version, expired); e != nil {
		t.Fatal(e)
	}
	rows, _, _, e = f.s.CouponsV209(ctx, "new_player", false, "", "", 30)
	if e != nil || len(rows) != 0 {
		t.Fatal("expired coupon remained visible", rows, e)
	}
}

func TestPromotionCheckoutClearsPurchasedUnitsOnceAndFreezesCurrentShopV209(t *testing.T) {
	f := newCatalogFixture(t)
	core := newCommerceTestCore()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	content["discountRate"] = 8000
	p = promotionPublishTestV209(t, f, p, content)
	shop, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.StoreID, "store", true)
	if e != nil {
		t.Fatal(e)
	}
	shop.Body["name"] = "EOS Lab旗舰店"
	if _, e = f.s.CatalogEditV2(ctx, f.admin.ID, shop.ID, "store", "rename", shop.Version, shop.Body, false); e != nil {
		t.Fatal(e)
	}
	promotionSaveTestV209(t, f, promotionCouponTestV209())
	cart, e := f.s.CatalogCartChangeV2(ctx, f.buyer.ID, p.ID, "bag-two", 1, 2, false)
	if e != nil {
		t.Fatal(e)
	}
	q := promotionQuoteTestV209(t, f, p, "discount-quote", 2, "CART")
	if q["storeName"] != "EOS Lab旗舰店" || q["totalAmount"] != "14.76" {
		t.Fatal(q)
	}
	m := promotionPrepareTestV209(t, f, q, "checkout")
	if _, e = f.s.CatalogCartChangeV2(ctx, f.buyer.ID, p.ID, "bag-extra", cart.Version, 3, false); e != nil {
		t.Fatal(e)
	}
	before, _ := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if before.Items[0].Quantity != 3 {
		t.Fatal("cart changed before payment")
	}
	d := commerceRunTest(t, f, m)
	view, e := f.s.CommerceViewV2(ctx, f.buyer.ID, d.ID, false)
	if e != nil || view["amount"] != "14.76" || view["originalTotal"] != "22.20" || view["couponDiscount"] != "3.00" {
		t.Fatal(view, e)
	}
	plan := d.Body["mailboxPlan"].(map[string]any)
	if plan["sender"] != "EOS Lab旗舰店" {
		t.Fatal(plan["sender"])
	}
	after, _ := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if len(after.Items) != 1 || after.Items[0].Quantity != 1 {
		t.Fatal("extra bag units lost", after)
	}
	replay := promotionPrepareTestV209(t, f, q, "checkout")
	commerceRunTest(t, f, replay)
	again, _ := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if again.Version != after.Version || again.Items[0].Quantity != 1 {
		t.Fatal("replay removed items twice", again)
	}
	rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 30)
	if e != nil || len(rows) != 0 {
		t.Fatal("used coupon still usable", rows, e)
	}
	refund, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, store.CatalogObjectV2{"reasonCode": "NO_LONGER_NEEDED", "description": "退回测试商品", "evidenceAssetIds": []any{}}, true)
	if e != nil {
		t.Fatal(e)
	}
	commerceRunTest(t, f, refund)
	rows, _, _, _ = f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 30)
	if len(rows) != 0 {
		t.Fatal("refund duplicated coupon")
	}
}

func TestPromotionConcurrentCouponAndLimitReservationsV209(t *testing.T) {
	for _, kind := range []string{"coupon", "daily-limit"} {
		t.Run(kind, func(t *testing.T) {
			f := newCatalogFixture(t)
			ctx := context.Background()
			p, content := catalogProductFixture(t, f)
			if kind == "coupon" {
				promotionSaveTestV209(t, f, promotionCouponTestV209())
			} else {
				content["purchaseLimits"] = map[string]any{"daily": 1}
				p = promotionPublishTestV209(t, f, p, content)
			}
			quotes := []store.CatalogObjectV2{promotionQuoteTestV209(t, f, p, "one", 1, "CART"), promotionQuoteTestV209(t, f, p, "two", 1, "CART")}
			var wg sync.WaitGroup
			results := make(chan error, 2)
			for i, q := range quotes {
				wg.Add(1)
				go func(i int, q store.CatalogObjectV2) {
					defer wg.Done()
					_, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, fmt.Sprintf("pay-%d", i), fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil)
					results <- e
				}(i, q)
			}
			wg.Wait()
			close(results)
			success := 0
			for e := range results {
				if e == nil {
					success++
				} else if kind == "coupon" {
					assertCatalogCode(t, e, "COUPON_ALREADY_USED")
				} else {
					assertCatalogCode(t, e, "PLAYER_PURCHASE_LIMIT")
				}
			}
			if success != 1 || catalogTestCount(t, f.s, "commerce_resources_v2") != 1 {
				t.Fatal("parallel spending", success)
			}
		})
	}
}

func TestPromotionKnownFailedPaymentReleasesCouponAndLimitButKeepsBagV209(t *testing.T) {
	f := newCatalogFixture(t)
	core := newCommerceTestCore()
	core.failOnce = "wallet.escrow.reserve"
	f.server.CommerceCore = core
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	content["purchaseLimits"] = map[string]any{"lifetime": 1, "daily": 1, "weekly": 1, "monthly": 1}
	p = promotionPublishTestV209(t, f, p, content)
	promotionSaveTestV209(t, f, promotionCouponTestV209())
	if _, e := f.s.CatalogCartChangeV2(ctx, f.buyer.ID, p.ID, "bag", 1, 1, false); e != nil {
		t.Fatal(e)
	}
	m := promotionPrepareTestV209(t, f, promotionQuoteTestV209(t, f, p, "q", 1, "CART"), "pay")
	op, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || op.State != "FAILED" {
		t.Fatal(op, e)
	}
	cart, _ := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if len(cart.Items) != 1 {
		t.Fatal(cart)
	}
	rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 30)
	if e != nil || len(rows) != 1 {
		t.Fatal("failed payment spent coupon", rows, e)
	}
	q := promotionQuoteTestV209(t, f, p, "retry-new-quote", 1, "DIRECT")
	m = promotionPrepareTestV209(t, f, q, "pay-after-failure")
	commerceRunTest(t, f, m)
	cart, _ = f.s.CatalogCartV2(ctx, f.buyer.ID)
	if len(cart.Items) != 1 {
		t.Fatal("direct purchase consumed unrelated bag", cart)
	}
}

func TestPromotionCreditDeliveryAndZeroPaymentRefundV209(t *testing.T) {
	f := newCatalogFixture(t)
	core := newCommerceTestCore()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	content["deliveryCredits"] = 123
	p = promotionPublishTestV209(t, f, p, content)
	c := promotionCouponTestV209()
	c["amountOff"] = "100.00"
	promotionSaveTestV209(t, f, c)
	q := promotionQuoteTestV209(t, f, p, "zero", 2, "CART")
	if q["totalAmount"] != "0.00" {
		t.Fatal(q)
	}
	d := commerceRunTest(t, f, promotionPrepareTestV209(t, f, q, "free-order"))
	plan := d.Body["mailboxPlan"].(map[string]any)
	var snapshot map[string]any
	if e := json.Unmarshal([]byte(plan["snapshotJson"].(string)), &snapshot); e != nil {
		t.Fatal(e)
	}
	if snapshot["creditAmount"] != float64(246) || core.calls["wallet.escrow.reserve"] != 0 || core.calls["mailbox.create"] != 1 {
		t.Fatal(snapshot, core.calls)
	}
	m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "free-cancel", d.Version, store.CatalogObjectV2{"reasonCode": "NO_LONGER_NEEDED", "description": "取消免费订单", "evidenceAssetIds": []any{}}, true)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.FundsState != "REFUNDED" || core.calls["wallet.escrow.refund"] != 0 {
		t.Fatal(d, core.calls)
	}
}

func TestPromotionCreditCapabilityAndCancellationSnapshotV209(t *testing.T) {
	f := newCatalogFixture(t)
	p, content := catalogProductFixture(t, f)
	content["deliveryCredits"] = 25
	p = promotionPublishTestV209(t, f, p, content)
	q := promotionQuoteTestV209(t, f, p, "credits-capability", 1, "CART")
	unsupported := false
	_, e := f.s.PrepareOrderV2(context.Background(), f.buyer.ID, "unsupported", fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true, CreditRewards: &unsupported}})
	assertCatalogCode(t, e, "CREDIT_DELIVERY_UNAVAILABLE")
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 0 {
		t.Fatal("unavailable credit delivery reserved funds")
	}
	snapshot := map[string]any{"schemaVersion": 1, "orderId": "order_credits", "recipientUuid": f.buyer.ServerUUID, "inventoryDomain": "survival", "allowedServerIds": []string{"amiya"}, "attachments": []any{}, "creditAmount": 25}
	raw, _ := json.Marshal(snapshot)
	request := mailCancellationRequestV2{Source: "deuterium-commerce", DeliveryID: "delivery_credits", OrderID: "order_credits", ExpectedSnapshotSHA256: store.Digest(raw), ReasonCode: "CUSTOMER_REFUND", SnapshotJSON: string(raw)}
	body, _ := json.Marshal(request)
	if _, v, e := parseMailCancellationV2(body); e != nil || v.CreditAmount != 25 {
		t.Fatal("credit-only cancellation cannot be recovered", v, e)
	}
}

func TestPromotionAdminFilterUsesAllRecordsAndCouponExpiryBlocksSubmissionV209(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	p, _ := catalogProductFixture(t, f)
	c := promotionCouponTestV209()
	active := promotionSaveTestV209(t, f, c)
	scheduled := promotionCouponTestV209()
	scheduled["startsAt"] = time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	scheduled["endsAt"] = time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	promotionSaveTestV209(t, f, scheduled)
	rows, _, more, e := f.s.CouponsFilteredV209(ctx, f.admin.ID, true, "", "", 1, "ACTIVE")
	if e != nil || more || len(rows) != 1 || rows[0].ID != active.ID {
		t.Fatal("filter only searched the loaded page", rows, e)
	}
	q := promotionQuoteTestV209(t, f, p, "expires-before-payment", 1, "DIRECT")
	expired := active.Body
	expired["endsAt"] = time.Now().Add(-time.Second).UTC().Format(time.RFC3339)
	if _, e = f.s.SaveCouponV209(ctx, f.admin.ID, active.ID, "expire-before-pay", active.Version, expired); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.PrepareOrderV2(ctx, f.buyer.ID, "expired-pay", fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil); e == nil {
		t.Fatal("paid an expired/changed coupon")
	}
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 0 {
		t.Fatal("expired coupon created financial work")
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func checkoutProductsV221(t *testing.T, f catalogFixture) (store.CatalogRecordV2, store.CatalogRecordV2) {
	t.Helper()
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	create := func(kind, shop string, body store.CatalogObjectV2) store.CatalogRecordV2 {
		d, e := f.s.CatalogCreateV2(ctx, f.admin.ID, kind, shop, store.ID("create_"), body)
		if e != nil {
			t.Fatal(e)
		}
		return d
	}
	shop := create("store", "", store.CatalogObjectV2{"name": "第二家测试商店", "intro": "隔离测试", "logoAssetId": nil, "coverAssetId": nil, "contactQq": "123456789", "serviceHours": "", "notice": ""})
	brand := create("brand", shop.ID, store.CatalogObjectV2{"name": "第二品牌", "logoAssetId": nil, "sortOrder": 0, "active": true})
	category := create("category", shop.ID, store.CatalogObjectV2{"name": "第二分类", "sortOrder": 0, "active": true})
	template, e := f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, shop.ID, "", store.ID("template_"), 0, store.CatalogObjectV2{"name": "第二店交付", "summary": "原样物品模板", "inventoryDomain": "survival", "allowedServerIds": []any{"amiya"}, "attachments": []any{map[string]any{"itemRef": "catalog:stone", "revision": 1, "quantity": 2, "payloadSha256": strings.Repeat("a", 64)}}, "active": true}, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	content = catalogObjectClone(content)
	content["title"], content["brandId"], content["categoryId"], content["deliveryTemplateRef"], content["price"] = "第二店商品", brand.ID, category.ID, template.ID, "20.00"
	second := create("product", shop.ID, content)
	second, e = f.s.CatalogActionV2(ctx, f.admin.ID, second.ID, "product", store.ID("publish_"), second.Version, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	return p, second
}

func checkoutQuoteV221(t *testing.T, f catalogFixture, products ...store.CatalogRecordV2) store.CatalogObjectV2 {
	t.Helper()
	ctx := context.Background()
	items := []any{}
	cart, e := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range products {
		items = append(items, map[string]any{"productId": p.ID, "quantity": 1, "expectedProductVersion": p.PublishedVersion})
		cart, e = f.s.CatalogCartChangeV2(ctx, f.buyer.ID, p.ID, store.ID("cart_"), cart.Version, 1, false)
		if e != nil {
			t.Fatal(e)
		}
	}
	q, e := f.s.CatalogQuoteV2(ctx, f.buyer.ID, store.ID("quote_"), store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "source": "CART", "items": items, "delivery": map[string]any{"method": "MAILBOX"}})
	if e != nil {
		t.Fatal(e)
	}
	return q
}

type checkoutCoreV221 struct {
	*commerceTestCore
	sequence      []string
	counts        map[string]int
	fail, unknown map[string]int
}

func newCheckoutCoreV221() *checkoutCoreV221 {
	return &checkoutCoreV221{commerceTestCore: newCommerceTestCore(), counts: map[string]int{}, fail: map[string]int{}, unknown: map[string]int{}}
}
func (c *checkoutCoreV221) Execute(ctx context.Context, actor, key, command string, payload map[string]any) (CommerceCoreResultV2, error) {
	c.counts[command]++
	c.sequence = append(c.sequence, command)
	if c.fail[command] == c.counts[command] {
		c.failOnce = command
	}
	if c.unknown[command] == c.counts[command] {
		c.unknownOnce = command
	}
	return c.commerceTestCore.Execute(ctx, actor, key, command, payload)
}

func checkoutOrdersV221(t *testing.T, f catalogFixture, op store.CommerceOperationV2) []store.CommerceRecordV2 {
	t.Helper()
	out := []store.CommerceRecordV2{}
	for _, id := range store.CheckoutOrderIDsV221(op) {
		out = append(out, commerceRecordTest(t, f, id))
	}
	return out
}

func TestCheckoutV221TwoStoresCouponsMailRefundAndReplay(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	core := newCheckoutCoreV221()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	a, b := checkoutProductsV221(t, f)
	global := promotionSaveTestV209(t, f, promotionCouponTestV209())
	local := promotionCouponTestV209()
	local["amountOff"], local["storeIds"] = "2.00", []any{a.StoreID}
	own := promotionSaveTestV209(t, f, local)
	q := checkoutQuoteV221(t, f, a, b)
	if q["totalAmount"] != "26.10" || q["couponDiscount"] != "5.00" {
		t.Fatal(q)
	}
	const key = "same-cross-store-payment"
	var wg sync.WaitGroup
	mutations := make(chan store.CommerceMutationV2, 6)
	errs := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, key, fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil)
			if e != nil {
				errs <- e
			} else {
				mutations <- m
			}
		}()
	}
	wg.Wait()
	close(errs)
	close(mutations)
	for e := range errs {
		t.Fatal(e)
	}
	var first store.CommerceMutationV2
	for m := range mutations {
		if first.OperationID != "" && first.OperationID != m.OperationID {
			t.Fatal("duplicate group")
		}
		first = m
	}
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 2 || catalogTestCount(t, f.s, "commerce_operations_v2") != 1 {
		t.Fatal("not exactly two orders and one operation")
	}
	_, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "different-key", fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil)
	assertCatalogCode(t, e, "QUOTE_ALREADY_USED")
	op, e := f.server.RunCommerceOperationV2(ctx, first.OperationID, true)
	if e != nil || op.State != "COMPLETED" {
		t.Fatal(op, e)
	}
	if strings.Join(core.sequence, ",") != "wallet.escrow.reserve,wallet.escrow.reserve,mailbox.create,mailbox.create" {
		t.Fatal(core.sequence)
	}
	orders := checkoutOrdersV221(t, f, op)
	byStore := map[string]store.CommerceRecordV2{}
	deliveries := map[string]bool{}
	for _, d := range orders {
		byStore[d.StoreID] = d
		if d.PendingOperationID != "" || d.FundsState != "HELD" || d.State != "AWAITING_CLAIM" {
			t.Fatal(d)
		}
		plan := d.Body["mailboxPlan"].(map[string]any)
		deliveries[fmt.Sprint(plan["deliveryId"])] = true
		if plan["orderId"] != d.ID {
			t.Fatal("mail mapped to wrong order")
		}
		if len(d.Body["items"].([]any)) != 1 {
			t.Fatal("mixed order snapshot")
		}
	}
	if len(deliveries) != 2 || byStore[a.StoreID].Amount != "9.10" || byStore[b.StoreID].Amount != "17.00" {
		t.Fatal(orders)
	}
	for _, check := range []struct{ id, order string }{{global.ID, byStore[b.StoreID].ID}, {own.ID, byStore[a.StoreID].ID}} {
		var actual string
		if e = f.s.DB.QueryRow("SELECT resource_id FROM promotion_redemptions_v209 WHERE coupon_id=?", check.id).Scan(&actual); e != nil || actual != check.order {
			t.Fatal(actual, e)
		}
	}
	cart, e := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if e != nil || len(cart.Items) != 0 {
		t.Fatal("paid cart not cleared", cart, e)
	}
	ref := byStore[a.StoreID]
	m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, ref.ID, "ORDER", "refund-store-a", ref.Version, store.CatalogObjectV2{"reasonCode": "OTHER", "description": "隔离测试退款", "evidenceAssetIds": []any{}}, true)
	if e != nil {
		t.Fatal(e)
	}
	ref = commerceRunTest(t, f, m)
	if ref.RefundedAmount != "9.10" || ref.FundsState != "REFUNDED" || commerceRecordTest(t, f, byStore[b.StoreID].ID).FundsState != "HELD" {
		t.Fatal("refund crossed stores")
	}
	if catalogTestCount(t, f.s, "promotion_redemptions_v209") != 1 {
		t.Fatal("wrong coupon returned")
	}
	if _, _, e = f.s.CommerceVisibleOperationV2(ctx, f.other.ID, op.ID, "", ""); e == nil {
		t.Fatal("foreign group exposed")
	}
	_, e = f.server.RunCommerceOperationV2(ctx, first.OperationID, true)
	if e != nil || core.calls["wallet.escrow.reserve"] != 2 || core.calls["mailbox.create"] != 2 {
		t.Fatal("replay redispatched", e)
	}
}

func TestCheckoutV221UnknownReserveAndRollbackRecovery(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprint(rollback), func(t *testing.T) {
			f := newCatalogFixture(t)
			ctx := context.Background()
			core := newCheckoutCoreV221()
			f.server.CommerceCore = core
			a, b := checkoutProductsV221(t, f)
			promotionSaveTestV209(t, f, promotionCouponTestV209())
			q := checkoutQuoteV221(t, f, a, b)
			m := promotionPrepareTestV209(t, f, q, "pay")
			if rollback {
				core.fail["wallet.escrow.reserve"] = 2
				core.unknown["wallet.escrow.refund"] = 1
			} else {
				core.unknown["wallet.escrow.reserve"] = 1
			}
			op, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
			if e != nil || op.State != "UNKNOWN" || core.calls["mailbox.create"] != 0 {
				t.Fatal(op, e, core.sequence)
			}
			// A fresh Server reads the durable group/steps rather than client memory.
			server := New(f.s, f.server.Config)
			defer server.Close()
			server.CommerceCore = core
			op, e = server.RunCommerceOperationV2(ctx, m.OperationID, true)
			if e != nil {
				t.Fatal(e)
			}
			want := "COMPLETED"
			if rollback {
				want = "FAILED"
			}
			if op.State != want {
				t.Fatal(op)
			}
			if core.calls["wallet.escrow.reserve"] != 2 || core.queryCount != 1 {
				t.Fatal(core.calls, core.queryCount)
			}
			cart, e := f.s.CatalogCartV2(ctx, f.buyer.ID)
			if e != nil {
				t.Fatal(e)
			}
			if rollback {
				if core.calls["mailbox.create"] != 0 || core.calls["wallet.escrow.refund"] != 1 || len(cart.Items) != 2 {
					t.Fatal(core.sequence, cart)
				}
				for _, l := range core.ledger {
					if l.reserved != l.refunded {
						t.Fatal("money not returned", l)
					}
				}
				for _, d := range checkoutOrdersV221(t, f, op) {
					if d.PendingOperationID != "" || (d.FundsState != "REFUNDED" && d.FundsState != "UNPAID") {
						t.Fatal(d)
					}
				}
				if catalogTestCount(t, f.s, "promotion_redemptions_v209") != 0 {
					t.Fatal("coupon not released")
				}
				for _, p := range []store.CatalogRecordV2{a, b} {
					fresh, e := f.s.CatalogGetRecordV2(ctx, f.buyer.ID, p.ID, "product", false)
					if e != nil || fresh.Stock == nil || *fresh.Stock != 10 {
						t.Fatal("stock not restored", fresh, e)
					}
				}
			} else if len(cart.Items) != 0 || core.calls["mailbox.create"] != 2 {
				t.Fatal(cart, core.sequence)
			}
		})
	}
}

func TestCheckoutV221StockAndQuoteChangesRollbackWholePreparation(t *testing.T) {
	for _, mode := range []string{"stock", "price", "expired"} {
		t.Run(mode, func(t *testing.T) {
			f := newCatalogFixture(t)
			ctx := context.Background()
			a, b := checkoutProductsV221(t, f)
			q := checkoutQuoteV221(t, f, a, b)
			switch mode {
			case "stock":
				_, _ = f.s.DB.Exec("UPDATE catalog_records_v2 SET available_stock=0 WHERE resource_id=?", b.ID)
			case "price":
				body := catalogObjectClone(b.Published)
				body["price"] = "21.00"
				promotionPublishTestV209(t, f, b, body)
			case "expired":
				_, _ = f.s.DB.Exec("UPDATE catalog_quotes_v2 SET expires_at=? WHERE quote_id=?", time.Now().Add(-time.Hour), q["quoteId"])
			}
			_, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "pay", fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, nil)
			if e == nil {
				t.Fatal("invalid quote accepted")
			}
			for _, table := range []string{"commerce_resources_v2", "commerce_operations_v2", "commerce_stock_holds_v2", "promotion_redemptions_v209"} {
				if catalogTestCount(t, f.s, table) != 0 {
					t.Fatal("partial preparation", table)
				}
			}
		})
	}
}

func TestCheckoutV221MailFailureIndependentRetryAndHTTPGroup(t *testing.T) {
	f := newCatalogFixture(t)
	f.server.Config.Nodes = []config.Node{{ID: "amiya", InventoryDomain: "survival", ClaimEnabled: true}}
	ctx := context.Background()
	core := newCheckoutCoreV221()
	core.fail["mailbox.create"] = 1
	f.server.CommerceCore = core
	a, b := checkoutProductsV221(t, f)
	q := checkoutQuoteV221(t, f, a, b)
	mux := http.NewServeMux()
	f.server.registerCommerceV2(mux)
	raw, _ := json.Marshal(map[string]any{"clientRequestId": "group-http", "quoteId": q["quoteId"], "expectedQuoteVersion": 1})
	r := httptest.NewRequest("POST", "/api/v1/store/orders", strings.NewReader(string(raw)))
	r.Header.Set("Authorization", "Bearer "+f.buyerToken)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Orders    []map[string]any `json:"orders"`
			Operation map[string]any   `json:"operation"`
		} `json:"data"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &response); e != nil {
		t.Fatal(e)
	}
	if len(response.Data.Orders) != 2 || response.Data.Operation["status"] != "COMPLETED" || response.Data.Operation["amount"] != "31.10" {
		t.Fatal(w.Body.String())
	}
	if path := os.Getenv("DEUTERIUM_CHECKOUT_QA_FILE"); path != "" {
		var envelope map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &envelope); e != nil {
			t.Fatal(e)
		}
		samples := map[string]any{"CheckoutQuote": q, "OrderCreationResult": envelope["data"], "OperationLookup": response.Data.Operation}
		raw, e := json.MarshalIndent(samples, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	var failed store.CommerceRecordV2
	for _, row := range response.Data.Orders {
		d := commerceRecordTest(t, f, fmt.Sprint(row["orderId"]))
		if d.State == "PAYMENT_PROCESSING" {
			failed = d
		}
		if d.FundsState != "HELD" || d.PendingOperationID != "" {
			t.Fatal(d)
		}
	}
	if failed.ID == "" {
		t.Fatal("missing failed mail order")
	}
	cart, e := f.s.CatalogCartV2(ctx, f.buyer.ID)
	if e != nil || len(cart.Items) != 0 {
		t.Fatal(cart, e)
	}
	m, e := f.s.PrepareDeliveryRetryV2(ctx, f.admin.ID, failed.ID, "retry-one-mail", failed.Version, "隔离重试", true)
	if e != nil {
		t.Fatal(e)
	}
	d := commerceRunTest(t, f, m)
	if d.State != "AWAITING_CLAIM" || core.calls["wallet.escrow.reserve"] != 2 || core.calls["mailbox.create"] != 3 {
		t.Fatal(d, core.calls)
	}
}

func TestCheckoutV221FreeOrdersAndFirstReserveFailure(t *testing.T) {
	for _, mode := range []string{"all-free", "one-free", "first-fails"} {
		t.Run(mode, func(t *testing.T) {
			f := newCatalogFixture(t)
			ctx := context.Background()
			core := newCheckoutCoreV221()
			f.server.CommerceCore = core
			a, b := checkoutProductsV221(t, f)
			first, second := a, b
			if first.StoreID > second.StoreID {
				first, second = second, first
			}
			if mode != "first-fails" {
				coupon := promotionCouponTestV209()
				coupon["amountOff"], coupon["storeIds"] = "50.00", []any{first.StoreID}
				promotionSaveTestV209(t, f, coupon)
				if mode == "all-free" {
					coupon = catalogObjectClone(coupon)
					coupon["storeIds"] = []any{second.StoreID}
					promotionSaveTestV209(t, f, coupon)
				}
			} else {
				core.fail["wallet.escrow.reserve"] = 1
			}
			q := checkoutQuoteV221(t, f, a, b)
			m := promotionPrepareTestV209(t, f, q, "pay")
			op, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
			if e != nil {
				t.Fatal(e)
			}
			if mode == "first-fails" {
				if op.State != "FAILED" || core.calls["mailbox.create"] != 0 || core.calls["wallet.escrow.refund"] != 0 {
					t.Fatal(op, core.calls)
				}
				return
			}
			wantReserves := 1
			if mode == "all-free" {
				wantReserves = 0
			}
			if op.State != "COMPLETED" || core.calls["mailbox.create"] != 2 || core.calls["wallet.escrow.reserve"] != wantReserves {
				t.Fatal(op, core.calls)
			}
			for _, d := range checkoutOrdersV221(t, f, op) {
				if d.FundsState != "HELD" || d.PendingOperationID != "" {
					t.Fatal(d)
				}
			}
			cart, e := f.s.CatalogCartV2(ctx, f.buyer.ID)
			if e != nil || len(cart.Items) != 0 {
				t.Fatal(cart, e)
			}
		})
	}
}

func TestCheckoutV221FailedCompensationRemainsIndividuallyRefundable(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	core := newCheckoutCoreV221()
	f.server.CommerceCore = core
	a, b := checkoutProductsV221(t, f)
	q := checkoutQuoteV221(t, f, a, b)
	m := promotionPrepareTestV209(t, f, q, "pay")
	core.fail["wallet.escrow.reserve"], core.fail["wallet.escrow.refund"] = 2, 1
	op, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || op.State != "FAILED" || op.ErrorCode != "CHECKOUT_REFUND_REQUIRES_ATTENTION" {
		t.Fatal(op, e)
	}
	var held store.CommerceRecordV2
	for _, d := range checkoutOrdersV221(t, f, op) {
		if d.FundsState == "HELD" {
			held = d
		}
		if d.PendingOperationID != "" {
			t.Fatal("stuck operation", d)
		}
	}
	if held.ID == "" || core.calls["mailbox.create"] != 0 {
		t.Fatal(held, core.calls)
	}
	refund, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, held.ID, "ORDER", "recover-refund", held.Version, store.CatalogObjectV2{"reasonCode": "OTHER", "description": "退回未完成合并付款", "evidenceAssetIds": []any{}}, true)
	if e != nil {
		t.Fatal(e)
	}
	d := commerceRunTest(t, f, refund)
	if d.FundsState != "REFUNDED" || d.RefundedAmount != d.Amount {
		t.Fatal(d)
	}
	for _, l := range core.ledger {
		if l.refunded != l.reserved {
			t.Fatal("remaining money", l)
		}
	}
}

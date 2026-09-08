//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
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

const commerceTestPool = "20000000-0000-0000-0000-000000000001"
const commerceTestRevenue = "20000000-0000-0000-0000-000000000002"

type commerceTestLedger struct {
	payer, payee, kind, ref, business string
	reserved, settled, refunded       int64
}
type commerceTestCore struct {
	mu                               sync.Mutex
	ledger                           map[string]*commerceTestLedger
	result                           map[string]CommerceCoreResultV2
	keys                             map[string]string
	calls                            map[string]int
	unknownOnce, failOnce, forgeOnce string
	mailMode                         string
	queryCount                       int
}

func newCommerceTestCore() *commerceTestCore {
	return &commerceTestCore{ledger: map[string]*commerceTestLedger{}, result: map[string]CommerceCoreResultV2{}, keys: map[string]string{}, calls: map[string]int{}}
}
func (c *commerceTestCore) Available(command string) bool { return true }
func commerceTestCents(v any) int64 {
	r, ok := new(big.Rat).SetString(fmt.Sprint(v))
	if !ok {
		panic("invalid test amount")
	}
	r.Mul(r, big.NewRat(100, 1))
	if !r.IsInt() {
		panic("fractional cent")
	}
	return r.Num().Int64()
}
func commerceTestMoney(n int64) string { return fmt.Sprintf("%d.%02d", n/100, n%100) }
func (c *commerceTestCore) Query(_ context.Context, _, id string) (CommerceCoreResultV2, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queryCount++
	r, ok := c.result[id]
	if !ok {
		return CommerceCoreResultV2{OperationID: id, Status: "UNKNOWN"}, nil
	}
	return r, nil
}
func (c *commerceTestCore) Execute(_ context.Context, actor, key, command string, p map[string]any) (CommerceCoreResultV2, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[command]++
	if id := c.keys[actor+":"+key]; id != "" {
		return c.result[id], nil
	}
	id := fmt.Sprintf("core_%03d", len(c.result)+1)
	c.keys[actor+":"+key] = id
	r := CommerceCoreResultV2{OperationID: id, Status: "COMPLETED"}
	if c.failOnce == command {
		c.failOnce = ""
		r.Status = "FAILED"
		r.ErrorCode = "ECONOMY_REJECTED"
		c.result[id] = r
		return r, nil
	}
	if strings.HasPrefix(command, "wallet.") {
		ref := fmt.Sprint(p["escrowRef"])
		l := c.ledger[ref]
		amount := int64(0)
		from, to := "", ""
		if command == "wallet.escrow.reserve" {
			if l != nil {
				panic("test double duplicate reserve")
			}
			payee, _ := p["payeeUuid"].(string)
			if p["businessType"] == "OFFICIAL_STORE" {
				payee = commerceTestRevenue
			}
			l = &commerceTestLedger{payer: fmt.Sprint(p["payerUuid"]), payee: payee, kind: fmt.Sprint(p["businessType"]), ref: ref, business: fmt.Sprint(p["businessRef"]), reserved: commerceTestCents(p["amount"])}
			c.ledger[ref] = l
			amount = l.reserved
			from = l.payer
			to = commerceTestPool
		} else {
			if l == nil {
				panic("missing escrow")
			}
			switch command {
			case "wallet.escrow.bind":
				l.payee = fmt.Sprint(p["payeeUuid"])
			case "wallet.escrow.settle":
				amount = commerceTestCents(p["amount"])
				l.settled += amount
				from = commerceTestPool
				to = l.payee
			case "wallet.escrow.refund":
				amount = commerceTestCents(p["amount"])
				l.refunded += amount
				from = commerceTestPool
				to = l.payer
			default:
				panic(command)
			}
		}
		if l.settled+l.refunded > l.reserved {
			panic("double spending")
		}
		r.Data = map[string]any{"operationId": id, "businessRef": l.business, "escrowRef": l.ref, "businessType": l.kind, "payerUuid": l.payer, "payeeUuid": l.payee, "fromUuid": from, "toUuid": to, "amount": commerceTestMoney(amount), "currency": "CREDIT", "reservedAmount": commerceTestMoney(l.reserved), "settledAmount": commerceTestMoney(l.settled), "refundedAmount": commerceTestMoney(l.refunded), "heldAmount": commerceTestMoney(l.reserved - l.settled - l.refunded), "escrowStatus": "HELD", "status": "COMPLETED", "committedAt": time.Now().UTC().Format(time.RFC3339Nano)}
	} else {
		var plan map[string]any
		if e := json.Unmarshal([]byte(fmt.Sprint(p["snapshotJson"])), &plan); e != nil {
			panic(e)
		}
		hash := store.Digest([]byte(fmt.Sprint(p["snapshotJson"])))
		value := map[string]any{"source": "deuterium-commerce", "deliveryId": p["deliveryId"], "orderId": plan["orderId"], "recipientUuid": plan["recipientUuid"], "snapshotSha256": hash, "inventoryDomain": plan["inventoryDomain"], "allowedServerIds": plan["allowedServerIds"], "mailId": "mail_test", "revision": 1, "status": "CREATED"}
		if command == "mailbox.revoke" {
			if c.mailMode == "notfound" {
				r.Data = map[string]any{"code": "NOT_FOUND"}
				c.result[id] = r
				return r, nil
			}
			delete(value, "mailId")
			delete(value, "revision")
			delete(value, "status")
			value["operationId"] = id
			value["cancelledAt"] = time.Now().Unix()
			value["proofKind"] = "CANCELLED_BEFORE_CREATE"
			if c.mailMode == "wrong-tombstone" {
				value["recipientUuid"] = commerceTestRevenue
			}
		}
		r.Data = map[string]any{"code": "OK", "value": value}
	}
	c.result[id] = r
	if c.unknownOnce == command {
		c.unknownOnce = ""
		return CommerceCoreResultV2{OperationID: id, Status: "UNKNOWN", ErrorCode: "RESULT_UNKNOWN"}, nil
	}
	if c.forgeOnce == command {
		c.forgeOnce = ""
		raw, _ := json.Marshal(r.Data)
		var bad map[string]any
		_ = json.Unmarshal(raw, &bad)
		bad["amount"] = "0.00"
		return CommerceCoreResultV2{OperationID: id, Status: "COMPLETED", Data: bad}, nil
	}
	return r, nil
}
func commerceRunTest(t *testing.T, f catalogFixture, m store.CommerceMutationV2) store.CommerceRecordV2 {
	t.Helper()
	if m.OperationID != "" {
		o, e := f.server.RunCommerceOperationV2(context.Background(), m.OperationID, true)
		if e != nil || o.State != "COMPLETED" {
			t.Fatalf("operation %s: state=%s code=%s error=%v", m.OperationID, o.State, o.ErrorCode, e)
		}
	}
	return commerceRecordTest(t, f, m.ResourceID)
}
func commerceRecordTest(t *testing.T, f catalogFixture, id string) store.CommerceRecordV2 {
	t.Helper()
	d, e := f.s.CommerceRecordV2(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func commerceMarketTest(t *testing.T, f catalogFixture, key string, stock int) (store.CatalogRecordV2, store.CatalogObjectV2) {
	t.Helper()
	ctx := context.Background()
	asset := catalogTestAsset(t, f.s, f.admin.ID, "MARKET_PHOTO")
	in := catalogListingInput(asset)
	in["stock"] = stock
	l, e := f.s.CatalogCreateV2(ctx, f.admin.ID, "listing", "", key, in)
	if e != nil {
		t.Fatal(e)
	}
	q, e := f.s.CatalogQuoteV2(ctx, f.buyer.ID, key+"-quote", store.CatalogObjectV2{"channel": "PLAYER_MARKET", "items": []any{map[string]any{"productId": l.ID, "quantity": 1, "expectedProductVersion": l.Version}}, "delivery": map[string]any{"method": "PICKUP", "location": "主城仓库", "projectName": ""}})
	if e != nil {
		t.Fatal(e)
	}
	return l, q
}
func commerceOrderTest(t *testing.T, f catalogFixture, key string) (store.CommerceRecordV2, store.CatalogRecordV2) {
	t.Helper()
	l, q := commerceMarketTest(t, f, key, 1)
	m, e := f.s.PrepareOrderV2(context.Background(), f.buyer.ID, key+"-pay", fmt.Sprint(q["quoteId"]), 1, "PLAYER_MARKET", true, nil)
	if e != nil {
		t.Fatal(e)
	}
	return commerceRunTest(t, f, m), l
}
func commerceCommissionTest(t *testing.T, f catalogFixture, key string) store.CommerceRecordV2 {
	t.Helper()
	a := catalogTestAsset(t, f.s, f.buyer.ID, "COMMISSION_COVER")
	m, e := f.s.PrepareCommissionV2(context.Background(), f.buyer.ID, key, store.CatalogObjectV2{"title": "隔离委托", "description": "仅隔离数据库中的资金集成验证。", "location": "主城东门", "urgency": "NORMAL", "reward": "20.00", "workHours": 24, "coverAssetId": a}, true)
	if e != nil {
		t.Fatal(e)
	}
	return commerceRunTest(t, f, m)
}
func commerceRefundInput(key string, v int64) store.CatalogObjectV2 {
	return store.CatalogObjectV2{"clientRequestId": key, "expectedVersion": v, "reasonCode": "NOT_AS_DESCRIBED", "description": "与约定交付内容不符", "evidenceAssetIds": []any{}}
}

func TestCommerceMoneyUnknownOriginalKeyAndProofRecovery(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	l, q := commerceMarketTest(t, f, "unknown", 1)
	_, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "offline", fmt.Sprint(q["quoteId"]), 1, "PLAYER_MARKET", false, nil)
	assertCatalogCode(t, e, "CAPABILITY_UNAVAILABLE")
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 0 {
		t.Fatal("queued offline payment")
	}
	m, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "pay", fmt.Sprint(q["quoteId"]), 1, "PLAYER_MARKET", true, nil)
	if e != nil {
		t.Fatal(e)
	}
	c.unknownOnce = "wallet.escrow.reserve"
	o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "UNKNOWN" {
		t.Fatal(o, e)
	}
	same, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "pay", fmt.Sprint(q["quoteId"]), 1, "PLAYER_MARKET", false, nil)
	if e != nil || !same.Replayed || same.OperationID != m.OperationID {
		t.Fatal(same, e)
	}
	d := commerceRecordTest(t, f, m.ResourceID)
	if d.FundsState != "UNKNOWN" || d.PendingOperationID == "" {
		t.Fatal(d)
	}
	_, e = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund-unknown", d.Version, commerceRefundInput("refund-unknown", d.Version), true)
	assertCatalogCode(t, e, "OPERATION_IN_PROGRESS")
	o, e = f.server.RunCommerceOperationV2(ctx, m.OperationID, false)
	if e != nil || o.State != "COMPLETED" || c.calls["wallet.escrow.reserve"] != 1 || c.queryCount != 1 {
		t.Fatal(o, e, c.calls)
	}
	d = commerceRecordTest(t, f, d.ID)
	if d.State != "AWAITING_SHIPMENT" || d.FundsState != "HELD" {
		t.Fatal(d)
	}
	snap, e := f.s.CommerceSnapshotV2(ctx, f.buyer.ID, d.ID, false)
	if e != nil || snap["sha256"] != d.SnapshotSHA256 {
		t.Fatal(snap, e)
	}
	_, e = f.s.CommerceViewV2(ctx, f.other.ID, d.ID, false)
	assertCatalogCode(t, e, "NOT_FOUND")
	_, _, e = f.s.CommerceVisibleOperationV2(ctx, f.other.ID, m.OperationID, "", "")
	assertCatalogCode(t, e, "NOT_FOUND")
	updated, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, l.ID, "listing", true)
	if e != nil || *updated.Stock != 0 {
		t.Fatal(updated, e)
	}
	// A generic query proof with amount zero cannot stand in for a reserve receipt.
	_, q2 := commerceMarketTest(t, f, "forged", 1)
	m2, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "pay2", fmt.Sprint(q2["quoteId"]), 1, "PLAYER_MARKET", true, nil)
	if e != nil {
		t.Fatal(e)
	}
	c.forgeOnce = "wallet.escrow.reserve"
	o, e = f.server.RunCommerceOperationV2(ctx, m2.OperationID, true)
	if e != nil || o.State != "UNKNOWN" || o.ErrorCode != "PAYMENT_PROOF_MISMATCH" {
		t.Fatal(o, e)
	}
	o, e = f.server.RunCommerceOperationV2(ctx, m2.OperationID, false)
	if e != nil || o.State != "COMPLETED" {
		t.Fatal(o, e)
	}
}
func TestCommerceFailedReserveRestoresStockAndImmediateRefund(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	l, q := commerceMarketTest(t, f, "failed", 1)
	m, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "failed-pay", fmt.Sprint(q["quoteId"]), 1, "PLAYER_MARKET", true, nil)
	if e != nil {
		t.Fatal(e)
	}
	c.failOnce = "wallet.escrow.reserve"
	o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "FAILED" {
		t.Fatal(o, e)
	}
	d := commerceRecordTest(t, f, m.ResourceID)
	if d.FundsState != "UNPAID" {
		t.Fatal(d)
	}
	p, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, l.ID, "listing", true)
	if e != nil || *p.Stock != 1 {
		t.Fatal(p, e)
	}
	d, l = commerceOrderTest(t, f, "refund")
	m, e = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund-now", d.Version, commerceRefundInput("refund-now", d.Version), true)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.State != "REFUNDED" || d.RefundedAmount != "12.30" || d.RefundAttempts != 1 {
		t.Fatal(d)
	}
	p, e = f.s.CatalogGetRecordV2(ctx, f.admin.ID, l.ID, "listing", true)
	if e != nil || *p.Stock != 1 {
		t.Fatal(p, e)
	}
	if c.calls["wallet.escrow.refund"] != 1 {
		t.Fatal(c.calls)
	}
}
func TestCommerceCommissionConcurrentAcceptDeadlineAndSettlement(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	d := commerceCommissionTest(t, f, "commission")
	var wg sync.WaitGroup
	type outcome struct {
		m store.CommerceMutationV2
		e error
	}
	ch := make(chan outcome, 2)
	for _, u := range []store.User{f.admin, f.other} {
		wg.Add(1)
		go func(u store.User) {
			defer wg.Done()
			m, e := f.s.PrepareCommissionAcceptV2(ctx, u.ID, d.ID, "accept-"+u.ID, d.Version, true)
			ch <- outcome{m, e}
		}(u)
	}
	wg.Wait()
	close(ch)
	wins := 0
	var accepted store.CommerceMutationV2
	for r := range ch {
		if r.e == nil {
			wins++
			accepted = r.m
		} else {
			assertCatalogCode(t, r.e, "STATE_VERSION_CONFLICT")
		}
	}
	if wins != 1 {
		t.Fatal("acceptors", wins)
	}
	d = commerceRunTest(t, f, accepted)
	if d.State != "ACTIVE" || d.PayeeUUID == "" || d.Deadline == nil {
		t.Fatal(d)
	}
	beforeSnapshot := d.SnapshotID
	input := store.CatalogObjectV2{"clientRequestId": "complete", "expectedVersion": d.Version, "description": "按清单全部完成", "evidenceAssetIds": []any{}}
	m, e := f.s.CommerceFulfillmentV2(ctx, d.PayeeID, d.ID, "COMMISSION", "complete", "complete", d.Version, input)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	if d.DeadlineKind != "ACCEPTANCE" || d.Deadline == nil || time.Until(*d.Deadline) < 71*time.Hour || d.SnapshotID != beforeSnapshot {
		t.Fatal(d)
	}
	_, e = f.s.PrepareCommerceSettlementV2(ctx, f.buyer.ID, d.ID, "COMMISSION", "too-early", d.Version, true, true, time.Now())
	assertCatalogCode(t, e, "DEADLINE_NOT_REACHED")
	m, e = f.s.PrepareCommerceSettlementV2(ctx, f.buyer.ID, d.ID, "COMMISSION", "due", d.Version, true, true, d.Deadline.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.State != "CONFIRMED" || !d.Automatic || d.SettledAmount != "20.00" || d.Deadline != nil {
		t.Fatal(d)
	}
	if c.calls["wallet.escrow.bind"] != 1 || c.calls["wallet.escrow.settle"] != 1 {
		t.Fatal(c.calls)
	}
}
func TestCommerceRefundNestedVersionPauseAndOnceOnly(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	d, _ := commerceOrderTest(t, f, "withdraw")
	m, e := f.s.CommerceFulfillmentV2(ctx, f.admin.ID, d.ID, "ORDER", "ship", "ship", d.Version, store.CatalogObjectV2{"clientRequestId": "ship", "expectedVersion": d.Version})
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	m, e = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, commerceRefundInput("refund", d.Version), false)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	if d.Deadline != nil || d.PausedRemaining == nil || *d.PausedRemaining < 259190 {
		t.Fatal(d)
	}
	_, e = f.s.WithdrawCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", d.RefundID, "wrong-version", d.Version)
	assertCatalogCode(t, e, "STATE_VERSION_CONFLICT")
	m, e = f.s.WithdrawCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", d.RefundID, "withdraw", 1)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	if d.Deadline == nil || d.PausedRemaining != nil {
		t.Fatal(d)
	}
	_, e = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "second-refund", d.Version, commerceRefundInput("second-refund", d.Version), true)
	assertCatalogCode(t, e, "REFUND_ATTEMPT_USED")
	m, e = f.s.PrepareCommerceSettlementV2(ctx, f.buyer.ID, d.ID, "ORDER", "confirm", d.Version, true, false, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.FundsState != "SETTLED" {
		t.Fatal(d)
	}
}
func TestCommerceInterventionPartialRefundFailedRemainderRecovery(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	d, _ := commerceOrderTest(t, f, "case")
	m, e := f.s.CommerceFulfillmentV2(ctx, f.admin.ID, d.ID, "ORDER", "ship", "ship", d.Version, store.CatalogObjectV2{"clientRequestId": "ship", "expectedVersion": d.Version})
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	m, e = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, commerceRefundInput("refund", d.Version), true)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	m, e = f.s.ResolveCommerceRefundV2(ctx, f.admin.ID, d.ID, "ORDER", d.RefundID, "reject", 1, "REJECT", "已经依照约定交付", false)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, m.ResourceID)
	input := store.CatalogObjectV2{"clientRequestId": "case", "expectedVersion": d.Version, "reasonCode": "REFUND_DISAGREEMENT", "description": "申请核对已经交付物品与原始约定差异。", "desiredResolution": "PARTIAL_REFUND", "requestedRefundAmount": "3.00", "evidenceAssetIds": []any{}}
	m, e = f.s.CreateCommerceInterventionV2(ctx, f.buyer.ID, d.ID, "ORDER", "case", d.Version, input)
	if e != nil {
		t.Fatal(e)
	}
	caseID := m.ResourceID
	d = commerceRecordTest(t, f, d.ID)
	if d.FundsState != "INTERVENTION_HOLD" || d.Deadline != nil || d.InterventionCaseID != caseID {
		t.Fatal(d)
	}
	_, e = f.s.CommerceCaseActionV2(ctx, f.admin.ID, caseID, "assign", "assign", 1, store.CatalogObjectV2{"clientRequestId": "assign", "expectedVersion": 1}, true)
	if e != nil {
		t.Fatal(e)
	}
	resolution := store.CatalogObjectV2{"clientRequestId": "resolve", "expectedVersion": 2, "decision": "PARTIAL_REFUND", "reason": "证据已核对，按原双方分别结算。", "refundAmount": "3.00"}
	manual := store.CatalogObjectV2{"clientRequestId": "manual", "expectedVersion": 2, "decision": "REQUIRE_MANUAL_RECOVERY", "reason": "仍有冻结款时不应关闭案件转成人工追回。"}
	_, e = f.s.CommerceCaseActionV2(ctx, f.admin.ID, caseID, "manual", "resolve", 2, manual, true)
	assertCatalogCode(t, e, "HELD_FUNDS_REQUIRE_DISPOSITION")
	m, e = f.s.CommerceCaseActionV2(ctx, f.admin.ID, caseID, "resolve", "resolve", 2, resolution, true)
	if e != nil {
		t.Fatal(e)
	}
	c.failOnce = "wallet.escrow.settle"
	o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "FAILED" {
		t.Fatal(o, e)
	}
	d = commerceRecordTest(t, f, d.ID)
	if d.RefundedAmount != "3.00" || d.SettledAmount != "0.00" || d.FundsState != "INTERVENTION_HOLD" {
		t.Fatal(d)
	}
	v, e := f.s.CommerceCaseViewV2(ctx, f.admin.ID, caseID, true)
	if e != nil {
		t.Fatal(e)
	}
	version := int64(commerceTestCents(v["version"]) / 100)
	resolution["expectedVersion"] = version
	resolution["clientRequestId"] = "resolve-retry"
	m, e = f.s.CommerceCaseActionV2(ctx, f.admin.ID, caseID, "resolve-retry", "resolve", version, resolution, true)
	if e != nil {
		t.Fatal(e)
	}
	o, e = f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "COMPLETED" {
		t.Fatal(o, e)
	}
	d = commerceRecordTest(t, f, d.ID)
	if d.RefundedAmount != "3.00" || d.SettledAmount != "9.30" || d.FundsState != "SETTLED" {
		t.Fatal(d)
	}
	if c.calls["wallet.escrow.refund"] != 1 || c.calls["wallet.escrow.settle"] != 2 {
		t.Fatal(c.calls)
	}
	v, e = f.s.CommerceCaseViewV2(ctx, f.buyer.ID, caseID, false)
	if e != nil || v["status"] != "RESOLVED" {
		t.Fatal(v, e)
	}
}

func commerceOfficialTest(t *testing.T, f catalogFixture, p store.CatalogRecordV2, key string) store.CommerceMutationV2 {
	t.Helper()
	q, e := f.s.CatalogQuoteV2(context.Background(), f.buyer.ID, key+"-quote", store.CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": p.ID, "quantity": 1, "expectedProductVersion": p.PublishedVersion}}, "delivery": map[string]any{"method": "MAILBOX", "location": "", "projectName": ""}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := f.s.PrepareOrderV2(context.Background(), f.buyer.ID, key, fmt.Sprint(q["quoteId"]), 1, "OFFICIAL_STORE", true, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestCommerceOfficialFrozenTemplateClaimAndRevenue(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	template, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, fmt.Sprint(content["deliveryTemplateRef"]), "delivery_template", true)
	if e != nil {
		t.Fatal(e)
	}
	changed := catalogObjectClone(template.Body)
	changed["attachments"].([]any)[0].(map[string]any)["quantity"] = 2
	_, e = f.s.CatalogWriteTemplateV2(ctx, f.admin.ID, p.StoreID, template.ID, "template-change", template.Version, changed, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
	if e != nil {
		t.Fatal(e)
	}
	m := commerceOfficialTest(t, f, p, "official")
	d := commerceRunTest(t, f, m)
	if d.FundsState != "HELD" || d.State != "AWAITING_CLAIM" || d.PayeeUUID != commerceTestRevenue || c.calls["wallet.escrow.settle"] != 0 {
		t.Fatal(d, c.calls)
	}
	plan, _ := store.CommerceContentV2(d.Body, "mailboxPlan")
	var snapshot map[string]any
	_ = json.Unmarshal([]byte(fmt.Sprint(plan["snapshotJson"])), &snapshot)
	if snapshot["attachments"].([]any)[0].(map[string]any)["quantity"] != float64(1) {
		t.Fatal("template edit changed published fulfillment", snapshot)
	}
	receipt := store.MailReceipt{DeliveryID: fmt.Sprint(plan["deliveryId"]), MailID: "mail_test", OrderID: d.ID, Source: "deuterium-commerce", SnapshotSHA256: fmt.Sprint(plan["snapshotSha256"]), RecipientUUID: d.OwnerUUID, AllowedServerIDs: []string{"amiya"}, InventoryDomain: "survival", Status: "CLAIMED", Revision: 2}
	raw, _ := json.Marshal(receipt)
	_, e = f.s.DB.Exec("INSERT INTO core_mail_receipts(delivery_id,order_id,mail_cluster,snapshot_sha256,recipient_uuid,status,revision,receipt,updated_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))", receipt.DeliveryID, d.ID, "test", receipt.SnapshotSHA256, d.OwnerUUID, "CLAIMED", 2, raw)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.server.RefreshCommerceMailboxV2(ctx, d.ID); e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, d.ID)
	if d.State != "CLAIMED" {
		t.Fatal(d)
	}
	m, e = f.s.PrepareCommerceSettlementV2(ctx, f.buyer.ID, d.ID, "ORDER", "revenue", d.Version, true, true, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.SettledAmount != "11.10" || d.FundsState != "SETTLED" || c.ledger[d.EscrowRef].payee != commerceTestRevenue {
		t.Fatal(d)
	}
}
func TestCommerceOfficialUnknownCreationMustCancelBeforeRefund(t *testing.T) {
	for _, mode := range []string{"notfound", "wrong-tombstone", "valid"} {
		t.Run(mode, func(t *testing.T) {
			f := newCatalogFixture(t)
			c := newCommerceTestCore()
			f.server.CommerceCore = c
			ctx := context.Background()
			p, _ := catalogProductFixture(t, f)
			m := commerceOfficialTest(t, f, p, "official")
			c.failOnce = "mailbox.create"
			o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
			if e != nil || o.State != "FAILED" {
				t.Fatal(o, e)
			}
			d := commerceRecordTest(t, f, m.ResourceID)
			if d.FundsState != "HELD" {
				t.Fatal(d)
			}
			c.mailMode = mode
			m, e = f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, commerceRefundInput("refund", d.Version), true)
			if e != nil {
				t.Fatal(e)
			}
			o, e = f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
			if e != nil {
				t.Fatal(e)
			}
			d = commerceRecordTest(t, f, d.ID)
			if mode == "valid" {
				if o.State != "COMPLETED" || d.FundsState != "REFUNDED" || c.calls["wallet.escrow.refund"] != 1 || d.Body["mailboxState"] != "CANCELLED_BEFORE_CREATE" {
					t.Fatal(o, d, c.calls)
				}
			} else if o.State == "COMPLETED" || d.FundsState == "REFUNDED" || c.calls["wallet.escrow.refund"] != 0 {
				t.Fatal("unsafe refund", o, d, c.calls)
			}
		})
	}
}

func TestCommerceWorkerSelectionSkipsTwentyOneBlockedResources(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	asset := catalogTestAsset(t, f.s, f.buyer.ID, "COMMISSION_COVER")
	ids := []string{}
	ops := []string{}
	for i := 0; i < 22; i++ {
		m, e := f.s.PrepareCommissionV2(ctx, f.buyer.ID, fmt.Sprintf("work-%02d", i), store.CatalogObjectV2{"title": "后台公平性", "description": "用于隔离数据库中后台队列公平性验证", "location": "主城东门", "urgency": "NORMAL", "reward": "1.00", "workHours": 24, "coverAssetId": asset}, true)
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, m.ResourceID)
		ops = append(ops, m.OperationID)
		if i < 21 {
			if e = f.s.DeferCommerceOperationV2(ctx, m.OperationID, time.Now().Add(time.Hour)); e != nil {
				t.Fatal(e)
			}
		}
	}
	pending, e := f.s.PendingCommerceOperationsV2(ctx, 20)
	if e != nil || len(pending) != 1 || pending[0].ID != ops[21] {
		t.Fatal("blocked pending starvation", pending, e)
	}
	// Simulate durable imported records: 21 unfinished construction deadlines
	// must be filtered by SQL before LIMIT; the last ordinary shipment is due.
	for i, id := range ids {
		body := map[string]any{"construction": i < 21}
		raw, _ := json.Marshal(body)
		_, e = f.s.DB.Exec("UPDATE commerce_resources_v2 SET resource_kind='ORDER',channel='PLAYER_MARKET',state='SHIPPED',funds_state='HELD',pending_operation_id=NULL,deadline_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 HOUR),body=? WHERE resource_id=?", raw, id)
		if e != nil {
			t.Fatal(e)
		}
	}
	due, e := f.s.DueCommerceResourcesV2(ctx, time.Now(), 20)
	if e != nil || len(due) != 1 || due[0].ID != ids[21] {
		t.Fatal("construction starvation", due, e)
	}
	// Unchanged mailbox revisions do not occupy the first page of receipts.
	for i, id := range ids {
		plan := map[string]any{"deliveryId": "delivery_" + id, "snapshotSha256": strings.Repeat("a", 64)}
		raw, _ := json.Marshal(map[string]any{"mailboxPlan": plan, "mailboxRevision": 1})
		_, e = f.s.DB.Exec("UPDATE commerce_resources_v2 SET channel='OFFICIAL_STORE',body=? WHERE resource_id=?", raw, id)
		if e != nil {
			t.Fatal(e)
		}
		revision := 1
		if i == 21 {
			revision = 2
		}
		_, e = f.s.DB.Exec("INSERT INTO core_mail_receipts VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))", plan["deliveryId"], id, "test", plan["snapshotSha256"], f.buyer.ServerUUID, "CREATED", revision, "{}")
		if e != nil {
			t.Fatal(e)
		}
	}
	official, e := f.s.OfficialCommerceResourcesV2(ctx, 20)
	if e != nil || len(official) != 1 || official[0].ID != ids[21] {
		t.Fatal("unchanged mail starvation", official, e)
	}
}

func TestCommerceFinancialHTTPContractEnvelopes(t *testing.T) {
	f := newCatalogFixture(t)
	f.server.CommerceCore = newCommerceTestCore()
	ctx := context.Background()
	mux := http.NewServeMux()
	f.server.registerCommerceV2(mux)
	samples := map[string]json.RawMessage{}
	request := func(schema, method, path, token string, body any) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code < 200 || w.Code > 299 {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		if schema != "" {
			samples[schema] = append(json.RawMessage{}, w.Body.Bytes()...)
		}
		var envelope map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &envelope); e != nil {
			t.Fatal(e)
		}
		return envelope["data"].(map[string]any)
	}
	_, quote := commerceMarketTest(t, f, "http-contract", 1)
	created := request("V2OrderCreationResultResponse", "POST", "/api/v1/market/orders", f.buyerToken, map[string]any{"clientRequestId": "http-pay", "quoteId": quote["quoteId"], "expectedQuoteVersion": 1})
	order := created["order"].(map[string]any)
	orderID := fmt.Sprint(order["orderId"])
	operation := created["operation"].(map[string]any)
	request("V2OrderViewResponse", "GET", "/api/v1/orders/"+orderID, f.buyerToken, nil)
	request("V2OrderContractSnapshotResponse", "GET", "/api/v1/orders/"+orderID+"/snapshot", f.buyerToken, nil)
	request("V2OperationLookupResponse", "GET", "/api/v1/operations/"+fmt.Sprint(operation["operationId"]), f.buyerToken, nil)
	order = request("", "POST", "/api/v1/orders/"+orderID+"/ship", f.adminToken, map[string]any{"clientRequestId": "http-ship", "expectedVersion": order["version"]})
	order = request("", "POST", "/api/v1/orders/"+orderID+"/refunds", f.buyerToken, commerceRefundInput("http-refund", int64(order["version"].(float64))))
	refund := order["refund"].(map[string]any)
	refundPath := "/api/v1/orders/" + orderID + "/refunds/" + fmt.Sprint(refund["refundId"])
	request("V2OrderRefundResponse", "GET", refundPath, f.buyerToken, nil)
	order = request("", "POST", refundPath+"/resolve", f.adminToken, map[string]any{"clientRequestId": "http-reject", "expectedVersion": refund["version"], "decision": "REJECT", "reason": "已依照双方约定交付"})
	caseView := request("V2InterventionViewResponse", "POST", "/api/v1/orders/"+orderID+"/interventions", f.buyerToken, map[string]any{"clientRequestId": "http-case", "expectedVersion": order["version"], "reasonCode": "REFUND_DISAGREEMENT", "description": "请求核对原始约定与目前交付之间的差异。", "desiredResolution": "FULL_REFUND", "evidenceAssetIds": []any{}})
	if caseView["caseId"] == nil {
		t.Fatal(caseView)
	}
	asset := catalogTestAsset(t, f.s, f.buyer.ID, "COMMISSION_COVER")
	commissionResult := request("V2CommissionCreationResultResponse", "POST", "/api/v1/commissions", f.buyerToken, map[string]any{"clientRequestId": "http-commission", "content": map[string]any{"title": "HTTP 委托", "description": "核对真实 HTTP 委托结构与数据库状态", "location": "主城东门", "urgency": "NORMAL", "reward": "2.30", "workHours": 24, "coverAssetId": asset}})
	commission := commissionResult["commission"].(map[string]any)
	id := fmt.Sprint(commission["commissionId"])
	commission = request("V2CommissionViewResponse", "POST", "/api/v1/commissions/"+id+"/accept", f.adminToken, map[string]any{"clientRequestId": "http-accept", "expectedVersion": commission["version"]})
	request("V2CommissionContractSnapshotResponse", "GET", "/api/v1/commissions/"+id+"/snapshot", f.buyerToken, nil)
	evidence := catalogTestAsset(t, f.s, f.admin.ID, "DISPUTE_EVIDENCE")
	commission = request("V2CommissionViewResponse", "POST", "/api/v1/commissions/"+id+"/complete", f.adminToken, map[string]any{"clientRequestId": "http-complete", "expectedVersion": commission["version"], "description": "委托完成并上传施工凭证", "evidenceAssetIds": []any{evidence}})
	if len(commission["completionAssets"].([]any)) != 1 {
		t.Fatal("completion image inaccessible")
	}
	if _, e := f.s.CommerceViewV2(ctx, f.other.ID, id, false); e == nil {
		t.Fatal("private completion leaked")
	}
	if directory := os.Getenv("DEUTERIUM_COMMERCE_QA_DIR"); directory != "" {
		if e := os.MkdirAll(directory, 0700); e != nil {
			t.Fatal(e)
		}
		raw, _ := json.MarshalIndent(samples, "", "  ")
		if e := os.WriteFile(filepath.Join(directory, "responses.json"), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
}

func TestCommerceOrderConcurrentRequestAndStockOwnership(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	l, q := commerceMarketTest(t, f, "race", 1)
	var wg sync.WaitGroup
	results := make(chan store.CommerceMutationV2, 12)
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := f.s.PrepareOrderV2(ctx, f.buyer.ID, "same-payment", fmt.Sprint(q["quoteId"]), 1, "PLAYER_MARKET", true, nil)
			if e != nil {
				errs <- e
			} else {
				results <- m
			}
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	var first store.CommerceMutationV2
	for m := range results {
		if first.OperationID != "" && m.OperationID != first.OperationID {
			t.Fatal("duplicate operation")
		}
		first = m
	}
	if catalogTestCount(t, f.s, "commerce_resources_v2") != 1 || catalogTestCount(t, f.s, "commerce_operations_v2") != 1 {
		t.Fatal("duplicate payment intent")
	}
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := f.server.RunCommerceOperationV2(ctx, first.OperationID, true)
			if e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	commerceRunTest(t, f, first)
	if c.calls["wallet.escrow.reserve"] != 1 {
		t.Fatal(c.calls)
	}
	product, e := f.s.CatalogGetRecordV2(ctx, f.admin.ID, l.ID, "listing", true)
	if e != nil || *product.Stock != 0 {
		t.Fatal(product, e)
	}
	// Two independently quoted buyers cannot both consume one inventory unit.
	l, q = commerceMarketTest(t, f, "stock-race", 1)
	q2, e := f.s.CatalogQuoteV2(ctx, f.other.ID, "other-quote", store.CatalogObjectV2{"channel": "PLAYER_MARKET", "items": []any{map[string]any{"productId": l.ID, "quantity": 1, "expectedProductVersion": l.Version}}, "delivery": map[string]any{"method": "PICKUP", "location": "主城仓库", "projectName": ""}})
	if e != nil {
		t.Fatal(e)
	}
	type outcome struct {
		m store.CommerceMutationV2
		e error
	}
	out := make(chan outcome, 2)
	for _, item := range []struct{ user, key, quote string }{{f.buyer.ID, "buyer-pay", fmt.Sprint(q["quoteId"])}, {f.other.ID, "other-pay", fmt.Sprint(q2["quoteId"])}} {
		wg.Add(1)
		go func(user, key, quote string) {
			defer wg.Done()
			m, e := f.s.PrepareOrderV2(ctx, user, key, quote, 1, "PLAYER_MARKET", true, nil)
			out <- outcome{m, e}
		}(item.user, item.key, item.quote)
	}
	wg.Wait()
	close(out)
	wins := 0
	for result := range out {
		if result.e == nil {
			wins++
			commerceRunTest(t, f, result.m)
		}
	}
	if wins != 1 {
		t.Fatal("oversold", wins)
	}
}

func TestCommerceCommissionFailedBindReopensAndCancelRefunds(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	d := commerceCommissionTest(t, f, "bind-fails")
	m, e := f.s.PrepareCommissionAcceptV2(ctx, f.admin.ID, d.ID, "failed-bind", d.Version, true)
	if e != nil {
		t.Fatal(e)
	}
	c.failOnce = "wallet.escrow.bind"
	o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "FAILED" {
		t.Fatal(o, e)
	}
	d = commerceRecordTest(t, f, d.ID)
	if d.State != "OPEN" || d.PayeeID != "" || d.Deadline != nil || d.FundsState != "HELD" {
		t.Fatal(d)
	}
	m, e = f.s.PrepareCommissionCancelV2(ctx, f.buyer.ID, d.ID, "cancel", d.Version, true)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if d.State != "CANCELLED" || d.FundsState != "REFUNDED" || d.RefundedAmount != "20.00" {
		t.Fatal(d)
	}
	d = commerceCommissionTest(t, f, "bind-unknown")
	m, e = f.s.PrepareCommissionAcceptV2(ctx, f.admin.ID, d.ID, "unknown-bind", d.Version, true)
	if e != nil {
		t.Fatal(e)
	}
	c.unknownOnce = "wallet.escrow.bind"
	o, e = f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "UNKNOWN" {
		t.Fatal(o, e)
	}
	d = commerceRecordTest(t, f, d.ID)
	_, e = f.s.PrepareCommissionAcceptV2(ctx, f.other.ID, d.ID, "steal-bind", d.Version, true)
	assertCatalogCode(t, e, "OPERATION_IN_PROGRESS")
	o, e = f.server.RunCommerceOperationV2(ctx, m.OperationID, false)
	if e != nil || o.State != "COMPLETED" {
		t.Fatal(o, e)
	}
	d = commerceRecordTest(t, f, d.ID)
	if d.PayeeID != f.admin.ID || d.FundsState != "HELD" {
		t.Fatal(d)
	}
}

func TestCommerceUnlimitedPurchaseDoesNotRestoreFiniteStock(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	p, content := catalogProductFixture(t, f)
	content["inventoryPolicy"] = "UNLIMITED"
	content["stock"] = 0
	draft, e := f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "unlimited-draft", p.Version, content, false)
	if e != nil {
		t.Fatal(e)
	}
	p, e = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "unlimited-publish", draft.Version, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	m := commerceOfficialTest(t, f, p, "unlimited-purchase")
	// Merchant changes future stock policy while the original charge is in flight.
	content["inventoryPolicy"] = "FINITE"
	content["stock"] = 7
	draft, e = f.s.CatalogEditV2(ctx, f.admin.ID, p.ID, "product", "finite-draft", p.Version, content, false)
	if e != nil {
		t.Fatal(e)
	}
	p, e = f.s.CatalogActionV2(ctx, f.admin.ID, p.ID, "product", "finite-publish", draft.Version, "publish", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	c.failOnce = "wallet.escrow.reserve"
	o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "FAILED" {
		t.Fatal(o, e)
	}
	p, e = f.s.CatalogGetRecordV2(ctx, f.admin.ID, p.ID, "product", true)
	if e != nil || p.Stock == nil || *p.Stock != 7 {
		t.Fatal("invented finite stock from unlimited purchase", p, e)
	}
}

func TestCommerceLateClaimClosesFailedImmediateRefundAndSettlesRevenue(t *testing.T) {
	f := newCatalogFixture(t)
	c := newCommerceTestCore()
	f.server.CommerceCore = c
	ctx := context.Background()
	p, _ := catalogProductFixture(t, f)
	d := commerceRunTest(t, f, commerceOfficialTest(t, f, p, "late-claim"))
	m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund-before-claim", d.Version, commerceRefundInput("refund-before-claim", d.Version), true)
	if e != nil {
		t.Fatal(e)
	}
	c.failOnce = "mailbox.revoke"
	o, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || o.State != "FAILED" {
		t.Fatal(o, e)
	}
	d = commerceRecordTest(t, f, d.ID)
	refund, e := f.s.CommerceRefundViewV2(ctx, f.buyer.ID, d.ID, d.RefundID)
	if e != nil || refund["status"] != "PROCESSING" || d.PendingOperationID != "" {
		t.Fatal(refund, d, e)
	}
	plan, _ := store.CommerceContentV2(d.Body, "mailboxPlan")
	receipt := store.MailReceipt{DeliveryID: fmt.Sprint(plan["deliveryId"]), MailID: "mail_test", OrderID: d.ID, Source: "deuterium-commerce", SnapshotSHA256: fmt.Sprint(plan["snapshotSha256"]), RecipientUUID: d.OwnerUUID, AllowedServerIDs: []string{"amiya"}, InventoryDomain: "survival", Status: "CLAIMED", Revision: 2}
	raw, _ := json.Marshal(receipt)
	_, e = f.s.DB.Exec("INSERT INTO core_mail_receipts(delivery_id,order_id,mail_cluster,snapshot_sha256,recipient_uuid,status,revision,receipt,updated_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))", receipt.DeliveryID, d.ID, "test", receipt.SnapshotSHA256, d.OwnerUUID, "CLAIMED", 2, raw)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.server.RefreshCommerceMailboxV2(ctx, d.ID); e != nil {
		t.Fatal(e)
	}
	d = commerceRecordTest(t, f, d.ID)
	refund, e = f.s.CommerceRefundViewV2(ctx, f.buyer.ID, d.ID, d.RefundID)
	if e != nil || refund["status"] != "REJECTED" || d.State != "CLAIMED" {
		t.Fatal(refund, d, e)
	}
	// Recreate the HTTP server over the same durable store to exercise restart
	// reconciliation instead of relying on an in-memory timer or caller.
	restarted := New(f.s, f.server.Config)
	t.Cleanup(restarted.Close)
	restarted.CommerceCore = c
	restarted.reconcileCommerceV2(ctx)
	d = commerceRecordTest(t, f, d.ID)
	if d.FundsState != "SETTLED" || d.SettledAmount != "11.10" || c.calls["wallet.escrow.refund"] != 0 || c.calls["wallet.escrow.settle"] != 1 {
		t.Fatal(d, c.calls)
	}
	restarted.reconcileCommerceV2(ctx)
	if c.calls["wallet.escrow.settle"] != 1 {
		t.Fatal("repeated automatic settlement", c.calls)
	}
}

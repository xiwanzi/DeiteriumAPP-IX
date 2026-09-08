package store

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCommerceMoneyProofBindsEveryAuthorityAndConservesFunds(t *testing.T) {
	d := CommerceRecordV2{ID: "order_test", Kind: "ORDER", Channel: "PLAYER_MARKET", EscrowRef: "escrow:order_test", OwnerUUID: "10000000-0000-0000-0000-000000000001", PayeeUUID: "10000000-0000-0000-0000-000000000002", Amount: "12.30", SettledAmount: "0.00", RefundedAmount: "0.00", Body: CatalogObjectV2{}}
	step := commerceReserveStepV2(d)
	good := CatalogObjectV2{"operationId": "core_test", "businessRef": d.ID, "escrowRef": d.EscrowRef, "businessType": "MARKET_ORDER", "payerUuid": d.OwnerUUID, "payeeUuid": d.PayeeUUID, "fromUuid": d.OwnerUUID, "toUuid": "20000000-0000-0000-0000-000000000001", "amount": "12.30", "currency": "CREDIT", "reservedAmount": "12.30", "settledAmount": "0.00", "refundedAmount": "0.00", "heldAmount": "12.30", "escrowStatus": "HELD", "status": "COMPLETED", "committedAt": time.Now().UTC().Format(time.RFC3339Nano)}
	if _, e := commerceCheckMoneyProofV2(d, step, CommerceStepResultV2{CoreOperationID: "core_test", Data: good}); e != nil {
		t.Fatal(e)
	}
	bad := map[string]any{"operationId": "core_other", "businessRef": "order_other", "escrowRef": "escrow:other", "businessType": "COMMISSION", "payerUuid": d.PayeeUUID, "payeeUuid": d.OwnerUUID, "fromUuid": d.PayeeUUID, "toUuid": d.PayeeUUID, "amount": "0.00", "currency": "USD", "reservedAmount": "12.31", "settledAmount": "0.01", "refundedAmount": "0.01", "heldAmount": "12.29", "escrowStatus": "", "status": "PENDING", "committedAt": ""}
	for field, value := range bad {
		raw, _ := json.Marshal(good)
		var p CatalogObjectV2
		_ = json.Unmarshal(raw, &p)
		p[field] = value
		if _, e := commerceCheckMoneyProofV2(d, step, CommerceStepResultV2{CoreOperationID: "core_test", Data: p}); e == nil {
			t.Fatalf("accepted forged %s", field)
		}
	}
	for _, code := range []string{"STORAGE_UNAVAILABLE", "ECONOMY_UNAVAILABLE", "IDEMPOTENCY_CONFLICT", "PAYEE_LOCKED"} {
		if !commerceUncertainCodeV2(code) {
			t.Fatalf("unsafe definite failure %s", code)
		}
	}
}

func TestCommerceCancellationRequiresExactTombstoneOrRealRevokedMail(t *testing.T) {
	d := CommerceRecordV2{ID: "order_test", OwnerUUID: "10000000-0000-0000-0000-000000000001", Body: CatalogObjectV2{"mailboxPlan": CatalogObjectV2{"deliveryId": "delivery_test", "snapshotSha256": "sha_test", "inventoryDomain": "survival", "allowedServerIds": []string{"amiya", "odyssey"}}}}
	good := CatalogObjectV2{"operationId": "core_cancel", "source": "deuterium-commerce", "deliveryId": "delivery_test", "orderId": d.ID, "recipientUuid": d.OwnerUUID, "snapshotSha256": "sha_test", "inventoryDomain": "survival", "allowedServerIds": []string{"odyssey", "amiya"}, "proofKind": "CANCELLED_BEFORE_CREATE", "cancelledAt": time.Now().Unix()}
	verify := func(p CatalogObjectV2) (string, bool) {
		raw, _ := json.Marshal(map[string]any{"code": "OK", "value": p})
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		status, _, _, tombstone := commerceCancellationResultV2(d, CommerceStepResultV2{CoreOperationID: "core_cancel", Data: decoded})
		return status, tombstone
	}
	if status, tombstone := verify(good); status != "COMPLETED" || !tombstone {
		t.Fatal(status, tombstone)
	}
	for field, value := range map[string]any{"operationId": "other", "recipientUuid": "other", "snapshotSha256": "other", "inventoryDomain": "other", "deliveryId": "other", "orderId": "other", "source": "other", "allowedServerIds": []string{"login"}, "cancelledAt": 0, "mailReceipt": map[string]any{"mailId": "fake"}, "proofKind": "NOT_FOUND"} {
		p := copyCatalogTest(good)
		p[field] = value
		if status, _ := verify(p); status == "COMPLETED" {
			t.Fatalf("accepted invalid cancellation %s", field)
		}
	}
	p := copyCatalogTest(good)
	p["proofKind"] = "REVOKED_MAIL"
	if status, _ := verify(p); status == "COMPLETED" {
		t.Fatal("accepted revoked without receipt")
	}
	d.Body["mailId"] = "mail_exists"
	d.Body["mailboxRevision"] = 1
	if status, _ := verify(good); status == "COMPLETED" {
		t.Fatal("tombstone accepted after known mail exists")
	}
}

func TestCommerceMailCreateRequiresConfirmedReceiptState(t *testing.T) {
	d := CommerceRecordV2{ID: "order_test", OwnerUUID: "10000000-0000-0000-0000-000000000001", Body: CatalogObjectV2{"mailboxPlan": CatalogObjectV2{"deliveryId": "delivery_test", "snapshotSha256": "sha_test", "inventoryDomain": "survival", "allowedServerIds": []string{"amiya"}}}}
	receipt := CatalogObjectV2{"source": "deuterium-commerce", "deliveryId": "delivery_test", "orderId": d.ID, "recipientUuid": d.OwnerUUID, "snapshotSha256": "sha_test", "inventoryDomain": "survival", "allowedServerIds": []string{"amiya"}, "mailId": "mail_test", "revision": 1, "status": "UNKNOWN"}
	result := CommerceStepResultV2{Status: "COMPLETED", Data: map[string]any{"code": "OK", "value": receipt}}
	state, code, _ := commerceMailResultV2(d, CommerceStepV2{Command: "mailbox.create"}, result)
	if state != "UNKNOWN" || code != "MAILBOX_STATUS_UNCONFIRMED" {
		t.Fatal(state, code)
	}
	for _, status := range []string{"CREATED", "CLAIMING", "CLAIMED"} {
		receipt["status"] = status
		state, _, _ = commerceMailResultV2(d, CommerceStepV2{Command: "mailbox.create"}, result)
		if state != "COMPLETED" {
			t.Fatal(status, state)
		}
	}
}

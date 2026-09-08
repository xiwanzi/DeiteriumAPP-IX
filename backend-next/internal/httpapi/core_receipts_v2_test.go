package httpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestCoreTransferCompletionRequiresMatchingCommittedProof(t *testing.T) {
	expected := map[string]any{"fromUuid": "a", "toUuid": "b", "amount": "1.00"}
	payload, _ := json.Marshal(expected)
	op := store.CoreOperation{ID: "original", Command: "wallet.transfer", Payload: payload}
	proof := map[string]any{"operationId": "original", "fromUuid": "a", "toUuid": "b", "amount": "1.00", "currency": "CREDIT", "status": "COMPLETED", "committedAt": time.Now().UTC()}
	server := &Server{}
	data, _ := json.Marshal(proof)
	if err := server.acceptCoreReceiptV2(context.Background(), config.Node{}, op, "COMPLETED", data); err != nil {
		t.Fatal(err)
	}
	for key, wrong := range map[string]any{"operationId": "other", "fromUuid": "other", "toUuid": "other", "amount": "2.00", "currency": "OTHER", "status": "FAILED", "committedAt": nil} {
		changed := map[string]any{}
		for k, v := range proof {
			changed[k] = v
		}
		changed[key] = wrong
		data, _ = json.Marshal(changed)
		if err := server.acceptCoreReceiptV2(context.Background(), config.Node{}, op, "COMPLETED", data); err == nil {
			t.Errorf("accepted mismatched %s", key)
		}
	}
	if err := server.acceptCoreReceiptV2(context.Background(), config.Node{}, op, "COMPLETED", json.RawMessage(`{"amount":"1.00"}`)); err == nil {
		t.Fatal("accepted amount-only completion")
	}
}

func TestCoreTransferCompletionAcceptsXConomyReceiptAndRejectsPartyConflicts(t *testing.T) {
	const payer = "10000000-0000-0000-0000-000000000001"
	const payee = "10000000-0000-0000-0000-000000000002"
	payload, _ := json.Marshal(map[string]any{"fromUuid": payer, "toUuid": payee, "amount": "1.00"})
	op := store.CoreOperation{ID: "original_transfer", Command: "wallet.transfer", Payload: payload}
	// Exact committed FundsEngine transfer shape, including its businessRef.
	proof := map[string]any{"operationId": op.ID, "businessRef": op.ID, "payerUuid": payer, "payeeUuid": payee, "amount": "1.00", "currency": "CREDIT", "status": "COMPLETED", "committedAt": "2026-09-08T05:58:43.123456789Z"}
	server := &Server{}
	check := func(value map[string]any) error {
		data, _ := json.Marshal(value)
		return server.acceptCoreReceiptV2(context.Background(), config.Node{}, op, "COMPLETED", data)
	}
	if err := check(proof); err != nil {
		t.Fatal("committed XConomy receipt was rejected", err)
	}
	proof["fromUuid"], proof["toUuid"] = payer, payee
	if err := check(proof); err != nil {
		t.Fatal("matching compatibility fields were rejected", err)
	}
	for key, wrong := range map[string]any{"payerUuid": payee, "payeeUuid": payer, "fromUuid": payee, "toUuid": payer, "operationId": "other", "amount": "2.00", "currency": "OTHER", "status": "FAILED", "committedAt": nil} {
		changed := map[string]any{}
		for k, v := range proof {
			changed[k] = v
		}
		changed[key] = wrong
		if err := check(changed); err == nil {
			t.Errorf("accepted mismatched %s", key)
		}
	}
	for _, key := range []string{"payerUuid", "payeeUuid", "fromUuid", "toUuid"} {
		changed := map[string]any{}
		for k, v := range proof {
			changed[k] = v
		}
		changed[key] = nil
		if err := check(changed); err == nil {
			t.Errorf("accepted explicit null %s alongside valid alias", key)
		}
	}
	delete(proof, "fromUuid")
	delete(proof, "toUuid")
	proof["payeeUuid"] = payer
	if err := check(proof); err == nil {
		t.Fatal("accepted canonical receipt for the wrong recipient")
	}
	delete(proof, "payeeUuid")
	if err := check(proof); err == nil {
		t.Fatal("accepted receipt without recipient proof")
	}
}

func cancellationFixtureV2() (config.Config, config.Node, store.CoreOperation, map[string]any) {
	node := config.Node{ID: "amiya", MailCluster: "cluster", InventoryDomain: "survival", ClaimEnabled: true}
	c := config.Config{Nodes: []config.Node{node}}
	snapshot := `{"schemaVersion":1,"orderId":"order_one","recipientUuid":"9a8b96da-27d4-4aee-9b94-0d5ce9d51a55","inventoryDomain":"survival","allowedServerIds":["amiya"],"attachments":[{"itemRef":"deuterium:archived","revision":1,"quantity":1}]}`
	hash := store.Digest([]byte(snapshot))
	payload, _ := json.Marshal(map[string]any{"source": "deuterium-commerce", "deliveryId": "delivery_one", "orderId": "order_one", "expectedSnapshotSha256": hash, "reasonCode": "CUSTOMER_REFUND", "snapshotJson": snapshot})
	op := store.CoreOperation{ID: "cancel_one", Command: "mailbox.revoke", Payload: payload}
	proof := map[string]any{"operationId": op.ID, "source": "deuterium-commerce", "deliveryId": "delivery_one", "orderId": "order_one", "snapshotSha256": hash, "recipientUuid": "9a8b96da-27d4-4aee-9b94-0d5ce9d51a55", "allowedServerIds": []string{"amiya"}, "inventoryDomain": "survival", "proofKind": "CANCELLED_BEFORE_CREATE", "mailReceipt": nil, "cancelledAt": time.Now().Unix(), "replayed": false}
	return c, node, op, proof
}

func TestCancelledDeliveryProofCannotFabricateMailOrChangeFrozenIdentity(t *testing.T) {
	c, node, op, proof := cancellationFixtureV2()
	raw, _ := json.Marshal(proof)
	if receipt, err := validateMailCancellationProofV2(c, node, op, raw); err != nil || receipt != nil {
		t.Fatal("cancellation should not fabricate mail", receipt, err)
	}
	for key, wrong := range map[string]any{"operationId": "other", "orderId": "other", "recipientUuid": "other", "snapshotSha256": "other", "inventoryDomain": "other", "allowedServerIds": []string{"amiya", "amiya"}, "proofKind": "REVOKED_MAIL", "mailReceipt": map[string]any{"mailId": "fake"}, "cancelledAt": 0} {
		changed := map[string]any{}
		for k, v := range proof {
			changed[k] = v
		}
		changed[key] = wrong
		raw, _ = json.Marshal(changed)
		if _, err := validateMailCancellationProofV2(c, node, op, raw); err == nil {
			t.Errorf("accepted changed %s", key)
		}
	}
	data, _ := json.Marshal(map[string]any{"code": "OK", "value": proof})
	// Nil Store proves this path does not manufacture core_mail_receipts.
	if err := (&Server{Config: c}).acceptCoreReceiptV2(context.Background(), node, op, "COMPLETED", data); err != nil {
		t.Fatal(err)
	}
}

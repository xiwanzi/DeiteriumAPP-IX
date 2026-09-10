//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestCommerceCoreAdapterSerializesConcurrentReplayAndRejectsOtherActor(t *testing.T) {
	db := testdb.New(t)
	s := New(db, config.Config{Nodes: []config.Node{{ID: "amiya", Economy: true}}})
	t.Cleanup(s.Close)
	a := s.CommerceCore
	var sent atomic.Int32
	s.Core.Connect("amiya", func(ctx context.Context, _ string, value any) error {
		frame := value.(map[string]any)
		id := frame["operationId"].(string)
		data := map[string]any{"operationId": id, "businessRef": "order_one", "amount": "1.00", "status": "RESERVED"}
		switch frame["command"] {
		case CommerceReserveV2:
			sent.Add(1)
		case "operation.query":
			// A concurrent caller can observe SENT and query its result while the
			// reserve callback commits. This read is not a duplicate funds command.
			data = map[string]any{"acquired": true, "state": "PROCESSING", "result": nil}
		default:
			t.Errorf("unexpected Core command: %v", frame["command"])
		}
		result, _ := json.Marshal(map[string]any{"operationId": id, "status": "COMPLETED", "data": data})
		if err := db.CoreReply(ctx, "amiya", id, "COMPLETED", result); err != nil {
			return err
		}
		s.Core.Notify(id)
		return nil
	})
	s.Core.SetStatus("amiya", []byte(`{"storageHealthy":true,"economy":true,"economyAuthority":true}`))
	if !a.Available(CommerceReserveV2) || a.Available("wallet.transfer") {
		t.Fatal("capability gating failed")
	}
	payload := map[string]any{"businessRef": "order_one", "amount": "1.00"}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := a.Execute(context.Background(), "payer", "reserve_one", CommerceReserveV2, payload)
			if e != nil {
				t.Error(e)
			}
			ids <- r.OperationID
		}()
	}
	wg.Wait()
	close(ids)
	var original string
	for id := range ids {
		if id == "" || (original != "" && original != id) {
			t.Fatal("duplicate operation", id)
		}
		original = id
	}
	if sent.Load() != 1 {
		t.Fatal("duplicate Core dispatch", sent.Load())
	}
	s.Core.Disconnect("amiya")
	if a.Available(CommerceReserveV2) {
		t.Fatal("offline capability accepted")
	}
	replayed, err := a.Execute(context.Background(), "payer", "reserve_one", CommerceReserveV2, payload)
	if err != nil || replayed.OperationID != original || replayed.Status != "COMPLETED" {
		t.Fatal("offline committed replay lost", replayed, err)
	}
	if _, err = a.Query(context.Background(), "other", original); !errors.Is(err, ErrForbidden) {
		t.Fatal("other actor queried receipt", err)
	}
	if _, err = a.Execute(context.Background(), "payer", "reserve_one", CommerceReserveV2, map[string]any{"amount": "2.00"}); !errors.Is(err, store.ErrConflict) {
		t.Fatal("mutated replay accepted", err)
	}
}

func TestCommerceCoreAdapterRequiresMailDomainAndFrozenCluster(t *testing.T) {
	db := testdb.New(t)
	s := New(db, config.Config{Nodes: []config.Node{{ID: "login", MailCluster: "cluster", InventoryDomain: "login"}, {ID: "amiya", MailCluster: "cluster", InventoryDomain: "survival", ClaimEnabled: true}}})
	t.Cleanup(s.Close)
	a := s.CommerceCore.(*commerceCoreAdapter)
	for _, id := range []string{"login", "amiya"} {
		s.Core.Connect(id, func(context.Context, string, any) error { return nil })
		s.Core.SetStatus(id, []byte(`{"storageHealthy":true,"mailbox":{"available":true,"commerceReady":true,"clusterId":"cluster"}}`))
	}
	payload := map[string]any{"inventoryDomain": "survival", "allowedServerIds": []string{"amiya"}, "snapshotJson": `{"creditAmount":0}`}
	node, err := a.selectNode(context.Background(), "mailbox.create", payload)
	if err != nil || node != "amiya" {
		t.Fatal(node, err)
	}
	payload["snapshotJson"] = `{"creditAmount":20}`
	if _, err = a.selectNode(context.Background(), "mailbox.create", payload); err == nil {
		t.Fatal("credit delivery routed to an old node")
	}
	s.Core.SetStatus("amiya", []byte(`{"storageHealthy":true,"mailbox":{"available":true,"commerceReady":true,"clusterId":"cluster","apiVersion":3,"creditRewards":true}}`))
	if node, err = a.selectNode(context.Background(), "mailbox.create", payload); err != nil || node != "amiya" {
		t.Fatal("credit-capable node rejected", node, err)
	}
	if _, err = a.selectNode(context.Background(), "mailbox.create", map[string]any{"inventoryDomain": "survival", "allowedServerIds": []string{"login"}, "snapshotJson": `{"creditAmount":0}`}); err == nil {
		t.Fatal("incompatible claim scope accepted")
	}
	if _, err = a.selectNode(context.Background(), "mailbox.revoke", map[string]any{"deliveryId": "missing", "orderId": "order", "expectedSnapshotSha256": "hash"}); err == nil {
		t.Fatal("revoke without creation receipt accepted")
	}
}

func TestCommerceCoreAdapterDoesNotTreatRPCCompletionAsMailSuccess(t *testing.T) {
	for _, test := range []struct{ command, code, mailStatus, want string }{
		{"mailbox.create", "OK", "CREATED", "COMPLETED"},
		{"mailbox.create", "DELIVERY_CONFLICT", "", "FAILED"},
		{"mailbox.revoke", "ALREADY_REVOKED", "REVOKED", "COMPLETED"},
		{"mailbox.revoke", "OK", "CLAIMED", "UNKNOWN"},
		{"mailbox.revoke", "ALREADY_CLAIMED", "CLAIMED", "FAILED"},
		{"mailbox.revoke", "CLAIM_IN_PROGRESS", "CLAIMING", "UNKNOWN"},
		{"mailbox.create", "STORAGE_UNAVAILABLE", "", "UNKNOWN"},
	} {
		raw, _ := json.Marshal(map[string]any{"data": map[string]any{"code": test.code, "value": map[string]any{"status": test.mailStatus}}})
		got := commerceCoreResult(store.CoreOperation{ID: "original", Command: test.command, State: "COMPLETED", Result: raw})
		if got.Status != test.want || got.OperationID != "original" {
			t.Errorf("%+v: %+v", test, got)
		}
	}
}

func TestCommerceCoreAdapterRoutesMissingDeliveryCancellationOnlyToAPI2(t *testing.T) {
	db := testdb.New(t)
	c, node, op, proof := cancellationFixtureV2()
	c.Nodes[0].ClaimEnabled = false
	s := New(db, c)
	t.Cleanup(s.Close)
	s.Core.Connect(node.ID, func(context.Context, string, any) error { return nil })
	s.Core.SetStatus(node.ID, []byte(`{"storageHealthy":true,"mailbox":{"available":true,"commerceReady":true,"clusterId":"cluster","apiVersion":2,"missingDeliveryCancellation":true}}`))
	a := s.CommerceCore.(*commerceCoreAdapter)
	var payload map[string]any
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	chosen, err := a.selectNode(context.Background(), "mailbox.revoke", payload)
	if err != nil || chosen != node.ID {
		t.Fatal("missing delivery could not be safely cancelled", chosen, err)
	}
	data, _ := json.Marshal(map[string]any{"data": map[string]any{"code": "OK", "value": proof}})
	op.Result = data
	op.State = "COMPLETED"
	if got := commerceCoreResult(op); got.Status != "COMPLETED" {
		t.Fatal(got)
	}
	s.Core.SetStatus(node.ID, []byte(`{"storageHealthy":true,"mailbox":{"available":true,"commerceReady":true,"clusterId":"cluster","apiVersion":1}}`))
	if a.Available("mailbox.revoke") {
		t.Fatal("old Mail API exposed strict cancellation")
	}
}

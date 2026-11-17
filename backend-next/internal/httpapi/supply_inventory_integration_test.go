//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func supplyFixtureInventory() []supplyInventorySlot {
	rows := []supplyInventorySlot{}
	for i := 0; i < 36; i++ {
		rows = append(rows, supplyInventorySlot{Slot: fmt.Sprintf("storage:%d", i)})
	}
	for _, key := range []string{"head", "chest", "legs", "feet", "offhand"} {
		rows = append(rows, supplyInventorySlot{Slot: key})
	}
	rows[0] = supplyInventorySlot{Slot: "storage:0", Name: "QBZ-191「印花集」", ItemID: "tacz:modern_kinetic_gun", Amount: 1, Fingerprint: strings.Repeat("a", 64), Savable: true}
	rows[1] = supplyInventorySlot{Slot: "storage:1", Name: "Vector「绯红禁域」", ItemID: "tacz:modern_kinetic_gun", Amount: 1, Fingerprint: strings.Repeat("b", 64), Savable: true}
	rows[9] = supplyInventorySlot{Slot: "storage:9", Name: "用于验证长名称的精炼强化材料样品", ItemID: "minecraft:amethyst_shard", Amount: 64, Fingerprint: strings.Repeat("c", 64), Savable: true}
	rows[10] = supplyInventorySlot{Slot: "storage:10", Name: "已初始化的储物背包", ItemID: "sophisticatedbackpacks:backpack", Amount: 1, Reason: "容器引用外部存储，不能复制"}
	return rows
}

func captureInput(f *supplyHTTPFixture, key string) supplyCaptureInput {
	return supplyCaptureInput{ClientRequestID: key, ServerID: "amiya", PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session", Slot: "storage:0", Fingerprint: strings.Repeat("a", 64)}
}

func TestSupplyInventoryCaptureRequiresAdminAndRejectsStaleSelection(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	f.request(t, "/api/v1/admin/supply/inventory?player=SupplyPlayer", f.playerToken, nil, 403)
	f.request(t, "/api/v1/admin/supply/inventory/capture", f.playerToken, captureInput(f, "forbidden"), 403)
	snapshot := f.request(t, "/api/v1/admin/supply/inventory?player=SupplyPlayer", f.adminToken, nil, 200)["data"].(map[string]any)
	if snapshot["playerUuid"] != f.player.ServerUUID || len(snapshot["slots"].([]any)) != 41 {
		t.Fatal(snapshot)
	}
	if strings.Contains(string(mustJSON(snapshot)), "payloadSha256") || strings.Contains(string(mustJSON(snapshot)), "serializeAsBytes") {
		t.Fatal("raw item representation leaked")
	}
	stale := captureInput(f, "stale-session")
	stale.SessionEpoch = "previous-session"
	f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, stale, 409)
	invalid := captureInput(f, "invalid-slot")
	invalid.Slot = "storage:99"
	f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, invalid, 400)
	f.mu.Lock()
	f.inventoryChanged = true
	f.mu.Unlock()
	result := f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, captureInput(f, "changed-item"), 409)
	if result["error"].(map[string]any)["code"] != "INVENTORY_CHANGED" {
		t.Fatal(result)
	}
	f.mu.Lock()
	captured := f.captures
	f.mu.Unlock()
	if captured != 0 {
		t.Fatal("rejected selection saved an item", captured)
	}
}

func TestSupplyInventoryCaptureReplayWorksAfterDisconnectAndDeduplicatesPublication(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	in := captureInput(f, "capture-once")
	first := f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, in, 200)["data"].(map[string]any)
	ref := first["itemRef"].(string)
	item, archived, err := f.store.CoreItem(context.Background(), ref, 1)
	if err != nil || archived || item.DisplayName != "QBZ-191「印花集」" {
		t.Fatal(item, archived, err)
	}
	op, err := f.store.CoreOperationByRequest(context.Background(), f.admin.ID, "inventory_"+in.ClientRequestID)
	if err != nil {
		t.Fatal(err)
	}
	// Lose the Go receipt after Core committed; recovery must use positive Core evidence.
	if _, err = f.store.DB.Exec("UPDATE core_operations SET state='UNKNOWN',result=NULL WHERE operation_id=?", op.ID); err != nil {
		t.Fatal(err)
	}
	f.app.Core.SetPresence("amiya", []byte(`{"players":[]}`))
	replayed := f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, in, 200)["data"].(map[string]any)
	if string(mustJSON(first)) != string(mustJSON(replayed)) {
		t.Fatal("capture replay changed item")
	}
	changed := in
	changed.Slot = "storage:1"
	changed.Fingerprint = strings.Repeat("b", 64)
	f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, changed, 409)
	f.mu.Lock()
	calls := f.captures
	f.mu.Unlock()
	if calls != 1 {
		t.Fatal("duplicate inventory capture", calls)
	}
	event := bridge.Envelope{EventID: "item_" + store.Digest([]byte(ref))[:32] + "_1", Type: "item.version.published", Payload: mustJSON(item)}
	payload, _, accepted, err := bridge.ValidateCore(f.app.Config, config.Node{ID: "amiya", ItemPrefix: "deuterium:"}, event)
	if err != nil {
		t.Fatal(err)
	}
	_, replay, err := f.store.CoreEvent(context.Background(), "amiya", event.EventID, event.Type, payload, nil, accepted)
	if err != nil || !replay {
		t.Fatal("Core outbox did not replay the indexed capture", replay, err)
	}
	var count int
	if err = f.store.DB.QueryRow("SELECT COUNT(*) FROM item_versions WHERE item_ref=?", ref).Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
}

func TestSupplyInventoryCaptureRetriesAnUnsentUnknownOperationWithItsOriginalID(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	in := captureInput(f, "unknown-before-capture")
	ctx := context.Background()
	payload := map[string]string{"playerUuid": in.PlayerUUID, "sessionEpoch": in.SessionEpoch, "slot": in.Slot, "fingerprint": in.Fingerprint, "actor": f.admin.ID}
	op, _, err := f.store.CreateCoreOperation(ctx, f.admin.ID, "inventory_"+in.ClientRequestID, in.ServerID, "items.capture", mustJSON(payload))
	if err != nil {
		t.Fatal(err)
	}
	if sent, err := f.store.MarkCoreSent(ctx, op.ID); err != nil || !sent {
		t.Fatal(sent, err)
	}
	if err = f.store.MarkCoreUnknown(ctx, op.ID); err != nil {
		t.Fatal(err)
	}
	got := f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, in, 200)["data"].(map[string]any)
	if got["itemRef"] != "deuterium:supply_"+store.Digest([]byte(op.ID))[:32] || got["revision"] != float64(1) {
		t.Fatal("retry changed operation or created another revision", got)
	}
	f.mu.Lock()
	calls := f.captures
	f.mu.Unlock()
	if calls != 1 {
		t.Fatal("unexpected capture count", calls)
	}
}

func TestSupplyInventoryCaptureRejectsClientItemPayloadAndUnpublishedNode(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	in := captureInput(f, "forged-bytes")
	data, _ := supplyRawObject(mustJSON(in))
	data["payload"] = "forged-item"
	f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, data, 400)
	f.app.Config.Nodes[0].ItemPrefix = ""
	f.request(t, "/api/v1/admin/supply/inventory?player=SupplyPlayer", f.adminToken, nil, 409)
	f.request(t, "/api/v1/admin/supply/inventory/capture", f.adminToken, in, 409)
}

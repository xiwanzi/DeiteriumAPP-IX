//go:build integration

package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestSupplyEditorLibraryImportsReusesAndArchivesOnlyEditorItems(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	node := f.app.Config.Nodes[0]
	const token = "isolated-supply-editor-test-only-token"
	f.app.Config.SupplyEditorNode = node.ID
	f.app.Config.SupplyEditorTokenSHA256 = store.Digest([]byte(token))
	input := supplyEditorSelection{"tacz:modern_kinetic_gun",
		`{id:"tacz:modern_kinetic_gun",count:1,components:{"minecraft:custom_data":{GunId:"tacz:scar_h"}}}`, "SCAR-H"}
	ref := input.reference(node.ItemPrefix)
	var calls atomic.Int32
	f.app.Core.Connect(node.ID, func(ctx context.Context, _ string, frame any) error {
		op, err := f.store.CoreOperation(ctx, frame.(map[string]any)["operationId"].(string))
		if err != nil {
			return err
		}
		calls.Add(1)
		var payload struct {
			supplyEditorSelection
			CatalogVersion int64 `json:"catalogVersion"`
		}
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		head := store.CoreCatalog{ItemRef: ref, LatestRevision: 1, CatalogVersion: payload.CatalogVersion + 1, Archived: op.Command == supplyEditorArchive}
		data := map[string]any{"head": head}
		if op.Command == supplyEditorImport {
			if payload.supplyEditorSelection != input {
				t.Error("registration changed the source ItemStack", payload.supplyEditorSelection)
			}
			data["item"] = store.ItemVersion{ItemRef: ref, Revision: 1, PayloadSHA256: strings.Repeat("a", 64),
				DisplayName: input.Name, MaxQuantity: 1, CompatibleServerIDs: []string{node.ID},
				Codec: "bukkit-bytes-v1", ItemID: input.ItemID, InventoryDomain: node.InventoryDomain,
				CompatibilityProfile: "youer"}
		}
		if err := f.app.acceptCoreReceiptV2(ctx, node, op, "COMPLETED", mustJSON(data)); err != nil {
			t.Errorf("editor Core receipt: %v", err)
			return err
		}
		reply := mustJSON(map[string]any{"operationId": op.ID, "status": "COMPLETED", "data": data})
		if err := f.store.CoreReply(ctx, node.ID, op.ID, "COMPLETED", reply); err != nil {
			return err
		}
		f.app.Core.Notify(op.ID)
		return nil
	})
	f.request(t, supplyEditorPath, f.adminToken, nil, 401)
	f.request(t, supplyEditorPath, f.nodeToken, input, 401)
	f.request(t, "/api/v1/admin/supply/pools", token, nil, 401)
	f.request(t, supplyEditorPath+"/archive", token, supplyEditorArchiveInput{"deuterium:printstream", 1}, 400)
	if _, err := f.store.SupplyEditorItem(ctx, node.ID, node.ItemPrefix+"gacha_", ref); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("looking up the unregistered item: %v", err)
	}

	first := f.request(t, supplyEditorPath, token, input, 200)["data"].(map[string]any)
	if first["itemRef"] != ref || first["revision"] != float64(1) || first["itemId"] != input.ItemID {
		t.Fatal(first)
	}
	f.request(t, supplyEditorPath, token, input, 200)
	if calls.Load() != 1 {
		t.Fatal("duplicate registration used another Core operation", calls.Load())
	}
	list := f.request(t, supplyEditorPath+"?query="+url.QueryEscape("scar-h"), token, nil, 200)["data"].(map[string]any)
	if len(list["items"].([]any)) != 1 || list["items"].([]any)[0].(map[string]any)["itemRef"] != ref ||
		list["items"].([]any)[0].(map[string]any)["stackData"] != input.StackData {
		t.Fatal(list)
	}
	all := f.request(t, supplyEditorPath, token, nil, 200)["data"].(map[string]any)
	if len(all["items"].([]any)) != 1 {
		t.Fatal("unrelated catalog items were returned", all)
	}

	pool := f.pool(t, "测试池", "STANDARD", "printstream")
	initial := pool.Definition
	withEditorItem := pool.Definition
	withEditorItem.Rewards = append(withEditorItem.Rewards[:0:0], withEditorItem.Rewards...)
	withEditorItem.Rewards[0].ItemRef = ref
	save := func(version int64, definitionName string, newReward bool) {
		t.Helper()
		definition := initial
		if newReward {
			definition = withEditorItem
		}
		_, err := f.store.ChangeSupplyPool(ctx, f.admin.ID, pool.ID, store.SupplyPoolChange{
			ClientRequestID: definitionName, Action: "save", ExpectedVersion: version, Definition: definition,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	save(pool.Version, "bind_editor_item", true)
	archive := supplyEditorArchiveInput{ref, 1}
	blocked := f.request(t, supplyEditorPath+"/archive", token, archive, 409)
	if blocked["error"].(map[string]any)["code"] != "ITEM_IN_USE" || calls.Load() != 1 {
		t.Fatal(blocked, calls.Load())
	}
	save(pool.Version+1, "unbind_editor_item", false)
	f.request(t, supplyEditorPath+"/archive", token, archive, 200)
	f.request(t, supplyEditorPath+"/archive", token, archive, 200)
	list = f.request(t, supplyEditorPath, token, nil, 200)["data"].(map[string]any)
	if len(list["items"].([]any)) != 0 || calls.Load() != 2 {
		t.Fatal(list, calls.Load())
	}
	if _, archived, err := f.store.CoreItem(ctx, ref, 1); err != nil || !archived {
		t.Fatal("the historical version must still exist", archived, err)
	}
	restored := f.request(t, supplyEditorPath, token, input, 200)["data"].(map[string]any)
	if restored["itemRef"] != ref || restored["revision"] != float64(1) || restored["catalogVersion"] != float64(3) {
		t.Fatal(restored)
	}
	f.request(t, supplyEditorPath+"/archive", token, archive, 409)
}

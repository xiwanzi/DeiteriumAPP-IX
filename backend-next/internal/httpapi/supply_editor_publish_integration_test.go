//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

func TestSupplyEditorPublishesAtomicImmutableCatalog(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	node := f.app.Config.Nodes[0]
	const token = "disposable-supply-editor-publisher-token"
	f.app.Config.SupplyEditorNode = node.ID
	f.app.Config.SupplyEditorTokenSHA256 = store.Digest([]byte(token))
	item := seedEditorPublicationItem(t, f, node)
	link := map[string]any{"itemRef": item.ItemRef, "revision": item.Revision, "payloadSha256": item.PayloadSHA256}
	art := editorPublicationArtwork(t, f, token)
	if res, err := f.http.Client().Get(f.http.URL + publicSupplyEditorArtworkPath + art); err != nil || res.StatusCode != 404 {
		t.Fatalf("draft artwork became public: %v %v", res, err)
	} else {
		res.Body.Close()
	}
	pool := func(id string, enabled bool) map[string]any {
		start := time.Now().UTC().Truncate(time.Second).Add(-time.Hour)
		return map[string]any{
			"id": id, "name": "编辑器测试池", "enabled": enabled, "currency": "LIMITED",
			"cover": "sha256:" + art, "thumbnail": "builtin:rail-up",
			"startsAt": start, "endsAt": start.Add(24 * time.Hour),
			"rules":            map[string]any{"mode": "EVENT", "activityId": id, "goldPpm": 5000, "legendaryPity": 10000},
			"featuredPrizeIds": []string{"featured"},
			"labels":           map[string]string{"rail": "侧栏", "single": "抽取一次"},
			"sections":         []map[string]string{{"title": "说明", "body": "原样冻结"}},
			"prizes": []map[string]any{
				{"id": "featured", "name": "SCAR-H", "itemId": item.ItemID, "gunId": "tacz:scar_h", "count": 1, "stars": 6, "probability": 0.5, "libraryItem": link},
				{"id": "normal", "name": "普通奖品", "itemId": item.ItemID, "gunId": "tacz:scar_h", "count": 1, "stars": 4, "probability": 99.5, "libraryItem": link},
			},
		}
	}
	catalog := func(revision int64, pools ...map[string]any) store.SupplyEditorPublication {
		return store.SupplyEditorPublication{Revision: revision, Catalog: mustJSON(map[string]any{
			"tickets": []map[string]any{{"id": "LIMITED", "enabled": true, "icon": "builtin:ticket"}}, "pools": pools,
		})}
	}
	valid := pool("editor_event", true)
	first := catalog(2, valid)
	manual := f.pool(t, "独立运营池", "STANDARD", "printstream")
	f.request(t, supplyEditorPublicationPath, f.adminToken, first, 401)
	initial := f.request(t, supplyEditorPublicationPath, token, first, 200)["data"].(map[string]any)
	result := initial["pools"].([]any)[0].(map[string]any)
	poolID := result["poolId"].(string)
	frozenRevision := int64(result["revision"].(float64))
	frozen, err := f.store.SupplyPoolRevision(ctx, poolID, frozenRevision)
	if err != nil || frozen.Rewards[0].ItemRef != item.ItemRef || frozen.Rewards[0].ItemRevision != item.Revision ||
		frozen.Rewards[0].Weight != 50 || frozen.EventPity != 10000 || frozen.EventGoldPPM != 5000 ||
		frozen.Labels.Rail != "侧栏" || frozen.Sections[0].Body != "原样冻结" || frozen.TicketArt != "builtin-ticket" {
		t.Fatal(frozen, err)
	}
	retry := f.request(t, supplyEditorPublicationPath, token, first, 200)["data"]
	if !bytes.Equal(mustJSON(initial), mustJSON(retry)) {
		t.Fatal("repeated release changed the receipt", retry)
	}
	if res, err := f.http.Client().Get(f.http.URL + publicSupplyEditorArtworkPath + art); err != nil || res.StatusCode != 200 {
		t.Fatalf("published artwork unavailable: %v %v", res, err)
	} else {
		res.Body.Close()
	}
	state := f.request(t, "/bridge/v1/supply/state", f.nodeToken, map[string]any{
		"playerUuid": f.player.ServerUUID, "sessionEpoch": "supply-session",
	}, 200)["data"].(map[string]any)
	if _, ok := state["catalog"].(map[string]any)["pools"].(map[string]any)[poolID+"_event"]; !ok {
		t.Fatal("the game state did not include the editor's publication", state)
	}
	storedItem, err := f.store.SupplyEditorItem(ctx, node.ID, node.ItemPrefix+"gacha_", item.ItemRef)
	if err != nil || storedItem.StackData != editorPublicationStack || storedItem.PayloadSHA256 != item.PayloadSHA256 {
		t.Fatal("the item library did not preserve the full selected stack", err)
	}
	if _, err = f.store.AdjustSupplyTickets(ctx, f.admin.ID, store.SupplyAdjustment{
		ClientRequestID: "editor_draw_fund", PlayerUUID: f.player.ServerUUID, Currency: "LIMITED",
		Action: "add", Amount: 10, Reason: "integration",
	}, true); err != nil {
		t.Fatal(err)
	}
	draw := f.request(t, "/bridge/v1/supply/draw", f.nodeToken, supplyGameRequest{
		PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session",
		Draw: &store.SupplyDrawRequest{
			ClientRequestID: "editor_draw", PoolID: poolID, Revision: frozenRevision, Scheme: "event", Quantity: 10,
		},
	}, 200)["data"].(map[string]any)
	weapons := draw["weapons"].(map[string]any)
	if rolls := draw["rolls"].([]any); len(rolls) != 10 || len(weapons) == 0 {
		t.Fatal("missing ten-draw presentations", draw)
	}
	for _, raw := range weapons {
		if weapon := raw.(map[string]any); weapon["modelKey"] != "tacz-scar_h" || weapon["gunId"] != "tacz:scar_h" {
			t.Fatal("the released gun could not address its client-side model", weapon)
		}
	}
	deliveries := f.request(t, "/bridge/v1/supply/deliveries", f.nodeToken, map[string]any{}, 200)["data"].(map[string]any)["items"].([]any)
	if len(deliveries) != 10 {
		t.Fatal("the node could not retrieve all ten deliveries", len(deliveries))
	}
	for _, raw := range deliveries {
		reward := raw.(map[string]any)["reward"].(map[string]any)
		if reward["itemRef"] != item.ItemRef || reward["itemRevision"] != float64(item.Revision) || reward["payloadSha256"] != item.PayloadSHA256 {
			t.Fatal("delivery changed the frozen item version", reward)
		}
	}

	bad := pool("bad_editor_event", true)
	bad["prizes"].([]map[string]any)[0]["libraryItem"] = nil
	// An invalid new pool must roll back both a previously published pool's pause and the rest of this release.
	f.request(t, supplyEditorPublicationPath, token, catalog(3, pool("editor_event", false), bad), 400)
	visible, err := f.store.SupplyPools(ctx, false)
	if err != nil || len(visible) != 2 {
		t.Fatal("failed publication changed active pools", visible, err)
	}
	f.request(t, supplyEditorPublicationPath, token, catalog(4, pool("editor_event", false)), 200)
	f.request(t, supplyEditorPublicationPath, token, catalog(3, valid), 409)
	visible, err = f.store.SupplyPools(ctx, false)
	if err != nil || len(visible) != 1 || visible[0].ID != manual.ID {
		t.Fatal("only editor-owned pools should be paused", visible, err)
	}
	if _, err := f.store.SupplyPoolRevision(ctx, poolID, frozenRevision); err != nil {
		t.Fatal("historical release was removed", err)
	}
}

func TestSupplyEditorTicketControlsQuoteIconAndIndependentBalance(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	node := f.app.Config.Nodes[0]
	const token = "disposable-supply-editor-ticket-token"
	const custom = "TICKET_00000000000000000000000000000001"
	f.app.Config.SupplyEditorNode = node.ID
	f.app.Config.SupplyEditorTokenSHA256 = store.Digest([]byte(token))
	item := seedEditorPublicationItem(t, f, node)
	link := map[string]any{"itemRef": item.ItemRef, "revision": item.Revision, "payloadSha256": item.PayloadSHA256}
	art := editorPublicationArtwork(t, f, token)
	pool := map[string]any{
		"id": "sun", "name": "日冕测试池", "enabled": true, "currency": custom,
		"cover": "builtin:pool-up", "thumbnail": "builtin:rail-up", "rules": map[string]any{"mode": "STANDARD"},
		"prizes": []map[string]any{
			{"id": "gold", "name": "六星奖品", "itemId": item.ItemID, "count": 1, "stars": 6, "probability": 50, "libraryItem": link},
			{"id": "normal", "name": "四星奖品", "itemId": item.ItemID, "count": 1, "stars": 4, "probability": 50, "libraryItem": link},
		},
	}
	catalog := func(revision int64, enabled bool) store.SupplyEditorPublication {
		return store.SupplyEditorPublication{Revision: revision, Catalog: mustJSON(map[string]any{
			"pools": []any{pool}, "tickets": []map[string]any{
				{"id": "STANDARD", "name": "常驻补给券", "enabled": true, "price": 100},
				{"id": custom, "name": "日冕观测券", "enabled": enabled, "icon": "sha256:" + art, "price": 12.34},
			},
		})}
	}
	result := f.request(t, supplyEditorPublicationPath, token, catalog(1, true), 200)["data"].(map[string]any)["pools"].([]any)[0].(map[string]any)
	poolID, revision := result["poolId"].(string), int64(result["revision"].(float64))
	state := func(capable bool) map[string]any {
		return f.request(t, "/bridge/v1/supply/state", f.nodeToken, map[string]any{
			"playerUuid": f.player.ServerUUID, "sessionEpoch": "supply-session", "ticketCatalog": capable,
		}, 200)["data"].(map[string]any)
	}
	if pools := state(false)["catalog"].(map[string]any)["pools"].(map[string]any); len(pools) != 0 {
		t.Fatal("legacy client received a pool with an unsupported ticket", pools)
	}
	current := state(true)
	settings := current["settings"].(map[string]any)
	if settings["standardPrice"] != float64(10000) || settings["limitedPrice"] != float64(0) {
		t.Fatal(settings)
	}
	page := current["catalog"].(map[string]any)["pools"].(map[string]any)[poolID+"_80"].(map[string]any)
	ticket := current["tickets"].(map[string]any)[custom].(map[string]any)
	icon := supply.EditorArtworkPrefix + art
	if page["currency"] != custom || page["ticketIcon"] != icon || ticket["name"] != "日冕观测券" || ticket["unitPrice"] != float64(1234) {
		t.Fatal(page, ticket)
	}
	if current["artwork"].(map[string]any)[icon] == nil {
		t.Fatal("published ticket icon cannot be downloaded by the player")
	}
	version := int64(settings["version"].(float64))
	request := store.SupplyExchangeRequest{ClientRequestID: "buy_custom", Currency: custom, Quantity: 3, SettingsVersion: version}
	buy := f.request(t, "/bridge/v1/supply/exchange", f.nodeToken, map[string]any{
		"playerUuid": f.player.ServerUUID, "sessionEpoch": "supply-session", "exchange": request,
	}, 200)["data"].(map[string]any)
	if buy["status"] != "COMPLETED" || buy["currency"] != custom || buy["totalPrice"] != float64(3702) {
		t.Fatal(buy)
	}
	order, err := f.store.SupplyExchange(ctx, buy["id"].(string), f.player.ServerUUID)
	if err != nil {
		t.Fatal(err)
	}
	settled := make(chan error, 2)
	for range 2 {
		go func() { settled <- f.store.CompleteSupplyExchange(ctx, order) }()
	}
	for range 2 {
		if err = <-settled; err != nil {
			t.Fatal(err)
		}
	}
	_, err = f.store.DrawSupply(ctx, f.player.ServerUUID, f.player.GameID, store.SupplyDrawRequest{
		ClientRequestID: "draw_custom", PoolID: poolID, Revision: revision, Scheme: "80", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	account := state(true)["account"].(map[string]any)
	if account["standard"] != float64(0) || account["limited"] != float64(0) || account["balances"].(map[string]any)[custom] != float64(2) {
		t.Fatal(account)
	}
	f.request(t, supplyEditorPublicationPath, token, catalog(2, false), 200)
	disabled := state(true)
	if len(disabled["catalog"].(map[string]any)["pools"].(map[string]any)) != 0 || disabled["account"].(map[string]any)["balances"].(map[string]any)[custom] != float64(2) {
		t.Fatal(disabled)
	}
	request.ClientRequestID = "buy_disabled_custom"
	if _, err = f.store.CreateSupplyExchange(ctx, f.player.ServerUUID, f.player.GameID, request); err == nil {
		t.Fatal("an older quote was accepted after the ticket was disabled")
	}
	request.SettingsVersion = int64(disabled["settings"].(map[string]any)["version"].(float64))
	if _, err = f.store.CreateSupplyExchange(ctx, f.player.ServerUUID, f.player.GameID, request); err == nil {
		t.Fatal("disabled ticket was offered for exchange")
	}
}

const editorPublicationStack = `{id:"tacz:modern_kinetic_gun",count:1,components:{"minecraft:custom_data":{GunId:"tacz:scar_h",GunCurrentAmmoCount:16},"minecraft:custom_model_data":77,"minecraft:lore":['{"text":"完整入库样品"}']}}`

func seedEditorPublicationItem(t *testing.T, f *supplyHTTPFixture, node config.Node) store.ItemVersion {
	t.Helper()
	ctx := context.Background()
	selection := supplyEditorSelection{ItemID: "tacz:modern_kinetic_gun", StackData: editorPublicationStack, Name: "SCAR-H"}
	ref := selection.reference(node.ItemPrefix)
	key := store.SupplyEditorOperationKey(node.ID, supplyEditorImport, ref, 0)
	op, _, err := f.store.CreateCoreOperation(ctx, store.SupplyEditorActor, key, node.ID, supplyEditorImport, mustJSON(selection))
	if err != nil {
		t.Fatal(err)
	}
	item := store.ItemVersion{ItemRef: ref, Revision: 1, PayloadSHA256: strings.Repeat("a", 64), DisplayName: selection.Name,
		MaxQuantity: 1, CompatibleServerIDs: []string{node.ID}, Codec: "bukkit-bytes-v1", ItemID: selection.ItemID,
		InventoryDomain: node.InventoryDomain, CompatibilityProfile: "youer"}
	err = f.app.acceptCoreReceiptV2(ctx, node, op, "COMPLETED", mustJSON(map[string]any{
		"item": item, "head": store.CoreCatalog{ItemRef: ref, CatalogVersion: 1, LatestRevision: 1},
	}))
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func editorPublicationArtwork(t *testing.T, f *supplyHTTPFixture, token string) string {
	t.Helper()
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 10, 8))); err != nil {
		t.Fatal(err)
	}
	hash := store.Digest(picture.Bytes())
	upload, err := http.NewRequest(http.MethodPut, f.http.URL+supplyEditorArtworkPath+hash, &picture)
	if err != nil {
		t.Fatal(err)
	}
	upload.Header.Set("Authorization", "Bearer "+token)
	upload.Header.Set("Content-Type", "image/png")
	response, err := f.http.Client().Do(upload)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatal(fmt.Errorf("editor artwork upload HTTP %d", response.StatusCode))
	}
	return hash
}

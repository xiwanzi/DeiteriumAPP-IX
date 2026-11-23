//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type supplyHTTPFixture struct {
	store            *store.Store
	app              *Server
	http             *httptest.Server
	admin            store.User
	player           store.User
	adminToken       string
	playerToken      string
	nodeToken        string
	funds            *commerceTestCore
	mu               sync.Mutex
	receipts         map[string]map[string]any
	captures         int
	inventoryChanged bool
	hideSettlement   bool
	hiddenSettlement string
}

func newSupplyHTTPFixture(t *testing.T, origin string) *supplyHTTPFixture {
	t.Helper()
	f := &supplyHTTPFixture{store: testdb.New(t), nodeToken: "supply-test-node-token-not-production-0123456789", funds: newCommerceTestCore(), receipts: map[string]map[string]any{}}
	for i, name := range []string{"SupplyAdmin", "SupplyPlayer"} {
		u := store.User{ID: fmt.Sprintf("supply_user_%d", i), PlayerRef: fmt.Sprintf("supply_player_%d", i), ServerUUID: fmt.Sprintf("11111111-2222-3333-4444-55555555555%d", i), GameID: name, Status: "active", QQ: fmt.Sprintf("3000000%d", i), PasswordHash: "fixture-only", IdentityStatus: "bound"}
		_, err := f.store.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,?,?,?,'fixture-only','active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, u.ID, u.PlayerRef, u.ServerUUID, u.GameID, u.QQ, strings.Repeat("0", 64))
		if err != nil {
			t.Fatal(err)
		}
		token := identity.Secret()
		if err = f.store.CreateSession(context.Background(), u, store.Digest([]byte(token)), "app", "", time.Now().Add(time.Hour), ""); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			f.admin = u
			f.adminToken = token
		} else {
			f.player = u
			f.playerToken = token
		}
	}
	if err := f.store.Grant(context.Background(), f.admin.ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.http = httptest.NewUnstartedServer(http.NotFoundHandler())
	if origin == "" {
		origin = "http://" + f.http.Listener.Addr().String()
	}
	node := config.Node{ID: "amiya", ItemPrefix: "deuterium:", Chat: true, Economy: true, ClaimEnabled: true, InventoryDomain: "survival", MailCluster: "supply-fixture", TokenSHA256: store.Digest([]byte(f.nodeToken))}
	f.app = New(f.store, config.Config{Development: true, PublicOrigin: origin, Nodes: []config.Node{node}})
	f.app.Core.Connect(node.ID, func(ctx context.Context, _ string, frame any) error { return f.coreReply(ctx, node, frame) })
	presence, _ := json.Marshal(map[string]any{"players": []map[string]string{{"playerUuid": f.player.ServerUUID, "gameId": f.player.GameID, "sessionEpoch": "supply-session"}, {"playerUuid": f.admin.ServerUUID, "gameId": f.admin.GameID, "sessionEpoch": "admin-session"}}})
	f.app.Core.SetPresence(node.ID, presence)
	f.app.Core.SetStatus(node.ID, []byte(`{"storageHealthy":true,"economy":true,"economyAuthority":true,"mailbox":{"available":true,"commerceReady":true,"clusterId":"supply-fixture","apiVersion":3,"missingDeliveryCancellation":true}}`))
	f.http.Config.Handler = f.app.Handler()
	f.http.Start()
	t.Cleanup(func() { f.app.Close(); f.http.Close() })
	for _, key := range []string{"printstream", "scarlet", "tacz-ak47"} {
		item := store.ItemVersion{ItemRef: "deuterium:" + key, Revision: 1, PayloadSHA256: strings.Repeat("a", 64), DisplayName: map[string]string{"printstream": "QBZ-191「印花集」", "scarlet": "Vector「绯红禁域」", "tacz-ak47": "AKM"}[key], MaxQuantity: 1, CompatibleServerIDs: []string{"amiya"}, Codec: "bukkit-bytes-v1", ItemID: "tacz:modern_kinetic_gun", InventoryDomain: "survival", CompatibilityProfile: "youer"}
		raw, _ := json.Marshal(item)
		if _, err := f.store.DB.Exec("INSERT INTO item_versions(item_ref,revision,publisher_node,payload_sha256,metadata,fingerprint,created_at) VALUES(?,1,'amiya',?,?,?,UTC_TIMESTAMP(6))", item.ItemRef, item.PayloadSHA256, raw, strings.Repeat("b", 64)); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *supplyHTTPFixture) coreReply(ctx context.Context, node config.Node, frame any) error {
	raw, _ := json.Marshal(frame)
	var request struct {
		OperationID string         `json:"operationId"`
		Command     string         `json:"command"`
		Payload     map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		return err
	}
	var data any
	var replyError map[string]string
	status := "COMPLETED"
	switch request.Command {
	case "items.inventory":
		data = map[string]any{"playerUuid": request.Payload["playerUuid"], "sessionEpoch": request.Payload["sessionEpoch"], "slots": supplyFixtureInventory()}
	case "items.capture":
		f.mu.Lock()
		changed := f.inventoryChanged
		f.mu.Unlock()
		var selected *supplyInventorySlot
		for _, slot := range supplyFixtureInventory() {
			if slot.Slot == request.Payload["slot"] && slot.Savable && slot.Fingerprint == request.Payload["fingerprint"] {
				copy := slot
				selected = &copy
			}
		}
		if changed || selected == nil {
			status = "FAILED"
			replyError = map[string]string{"code": "INVENTORY_CHANGED", "message": "该槽位的物品已变化，请刷新背包后重新选择。"}
		} else {
			f.mu.Lock()
			f.captures++
			f.mu.Unlock()
			item := store.ItemVersion{ItemRef: "deuterium:supply_" + store.Digest([]byte(request.OperationID))[:32], Revision: 1, PayloadSHA256: strings.Repeat("a", 64), DisplayName: selected.Name, MaxQuantity: 99999, CompatibleServerIDs: []string{"amiya"}, Codec: "bukkit-bytes-v1", ItemID: selected.ItemID, InventoryDomain: "survival", CompatibilityProfile: "youer"}
			data = map[string]any{"item": item}
		}
	case "wallet.balance":
		balance := int64(10000000)
		f.funds.mu.Lock()
		for _, entry := range f.funds.ledger {
			if entry.payer == request.Payload["playerUuid"] {
				balance -= entry.reserved - entry.refunded
			}
		}
		f.funds.mu.Unlock()
		data = map[string]any{"amount": supplyMoney(balance), "currency": "CREDIT", "heldAmount": "0.00", "heldBreakdown": []any{}}
	case "operation.query":
		id, _ := request.Payload["operationId"].(string)
		f.mu.Lock()
		receipt := f.receipts[id]
		hidden := f.hideSettlement && f.hiddenSettlement == id
		f.mu.Unlock()
		if hidden {
			data = map[string]any{"state": "UNKNOWN", "result": nil}
		} else if receipt == nil {
			data = map[string]any{"state": "NOT_FOUND", "result": nil}
		} else {
			encoded, _ := json.Marshal(receipt)
			data = map[string]any{"state": receipt["status"], "result": string(encoded)}
		}
	default:
		result, err := f.funds.Execute(ctx, "fixture-node", request.OperationID, request.Command, request.Payload)
		if err != nil {
			return err
		}
		status = result.Status
		copyJSON, _ := json.Marshal(result.Data)
		var copyData map[string]any
		json.Unmarshal(copyJSON, &copyData)
		if copyData != nil {
			copyData["operationId"] = request.OperationID
		}
		data = copyData
	}
	reply := map[string]any{"operationId": request.OperationID, "status": status, "data": data}
	if replyError != nil {
		reply["error"] = replyError
	}
	if request.Command == "wallet.escrow.settle" {
		f.mu.Lock()
		if f.hideSettlement {
			f.hiddenSettlement = request.OperationID
			f.receipts[request.OperationID] = reply
			f.mu.Unlock()
			return nil
		}
		f.mu.Unlock()
	}
	encoded, _ := json.Marshal(reply)
	if err := f.app.acceptCoreReceiptV2(ctx, node, store.CoreOperation{ID: request.OperationID, Command: request.Command, Payload: mustJSON(request.Payload)}, status, mustJSON(data)); err != nil {
		return err
	}
	if err := f.store.CoreReply(ctx, node.ID, request.OperationID, status, encoded); err != nil {
		return err
	}
	f.mu.Lock()
	f.receipts[request.OperationID] = reply
	f.mu.Unlock()
	f.app.Core.Notify(request.OperationID)
	return nil
}

func mustJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func (f *supplyHTTPFixture) request(t *testing.T, path, token string, input any, want int) map[string]any {
	t.Helper()
	method := "POST"
	var body *bytes.Reader
	if input == nil {
		method = "GET"
		body = bytes.NewReader(nil)
	} else {
		body = bytes.NewReader(mustJSON(input))
	}
	r, _ := http.NewRequest(method, f.http.URL+path, body)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	if token == f.nodeToken {
		r.Header.Set("X-Deuterium-Node-ID", "amiya")
	}
	response, err := f.http.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var data map[string]any
	if err = json.NewDecoder(response.Body).Decode(&data); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != want {
		t.Fatalf("%s: got %d want %d: %v", path, response.StatusCode, want, data)
	}
	return data
}

func (f *supplyHTTPFixture) pool(t *testing.T, name, kind, model string) store.SupplyPool {
	t.Helper()
	start, end := time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(48*time.Hour)
	d := supply.Definition{Name: name, Title: name, Kind: kind, StartsAt: &start, EndsAt: &end, Art: supply.Artwork{Hero: "builtin-hero-up", Rail: "builtin-rail-up", Theme: "violet"}}
	if model == "scarlet" {
		d.Art = supply.Artwork{Hero: "builtin-hero-vector", Rail: "builtin-rail-vector", Theme: "red"}
	}
	if kind == "STANDARD" {
		d.EndsAt = nil
		d.Art = supply.Artwork{Hero: "builtin-hero-standard", Rail: "builtin-rail-standard", Theme: "blue"}
	}
	if kind == "EVENT" {
		d.EventGoldPPM = 10000
		d.EventPity = 80
	}
	for i, key := range []string{model, "tacz-ak47"} {
		stars := 4
		if i == 0 {
			stars = 6
		}
		d.Rewards = append(d.Rewards, supply.Reward{Key: key, Name: map[string]string{"printstream": "QBZ-191「印花集」", "scarlet": "Vector「绯红禁域」", "tacz-ak47": "AKM"}[key], ModelKey: key, Stars: stars, Weight: 1, UP: stars == 6, ItemRef: "deuterium:" + key, ItemRevision: 1, PayloadSHA256: strings.Repeat("a", 64), InventoryDomain: "survival", Servers: []string{"amiya"}, Art: supply.Artwork{Detail: "builtin-detail-" + key, Fallback: "builtin-fallback-" + key}})
	}
	raw, err := f.store.ChangeSupplyPool(context.Background(), f.admin.ID, "", store.SupplyPoolChange{ClientRequestID: store.ID("pool_"), Action: "save", Definition: d})
	if err != nil {
		t.Fatal(err)
	}
	var p store.SupplyPool
	json.Unmarshal(raw, &p)
	raw, err = f.store.ChangeSupplyPool(context.Background(), f.admin.ID, p.ID, store.SupplyPoolChange{ClientRequestID: store.ID("publish_"), Action: "publish", ExpectedVersion: p.Version})
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(raw, &p)
	return p
}

func TestSupplyAdminAndBridgeEnforceDifferentAuthorities(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	f.request(t, "/api/v1/admin/supply/pools", f.playerToken, nil, 403)
	p := f.pool(t, "墨影流光", "LIMITED", "printstream")
	f.request(t, "/api/v1/admin/supply/pools", f.adminToken, nil, 200)
	in := supplyGameRequest{PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session"}
	f.request(t, "/bridge/v1/supply/state", f.playerToken, in, 401)
	in.SessionEpoch = "stale"
	f.request(t, "/bridge/v1/supply/state", f.nodeToken, in, 401)
	in.SessionEpoch = "supply-session"
	out := f.request(t, "/bridge/v1/supply/state", f.nodeToken, in, 200)
	encoded := string(mustJSON(out))
	if strings.Contains(encoded, "payloadSha256") || strings.Contains(encoded, "\"target\"") {
		t.Fatal("public state leaked private delivery or random target")
	}
	if !strings.Contains(encoded, p.ID+"_80") || !strings.Contains(encoded, p.ID+"_120") || !strings.Contains(encoded, "12000") {
		t.Fatal("missing paired rules", encoded)
	}
}

func TestSupplyExchangeWorkerCommitsCreditAndTicketsOnce(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	if _, err := f.store.SaveSupplySettings(ctx, f.admin.ID, "prices", store.SupplySettings{Version: 1, StandardPrice: 10000, LimitedPrice: 10000}); err != nil {
		t.Fatal(err)
	}
	raw, err := f.store.CreateSupplyExchange(ctx, f.player.ServerUUID, f.player.GameID, store.SupplyExchangeRequest{ClientRequestID: "exchange", Currency: "LIMITED", Quantity: 3, SettingsVersion: 2})
	if err != nil {
		t.Fatal(err)
	}
	var x store.SupplyExchange
	json.Unmarshal(raw, &x)
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		x, err = f.store.SupplyExchange(ctx, x.ID, x.PlayerUUID)
		if err != nil {
			t.Fatal(err)
		}
		if x.Status == "COMPLETED" {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if x.Status != "COMPLETED" {
		t.Fatal(x)
	}
	account, err := f.store.SupplyAccount(ctx, f.player.ServerUUID)
	if err != nil || account.Limited != 3 {
		t.Fatal(account, err)
	}
	f.funds.mu.Lock()
	held := *f.funds.ledger[x.ID]
	f.funds.mu.Unlock()
	if held.reserved != 30000 || held.settled != 30000 {
		t.Fatal(held)
	}
	if err = f.store.CompleteSupplyExchange(ctx, x); err != nil {
		t.Fatal(err)
	}
	account, _ = f.store.SupplyAccount(ctx, f.player.ServerUUID)
	if account.Limited != 3 {
		t.Fatal("duplicate credit", account)
	}
}

func TestSupplyExchangeConcurrentRequestsReturnTheSameSettledReceipt(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	if _, err := f.store.SaveSupplySettings(ctx, f.admin.ID, "prices", store.SupplySettings{
		Version: 1, LimitedPrice: 10000,
	}); err != nil {
		t.Fatal(err)
	}
	request := supplyGameRequest{
		PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session",
		Exchange: &store.SupplyExchangeRequest{ClientRequestID: "immediate_exchange", Currency: "LIMITED", Quantity: 3, SettingsVersion: 2},
	}
	path := "/bridge/v1/supply/exchange"
	payload := mustJSON(request)
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() {
			<-start
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
			req.Header.Set("Authorization", "Bearer "+f.nodeToken)
			req.Header.Set("X-Deuterium-Node-ID", "amiya")
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			f.http.Config.Handler.ServeHTTP(res, req)
			results <- res
		}()
	}
	started := time.Now()
	close(start)
	var orderID string
	for range 2 {
		res := <-results
		var reply struct {
			Data store.SupplyExchange `json:"data"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &reply); err != nil || res.Code != 200 ||
			reply.Data.ID == "" || reply.Data.Status != "COMPLETED" || (orderID != "" && orderID != reply.Data.ID) {
			t.Fatalf("concurrent exchange request did not return the same terminal receipt: %d %s", res.Code, res.Body)
		}
		orderID = reply.Data.ID
	}
	t.Logf("both exchange requests completed with simulated economy in %s", time.Since(started))
	f.funds.mu.Lock()
	held := *f.funds.ledger[orderID]
	reserves, settlements := f.funds.calls["wallet.escrow.reserve"], f.funds.calls["wallet.escrow.settle"]
	f.funds.mu.Unlock()
	account, err := f.store.SupplyAccount(ctx, f.player.ServerUUID)
	if err != nil || account.Limited != 3 || held.reserved != 30000 || held.settled != 30000 || reserves != 1 || settlements != 1 {
		t.Fatal("expected one reservation, settlement and credit", account, held, reserves, settlements, err)
	}
}

func TestSupplyExchangeRecoversTheOriginalCommittedSettlement(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	if _, err := f.store.SaveSupplySettings(ctx, f.admin.ID, "prices", store.SupplySettings{
		Version: 1, LimitedPrice: 10000,
	}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.hideSettlement = true
	f.mu.Unlock()
	path := "/bridge/v1/supply/exchange"
	request := supplyGameRequest{
		PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session",
		Exchange: &store.SupplyExchangeRequest{ClientRequestID: "delayed_settlement", Currency: "LIMITED", Quantity: 2, SettingsVersion: 2},
	}
	first := f.request(t, path, f.nodeToken, request, 200)["data"].(map[string]any)
	if first["status"] != "RESERVED" || first["account"].(map[string]any)["limited"] != float64(0) {
		t.Fatal("the exchange was credited before its settlement receipt", first)
	}
	f.mu.Lock()
	hidden := f.hiddenSettlement
	f.hideSettlement = false
	f.mu.Unlock()
	if hidden == "" {
		t.Fatal("the first request did not reach the economy settlement")
	}

	deadline := time.Now().Add(4 * time.Second)
	var recovered map[string]any
	for time.Now().Before(deadline) {
		recovered = f.request(t, path, f.nodeToken, request, 200)["data"].(map[string]any)
		if recovered["status"] == "COMPLETED" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if recovered["status"] != "COMPLETED" || recovered["id"] != first["id"] ||
		recovered["account"].(map[string]any)["limited"] != float64(2) {
		t.Fatal("the original settlement did not resume", first, recovered)
	}
	f.funds.mu.Lock()
	reserves, settlements := f.funds.calls["wallet.escrow.reserve"], f.funds.calls["wallet.escrow.settle"]
	f.funds.mu.Unlock()
	if reserves != 1 || settlements != 1 {
		t.Fatal("recovery repeated a money operation", reserves, settlements)
	}
}

func TestSupplyDeliveryCannotSwitchToMailAfterUncertainInventory(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	p := f.pool(t, "墨影流光", "LIMITED", "printstream")
	_, err := f.store.AdjustSupplyTickets(ctx, f.admin.ID, store.SupplyAdjustment{ClientRequestID: "fund", PlayerUUID: f.player.ServerUUID, Currency: "LIMITED", Action: "add", Amount: 1, Reason: "integration"}, true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.store.DrawSupply(ctx, f.player.ServerUUID, f.player.GameID, store.SupplyDrawRequest{ClientRequestID: "draw", PoolID: p.ID, Revision: p.PublishedRevision, Scheme: "80", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	var order store.SupplyOrder
	json.Unmarshal(raw, &order)
	detail, err := f.store.SupplyOrder(ctx, order.ID, f.player.ServerUUID)
	if err != nil {
		t.Fatal(err)
	}
	d := detail["deliveries"].([]store.SupplyDelivery)[0]
	d, err = f.store.ClaimSupplyDelivery(ctx, d.ID, "amiya", f.player.ServerUUID, "supply-session")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.BeginSupplyDelivery(ctx, d.ID, "amiya", d.SessionEpoch, "wrong-token", "inventory-epoch"); err == nil {
		t.Fatal("wrong token began grant")
	}
	if err = f.store.BeginSupplyDelivery(ctx, d.ID, "amiya", d.SessionEpoch, d.Token, "inventory-epoch"); err != nil {
		t.Fatal(err)
	}
	result := store.SupplyDeliveryResult{ID: d.ID, PlayerUUID: d.PlayerUUID, Token: d.Token, SessionEpoch: d.SessionEpoch, State: "REVIEW", Message: "save uncertain"}
	if _, err = f.store.FinishSupplyDelivery(ctx, "amiya", result); err != nil {
		t.Fatal(err)
	}
	result.State = "MAIL_PENDING"
	result.Message = "INVENTORY_FULL"
	if _, err = f.store.FinishSupplyDelivery(ctx, "amiya", result); err == nil {
		t.Fatal("uncertain write mailed again")
	}
	if _, err = f.store.RetrySupplyDelivery(ctx, f.admin.ID, "retry", d.ID, "operator retry"); err == nil {
		t.Fatal("review allowed blind retry")
	}
	result.State = "INVENTORY"
	result.Message = ""
	result.InventoryDomain = "survival"
	result.InventoryEpoch = "wrong"
	result.SaveReceipt = "proof"
	result.CommittedAt = time.Now().UnixMilli()
	if _, err = f.store.FinishSupplyDelivery(ctx, "amiya", result); err == nil {
		t.Fatal("wrong inventory owner accepted")
	}
	result.InventoryEpoch = "inventory-epoch"
	if _, err = f.store.FinishSupplyDelivery(ctx, "amiya", result); err != nil {
		t.Fatal(err)
	}
}

func TestSupplyDeliveryWaitsForTheOwnersReveal(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	pool := f.pool(t, "演出测试", "LIMITED", "printstream")
	if _, err := f.store.AdjustSupplyTickets(ctx, f.admin.ID, store.SupplyAdjustment{
		ClientRequestID: "reveal_fund", PlayerUUID: f.player.ServerUUID, Currency: "LIMITED",
		Action: "add", Amount: 10, Reason: "integration",
	}, true); err != nil {
		t.Fatal(err)
	}
	request := supplyGameRequest{PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session", AwaitReveal: true,
		Draw: &store.SupplyDrawRequest{ClientRequestID: "reveal_draw", PoolID: pool.ID, Revision: pool.PublishedRevision, Scheme: "80", Quantity: 10}}
	result := f.request(t, "/bridge/v1/supply/draw", f.nodeToken, request, 200)["data"].(map[string]any)
	orderID := result["id"].(string)
	if result["awaitReveal"] != true || len(result["rolls"].([]any)) != 10 {
		t.Fatal("wanted a revealed outcome with delivery held", result)
	}
	list := func() []any {
		return f.request(t, "/bridge/v1/supply/deliveries", f.nodeToken, map[string]any{}, 200)["data"].(map[string]any)["items"].([]any)
	}
	if len(list()) != 0 {
		t.Fatal("the inventory queue pulled a reward before the client finished the reveal")
	}
	if _, err := f.store.ClaimSupplyDelivery(ctx, orderID+"_01", "amiya", f.player.ServerUUID, request.SessionEpoch); err == nil {
		t.Fatal("a direct inventory claim bypassed the reveal")
	}
	if err := f.store.RouteOfflineSupplyDelivery(ctx, orderID+"_01"); err != nil {
		t.Fatal(err)
	}
	delivery, err := f.store.SupplyDelivery(ctx, orderID+"_01")
	if err != nil || delivery.State != "PENDING" {
		t.Fatal("offline routing bypassed the reveal hold", delivery.State, err)
	}
	ack := supplyGameRequest{PlayerUUID: f.admin.ServerUUID, SessionEpoch: "admin-session", OrderID: orderID}
	f.request(t, "/bridge/v1/supply/reveal-complete", f.nodeToken, ack, 404)
	ack.PlayerUUID = f.player.ServerUUID
	ack.SessionEpoch = "stale"
	f.request(t, "/bridge/v1/supply/reveal-complete", f.nodeToken, ack, 401)
	if len(list()) != 0 {
		t.Fatal("an unrelated player or stale session released the order")
	}

	ack.SessionEpoch = request.SessionEpoch
	for range 2 {
		f.request(t, "/bridge/v1/supply/reveal-complete", f.nodeToken, ack, 200)
	}
	if items := list(); len(items) != 10 {
		t.Fatal("the completion event must release the original ten rewards", len(items))
	}
	request.AwaitReveal = false
	replay := f.request(t, "/bridge/v1/supply/draw", f.nodeToken, request, 200)["data"].(map[string]any)
	if replay["id"] != orderID {
		t.Fatal("the transport preference changed the draw fingerprint", replay)
	}
	var count, spent int
	if err = f.store.DB.QueryRow("SELECT COUNT(*),SUM(amount) FROM supply_ledger WHERE business_id=? AND business_type='DRAW'", orderID).Scan(&count, &spent); err != nil || count != 1 || spent != -10 {
		t.Fatal("the draw and reveal must make one debit", count, spent, err)
	}
}

func TestSupplyInterruptedRevealRoutesToMailWithoutAnInventoryClaim(t *testing.T) {
	f := newSupplyHTTPFixture(t, "")
	ctx := context.Background()
	pool := f.pool(t, "掉线测试", "LIMITED", "printstream")
	if _, err := f.store.AdjustSupplyTickets(ctx, f.admin.ID, store.SupplyAdjustment{
		ClientRequestID: "timeout_fund", PlayerUUID: f.player.ServerUUID, Currency: "LIMITED",
		Action: "add", Amount: 1, Reason: "integration",
	}, true); err != nil {
		t.Fatal(err)
	}
	result := f.request(t, "/bridge/v1/supply/draw", f.nodeToken, supplyGameRequest{
		PlayerUUID: f.player.ServerUUID, SessionEpoch: "supply-session", AwaitReveal: true,
		Draw: &store.SupplyDrawRequest{ClientRequestID: "timeout_draw", PoolID: pool.ID, Revision: pool.PublishedRevision, Scheme: "80", Quantity: 1},
	}, 200)["data"].(map[string]any)
	orderID := result["id"].(string)
	if _, err := f.store.DB.Exec("UPDATE supply_deliveries SET reveal_until=UTC_TIMESTAMP(6)-INTERVAL 1 SECOND WHERE order_id=? AND reveal_until IS NOT NULL", orderID); err != nil {
		t.Fatal(err)
	}
	if err := f.store.MailExpiredSupplyReveals(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ReleaseSupplyReveal(ctx, orderID, f.player.ServerUUID); err != nil {
		t.Fatal(err)
	}
	delivery, err := f.store.SupplyDelivery(ctx, orderID+"_01")
	if err != nil || (delivery.State != "MAIL_PENDING" && delivery.State != "MAIL") || delivery.Token != "" {
		t.Fatal("recovery or a delayed acknowledgement exposed an inventory item", delivery, err)
	}
	if items, err := f.store.PendingSupplyDeliveries(ctx, []string{f.player.ServerUUID}); err != nil || len(items) != 0 {
		t.Fatal("the inventory worker saw an expired reveal", items, err)
	}
}

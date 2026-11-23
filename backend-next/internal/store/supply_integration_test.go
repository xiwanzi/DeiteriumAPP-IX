//go:build integration

package store_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

const supplyPlayer = "11111111-2222-3333-4444-555555555555"

func supplyFixture(t *testing.T) (*store.Store, string) {
	t.Helper()
	s := testdb.New(t)
	actor := "supply_admin"
	_, err := s.DB.Exec(`INSERT INTO identities(id,player_ref,server_uuid,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint) VALUES(?,?,'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee','SupplyAdmin','10000000','unused','active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)`, actor, "player_supply", strings.Repeat("0", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO identity_permissions VALUES(?,'platform.admin')", actor); err != nil {
		t.Fatal(err)
	}
	item := store.ItemVersion{ItemRef: "deuterium:supply_test", Revision: 1, PayloadSHA256: strings.Repeat("a", 64), DisplayName: "Supply test", MaxQuantity: 1, CompatibleServerIDs: []string{"amiya"}, Codec: "bukkit-bytes-v1", ItemID: "tacz:modern_kinetic_gun", InventoryDomain: "survival", CompatibilityProfile: "youer"}
	metadata, _ := json.Marshal(item)
	if _, err = s.DB.Exec("INSERT INTO item_versions(item_ref,revision,publisher_node,payload_sha256,metadata,fingerprint,created_at) VALUES(?,1,'amiya',?,?,?,UTC_TIMESTAMP(6))", item.ItemRef, item.PayloadSHA256, metadata, strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
	return s, actor
}

func supplyDefinition(kind string) supply.Definition {
	start, end := time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour)
	d := supply.Definition{Name: "测试补给", Title: "补给测试", Kind: kind, StartsAt: &start, EndsAt: &end, Art: supply.Artwork{Hero: "builtin-hero-up", Rail: "builtin-rail-up", Theme: "violet"}, Rewards: []supply.Reward{}}
	for i, stars := range []int{6, 4} {
		d.Rewards = append(d.Rewards, supply.Reward{Key: fmt.Sprintf("weapon%d", i), Name: "测试武器", Stars: stars, UP: stars == 6, Weight: 1, ItemRef: "deuterium:supply_test", ItemRevision: 1, PayloadSHA256: strings.Repeat("a", 64), InventoryDomain: "survival", Servers: []string{"amiya"}})
	}
	if kind == "STANDARD" {
		d.EndsAt = nil
	}
	if kind == "EVENT" {
		d.EventGoldPPM = 5000
		d.EventPity = 80
	}
	return d
}

func supplyPublish(t *testing.T, s *store.Store, actor, key string, d supply.Definition) store.SupplyPool {
	t.Helper()
	ctx := context.Background()
	raw, err := s.ChangeSupplyPool(ctx, actor, "", store.SupplyPoolChange{ClientRequestID: key, Action: "save", Definition: d})
	if err != nil {
		t.Fatal(err)
	}
	var pool store.SupplyPool
	if err = json.Unmarshal(raw, &pool); err != nil {
		t.Fatal(err)
	}
	raw, err = s.ChangeSupplyPool(ctx, actor, pool.ID, store.SupplyPoolChange{ClientRequestID: key + "_publish", ExpectedVersion: pool.Version, Action: "publish"})
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func supplyFund(t *testing.T, s *store.Store, actor string, amount int64) {
	t.Helper()
	_, err := s.AdjustSupplyTickets(context.Background(), actor, store.SupplyAdjustment{ClientRequestID: store.ID("fund_"), PlayerUUID: supplyPlayer, Currency: "LIMITED", Action: "add", Amount: amount, Reason: "本地集成测试"}, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSupplyDeliveryPollingReachesRewardsBeyondAnUnresolvedBatch(t *testing.T) {
	s, actor := supplyFixture(t)
	ctx := context.Background()
	pool := supplyPublish(t, s, actor, "rotation", supplyDefinition("LIMITED"))
	supplyFund(t, s, actor, 40)
	for i := 0; i < 4; i++ {
		if _, err := s.DrawSupply(ctx, supplyPlayer, "SupplyPlayer", store.SupplyDrawRequest{ClientRequestID: fmt.Sprintf("rotate-%d", i), PoolID: pool.ID, Revision: pool.PublishedRevision, Scheme: "80", Quantity: 10}); err != nil {
			t.Fatal(err)
		}
	}
	for _, state := range []string{"REVIEW", "MAIL_PENDING"} {
		if _, err := s.DB.Exec("UPDATE supply_deliveries SET state=?,updated_at='2000-01-01'", state); err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for poll := 0; poll < 3; poll++ {
			var batch []store.SupplyDelivery
			var err error
			if state == "REVIEW" {
				batch, err = s.PendingSupplyDeliveries(ctx, []string{supplyPlayer})
			} else {
				batch, err = s.SupplyDeliveryQueue(ctx, state, 16)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range batch {
				if row.State != state {
					t.Fatalf("polling changed delivery state: %s", row.State)
				}
				seen[row.ID] = true
			}
		}
		if len(seen) != 40 {
			t.Fatalf("%s queue starved later rewards: reached %d/40", state, len(seen))
		}
	}
	account, err := s.SupplyAccount(ctx, supplyPlayer)
	if err != nil || account.Limited != 0 {
		t.Fatal("polling changed the committed ticket debit", account.Limited, err)
	}
}

func TestSupplyAtomicDrawReplayAndCrossPoolPity(t *testing.T) {
	s, actor := supplyFixture(t)
	ctx := context.Background()
	a := supplyPublish(t, s, actor, "a", supplyDefinition("LIMITED"))
	b := supplyPublish(t, s, actor, "b", supplyDefinition("LIMITED"))
	supplyFund(t, s, actor, 20)
	if _, err := s.DB.Exec("INSERT INTO supply_pity VALUES(?,'limited:80',79,80,0)", supplyPlayer); err != nil {
		t.Fatal(err)
	}
	in := store.SupplyDrawRequest{ClientRequestID: "same_draw", PoolID: b.ID, Revision: b.PublishedRevision, Scheme: "80", Quantity: 1}
	raw, err := s.DrawSupply(ctx, supplyPlayer, "Player", in)
	if err != nil {
		t.Fatal(err)
	}
	var order store.SupplyOrder
	if err = json.Unmarshal(raw, &order); err != nil {
		t.Fatal(err)
	}
	if len(order.Rolls) != 1 || !order.Rolls[0].Guaranteed || order.Rolls[0].Reward.Stars != 6 || order.Rolls[0].After != 0 {
		t.Fatal(order)
	}
	if _, err = s.ChangeSupplyPool(ctx, actor, b.ID, store.SupplyPoolChange{ClientRequestID: "pause_b", Action: "pause", ExpectedVersion: b.Version}); err != nil {
		t.Fatal(err)
	}
	replay, err := s.DrawSupply(ctx, supplyPlayer, "RenamedPlayer", in)
	if err != nil || string(replay) != string(raw) {
		t.Fatal("closed-pool replay changed", err)
	}
	in.PoolID = a.ID
	in.Revision = a.PublishedRevision
	if _, err = s.DrawSupply(ctx, supplyPlayer, "Player", in); err == nil {
		t.Fatal("same key accepted changed pool")
	}
	account, err := s.SupplyAccount(ctx, supplyPlayer)
	if err != nil || account.Limited != 19 || account.Pity["limited:80"].Count != 0 {
		t.Fatal(account, err)
	}
	var deliveries int
	if err = s.DB.QueryRow("SELECT COUNT(*) FROM supply_deliveries").Scan(&deliveries); err != nil || deliveries != 1 {
		t.Fatal(deliveries, err)
	}
}

func TestSupplyConcurrentRequestsNeverOverdrawAndSameRequestOnlyChargesOnce(t *testing.T) {
	s, actor := supplyFixture(t)
	pool := supplyPublish(t, s, actor, "a", supplyDefinition("LIMITED"))
	supplyFund(t, s, actor, 10)
	ctx := context.Background()
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.DrawSupply(ctx, supplyPlayer, "Player", store.SupplyDrawRequest{ClientRequestID: fmt.Sprintf("roll_%d", i), PoolID: pool.ID, Revision: pool.PublishedRevision, Scheme: "120", Quantity: 1})
			if err == nil {
				success.Add(1)
			}
		}(i)
	}
	wg.Wait()
	a, err := s.SupplyAccount(ctx, supplyPlayer)
	if err != nil || success.Load() != 10 || a.Limited != 0 {
		t.Fatal(success.Load(), a, err)
	}
	var count int
	if err = s.DB.QueryRow("SELECT COUNT(*) FROM supply_orders").Scan(&count); err != nil || count != 10 {
		t.Fatal(count, err)
	}
	supplyFund(t, s, actor, 1)
	success.Store(0)
	in := store.SupplyDrawRequest{ClientRequestID: "one_key", PoolID: pool.ID, Revision: pool.PublishedRevision, Scheme: "120", Quantity: 1}
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.DrawSupply(ctx, supplyPlayer, "Player", in)
			if err == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	a, err = s.SupplyAccount(ctx, supplyPlayer)
	if err != nil || success.Load() != 12 || a.Limited != 0 {
		t.Fatal(success.Load(), a, err)
	}
	if err = s.DB.QueryRow("SELECT COUNT(*) FROM supply_orders").Scan(&count); err != nil || count != 11 {
		t.Fatal(count, err)
	}
}

func TestSupplyRandomTargetSurvivesAnotherPoolAndAccountRead(t *testing.T) {
	s, actor := supplyFixture(t)
	ctx := context.Background()
	a := supplyPublish(t, s, actor, "a", supplyDefinition("LIMITED"))
	b := supplyPublish(t, s, actor, "b", supplyDefinition("LIMITED"))
	supplyFund(t, s, actor, 2)
	if _, err := s.DB.Exec("INSERT INTO supply_pity VALUES(?,'limited:120',0,117,8)", supplyPlayer); err != nil {
		t.Fatal(err)
	}
	for i, p := range []store.SupplyPool{a, b} {
		raw, err := s.DrawSupply(ctx, supplyPlayer, "Player", store.SupplyDrawRequest{ClientRequestID: fmt.Sprintf("r%d", i), PoolID: p.ID, Revision: p.PublishedRevision, Scheme: "120", Quantity: 1})
		if err != nil {
			t.Fatal(err)
		}
		var order store.SupplyOrder
		json.Unmarshal(raw, &order)
		if i == 0 && order.Rolls[0].Target != 117 {
			t.Fatal("existing target replaced", order)
		}
		state, err := s.SupplyAccount(ctx, supplyPlayer)
		if err != nil {
			t.Fatal(err)
		}
		if order.Rolls[0].Reward.Stars != 6 && state.Pity["limited:120"].Target != order.Rolls[0].Target {
			t.Fatal("target replaced without gold")
		}
	}
}

func TestSupplyDraftDoesNotChangePublishedRulesAndEventRerunCannotChangePity(t *testing.T) {
	s, actor := supplyFixture(t)
	ctx := context.Background()
	p := supplyPublish(t, s, actor, "event", supplyDefinition("EVENT"))
	d := p.Definition
	d.Title = "新的草稿"
	d.EventPity = 120
	raw, err := s.ChangeSupplyPool(ctx, actor, p.ID, store.SupplyPoolChange{ClientRequestID: "edit", Action: "save", ExpectedVersion: p.Version, Definition: d})
	if err != nil {
		t.Fatal(err)
	}
	var draft store.SupplyPool
	json.Unmarshal(raw, &draft)
	public, err := s.SupplyPools(ctx, false)
	if err != nil || public[0].Definition.Title == d.Title {
		t.Fatal("draft leaked", public, err)
	}
	if _, err = s.ChangeSupplyPool(ctx, actor, p.ID, store.SupplyPoolChange{ClientRequestID: "bad_publish", Action: "publish", ExpectedVersion: draft.Version}); err == nil {
		t.Fatal("event hard pity changed")
	}
	if _, err = s.ChangeSupplyPool(ctx, actor, "", store.SupplyPoolChange{ClientRequestID: "bad_rerun", Action: "save", SourcePoolID: p.ID, Definition: d}); err == nil {
		t.Fatal("rerun changed rules")
	}
	d = p.Definition
	d.Title = "活动复刻"
	raw, err = s.ChangeSupplyPool(ctx, actor, "", store.SupplyPoolChange{ClientRequestID: "rerun", Action: "save", SourcePoolID: p.ID, Definition: d})
	if err != nil {
		t.Fatal(err)
	}
	var rerun store.SupplyPool
	json.Unmarshal(raw, &rerun)
	if rerun.ID == p.ID || rerun.Definition.SeriesID != p.Definition.SeriesID {
		t.Fatal(rerun, p)
	}
}

func TestSupplyPriceChangeRequiresConfirmationAndSettlementIsIdempotent(t *testing.T) {
	s, actor := supplyFixture(t)
	ctx := context.Background()
	if _, err := s.SaveSupplySettings(ctx, actor, "price", store.SupplySettings{Version: 1, StandardPrice: 500, LimitedPrice: 1000}); err != nil {
		t.Fatal(err)
	}
	in := store.SupplyExchangeRequest{ClientRequestID: "exchange", Currency: "LIMITED", Quantity: 10, SettingsVersion: 1}
	if _, err := s.CreateSupplyExchange(ctx, supplyPlayer, "Player", in); err == nil {
		t.Fatal("stale quote charged")
	}
	in.SettingsVersion = 2
	raw, err := s.CreateSupplyExchange(ctx, supplyPlayer, "Player", in)
	if err != nil {
		t.Fatal(err)
	}
	var exchange store.SupplyExchange
	json.Unmarshal(raw, &exchange)
	if exchange.TotalPrice != 10000 {
		t.Fatal(exchange)
	}
	if err = s.CompleteSupplyExchange(ctx, exchange); err == nil {
		t.Fatal("credited without a reserved exchange")
	}
	if err = s.UpdateSupplyExchange(ctx, exchange.ID, "PENDING", "RESERVED", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.CompleteSupplyExchange(ctx, exchange); err != nil {
			t.Fatal(err)
		}
	}
	a, err := s.SupplyAccount(ctx, supplyPlayer)
	if err != nil || a.Limited != 10 {
		t.Fatal(a, err)
	}
}

func TestSupplyTicketCatalogBackfillsAnExistingPublicationOnce(t *testing.T) {
	s, _ := supplyFixture(t)
	ctx := context.Background()
	const migration = "048_supply_ticket_catalogs.sql"
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM schema_migrations_next WHERE version=?", migration); err != nil {
		t.Fatal(err)
	}
	const previous = `{"tickets":[{"id":"STANDARD","name":"常驻补给券","price":100,"enabled":true},{"id":"LIMITED","name":"限定补给券","price":100,"enabled":true}],"pools":[]}`
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO supply_editor_publications VALUES(4,?,UTC_TIMESTAMP(6))", previous); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		settings, err := s.SupplySettings(ctx)
		if err != nil || settings.Version != 2 || settings.StandardPrice != 10000 || settings.LimitedPrice != 10000 {
			t.Fatal(settings, err)
		}
	}
}

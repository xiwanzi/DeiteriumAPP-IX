//go:build integration && supplyfixture

package httpapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Opt-in visual harness using a random loopback SQL schema and the real supply HTTP handlers.
// The existing Core contract double stands in for funds and Mail; it never reaches a game instance.
func TestSupplyBrowserFixture(t *testing.T) {
	output := os.Getenv("DEUTERIUM_SUPPLY_FIXTURE")
	origin := os.Getenv("DEUTERIUM_SUPPLY_ORIGIN")
	if output == "" || origin != "http://127.0.0.1:5237" {
		t.Fatal("explicit loopback fixture output and origin required")
	}
	f := newSupplyHTTPFixture(t, origin)
	ctx := context.Background()
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, value any) {
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(output, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	pools := map[string]store.SupplyPool{}
	for _, entry := range []struct{ key, name, kind, model string }{{"a", "墨影流光", "LIMITED", "printstream"}, {"b", "绯红回响", "LIMITED", "scarlet"}, {"standard", "常驻补给", "STANDARD", "printstream"}, {"event", "周年活动", "EVENT", "scarlet"}, {"c", "远航限定", "LIMITED", "printstream"}, {"d", "寒冬限定", "LIMITED", "scarlet"}, {"e", "星海限定", "LIMITED", "printstream"}} {
		pools[entry.key] = f.pool(t, entry.name, entry.kind, entry.model)
	}
	if _, err := f.store.SaveSupplySettings(ctx, f.admin.ID, "fixture-prices", store.SupplySettings{Version: 1, StandardPrice: 10000, LimitedPrice: 10000}); err != nil {
		t.Fatal(err)
	}
	for _, currency := range []string{"STANDARD", "LIMITED"} {
		if _, err := f.store.AdjustSupplyTickets(ctx, f.admin.ID, store.SupplyAdjustment{ClientRequestID: "fixture-" + currency, PlayerUUID: f.player.ServerUUID, Currency: currency, Action: "add", Amount: 2000, Reason: "隔离验收初始余额"}, true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.store.DB.Exec("INSERT INTO supply_pity VALUES(?,'limited:80',79,80,0),(?,'limited:120',30,115,0)", f.player.ServerUUID, f.player.ServerUUID); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		name string
		user store.User
	}{{"admin", f.admin}, {"player", f.player}} {
		token := identity.Secret()
		expires := time.Now().Add(2 * time.Hour)
		if err := f.store.CreateSession(ctx, entry.user, store.Digest([]byte(token)), "web", identity.Secret(), expires, ""); err != nil {
			t.Fatal(err)
		}
		write(entry.name+"-state.json", map[string]any{"cookies": []any{map[string]any{"name": "deuterium_dev_session", "value": token, "domain": "127.0.0.1", "path": "/", "expires": expires.Unix(), "httpOnly": true, "secure": false, "sameSite": "Lax"}}, "origins": []any{}})
	}
	write("connection.json", map[string]any{"backendOrigin": f.http.URL, "webOrigin": origin, "nodeToken": f.nodeToken, "nodeId": "amiya", "playerUuid": f.player.ServerUUID, "sessionEpoch": "supply-session", "pools": pools})
	if err := os.WriteFile(filepath.Join(output, "api-url.txt"), []byte(f.http.URL), 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("Supply visual fixture ready; local connection metadata written.")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	deadline := time.NewTimer(90 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(output, "stop")); err == nil {
				return
			}
		case <-deadline.C:
			t.Fatal("supply visual fixture timed out")
		}
	}
}

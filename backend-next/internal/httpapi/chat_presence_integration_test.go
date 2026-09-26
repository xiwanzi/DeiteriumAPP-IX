//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestChatPresenceAuthLiveSnapshotsAndDeletedAccounts(t *testing.T) {
	f := newSocialFixtureV2(t)
	f.app.Config.Nodes = []config.Node{{ID: "a", Chat: true}, {ID: "b", Chat: true}, {ID: "hidden"}}
	for _, path := range []string{"/api/v1/chat/presence", "/api/v1/chat/online-players"} {
		f.request(t, "", "GET", path, nil, 401)
	}
	read := func() map[string]any {
		return socialDataV2(f.request(t, "Alice", "GET", "/api/v1/chat/online-players", nil, 200))
	}
	if data := read(); data["available"] != false || data["onlineCount"] != float64(0) {
		t.Fatalf("no snapshots: %v", data)
	}
	send := func(context.Context, string, any) error { return nil }
	f.app.Core.Connect("a", send)
	f.app.Core.SetPresence("a", []byte(`{"players":[]}`))
	if data := read(); data["available"] != true || len(data["players"].([]any)) != 0 {
		t.Fatalf("empty snapshot: %v", data)
	}
	var live []store.CorePlayerIdentity
	for _, name := range []string{"Alice", "Bob", "Carol"} {
		u := f.users[name]
		p := store.CorePlayerIdentity{PlayerUUID: u.ServerUUID, GameID: name, ServerID: "a", Online: true}
		if _, err := f.store.RememberCorePlayer(context.Background(), p); err != nil {
			t.Fatal(err)
		}
		live = append(live, p)
	}
	guest := store.CorePlayerIdentity{PlayerUUID: "d97161f9-2a7c-4abd-a8e7-6fd64a64c004", GameID: "Guest", ServerID: "a", Online: true}
	if _, err := f.store.RememberCorePlayer(context.Background(), guest); err != nil {
		t.Fatal(err)
	}
	live = append(live, guest)
	encoded, _ := json.Marshal(map[string]any{"players": live})
	f.app.Core.SetPresence("a", encoded)
	f.app.Core.Connect("b", send)
	f.app.Core.SetPresence("b", encoded)
	data := read()
	players := data["players"].([]any)
	if len(players) != 4 || data["onlineCount"] != float64(4) {
		t.Fatalf("live list not deduplicated: %v", data)
	}
	for _, raw := range players {
		p := raw.(map[string]any)
		if p["playerRef"] == "" || p["online"] != true || p["registered"] != (p["gameId"] != "Guest") {
			t.Fatalf("wrong identity: %v", p)
		}
	}
	// A newly seen guest can be mentioned before ever sending a chat message.
	guestRef := players[len(players)-1].(map[string]any)["playerRef"].(string)
	if _, _, err := f.store.PublishAppChatV2(context.Background(), f.users["Alice"], "mention-new-guest", "@Guest hello", "", []string{guestRef}, nil); err != nil {
		t.Fatalf("online guest mention failed: %v", err)
	}
	if _, _, err := f.store.PublishAppChatV2(context.Background(), f.users["Alice"], "mention-forged", "hello", "", []string{"player_unknown"}, nil); err != store.ErrSocialNotFound {
		t.Fatalf("unknown mention was accepted: %v", err)
	}
	if _, err := f.store.DB.Exec(`INSERT INTO account_deletions(user_id,player_ref,original_uuid,deleted_at) VALUES(?,?,?,UTC_TIMESTAMP(6))`, f.users["Bob"].ID, f.users["Bob"].PlayerRef, f.users["Bob"].ServerUUID); err != nil {
		t.Fatal(err)
	}
	if data = read(); len(data["players"].([]any)) != 3 {
		t.Fatalf("deleted player leaked through stale snapshot: %v", data)
	}
	f.app.Core.Disconnect("a")
	f.app.Core.Disconnect("b")
	f.app.Core.Connect("hidden", send)
	f.app.Core.SetPresence("hidden", encoded)
	if data = read(); data["available"] != false || data["onlineCount"] != float64(0) {
		t.Fatalf("non-chat node leaked: %v", data)
	}
	f.app.Core.Connect("a", send)
	f.app.Core.SetPresence("a", []byte(`{"players":[]}`))
	if data = read(); data["available"] != true || len(data["players"].([]any)) != 0 {
		t.Fatalf("historical directory supplied false online players: %v", data)
	}
}

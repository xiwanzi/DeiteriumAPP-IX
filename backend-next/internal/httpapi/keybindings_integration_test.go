//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestKeybindingLifecycleUsesAuthenticatedGameUUID(t *testing.T) {
	f := newSocialFixtureV2(t)
	const uuid = "665d6c06-2bdb-44e0-a141-1b99bbacf111" // No App identity.
	const secret = "isolated-node-test-secret-for-keybindings"
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, TokenSHA256: store.Digest([]byte(secret))}}
	var hints int
	f.app.Core.Connect("amiya", func(_ context.Context, kind string, p any) error {
		if kind == "game.chat.delivery" {
			hint := p.(map[string]string)
			if hint["playerUuid"] != uuid {
				t.Fatal("misdirected change")
			}
			hints++
		}
		return nil
	})
	f.app.Core.SetGameChat("amiya", true)
	presence := func(name, epoch string) {
		players := []map[string]string{}
		if epoch != "" {
			players = append(players, map[string]string{"playerUuid": uuid, "gameId": name, "sessionEpoch": epoch})
		}
		body, _ := json.Marshal(map[string]any{"players": players})
		f.app.Core.SetPresence("amiya", body)
	}
	post := func(path string, headers map[string]string, body any, status int) json.RawMessage {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r, err := http.NewRequest("POST", f.http.URL+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		reply, err := f.http.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer reply.Body.Close()
		var envelope map[string]json.RawMessage
		if err = json.NewDecoder(reply.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if reply.StatusCode != status {
			t.Fatalf("%s: got %d expected %d: %s", path, reply.StatusCode, status, envelope["error"])
		}
		return envelope["data"]
	}
	issue := func(epoch string, status int) string {
		t.Helper()
		data := post("/bridge/v1/game-chat/keybindings/session", map[string]string{"Authorization": "Bearer " + secret, "X-Deuterium-Node-ID": "amiya"}, map[string]string{"playerUuid": uuid, "sessionEpoch": epoch}, status)
		if status != 200 {
			return ""
		}
		var session map[string]string
		json.Unmarshal(data, &session)
		return session["token"]
	}
	presence("Pilot", "session-1")
	issue("wrong", 401)
	token := issue("session-1", 200)
	syncKeys := func(in store.KeySync, status int) store.KeyState {
		t.Helper()
		data := post("/api/v1/game/keybindings/sync", map[string]string{"Authorization": "Bearer " + token, "X-Deuterium-Player": in.PlayerUUID}, in, status)
		var out store.KeyState
		if status == 200 {
			if err := json.Unmarshal(data, &out); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	in := store.KeySync{PlayerUUID: uuid, AllowRemote: true}
	if state := syncKeys(in, 200); state.Revision != 0 || state.UUID != uuid {
		t.Fatal(state)
	}
	in.Seed = true
	in.Changes = map[string]store.KeyChange{"key.chat": {Value: "key.keyboard.t", Label: "聊天", Category: "多人游戏", Context: "game"}, "key.jump": {Value: "key.keyboard.space", Label: "跳跃", Category: "移动", Context: "game"}}
	if state := syncKeys(in, 200); !state.Seeded || state.Revision != 1 || len(state.Rows) != 0 {
		t.Fatal(state)
	}
	// A lost seed response or a fresh device cannot overwrite an existing profile.
	changed := in.Changes["key.chat"]
	changed.Value = "key.keyboard.b"
	in.Changes["key.chat"] = changed
	if state := syncKeys(in, 200); !state.Full || state.Rows["key.chat"].Value != "key.keyboard.t" {
		t.Fatal(state)
	}
	in.Seed = false
	in.Since = 1
	in.Changes = nil
	if state := syncKeys(in, 200); len(state.Rows) != 0 {
		t.Fatal("unchanged connection downloaded full data")
	}
	in.Changes = map[string]store.KeyChange{"key.jump": {Before: "key.keyboard.space", Value: "key.mouse.4"}}
	if state := syncKeys(in, 200); len(state.Rows) != 1 || state.Revision != 2 || state.Rows["key.jump"].Observed != "key.mouse.4" {
		t.Fatal(state)
	}
	if state := syncKeys(in, 200); state.Revision != 2 {
		t.Fatal("non-idempotent delta")
	}

	admin := "/api/v1/admin/keybindings"
	path := admin + "/" + uuid
	f.request(t, "Bob", "GET", admin, nil, 403)
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	edit := func(value string, version int, status int) {
		t.Helper()
		f.request(t, "Alice", "PUT", path, map[string]any{"action": "key.chat", "value": value, "expectedRevision": version}, status)
	}
	detail := func() store.KeyState {
		t.Helper()
		d, _ := json.Marshal(socialDataV2(f.request(t, "Alice", "GET", path, nil, 200)))
		var out store.KeyState
		if e := json.Unmarshal(d, &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	edit("key.keyboard.g:ALT", 1, 200)
	adminDelta := socialDataV2(f.request(t, "Alice", "GET", path+"?since=2", nil, 200))
	if len(adminDelta["rows"].(map[string]any)) != 1 || adminDelta["full"] == true {
		t.Fatal("admin refresh returned the entire keymap")
	}
	if hints != 1 {
		t.Fatal("online change not signaled")
	}
	edit("key.keyboard.h", 1, 409)
	if row := detail().Rows["key.chat"]; row.Value != "key.keyboard.g:ALT" || row.Observed != "key.keyboard.t" {
		t.Fatal(row)
	}
	in.Since = 2
	in.Changes = nil
	if state := syncKeys(in, 200); len(state.Rows) != 1 || state.Rows["key.chat"].Revision != 3 {
		t.Fatal(state)
	}
	in.Changes = map[string]store.KeyChange{"key.chat": {Value: "key.keyboard.g:ALT", Ack: 3}}
	if state := syncKeys(in, 200); state.Rows["key.chat"].Observed != "key.keyboard.g:ALT" || state.Revision != 4 {
		t.Fatal(state)
	}

	presence("", "")
	edit("key.keyboard.x", 4, 200)
	if hints != 1 {
		t.Fatal("offline edit signaled another client")
	}
	syncKeys(in, 401)
	presence("RenamedPilot", "session-2")
	syncKeys(in, 401)
	token = issue("session-2", 200)
	in.Since = 4
	in.Changes = map[string]store.KeyChange{"key.chat": {Value: "key.keyboard.p", Before: "key.keyboard.g:ALT"}}
	state := syncKeys(in, 200)
	key := state.Rows["key.chat"]
	if !key.Conflict || key.Observed != "key.keyboard.p" || key.Value != "key.keyboard.x" || state.GameID != "RenamedPilot" {
		t.Fatal(state)
	}
	// A late acknowledgement for the previous session cannot resolve a newer conflict.
	in.Changes = map[string]store.KeyChange{"key.chat": {Value: "key.keyboard.g:ALT", Ack: 3}}
	syncKeys(in, 200)
	if !detail().Rows["key.chat"].Conflict {
		t.Fatal("stale ack cleared a newer conflict")
	}
	edit("key.keyboard.p", int(key.Revision), 200)
	resolved := detail().Rows["key.chat"]
	if resolved.Conflict || resolved.Value != resolved.Observed {
		t.Fatal(resolved)
	}
	in.PlayerUUID = f.users["Bob"].ServerUUID
	syncKeys(in, 401)
	f.tokens["keys-only"] = token
	f.request(t, "keys-only", "GET", admin, nil, 401)
}

func TestKeybindingConcurrentEditsAndLargeSeed(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	const id = "665d6c06-2bdb-44e0-a141-1b99bbacf111"
	keys := make(map[string]store.KeyChange)
	for i := range 212 {
		keys[fmt.Sprintf("key.test.%d", i)] = store.KeyChange{Value: "key.keyboard.t", Label: "Action", Category: "Test", Context: "game"}
	}
	if out, err := db.SyncKeybindings(ctx, id, "Pilot", store.KeySync{Seed: true, Changes: keys}); err != nil || !out.Seeded {
		t.Fatalf("seed: %v", err)
	}
	result := make(chan error, 2)
	for _, value := range []string{"key.keyboard.a", "key.keyboard.b"} {
		go func() { result <- db.SetAdminKey(ctx, "test", id, "key.test.1", value, 1) }()
	}
	first, second := <-result, <-result
	if !((first == nil && errors.Is(second, store.ErrConflict)) || (second == nil && errors.Is(first, store.ErrConflict))) {
		t.Fatalf("concurrent results: %v / %v", first, second)
	}
	for _, action := range []string{"key.test.0", "key.test.2"} {
		go func() { result <- db.SetAdminKey(ctx, "test", id, action, "key.mouse.left", 1) }()
	}
	for range 2 {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	delta, err := db.SyncKeybindings(ctx, id, "Pilot", store.KeySync{Since: 1})
	if err != nil || len(delta.Rows) != 3 || delta.Revision != 4 {
		t.Fatalf("expected exactly three deltas from 212 actions: %d %d %v", len(delta.Rows), delta.Revision, err)
	}
	absent, err := db.SyncKeybindings(ctx, id, "Pilot", store.KeySync{Since: 4, Availability: map[string]bool{"key.test.1": false}})
	if err != nil || len(absent.Rows) != 1 || !absent.Rows["key.test.1"].Missing {
		t.Fatal("removed mod was not marked unavailable", absent, err)
	}
	back, err := db.SyncKeybindings(ctx, id, "Pilot", store.KeySync{Since: absent.Revision, Availability: map[string]bool{"key.test.1": true}})
	if err != nil || len(back.Rows) != 1 || back.Rows["key.test.1"].Missing || back.Rows["key.test.1"].Value != absent.Rows["key.test.1"].Value {
		t.Fatal("reinstalled mod lost its last binding", back, err)
	}
}

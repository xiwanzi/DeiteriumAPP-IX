//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNameLookupNetworkWaitDoesNotOccupyPasswordWorkers(t *testing.T) {
	f := newSocialFixtureV2(t)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	f.app.AdmissionResolver = func(ctx context.Context, name string) (store.AdmissionProfile, error) {
		started <- struct{}{}
		select {
		case <-ctx.Done():
			return store.AdmissionProfile{}, ctx.Err()
		case <-release:
		}
		return store.AdmissionProfile{}, store.ErrSocialNotFound
	}
	done := make(chan struct{}, 2)
	for _, name := range []string{"NewAlice", "NewBob"} {
		go func(name string) {
			defer func() { done <- struct{}{} }()
			_, _, _ = f.app.Identity.Login(context.Background(), name, "original-password", "127.0.0.1", "app")
		}(name)
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("lookups did not enter")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := f.app.Identity.HashPassword(ctx, "independent-password")
	cancel()
	once.Do(func() { close(release) })
	for range 2 {
		<-done
	}
	if err != nil {
		t.Fatal("external profile lookup exhausted password workers", err)
	}
}

func TestNamePresenceEventSyncsWithoutIdleHeartbeatWrites(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	alice := f.users["Alice"]
	f.app.Core.Connect("amiya", func(context.Context, string, any) error { return nil })
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: alice.ServerUUID, Name: "NewAlice"}, nil
	}
	update := func(name string) {
		raw, _ := json.Marshal(map[string]any{"players": []any{map[string]string{"playerUuid": alice.ServerUUID, "gameId": name, "sessionEpoch": "11111111-1111-4111-8111-111111111111"}}})
		if _, err := f.app.coreSpecial(ctx, config.Node{ID: "amiya", Chat: true}, nil, bridge.Envelope{Type: "core.presence.snapshot", Payload: raw}); err != nil {
			t.Fatal(err)
		}
	}
	update("Alice")
	var first, second time.Time
	if err := f.store.DB.QueryRow("SELECT last_seen FROM core_player_directory WHERE player_uuid=?", alice.ServerUUID).Scan(&first); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		update("Alice")
	}
	if err := f.store.DB.QueryRow("SELECT last_seen FROM core_player_directory WHERE player_uuid=?", alice.ServerUUID).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if !first.Equal(second) {
		t.Fatal("unchanged heartbeat wrote player directory")
	}
	update("NewAlice")
	deadline := time.Now().Add(2 * time.Second)
	for {
		u, err := f.store.UserByUUID(ctx, alice.ServerUUID)
		if err == nil && u.GameID == "NewAlice" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("event did not trigger rename", u, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestNameRealtimeEventsAndSingleConversationRemainAccountScoped(t *testing.T) {
	f := newSocialFixtureV2(t)
	conversation := f.conversation(t, "Alice", "Bob", "event-pair")
	f.request(t, "Carol", "GET", "/api/v1/chat/conversations/"+conversation, nil, 404)
	f.request(t, "Bob", "GET", "/api/v1/chat/conversations/"+conversation, nil, 200)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	dial := func(user string) *websocket.Conn {
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(f.http.URL, "http")+"/api/v1/chat/ws", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + f.tokens[user]}}, Subprotocols: []string{"deuterium.sync.v1"}})
		if err != nil {
			t.Fatal(err)
		}
		if c.Subprotocol() != "deuterium.sync.v1" {
			t.Fatal("browser protocol negotiation failed")
		}
		t.Cleanup(func() { c.CloseNow() })
		event, err := readSocket(ctx, c)
		if err != nil || event.Type != "sync.changed" {
			t.Fatal("missing reconnect invalidation", event, err)
		}
		return c
	}
	bob, carol := dial("Bob"), dial("Carol")
	f.request(t, "Alice", "POST", "/api/v1/chat/conversations/"+conversation+"/messages", map[string]any{"clientMessageId": "event-one", "content": "scoped message"}, 200)
	first, err := readSocket(ctx, bob)
	if err != nil || first.Type != "chat.conversation.changed" {
		t.Fatal(first, err)
	}
	second, err := readSocket(ctx, bob)
	if err != nil || second.Type != "chat.message" {
		t.Fatal(second, err)
	}
	readCtx, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stop()
	if event, err := readSocket(readCtx, carol); err == nil {
		t.Fatal("private event escaped recipient scope", event)
	}
}

func TestNameTypedGameHistoryKeepsStoredSnapshot(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	alice, bob := f.users["Alice"], f.users["Bob"]
	message := store.GameChatMessage{ChatMessage: store.ChatMessage{ID: "gmsg_name_fixture", Sender: store.Player{PlayerRef: alice.ServerUUID, GameID: "Alice"}, Content: "Alice original text", SentAt: time.Now().UTC()}, Type: "message", Channel: "team", Members: []string{bob.ServerUUID}}
	raw, _ := json.Marshal(message)
	if _, err := f.store.DB.Exec(`INSERT INTO game_chat_messages(message_id,sender_id,client_message_id,fingerprint,channel,message_json,sent_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6))`, message.ID, alice.ID, "name-fixture", store.Digest(raw), "team", raw); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.DB.Exec("INSERT INTO game_chat_recipients(message_id,player_uuid) VALUES(?,?)", message.ID, bob.ServerUUID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: alice.ServerUUID, Name: "NewAlice"}}); err != nil {
		t.Fatal(err)
	}
	rows, err := f.store.GameChatMessages(ctx, bob.ServerUUID, "team")
	if err != nil || len(rows) != 1 || rows[0].Sender.GameID != "NewAlice" || rows[0].Content != "Alice original text" {
		t.Fatal(rows, err)
	}
	var stored string
	if err = f.store.DB.QueryRow("SELECT message_json FROM game_chat_messages WHERE message_id=?", message.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != string(raw) {
		t.Fatal("stored game message was rewritten")
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestDirectChatSocketImmediateDeliveryAndIsolation(t *testing.T) {
	f := newSocialFixtureV2(t)
	conversation := f.conversation(t, "Alice", "Bob", "realtime-pair")
	dial := func(user string, private bool) *websocket.Conn {
		t.Helper()
		header := http.Header{"Authorization": []string{"Bearer " + f.tokens[user]}}
		if private {
			header.Set("X-Deuterium-Private-Chat", "1")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(f.http.URL, "http")+"/api/v1/chat/ws", &websocket.DialOptions{HTTPHeader: header})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.CloseNow() })
		return c
	}
	bob, alice, carol, legacy := dial("Bob", true), dial("Alice", true), dial("Carol", true), dial("Bob", false)
	sentAt := time.Now()
	response := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/chat/conversations/"+conversation+"/messages", map[string]any{"clientMessageId": "socket-message", "content": "即时私聊"}, 200))
	for _, c := range []*websocket.Conn{bob, alice} {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		event, err := readSocket(ctx, c)
		cancel()
		if err != nil || event.Type != "chat.message" {
			t.Fatalf("private event missing: %s %v", event.Type, err)
		}
		var payload struct {
			Message store.SocialMessage `json:"message"`
		}
		if err = json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Message.MessageID != response["messageId"] || payload.Message.ConversationID != conversation || payload.Message.Kind != "private_chat" {
			t.Fatal("wrong private envelope", payload)
		}
	}
	if time.Since(sentAt) >= 2*time.Second {
		t.Fatal("private delivery waited for polling")
	}
	// A retry wakes the hub but cannot emit a duplicate event.
	f.request(t, "Alice", "POST", "/api/v1/chat/conversations/"+conversation+"/messages", map[string]any{"clientMessageId": "socket-message", "content": "即时私聊"}, 200)
	for _, c := range []*websocket.Conn{bob, carol, legacy} {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		event, err := readSocket(ctx, c)
		cancel()
		if err == nil {
			t.Fatalf("duplicate, unauthorized, or unrequested event: %s", event.Type)
		}
	}
	list, err := f.store.DirectChatAfter(context.Background(), f.users["Carol"].ID, 0)
	if err != nil || len(list) != 0 {
		t.Fatal("private history crossed account boundary", err)
	}
}

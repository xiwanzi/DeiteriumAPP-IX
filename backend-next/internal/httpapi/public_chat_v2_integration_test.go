//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestPublicChatV2ReplyMetadataMentionNotificationsAndAtomicRejection(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true}}
	if !f.app.Hub.JoinNode("amiya") {
		t.Fatal("test node not joined")
	}
	defer f.app.Hub.LeaveNode("amiya")
	original, _, err := f.store.PublishAppChat(ctx, f.users["Bob"], "source-public", "公共原文", nil)
	if err != nil {
		t.Fatal(err)
	}
	ab := f.conversation(t, "Alice", "Bob", "metadata-ab")
	private := f.send(t, "Alice", ab, "source-private", "私密正文")["messageId"].(string)
	send := func(value map[string]any) map[string]any {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return f.app.sendChat(ctx, store.Digest([]byte(f.tokens["Alice"])), raw)
	}
	payload := map[string]any{"clientMessageId": "public-reply", "content": "这是公开回复", "replyToMessageId": original, "mentionedPlayerRefs": []string{"player_Bob"}}
	accepted := send(payload)
	if accepted["status"] != "accepted" {
		t.Fatalf("public metadata rejected: %v", accepted)
	}
	messageID := accepted["messageId"].(string)
	if replay := send(payload); replay["messageId"] != messageID {
		t.Fatal("public retry generated another message")
	}
	history := socialDataV2(f.request(t, "Carol", "GET", "/api/v1/chat/messages", nil, 200))["messages"].([]any)
	found := false
	for _, raw := range history {
		m := raw.(map[string]any)
		if m["messageId"] == messageID {
			found = true
			if m["reply"].(map[string]any)["content"] != "公共原文" {
				t.Fatal("public reply snapshot incorrect")
			}
			if refs := m["mentionedPlayerRefs"].([]any); len(refs) != 1 || refs[0] != "player_Bob" {
				t.Fatal("explicit public mention not returned")
			}
		}
		if m["content"] == "私密正文" {
			t.Fatal("private message leaked into public history")
		}
	}
	if !found {
		t.Fatal("public metadata not in history")
	}
	notices := f.request(t, "Bob", "GET", "/api/v1/notifications", nil, 200)["data"].([]any)
	mentions := 0
	for _, raw := range notices {
		n := raw.(map[string]any)
		if n["topic"] == "MENTIONS" {
			mentions++
		}
	}
	if mentions != 1 {
		t.Fatalf("explicit mention without @text was not notified once: %d", mentions)
	}
	payload["replyToMessageId"] = private
	conflict := send(payload)
	if conflict["error"].(map[string]string)["code"] != "IDEMPOTENCY_CONFLICT" {
		t.Fatal("reply change did not conflict")
	}
	payload["clientMessageId"] = "private-reply-attempt"
	denied := send(payload)
	if denied["error"].(map[string]string)["code"] != "NOT_FOUND" {
		t.Fatal("private source accepted in public reply")
	}
	payload["clientMessageId"] = "unknown-mention-attempt"
	payload["replyToMessageId"] = original
	payload["mentionedPlayerRefs"] = []string{"player_nonexistent"}
	denied = send(payload)
	if denied["error"].(map[string]string)["code"] != "NOT_FOUND" {
		t.Fatal("forged player reference accepted")
	}
	var count int
	for _, table := range []string{"public_chat_metadata_v2", "core_chat_deliveries"} {
		if err = f.store.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("rejected public send left partial effects in %s: %d", table, count)
		}
	}
	pending, err := f.store.PendingChats(ctx, "amiya")
	if err != nil || len(pending) != 1 || pending[0].Content != "这是公开回复" {
		t.Fatal("Core delivery changed normal content or leaked private content", err)
	}
	if err = f.store.ReconcilePublicSocialNotificationsV2(ctx); err != nil {
		t.Fatal(err)
	}
	if got := f.request(t, "Bob", "GET", "/api/v1/notifications", nil, 200)["data"].([]any); len(got) != len(notices) {
		t.Fatal("reconciliation duplicated immediate explicit mention")
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"testing"
)

func TestPublicChatV203HTTPAcceptsWithoutSocketAndSharesMessageIdentity(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	payload := map[string]any{"clientMessageId": "v203-public", "content": "无需等待历史同步的消息"}
	offline := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/chat/messages", payload, 200))
	if offline["status"] != "failed" {
		t.Fatal("offline game bridge bypassed")
	}
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true}}
	if !f.app.Hub.JoinNode("amiya") {
		t.Fatal("test node failed")
	}
	defer f.app.Hub.LeaveNode("amiya")
	acceptedResponse := f.request(t, "Alice", "POST", "/api/v1/chat/messages", payload, 200)
	writeV203ContractSample(t, "PublicSendResponseV203", acceptedResponse)
	accepted := socialDataV2(acceptedResponse)
	if accepted["status"] != "accepted" {
		t.Fatal(accepted)
	}
	message := accepted["message"].(map[string]any)
	if message["messageId"] != accepted["messageId"] || message["content"] != payload["content"] || message["sender"].(map[string]any)["playerRef"] != f.users["Alice"].PlayerRef {
		t.Fatal("acceptance did not return canonical message")
	}
	raw, _ := json.Marshal(payload)
	socketReplay := f.app.sendChat(ctx, store.Digest([]byte(f.tokens["Alice"])), raw)
	if socketReplay["messageId"] != accepted["messageId"] {
		t.Fatal("cross-transport replay duplicated message")
	}
	httpReplay := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/chat/messages", payload, 200))
	if httpReplay["messageId"] != accepted["messageId"] {
		t.Fatal("HTTP replay duplicated message")
	}
	payload["content"] = "同键更换正文"
	conflict := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/chat/messages", payload, 200))
	if conflict["status"] != "failed" || conflict["error"].(map[string]any)["code"] != "IDEMPOTENCY_CONFLICT" {
		t.Fatal("changed payload accepted")
	}
	var count int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM chat_messages_next WHERE client_message_id='v203-public'").Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate durable message", count, err)
	}
	payload["senderUuid"] = "forged"
	f.request(t, "Alice", "POST", "/api/v1/chat/messages", payload, 400)
}

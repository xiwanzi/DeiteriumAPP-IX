package bridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestNodeCannotForgeSourceOrPublishOutsideNamespace(t *testing.T) {
	node := config.Node{ID: "amiya", Chat: true, ItemPrefix: "deuterium:"}
	cfg := config.Config{Nodes: []config.Node{node}}
	for _, raw := range []string{
		`{"playerUuid":"d97161f9-2a7c-4abd-a8e7-6fd64a64c001","gameId":"Alice","content":"hello","serverId":"odyssey"}`,
		`{"playerUuid":"forged","gameId":"Alice","content":"hello"}`,
	} {
		if _, _, _, err := ValidateCore(cfg, node, Envelope{Type: "chat.public.event", EventID: "one", Payload: json.RawMessage(raw)}); err == nil {
			t.Fatal("forged source or invalid identity accepted")
		}
	}
	item := store.ItemVersion{ItemRef: "other:item", Revision: 1, PayloadSHA256: strings.Repeat("a", 64), DisplayName: "Test", MaxQuantity: 1, CompatibleServerIDs: []string{"amiya"}}
	raw, _ := json.Marshal(item)
	if _, _, _, err := ValidateCore(cfg, node, Envelope{Type: "item.version.published", EventID: "item", Payload: raw}); err == nil {
		t.Fatal("unauthorized item namespace accepted")
	}
	item.ItemRef = "deuterium:item"
	item.CompatibleServerIDs = []string{"unknown"}
	raw, _ = json.Marshal(item)
	if _, _, _, err := ValidateCore(cfg, node, Envelope{Type: "item.version.published", EventID: "item", Payload: raw}); err == nil {
		t.Fatal("unknown server scope accepted")
	}
}

func TestHubWakeupDoesNotBlockOnSlowSubscriber(t *testing.T) {
	h := NewHub()
	wake, stop := h.Subscribe()
	defer stop()
	for range 10000 {
		h.Wake()
	}
	if len(wake) != 1 {
		t.Fatal("wakeups must coalesce")
	}
	if !h.JoinNode("amiya") || h.JoinNode("amiya") {
		t.Fatal("duplicate node connection accepted")
	}
}

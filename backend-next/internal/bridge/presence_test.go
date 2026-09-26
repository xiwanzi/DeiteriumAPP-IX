package bridge

import (
	"context"
	"testing"
)

func TestPresenceUsesConnectedSelectedSnapshotsAndDeduplicatesUUID(t *testing.T) {
	r := NewRuntime()
	send := func(context.Context, string, any) error { return nil }
	r.Connect("a", send)
	if p, ok := r.Presence([]string{"a"}); ok || len(p) != 0 {
		t.Fatal("missing snapshot was treated as empty online state")
	}
	r.SetPresence("a", []byte(`{"players":[]}`))
	if p, ok := r.Presence([]string{"a"}); !ok || len(p) != 0 {
		t.Fatal("received empty snapshot must be available")
	}
	r.SetPresence("a", []byte(`{"players":[{"playerUuid":"one","gameId":"Alice"}]}`))
	r.Connect("b", send)
	r.SetPresence("b", []byte(`{"players":[{"playerUuid":"one","gameId":"Alice"},{"playerUuid":"two","gameId":"Bob"}]}`))
	r.Connect("excluded", send)
	r.SetPresence("excluded", []byte(`{"players":[{"playerUuid":"three","gameId":"Carol"}]}`))
	if p, ok := r.Presence([]string{"a", "b"}); !ok || len(p) != 2 || p[0].GameID != "Alice" || p[1].GameID != "Bob" {
		t.Fatalf("bad merged snapshot: %v %v", p, ok)
	}
	r.Disconnect("b")
	r.SetPresence("b", []byte(`{"players":[{"playerUuid":"two","gameId":"Bob"}]}`))
	if p, ok := r.Presence([]string{"a", "b"}); !ok || len(p) != 1 {
		t.Fatal("disconnected node remained online")
	}
	r.Disconnect("a")
	if p, ok := r.Presence([]string{"a", "b"}); ok || len(p) != 0 {
		t.Fatal("all nodes disconnected must be unavailable")
	}
}

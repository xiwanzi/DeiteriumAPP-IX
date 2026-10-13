package bridge

import (
	"context"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"testing"
)

func TestScopedWakeRetainsAllTopicsWithoutWakingOtherUsers(t *testing.T) {
	h := NewHub()
	alice, stopA := h.Watch(UserTopic("alice"), TopicNames)
	defer stopA()
	bob, stopB := h.Watch(UserTopic("bob"))
	defer stopB()
	revision := h.Revision()
	h.Wake(ConversationTopic("alice", "one"))
	h.Wake(ConversationTopic("alice", "two"))
	h.Wake(TopicNames)
	if len(alice.Signal) != 1 || len(bob.Signal) != 0 {
		t.Fatal("unrelated connection woke or events did not coalesce")
	}
	items := alice.Take()
	if !items[ConversationTopic("alice", "one")] || !items[ConversationTopic("alice", "two")] || !items[TopicNames] {
		t.Fatal(items)
	}
	if h.Revision() != revision {
		t.Fatal("private events invalidated public message cache")
	}
}

func TestPresenceOnlyPersistsActualSessionChanges(t *testing.T) {
	r := NewRuntime()
	r.Connect("a", func(context.Context, string, any) error { return nil })
	r.SetPresence("a", []byte(`{"players":[{"playerUuid":"uuid","gameId":"Alice","sessionEpoch":"first"}]}`))
	players := []store.CorePlayerIdentity{{PlayerUUID: "uuid", GameID: "Alice", SessionEpoch: "first"}}
	changed, affected := r.PresenceChanges("a", players)
	if len(changed) != 0 || len(affected) != 0 {
		t.Fatal("unchanged heartbeat requires persistence")
	}
	players[0].SessionEpoch = "second"
	changed, affected = r.PresenceChanges("a", players)
	if len(changed) != 1 || len(affected) != 1 || !changed[0].Online || changed[0].ServerID != "a" {
		t.Fatal(changed, affected)
	}
	changed, affected = r.PresenceChanges("a", nil)
	if len(changed) != 1 || changed[0].Online || len(affected) != 1 {
		t.Fatal("departure did not reconcile ambiguous sessions")
	}
}

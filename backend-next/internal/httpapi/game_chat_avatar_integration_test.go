//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"
)

func TestGameAvatarPersistsForOfflinePlayersAndKeepsLastOnUpstreamFailure(t *testing.T) {
	f := newSocialFixtureV2(t)
	calls := 0
	portrait := []byte("persisted fixture portrait")
	f.app.GameAvatarResolver = func(ctx context.Context, uuid string) ([]byte, error) {
		calls++
		if uuid != f.users["Bob"].ServerUUID {
			t.Error("untrusted identity")
		}
		return portrait, nil
	}
	// Bob has no online Core presence. Only his authoritative saved identity is needed.
	got := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/chat/avatars/Bob", nil, 200))
	if got["png"] != base64.StdEncoding.EncodeToString(portrait) || calls != 1 {
		t.Fatal(got, calls)
	}
	failing := func(context.Context, string) ([]byte, error) { calls++; return nil, errors.New("upstream offline") }
	// A separate Store instance has no process cache; it must retrieve the persisted portrait.
	copyStore := *f.store
	out, err := copyStore.GameChatAvatar(context.Background(), "bob", failing)
	if err != nil || !bytes.Equal(out, portrait) || calls != 1 {
		t.Fatal(err, calls)
	}
	_, err = f.store.DB.Exec(`UPDATE game_chat_avatars SET refresh_after=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)`)
	if err != nil {
		t.Fatal(err)
	}
	out, err = copyStore.GameChatAvatar(context.Background(), "Bob", failing)
	if err != nil || !bytes.Equal(out, portrait) || calls != 2 {
		t.Fatal(err, calls)
	}
	f.request(t, "Alice", "GET", "/api/v1/chat/avatars/UnknownUser", nil, 404)
	f.request(t, "Alice", "POST", "/api/v1/chat/avatars/Bob", nil, 404)
}

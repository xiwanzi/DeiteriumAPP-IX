//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"testing"
)

func TestGameChatSessionIsScopedAndBoundToLiveNode(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	nodeToken := "local-test-node-secret-0123456789012345"
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, TokenSHA256: store.Digest([]byte(nodeToken))}}
	f.app.Core.Connect("amiya", func(context.Context, string, any) error { return nil })
	raw, _ := json.Marshal(map[string]any{"players": []map[string]string{{"playerUuid": f.users["Alice"].ServerUUID, "gameId": "Alice"}}})
	f.app.Core.SetPresence("amiya", raw)
	data, _ := json.Marshal(map[string]string{"playerUuid": f.users["Alice"].ServerUUID})
	r, _ := http.NewRequest("POST", f.http.URL+"/bridge/v1/game-chat/session", bytes.NewReader(data))
	r.Header.Set("Authorization", "Bearer "+nodeToken)
	r.Header.Set("X-Deuterium-Node-ID", "amiya")
	r.Header.Set("Content-Type", "application/json")
	response, err := f.http.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope map[string]any
	_ = json.NewDecoder(response.Body).Decode(&envelope)
	if response.StatusCode != 200 {
		t.Fatalf("session: %d %v", response.StatusCode, envelope)
	}
	token := socialDataV2(envelope)["token"].(string)
	f.tokens["game"] = token
	session, err := f.store.Session(ctx, store.Digest([]byte(token)))
	if err != nil || session.Kind != "gamechat" || session.CSRF != "amiya" {
		t.Fatal("wrong token authority", err)
	}
	f.request(t, "game", "GET", "/api/v1/chat/messages", nil, 200)
	for _, path := range []string{"/api/v1/wallet/balance", "/api/v1/account/me", "/api/v1/admin/core/nodes", "/api/v1/chat/ws"} {
		f.request(t, "game", "GET", path, nil, 403)
	}
	f.app.Core.Disconnect("amiya")
	f.request(t, "game", "GET", "/api/v1/chat/messages", nil, 401)
}

func TestPrivateAndNearbyMediaCannotEscapeRecipientGrant(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	a, b, c := f.users["Alice"], f.users["Bob"], f.users["Carol"]
	media := chatMediaFixture(t, f, "Alice", 501).Media
	conversation := f.conversation(t, "Alice", "Bob", "private-media")
	response := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/chat/conversations/"+conversation+"/messages", map[string]any{"clientMessageId": "private-image", "content": "", "media": map[string]string{"mediaId": media.ID, "type": "image"}}, 200))
	messageID := response["messageId"].(string)
	if response["media"] == nil {
		t.Fatal("private media descriptor missing")
	}
	if _, _, err := f.store.ChatMediaAccess(ctx, b.ID, media.ID, messageID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.ChatMediaAccess(ctx, c.ID, media.ID, messageID); err == nil {
		t.Fatal("private image leaked to third party")
	}
	nearbyMedia := chatMediaFixture(t, f, "Alice", 502).Media
	in := store.GameChatMediaRequest{PlayerUUID: a.ServerUUID, ClientMessageID: "near-image", Channel: "nearby", Media: store.ChatMediaSend{ID: nearbyMedia.ID, Type: "image"}, Members: []string{a.ServerUUID, b.ServerUUID}}
	m, err := f.store.GameChatMedia(ctx, "amiya", in)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, b.ID, nearbyMedia.ID, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, c.ID, nearbyMedia.ID, m.ID); err == nil {
		t.Fatal("nearby image leaked beyond original audience")
	}
	in.Members = append(in.Members, c.ServerUUID)
	retry, err := f.store.GameChatMedia(ctx, "amiya", in)
	if err != nil || retry.ID != m.ID {
		t.Fatal("idempotent retry failed", err)
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, c.ID, nearbyMedia.ID, m.ID); err == nil {
		t.Fatal("retry widened recipient grant")
	}
	if list, err := f.store.GameChatMessages(ctx, c.ServerUUID, "nearby"); err != nil || len(list) != 0 {
		t.Fatal("unrelated history visible", err)
	}
	if _, err = f.store.DB.Exec(`UPDATE game_chat_media SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE message_id=?`, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.store.ChatMediaAccess(ctx, b.ID, nearbyMedia.ID, m.ID); !errors.Is(err, store.ErrMediaExpired) {
		t.Fatal("message expiry not enforced", err)
	}
}

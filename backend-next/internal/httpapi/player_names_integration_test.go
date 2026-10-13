//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestPlayerNameLoginReclaimsVerifiedReusedNameWithoutMovingAccounts(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	a, b := f.users["Alice"], f.users["Bob"]
	_, e := f.store.DB.Exec("UPDATE identities SET password_hash=? WHERE id=?", identity.Hash("alice-original-pass"), a.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.store.DB.Exec("INSERT INTO identity_aliases VALUES('alice',?),('bob',?)", a.ID, b.ID)
	if e != nil {
		t.Fatal(e)
	}
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: a.ServerUUID, Name: "Bob"}, nil
	}
	f.app.OfficialUUIDResolver = func(_ context.Context, uuid string) (store.AdmissionProfile, error) {
		if uuid == b.ServerUUID {
			return store.AdmissionProfile{UUID: uuid, Name: "RenamedBob"}, nil
		}
		return store.AdmissionProfile{}, store.ErrSocialNotFound
	}
	_, session, e := f.app.Identity.Login(ctx, "Bob", "alice-original-pass", "127.0.0.1", "app")
	if e != nil || session.User.ID != a.ID {
		t.Fatal(session, e)
	}
	owner, e := f.store.UserByAlias(ctx, "renamedbob")
	if e != nil || owner.ID != b.ID {
		t.Fatal("former owner lost its identity", owner, e)
	}
}

func TestPlayerNameSlowProofCannotRollBackAndCoreFeedIsAuthenticated(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	a := f.users["Alice"]
	oldObservation := time.Now().Add(-time.Minute)
	_, e := f.store.SyncVerifiedNamesObserved(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "LatestName"}}, time.Now().Add(-time.Second))
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.store.SyncVerifiedNamesObserved(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "Alice"}}, oldObservation)
	if !errors.Is(e, store.ErrNameProofStale) {
		t.Fatal("late official response rolled back identity", e)
	}
	current, _ := f.store.UserByUUID(ctx, a.ServerUUID)
	if current.GameID != "LatestName" {
		t.Fatal(current)
	}
	if _, e = f.store.SyncVerifiedNamesObserved(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "LatestName"}}, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e = f.store.SyncVerifiedNamesObserved(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "Alice"}}, time.Now().Add(-500*time.Millisecond)); !errors.Is(e, store.ErrNameProofStale) {
		t.Fatal("unchanged fresh proof did not fence older observation", e)
	}
	if _, e = f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "NewestName"}}); e != nil {
		t.Fatal(e)
	}
	token := "name-sync-node-test-token-1234567890123456789"
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, TokenSHA256: store.Digest([]byte(token))}}
	call := func(auth string, want int) {
		r := httptest.NewRequest("POST", "/bridge/v1/game-chat/player-names", bytes.NewBufferString(`{"after":0}`))
		r.Header.Set("Authorization", "Bearer "+auth)
		r.Header.Set("X-Deuterium-Node-ID", "amiya")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		f.app.Handler().ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("got %d want %d: %s", w.Code, want, w.Body.String())
		}
		if want == http.StatusOK {
			var out struct {
				Data struct {
					Items []store.PlayerName `json:"items"`
				} `json:"data"`
			}
			if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || len(out.Data.Items) != 2 {
				t.Fatal("invalid journal", w.Body.String(), e)
			}
			if out.Data.Items[0].GameID != "LatestName" || out.Data.Items[1].GameID != "NewestName" {
				t.Fatal("journal names projected across versions", out)
			}
		}
	}
	call(f.tokens["Alice"], http.StatusUnauthorized)
	call(token, http.StatusOK)
}

func TestPlayerNameSyncPreservesAccountSessionsNicknameAndSnapshots(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	u := f.users["Alice"]
	_, e := f.store.DB.Exec("INSERT INTO identity_aliases VALUES('alice',?),('10001',?)", u.ID, u.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.store.DB.Exec("UPDATE identities SET display_name='保留的中文昵称' WHERE id=?", u.ID); e != nil {
		t.Fatal(e)
	}
	changes, e := f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: u.ServerUUID, Name: "NewAlice"}})
	if e != nil || len(changes) != 1 {
		t.Fatal(changes, e)
	}
	again, e := f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: u.ServerUUID, Name: "NewAlice"}})
	if e != nil || len(again) != 0 {
		t.Fatal("rename not idempotent", again, e)
	}
	updated, e := f.store.UserByUUID(ctx, u.ServerUUID)
	if e != nil || updated.ID != u.ID || updated.PlayerRef != u.PlayerRef || updated.GameID != "NewAlice" || updated.DisplayName != "保留的中文昵称" || updated.PasswordHash != u.PasswordHash || updated.QQ != u.QQ {
		t.Fatal(updated, e)
	}
	if _, e = f.store.Session(ctx, store.Digest([]byte(f.tokens["Alice"]))); e != nil {
		t.Fatal("existing session revoked", e)
	}
	if old, e := f.store.UserByAlias(ctx, "alice"); e == nil {
		t.Fatal("old alias retained", old)
	}
	if qq, e := f.store.UserByAlias(ctx, "10001"); e != nil || qq.ID != u.ID {
		t.Fatal("QQ changed", qq, e)
	}
	body := map[string]any{"buyer": map[string]any{"kind": "PLAYER", "playerRef": u.PlayerRef, "displayName": "Alice"}, "amount": "9007199254740993.25", "content": "Alice wrote this text", "snapshot": map[string]any{"buyer": map[string]any{"playerRef": u.PlayerRef, "gameId": "Alice"}}, "profile": map[string]any{"playerRef": u.PlayerRef, "gameId": "Alice", "displayName": "保留的中文昵称"}}
	before, _ := json.Marshal(body)
	projected, e := f.store.ProjectPlayerNames(ctx, body)
	if e != nil {
		t.Fatal(e)
	}
	after, _ := json.Marshal(body)
	if string(before) != string(after) {
		t.Fatal("source snapshot mutated")
	}
	out := projected.(map[string]any)
	if out["buyer"].(map[string]any)["displayName"] != "NewAlice" || out["amount"] != "9007199254740993.25" || out["content"] != "Alice wrote this text" || out["snapshot"].(map[string]any)["buyer"].(map[string]any)["gameId"] != "Alice" || out["profile"].(map[string]any)["displayName"] != "保留的中文昵称" {
		t.Fatal(out)
	}
	journal, cursor, more, e := f.store.PlayerNameChanges(ctx, 0)
	if e != nil || len(journal) != 1 || cursor < 1 || more {
		t.Fatal(journal, cursor, more, e)
	}
	f.request(t, "Alice", "GET", "/api/v1/chat/player-names", nil, 200)
}

func TestPlayerNameLoginNewAliasRequiresOriginalPassword(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	u := f.users["Alice"]
	hash := identity.Hash("correct-password")
	if _, e := f.store.DB.Exec("UPDATE identities SET password_hash=? WHERE id=?", hash, u.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := f.store.DB.Exec("INSERT INTO identity_aliases VALUES('alice',?)", u.ID); e != nil {
		t.Fatal(e)
	}
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		if name != "newalice" {
			return store.AdmissionProfile{}, store.ErrSocialNotFound
		}
		return store.AdmissionProfile{UUID: u.ServerUUID, Name: "NewAlice"}, nil
	}
	_, _, e := f.app.Identity.Login(ctx, "NewAlice", "wrong-password", "127.0.0.1", "app")
	if e == nil {
		t.Fatal("wrong password accepted")
	}
	original, _ := f.store.UserByUUID(ctx, u.ServerUUID)
	if original.GameID != "Alice" {
		t.Fatal("wrong password changed identity")
	}
	_, session, e := f.app.Identity.Login(ctx, "NewAlice", "correct-password", "127.0.0.1", "app")
	if e != nil || session.User.ID != u.ID || session.User.GameID != "NewAlice" {
		t.Fatal(session, e)
	}
	var count int
	f.store.DB.QueryRow("SELECT COUNT(*) FROM identities").Scan(&count)
	if count != 3 {
		t.Fatal("duplicate account created")
	}
}

func TestPlayerNameReuseBatchCannotStealQQOrAssets(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	a, b := f.users["Alice"], f.users["Bob"]
	_, e := f.store.DB.Exec("INSERT INTO identity_aliases VALUES('alice',?),('bob',?),('10002',?)", a.ID, b.ID, b.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "Bob"}})
	if e == nil {
		t.Fatal("conflicting alias stolen")
	}
	_, e = f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "Bob"}, {UUID: b.ServerUUID, Name: "Alice"}})
	if e != nil {
		t.Fatal("verified name swap failed", e)
	}
	renamedA, e := f.store.UserByAlias(ctx, "bob")
	if e != nil || renamedA.ID != a.ID || renamedA.ServerUUID != a.ServerUUID {
		t.Fatal(renamedA, e)
	}
	renamedB, e := f.store.UserByAlias(ctx, "alice")
	if e != nil || renamedB.ID != b.ID || renamedB.ServerUUID != b.ServerUUID {
		t.Fatal(renamedB, e)
	}
	_, e = f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: a.ServerUUID, Name: "10002"}, {UUID: b.ServerUUID, Name: "OtherBob"}})
	if e == nil {
		t.Fatal("QQ alias stolen")
	}
	qq, _ := f.store.UserByAlias(ctx, "10002")
	if qq.ID != b.ID {
		t.Fatal("QQ owner changed")
	}
	if _, e = f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: strings.Repeat("x", 36), Name: "Bad"}}); e == nil {
		t.Fatal("invalid UUID accepted")
	}
}

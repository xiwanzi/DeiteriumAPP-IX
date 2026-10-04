//go:build integration

package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestAdminGameIDRenameVerifiedIdentityAndReplay(t *testing.T) {
	f := deletionFixture(t)
	ctx := context.Background()
	bob := f.users["Bob"]
	path := "/api/v1/admin/accounts/" + bob.ID + "/game-id"
	in := store.AdminGameIDChange{ClientRequestID: "rename-bob", ExpectedVersion: 1, GameID: "NewBob"}
	f.request(t, "Bob", "POST", path, in, 403)
	if err := f.store.Grant(ctx, f.users["Carol"].ID, "support.manage"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Carol", "POST", path, in, 403)
	f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+f.users["Alice"].ID+"/game-id", in, 409)
	deletionSQL(t, f, "UPDATE identities SET display_name='自定义昵称',email='bob@example.com' WHERE id=?", bob.ID)
	deletionSQL(t, f, "INSERT INTO identity_aliases VALUES('bob@example.com',?)", bob.ID)
	orderHash := deletionOrder(t, f, "COMPLETED", "SETTLED")
	f.app.Core.Connect("rename-node", func(context.Context, string, any) error { return nil })
	f.app.Core.SetPresence("rename-node", []byte(`{"players":[{"playerUuid":"`+bob.ServerUUID+`","gameId":"NewBob"}]}`))
	changed := socialDataV2(f.request(t, "Alice", "POST", path, in, 200))
	if changed["gameId"] != "NewBob" || changed["previousGameId"] != "Bob" || changed["version"] != float64(2) {
		t.Fatal(changed)
	}
	f.app.Core.Disconnect("rename-node")
	f.request(t, "Alice", "POST", path, in, 200)
	current, err := f.store.UserByAlias(ctx, "newbob")
	if err != nil || current.ID != bob.ID || current.PlayerRef != bob.PlayerRef || current.ServerUUID != bob.ServerUUID || current.PasswordHash != bob.PasswordHash || current.DisplayName != "自定义昵称" || current.Email != "bob@example.com" || !current.GameVerified {
		t.Fatal("rename altered account identity", err)
	}
	if _, err := f.store.UserByAlias(ctx, "bob"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("old game ID still logs in", err)
	}
	for _, alias := range []string{bob.QQ, "bob@example.com"} {
		u, err := f.store.UserByAlias(ctx, alias)
		if err != nil || u.ID != bob.ID {
			t.Fatal("non-game login alias changed", err)
		}
	}
	f.request(t, "Bob", "GET", "/api/v1/account/me", nil, 401)
	login := socialDataV2(f.request(t, "", "POST", "/api/v1/account/login", map[string]string{"account": "NewBob", "password": deletionTargetPassword}, 200))
	if login["user"].(map[string]any)["userId"] != bob.ID {
		t.Fatal("new login name did not retain user")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM commerce_resources_v2 WHERE payee_id=? AND payee_uuid=? AND amount='20.00' AND snapshot_sha256=?", bob.ID, bob.ServerUUID, orderHash) != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM audit_events_next WHERE action='account.game-id'") != 1 {
		t.Fatal("commerce or audit changed on retry")
	}
	f.app.Core.Connect("rename-node", func(context.Context, string, any) error { return nil })
	f.app.Core.SetPresence("rename-node", []byte(`{"players":[{"playerUuid":"`+f.users["Carol"].ServerUUID+`","gameId":"OtherPlayer"}]}`))
	in = store.AdminGameIDChange{ClientRequestID: "different-uuid", ExpectedVersion: 2, GameID: "OtherPlayer"}
	response := f.request(t, "Alice", "POST", path, in, 409)
	if response["error"].(map[string]any)["code"] != "GAME_ID_IDENTITY_MISMATCH" {
		t.Fatal("different game identity was not rejected")
	}
}

func TestAdminGameIDRenamePendingProofAndConcurrentNames(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	u := emailSignup(t, f, "TypoPlayer", "typo@example.com")
	v := store.GameVerification{ID: "proof-before-rename", TokenHash: store.Digest([]byte("proof-token")), CodeHash: store.Digest([]byte("proof-code")), UserID: u.ID, Purpose: "game_verify", GameID: "TypoPlayer", PlayerUUID: "11111111-2222-3333-4444-555555555555", QQ: "889900", NodeID: "test", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewGameVerification(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ActivateGameVerification(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/admin/accounts/" + u.ID + "/game-id"
	in := store.AdminGameIDChange{ClientRequestID: "fix-typo", ExpectedVersion: 1, GameID: "CorrectPlayer"}
	for _, invalid := range []string{"含中文", "invalid-name", "DIMA", "DaoYu", strings.Repeat("a", 33)} {
		bad := in
		bad.GameID = invalid
		f.request(t, "Alice", "POST", path, bad, 400)
	}
	f.request(t, "Alice", "POST", path, in, 200)
	f.request(t, "Alice", "POST", path, in, 200)
	current, err := f.store.UserByAlias(ctx, "correctplayer")
	if err != nil || current.ID != u.ID || current.GameVerified || current.ServerUUID != "" {
		t.Fatal("correcting the name verified or rebound the account", err)
	}
	if err := f.store.BindGameIdentity(ctx, u.ID, v); !errors.Is(err, store.ErrVerification) {
		t.Fatal("old proof could overwrite corrected game name", err)
	}
	if err := f.store.NewGameVerification(ctx, v); !errors.Is(err, store.ErrVerification) {
		t.Fatal("stale verification issuance accepted", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM game_verifications WHERE user_id=? AND state='ACTIVE'", u.ID) != 0 {
		t.Fatal("old game code still active")
	}
	f.request(t, "TypoPlayer", "GET", "/api/v1/account/me", nil, 401)
	stale := in
	stale.ClientRequestID, stale.GameID = "stale-version", "AnotherName"
	f.request(t, "Alice", "POST", path, stale, 409)
	second := emailSignup(t, f, "SecondPlayer", "second@example.com")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, account := range []struct {
		id      string
		version int64
	}{{u.ID, 2}, {second.ID, 1}} {
		wg.Add(1)
		go func(target string, version int64) {
			defer wg.Done()
			_, err := f.store.RenameAccountGameID(ctx, f.users["Alice"].ID, target, store.AdminGameIDChange{ClientRequestID: "claim-common-name", ExpectedVersion: version, GameID: "SharedName"}, "")
			results <- err
		}(account.id, account.version)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM identity_aliases WHERE alias_key='sharedname'") != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE game_id='SharedName'") != 1 {
		t.Fatal("concurrent rename did not enforce unique game ID")
	}
}

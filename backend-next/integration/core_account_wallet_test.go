//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestVerificationCooldownAttemptsSingleConsumptionAndResetRevocation(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	v := store.GameVerification{ID: "verify_parallel", TokenHash: store.Digest([]byte("token")), CodeHash: store.Digest([]byte("correct")), Purpose: "register", PlayerUUID: aliceUUID, GameID: "Alice", QQ: "10001", NodeID: "amiya", ExpiresAt: time.Now().Add(10 * time.Minute)}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			copy := v
			copy.ID = fmt.Sprintf("verify_%d", i)
			copy.TokenHash = store.Digest([]byte(copy.ID))
			e := db.NewGameVerification(ctx, copy)
			if e == nil {
				successes.Add(1)
			} else if !errors.Is(e, store.ErrRateLimited) {
				t.Errorf("issue: %v", e)
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal("parallel cooldown bypass", successes.Load())
	}
	if _, err := db.DB.Exec("DELETE FROM game_verifications"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.Exec("DELETE FROM game_verification_cooldowns"); err != nil {
		t.Fatal(err)
	}
	if err := db.NewGameVerification(ctx, v); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CheckGameVerification(ctx, v.TokenHash, v.CodeHash, "register"); !errors.Is(err, store.ErrVerification) {
		t.Fatal("undelivered code accepted", err)
	}
	if err := db.ActivateGameVerification(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := db.CheckGameVerification(ctx, v.TokenHash, "wrong", "register"); !errors.Is(err, store.ErrVerification) {
			t.Fatal(err)
		}
	}
	if _, err := db.CheckGameVerification(ctx, v.TokenHash, v.CodeHash, "register"); !errors.Is(err, store.ErrVerification) {
		t.Fatal("attempt cap bypass")
	}
	if _, err := db.DB.Exec("UPDATE game_verifications SET attempts=0"); err != nil {
		t.Fatal(err)
	}
	verified, err := db.CheckGameVerification(ctx, v.TokenHash, v.CodeHash, "register")
	if err != nil {
		t.Fatal(err)
	}
	hash := identity.Hash(password)
	successes.Store(0)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := db.RegisterGameUser(ctx, verified, hash)
			if e == nil {
				successes.Add(1)
			} else if !errors.Is(e, store.ErrVerification) {
				t.Errorf("consume: %v", e)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal("verification consumed multiple times")
	}
	service := identity.New(db)
	_, session, err := service.Login(ctx, "Alice", password, "192.0.2.9", "app")
	if err != nil {
		t.Fatal(err)
	}
	reset := v
	reset.ID = "verify_reset"
	reset.Purpose = "password_reset"
	reset.UserID = session.User.ID
	reset.TokenHash = store.Digest([]byte("reset-token"))
	reset.CodeHash = store.Digest([]byte("reset-code"))
	if err = db.NewGameVerification(ctx, reset); err != nil {
		t.Fatal(err)
	}
	if err = db.ActivateGameVerification(ctx, reset.ID); err != nil {
		t.Fatal(err)
	}
	reset, err = db.CheckGameVerification(ctx, reset.TokenHash, reset.CodeHash, reset.Purpose)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.ResetGamePassword(ctx, reset, identity.Hash("a-new-password-123")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Session(ctx, session.TokenHash); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("reset left session active")
	}
	if err = db.ResetGamePassword(ctx, reset, hash); !errors.Is(err, store.ErrVerification) {
		t.Fatal("reset replay reused code")
	}
}

// This peer stands at the game-command boundary; it never simulates provider
// funds. Separate plugin tests exercise atomic XConomy writes and failures.
func corePeer(f *fixture, handler func(string, map[string]any) (string, any)) {
	f.app.Core.Connect("amiya", func(ctx context.Context, kind string, payload any) error {
		frame := payload.(map[string]any)
		id := frame["operationId"].(string)
		command := frame["command"].(string)
		raw, _ := json.Marshal(frame["payload"])
		var input map[string]any
		_ = json.Unmarshal(raw, &input)
		status, data := handler(command, input)
		reply, _ := json.Marshal(map[string]any{"operationId": id, "status": status, "data": data})
		err := f.store.CoreReply(ctx, "amiya", id, status, reply)
		f.app.Core.Notify(id)
		return err
	})
}
func TestGameRegistrationHTTPUsesSessionUUIDAndCookieBoundary(t *testing.T) {
	f := newFixture(t)
	uuid := "d97161f9-2a7c-4abd-a8e7-6fd64a64c009"
	var code string
	corePeer(f, func(command string, p map[string]any) (string, any) {
		if command != "verification.deliver" || p["playerUuid"] != uuid {
			t.Error("wrong identity target", command)
		}
		code = p["code"].(string)
		return "COMPLETED", map[string]bool{"delivered": true}
	})
	presence, _ := json.Marshal(map[string]any{"players": []any{map[string]string{"playerUuid": uuid, "gameId": "NewPlayer", "sessionEpoch": "d97161f9-2a7c-4abd-a8e7-6fd64a64c099"}}})
	f.app.Core.SetPresence("amiya", presence)
	resp, result := f.request(t, "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "qq": "10009"}, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode, result)
	}
	token := result["data"].(map[string]any)["verificationToken"].(string)
	for _, invalidPassword := range []string{"", "short", string(make([]byte, 65))} {
		resp, result = f.request(t, "POST", "/api/v1/account/register", map[string]string{"verificationToken": token, "code": code, "password": invalidPassword}, nil, nil)
		if resp.StatusCode != 400 || result["error"].(map[string]any)["code"] != "PASSWORD_INVALID" {
			t.Fatal("registration accepted an invalid password", resp.StatusCode, result)
		}
	}
	in := map[string]string{"verificationToken": token, "code": code, "password": password}
	resp, _ = f.request(t, "POST", "/api/v1/account/register", in, http.Header{"Origin": []string{f.http.URL}}, nil)
	if resp.StatusCode != 403 {
		t.Fatal("browser acquired bearer")
	}
	resp, result = f.request(t, "POST", "/api/v1/web/register", in, http.Header{"Origin": []string{f.http.URL}}, nil)
	if resp.StatusCode != 200 {
		t.Fatal(result)
	}
	if _, ok := result["data"].(map[string]any)["token"]; ok {
		t.Fatal("web leaked bearer")
	}
	if len(resp.Cookies()) != 1 || !resp.Cookies()[0].HttpOnly {
		t.Fatal("missing HttpOnly registration session")
	}
	u, err := f.store.UserByAlias(context.Background(), "newplayer")
	if err != nil || u.ServerUUID != uuid {
		t.Fatal("registration guessed UUID", err)
	}
	resp, _ = f.request(t, "POST", "/api/v1/account/register", in, nil, nil)
	if resp.StatusCode != 400 {
		t.Fatal("consumed code accepted", resp.StatusCode)
	}
	for _, name := range []string{"DIMA", "DaoYu"} {
		resp, _ = f.request(t, "POST", "/api/v1/account/registration-code", map[string]string{"gameId": name, "qq": "10019", "password": password}, nil, nil)
		if resp.StatusCode != 400 {
			t.Fatal("system registration allowed")
		}
	}
}

func TestRegistrationCodeIgnoresLegacyPasswordAndKeepsValidationAndCooldown(t *testing.T) {
	for _, legacyPassword := range []string{"", "short", password} {
		t.Run(fmt.Sprintf("legacy-password-length-%d", len(legacyPassword)), func(t *testing.T) {
			f := newFixture(t)
			uuid := "d97161f9-2a7c-4abd-a8e7-6fd64a64c009"
			var deliveries atomic.Int32
			corePeer(f, func(command string, p map[string]any) (string, any) {
				if command != "verification.deliver" || p["playerUuid"] != uuid || p["password"] != nil {
					t.Error("unexpected verification payload", command)
				}
				deliveries.Add(1)
				return "COMPLETED", map[string]bool{"delivered": true}
			})
			presence, _ := json.Marshal(map[string]any{"players": []any{map[string]string{"playerUuid": uuid, "gameId": "NewPlayer", "sessionEpoch": "d97161f9-2a7c-4abd-a8e7-6fd64a64c099"}}})
			f.app.Core.SetPresence("amiya", presence)
			for _, invalid := range []struct{ gameID, qq, code string }{{"", "10009", "GAME_ID_INVALID"}, {"NewPlayer", "123", "QQ_INVALID"}, {"NewPlayer", "１２３４５", "QQ_INVALID"}} {
				resp, result := f.request(t, "POST", "/api/v1/account/registration-code", map[string]string{"gameId": invalid.gameID, "qq": invalid.qq}, nil, nil)
				if resp.StatusCode != 400 || result["error"].(map[string]any)["code"] != invalid.code {
					t.Fatal("incorrect identity validation", resp.StatusCode, result)
				}
			}
			request := map[string]string{"gameId": "NewPlayer", "qq": "10009", "password": legacyPassword}
			resp, result := f.request(t, "POST", "/api/v1/account/registration-code", request, nil, nil)
			if resp.StatusCode != 200 {
				t.Fatal("legacy password prevented code delivery", resp.StatusCode, result)
			}
			if _, exposed := result["data"].(map[string]any)["code"]; exposed {
				t.Fatal("verification code exposed in HTTP response")
			}
			resp, result = f.request(t, "POST", "/api/v1/account/registration-code", request, nil, nil)
			if resp.StatusCode != 429 || deliveries.Load() != 1 {
				t.Fatal("verification cooldown bypass", resp.StatusCode, result, deliveries.Load())
			}
		})
	}
}

func TestWalletIdempotencyOfflineReplayRecipientIsolationAndUnknown(t *testing.T) {
	f := newFixture(t)
	for i := range f.app.Config.Nodes {
		if f.app.Config.Nodes[i].ID == "amiya" {
			f.app.Config.Nodes[i].Economy = true
		}
	}
	p := store.CorePlayerIdentity{PlayerUUID: "d97161f9-2a7c-4abd-a8e7-6fd64a64c002", GameID: "Bob", ServerID: "amiya"}
	ref, err := f.store.RememberCorePlayer(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	corePeer(f, func(command string, payload map[string]any) (string, any) {
		if command != "wallet.transfer" {
			t.Error(command)
		}
		calls.Add(1)
		return "COMPLETED", map[string]string{"amount": payload["amount"].(string)}
	})
	auth := bearer(f.appToken(t))
	in := map[string]any{"clientRequestId": "wallet_idempotency_1", "recipientPlayerRef": ref, "amount": "12.50", "note": "hello"}
	resp, result := f.request(t, "POST", "/api/v1/wallet/transfers", in, auth, nil)
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode, result)
	}
	transfer := result["data"].(map[string]any)["transfer"].(map[string]any)
	if transfer["status"] != "success" {
		t.Fatal(transfer)
	}
	id := transfer["transferId"].(string)
	f.app.Core.Disconnect("amiya")
	resp, result = f.request(t, "POST", "/api/v1/wallet/transfers", in, auth, nil)
	if resp.StatusCode != 200 || calls.Load() != 1 {
		t.Fatal("offline replay mutated again", result, calls.Load())
	}
	in["amount"] = "13"
	resp, _ = f.request(t, "POST", "/api/v1/wallet/transfers", in, auth, nil)
	if resp.StatusCode != 409 {
		t.Fatal("changed payload accepted")
	}
	resp, result = f.request(t, "GET", "/api/v1/wallet/records/"+id, nil, auth, nil)
	if resp.StatusCode != 200 || result["data"].(map[string]any)["direction"] != "expense" {
		t.Fatal(result)
	}
	resp, _ = f.request(t, "GET", "/api/v1/wallet/transfers/"+id, nil, nil, nil)
	if resp.StatusCode != 401 {
		t.Fatal("unauthenticated transfer visible")
	}
	f.app.Core.Connect("amiya", func(context.Context, string, any) error { return errors.New("connection lost") })
	in["clientRequestId"] = "wallet_unknown_2"
	resp, result = f.request(t, "POST", "/api/v1/wallet/transfers", in, auth, nil)
	if resp.StatusCode != 202 || result["data"].(map[string]any)["transfer"].(map[string]any)["status"] != "unknown" {
		t.Fatal("uncertain transfer not preserved", result)
	}
	if count(t, f.store, "wallet_transfers_next") != 2 {
		t.Fatal("duplicate transfers")
	}
}

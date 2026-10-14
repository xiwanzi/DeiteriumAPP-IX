//go:build integration

package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func registeredProfileSignup(t *testing.T, f *socialFixtureV2, name, email string) store.User {
	t.Helper()
	data := socialDataV2(f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": name, "email": email}, 200))
	token := data["verificationToken"].(string)
	code := registrationIssuedCode(t, f, token)
	f.request(t, "", "POST", "/api/v1/account/register", map[string]string{"verificationToken": token, "code": code, "password": "test-password"}, 200)
	u, e := f.store.UserByAlias(context.Background(), email)
	if e != nil || u.ProfileUUID != admissionTestUUID || u.ServerUUID != "" || u.GameVerified {
		t.Fatal("registration lost profile or granted game ownership", e)
	}
	return u
}

func TestRegisteredProfileNewNameLoginWithoutGameServer(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	auditProfile(f)
	u := registeredProfileSignup(t, f, "OldPlayer", "registered@example.com")
	if _, e := f.store.DB.Exec("UPDATE identities SET display_name='保留的昵称' WHERE id=?", u.ID); e != nil {
		t.Fatal(e)
	}
	before := deletionCount(t, f, "SELECT COUNT(*) FROM identity_sessions WHERE user_id=? AND revoked_at IS NULL", u.ID)
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: admissionTestUUID, Name: "NewPlayer"}, nil
	}
	if _, _, e := f.app.Identity.Login(ctx, "NewPlayer", "wrong-password", "127.0.0.2", "app"); !errors.Is(e, store.ErrUnauthorized) {
		t.Fatal(e)
	}
	old, e := f.store.UserByAlias(ctx, "oldplayer")
	if e != nil || old.ID != u.ID {
		t.Fatal("wrong password changed old alias", e)
	}
	if _, e = f.store.UserByAlias(ctx, "newplayer"); !errors.Is(e, sql.ErrNoRows) {
		t.Fatal("wrong password created new alias", e)
	}
	token, session, e := f.app.Identity.Login(ctx, "NewPlayer", "test-password", "127.0.0.2", "app")
	if e != nil || session.User.ID != u.ID || session.User.PlayerRef != u.PlayerRef || session.User.GameID != "NewPlayer" || session.User.DisplayName != "保留的昵称" || session.User.GameVerified || session.User.ServerUUID != "" {
		t.Fatal("new name login did not preserve unverified account", session.User.GameID, e)
	}
	if _, e = f.store.UserByAlias(ctx, "oldplayer"); !errors.Is(e, sql.ErrNoRows) {
		t.Fatal("stale alias remained", e)
	}
	if got := deletionCount(t, f, "SELECT COUNT(*) FROM identity_sessions WHERE user_id=? AND revoked_at IS NULL", u.ID); got != before+1 {
		t.Fatal("existing sessions revoked", got, before)
	}
	f.tokens["Renamed"] = token
	for _, path := range []string{"/api/v1/chat/messages", "/api/v1/wallet/balance"} {
		f.request(t, "Renamed", "GET", path, nil, 403)
	}
	if got := deletionCount(t, f, "SELECT COUNT(*) FROM player_names_state WHERE player_uuid=?", admissionTestUUID); got != 0 {
		t.Fatal("unverified profile published game identity", got)
	}
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		t.Fatal("email login called official API")
		return store.AdmissionProfile{}, nil
	}
	if _, session, e = f.app.Identity.Login(ctx, "registered@example.com", "test-password", "127.0.0.2", "app"); e != nil || session.User.ID != u.ID {
		t.Fatal("email fallback broke", e)
	}
}

func TestRegisteredProfileDuplicateUUIDAndReusedName(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	auditProfile(f)
	u := registeredProfileSignup(t, f, "OldPlayer", "registered@example.com")
	data := socialDataV2(f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "other@example.com"}, 200))
	token := data["verificationToken"].(string)
	code := registrationIssuedCode(t, f, token)
	f.request(t, "", "POST", "/api/v1/account/register", map[string]string{"verificationToken": token, "code": code, "password": "test-password"}, 409)
	if got := deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE profile_uuid=?", admissionTestUUID); got != 1 {
		t.Fatal(got)
	}
	// Another official UUID using the old name cannot authenticate to this account.
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: "99999999-0000-4000-8000-000000000099", Name: name}, nil
	}
	if _, _, e := f.app.Identity.Login(ctx, "OldPlayer", "test-password", "127.0.0.3", "app"); !errors.Is(e, store.ErrUnauthorized) {
		t.Fatal("name reuse selected previous owner", e)
	}
	still, e := f.store.UserByAlias(ctx, "registered@example.com")
	if e != nil || still.ID != u.ID || still.GameID != "OldPlayer" {
		t.Fatal(e)
	}
}

func TestRegisteredProfileFencesStaleProofCredentialChangeAndWrongGameUUID(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	auditProfile(f)
	u := registeredProfileSignup(t, f, "OldPlayer", "registered@example.com")
	now := time.Now().UTC()
	p := store.AdmissionProfile{UUID: admissionTestUUID, Name: "NewPlayer"}
	if e := f.store.SyncRegisteredName(ctx, u, p, now); e != nil {
		t.Fatal(e)
	}
	if e := f.store.SyncRegisteredName(ctx, u, p, now.Add(time.Second)); e != nil {
		t.Fatal(e)
	}
	if e := f.store.SyncRegisteredName(ctx, u, store.AdmissionProfile{UUID: admissionTestUUID, Name: "OldPlayer"}, now); !errors.Is(e, store.ErrNameProofStale) {
		t.Fatal("late proof reverted name", e)
	}
	proof := store.GameVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(identity.Secret())), CodeHash: store.Digest([]byte("proof")), Purpose: "game_verify", PlayerUUID: "99999999-0000-4000-8000-000000000099", GameID: "NewPlayer", QQ: "99887766", UserID: u.ID, NodeID: "test", ExpiresAt: time.Now().Add(time.Minute)}
	if e := f.store.NewGameVerification(ctx, proof); e != nil {
		t.Fatal(e)
	}
	if e := f.store.ActivateGameVerification(ctx, proof.ID); e != nil {
		t.Fatal(e)
	}
	if e := f.store.BindGameIdentity(ctx, u.ID, proof); !errors.Is(e, store.ErrVerification) {
		t.Fatal("bound different game UUID", e)
	}
	if _, e := f.store.DB.Exec("UPDATE identities SET password_hash=? WHERE id=?", identity.Hash("new-password"), u.ID); e != nil {
		t.Fatal(e)
	}
	if e := f.store.SyncRegisteredName(ctx, u, store.AdmissionProfile{UUID: admissionTestUUID, Name: "LaterPlayer"}, now.Add(2*time.Second)); !errors.Is(e, store.ErrCredentialChanged) {
		t.Fatal("old password proof changed name", e)
	}
}

func TestRegisteredProfileConcurrentObservationsKeepNewestName(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	auditProfile(f)
	u := registeredProfileSignup(t, f, "OldPlayer", "registered@example.com")
	start := make(chan struct{})
	out := make(chan error, 2)
	var wg sync.WaitGroup
	now := time.Now().UTC()
	for i, name := range []string{"NewPlayer", "NewestPlayer"} {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			<-start
			out <- f.store.SyncRegisteredName(ctx, u, store.AdmissionProfile{UUID: admissionTestUUID, Name: name}, now.Add(time.Duration(i)*time.Second))
		}(i, name)
	}
	close(start)
	wg.Wait()
	close(out)
	for e := range out {
		if e != nil && !errors.Is(e, store.ErrNameProofStale) {
			t.Fatal(e)
		}
	}
	after, e := f.store.UserByAlias(ctx, "registered@example.com")
	if e != nil || after.GameID != "NewestPlayer" {
		t.Fatal("concurrent stale name won", after.GameID, e)
	}
}

func TestRegisteredProfileAdminCorrectionAndDeletion(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	auditProfile(f)
	u := registeredProfileSignup(t, f, "OldPlayer", "registered@example.com")
	nextUUID := "99999999-0000-4000-8000-000000000099"
	observed := time.Now().Add(-time.Second)
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: nextUUID, Name: name}, nil
	}
	f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+u.ID+"/game-id", store.AdminGameIDChange{ClientRequestID: "correct-registered", ExpectedVersion: 1, GameID: "CorrectPlayer"}, 200)
	current, e := f.store.UserByAlias(ctx, "registered@example.com")
	if e != nil || current.ProfileUUID != nextUUID || current.GameVerified || current.ServerUUID != "" {
		t.Fatal("admin correction retained wrong profile or granted ownership", e)
	}
	if e = f.store.SyncRegisteredName(ctx, u, store.AdmissionProfile{UUID: admissionTestUUID, Name: "OldPlayer"}, observed); !errors.Is(e, store.ErrCredentialChanged) {
		t.Fatal("late pre-correction proof accepted", e)
	}
	deletionSQL(t, f, "UPDATE identities SET password_hash=? WHERE id=?", identity.Hash(deletionAdminPassword), f.users["Alice"].ID)
	body := deleteBody(deletionAdminPassword, "delete-registered")
	body["expectedVersion"] = 2
	f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+u.ID+"/delete", body, 200)
	if _, e = f.store.UserByProfileUUID(ctx, nextUUID); !errors.Is(e, sql.ErrNoRows) {
		t.Fatal("deleted profile can still locate login", e)
	}
	if got := deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE id=? AND profile_uuid IS NOT NULL", u.ID); got != 0 {
		t.Fatal("deletion retained public profile", got)
	}
}

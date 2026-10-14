//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func profileConflictFixture(t *testing.T) (*socialFixtureV2, store.User, map[string]any) {
	f := mailSupportFixture(t)
	auditProfile(f)
	u := registeredProfileSignup(t, f, "Occupied", "original@example.com")
	deletionSQL(t, f, "UPDATE identities SET password_hash=? WHERE id=?", identity.Hash(deletionAdminPassword), f.users["Alice"].ID)
	body := map[string]any{"clientRequestId": "release-case-1", "expectedVersion": 1, "claimantEmail": "rightful@example.com", "evidenceType": "SUPPORT_REVIEW", "evidenceReference": "support-test-case-1001", "reason": "客服已核实原登记非游戏账号持有人", "ownershipConfirmed": true, "password": deletionAdminPassword}
	return f, u, body
}

func TestProfileConflictReleasePreservesAccountAndProtectsClaim(t *testing.T) {
	f, u, body := profileConflictFixture(t)
	ctx := context.Background()
	before := deletionCount(t, f, "SELECT COUNT(*) FROM identities")
	path := "/api/v1/admin/accounts/" + u.ID + "/profile-release"
	f.request(t, "Bob", "POST", path, body, 403)
	result := socialDataV2(f.request(t, "Alice", "POST", path, body, 200))
	replay := socialDataV2(f.request(t, "Alice", "POST", path, body, 200))
	if result["caseId"] != replay["caseId"] {
		t.Fatal("retry created a second release")
	}
	old, err := f.store.UserByAlias(ctx, u.Email)
	if err != nil || old.ID != u.ID || old.PlayerRef != u.PlayerRef || old.PasswordHash != u.PasswordHash || old.GameID != "" || old.ProfileUUID != "" || old.GameVerified || old.ServerUUID != "" {
		t.Fatal("release moved or destroyed the platform account", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM identities") != before {
		t.Fatal("release created/deleted a platform account")
	}
	token, _, err := f.app.Identity.Login(ctx, u.Email, "test-password", "127.0.0.5", "app")
	if err != nil {
		t.Fatal("email recovery login failed", err)
	}
	f.tokens["Original"] = token
	f.request(t, "Original", "POST", "/api/v1/account/game-verification-code", map[string]string{"qq": "12345678"}, 409)
	mailBefore := deletionCount(t, f, "SELECT COUNT(*) FROM email_verifications")
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "Occupied", "email": "attacker@example.com"}, 409)
	if deletionCount(t, f, "SELECT COUNT(*) FROM email_verifications") != mailBefore {
		t.Fatal("reserved profile sent an unnecessary email challenge")
	}
	f.request(t, "Original", "POST", "/api/v1/account/game-profile", map[string]string{"clientRequestId": "reoccupy", "gameId": "Occupied"}, 409)
	// The rightful holder must still prove email control and later game control.
	owner := registeredProfileSignup(t, f, "Occupied", "rightful@example.com")
	if owner.ID == u.ID || owner.PlayerRef == u.PlayerRef || owner.GameVerified || owner.ServerUUID != "" {
		t.Fatal("public profile claim inherited the other account or game privileges")
	}
	otherUUID := "33333333-3333-4333-8333-333333333333"
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: otherUUID, Name: name}, nil
	}
	f.request(t, "Original", "POST", "/api/v1/account/game-profile", map[string]string{"clientRequestId": "own-profile", "gameId": "OwnPlayer"}, 200)
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		t.Error("successful registration retry queried upstream")
		return store.AdmissionProfile{}, identity.ErrNameLookupUnavailable
	}
	f.request(t, "Original", "POST", "/api/v1/account/game-profile", map[string]string{"clientRequestId": "own-profile", "gameId": "OwnPlayer"}, 200)
	current, err := f.store.UserByAlias(ctx, u.Email)
	if err != nil || current.ID != u.ID || current.ProfileUUID != otherUUID || current.GameVerified {
		t.Fatal("original account cannot register own identity", err)
	}
	history := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/admin/accounts/"+u.ID+"/profile-resolutions", nil, 200))["items"].([]any)
	if len(history) != 1 || history[0].(map[string]any)["claimedUserId"] != owner.ID {
		t.Fatal("claim audit is incomplete", history)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM player_names_state WHERE player_uuid=?", admissionTestUUID) != 0 {
		t.Fatal("release published unverified game identity")
	}
}

func TestProfileConflictProtectsVerifiedIdentityAndRequiresEvidence(t *testing.T) {
	f, u, body := profileConflictFixture(t)
	body["ownershipConfirmed"] = false
	f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+u.ID+"/profile-release", body, 400)
	body["ownershipConfirmed"] = true
	f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+f.users["Bob"].ID+"/profile-release", body, 409)
	if deletionCount(t, f, "SELECT COUNT(*) FROM identity_profile_resolutions") != 0 {
		t.Fatal("invalid request wrote a resolution")
	}
}

func TestProfileConflictGameProofIsScopedAndConsumedOnce(t *testing.T) {
	f, u, body := profileConflictFixture(t)
	ctx := context.Background()
	token := identity.Secret()
	v := store.GameVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(token)), CodeHash: store.Digest([]byte(token + ":123456")), UserID: u.ID, PlayerUUID: admissionTestUUID, GameID: u.GameID, Purpose: "profile_claim", NodeID: "test", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewGameVerification(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ActivateGameVerification(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	body["evidenceType"], body["verificationToken"], body["code"] = "GAME_CODE", token, "000000"
	path := "/api/v1/admin/accounts/" + u.ID + "/profile-release"
	f.request(t, "Alice", "POST", path, body, 400)
	if deletionCount(t, f, "SELECT attempts FROM game_verifications WHERE verification_id=?", v.ID) != 1 {
		t.Fatal("wrong proof attempts not retained")
	}
	body["code"] = "123456"
	f.request(t, "Alice", "POST", path, body, 200)
	f.request(t, "Alice", "POST", path, body, 200)
	if _, err := f.store.CheckGameVerification(ctx, v.TokenHash, v.CodeHash, "profile_claim"); !errors.Is(err, store.ErrVerification) {
		t.Fatal("claim proof can be consumed again", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM identity_profile_resolutions") != 1 {
		t.Fatal("retry duplicated case")
	}
}

func TestProfileConflictPendingEmailCorrectionKeepsAuditAndCannotMoveClaim(t *testing.T) {
	f, u, body := profileConflictFixture(t)
	result := socialDataV2(f.request(t, "Alice", "POST", "/api/v1/admin/accounts/"+u.ID+"/profile-release", body, 200))
	caseID := result["caseId"].(string)
	input := map[string]any{"clientRequestId": "correct-claim-email", "expectedEmail": "rightful@example.com", "claimantEmail": "corrected@example.com", "reason": "工单核实收件邮箱填写错误", "password": deletionAdminPassword}
	f.request(t, "Bob", "POST", "/api/v1/admin/profile-reservations/"+caseID, input, 403)
	f.request(t, "Alice", "POST", "/api/v1/admin/profile-reservations/"+caseID, input, 200)
	f.request(t, "Alice", "POST", "/api/v1/admin/profile-reservations/"+caseID, input, 200)
	owner := registeredProfileSignup(t, f, "Occupied", "corrected@example.com")
	if owner.GameVerified {
		t.Fatal("email correction granted game ownership")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM identity_profile_reservation_edits WHERE case_id=?", caseID) != 1 {
		t.Fatal("correction audit duplicated/missing")
	}
	history := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/admin/accounts/"+u.ID+"/profile-resolutions", nil, 200))["items"].([]any)
	corrections := history[0].(map[string]any)["emailCorrections"].([]any)
	if len(corrections) != 1 || corrections[0].(map[string]any)["nextEmail"] != "corrected@example.com" {
		t.Fatal("correction is not reviewable", history)
	}
	input["clientRequestId"], input["expectedEmail"], input["claimantEmail"] = "correct-after-claim", "corrected@example.com", "someone@example.com"
	f.request(t, "Alice", "POST", "/api/v1/admin/profile-reservations/"+caseID, input, 409)
	if deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE id=? AND email='corrected@example.com'", owner.ID) != 1 {
		t.Fatal("completed registration was moved")
	}
}

func TestProfileConflictConcurrentReleaseAndRegistrationCannotStealReservation(t *testing.T) {
	f, u, body := profileConflictFixture(t)
	ctx := context.Background()
	encoded, _ := json.Marshal(body)
	var in store.ProfileReleaseRequest
	if err := json.Unmarshal(encoded, &in); err != nil {
		t.Fatal(err)
	}
	proof := store.ProfileReleaseProof{AccountDeletionProof: store.AccountDeletionProof{ActorID: f.users["Alice"].ID, SessionHash: store.Digest([]byte(f.tokens["Alice"]))}}
	if err := f.store.DB.QueryRow("SELECT password_hash FROM identities WHERE id=?", proof.ActorID).Scan(&proof.CredentialHash); err != nil {
		t.Fatal(err)
	}
	v := store.EmailVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(identity.Secret())), CodeHash: store.Digest([]byte("race-proof")), Email: "attacker@example.com", GameID: u.GameID, ProfileChecked: true, ProfileUUID: u.ProfileUUID, Purpose: "register", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewEmailVerification(ctx, v, "123456", f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan string, 2)
	errs := make(chan error, 3)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			value, err := f.store.ReleaseRegisteredProfile(ctx, u.ID, in, proof)
			results <- string(value)
			errs <- err
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		_, err := f.store.RegisterEmailUser(ctx, v, u.PasswordHash)
		if err == nil {
			errs <- errors.New("competing registration stole reserved UUID")
		} else if !errors.Is(err, store.ErrConflict) {
			var problem *store.CatalogErrorV2
			if !errors.As(err, &problem) || problem.Code != "GAME_PROFILE_RESERVED" {
				errs <- err
			}
		}
	}()
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	first := ""
	for response := range results {
		if first == "" {
			first = response
		} else if first != response {
			t.Error("concurrent release did not replay same result")
		}
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM identity_profile_resolutions WHERE target_user_id=?", u.ID) != 1 {
		t.Fatal("release was applied twice")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE email='attacker@example.com'") != 0 {
		t.Fatal("competing account was created")
	}
}

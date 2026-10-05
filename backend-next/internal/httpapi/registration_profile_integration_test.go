//go:build integration

package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func registrationProfileCode(t *testing.T, f *socialFixtureV2, name string) (store.EmailVerification, map[string]string) {
	t.Helper()
	token := identity.Secret()
	v := store.EmailVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(token)), CodeHash: store.Digest([]byte(token + ":123456")), Email: "signup@example.com", GameID: name, Purpose: "register", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewEmailVerification(context.Background(), v, "123456", f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	return v, map[string]string{"verificationToken": token, "code": "123456", "password": "test-password"}
}

func assertRegistrationProfileUnchanged(t *testing.T, f *socialFixtureV2, v store.EmailVerification) {
	t.Helper()
	var state string
	var attempts, accounts, aliases, sessions int
	if err := f.store.DB.QueryRow("SELECT state,attempts FROM email_verifications WHERE verification_id=?", v.ID).Scan(&state, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "ACTIVE" || attempts != 0 {
		t.Fatalf("lookup failure changed email proof: %s / %d", state, attempts)
	}
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM identities WHERE email=?", v.Email).Scan(&accounts); err != nil {
		t.Fatal(err)
	}
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM identity_aliases WHERE alias_key IN(LOWER(?),?)", v.GameID, v.Email).Scan(&aliases); err != nil {
		t.Fatal(err)
	}
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM identity_sessions").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if accounts != 0 || aliases != 0 || sessions != 3 {
		t.Fatalf("lookup failure wrote account state: accounts=%d aliases=%d sessions=%d", accounts, aliases, sessions)
	}
}

func TestRegistrationProfileRejectsMissingAndUnavailable(t *testing.T) {
	for _, path := range []string{"/api/v1/account/register", "/api/v1/web/register"} {
		for _, missing := range []bool{true, false} {
			t.Run(path+map[bool]string{true: "/missing", false: "/unavailable"}[missing], func(t *testing.T) {
				f := mailSupportFixture(t)
				v, input := registrationProfileCode(t, f, "NewPlayer")
				calls := 0
				f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
					calls++
					if name != v.GameID {
						t.Fatalf("lookup used unbound name %q", name)
					}
					if missing {
						return store.AdmissionProfile{}, errAdmissionPlayerMissing
					}
					return store.AdmissionProfile{}, errors.New("upstream unavailable")
				}
				headers := map[string]string{}
				if path == "/api/v1/web/register" {
					headers["Origin"] = f.http.URL
				}
				status, code := http.StatusServiceUnavailable, "MINECRAFT_LOOKUP_UNAVAILABLE"
				if missing {
					status, code = http.StatusBadRequest, "MINECRAFT_PLAYER_NOT_FOUND"
				}
				result := admissionRaw(t, f, path, input, headers, status)
				if calls != 1 || result["error"].(map[string]any)["code"] != code {
					t.Fatalf("unexpected profile failure: %v calls=%d", result, calls)
				}
				assertRegistrationProfileUnchanged(t, f, v)
				// A failed lookup must permit retry with the same still-valid proof.
				f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
					return store.AdmissionProfile{UUID: admissionTestUUID, Name: "newplayer"}, nil
				}
				admissionRaw(t, f, path, input, headers, http.StatusOK)
				u, err := f.store.UserByAlias(context.Background(), v.Email)
				if err != nil || u.GameVerified || u.ServerUUID != "" || u.GameID != v.GameID {
					t.Fatalf("existence check changed ownership or ID: %v %v", u.GameID, err)
				}
				admissionRaw(t, f, path, input, headers, http.StatusBadRequest)
				var count int
				if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM identities WHERE email=?", v.Email).Scan(&count); err != nil || count != 1 {
					t.Fatalf("registration proof reused: count=%d err=%v", count, err)
				}
			})
		}
	}
}

func TestRegistrationProfileInvalidNameAndProofOrdering(t *testing.T) {
	t.Run("invalid-name", func(t *testing.T) {
		f := mailSupportFixture(t)
		v, input := registrationProfileCode(t, f, "NameLongerThanSixteen")
		f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
			t.Fatal("invalid name reached upstream")
			return store.AdmissionProfile{}, nil
		}
		result := f.request(t, "", "POST", "/api/v1/account/register", input, 400)
		if result["error"].(map[string]any)["code"] != "MINECRAFT_PLAYER_NOT_FOUND" {
			t.Fatal(result)
		}
		assertRegistrationProfileUnchanged(t, f, v)
	})
	t.Run("invalid-code-and-origin", func(t *testing.T) {
		f := mailSupportFixture(t)
		_, input := registrationProfileCode(t, f, "NewPlayer")
		f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
			t.Fatal("rejected request reached upstream")
			return store.AdmissionProfile{}, nil
		}
		admissionRaw(t, f, "/api/v1/web/register", input, map[string]string{"Origin": "https://untrusted.invalid"}, 403)
		input["code"] = "654321"
		f.request(t, "", "POST", "/api/v1/account/register", input, 400)
	})
}

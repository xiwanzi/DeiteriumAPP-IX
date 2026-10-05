//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestRegistrationCodeProfileFailureDoesNotSendMail(t *testing.T) {
	for _, missing := range []bool{true, false} {
		t.Run(map[bool]string{true: "missing", false: "unavailable"}[missing], func(t *testing.T) {
			f := mailSupportFixture(t)
			f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
				if missing {
					return store.AdmissionProfile{}, errAdmissionPlayerMissing
				}
				return store.AdmissionProfile{}, errors.New("lookup unavailable")
			}
			status, code := 503, "MINECRAFT_LOOKUP_UNAVAILABLE"
			if missing {
				status, code = 400, "MINECRAFT_PLAYER_NOT_FOUND"
			}
			r := f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "signup@example.com"}, status)
			if r["error"].(map[string]any)["code"] != code {
				t.Fatal(r)
			}
			for _, table := range []string{"email_verifications", "mail_outbox", "minecraft_profile_checks"} {
				var count int
				if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
					t.Fatalf("failed lookup wrote %s: %d %v", table, count, err)
				}
			}
		})
	}
}

func registrationIssuedCode(t *testing.T, f *socialFixtureV2, token string) string {
	t.Helper()
	var cipher string
	if err := f.store.DB.QueryRow("SELECT m.payload_cipher FROM mail_outbox m JOIN email_verifications v ON m.reference_id=v.verification_id WHERE v.token_hash=?", store.Digest([]byte(token))).Scan(&cipher); err != nil {
		t.Fatal(err)
	}
	plain, err := notify.Open(f.app.Config.SMTPKey, cipher)
	if err != nil {
		t.Fatal(err)
	}
	var mail notify.VerificationMail
	if err = json.Unmarshal([]byte(plain), &mail); err != nil {
		t.Fatal(err)
	}
	return mail.Code
}

func TestRegistrationCodeProfileCacheAndFinalSubmit(t *testing.T) {
	f := mailSupportFixture(t)
	calls := 0
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		calls++
		return store.AdmissionProfile{UUID: admissionTestUUID, Name: name}, nil
	}
	first := socialDataV2(f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "one@example.com"}, 200))
	token := first["verificationToken"].(string)
	code := registrationIssuedCode(t, f, token)
	var checked bool
	if err := f.store.DB.QueryRow("SELECT profile_checked FROM email_verifications WHERE token_hash=?", store.Digest([]byte(token))).Scan(&checked); err != nil || !checked {
		t.Fatal("proof did not record validated ID", err)
	}
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "newplayer", "email": "two@example.com"}, 200)
	if calls != 1 {
		t.Fatalf("changing email or ID case repeated lookup: %d", calls)
	}
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "OtherPlayer", "email": "three@example.com"}, 200)
	if calls != 2 {
		t.Fatal("changed ID reused wrong profile")
	}
	if _, err := f.store.DB.Exec("UPDATE minecraft_profile_checks SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE game_id='newplayer'"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "four@example.com"}, 200)
	if calls != 3 {
		t.Fatal("expired success cache reused")
	}
	// Minted proof remains sufficient even after cache expiry or service restart.
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		t.Fatal("final submit called upstream")
		return store.AdmissionProfile{}, nil
	}
	f.request(t, "", "POST", "/api/v1/account/register", map[string]string{"verificationToken": token, "code": code, "password": "test-password"}, 200)
	u, err := f.store.UserByAlias(context.Background(), "one@example.com")
	if err != nil || u.GameVerified || u.ServerUUID != "" {
		t.Fatal("existence check granted ownership", err)
	}
	var state string
	if err := f.store.DB.QueryRow("SELECT state FROM email_verifications WHERE token_hash=?", store.Digest([]byte(token))).Scan(&state); err != nil || state != "CONSUMED" {
		t.Fatal("proof was not consumed once", err)
	}
}

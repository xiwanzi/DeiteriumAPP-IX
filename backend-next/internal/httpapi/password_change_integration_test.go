//go:build integration

package httpapi

import (
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"strings"
	"testing"
)

func TestPasswordChangeProofPurposeOwnerAndSessionRevocation(t *testing.T) {
	f := mailSupportFixture(t)
	emailSignup(t, f, "PasswordUser", "password-user@example.com")
	emailSignup(t, f, "OtherUser", "other-user@example.com")
	if _, err := f.store.DB.Exec("UPDATE game_verification_cooldowns SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)"); err != nil {
		t.Fatal(err)
	}
	response := socialDataV2(f.request(t, "PasswordUser", "POST", "/api/v1/account/password-change-code", map[string]any{}, 200))
	token := response["verificationToken"].(string)
	var cipher string
	if err := f.store.DB.QueryRow("SELECT o.payload_cipher FROM mail_outbox o JOIN email_verifications v ON v.verification_id=o.reference_id WHERE v.token_hash=?", store.Digest([]byte(token))).Scan(&cipher); err != nil {
		t.Fatal(err)
	}
	plain, err := notify.Open(f.app.Config.SMTPKey, cipher)
	if err != nil {
		t.Fatal(err)
	}
	var payload notify.VerificationMail
	if json.Unmarshal([]byte(plain), &payload) != nil || payload.Purpose != "password_change" || payload.GameID != "PasswordUser" || len(payload.Code) != 6 {
		t.Fatal("wrong change template payload")
	}
	body := map[string]string{"verificationToken": token, "code": payload.Code, "newPassword": "new-password-123"}
	f.request(t, "", "POST", "/api/v1/account/password-reset", body, 400)
	f.request(t, "OtherUser", "POST", "/api/v1/account/password-change", body, 400)
	f.request(t, "PasswordUser", "POST", "/api/v1/account/password-change", body, 200)
	f.request(t, "PasswordUser", "GET", "/api/v1/account/me", nil, 401)
	f.request(t, "OtherUser", "GET", "/api/v1/account/me", nil, 200)
	f.request(t, "", "POST", "/api/v1/account/login", map[string]string{"account": "password-user@example.com", "password": "test-password"}, 401)
	f.request(t, "", "POST", "/api/v1/account/login", map[string]string{"account": "password-user@example.com", "password": "new-password-123"}, 200)
	for _, purpose := range []string{"register", "password_reset", "password_change"} {
		r := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/admin/mail/templates/verification/"+purpose, nil, 200))
		if !strings.Contains(r["html"].(string), `<img src="https://xiwanzi.cn-nb2.rains3.com/deuterium-official/mail-assets/`) || r["preview"] != true {
			t.Fatal("preview missing published HTTPS logo or preview flag")
		}
	}
	f.request(t, "Bob", "GET", "/api/v1/admin/mail/templates/verification/register", nil, 403)
}

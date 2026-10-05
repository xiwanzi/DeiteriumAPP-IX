//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func invalidEmailFixture(t *testing.T) (*socialFixtureV2, string) {
	t.Helper()
	f := mailSupportFixture(t)
	auditProfile(f)
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "BlockedPlayer", "email": "blocked@example.com"}, 200)
	var event string
	if err := f.store.DB.QueryRow("SELECT event_id FROM mail_outbox WHERE recipient='blocked@example.com'").Scan(&event); err != nil {
		t.Fatal(err)
	}
	return f, event
}

func TestInvalidEmailSMTPFreezeBlocksRequestsAndExpires(t *testing.T) {
	f, event := invalidEmailFixture(t)
	ctx := context.Background()
	if err := f.app.processSystemMail(ctx, func(context.Context, notify.Settings, string, string, notify.Message) error {
		return notify.ErrInvalidRecipient
	}); err != nil {
		t.Fatal(err)
	}
	var state, cipher string
	var until time.Time
	if err := f.store.DB.QueryRow("SELECT status,payload_cipher FROM mail_outbox WHERE event_id=?", event).Scan(&state, &cipher); err != nil || state != "FAILED" || cipher != "" {
		t.Fatal("invalid mail retried or retained code", state, err)
	}
	if err := f.store.DB.QueryRow("SELECT blocked_until FROM mail_invalid_addresses WHERE event_id=?", event).Scan(&until); err != nil || time.Until(until) < 23*time.Hour+59*time.Minute || time.Until(until) > 24*time.Hour {
		t.Fatal("wrong freeze duration", until, err)
	}
	for i := 0; i < 3; i++ {
		r := f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "AnotherPlayer", "email": "BLOCKED@example.com"}, 400)
		e := r["error"].(map[string]any)
		if e["code"] != "EMAIL_ADDRESS_INVALID" || e["message"] != "邮箱地址无效" {
			t.Fatal("wrong public error", e)
		}
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _, err := f.call("", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "AnotherPlayer", "email": "blocked@example.com"})
			if err != nil {
				statuses <- 0
			} else {
				statuses <- status
			}
		}()
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != 400 {
			t.Fatal("concurrent request bypassed freeze", status)
		}
	}
	var attempts, mails int
	if err := f.store.DB.QueryRow("SELECT attempts FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("verification-ip:127.0.0.1"))).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatal("blocked requests spent send quota", attempts, err)
	}
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM mail_outbox").Scan(&mails); err != nil || mails != 1 {
		t.Fatal("blocked requests queued mail", mails, err)
	}
	if err := f.store.NewEmailVerification(ctx, store.EmailVerification{Email: "blocked@example.com"}, "123456", f.app.Config.SMTPKey); !errors.Is(err, store.ErrEmailAddressInvalid) {
		t.Fatal("transaction bypassed freeze", err)
	}
	if _, err := f.store.RecordInvalidEmail(ctx, event, "blocked@example.com", "DM_EVENT", time.Now()); err != nil {
		t.Fatal(err)
	}
	var repeated time.Time
	if err := f.store.DB.QueryRow("SELECT blocked_until FROM mail_invalid_addresses WHERE event_id=?", event).Scan(&repeated); err != nil || !until.Equal(repeated) {
		t.Fatal("duplicate extended freeze", err)
	}
	if _, err := f.store.DB.Exec("UPDATE mail_invalid_addresses SET blocked_until=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND)"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.RecordInvalidEmail(ctx, event, "blocked@example.com", "DM_EVENT", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.CheckEmailAddress(ctx, "blocked@example.com"); err != nil {
		t.Fatal("duplicate resurrected expired freeze", err)
	}
	if _, err := f.store.DB.Exec("UPDATE game_verification_cooldowns SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "AnotherPlayer", "email": "blocked@example.com"}, 200)
}

func TestInvalidEmailCallbackAuthenticationMatchingAndLateFinish(t *testing.T) {
	f, event := invalidEmailFixture(t)
	ctx := context.Background()
	f.app.Config.DMEventToken = strings.Repeat("test-event-token-", 3)
	e, err := f.store.ClaimMail(ctx)
	if err != nil {
		t.Fatal(err)
	}
	post := func(token, source, status, recipient, id string, at time.Time, want int) bool {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"source": source, "type": "dm:Deliver:Fail", "time": at, "data": map[string]any{"status": status, "rcpt": recipient, "msg_id": id}})
		req, _ := http.NewRequest("POST", f.http.URL+"/api/v1/mail/aliyun/events", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-eventbridge-signature-token", token)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		var out struct {
			Data struct {
				Recorded bool `json:"recorded"`
			} `json:"data"`
		}
		if err = json.NewDecoder(r.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if r.StatusCode != want {
			t.Fatalf("callback status %d expected %d", r.StatusCode, want)
		}
		return out.Data.Recorded
	}
	token := f.app.Config.DMEventToken
	id := "<" + event + "@deuterium-notifications>"
	now := time.Now().UTC()
	post("wrong", "acs:dm", "2", "blocked@example.com", id, now, 401)
	for _, tc := range []struct {
		source, status, email, id string
		at                        time.Time
	}{
		{"acs:dm", "3", "blocked@example.com", id, now},
		{"acs:dm", "4", "blocked@example.com", id, now},
		{"untrusted", "2", "blocked@example.com", id, now},
		{"acs:dm", "2", "other@example.com", id, now},
		{"acs:dm", "2", "blocked@example.com", "unrelated@elsewhere", now},
		{"acs:dm", "2", "blocked@example.com", id, now.Add(-25 * time.Hour)},
	} {
		if post(token, tc.source, tc.status, tc.email, tc.id, tc.at, 200) {
			t.Fatal("unrelated failure froze email")
		}
	}
	post(token, "acs:dm", "2", "blocked@example.com", id, time.Time{}, 400)
	post(token, "acs:dm", "2", "blocked@example.com", id, now.Add(time.Hour), 400)
	if err := f.store.CheckEmailAddress(ctx, "blocked@example.com"); err != nil {
		t.Fatal("invalid callback changed state", err)
	}
	if !post(token, "acs:dm", "2", "blocked@example.com", id, now, 200) {
		t.Fatal("valid callback ignored")
	}
	// SMTP acceptance arriving after the invalid-address event cannot erase it.
	if err := f.store.FinishMail(ctx, e, nil); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := f.store.DB.QueryRow("SELECT status FROM mail_outbox WHERE event_id=?", event).Scan(&state); err != nil || state != "FAILED" {
		t.Fatal("late acceptance overwrote failure", state, err)
	}
	if !errors.Is(f.store.CheckEmailAddress(ctx, "blocked@example.com"), store.ErrEmailAddressInvalid) {
		t.Fatal("valid callback did not persist freeze")
	}
	f.app.Config.DMEventToken = ""
	post(token, "acs:dm", "2", "blocked@example.com", id, now, 503)
}

func TestInvalidEmailPendingCodeAndPasswordResetQuota(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	emailSignup(t, f, "ResetPlayer", "reset@example.com")
	if _, err := f.store.DB.Exec("UPDATE game_verification_cooldowns SET expires_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE)"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "", "POST", "/api/v1/account/password-reset-code", map[string]string{"account": "ResetPlayer"}, 200)
	f.request(t, "", "POST", "/api/v1/account/password-reset-code", map[string]string{"account": "ResetPlayer"}, 429)
	var n int
	if err := f.store.DB.QueryRow("SELECT attempts FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("verification-ip:127.0.0.1"))).Scan(&n); err != nil || n != 1 {
		t.Fatal("reset alias double charged", n, err)
	}
	var event string
	if err := f.store.DB.QueryRow("SELECT event_id FROM mail_outbox WHERE kind='CODE_PASSWORD_RESET'").Scan(&event); err != nil {
		t.Fatal(err)
	}
	// A previously sent message receives an invalid result while a newer code is queued.
	if _, err := f.store.DB.Exec("UPDATE mail_outbox SET attempts=1,status='SENT' WHERE kind='CODE_REGISTER'"); err != nil {
		t.Fatal(err)
	}
	var original string
	if err := f.store.DB.QueryRow("SELECT event_id FROM mail_outbox WHERE kind='CODE_REGISTER'").Scan(&original); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.RecordInvalidEmail(ctx, original, "reset@example.com", "DM_EVENT", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.ClaimMail(ctx); !errors.Is(err, store.ErrMailSkipped) {
		t.Fatal("frozen pending code claimed", err)
	}
	var state string
	if err := f.store.DB.QueryRow("SELECT status FROM mail_outbox WHERE event_id=?", event).Scan(&state); err != nil || state != "FAILED" {
		t.Fatal("pending code not stopped", state, err)
	}
	for _, account := range []string{"ResetPlayer", "reset@example.com"} {
		r := f.request(t, "", "POST", "/api/v1/account/password-reset-code", map[string]string{"account": account}, 400)
		if r["error"].(map[string]any)["code"] != "EMAIL_ADDRESS_INVALID" {
			t.Fatal("reset bypassed freeze")
		}
	}
	f.request(t, "", "POST", "/api/v1/account/password-change-code", map[string]any{}, 401)
	f.request(t, "ResetPlayer", "POST", "/api/v1/account/password-change-code", map[string]any{}, 400)
	f.request(t, "", "POST", "/api/v1/account/login", map[string]string{"account": "reset@example.com", "password": "test-password"}, 200)
	if err := f.store.DB.QueryRow("SELECT attempts FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("verification-ip:127.0.0.1"))).Scan(&n); err != nil || n != 1 {
		t.Fatal("frozen password requests spent quota", n, err)
	}
}

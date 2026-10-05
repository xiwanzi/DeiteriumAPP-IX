//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func auditProfile(f *socialFixtureV2) {
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: admissionTestUUID, Name: name}, nil
	}
}

func TestAuthAuditFailedLookupDoesNotExhaustSendQuota(t *testing.T) {
	f := mailSupportFixture(t)
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{}, errAdmissionPlayerMissing
	}
	for n := 0; n < 15; n++ {
		f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "MissingPlayer", "email": "unused@example.com"}, 400)
	}
	var attempts int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("verification-ip:127.0.0.1"))).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal("failed profile consumed send budget", attempts, err)
	}
	auditProfile(f)
	f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "new@example.com", "password": "ignored-for-old-app"}, 200)
	if err := f.store.DB.QueryRow("SELECT attempts FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("verification-ip:127.0.0.1"))).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatal("successful mail did not consume exactly one send", attempts, err)
	}
}

func TestAuthAuditCooldownConcurrentSendAndDailyLimit(t *testing.T) {
	f := mailSupportFixture(t)
	auditProfile(f)
	if err := f.store.CacheMinecraftProfileCheck(context.Background(), "NewPlayer"); err != nil {
		t.Fatal(err)
	}
	input := map[string]string{"gameId": "NewPlayer", "email": "same@example.com"}
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _, err := f.call("", "POST", "/api/v1/account/registration-code", input)
			if err != nil {
				statuses <- 0
			} else {
				statuses <- status
			}
		}()
	}
	wg.Wait()
	close(statuses)
	ok, limited := 0, 0
	for status := range statuses {
		if status == 200 {
			ok++
		} else if status == 429 {
			limited++
		}
	}
	if ok != 1 || limited != 1 {
		t.Fatal("concurrent duplicate send", ok, limited)
	}
	r := f.request(t, "", "POST", "/api/v1/account/registration-code", input, 429)
	retry := r["error"].(map[string]any)["retryAfterSeconds"].(float64)
	if retry < 1 || retry > 60 {
		t.Fatal("wrong cooldown duration", retry)
	}
	var quota, mails int
	if err := f.store.DB.QueryRow("SELECT attempts FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("verification-ip:127.0.0.1"))).Scan(&quota); err != nil || quota != 1 {
		t.Fatal("rejected cooldown consumed quota", quota, err)
	}
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM mail_outbox").Scan(&mails); err != nil || mails != 1 {
		t.Fatal("duplicate mail", mails, err)
	}
	for i := 0; i < 20; i++ {
		_, err := f.store.DB.Exec("INSERT INTO email_verifications(verification_id,token_hash,code_hash,email,game_id,purpose,state,expires_at,created_at) VALUES(?,?,?,'daily@example.com','NewPlayer','register','EXPIRED',UTC_TIMESTAMP(6),DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 2 HOUR))", store.ID("verify_"), store.Digest([]byte(fmt.Sprint(i))), strings.Repeat("0", 64))
		if err != nil {
			t.Fatal(err)
		}
	}
	r = f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "daily@example.com"}, 429)
	retry = r["error"].(map[string]any)["retryAfterSeconds"].(float64)
	if retry < 21*3600 || retry > 23*3600 {
		t.Fatal("daily limit used a 60s retry", retry)
	}
}

func TestAuthAuditSendBudgetIsBoundedAndReportsRemainingTime(t *testing.T) {
	f := mailSupportFixture(t)
	auditProfile(f)
	for n := 0; n < 10; n++ {
		f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": fmt.Sprintf("send%d@example.com", n)}, 200)
	}
	r := f.request(t, "", "POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "eleventh@example.com"}, 429)
	retry := r["error"].(map[string]any)["retryAfterSeconds"].(float64)
	if retry <= 60 || retry > 900 {
		t.Fatal("IP wait was not actual 15min remainder", retry)
	}
}

func TestAuthAuditFullRegistrationLoginAndLogout(t *testing.T) {
	for _, origin := range []string{"app", "https://www.deuteriumix.com", "https://help.deuteriumix.com", "https://chat.deuteriumix.com"} {
		t.Run(origin, func(t *testing.T) {
			f := mailSupportFixture(t)
			auditProfile(f)
			f.app.Config.AdditionalPublicOrigins = []string{"https://www.deuteriumix.com", "https://help.deuteriumix.com", "https://chat.deuteriumix.com"}
			jar, _ := cookiejar.New(nil)
			client := &http.Client{Jar: jar}
			token, csrf := "", ""
			call := func(method, path string, input any, status int) map[string]any {
				t.Helper()
				encoded, _ := json.Marshal(input)
				req, _ := http.NewRequest(method, f.http.URL+path, strings.NewReader(string(encoded)))
				if input != nil {
					req.Header.Set("Content-Type", "application/json")
				}
				if origin != "app" {
					req.Header.Set("Origin", origin)
				} else if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				if csrf != "" {
					req.Header.Set("X-CSRF-Token", csrf)
				}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				var body map[string]any
				if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != status {
					t.Fatalf("%s %s: %d want %d: %v", method, path, resp.StatusCode, status, body)
				}
				return body
			}
			issue := socialDataV2(call("POST", "/api/v1/account/registration-code", map[string]string{"gameId": "NewPlayer", "email": "full@example.com"}, 200))
			proof := issue["verificationToken"].(string)
			code := registrationIssuedCode(t, f, proof)
			path := "/api/v1/account/register"
			if origin != "app" {
				path = "/api/v1/web/register"
			}
			body := map[string]string{"verificationToken": proof, "code": "999999", "password": "test-password"}
			if code == body["code"] {
				body["code"] = "888888"
			}
			call("POST", path, body, 400)
			body["code"] = code
			created := socialDataV2(call("POST", path, body, 200))
			user := created["user"].(map[string]any)
			if user["gameVerified"] != false || user["gameId"] != "NewPlayer" {
				t.Fatal("registration bypassed game proof", user)
			}
			call("POST", path, body, 400)
			if origin == "app" {
				token = created["token"].(string)
				call("GET", "/api/v1/account/me", nil, 200)
				call("POST", "/api/v1/account/logout", map[string]any{}, 200)
				call("GET", "/api/v1/account/me", nil, 401)
				token = ""
			} else {
				csrf = created["csrfToken"].(string)
				call("GET", "/api/v1/web/session", nil, 200)
				call("DELETE", "/api/v1/web/session", nil, 200)
				call("GET", "/api/v1/web/session", nil, 401)
				csrf = ""
			}
			loginPath := "/api/v1/account/login"
			if origin != "app" {
				loginPath = "/api/v1/web/session"
			}
			call("POST", loginPath, map[string]string{"account": "full@example.com", "password": "wrong-password"}, 401)
			login := socialDataV2(call("POST", loginPath, map[string]string{"account": " FULL@EXAMPLE.COM ", "password": "test-password"}, 200))
			if login["user"].(map[string]any)["userId"] != user["userId"] {
				t.Fatal("login changed account")
			}
			if origin != "app" {
				expires, err := time.Parse(time.RFC3339Nano, login["expiresAt"].(string))
				if err != nil || time.Until(expires) < 7*24*time.Hour-time.Minute {
					t.Fatal("web session shortened", err)
				}
				csrf = login["csrfToken"].(string)
				call("GET", "/api/v1/web/session", nil, 200)
				call("DELETE", "/api/v1/web/session", nil, 200)
			} else {
				token = login["token"].(string)
				call("GET", "/api/v1/account/me", nil, 200)
				call("POST", "/api/v1/account/logout", map[string]any{}, 200)
			}
		})
	}
}

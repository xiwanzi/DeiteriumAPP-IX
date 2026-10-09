//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestYSMCloudAdminBinaryUploadAuditAndRevocation(t *testing.T) {
	f := newSocialFixtureV2(t)
	csrf := strings.Repeat("c", 43)
	payload := bytes.Repeat([]byte{0, 255, 17}, 90000)
	var uploads atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/config":
			http.SetCookie(w, &http.Cookie{Name: "dysm_csrf", Value: csrf})
			json.NewEncoder(w).Encode(map[string]any{"csrfToken": csrf})
		case "/api/cloud/connection":
			json.NewEncoder(w).Encode(map[string]any{"csrfToken": csrf, "origin": "https://cloud.example.invalid", "tokenConfigured": true})
		case "/api/cloud/upload":
			uploads.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || !bytes.Equal(body, payload) || r.ContentLength != int64(len(payload)) {
				t.Error("binary upload changed")
			}
			if r.Header.Get("Authorization") != "" || r.Header.Get("X-CSRF-Token") != csrf || r.Header.Get("Idempotency-Key") != "stable-upload" {
				t.Error("credential or metadata boundary failed")
			}
			w.WriteHeader(201)
			io.WriteString(w, `{"revision":1}`)
		case "/api/cloud/models/d9_182faf5bbb8d912fe91251dfb01ee3656afd0e87/archive":
			io.WriteString(w, `{"ok":true}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer upstream.Close()
	f.app.ysm = newYSMBridge(config.Config{YSMAdminURL: upstream.URL})
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Bob", "GET", ysmPrefix+"cloud/connection", nil, 403)
	data := socialDataV2(f.request(t, "Alice", "GET", ysmPrefix+"cloud/connection", nil, 200))
	if _, exists := data["csrfToken"]; exists {
		t.Fatal("upstream csrf exposed")
	}
	request, _ := http.NewRequest("POST", f.http.URL+ysmPrefix+"cloud/upload", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+f.tokens["Alice"])
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Idempotency-Key", "stable-upload")
	response, err := f.http.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("upload: %d %s", response.StatusCode, raw)
	}
	var audits int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM audit_events_next WHERE actor_id=? AND action IN ('ysm.cloud.upload.requested','ysm.cloud.upload.accepted')", f.users["Alice"].ID).Scan(&audits); err != nil || audits != 2 {
		t.Fatalf("audit: %d %v", audits, err)
	}
	f.request(t, "Alice", "POST", ysmPrefix+"cloud/models/d9_182faf5bbb8d912fe91251dfb01ee3656afd0e87/archive", map[string]any{"expectedVersion": 1}, 200)
	if err := f.store.Revoke(ctx, store.Digest([]byte(f.tokens["Alice"]))); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Alice", "POST", ysmPrefix+"cloud/models/a/archive", map[string]any{"expectedVersion": 1}, 401)
	if uploads.Load() != 1 {
		t.Fatal("unexpected extra upload")
	}
}

func TestYSMAdminAuthorizationPreviewConfirmAndRevocation(t *testing.T) {
	f := newSocialFixtureV2(t)
	var calls, confirms atomic.Int32
	csrf := strings.Repeat("c", 43)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/config":
			http.SetCookie(w, &http.Cookie{Name: "dysm_csrf", Value: csrf})
			json.NewEncoder(w).Encode(map[string]any{"config": map[string]any{"configRevision": 2}, "csrfToken": csrf})
		case "/api/models/delete/preview":
			cookie, err := r.Cookie("dysm_csrf")
			if err != nil || cookie.Value != csrf || r.Header.Get("X-CSRF-Token") != csrf {
				t.Error("upstream CSRF missing")
			}
			json.NewEncoder(w).Encode(map[string]any{"confirmationToken": "upstream-confirmation", "expiresAtEpochSecond": time.Now().Add(time.Minute).Unix(), "modelId": "中文.ysm"})
		case "/api/models/confirm":
			if confirms.Add(1) > 1 {
				w.WriteHeader(409)
				io.WriteString(w, `{"error":"already consumed"}`)
				return
			}
			var input map[string]string
			json.NewDecoder(r.Body).Decode(&input)
			if input["confirmationToken"] != "upstream-confirmation" {
				t.Error("wrong upstream token")
			}
			w.WriteHeader(202)
			io.WriteString(w, `{"taskId":"task-one"}`)
		default:
			t.Errorf("unexpected upstream %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer upstream.Close()
	f.app.ysm = newYSMBridge(config.Config{YSMAdminURL: upstream.URL})
	ctx := context.Background()
	for _, user := range []string{"Alice", "Carol"} {
		if err := f.store.Grant(ctx, f.users[user].ID, "platform.admin"); err != nil {
			t.Fatal(err)
		}
	}
	f.request(t, "", "GET", ysmPrefix+"config", nil, 401)
	f.request(t, "Bob", "GET", ysmPrefix+"config", nil, 403)
	f.request(t, "Bob", "POST", ysmPrefix+"models/delete/preview", map[string]any{}, 403)
	f.request(t, "Alice", "GET", ysmPrefix+"unknown", nil, 404)
	f.request(t, "Alice", "GET", ysmPrefix+"config?target=evil", nil, 404)
	f.request(t, "Alice", "DELETE", ysmPrefix+"config", nil, 405)
	if calls.Load() != 0 {
		t.Fatal("rejected request reached upstream")
	}
	data := socialDataV2(f.request(t, "Alice", "GET", ysmPrefix+"config", nil, 200))
	if _, exists := data["csrfToken"]; exists {
		t.Fatal("upstream csrf exposed")
	}
	preview := socialDataV2(f.request(t, "Alice", "POST", ysmPrefix+"models/delete/preview", map[string]any{"modelId": "中文.ysm", "expectedRevision": 3}, 200))
	input := map[string]any{"confirmationToken": preview["confirmationToken"]}
	f.request(t, "Carol", "POST", ysmPrefix+"models/confirm", input, 409)
	f.request(t, "Alice", "POST", ysmPrefix+"config/confirm", input, 409)
	if confirms.Load() != 0 {
		t.Fatal("cross-session confirmation reached upstream")
	}
	f.request(t, "Alice", "POST", ysmPrefix+"models/confirm", input, 200)
	f.request(t, "Alice", "POST", ysmPrefix+"models/confirm", input, 409)
	var audits int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM audit_events_next WHERE actor_id=? AND action='ysm.models.confirm.accepted'", f.users["Alice"].ID).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("actor audit: %d %v", audits, err)
	}
	if err := f.store.Revoke(ctx, store.Digest([]byte(f.tokens["Alice"]))); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Alice", "GET", ysmPrefix+"config", nil, 401)
	f.request(t, "Alice", "POST", ysmPrefix+"models/confirm", input, 401)

	// Browser mutations use the existing Origin + session CSRF gate before proxying.
	token, webCSRF := identity.Secret(), identity.Secret()
	if err := f.store.CreateSession(ctx, f.users["Carol"], store.Digest([]byte(token)), "web", webCSRF, time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		origin, csrf string
		status       int
	}{{f.http.URL, "", 403}, {"https://evil.example", webCSRF, 403}, {f.http.URL, webCSRF, 200}} {
		r, _ := http.NewRequest("POST", f.http.URL+ysmPrefix+"models/delete/preview", strings.NewReader(`{"modelId":"中文.ysm","expectedRevision":3}`))
		r.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: token})
		r.Header.Set("Origin", attempt.origin)
		r.Header.Set("X-CSRF-Token", attempt.csrf)
		response, err := f.http.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != attempt.status {
			t.Fatalf("browser status %d want %d", response.StatusCode, attempt.status)
		}
	}
}

func TestYSMUnavailableUploadLimitAndUnknownConfirmation(t *testing.T) {
	f := newSocialFixtureV2(t)
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Alice", "GET", ysmPrefix+"config", nil, 503)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/models/confirm" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		time.Sleep(100 * time.Millisecond)
		io.WriteString(w, `{"taskId":"task-accepted-before-timeout"}`)
	}))
	defer upstream.Close()
	f.app.ysm = newYSMBridge(config.Config{YSMAdminURL: upstream.URL})
	f.app.ysm.client.Timeout = 20 * time.Millisecond
	r := httptest.NewRequest("POST", ysmPrefix+"models/stage", strings.NewReader("large"))
	r.Header.Set("Authorization", "Bearer "+f.tokens["Alice"])
	r.ContentLength = ysmUploadLimit + 1
	w := httptest.NewRecorder()
	f.app.Handler().ServeHTTP(w, r)
	if w.Code != 413 || calls.Load() != 0 {
		t.Fatalf("oversized upload reached upstream: %d", w.Code)
	}
	sealed, err := f.app.ysm.sealConfirmation(ysmConfirmation{"confirm", strings.Repeat("c", 43), time.Now().Add(time.Minute).Unix()}, store.Digest([]byte(f.tokens["Alice"])), "models/confirm")
	if err != nil {
		t.Fatal(err)
	}
	body := f.request(t, "Alice", "POST", ysmPrefix+"models/confirm", map[string]any{"confirmationToken": sealed}, 504)
	if body["error"].(map[string]any)["code"] != "YSM_RESULT_UNKNOWN" || calls.Load() != 1 {
		t.Fatal("unknown confirmation must not retry")
	}
}

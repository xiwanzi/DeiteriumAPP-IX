package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
)

func TestYSMConfirmationsBindLoginActionAndExpiry(t *testing.T) {
	b := newYSMBridge(config.Config{YSMAdminURL: "http://127.0.0.1:17656"})
	v := ysmConfirmation{Token: "upstream-confirmation", CSRF: "upstream-csrf", Expires: time.Now().Add(time.Minute).Unix()}
	ticket, err := b.sealConfirmation(v, "login-one", "models/confirm")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ticket, v.Token) || strings.Contains(ticket, v.CSRF) {
		t.Fatal("upstream credential exposed")
	}
	got, err := b.openConfirmation(ticket, "login-one", "models/confirm")
	if err != nil || got != v {
		t.Fatalf("valid confirmation failed: %v", err)
	}
	for _, scope := range [][2]string{{"login-two", "models/confirm"}, {"login-one", "config/confirm"}} {
		if _, err := b.openConfirmation(ticket, scope[0], scope[1]); err == nil {
			t.Fatal("cross-session/action confirmation accepted")
		}
	}
	changed := []byte(ticket)
	changed[20] = 'A'
	if ticket[20] == 'A' {
		changed[20] = 'B'
	}
	if _, err := b.openConfirmation(string(changed), "login-one", "models/confirm"); err == nil {
		t.Fatal("tampering accepted")
	}
	v.Expires = time.Now().Add(-time.Second).Unix()
	expired, _ := b.sealConfirmation(v, "login-one", "models/confirm")
	if _, err := b.openConfirmation(expired, "login-one", "models/confirm"); err == nil {
		t.Fatal("expired confirmation accepted")
	}
	other := newYSMBridge(config.Config{YSMAdminURL: "http://127.0.0.1:17656"})
	if _, err := other.openConfirmation(ticket, "login-one", "models/confirm"); err == nil {
		t.Fatal("restart must invalidate confirmation")
	}
}

func TestYSMForwardingDropsBrowserCredentialsAndRedirects(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/models/stage" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Host != "127.0.0.1:17656" || r.Header.Get("Origin") != "http://127.0.0.1:17656" || r.Header.Get("Referer") != "http://127.0.0.1:17656/" {
			t.Error("loopback authority not preserved")
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Forwarded-Host") != "" || r.Header.Get("Cookie") != "dysm_csrf=upstream-token" {
			t.Error("browser credential/forwarded header leaked")
		}
		if r.Header.Get("X-CSRF-Token") != "upstream-token" || r.Header.Get("X-Model-Id") != "%E6%A8%A1%E5%9E%8B.ysm" {
			t.Error("upload headers not preserved")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "model-bytes" {
			t.Error("binary upload changed")
		}
		http.Redirect(w, r, "/unexpected", http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()
	b := newYSMBridge(config.Config{YSMAdminURL: upstream.URL, YSMAdminOrigin: "http://127.0.0.1:17656"})
	h := http.Header{"Authorization": {"Bearer secret"}, "Cookie": {"deuterium_session=secret"}, "X-Csrf-Token": {"browser-token"}, "X-Forwarded-Host": {"evil"}, "X-Model-Id": {"%E6%A8%A1%E5%9E%8B.ysm"}, "Content-Type": {"application/zip"}}
	r, err := b.request(context.Background(), "models/stage", "POST", "upstream-token", strings.NewReader("model-bytes"), h)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if calls != 1 || r.StatusCode != 307 {
		t.Fatal("redirect followed")
	}
}

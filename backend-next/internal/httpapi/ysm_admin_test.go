package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
)

func TestYSMCloudRouteAllowlist(t *testing.T) {
	for _, test := range []struct{ path, method, want string }{
		{"cloud/overview", "GET", "GET"}, {"cloud/connection", "POST", "POST"},
		{"cloud/models/a_1/versions", "GET", "GET"}, {"cloud/models/a_1/rollback", "POST", "POST"},
		{"cloud/npcs/login/00000000-0000-0000-0000-000000000001", "POST", "POST"},
	} {
		got, ok := ysmRouteMethod(test.path, test.method)
		if !ok || got != test.want {
			t.Fatalf("route %s: %s %v", test.path, got, ok)
		}
	}
	for _, path := range []string{"cloud/../config", "cloud/accounts", "cloud/models/x/delete", "cloud/models/a%2Fb/archive", "cloud/models/x/archive/extra", "cloud/npcs/login/not-a-uuid"} {
		if _, ok := ysmRouteMethod(path, "POST"); ok {
			t.Fatalf("unexpected route allowed: %s", path)
		}
	}
}

func TestYSMCloudUploadPreservesLengthMetadataAndPrivateBoundary(t *testing.T) {
	payload := bytes.Repeat([]byte{0, 255, 42}, 90000)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cloud/upload" || r.ContentLength != int64(len(payload)) {
			t.Errorf("upload path/size: %s %d", r.URL.Path, r.ContentLength)
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "dysm_csrf=upstream-csrf" || r.Header.Get("X-CSRF-Token") != "upstream-csrf" {
			t.Error("browser credentials escaped boundary")
		}
		if r.Header.Get("X-Asset-Base-Revision") != "7" || r.Header.Get("X-Deuterium-Model-Metadata") != "%7B%7D" || r.Header.Get("Idempotency-Key") != "same-key" {
			t.Error("Cloud metadata missing")
		}
		got, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(got, payload) {
			t.Error("upload body changed")
		}
		io.WriteString(w, `{"revision":8}`)
	}))
	defer upstream.Close()
	b := newYSMBridge(config.Config{YSMAdminURL: upstream.URL})
	headers := http.Header{"Authorization": {"Bearer browser-secret"}, "Cookie": {"session=browser-secret"}, "Content-Length": {"270000"}, "Content-Type": {"application/octet-stream"}, "X-Asset-Base-Revision": {"7"}, "X-Deuterium-Model-Metadata": {"%7B%7D"}, "Idempotency-Key": {"same-key"}}
	r, err := b.request(context.Background(), "cloud/upload", "POST", "upstream-csrf", bytes.NewReader(payload), headers)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatal(r.StatusCode)
	}
	headers.Del("Content-Length")
	if _, err := b.request(context.Background(), "cloud/upload", "POST", "upstream-csrf", bytes.NewReader(payload), headers); err == nil {
		t.Fatal("unknown length accepted")
	}
}

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

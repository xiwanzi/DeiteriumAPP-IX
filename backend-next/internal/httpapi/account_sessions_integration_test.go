//go:build integration

package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestAccountSessionLifetimesDevicesAndIsolation(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	u := f.users["Alice"]
	token, appSession, err := f.app.Identity.IssueSession(store.WithSessionDevice(ctx, "Android · 测试手机"), u, "app")
	if err != nil || !appSession.ExpiresAt.IsZero() {
		t.Fatal("native login must not expire", err, appSession.ExpiresAt)
	}
	f.tokens["NewAlice"] = token
	gameToken := identity.Secret()
	gameHash := store.Digest([]byte(gameToken))
	if err = f.store.CreateSession(ctx, u, gameHash, "gamechat", "test", time.Now().Add(10*time.Minute), ""); err != nil {
		t.Fatal(err)
	}
	f.tokens["GameAlice"] = gameToken
	f.request(t, "GameAlice", "GET", "/api/v1/account/sessions", nil, 403)
	_, web, err := f.app.Identity.IssueSession(ctx, u, "web")
	if err != nil || time.Until(web.ExpiresAt) < 30*24*time.Hour-time.Minute {
		t.Fatal("web session is not 30 days", err)
	}
	var expiry any
	if err = f.store.DB.QueryRow("SELECT expires_at FROM identity_sessions WHERE token_hash=?", appSession.TokenHash).Scan(&expiry); err != nil || expiry != nil {
		t.Fatal("native session used a far-future fake expiry", err, expiry)
	}
	data := socialDataV2(f.request(t, "NewAlice", "GET", "/api/v1/account/sessions", nil, 200))
	items := data["items"].([]any)
	if len(items) != 3 {
		t.Fatal("unexpected devices", items)
	}
	current := items[0].(map[string]any)
	if current["current"] != true || current["expiresAt"] != nil || current["deviceName"] != "Android · 测试手机" {
		t.Fatal("current device not identified", current)
	}
	for _, item := range items {
		if strings.Contains(storeJSONForTest(item), appSession.TokenHash) {
			t.Fatal("session digest exposed")
		}
	}
	target := items[1].(map[string]any)["sessionId"].(string)
	f.request(t, "Bob", "DELETE", "/api/v1/account/sessions/"+target, nil, 404)
	f.request(t, "NewAlice", "DELETE", "/api/v1/account/sessions/"+target, nil, 200)
	f.request(t, "NewAlice", "DELETE", "/api/v1/account/sessions/"+target, nil, 200)
	f.request(t, "NewAlice", "POST", "/api/v1/account/sessions/revoke-others", map[string]any{}, 200)
	if _, err = f.store.Session(ctx, gameHash); err != nil {
		t.Fatal("device exit disrupted game-only token", err)
	}
	if _, err = f.store.Session(ctx, web.TokenHash); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("revoked web session accepted", err)
	}
	f.request(t, "Alice", "GET", "/api/v1/account/me", nil, 401)
	f.request(t, "Bob", "GET", "/api/v1/account/me", nil, 200)
	f.request(t, "NewAlice", "GET", "/api/v1/account/me", nil, 200)
	f.request(t, "NewAlice", "DELETE", "/api/v1/account/sessions/"+current["sessionId"].(string), nil, 200)
	f.request(t, "NewAlice", "GET", "/api/v1/account/me", nil, 401)
}

func storeJSONForTest(v any) string { return fmt.Sprint(v) }

func TestAccountSessionWebCookieCsrfAndUnverifiedDevices(t *testing.T) {
	f := mailSupportFixture(t)
	u := emailSignup(t, f, "NewPlayer", "new@example.com")
	token, web, err := f.app.Identity.IssueSession(context.Background(), u, "web")
	if err != nil {
		t.Fatal(err)
	}
	f.request(t, "NewPlayer", "GET", "/api/v1/account/sessions", nil, 200)
	headers := map[string]string{"Cookie": f.app.cookieName() + "=" + token, "Origin": f.http.URL}
	admissionRaw(t, f, "/api/v1/account/sessions/revoke-others", map[string]any{}, headers, http.StatusForbidden)
	headers["X-CSRF-Token"] = web.CSRF
	admissionRaw(t, f, "/api/v1/account/sessions/revoke-others", map[string]any{}, headers, http.StatusOK)
	f.request(t, "NewPlayer", "GET", "/api/v1/account/me", nil, 401)
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Opt-in, disposable loopback fixture for the real admin page. Never uses a
// production database or issues a player-facing notification.
func TestLauncherIconBrowserFixture(t *testing.T) {
	output := os.Getenv("DEUTERIUM_ICON_QA_OUTPUT")
	if output == "" {
		t.Skip("opt-in icon browser fixture")
	}
	origin := os.Getenv("DEUTERIUM_ICON_QA_ORIGIN")
	if origin != "http://127.0.0.1:5188" {
		t.Fatal("explicit loopback origin required")
	}
	f := newSocialFixtureV2(t)
	f.app.Config.PublicOrigin = origin
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, value any) {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(output, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"Alice", "Bob"} {
		token := identity.Secret()
		expires := time.Now().Add(time.Hour)
		if err := f.store.CreateSession(ctx, f.users[name], store.Digest([]byte(token)), "web", identity.Secret(), expires, ""); err != nil {
			t.Fatal(err)
		}
		write(name+"-state.json", map[string]any{"cookies": []any{map[string]any{"name": "deuterium_dev_session", "value": token, "domain": "127.0.0.1", "path": "/", "expires": expires.Unix(), "httpOnly": true, "secure": false, "sameSite": "Lax"}}, "origins": []any{}})
	}
	write("connection.json", map[string]any{"backend": f.http.URL, "origin": origin})
	t.Log("isolated launcher icon fixture ready")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(50 * time.Minute)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			t.Fatal("fixture timeout; stop file not received")
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(output, "stop")); err == nil {
				return
			}
		}
	}
}

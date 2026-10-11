//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Opt-in local UI fixture. The gateway must point to an isolated Cloud instance.
func TestYSMCloudBrowserFixture(t *testing.T) {
	output := os.Getenv("DEUTERIUM_YSM_BROWSER_FIXTURE")
	if output == "" {
		t.Skip("interactive fixture not requested")
	}
	gateway := os.Getenv("DEUTERIUM_YSM_BROWSER_GATEWAY")
	settings := config.Config{YSMAdminURL: gateway}
	if gateway == "" || settings.ValidateYSM() != nil {
		t.Fatal("explicit loopback gateway required")
	}
	f := newSocialFixtureV2(t)
	f.http.Close()
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	token := identity.Secret()
	if err := f.store.CreateSession(ctx, f.users["Alice"], store.Digest([]byte(token)), "web", identity.Secret(), time.Now().Add(30*time.Minute), ""); err != nil {
		t.Fatal(err)
	}
	f.app.ysm = newYSMBridge(settings)
	web, _ := filepath.Abs("../../../web-app/dist")
	api := f.app.Handler()
	static := http.FileServer(http.Dir(web))
	f.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/fixture-login":
			f.app.setCookie(w, token, time.Now().Add(30*time.Minute))
			http.Redirect(w, r, "/admin?section=ysm", 303)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			api.ServeHTTP(w, r)
		case r.URL.Path == "/web-config.json":
			json.NewEncoder(w).Encode(map[string]any{"apiBase": "/api/v1", "version": "ysm-local-qa"})
		case strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/media/"):
			static.ServeHTTP(w, r)
		default:
			http.ServeFile(w, r, filepath.Join(web, "index.html"))
		}
	}))
	f.app.Config.PublicOrigin = f.http.URL
	data, _ := json.Marshal(map[string]any{"url": f.http.URL, "adminURL": f.http.URL + "/fixture-login"})
	if err := os.WriteFile(output, data, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(output + ".stop"); err == nil {
			os.Remove(output + ".stop")
			return
		}
		time.Sleep(time.Second)
	}
}

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

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Explicit, loopback-only UI fixture. No test login routes enter production builds.
func TestAdmissionBrowserFixture(t *testing.T) {
	file := os.Getenv("DEUTERIUM_ADMISSION_BROWSER_FIXTURE")
	if file == "" {
		t.Skip("interactive fixture not requested")
	}
	f := admissionFixture(t)
	f.http.Close()
	web, _ := filepath.Abs("../../../web-app/dist")
	portal, _ := filepath.Abs("../../../admission-web")
	adminToken, csrf := identity.Secret(), identity.Secret()
	if err := f.store.CreateSession(context.Background(), f.users["Alice"], store.Digest([]byte(adminToken)), "web", csrf, time.Now().Add(30*time.Minute), ""); err != nil {
		t.Fatal(err)
	}
	api := f.app.Handler()
	static := http.FileServer(http.Dir(web))
	admission := http.StripPrefix("/admission/", http.FileServer(http.Dir(portal)))
	f.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/fixture-login":
			f.app.setCookie(w, adminToken, time.Now().Add(30*time.Minute))
			http.Redirect(w, r, "/admin?section=whitelist", http.StatusSeeOther)
		case strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/bridge/"):
			api.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/admission/"):
			admission.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/web-config.json" || strings.HasPrefix(r.URL.Path, "/media/"):
			static.ServeHTTP(w, r)
		default:
			http.ServeFile(w, r, filepath.Join(web, "index.html"))
		}
	}))
	f.app.Config.PublicOrigin = f.http.URL
	f.app.Config.AdmissionPublicOrigin = f.http.URL
	data, _ := json.Marshal(map[string]any{"url": f.http.URL, "adminURL": f.http.URL + "/fixture-login", "applicationURL": f.http.URL + "/admission/"})
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	stop := file + ".stop"
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(stop); err == nil {
			os.Remove(stop)
			return
		}
		time.Sleep(time.Second)
	}
}

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

// Opt-in, disposable loopback fixture. No fixture routes or accounts are built
// into the application binary, and no mail/game connections leave this process.
func TestAccountBrowserFixture(t *testing.T) {
	file := os.Getenv("DEUTERIUM_ACCOUNT_BROWSER_FIXTURE")
	if file == "" {
		t.Skip("interactive fixture not requested")
	}
	f, u, _ := profileConflictFixture(t)
	deletionSQL(t, f, "INSERT INTO identity_aliases(alias_key,user_id) VALUES('alice',?)", f.users["Alice"].ID)
	admin := f.users["Alice"]
	admin.PasswordHash = identity.Hash(deletionAdminPassword)
	deletionSQL(t, f, "UPDATE identities SET password_hash=? WHERE id=?", admin.PasswordHash, admin.ID)
	nativeToken, _, err := f.app.Identity.IssueSession(store.WithSessionDevice(context.Background(), "Android · 测试手机"), admin, "app")
	if err != nil {
		t.Fatal(err)
	}
	_ = nativeToken
	adminToken, adminSession, err := f.app.Identity.IssueSession(store.WithSessionDevice(context.Background(), "Windows · Chrome"), admin, "web")
	if err != nil {
		t.Fatal(err)
	}
	userToken, userSession, err := f.app.Identity.IssueSession(store.WithSessionDevice(context.Background(), "Windows · Edge"), u, "web")
	if err != nil {
		t.Fatal(err)
	}
	f.app.AdmissionResolver = func(_ context.Context, name string) (store.AdmissionProfile, error) {
		if strings.EqualFold(name, "Occupied") {
			return store.AdmissionProfile{UUID: admissionTestUUID, Name: "Occupied"}, nil
		}
		if strings.EqualFold(name, "Alice") {
			return store.AdmissionProfile{UUID: admin.ServerUUID, Name: "Alice"}, nil
		}
		return store.AdmissionProfile{UUID: "33333333-3333-4333-8333-333333333333", Name: name}, nil
	}
	f.http.Close()
	web, _ := filepath.Abs("../../../web-app/dist")
	static := http.FileServer(http.Dir(web))
	api := f.app.Handler()
	f.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/fixture-admin":
			f.app.setCookie(w, adminToken, adminSession.ExpiresAt)
			http.Redirect(w, r, "/me", http.StatusSeeOther)
		case r.URL.Path == "/fixture-unverified":
			f.app.setCookie(w, userToken, userSession.ExpiresAt)
			http.Redirect(w, r, "/verify", http.StatusSeeOther)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			api.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/web-config.json" || strings.HasPrefix(r.URL.Path, "/media/"):
			static.ServeHTTP(w, r)
		default:
			http.ServeFile(w, r, filepath.Join(web, "index.html"))
		}
	}))
	f.app.Config.PublicOrigin = f.http.URL
	data, _ := json.Marshal(map[string]any{"url": f.http.URL, "adminURL": f.http.URL + "/fixture-admin", "unverifiedURL": f.http.URL + "/fixture-unverified", "accountId": u.ID, "adminPassword": deletionAdminPassword})
	if err = os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(40 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err = os.Stat(file + ".stop"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}

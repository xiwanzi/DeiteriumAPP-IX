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

// Optional, loopback-only real-game fixture. No test endpoint or credentials enter production.
func TestYSMGameIdentityFixture(t *testing.T) {
	output := os.Getenv("DEUTERIUM_YSM_GAME_FIXTURE")
	if output == "" {
		t.Skip("set DEUTERIUM_YSM_GAME_FIXTURE for isolated game acceptance")
	}
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	profiles := map[string]string{"Alice": "84ddf7da-d05a-3328-9274-dd39efd46873", "Bob": "ac3afbae-0d9d-32b0-828a-26b5b71a499f"}
	// UUIDs are synthetic fixture identities, not production Mojang claims.
	for name, profile := range profiles {
		if _, err := f.store.DB.Exec("UPDATE identities SET server_uuid=?,game_verified=TRUE WHERE id=?", profile, f.users[name].ID); err != nil {
			t.Fatal(err)
		}
	}
	secret := identity.Secret()
	f.app.Config.YSMIdentityTokenSHA256 = store.Digest([]byte(secret))
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	var schema string
	if err := f.store.DB.QueryRow("SELECT DATABASE()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"origin": f.http.URL, "serviceToken": secret, "schema": schema, "profiles": profiles, "adminUserId": f.users["Alice"].ID})
	if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, payload, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("Isolated website identity fixture ready")
	deadline := time.Now().Add(90 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(output + ".stop"); err == nil {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatal("fixture deadline reached")
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strings"
	"testing"
)

func TestYSMExistingVerifiedIdentityRolesAndRevocation(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	secret := strings.Repeat("service-", 6)
	f.app.Config.YSMIdentityTokenSHA256 = store.Digest([]byte(secret))
	lookup := func(uuid, token string, want int) map[string]any {
		t.Helper()
		req, _ := http.NewRequest("GET", f.http.URL+"/internal/v1/ysm/identities/"+uuid, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := f.http.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != want {
			t.Fatalf("identity status %d want %d", response.StatusCode, want)
		}
		var body map[string]any
		if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	alice, bob, carol := f.users["Alice"], f.users["Bob"], f.users["Carol"]
	lookup(alice.ServerUUID, f.tokens["Alice"], 401)
	if err := f.store.Grant(ctx, alice.ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Grant(ctx, carol.ID, "support.manage"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		user  store.User
		admin bool
	}{{alice, true}, {bob, false}, {carol, false}} {
		data := socialDataV2(lookup(test.user.ServerUUID, secret, 200))
		if data["userId"] != test.user.ID || data["modelAccess"] != true || data["administrator"] != test.admin {
			t.Fatal("wrong website identity projection", data)
		}
		for _, field := range []string{"email", "qq", "passwordHash", "token"} {
			if _, ok := data[field]; ok {
				t.Fatal("private account field exposed")
			}
		}
	}
	if _, err := f.store.DB.Exec("DELETE FROM identity_permissions WHERE user_id=? AND permission='platform.admin'", alice.ID); err != nil {
		t.Fatal(err)
	}
	if socialDataV2(lookup(alice.ServerUUID, secret, 200))["administrator"] != false {
		t.Fatal("stale admin role")
	}
	if _, err := f.store.DB.Exec("UPDATE identities SET game_verified=FALSE WHERE id=?", bob.ID); err != nil {
		t.Fatal(err)
	}
	lookup(bob.ServerUUID, secret, 403)
	if _, err := f.store.DB.Exec("UPDATE identities SET status='banned' WHERE id=?", carol.ID); err != nil {
		t.Fatal(err)
	}
	lookup(carol.ServerUUID, secret, 403)
	lookup("00000000-0000-0000-0000-000000000099", secret, 403)
	var count int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM identities").Scan(&count); err != nil || count != 3 {
		t.Fatal("lookup created or deleted accounts", count, err)
	}
}

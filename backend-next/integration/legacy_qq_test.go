//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestInvalidLegacyQQRetainedWithoutLoginAlias(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	u := legacy()
	u.QQ = "legacy.qq!"
	result, err := identity.Import(ctx, s, []identity.LegacyUser{u}, false)
	if err != nil || result.SkippedQQAliases != 1 || count(t, s, "identities") != 0 {
		t.Fatal("preflight did not identify skipped alias", result, err)
	}
	result, err = identity.Import(ctx, s, []identity.LegacyUser{u}, true)
	if err != nil || result.SkippedQQAliases != 1 {
		t.Fatal(err)
	}
	current, err := s.UserByAlias(ctx, "alice")
	if err != nil || current.QQ != u.QQ || current.ID != u.ID || current.ServerUUID != u.ServerUUID {
		t.Fatal("legacy identity/data lost", err)
	}
	var aliases int
	if err = s.DB.QueryRow("SELECT COUNT(*) FROM identity_aliases WHERE user_id=?", u.ID).Scan(&aliases); err != nil || aliases != 1 {
		t.Fatal("invalid QQ alias created")
	}
	service := identity.New(s)
	if _, _, err = service.Login(ctx, "Alice", password, "192.0.2.20", "app"); err != nil {
		t.Fatal("game ID login failed", err)
	}
	if _, _, err = service.Login(ctx, "legacy.qq!", password, "192.0.2.20", "app"); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("invalid QQ accepted as login")
	}
	result, err = identity.Import(ctx, s, []identity.LegacyUser{u}, true)
	if err != nil || result.Unchanged != 1 || result.SkippedQQAliases != 1 {
		t.Fatal("repeat import changed legacy row", err)
	}
}

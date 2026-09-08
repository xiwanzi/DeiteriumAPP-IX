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

func TestGameNameAndQQShareIdentityFailureBudget(t *testing.T) {
	s := testdb.New(t)
	imported(t, s)
	service := identity.New(s)
	ctx := context.Background()
	for i := range 8 {
		alias := "Alice"
		if i%2 == 1 {
			alias = "10001"
		}
		if _, _, err := service.Login(ctx, alias, "incorrect-password", "192.0.2.9", "app"); !errors.Is(err, store.ErrUnauthorized) {
			t.Fatal("unexpected failed login result", err)
		}
	}
	if _, _, err := service.Login(ctx, "Alice", password, "192.0.2.9", "app"); !errors.Is(err, store.ErrRateLimited) {
		t.Fatal("changing login alias bypassed account budget", err)
	}
}

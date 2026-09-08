//go:build integration

package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestConcurrentFirstLoginsAndChangedPasswordProtection(t *testing.T) {
	s := testdb.New(t)
	imported(t, s)
	service := identity.New(s)
	ctx := context.Background()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, kind := range []string{"app", "web"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := service.Login(ctx, "Alice", password, "192.0.2.17", kind)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal("simultaneous valid first login failed", err)
		}
	}
	if count(t, s, "identity_sessions") != 2 {
		t.Fatal("first logins did not create both sessions")
	}
	verified, err := s.UserByAlias(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE identities SET password_hash=? WHERE id=?", identity.Hash("new-password-456"), verified.ID); err != nil {
		t.Fatal(err)
	}
	err = s.CreateSession(ctx, verified, store.Digest([]byte("stale-verification-token")), "app", "csrf", time.Now().Add(time.Hour), "")
	if !errors.Is(err, store.ErrCredentialChanged) {
		t.Fatal("stale password verification created a session", err)
	}
	if _, _, err = service.Login(ctx, "Alice", password, "192.0.2.17", "app"); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("old password accepted after password change", err)
	}
}

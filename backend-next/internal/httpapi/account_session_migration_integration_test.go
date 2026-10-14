//go:build integration

package httpapi

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestAccountSessionMigrationExtendsOnlyValidSessions(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	u := f.users["Alice"]
	for _, v := range []struct {
		key, kind string
		expiry    time.Time
	}{{"valid-native", "app", time.Now().Add(7 * 24 * time.Hour)}, {"expired-native", "app", time.Now().Add(-time.Hour)}, {"revoked-native", "app", time.Now().Add(time.Hour)}, {"valid-web", "web", time.Now().Add(7 * 24 * time.Hour)}, {"game-only", "gamechat", time.Now().Add(10 * time.Minute)}} {
		if err := f.store.CreateSession(ctx, u, store.Digest([]byte(v.key)), v.kind, "test", v.expiry, ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.store.Revoke(ctx, store.Digest([]byte("revoked-native"))); err != nil {
		t.Fatal(err)
	}
	beforeGame, err := f.store.Session(ctx, store.Digest([]byte("game-only")))
	if err != nil {
		t.Fatal(err)
	}
	sql, err := os.ReadFile("../store/migrations/043_account_sessions.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(sql), ";") {
		if strings.TrimSpace(statement) != "" {
			if _, err = f.store.DB.Exec(statement); err != nil {
				t.Fatal(err)
			}
		}
	}
	native, err := f.store.Session(ctx, store.Digest([]byte("valid-native")))
	if err != nil || !native.ExpiresAt.IsZero() {
		t.Fatal("valid native session not extended", err)
	}
	web, err := f.store.Session(ctx, store.Digest([]byte("valid-web")))
	if err != nil || time.Until(web.ExpiresAt) < 30*24*time.Hour-time.Minute {
		t.Fatal("web migration did not extend to 30 days", err)
	}
	for _, key := range []string{"expired-native", "revoked-native"} {
		if _, err = f.store.Session(ctx, store.Digest([]byte(key))); !errors.Is(err, store.ErrUnauthorized) {
			t.Fatal("invalid session revived", key, err)
		}
	}
	game, err := f.store.Session(ctx, store.Digest([]byte("game-only")))
	if err != nil || !game.ExpiresAt.Equal(beforeGame.ExpiresAt) {
		t.Fatal("migration changed game token", err)
	}
}

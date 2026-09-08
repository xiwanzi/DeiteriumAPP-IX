//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

const legacyHash = "$argon2i$v=19$m=65536,t=3,p=2$o+jTlhl/fSdGLpHQ81nOng$kXV9sfxMH/x7dtnoJCG0EPRudDCmCUlzVFElYGthqzY"
const password = "migration-password-123"
const aliceUUID = "d97161f9-2a7c-4abd-a8e7-6fd64a64c001"

func legacy() identity.LegacyUser {
	date := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	return identity.LegacyUser{ID: "legacy_alice", ServerUUID: aliceUUID, GameID: "Alice", QQ: "10001", PasswordHash: legacyHash, Status: "active", CreatedAt: date, UpdatedAt: date}
}
func imported(t *testing.T, s *store.Store) identity.LegacyUser {
	t.Helper()
	u := legacy()
	if _, err := identity.Import(context.Background(), s, []identity.LegacyUser{u}, true); err != nil {
		t.Fatal(err)
	}
	return u
}
func count(t *testing.T, s *store.Store, table string) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestImportIsReadOnlyUntilApplyAndAtomicOnConflict(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	u := legacy()
	result, err := identity.Import(ctx, s, []identity.LegacyUser{u}, false)
	if err != nil || result.Create != 1 || result.Applied || count(t, s, "identities") != 0 {
		t.Fatalf("dry-run changed data or failed: %+v %v", result, err)
	}
	result, err = identity.Import(ctx, s, []identity.LegacyUser{u}, true)
	if err != nil || !result.Applied || count(t, s, "identities") != 1 {
		t.Fatal("apply failed", err)
	}
	result, err = identity.Import(ctx, s, []identity.LegacyUser{u}, true)
	if err != nil || result.Unchanged != 1 || count(t, s, "identities") != 1 {
		t.Fatal("replay duplicated identity", err)
	}
	bob := u
	bob.ID = "legacy_bob"
	bob.ServerUUID = "d97161f9-2a7c-4abd-a8e7-6fd64a64c002"
	bob.GameID = "Bob"
	bob.QQ = "10002"
	conflict := u
	conflict.ID = "conflicting_account"
	conflict.GameID = "Carol"
	conflict.QQ = "10003"
	if _, err = identity.Import(ctx, s, []identity.LegacyUser{bob, conflict}, true); err == nil {
		t.Fatal("existing UUID conflict accepted")
	}
	if count(t, s, "identities") != 1 {
		t.Fatal("failed import left partial identities")
	}
	bob.GameID = "10001"
	if _, err = identity.Import(ctx, s, []identity.LegacyUser{bob}, false); err == nil {
		t.Fatal("cross game-name / QQ alias collision accepted")
	}
	bob.GameID = "Bob"
	bob.PasswordHash = strings.Replace(legacyHash, "m=65536", "m=2000000", 1)
	if _, err = identity.Import(ctx, s, []identity.LegacyUser{bob}, true); err == nil {
		t.Fatal("unbounded legacy KDF accepted")
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("migration not repeatable", err)
	}
	if _, err = s.DB.Exec("UPDATE schema_migrations_next SET checksum=?", strings.Repeat("0", 64)); err != nil {
		t.Fatal(err)
	}
	if s.Migrate(ctx) == nil {
		t.Fatal("modified applied migration was silently accepted")
	}
}

func TestIdentitySessionsUpgradeRevokeAndStatus(t *testing.T) {
	s := testdb.New(t)
	u := imported(t, s)
	ctx := context.Background()
	service := identity.New(s)
	token, session, err := service.Login(ctx, "ALICE", password, "192.0.2.1", "app")
	if err != nil || len(token) != 43 || session.User.ID != u.ID {
		t.Fatal("legacy login failed", err)
	}
	stored, err := s.UserByAlias(ctx, "alice")
	if err != nil || identity.NeedsUpgrade(stored.PasswordHash) {
		t.Fatal("successful login did not upgrade legacy hash", err)
	}
	if _, err = s.Session(ctx, store.Digest([]byte(token))); err != nil {
		t.Fatal("session unavailable", err)
	}
	if _, err = identity.Import(ctx, s, []identity.LegacyUser{u}, true); err != nil {
		t.Fatal("import replay should not undo upgraded password", err)
	}
	if err = s.Revoke(ctx, session.TokenHash); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, session.TokenHash); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("revoked session accepted")
	}
	if _, _, err = service.Login(ctx, "10001", "incorrect-password", "192.0.2.1", "app"); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("wrong password accepted", err)
	}
	if _, err = s.DB.Exec("UPDATE identities SET status='disabled' WHERE id=?", u.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.Login(ctx, "Alice", password, "192.0.2.1", "app"); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("disabled account logged in", err)
	}
	if err = s.CreateSession(ctx, stored, store.Digest([]byte("stale-token")), "app", "csrf", time.Now().Add(time.Hour), ""); !errors.Is(err, store.ErrUnauthorized) {
		t.Fatal("session created after account disabled")
	}
}

func TestLoginBudgetIsConcurrentAndSurvivesNewService(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	var passed atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.ReserveLogin(ctx, store.Digest([]byte("account:alice")), store.Digest([]byte("ip:192.0.2.1")))
			if err == nil {
				passed.Add(1)
			} else if !errors.Is(err, store.ErrRateLimited) {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if passed.Load() != 8 {
		t.Fatalf("parallel attempts bypassed budget: %d", passed.Load())
	}
	imported(t, s)
	service := identity.New(s)
	if _, _, err := service.Login(ctx, "Alice", password, "192.0.2.1", "app"); !errors.Is(err, store.ErrRateLimited) {
		t.Fatal("new service instance reset durable failure budget", err)
	}
}

func TestConcurrentCoreReplayIsOneMessageAndRejectsDifferentPayload(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	chat := store.CoreChat{PlayerUUID: aliceUUID, GameID: "Alice", Content: "hello"}
	payload, _ := json.Marshal(chat)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := s.CoreEvent(ctx, "amiya", "stable-event", "chat.public.event", payload, &chat, nil)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if count(t, s, "core_events") != 1 || count(t, s, "chat_messages_next") != 1 {
		t.Fatal("duplicate event created multiple messages")
	}
	chat.Content = "tampered"
	payload, _ = json.Marshal(chat)
	if _, _, err := s.CoreEvent(ctx, "amiya", "stable-event", "chat.public.event", payload, &chat, nil); !errors.Is(err, store.ErrConflict) {
		t.Fatal("same event identity changed contents", err)
	}
	if count(t, s, "chat_messages_next") != 1 {
		t.Fatal("conflicting event changed chat")
	}
}

func TestItemVersionsAreImmutableAndTransactionRollsBack(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	item := store.ItemVersion{ItemRef: "deuterium:battery", Revision: 1, PayloadSHA256: strings.Repeat("a", 64), DisplayName: "Battery", MaxQuantity: 64, CompatibleServerIDs: []string{"amiya"}}
	payload, _ := json.Marshal(item)
	if _, _, err := s.CoreEvent(ctx, "amiya", "item-1", "item.version.published", payload, nil, &item); err != nil {
		t.Fatal(err)
	}
	item.MaxQuantity = 32
	payload, _ = json.Marshal(item)
	if _, _, err := s.CoreEvent(ctx, "amiya", "item-2", "item.version.published", payload, nil, &item); !errors.Is(err, store.ErrConflict) {
		t.Fatal("immutable version overwritten", err)
	}
	if count(t, s, "core_events") != 1 || count(t, s, "item_versions") != 1 {
		t.Fatal("failed business write committed event marker")
	}
}

func TestLegacyReadOnlyExportRoundTrip(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	u := legacy()
	_, err := s.DB.Exec(`CREATE TABLE app_users (id VARCHAR(40) PRIMARY KEY,server_uuid VARCHAR(80),current_game_id VARCHAR(32),qq VARCHAR(20),password_hash VARCHAR(255),status VARCHAR(20),created_at DATETIME,updated_at DATETIME)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.DB.Exec("INSERT INTO app_users VALUES (?,?,?,?,?,?,?,?)", u.ID, u.ServerUUID, u.GameID, u.QQ, u.PasswordHash, u.Status, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	n, err := identity.ExportLegacy(ctx, s, &out)
	if err != nil || n != 1 {
		t.Fatal("export failed", err)
	}
	users, err := identity.ReadLegacy(&out)
	if err != nil || len(users) != 1 || users[0].PasswordHash != legacyHash {
		t.Fatal("legacy export is not importable", err)
	}
	if result, err := identity.Import(ctx, s, users, true); err != nil || result.Create != 1 {
		t.Fatal("round-trip import failed", err)
	}
	if count(t, s, "app_users") != 1 {
		t.Fatal("legacy source changed")
	}
}

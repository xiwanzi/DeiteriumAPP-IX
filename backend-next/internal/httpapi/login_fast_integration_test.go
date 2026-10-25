//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func fastLoginUser(t *testing.T, f *socialFixtureV2, name, password string) store.User {
	t.Helper()
	u := f.users[name]
	deletionSQL(t, f, "UPDATE identities SET password_hash=? WHERE id=?", identity.Hash(password), u.ID)
	deletionSQL(t, f, "INSERT INTO identity_aliases VALUES(LOWER(?),?)", name, u.ID)
	u, err := f.store.UserByAlias(context.Background(), strings.ToLower(name))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestLoginFastReturnsWhileOfficialRefreshBlockedAndPreservesSession(t *testing.T) {
	f := newSocialFixtureV2(t)
	u := fastLoginUser(t, f, "Alice", "correct-password")
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	var once sync.Once
	defer once.Do(func() { close(release) })
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		t.Error("known correct password synchronously queried name")
		return store.AdmissionProfile{}, identity.ErrNameLookupUnavailable
	}
	f.app.OfficialUUIDResolver = func(ctx context.Context, uuid string) (store.AdmissionProfile, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
		case <-ctx.Done():
			return store.AdmissionProfile{}, ctx.Err()
		}
		return store.AdmissionProfile{UUID: uuid, Name: "NewAlice"}, nil
	}
	begin := time.Now()
	token, session, err := f.app.Identity.Login(context.Background(), "Alice", "correct-password", "127.0.0.8", "web")
	if err != nil || session.User.ID != u.ID {
		t.Fatal("fast path failed", err)
	}
	t.Logf("known game ID login returned in %s while UUID upstream remained blocked", time.Since(begin))
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("background refresh did not run")
	}
	for range 3 {
		if _, _, e := f.app.Identity.Login(context.Background(), "Alice", "correct-password", "127.0.0.8", "app"); e != nil {
			t.Fatal(e)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate refreshes", calls.Load())
	}
	once.Do(func() { close(release) })
	deadline := time.Now().Add(3 * time.Second)
	for {
		current, e := f.store.UserByUUID(context.Background(), u.ServerUUID)
		if e == nil && current.GameID == "NewAlice" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background rename not applied", e)
		}
		time.Sleep(10 * time.Millisecond)
	}
	saved, e := f.store.Session(context.Background(), store.Digest([]byte(token)))
	if e != nil || saved.User.ID != u.ID || saved.User.ServerUUID != u.ServerUUID || saved.User.GameID != "NewAlice" {
		t.Fatal("session identity changed", e)
	}
}

func TestLoginFastWrongPasswordReusesKDFAndDoesNotBypassLock(t *testing.T) {
	f := newSocialFixtureV2(t)
	u := fastLoginUser(t, f, "Alice", "correct-password")
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: u.ServerUUID, Name: "Alice"}, nil
	}
	for range 8 {
		_, _, e := f.app.Identity.Login(context.Background(), "Alice", "wrong-password", "127.0.0.8", "app")
		if !errors.Is(e, store.ErrUnauthorized) {
			t.Fatal(e)
		}
	}
	stats := f.app.Identity.CapacitySnapshot()["passwordCalculation"].(map[string]uint64)
	if stats["count"] != 8 {
		t.Fatal("fallback repeated password KDF", stats)
	}
	if _, _, e := f.app.Identity.Login(context.Background(), "Alice", "correct-password", "127.0.0.8", "app"); !errors.Is(e, store.ErrRateLimited) {
		t.Fatal("lock bypassed", e)
	}
}

func TestLoginFastUpstreamOutageCannotEraseActualPasswordFailures(t *testing.T) {
	f := newSocialFixtureV2(t)
	fastLoginUser(t, f, "Alice", "correct-password")
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{}, identity.ErrNameLookupUnavailable
	}
	for range 8 {
		if _, _, e := f.app.Identity.Login(context.Background(), "Alice", "wrong-password", "127.0.0.8", "app"); !errors.Is(e, identity.ErrNameLookupUnavailable) {
			t.Fatal(e)
		}
	}
	if _, _, e := f.app.Identity.Login(context.Background(), "Alice", "correct-password", "127.0.0.8", "app"); !errors.Is(e, store.ErrRateLimited) {
		t.Fatal("upstream outage bypassed real credential failures", e)
	}
}

func TestLoginFastReusedNameCanResolvePastLockedPreviousOwner(t *testing.T) {
	f := newSocialFixtureV2(t)
	a := fastLoginUser(t, f, "Alice", "alice-password")
	b := fastLoginUser(t, f, "Bob", "bob-password")
	for range 8 {
		if _, e := f.store.ReserveIdentityLogin(context.Background(), store.Digest([]byte("identity:"+b.ID))); e != nil {
			t.Fatal(e)
		}
	}
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: a.ServerUUID, Name: "Bob"}, nil
	}
	f.app.OfficialUUIDResolver = func(_ context.Context, uuid string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: uuid, Name: "RenamedBob"}, nil
	}
	_, session, e := f.app.Identity.Login(context.Background(), "Bob", "alice-password", "127.0.0.8", "app")
	if e != nil || session.User.ID != a.ID {
		t.Fatal("previous owner's lock blocked current owner", e)
	}
	if got := deletionCount(t, f, "SELECT attempts FROM auth_budgets WHERE budget_key=?", store.Digest([]byte("identity:"+b.ID))); got != 8 {
		t.Fatal("previous lock erased", got)
	}
}

func TestLoginFastAliasRecheckedInsideSessionTransaction(t *testing.T) {
	f := newSocialFixtureV2(t)
	u := fastLoginUser(t, f, "Alice", "correct-password")
	ctx := context.Background()
	_, e := f.store.SyncVerifiedNames(ctx, []store.AdmissionProfile{{UUID: u.ServerUUID, Name: "NewAlice"}})
	if e != nil {
		t.Fatal(e)
	}
	// Simulate completion of KDF after a concurrent rename removed its alias.
	_, _, e = f.app.Identity.IssueSession(store.WithLoginAlias(ctx, "alice"), u, "app")
	if !errors.Is(e, store.ErrNameProofStale) {
		t.Fatal("stale alias issued a session", e)
	}
}

func TestLoginFastUnverifiedRefreshNeverGrantsGameOwnership(t *testing.T) {
	f := newSocialFixtureV2(t)
	u := fastLoginUser(t, f, "Alice", "test-password")
	deletionSQL(t, f, "UPDATE identities SET game_verified=FALSE,profile_uuid=server_uuid,server_uuid=NULL,email='fast@example.com' WHERE id=?", u.ID)
	deletionSQL(t, f, "INSERT INTO identity_aliases VALUES('fast@example.com',?)", u.ID)
	f.app.OfficialUUIDResolver = func(_ context.Context, uuid string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: uuid, Name: "NewPlayer"}, nil
	}
	token, _, e := f.app.Identity.Login(context.Background(), "Alice", "test-password", "127.0.0.8", "app")
	if e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		current, e := f.store.UserByAlias(context.Background(), "fast@example.com")
		if e == nil && current.GameID == "NewPlayer" {
			if current.ID != u.ID || current.GameVerified || current.ServerUUID != "" {
				t.Fatal("unverified identity changed")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("unverified rename failed", e, "worker failures", f.app.loginProfiles.failed.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.tokens["Unverified"] = token
	f.request(t, "Unverified", "GET", "/api/v1/wallet/balance", nil, 403)
}

func TestLoginFastCloseCancelsBackgroundRequest(t *testing.T) {
	f := newSocialFixtureV2(t)
	fastLoginUser(t, f, "Alice", "correct-password")
	started := make(chan struct{})
	f.app.OfficialUUIDResolver = func(ctx context.Context, _ string) (store.AdmissionProfile, error) {
		close(started)
		<-ctx.Done()
		return store.AdmissionProfile{}, ctx.Err()
	}
	if _, _, e := f.app.Identity.Login(context.Background(), "Alice", "correct-password", "127.0.0.8", "app"); e != nil {
		t.Fatal(e)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker not started")
	}
	done := make(chan struct{})
	go func() { f.app.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not stop refresh")
	}
}

func TestLoginFastHTTPKeepsSlowLookupOnlyOnNewName(t *testing.T) {
	f := newSocialFixtureV2(t)
	u := fastLoginUser(t, f, "Alice", "correct-password")
	const upstreamDelay = 1500 * time.Millisecond
	f.app.AdmissionResolver = func(ctx context.Context, name string) (store.AdmissionProfile, error) {
		select {
		case <-time.After(upstreamDelay):
		case <-ctx.Done():
			return store.AdmissionProfile{}, ctx.Err()
		}
		return store.AdmissionProfile{UUID: u.ServerUUID, Name: name}, nil
	}
	login := func(path, name string) time.Duration {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"account": name, "password": "correct-password"})
		req, _ := http.NewRequest("POST", f.http.URL+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if path == "/api/v1/web/session" {
			req.Header.Set("Origin", f.http.URL)
		}
		start := time.Now()
		r, e := f.http.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer r.Body.Close()
		var result struct {
			Data struct {
				User store.User `json:"user"`
			} `json:"data"`
		}
		if e = json.NewDecoder(r.Body).Decode(&result); e != nil || r.StatusCode != 200 || result.Data.User.ID != u.ID {
			t.Fatal("HTTP login failed", r.StatusCode, e)
		}
		return time.Since(start)
	}
	web := login("/api/v1/web/session", "Alice")
	app := login("/api/v1/account/login", "Alice")
	slow := login("/api/v1/web/session", "NewAlice")
	if web >= upstreamDelay || app >= upstreamDelay || slow < upstreamDelay {
		t.Fatal("upstream latency leaked into known-name login", web, app, slow)
	}
	t.Logf("same HTTP server + real Argon2: known ID web=%s app=%s; new ID with 1.5s synthetic official service=%s", web, app, slow)
}

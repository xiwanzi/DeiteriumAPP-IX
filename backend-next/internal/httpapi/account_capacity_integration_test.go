//go:build integration

package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestAccountCapacityHundredLoginsLeaveAdmissionResponsive(t *testing.T) {
	f := newSocialFixtureV2(t)
	hash := identity.Hash("capacity-test-password")
	profiles := map[string]store.AdmissionProfile{}
	for n := 0; n < 100; n++ {
		name := fmt.Sprintf("Burst%03d", n)
		uuid := fmt.Sprintf("11111111-0000-4000-8000-%012d", n+1)
		id, ref := fmt.Sprintf("burst_%d", n), fmt.Sprintf("burst_player_%d", n)
		deletionSQL(t, f, `INSERT INTO identities(id,player_ref,game_id,qq,password_hash,status,created_at,updated_at,legacy_fingerprint,profile_uuid,game_verified,email) VALUES(?,?,?,'',?,'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),'test',?,FALSE,?)`, id, ref, name, hash, uuid, strings.ToLower(name)+"@example.com")
		deletionSQL(t, f, "INSERT INTO identity_aliases(alias_key,user_id) VALUES(LOWER(?),?)", name, id)
		profiles[strings.ToLower(name)] = store.AdmissionProfile{UUID: uuid, Name: name}
	}
	f.app.AdmissionResolver = func(ctx context.Context, name string) (store.AdmissionProfile, error) {
		select {
		case <-ctx.Done():
			return store.AdmissionProfile{}, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
		return profiles[strings.ToLower(name)], nil
	}
	f.app.Config.AdmissionGatewayTokenSHA256 = store.Digest([]byte(strings.Repeat("g", 32)))
	controlServerTimes := make(chan time.Duration, 50)
	f.http.Close()
	handler := f.app.Handler()
	f.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bridge/v1/admission/check" {
			began := time.Now()
			defer func() { controlServerTimes <- time.Since(began) }()
		}
		handler.ServeHTTP(w, r)
	}))
	f.app.Config.PublicOrigin = f.http.URL
	type result struct {
		status  int
		elapsed time.Duration
		code    string
		err     error
	}
	results := make(chan result, 100)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for n := 0; n < 100; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			began := time.Now()
			status, body, err := f.call("", "POST", "/api/v1/account/login", map[string]string{"account": fmt.Sprintf("Burst%03d", n), "password": "capacity-test-password"})
			code := ""
			if e, ok := body["error"].(map[string]any); ok {
				code, _ = e["code"].(string)
			}
			results <- result{status, time.Since(began), code, err}
		}(n)
	}
	close(start)
	controlTimes := []time.Duration{}
	for range 50 {
		began := time.Now()
		admissionRaw(t, f, "/bridge/v1/admission/check", map[string]string{"uuid": admissionTestProfileUUID}, map[string]string{"Authorization": "Bearer " + strings.Repeat("g", 32)}, 200)
		controlTimes = append(controlTimes, time.Since(began))
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()
	close(results)
	close(controlServerTimes)
	serverTimes := []time.Duration{}
	for elapsed := range controlServerTimes {
		serverTimes = append(serverTimes, elapsed)
	}
	sort.Slice(serverTimes, func(i, j int) bool { return serverTimes[i] < serverTimes[j] })
	var success, busy int
	loginTimes := []time.Duration{}
	for r := range results {
		if r.err != nil {
			t.Error(r.err)
			continue
		}
		loginTimes = append(loginTimes, r.elapsed)
		if r.status == 200 {
			success++
		} else if r.status == 503 && (r.code == "SERVER_BUSY" || r.code == "NAME_LOOKUP_BUSY") {
			busy++
		} else {
			t.Errorf("healthy shared-network login rejected as %d/%s", r.status, r.code)
		}
	}
	sort.Slice(loginTimes, func(i, j int) bool { return loginTimes[i] < loginTimes[j] })
	sort.Slice(controlTimes, func(i, j int) bool { return controlTimes[i] < controlTimes[j] })
	if success+busy != 100 || success == 0 {
		t.Fatal("burst did not finish with bounded responses", success, busy)
	}
	if controlTimes[len(controlTimes)-1] > 2*time.Second {
		t.Fatal("game admission was stalled", controlTimes[len(controlTimes)-1])
	}
	t.Logf("100 cold logins on one IP: success=%d busy=%d p50=%s p95=%s max=%s; admission p95=%s max=%s; capacity=%v", success, busy, loginTimes[len(loginTimes)/2], loginTimes[len(loginTimes)*95/100], loginTimes[len(loginTimes)-1], controlTimes[47], controlTimes[49], f.app.Identity.CapacitySnapshot())
	t.Logf("admission handler p95=%s max=%s; database waits=%d totalWait=%s", serverTimes[47], serverTimes[49], f.store.DB.Stats().WaitCount, f.store.DB.Stats().WaitDuration)
}

func TestAccountCapacityUpstreamFailureDoesNotConsumeCredentialBudget(t *testing.T) {
	f := mailSupportFixture(t)
	auditProfile(f)
	u := registeredProfileSignup(t, f, "OldPlayer", "lookup@example.com")
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{}, identity.ErrNameLookupBusy
	}
	for range 10 {
		_, _, err := f.app.Identity.Login(context.Background(), "UnknownNewName", "test-password", "127.0.0.10", "app")
		if !errors.Is(err, identity.ErrNameLookupBusy) {
			t.Fatal("busy lookup became a password lock", err)
		}
	}
	f.app.AdmissionResolver = func(context.Context, string) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: admissionTestUUID, Name: u.GameID}, nil
	}
	if _, _, err := f.app.Identity.Login(context.Background(), u.GameID, "test-password", "127.0.0.10", "app"); err != nil {
		t.Fatal("recovered lookup cannot log in", err)
	}
	for range 8 {
		if _, _, err := f.app.Identity.Login(context.Background(), u.GameID, "wrong-password", "127.0.0.10", "app"); !errors.Is(err, store.ErrUnauthorized) {
			t.Fatal("wrong password not counted", err)
		}
	}
	if _, _, err := f.app.Identity.Login(context.Background(), u.GameID, "test-password", "127.0.0.10", "app"); !errors.Is(err, store.ErrRateLimited) {
		t.Fatal("failure protection removed", err)
	}
}

func TestAccountCapacityLateReservationCannotEraseNewFailures(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	key := store.Digest([]byte("window-boundary"))
	lease, err := f.store.ReserveIdentityLogin(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	deletionSQL(t, f, "UPDATE auth_budgets SET attempts=3,expires_at=DATE_ADD(expires_at,INTERVAL 1 MINUTE) WHERE budget_key=?", key)
	if err = f.store.ReleaseLoginBudget(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if deletionCount(t, f, "SELECT attempts FROM auth_budgets WHERE budget_key=?", key) != 3 {
		t.Fatal("old reservation erased new failures")
	}
	current, err := f.store.ReserveIdentityLogin(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.ReleaseLoginBudget(ctx, current); err != nil {
		t.Fatal(err)
	}
	if deletionCount(t, f, "SELECT attempts FROM auth_budgets WHERE budget_key=?", key) != 3 {
		t.Fatal("successful attempt erased concurrent failures")
	}
}

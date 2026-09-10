//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestLauncherIconBrowserMutationRequiresOriginAndCSRF(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	token, csrf := identity.Secret(), identity.Secret()
	if err := f.store.CreateSession(ctx, f.users["Alice"], store.Digest([]byte(token)), "web", csrf, time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		origin, csrf string
		status       int
	}{
		{f.http.URL, "", 403}, {"https://other.example", csrf, 403}, {f.http.URL, csrf, 200},
	} {
		r, err := http.NewRequest("PUT", f.http.URL+"/api/v1/admin/launcher-icon", bytes.NewBufferString(`{"clientRequestId":"browser-publish","expectedVersion":1,"iconId":"anniversary_911"}`))
		if err != nil {
			t.Fatal(err)
		}
		r.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: token})
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", attempt.origin)
		r.Header.Set("X-CSRF-Token", attempt.csrf)
		response, err := f.http.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != attempt.status {
			t.Fatalf("CSRF result %d want %d", response.StatusCode, attempt.status)
		}
	}
}

func TestLauncherIconPermissionsReplayAndPersistentSelection(t *testing.T) {
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	public := "/api/v1/app/launcher-icon"
	admin := "/api/v1/admin/launcher-icon"
	initial := socialDataV2(f.request(t, "", "GET", public, nil, 200))
	if initial["iconId"] != "default" || initial["version"] != float64(1) || initial["minAppVersionCode"] != float64(20800) {
		t.Fatalf("unexpected initial selection %v", initial)
	}
	f.request(t, "", "GET", admin, nil, 401)
	f.request(t, "Bob", "GET", admin, nil, 403)
	f.request(t, "Bob", "PUT", admin, map[string]any{"clientRequestId": "denied", "expectedVersion": 1, "iconId": "anniversary_911"}, 403)
	if err := f.store.Grant(ctx, f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	list := socialDataV2(f.request(t, "Alice", "GET", admin, nil, 200))
	if len(list["icons"].([]any)) != 2 {
		t.Fatal("icon catalogue missing")
	}
	for _, icon := range []string{"", "uploaded.png", "../default", "DEFAULT"} {
		f.request(t, "Alice", "PUT", admin, map[string]any{"clientRequestId": "invalid", "expectedVersion": 1, "iconId": icon}, 400)
	}
	input := map[string]any{"clientRequestId": "select-anniversary", "expectedVersion": 1, "iconId": "anniversary_911"}
	first := socialDataV2(f.request(t, "Alice", "PUT", admin, input, 200))
	replay := socialDataV2(f.request(t, "Alice", "PUT", admin, input, 200))
	if first["version"] != float64(2) || replay["version"] != first["version"] || first["iconId"] != "anniversary_911" {
		t.Fatal("selection was not idempotent")
	}
	f.request(t, "Alice", "PUT", admin, map[string]any{"clientRequestId": "select-anniversary", "expectedVersion": 1, "iconId": "default"}, 409)
	f.request(t, "Alice", "PUT", admin, map[string]any{"clientRequestId": "stale", "expectedVersion": 1, "iconId": "default"}, 409)
	unchanged := socialDataV2(f.request(t, "Alice", "PUT", admin, map[string]any{"clientRequestId": "same-value", "expectedVersion": 2, "iconId": "anniversary_911"}, 200))
	if unchanged["version"] != float64(2) {
		t.Fatal("same value changed revision")
	}
	current := socialDataV2(f.request(t, "Alice", "PUT", admin, map[string]any{"clientRequestId": "restore-default", "expectedVersion": 2, "iconId": "default"}, 200))
	if current["version"] != float64(3) {
		t.Fatal("revision did not advance")
	}
	var count int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM audit_events_next WHERE action='app.launcher-icon'").Scan(&count); err != nil || count != 2 {
		t.Fatalf("audit duplicated: %d %v", count, err)
	}
	// A recreated handler reads the committed selection rather than process memory.
	f.app.Close()
	other := New(f.store, f.app.Config)
	defer other.Close()
	rr := httptest.NewRecorder()
	other.Handler().ServeHTTP(rr, httptest.NewRequest("GET", public, nil))
	var saved map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if socialDataV2(saved)["version"] != float64(3) || socialDataV2(saved)["iconId"] != "default" {
		t.Fatal("selection lost after restart")
	}
}

func TestLauncherIconConcurrentPublishHasOneWinner(t *testing.T) {
	f := newSocialFixtureV2(t)
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			code, _, err := f.call("Alice", "PUT", "/api/v1/admin/launcher-icon", map[string]any{"clientRequestId": "parallel-" + string(rune('a'+n)), "expectedVersion": 1, "iconId": "anniversary_911"})
			if err != nil {
				statuses <- 0
			} else {
				statuses <- code
			}
		}(i)
	}
	wg.Wait()
	close(statuses)
	won := 0
	for code := range statuses {
		if code == 200 {
			won++
		} else if code != 409 {
			t.Fatalf("unexpected race result %d", code)
		}
	}
	if won != 1 {
		t.Fatalf("expected one publisher, got %d", won)
	}
}

func TestLauncherIconSocketOptInAndCatchUp(t *testing.T) {
	f := newSocialFixtureV2(t)
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(f.http.URL, "http") + "/api/v1/chat/ws"
	header := http.Header{"Authorization": []string{"Bearer " + f.tokens["Bob"]}, "X-Deuterium-Launcher-Icon": []string{"1"}}
	c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	first, err := readSocket(ctx, c)
	if err != nil || first.Type != "app.launcher-icon.changed" {
		t.Fatalf("missing initial sync %v %v", first.Type, err)
	}
	f.request(t, "Alice", "PUT", "/api/v1/admin/launcher-icon", map[string]any{"clientRequestId": "socket-update", "expectedVersion": 1, "iconId": "anniversary_911"}, 200)
	changed, err := readSocket(ctx, c)
	if err != nil || changed.Type != "app.launcher-icon.changed" {
		t.Fatalf("missing change %v %v", changed.Type, err)
	}
	var payload struct {
		Version int64 `json:"version"`
	}
	if json.Unmarshal(changed.Payload, &payload) != nil || payload.Version != 2 {
		t.Fatal("wrong revision")
	}
	// Older clients do not receive the new frame ahead of their chat response.
	old, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + f.tokens["Bob"]}}})
	if err != nil {
		t.Fatal(err)
	}
	defer old.CloseNow()
	if err = sendSocket(ctx, old, "chat.send", "old-client", map[string]any{"clientMessageId": "old-message", "content": ""}); err != nil {
		t.Fatal(err)
	}
	reply, err := readSocket(ctx, old)
	if err != nil || reply.Type != "chat.send.result" {
		t.Fatalf("old protocol changed: %v %v", reply.Type, err)
	}
	// Reconnecting clients immediately learn the persisted latest revision.
	reconnected, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatal(err)
	}
	defer reconnected.CloseNow()
	latest, err := readSocket(ctx, reconnected)
	if err != nil || json.Unmarshal(latest.Payload, &payload) != nil || payload.Version != 2 {
		t.Fatalf("reconnect missed update: %v", err)
	}
}

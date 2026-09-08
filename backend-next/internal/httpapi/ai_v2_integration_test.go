//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type aiFixtureV2 struct {
	db     *store.Store
	app    *Server
	http   *httptest.Server
	users  map[string]store.User
	tokens map[string]string
	config aiConfigV2
}

func newAIFixtureV2(t *testing.T, provider http.Handler) *aiFixtureV2 {
	t.Helper()
	db := testdb.New(t)
	p := httptest.NewServer(provider)
	t.Cleanup(p.Close)
	f := &aiFixtureV2{db: db, users: map[string]store.User{}, tokens: map[string]string{}}
	for i, name := range []string{"Alice", "Bob", "Admin"} {
		u := store.User{ID: "ai_" + name, PlayerRef: "player_" + name, ServerUUID: fmt.Sprintf("01919e0f-00cb-7a82-88e3-b1d498cc001%d", i), GameID: name, QQ: fmt.Sprintf("2000%d", i), PasswordHash: "test-unused", Status: "active", IdentityStatus: "bound"}
		if _, err := db.DB.Exec("INSERT INTO identities VALUES(?,?,?,?,?,?, 'active',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?)", u.ID, u.PlayerRef, u.ServerUUID, u.GameID, u.QQ, u.PasswordHash, strings.Repeat("0", 64)); err != nil {
			t.Fatal(err)
		}
		token := identity.Secret()
		if err := db.CreateSession(context.Background(), u, store.Digest([]byte(token)), "app", "", time.Now().Add(time.Hour), ""); err != nil {
			t.Fatal(err)
		}
		f.users[name] = u
		f.tokens[name] = token
	}
	if _, err := db.DB.Exec("INSERT INTO identity_permissions VALUES(?,?)", f.users["Admin"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.config = aiDefaultsV2()
	f.config.Enabled = true
	f.config.APIKey = "fake-provider"
	f.config.Prompt = "private test system prompt"
	f.config.BaseURL = p.URL
	f.config.AllowHTTPForTest = true
	f.config.FreeQuota = 2
	f.config.TimeoutSeconds = 10
	f.app = New(db, config.Config{Development: true, PublicOrigin: "http://127.0.0.1"})
	mux := http.NewServeMux()
	f.app.registerAIConfigV2(mux, f.config, nil)
	f.http = httptest.NewServer(mux)
	t.Cleanup(func() { f.app.Close(); f.http.Close() })
	return f
}
func (f *aiFixtureV2) call(t *testing.T, user, method, path string, input any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(input)
	r, err := http.NewRequest(method, f.http.URL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if input != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		r.Header.Set("Authorization", "Bearer "+f.tokens[user])
	}
	resp, err := f.http.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}
func aiReadTestBodyV2(t *testing.T, r *http.Response) string {
	t.Helper()
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func (f *aiFixtureV2) json(t *testing.T, user, method, path string, input any, status int) map[string]any {
	t.Helper()
	r := f.call(t, user, method, path, input)
	text := aiReadTestBodyV2(t, r)
	if r.StatusCode != status {
		t.Fatalf("%s %s status=%d expected=%d", method, path, r.StatusCode, status)
	}
	var result map[string]any
	if json.Unmarshal([]byte(text), &result) != nil {
		t.Fatal("invalid response JSON")
	}
	return result
}
func aiFastTestProviderV2(calls *atomic.Int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		aiTestEventV2(w, "response.output_text.delta", map[string]string{"type": "response.output_text.delta", "delta": "AI answer"})
		aiTestCompleteV2(w, "AI answer", "completed", nil, false)
	})
}
func TestAIConcurrentIdempotencyQuotaAndIsolationV2(t *testing.T) {
	var calls atomic.Int32
	f := newAIFixtureV2(t, aiFastTestProviderV2(&calls))
	var wg sync.WaitGroup
	failures := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(map[string]string{"clientMessageId": "same-key", "content": "Question"})
			req, _ := http.NewRequest("POST", f.http.URL+"/api/v1/ai/chat/stream", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+f.tokens["Alice"])
			response, err := f.http.Client().Do(req)
			if err != nil {
				failures <- "request failed"
				return
			}
			data, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != 200 || !strings.Contains(string(data), "event: done") || strings.Contains(string(data), "SECRET_REASONING") {
				failures <- fmt.Sprintf("stream outcome invalid: status=%d body=%s", response.StatusCode, string(data))
			}
		}()
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider invoked %d times", calls.Load())
	}
	state, err := f.db.AIStateV2(context.Background(), f.users["Alice"].ID, f.config.policy(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Quota.Used != 1 || state.Quota.Reserved != 0 || state.Quota.Remaining != 1 {
		t.Fatalf("quota=%+v", state.Quota)
	}
	f.json(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "same-key", "content": "Different"}, 409)
	text := aiReadTestBodyV2(t, f.call(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "second", "content": "Question 2"}))
	if !strings.Contains(text, "event: done") {
		t.Fatal("second did not complete")
	}
	f.json(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "exhausted", "content": "Question 3"}, 429)
	if calls.Load() != 2 {
		t.Fatal("quota bypass called provider")
	}
	data := f.json(t, "Bob", "GET", "/api/v1/ai/messages", nil, 200)["data"].(map[string]any)
	if len(data["messages"].([]any)) != 0 {
		t.Fatal("cross-user history leaked")
	}
	messages, err := f.db.AIMessagesV2(context.Background(), f.users["Alice"].ID, state.Conversation.ID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	f.json(t, "Bob", "GET", "/api/v1/ai/messages?beforeMessageId="+messages[0].ID, nil, 404)
	f.json(t, "Alice", "POST", "/api/v1/ai/conversation/reset", map[string]any{}, 200)
	data = f.json(t, "Alice", "GET", "/api/v1/ai/messages", nil, 200)["data"].(map[string]any)
	if len(data["messages"].([]any)) != 0 {
		t.Fatal("old conversation restored")
	}
	f.json(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "same-key", "content": "Question"}, 409)
}
func TestAIClientDisconnectContinuesAndBlocksParallelGenerationV2(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	f := newAIFixtureV2(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		aiTestEventV2(w, "response.created", map[string]any{"type": "response.created", "response": map[string]string{"id": "ongoing"}})
		once.Do(func() { close(started) })
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		aiTestCompleteV2(w, "Completed after disconnect", "completed", nil, false)
	}))
	resp := f.call(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "disconnect", "content": "Keep working"})
	resp.Body.Close()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("provider never started")
	}
	f.json(t, "Alice", "POST", "/api/v1/ai/conversation/reset", map[string]any{}, 409)
	f.json(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "parallel", "content": "Not allowed yet"}, 409)
	close(release)
	text := aiReadTestBodyV2(t, f.call(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "disconnect", "content": "Keep working"}))
	if !strings.Contains(text, "Completed after disconnect") || !strings.Contains(text, "event: done") || calls.Load() != 1 {
		t.Fatalf("recovery failed, provider calls %d", calls.Load())
	}
}
func TestAIUnknownRetainsReservationAndNeverCallsAgainV2(t *testing.T) {
	var calls atomic.Int32
	f := newAIFixtureV2(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		aiTestEventV2(w, "response.output_text.delta", map[string]string{"type": "response.output_text.delta", "delta": "Partial retained"})
	}))
	for range 2 {
		text := aiReadTestBodyV2(t, f.call(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "uncertain", "content": "Question"}))
		if !strings.Contains(text, "Partial retained") || !strings.Contains(text, "AI_RESULT_UNKNOWN") || strings.Contains(text, "event: done") {
			t.Fatal("unknown became success or partial vanished")
		}
	}
	if calls.Load() != 1 {
		t.Fatal("unknown retried provider")
	}
	state, err := f.db.AIStateV2(context.Background(), f.users["Alice"].ID, f.config.policy(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Quota.Used != 0 || state.Quota.Reserved != 1 || state.Quota.Remaining != 1 {
		t.Fatalf("unknown quota=%+v", state.Quota)
	}
}
func TestAIKnownFailureReleasesAndAdminIsServerSideV2(t *testing.T) {
	var calls atomic.Int32
	f := newAIFixtureV2(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "PROVIDER_PRIVATE_ERROR", 429)
	}))
	text := aiReadTestBodyV2(t, f.call(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "failed", "content": "Question"}))
	if strings.Contains(text, "PROVIDER_PRIVATE_ERROR") || !strings.Contains(text, "AI_PROVIDER_UNAVAILABLE") {
		t.Fatal("provider error leaked or missing safe error")
	}
	state, err := f.db.AIStateV2(context.Background(), f.users["Alice"].ID, f.config.policy(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Quota.Used != 0 || state.Quota.Reserved != 0 {
		t.Fatal("known failure consumed quota")
	}
	var goodCalls atomic.Int32
	admin := newAIFixtureV2(t, aiFastTestProviderV2(&goodCalls))
	for i := range 4 {
		text = aiReadTestBodyV2(t, admin.call(t, "Admin", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": fmt.Sprintf("admin-%d", i), "content": "Question"}))
		if !strings.Contains(text, "event: done") {
			t.Fatal("admin limited by free quota")
		}
	}
	state, err = admin.db.AIStateV2(context.Background(), admin.users["Admin"].ID, admin.config.policy(), time.Now())
	if err != nil || !state.Quota.Unlimited || state.Quota.Used != 0 {
		t.Fatal("admin exemption not persisted correctly")
	}
	admin.json(t, "Alice", "POST", "/api/v1/ai/chat/stream", map[string]any{"clientMessageId": "spoof", "content": "Question", "admin": true}, 400)
}
func TestAIDeadlineFencesLateWorkerAndOriginalWindowV2(t *testing.T) {
	var calls atomic.Int32
	f := newAIFixtureV2(t, aiFastTestProviderV2(&calls))
	user := f.users["Alice"].ID
	now := time.Now().UTC()
	e, created, err := f.db.BeginAIV2(context.Background(), user, "stale", "Question", "model", f.config.policy(), now, time.Second)
	if err != nil || !created {
		t.Fatal(err)
	}
	if _, err = f.db.DB.Exec("UPDATE ai_requests_v2 SET deadline_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 5 SECOND) WHERE request_id=?", e.ID); err != nil {
		t.Fatal(err)
	}
	state, err := f.db.AIStateV2(context.Background(), user, f.config.policy(), now)
	if err != nil || state.Pending == nil || state.Pending.Status != "unknown" {
		t.Fatal("stale request not recovered")
	}
	e.Answer = "late answer"
	if err = f.db.FinishAIV2(context.Background(), e, "completed", "", "completed", 1, 1); err != nil {
		t.Fatal(err)
	}
	again, created, err := f.db.BeginAIV2(context.Background(), user, "stale", "Question", "model", f.config.policy(), now, time.Minute)
	if err != nil || created || again.Status != "unknown" {
		t.Fatal("late request was restarted")
	}
	boundary := store.AIWindowStartV2(now, 24).Add(24 * time.Hour)
	other := f.users["Bob"].ID
	e, created, err = f.db.BeginAIV2(context.Background(), other, "cross-window", "Question", "model", f.config.policy(), boundary.Add(-time.Second), time.Minute)
	if err != nil || !created {
		t.Fatal(err)
	}
	e.Answer = "answer"
	if err = f.db.FinishAIV2(context.Background(), e, "completed", "", "completed", 1, 1); err != nil {
		t.Fatal(err)
	}
	state, err = f.db.AIStateV2(context.Background(), other, f.config.policy(), boundary.Add(time.Second))
	if err != nil || state.Quota.Used != 0 || state.Quota.Remaining != 2 {
		t.Fatal("completion charged a different window")
	}
}
func TestAIChineseAndEmojiInputBoundaryV2(t *testing.T) {
	var calls atomic.Int32
	f := newAIFixtureV2(t, aiFastTestProviderV2(&calls))
	for i, text := range []string{strings.Repeat("中", 2000), strings.Repeat("😀", 2000)} {
		response := aiReadTestBodyV2(t, f.call(t, "Admin", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": fmt.Sprintf("long-%d", i), "content": text}))
		if !strings.Contains(response, "event: done") {
			t.Fatal("legal multi-byte message rejected")
		}
	}
	f.json(t, "Admin", "POST", "/api/v1/ai/chat/stream", map[string]string{"clientMessageId": "too-long", "content": strings.Repeat("😀", 2001)}, 400)
	if calls.Load() != 2 {
		t.Fatal("over-limit message reached provider")
	}
}

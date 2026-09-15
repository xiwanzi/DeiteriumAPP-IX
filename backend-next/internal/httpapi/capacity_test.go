package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestLongRequestSaturationLeavesControlAndHTTPAvailable(t *testing.T) {
	c := newRequestCapacity(config.Concurrency{HTTPRequests: 1, RealtimeConnections: 1, AIStreams: 1, ControlRequests: 1})
	for _, b := range []*requestBudget{c.realtime, c.ai, c.core} {
		for b.enter() {
		}
	}
	h := c.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, path := range []string{"/api/v1/account/me", "/api/v1/admin/runtime/capacity", "/health/ready"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("long requests blocked %s: %d", path, w.Code)
		}
	}
	for _, path := range []string{"/bridge/v1/admission/check", "/bridge/v1/admission/poll"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("admission blocked: %d", w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/chat/ws", nil))
	if w.Code != 503 || w.Header().Get("Retry-After") == "" {
		t.Fatal("saturated realtime request was not rejected")
	}
	if len(c.http.slots) != 0 || len(c.control.slots) != 0 {
		t.Fatal("short request slot leaked")
	}
	// An Upgrade header must not move an ordinary request into a separate pool.
	c.http.enter()
	r := httptest.NewRequest("GET", "/api/v1/account/me", nil)
	r.Header.Set("Upgrade", "websocket")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatal("Upgrade header bypassed the HTTP budget")
	}
}

func TestPublicChatReadSharesPageAndInvalidatesOnWake(t *testing.T) {
	var cache sharedChatReads
	var calls atomic.Int32
	query := func(context.Context) ([]store.ChatMessage, error) {
		calls.Add(1)
		return []store.ChatMessage{{Sequence: 7}}, nil
	}
	for i := 0; i < 100; i++ {
		if _, err := cache.load(context.Background(), 6, 1, query); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("same page read %d times", calls.Load())
	}
	if _, err := cache.load(context.Background(), 6, 2, query); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("wakeup revision did not invalidate")
	}
	cache.mu.Lock()
	cache.entries[6].expires = time.Now().Add(-time.Second)
	cache.mu.Unlock()
	if _, err := cache.load(context.Background(), 6, 2, query); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatal("TTL did not recover a missing wakeup")
	}
}

func TestPublicChatWaiterCancellationDoesNotCancelSharedRead(t *testing.T) {
	var cache sharedChatReads
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	query := func(context.Context) ([]store.ChatMessage, error) {
		close(started)
		<-release
		return []store.ChatMessage{{Sequence: 1}}, nil
	}
	go func() { _, err := cache.load(context.Background(), 0, 1, query); done <- err }()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	waiter := make(chan error, 1)
	go func() { _, err := cache.load(ctx, 0, 1, query); waiter <- err }()
	cancel()
	if err := <-waiter; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := cache.load(context.Background(), 0, 1, func(context.Context) ([]store.ChatMessage, error) { t.Fatal("shared result lost"); return nil, nil }); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentPublicChatReadersShareOneQuery(t *testing.T) {
	var cache sharedChatReads
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 20)
	query := func(context.Context) ([]store.ChatMessage, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return []store.ChatMessage{{Sequence: 8}}, nil
	}
	go func() { _, err := cache.load(context.Background(), 7, 1, query); done <- err }()
	<-started
	for i := 1; i < 20; i++ {
		go func() { _, err := cache.load(context.Background(), 7, 1, query); done <- err }()
	}
	close(release)
	for i := 0; i < 20; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("concurrent readers executed %d queries", calls.Load())
	}
}

func TestPublicChatErrorsAreNotCachedAndCursorStorageIsBounded(t *testing.T) {
	var cache sharedChatReads
	_, err := cache.load(context.Background(), 0, 1, func(context.Context) ([]store.ChatMessage, error) { return nil, errors.New("database unavailable") })
	if err == nil {
		t.Fatal("missing database error")
	}
	for i := int64(0); i < 500; i++ {
		_, err = cache.load(context.Background(), i, 1, func(context.Context) ([]store.ChatMessage, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(cache.entries) > 128 {
		t.Fatal("cursor cache grew without bound")
	}
}

func TestAIWaitingResumesWithinGenerationLimit(t *testing.T) {
	g := &aiGatewayV2{config: aiConfigV2{MaxConcurrent: 1}, active: &atomic.Int32{}}
	if !g.tryGeneration() {
		t.Fatal("initial slot unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan bool, 1)
	go func() { done <- g.waitGeneration(ctx) }()
	if g.tryGeneration() {
		t.Fatal("generation limit exceeded")
	}
	g.active.Add(-1)
	if !<-done || g.active.Load() != 1 {
		t.Fatal("waiting generation not resumed")
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if g.waitGeneration(canceled) || g.active.Load() != 1 {
		t.Fatal("cancelled wait acquired or leaked a slot")
	}
}

package httpapi

import (
	"net/http"
	"sync/atomic"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
)

type requestBudget struct {
	slots    chan struct{}
	rejected atomic.Uint64
}

func newRequestBudget(limit int) *requestBudget {
	return &requestBudget{slots: make(chan struct{}, limit)}
}

func (b *requestBudget) enter() bool {
	select {
	case b.slots <- struct{}{}:
		return true
	default:
		b.rejected.Add(1)
		return false
	}
}

type requestCapacity struct {
	http, realtime, ai, control, core *requestBudget
}

func newRequestCapacity(c config.Concurrency) *requestCapacity {
	c = c.WithDefaults()
	return &requestCapacity{newRequestBudget(c.HTTPRequests), newRequestBudget(c.RealtimeConnections),
		newRequestBudget(c.AIStreams), newRequestBudget(c.ControlRequests), newRequestBudget(32)}
}

// Classify registered operations, never client-supplied Upgrade/Accept headers.
// Core sockets and admission calls have separate budgets even during a burst of
// ordinary HTTP requests, AI streams or player reconnects.
func (c *requestCapacity) budget(r *http.Request) *requestBudget {
	switch r.Method + " " + r.URL.Path {
	case "GET /api/v1/chat/ws":
		return c.realtime
	case "GET /bridge/v1/connect":
		return c.core
	case "POST /api/v1/ai/chat/stream":
		return c.ai
	case "POST /bridge/v1/admission/check", "POST /bridge/v1/admission/poll",
		"GET /health/live", "GET /health/ready", "GET /api/v1/admin/runtime/capacity":
		return c.control
	default:
		return c.http
	}
}

func (c *requestCapacity) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		budget := c.budget(r)
		if !budget.enter() {
			w.Header().Set("Retry-After", "1")
			failure(w, r, http.StatusServiceUnavailable, "SERVER_BUSY", "请求过多，请稍后重试。")
			return
		}
		defer func() { <-budget.slots }()
		next.ServeHTTP(w, r)
	})
}

type budgetSnapshot struct {
	Active   int    `json:"active"`
	Limit    int    `json:"limit"`
	Rejected uint64 `json:"rejected"`
}

func (c *requestCapacity) snapshot() map[string]budgetSnapshot {
	out := make(map[string]budgetSnapshot)
	for name, b := range map[string]*requestBudget{"http": c.http, "realtime": c.realtime, "aiStreams": c.ai, "control": c.control, "core": c.core} {
		out[name] = budgetSnapshot{len(b.slots), cap(b.slots), b.rejected.Load()}
	}
	return out
}

func (s *Server) runtimeCapacity(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	db := s.Store.DB.Stats()
	success(w, r, map[string]any{
		"requests": s.capacity.snapshot(),
		"database": map[string]any{"limit": db.MaxOpenConnections, "open": db.OpenConnections, "inUse": db.InUse,
			"idle": db.Idle, "waitCount": db.WaitCount, "waitMilliseconds": db.WaitDuration.Milliseconds()},
		"publicChat": map[string]uint64{"databaseReads": s.chatReads.reads.Load(), "sharedReads": s.chatReads.shared.Load()},
		"ai":         map[string]any{"generating": s.ai.active.Load(), "waiting": len(s.ai.queue.waiting), "queueLimit": cap(s.ai.queue.waiting), "rejected": s.ai.queue.rejected.Load()},
	})
}

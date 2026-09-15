package httpapi

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type aiQueue struct {
	waiting  chan struct{}
	rejected atomic.Uint64
}

func (g *aiGatewayV2) tryGeneration() bool {
	for {
		active := g.active.Load()
		if active >= int32(g.config.MaxConcurrent) {
			return false
		}
		if g.active.CompareAndSwap(active, active+1) {
			return true
		}
	}
}

// Keep the existing pending/queued SSE state and request ID. Queue wait is
// bounded to ten seconds, inside the existing generation deadline's 30s margin.
// Restart recovery remains the durable request deadline; never replay a provider
// request whose outcome may be unknown.
func (g *aiGatewayV2) schedule(e store.AIExchangeV2, tokenHash string) {
	if g.tryGeneration() {
		go g.runAuthorized(e, tokenHash)
		return
	}
	select {
	case g.queue.waiting <- struct{}{}:
	default:
		g.busy(e)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(g.server.ctx, 10*time.Second)
		defer cancel()
		acquired := g.waitGeneration(ctx)
		<-g.queue.waiting
		if !acquired {
			g.busy(e)
			return
		}
		g.runAuthorized(e, tokenHash)
	}()
}

func (g *aiGatewayV2) waitGeneration(ctx context.Context) bool {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		if ctx.Err() != nil {
			return false
		}
		if g.tryGeneration() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-tick.C:
		}
	}
}

func (g *aiGatewayV2) busy(e store.AIExchangeV2) {
	g.queue.rejected.Add(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = g.server.Store.FinishAIV2(ctx, e, "failed", "AI_SERVER_BUSY", "", 0, 0)
}

func (g *aiGatewayV2) runAuthorized(e store.AIExchangeV2, tokenHash string) {
	ctx, cancel := context.WithTimeout(g.server.ctx, 5*time.Second)
	session, err := g.server.Store.Session(ctx, tokenHash)
	current, readErr := g.server.Store.AIExchangeV2(ctx, e.UserID, e.ID)
	cancel()
	if err != nil || readErr != nil || session.User.ID != e.UserID || current.Status != "pending" || !time.Now().Before(current.DeadlineAt) {
		g.active.Add(-1)
		// No provider call has started, so a known failure can release the quota.
		g.busy(e)
		return
	}
	g.run(current)
}

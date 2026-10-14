package httpapi

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// All public name/UUID lookups share this bounded pool. No completed result is
// cached here: a reused Minecraft name must never select its previous owner.
// Registration's existing, purpose-specific cache remains separate.
type profileCapacity struct {
	ctx                               context.Context
	mu                                sync.Mutex
	pending                           map[string]*profileCall
	slots                             chan struct{}
	callers                           chan struct{}
	queueMax                          int
	wait                              time.Duration
	timeout                           time.Duration
	queries, shared, rejected, failed atomic.Uint64
	queueNanos, queryNanos            atomic.Uint64
}

type profileCall struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	profile store.AdmissionProfile
	err     error
}

func (s *Server) profilePool() *profileCapacity {
	s.profileInit.Do(func() {
		ctx := s.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		s.profiles = newProfileCapacity(ctx)
	})
	return s.profiles
}

func newProfileCapacity(ctx context.Context) *profileCapacity {
	return &profileCapacity{ctx: ctx, pending: make(map[string]*profileCall), slots: make(chan struct{}, 8), callers: make(chan struct{}, 256), queueMax: 128, wait: 3 * time.Second, timeout: 8 * time.Second}
}

func (p *profileCapacity) resolve(ctx context.Context, key string, query func(context.Context) (store.AdmissionProfile, error)) (store.AdmissionProfile, error) {
	if err := ctx.Err(); err != nil {
		return store.AdmissionProfile{}, err
	}
	select {
	case p.callers <- struct{}{}:
		defer func() { <-p.callers }()
	default:
		p.rejected.Add(1)
		return store.AdmissionProfile{}, identity.ErrNameLookupBusy
	}
	p.mu.Lock()
	call := p.pending[key]
	if call == nil {
		if len(p.pending) >= cap(p.slots)+p.queueMax {
			p.mu.Unlock()
			p.rejected.Add(1)
			return store.AdmissionProfile{}, identity.ErrNameLookupBusy
		}
		workCtx, cancel := context.WithTimeout(p.ctx, p.timeout)
		call = &profileCall{done: make(chan struct{}), cancel: cancel}
		p.pending[key] = call
		go p.run(workCtx, key, call, query)
	} else {
		p.shared.Add(1)
	}
	call.waiters++
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		call.waiters--
		if call.waiters == 0 {
			call.cancel()
			// A subsequent request must not inherit cancelled work.
			if p.pending[key] == call {
				delete(p.pending, key)
			}
		}
		p.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return store.AdmissionProfile{}, ctx.Err()
	case <-call.done:
		return call.profile, call.err
	}
}

func (p *profileCapacity) run(ctx context.Context, key string, call *profileCall, query func(context.Context) (store.AdmissionProfile, error)) {
	defer call.cancel()
	defer func() {
		if call.err != nil {
			p.failed.Add(1)
		}
		p.mu.Lock()
		if p.pending[key] == call {
			delete(p.pending, key)
		}
		close(call.done)
		p.mu.Unlock()
	}()
	start := time.Now()
	timer := time.NewTimer(p.wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		call.err = ctx.Err()
		return
	case <-timer.C:
		p.rejected.Add(1)
		call.err = identity.ErrNameLookupBusy
		return
	case p.slots <- struct{}{}:
	}
	defer func() { <-p.slots }()
	p.queueNanos.Add(uint64(time.Since(start)))
	if call.err = ctx.Err(); call.err != nil {
		return
	}
	p.queries.Add(1)
	began := time.Now()
	observed := began.UTC()
	call.profile, call.err = query(ctx)
	// Every caller sharing this response must use the actual observation time,
	// not its later arrival time, for stale-proof protection.
	call.profile.ObservedAt = observed
	p.queryNanos.Add(uint64(time.Since(began)))
}

func (p *profileCapacity) snapshot() map[string]any {
	p.mu.Lock()
	pending := len(p.pending)
	p.mu.Unlock()
	return map[string]any{"active": len(p.slots), "limit": cap(p.slots), "pending": pending,
		"waitingClients": len(p.callers), "queueLimit": p.queueMax, "queries": p.queries.Load(),
		"shared": p.shared.Load(), "rejected": p.rejected.Load(), "failed": p.failed.Load(),
		"queueMilliseconds": p.queueNanos.Load() / uint64(time.Millisecond), "queryMilliseconds": p.queryNanos.Load() / uint64(time.Millisecond)}
}

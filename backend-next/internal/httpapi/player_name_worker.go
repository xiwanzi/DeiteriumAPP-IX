package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"log/slog"
	"sync"
	"time"
)

type nameRetry struct {
	due      time.Time
	attempts int
}
type playerNameQueue struct {
	mu      sync.Mutex
	pending map[string]nameRetry
	wake    chan struct{}
}

func (s *Server) observePlayerNames(uuids []string) {
	if len(uuids) == 0 {
		return
	}
	q := &s.nameQueue
	q.mu.Lock()
	for _, uuid := range uuids {
		if identity.ValidUUID(uuid) {
			q.pending[uuid] = nameRetry{}
		}
	}
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// No periodic online-player scan. Only a new authenticated session/name event,
// or an outstanding failed observation, schedules work. Reconnect snapshots
// reconcile observations lost across process restarts.
func (s *Server) playerNameWorker() {
	q := &s.nameQueue
	for s.ctx.Err() == nil {
		q.mu.Lock()
		var uuid string
		var job nameRetry
		for id, candidate := range q.pending {
			if uuid == "" || candidate.due.Before(job.due) {
				uuid = id
				job = candidate
			}
		}
		ready := uuid != "" && !time.Now().Before(job.due)
		if ready {
			delete(q.pending, uuid)
		}
		q.mu.Unlock()
		if !ready {
			var timer *time.Timer
			var due <-chan time.Time
			if uuid != "" {
				timer = time.NewTimer(time.Until(job.due))
				due = timer.C
			}
			select {
			case <-s.ctx.Done():
			case <-q.wake:
			case <-due:
			}
			if timer != nil {
				timer.Stop()
			}
			continue
		}
		if s.Config.Development && s.AdmissionResolver == nil {
			continue
		}
		err := s.syncObservedPlayerName(uuid)
		if err == nil || errors.Is(err, store.ErrSocialNotFound) {
			continue
		}
		job.attempts++
		backoff := time.Second * time.Duration(1<<min(job.attempts, 8))
		job.due = time.Now().Add(backoff)
		q.mu.Lock()
		if _, newer := q.pending[uuid]; !newer {
			q.pending[uuid] = job
		}
		q.mu.Unlock()
		slog.Warn("player name observation deferred", "uuid", uuid, "retryIn", backoff, "error", err)
	}
}

func (s *Server) syncObservedPlayerName(uuid string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
	defer cancel()
	live, e := s.Core.Player("", uuid)
	if e != nil {
		return e
	}
	if !live.Online {
		return nil
	}
	previous, e := s.Store.KnownPlayerName(ctx, uuid)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	if previous == live.GameID {
		return nil
	}
	observedAt := time.Now().UTC()
	canonical, e := s.resolveAdmissionProfile(ctx, live.GameID)
	if e != nil {
		return e
	}
	if canonical.UUID != uuid {
		return identity.ErrNameLookupUnavailable
	}
	if !canonical.ObservedAt.IsZero() {
		observedAt = canonical.ObservedAt
	}
	return s.syncOfficialNameObserved(ctx, canonical, observedAt)
}

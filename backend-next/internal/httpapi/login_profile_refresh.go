package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type loginProfileJob struct{ userID, uuid string }
type loginProfileRefresh struct {
	mu                         sync.Mutex
	next                       map[string]time.Time
	jobs                       chan loginProfileJob
	done                       chan struct{}
	completed, failed, dropped atomic.Uint64
}

func (s *Server) queueLoginProfile(user store.User) {
	uuid := user.ProfileUUID
	if uuid == "" && user.GameVerified {
		uuid = user.ServerUUID
	}
	if !store.ValidAdmissionUUID(uuid) || s.ctx.Err() != nil {
		return
	}
	q := &s.loginProfiles
	now := time.Now()
	q.mu.Lock()
	defer q.mu.Unlock()
	if next, exists := q.next[uuid]; exists && (next.IsZero() || now.Before(next)) {
		return
	}
	// Only successful, authenticated accounts enter here; still bound memory.
	if len(q.next) >= 512 {
		for key, next := range q.next {
			if !next.IsZero() && !now.Before(next) {
				delete(q.next, key)
			}
		}
		if len(q.next) >= 512 {
			q.dropped.Add(1)
			return
		}
	}
	select {
	case q.jobs <- loginProfileJob{user.ID, uuid}:
		q.next[uuid] = time.Time{} // queued/running, never evicted or duplicated
	default: // A later login/online event can retry; login never waits for the worker.
		q.dropped.Add(1)
	}
}

func (s *Server) loginProfileWorker() {
	defer close(s.loginProfiles.done)
	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.loginProfiles.jobs:
			if s.ctx.Err() != nil {
				return
			}
			cooldown := 10 * time.Minute
			if err := s.refreshLoginProfile(job); err != nil {
				s.loginProfiles.failed.Add(1)
				cooldown = time.Minute
			} else {
				s.loginProfiles.completed.Add(1)
			}
			s.loginProfiles.mu.Lock()
			s.loginProfiles.next[job.uuid] = time.Now().Add(cooldown)
			s.loginProfiles.mu.Unlock()
		}
	}
}

func (s *Server) refreshLoginProfile(job loginProfileJob) error {
	// Tests/dev never accidentally contact Mojang. Production shares the same
	// bounded resolver and stale-proof fences as explicit name/UUID resolution.
	if s.Config.Development && s.OfficialUUIDResolver == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
	defer cancel()
	user, err := s.Store.UserByProfileUUID(ctx, job.uuid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if user.ID != job.userID || user.Status != "active" {
		return nil
	}
	profile, err := s.resolveOfficialUUID(ctx, job.uuid)
	if err != nil {
		return err
	}
	if profile.UUID != job.uuid {
		return store.ErrNameConflict
	}
	if user.GameVerified {
		return s.syncOfficialNameObserved(ctx, profile, profile.ObservedAt)
	}
	return s.Store.SyncRegisteredName(ctx, user, profile, profile.ObservedAt)
}

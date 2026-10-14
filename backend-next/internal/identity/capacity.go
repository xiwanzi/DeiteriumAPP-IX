package identity

import (
	"context"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"time"
)

// External identity calls never occupy a memory-heavy password worker.
func (s *Service) resolveGameName(ctx context.Context, name string) (store.AdmissionProfile, error) {
	// The server-wide resolver owns name/UUID capacity for every entry point.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return s.GameNameResolver(ctx, name)
}

func (s *Service) verifyPassword(ctx context.Context, hash, password string) (bool, error) {
	if err := s.acquirePasswordSlot(ctx); err != nil {
		return false, err
	}
	defer func() { <-s.slots }()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	start := time.Now()
	defer s.passwordTiming.observe(start)
	return Verify(hash, password), nil
}

// The KDF still runs in two slots. A small, time-bounded waiting room absorbs
// normal login bursts without allocating more password-hashing memory.
func (s *Service) acquirePasswordSlot(ctx context.Context) error {
	start := time.Now()
	defer s.passwordQueueTiming.observe(start)
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case s.slots <- struct{}{}:
		return nil
	default:
	}
	select {
	case s.waiting <- struct{}{}:
	default:
		s.passwordRejected.Add(1)
		return ErrBusy
	}
	defer func() { <-s.waiting }()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		s.passwordRejected.Add(1)
		return ErrBusy
	case s.slots <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-s.slots
			return err
		}
		return nil
	}
}

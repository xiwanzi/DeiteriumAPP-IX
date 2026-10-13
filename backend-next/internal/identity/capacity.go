package identity

import (
	"context"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"time"
)

// External identity calls never occupy a memory-heavy password worker.
func (s *Service) resolveGameName(ctx context.Context, name string) (store.AdmissionProfile, error) {
	select {
	case s.nameLookups <- struct{}{}:
		defer func() { <-s.nameLookups }()
	default:
		return store.AdmissionProfile{}, ErrNameLookupUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return s.GameNameResolver(ctx, name)
}

func (s *Service) verifyPassword(ctx context.Context, hash, password string) (bool, error) {
	if err := s.acquirePasswordSlot(ctx); err != nil {
		return false, err
	}
	defer func() { <-s.slots }()
	return Verify(hash, password), nil
}

// The KDF still runs in two slots. A small, time-bounded waiting room absorbs
// normal login bursts without allocating more password-hashing memory.
func (s *Service) acquirePasswordSlot(ctx context.Context) error {
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
		return ErrBusy
	}
	defer func() { <-s.waiting }()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ErrBusy
	case s.slots <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-s.slots
			return err
		}
		return nil
	}
}

package identity

import (
	"context"
	"time"
)

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

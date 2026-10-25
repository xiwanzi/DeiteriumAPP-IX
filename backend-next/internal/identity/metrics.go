package identity

import (
	"sync/atomic"
	"time"
)

type timing struct {
	count, nanos, maximum atomic.Uint64
}

func (t *timing) observe(start time.Time) {
	n := uint64(time.Since(start))
	t.count.Add(1)
	t.nanos.Add(n)
	for old := t.maximum.Load(); old < n; old = t.maximum.Load() {
		if t.maximum.CompareAndSwap(old, n) {
			break
		}
	}
}

func (t *timing) snapshot() map[string]uint64 {
	return map[string]uint64{"count": t.count.Load(), "totalMilliseconds": t.nanos.Load() / uint64(time.Millisecond), "maxMilliseconds": t.maximum.Load() / uint64(time.Millisecond)}
}

func (s *Service) CapacitySnapshot() map[string]any {
	return map[string]any{"passwordWorkers": len(s.slots), "passwordWorkerLimit": cap(s.slots),
		"passwordWaiting": len(s.waiting), "passwordQueueLimit": cap(s.waiting), "passwordRejected": s.passwordRejected.Load(),
		"localLogins": s.localLogins.Load(), "nameLoginLookups": s.nameLoginLookups.Load(),
		"login": s.loginTiming.snapshot(), "passwordQueue": s.passwordQueueTiming.snapshot(),
		"passwordCalculation": s.passwordTiming.snapshot(), "sessionWrite": s.sessionTiming.snapshot()}
}

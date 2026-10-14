package httpapi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func waitProfileClients(t *testing.T, pool *profileCapacity, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(pool.callers) == n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected %d waiting clients, got %d", n, len(pool.callers))
}

func TestProfileCapacitySharesProofAndSurvivesFirstCallerCancellation(t *testing.T) {
	pool := newProfileCapacity(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	query := func(ctx context.Context) (store.AdmissionProfile, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-ctx.Done():
			return store.AdmissionProfile{}, ctx.Err()
		case <-release:
			return store.AdmissionProfile{UUID: admissionTestProfileUUID, Name: "Player"}, nil
		}
	}
	first := make(chan error, 1)
	go func() { _, e := pool.resolve(ctx, "name:player", query); first <- e }()
	<-started
	followerStarted := time.Now().UTC()
	second := make(chan store.AdmissionProfile, 1)
	errorsOut := make(chan error, 1)
	go func() { p, e := pool.resolve(context.Background(), "name:player", query); second <- p; errorsOut <- e }()
	waitProfileClients(t, pool, 2)
	cancel()
	if !errors.Is(<-first, context.Canceled) {
		t.Fatal("cancelled caller succeeded")
	}
	close(release)
	profile := <-second
	if e := <-errorsOut; e != nil || calls.Load() != 1 || profile.ObservedAt.IsZero() {
		t.Fatal("shared proof lost", e, calls.Load(), profile)
	}
	if profile.ObservedAt.After(followerStarted) {
		t.Fatal("follower gave an older proof a newer observation time")
	}
	// A completed result must not hide immediate name reuse.
	next, e := pool.resolve(context.Background(), "name:player", func(context.Context) (store.AdmissionProfile, error) {
		return store.AdmissionProfile{UUID: "22222222-2222-4222-8222-222222222222", Name: "Player"}, nil
	})
	if e != nil || next.UUID == profile.UUID {
		t.Fatal("completed lookup cached a stale owner", e)
	}
}

const admissionTestProfileUUID = "11111111-1111-4111-8111-111111111111"

func TestProfileCapacityHundredDistinctQueriesStayBounded(t *testing.T) {
	pool := newProfileCapacity(context.Background())
	var active, peak atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	start := time.Now()
	for n := 0; n < 100; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, e := pool.resolve(context.Background(), fmt.Sprint(n), func(ctx context.Context) (store.AdmissionProfile, error) {
				a := active.Add(1)
				defer active.Add(-1)
				for old := peak.Load(); old < a; old = peak.Load() {
					if peak.CompareAndSwap(old, a) {
						break
					}
				}
				select {
				case <-ctx.Done():
					return store.AdmissionProfile{}, ctx.Err()
				case <-time.After(20 * time.Millisecond):
				}
				return store.AdmissionProfile{UUID: admissionTestProfileUUID, Name: fmt.Sprint("Player", n)}, nil
			})
			errs <- e
		}(n)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	if peak.Load() > 8 || len(pool.slots) != 0 || len(pool.callers) != 0 {
		t.Fatal("capacity leaked or exceeded", peak.Load())
	}
	t.Logf("100 distinct lookups, 20ms simulated upstream: elapsed=%s peak=%d", time.Since(start), peak.Load())
}

func TestProfileCapacityQueueTimeoutAndCancellationReleaseWork(t *testing.T) {
	pool := newProfileCapacity(context.Background())
	pool.slots = make(chan struct{}, 1)
	pool.queueMax = 1
	pool.wait = 30 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, done := make(chan struct{}), make(chan error, 1)
	go func() {
		_, e := pool.resolve(ctx, "held", func(ctx context.Context) (store.AdmissionProfile, error) {
			close(started)
			<-ctx.Done()
			return store.AdmissionProfile{}, ctx.Err()
		})
		done <- e
	}()
	<-started
	_, e := pool.resolve(context.Background(), "queued", func(context.Context) (store.AdmissionProfile, error) {
		t.Error("timed-out query executed")
		return store.AdmissionProfile{}, nil
	})
	if !errors.Is(e, identity.ErrNameLookupBusy) {
		t.Fatal("no queue deadline", e)
	}
	cancel()
	<-done
	deadline := time.Now().Add(time.Second)
	for len(pool.slots) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(pool.slots) != 0 || len(pool.callers) != 0 {
		t.Fatal("abandoned lookup retained capacity")
	}
}

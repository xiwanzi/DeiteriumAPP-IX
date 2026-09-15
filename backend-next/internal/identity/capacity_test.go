package identity

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPasswordBurstWaitsWithoutIncreasingKDFConcurrency(t *testing.T) {
	s := New(nil)
	for i := 0; i < 2; i++ {
		if err := s.acquirePasswordSlot(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.acquirePasswordSlot(ctx) }()
	<-s.slots
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(s.slots) != 2 || len(s.waiting) != 0 {
		t.Fatal("KDF limit or waiting slot leaked")
	}
	for i := 0; i < cap(s.waiting); i++ {
		s.waiting <- struct{}{}
	}
	if err := s.acquirePasswordSlot(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatal("full waiting room accepted a request")
	}
}

func TestCancelledPasswordRequestDoesNotReserveSlot(t *testing.T) {
	s := New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.acquirePasswordSlot(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(s.slots) != 0 || len(s.waiting) != 0 {
		t.Fatal("cancelled request leaked capacity")
	}
}

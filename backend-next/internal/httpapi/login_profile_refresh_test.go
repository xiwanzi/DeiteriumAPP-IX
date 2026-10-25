package httpapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestLoginProfileQueueCoalescesAndBoundsPendingWork(t *testing.T) {
	s := &Server{ctx: context.Background(), loginProfiles: loginProfileRefresh{next: make(map[string]time.Time), jobs: make(chan loginProfileJob, 128)}}
	user := func(n int) store.User {
		return store.User{ID: fmt.Sprint(n), ProfileUUID: fmt.Sprintf("d97161f9-2a7c-4abd-a8e7-%012d", n)}
	}
	for range 20 {
		s.queueLoginProfile(user(1))
	}
	if len(s.loginProfiles.jobs) != 1 {
		t.Fatal("same UUID was queued repeatedly")
	}
	for n := 2; n <= 300; n++ {
		s.queueLoginProfile(user(n))
	}
	if len(s.loginProfiles.jobs) != 128 || len(s.loginProfiles.next) != 128 {
		t.Fatal("queue or bookkeeping grew beyond bound")
	}
	// A task older than the cooldown still counts as pending until completed.
	job := <-s.loginProfiles.jobs
	s.queueLoginProfile(user(1))
	if len(s.loginProfiles.jobs) != 127 {
		t.Fatal("in-flight task was duplicated")
	}
	s.loginProfiles.next[job.uuid] = time.Now().Add(-time.Minute)
	s.queueLoginProfile(user(1))
	if len(s.loginProfiles.jobs) != 128 {
		t.Fatal("expired completed task could not refresh")
	}
}

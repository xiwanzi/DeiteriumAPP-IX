//go:build integration

package store_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

func TestEmailClaimsAreExclusiveAndExpiredLeasesRecover(t *testing.T) {
	s := testdb.New(t)
	ctx := context.Background()
	for _, id := range []string{"mail-first", "mail-second"} {
		if _, err := s.DB.Exec(`INSERT INTO admin_email_outbox_v204(event_id,case_version,case_state,next_attempt_at,created_at) VALUES(?,0,'TEST',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, id); err != nil {
			t.Fatal(err)
		}
	}
	var wait sync.WaitGroup
	ids := make(chan string, 2)
	errorsFound := make(chan error, 2)
	start := make(chan struct{})
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			event, err := s.ClaimEmailV204(ctx)
			if err != nil {
				errorsFound <- err
				return
			}
			ids <- event.EventID
		}()
	}
	close(start)
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
	if len(ids) != 2 || <-ids == <-ids {
		t.Fatal("concurrent workers did not claim distinct emails")
	}
	if _, err := s.ClaimEmailV204(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unexpired delivery was claimed again: %v", err)
	}
	if _, err := s.DB.Exec(`UPDATE admin_email_outbox_v204 SET lease_until=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE event_id='mail-first'`); err != nil {
		t.Fatal(err)
	}
	event, err := s.ClaimEmailV204(ctx)
	if err != nil || event.EventID != "mail-first" || event.Attempts != 2 {
		t.Fatalf("expired lease was not recovered: event=%+v error=%v", event, err)
	}
	stale := event
	stale.Attempts = 1
	if err := s.FinishEmailV204(ctx, stale, nil); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM admin_email_outbox_v204 WHERE event_id=?`, event.EventID).Scan(&status); err != nil || status != "SENDING" {
		t.Fatalf("stale sender changed a newer lease: status=%s error=%v", status, err)
	}
	if err := s.FinishEmailV204(ctx, event, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimEmailV204(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("accepted email was claimed again: %v", err)
	}
}

package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLedgerCursorBindsIdentityFiltersAndCollectionV204(t *testing.T) {
	now := time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC)
	f, limit, err := ledgerQueryV204(httptest.NewRequest("GET", "/wallet/records?limit=100&direction=expense", nil), "alice", "personal", now)
	if err != nil || limit != 25 {
		t.Fatal("invalid initial page")
	}
	f.Snapshot = "30"
	f.BeforeSequence = "20"
	f.BeforeTime = now.Add(-time.Hour)
	cursor := encodeCursorV204(f)
	for _, test := range []struct {
		actor, scope, path string
		valid              bool
	}{{"alice", "personal", "?cursor=" + cursor, true}, {"bob", "personal", "?cursor=" + cursor, false}, {"alice", "admin", "?cursor=" + cursor, false}, {"alice", "personal", "?cursor=" + cursor + "&direction=income", false}, {"alice", "personal", "?playerRef=someone-else", false}, {"alice", "personal", "?direction=expense&direction=income", false}, {"alice", "personal", "?businessType=INVALID", false}} {
		_, _, err := ledgerQueryV204(httptest.NewRequest("GET", "/records"+test.path, nil), test.actor, test.scope, now)
		if (err == nil) != test.valid {
			t.Fatalf("unexpected query result: %+v %v", test, err)
		}
	}
}

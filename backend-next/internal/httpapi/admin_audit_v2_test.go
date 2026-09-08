package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAdminAuditV2QueryBoundsAndCursorScope(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	request := func(query, user string) (int, error) {
		limit, _, err := adminAuditQueryV2(httptest.NewRequest("GET", "/"+query, nil), user, now)
		return limit, err
	}
	limit, filter, err := adminAuditQueryV2(httptest.NewRequest("GET", "/?actorId=local-cli&limit=100", nil), "alice", now)
	if err != nil || limit != 100 || filter.To.Sub(filter.From) != 30*24*time.Hour || filter.ActorID != "local-cli" {
		t.Fatal("default interval or actor filter changed")
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?limit=1&limit=2", "?cursor=invalid", "?password=secret", "?actorId=" + strings.Repeat("a", 65), "?actorId=a%20b", "?from=2026-01-01T00:00:00Z", "?to=2027-01-01T00:00:00Z", "?from=2026-09-09T00:00:00Z", "?from=not-a-date"} {
		if _, err := request(query, "alice"); err == nil {
			t.Fatalf("invalid query accepted: %s", query)
		}
	}
	filter.SnapshotSequence = 20
	filter.BeforeSequence = 10
	filter.BeforeTime = now.Add(-time.Hour)
	cursor := encodeAdminAuditCursorV2(adminAuditCursorV2{UserID: "alice", Filter: filter})
	if _, err := request("?cursor="+cursor, "alice"); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"?cursor=" + cursor + "&actorId=bob", "?cursor=" + cursor + "&to=2026-09-08T11:00:00Z"} {
		if _, err := request(query, "alice"); err == nil {
			t.Fatal("cursor filter changed")
		}
	}
	if _, err := request("?cursor="+cursor, "bob"); err == nil {
		t.Fatal("cursor crossed accounts")
	}
}

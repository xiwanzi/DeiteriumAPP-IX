//go:build integration

package httpapi

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAdminAuditV2PermissionFilteringAndFrozenPagination(t *testing.T) {
	f := newSocialFixtureV2(t)
	path := "/api/v1/admin/audit-events"
	f.request(t, "", "GET", path, nil, 401)
	f.request(t, "Alice", "GET", path, nil, 403)
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "audit.read"); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Grant(context.Background(), f.users["Bob"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Bob", "GET", path, nil, 200)
	stamp := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	for _, resource := range []string{"audit-one", "audit-two", "audit-three"} {
		if _, err := f.store.DB.Exec(`INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,?)`, "test-actor", "test.action", resource, stamp); err != nil {
			t.Fatal(err)
		}
	}
	page := f.request(t, "Alice", "GET", path+"?actorId=test-actor&action=test.action&limit=2", nil, 200)
	events := page["data"].([]any)
	if len(events) != 2 || events[0].(map[string]any)["resourceId"] != "audit-three" || events[1].(map[string]any)["resourceId"] != "audit-two" {
		t.Fatal("same-time order unstable")
	}
	for _, event := range events {
		row := event.(map[string]any)
		if len(row) != 5 || row["actorId"] != "test-actor" || row["action"] != "test.action" {
			t.Fatal("audit projection leaked or changed")
		}
		if _, ok := row["eventId"].(string); !ok {
			t.Fatal("sequence precision lost")
		}
	}
	cursor := page["page"].(map[string]any)["nextCursor"].(string)
	// Even a newly inserted event with an old timestamp cannot enter this snapshot.
	if _, err := f.store.DB.Exec(`INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,?)`, "test-actor", "test.action", "audit-late", stamp.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	last := f.request(t, "Alice", "GET", path+"?cursor="+url.QueryEscape(cursor), nil, 200)
	if list := last["data"].([]any); len(list) != 1 || list[0].(map[string]any)["resourceId"] != "audit-one" || last["page"].(map[string]any)["hasMore"] != false {
		t.Fatal("snapshot changed between pages")
	}
	f.request(t, "Bob", "GET", path+"?cursor="+url.QueryEscape(cursor), nil, 400)
	f.request(t, "Alice", "GET", path+"?cursor="+url.QueryEscape(cursor)+"&action=another", nil, 400)
	empty := f.request(t, "Alice", "GET", path+"?resourceId=missing", nil, 200)
	if len(empty["data"].([]any)) != 0 {
		t.Fatal("unknown resource returned invented records")
	}
	filtered := f.request(t, "Alice", "GET", path+"?resourceId=audit-two", nil, 200)
	if len(filtered["data"].([]any)) != 1 {
		t.Fatal("exact resource filter failed")
	}
	caseChanged := f.request(t, "Alice", "GET", path+"?resourceId=AUDIT-TWO", nil, 200)
	if len(caseChanged["data"].([]any)) != 0 {
		t.Fatal("resource filter ignored case")
	}
	if _, err := f.store.DB.Exec(`DELETE FROM identity_permissions WHERE user_id=?`, f.users["Alice"].ID); err != nil {
		t.Fatal(err)
	}
	f.request(t, "Alice", "GET", path+"?cursor="+url.QueryEscape(cursor), nil, 403)
}

func TestAdminAuditV2RejectsUnboundedReads(t *testing.T) {
	f := newSocialFixtureV2(t)
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "audit.read"); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"limit=101", "limit=1&limit=2", "actorId=" + strings.Repeat("a", 65), "from=2020-01-01T00:00:00Z", "to=2099-01-01T00:00:00Z", "content=private-body", "cursor=broken"} {
		f.request(t, "Alice", "GET", "/api/v1/admin/audit-events?"+query, nil, 400)
	}
}

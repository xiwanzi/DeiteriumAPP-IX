//go:build integration

package httpapi

import (
	"context"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"testing"
	"time"
)

func TestSupportHeaderlessReplyAndAlias(t *testing.T) {
	f := mailSupportFixture(t)
	inbox := supportTestInbox(t, f)
	ctx := context.Background()
	create := func(key string) string {
		data := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", store.SupportCreate{ClientRequestID: key, Email: "player@example.com", Subject: "核对游戏 ID", Category: "GAME", Body: "请核对。"}, 200))
		return data["ticketId"].(string)
	}
	ticket := create("headerless-create")
	alias := "199901010001"
	deletionSQL(t, f, "INSERT INTO support_ticket_aliases VALUES(?,?,UTC_TIMESTAMP(6))", alias, ticket)
	deletionSQL(t, f, "UPDATE support_tickets SET status='CLOSED' WHERE ticket_id=?", ticket)
	ingest := func(uid uint32, email, body string) store.IngestResult {
		result, err := f.store.IngestSupportMail(ctx, inbox, 11, uid, supportRawMail(email, email, store.ID("reply-")+"@example.com", "Re: 核对游戏 ID", body, ""), time.Now(), f.app.Config.SMTPKey)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	if r := ingest(1, "player@example.com", "回复内容\n工单编号：\u00a0"+ticket); r.TicketID != ticket {
		t.Fatalf("headerless split: %+v", r)
	}
	if r := ingest(2, "player@example.com", "工单编号 "+alias+"\n工单编号 "+ticket); r.TicketID != ticket {
		t.Fatalf("alias split: %+v", r)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM support_tickets WHERE status='OPEN' AND ticket_id=?", ticket) != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM mail_outbox WHERE kind='TICKET_CREATED'") != 0 {
		t.Fatal("reply did not reopen or sent receipt")
	}
	detail, err := f.store.SupportDetail(ctx, alias)
	if err != nil || detail["ticket"].(store.SupportTicket).ID != ticket || len(detail["messages"].([]store.SupportMessage)) != 3 {
		t.Fatal("alias/history", err)
	}
	if r := ingest(3, "outsider@example.com", "工单编号："+ticket); r.TicketID == ticket {
		t.Fatal("cross-email merge")
	}
	second := create("second-headerless")
	before := deletionCount(t, f, "SELECT COUNT(*) FROM support_messages")
	if r := ingest(4, "player@example.com", "工单编号："+ticket+"\n工单编号："+second); r.Status != "NEEDS_REVIEW" {
		t.Fatalf("ambiguous reply: %+v", r)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM support_messages") != before {
		t.Fatal("ambiguous reply mutated history")
	}
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets/"+alias, store.SupportChange{ClientRequestID: "alias-close", ExpectedVersion: 3, Status: "CLOSED"}, 200)
	if deletionCount(t, f, "SELECT COUNT(*) FROM support_tickets WHERE ticket_id=? AND status='CLOSED'", ticket) != 1 {
		t.Fatal("alias change missed canonical")
	}
}

func TestSupportQueueOrderAndClosedSearch(t *testing.T) {
	f := mailSupportFixture(t)
	supportTestInbox(t, f)
	ctx := context.Background()
	expected := []string{}
	for _, state := range []string{"OPEN", "IN_PROGRESS", "WAITING_CUSTOMER", "RESOLVED", "CLOSED"} {
		data := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", store.SupportCreate{ClientRequestID: "queue-" + state, Email: "player@example.com", Subject: "排序工单 " + state, Category: "GAME", Body: "列表验证。"}, 200))
		id := data["ticketId"].(string)
		expected = append(expected, id)
		deletionSQL(t, f, "UPDATE support_tickets SET status=? WHERE ticket_id=?", state, id)
	}
	for _, tc := range []struct {
		status, search string
		want           []string
	}{{"", "", expected[:4]}, {"", "  ", expected[:4]}, {"", "排序工单", expected}, {"CLOSED", "", expected[4:]}, {"", expected[4], expected[4:]}} {
		rows, err := f.store.SupportTickets(ctx, f.users["Bob"].ID, tc.status, "", tc.search, "", 0)
		if err != nil || len(rows) != len(tc.want) {
			t.Fatalf("queue %q %q: %v rows=%d", tc.status, tc.search, err, len(rows))
		}
		for i, row := range rows {
			if row.ID != tc.want[i] {
				t.Fatalf("queue order: %v", rows)
			}
		}
	}
}

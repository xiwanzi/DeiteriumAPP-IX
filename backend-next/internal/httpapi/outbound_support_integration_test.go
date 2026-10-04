//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestStaffCreateAndRenameWebOriginAndCSRF(t *testing.T) {
	f := mailSupportFixture(t)
	supportTestInbox(t, f)
	token := identity.Secret()
	if err := f.store.CreateSession(context.Background(), f.users["Alice"], store.Digest([]byte(token)), "web", "staff-action-csrf", time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct {
		path string
		body any
	}{
		{"/api/v1/admin/support/tickets", store.SupportCreate{ClientRequestID: "web-create", Email: "player@example.com", Subject: "主动邮件", Category: "OTHER", Body: "玩家支持邮件。"}},
		{"/api/v1/admin/accounts/" + f.users["Bob"].ID + "/game-id", store.AdminGameIDChange{ClientRequestID: "web-rename", ExpectedVersion: 1, GameID: "bOB"}},
	} {
		for _, invalid := range []struct{ origin, csrf string }{{"https://outside.invalid", "staff-action-csrf"}, {f.http.URL, ""}} {
			raw, _ := json.Marshal(route.body)
			request, _ := http.NewRequest("POST", f.http.URL+route.path, strings.NewReader(string(raw)))
			request.AddCookie(&http.Cookie{Name: "deuterium_dev_session", Value: token})
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", invalid.origin)
			request.Header.Set("X-CSRF-Token", invalid.csrf)
			response, err := f.http.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != http.StatusForbidden {
				t.Fatalf("unsafe web mutation accepted: %s status %d", route.path, response.StatusCode)
			}
		}
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM support_tickets") != 0 || deletionCount(t, f, "SELECT COUNT(*) FROM identities WHERE id=? AND game_id='Bob'", f.users["Bob"].ID) != 1 {
		t.Fatal("rejected web mutation altered data")
	}
}

func TestOutboundSupportCreateAndReplyThread(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	inbox := supportTestInbox(t, f)
	in := store.SupportCreate{ClientRequestID: "staff-create", Email: "NewPlayer@example.com", Subject: "主动联系玩家", Category: "GAME", Body: "您好，我们想确认您反馈的游戏情况。"}
	f.request(t, "Carol", "POST", "/api/v1/admin/support/tickets", in, 403)
	f.request(t, "", "POST", "/api/v1/admin/support/tickets", in, 401)
	created := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", in, 200))
	ticket, event := created["ticketId"].(string), created["eventId"].(string)
	if !regexp.MustCompile(`^[0-9]{12}$`).MatchString(ticket) || created["status"] != "WAITING_CUSTOMER" {
		t.Fatal(created)
	}
	replay := socialDataV2(f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", in, 200))
	if replay["ticketId"] != ticket || replay["eventId"] != event {
		t.Fatal("create retry duplicated the ticket or mail")
	}
	changed := in
	changed.Body += "changed"
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", changed, 409)
	detail, err := f.store.SupportDetail(ctx, ticket)
	if err != nil {
		t.Fatal(err)
	}
	row := detail["ticket"].(store.SupportTicket)
	if row.Source != "ADMIN" || row.AssigneeID != f.users["Bob"].ID || row.RequesterID != "" || row.RequesterEmail != "newplayer@example.com" {
		t.Fatal("incorrect staff ticket metadata", row)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM support_messages WHERE ticket_id=? AND direction='OUTBOUND'", ticket) != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM mail_outbox WHERE reference_id=?", ticket) != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM mail_outbox WHERE reference_id=? AND kind='TICKET_CREATED'", ticket) != 0 {
		t.Fatal("unexpected receipt or duplicate message")
	}
	if err := f.app.processSystemMail(ctx, func(_ context.Context, settings notify.Settings, _, _ string, message notify.Message) error {
		if settings.From != "enterprise@example.com" || message.FromName != notify.SupportDisplayName || !strings.Contains(message.Subject, ticket) || !strings.Contains(message.HTML, ticket) || !strings.Contains(message.Text, ticket) || message.InReplyTo != "" || message.ReplyTo != "support@example.com" || len(message.Headers) != 0 {
			t.Fatal("first mail is not a numbered human reply", message.Subject)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// A second staff mail must work before the player has sent any inbound mail.
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets/"+ticket, store.SupportChange{ClientRequestID: "second-staff-mail", ExpectedVersion: 1, Body: "补充说明。", Visibility: "PUBLIC"}, 200)
	var reference string
	if err := f.store.DB.QueryRow("SELECT in_reply_to FROM mail_outbox WHERE reference_id=? AND event_id<>?", ticket, event).Scan(&reference); err != nil || reference != event+"@deuterium-notifications" {
		t.Fatal("follow-up without inbound mail lost the thread", err)
	}
	result, err := f.store.IngestSupportMail(ctx, inbox, 11, 1, supportRawMail("newplayer@example.com", "newplayer@example.com", "staff-ticket-reply@example.com", "回复", "已经收到，补充情况。", "In-Reply-To: <"+event+"@deuterium-notifications>\r\n"), time.Now(), f.app.Config.SMTPKey)
	if err != nil || result.TicketID != ticket {
		t.Fatal("player reply did not join the staff-created ticket", err)
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM mail_outbox WHERE reference_id=? AND kind='TICKET_CREATED'", ticket) != 0 || deletionCount(t, f, "SELECT COUNT(*) FROM support_tickets WHERE ticket_id=? AND status='OPEN'", ticket) != 1 {
		t.Fatal("player reply queued a receipt or did not reopen")
	}
	other, err := f.store.IngestSupportMail(ctx, inbox, 11, 2, supportRawMail("outsider@example.com", "outsider@example.com", "outsider-staff@example.com", "[工单 "+ticket+"] 回复", "不能进入他人的工单。", "In-Reply-To: <"+event+"@deuterium-notifications>\r\n"), time.Now(), f.app.Config.SMTPKey)
	if err != nil || other.TicketID == ticket {
		t.Fatal("another recipient joined the original ticket", err)
	}
}

func TestOutboundSupportValidationAtomicityAndConcurrency(t *testing.T) {
	f := mailSupportFixture(t)
	supportTestInbox(t, f)
	ctx := context.Background()
	in := store.SupportCreate{ClientRequestID: "parallel-create", Email: "player@example.com", Subject: "并发建单", Category: "OTHER", Body: "确认不重复发送。"}
	for _, email := range []string{"not-an-email", "support@example.com", "enterprise@example.com", "dm@example.com", "system@example.com", "victim@example.com\r\nBcc: third@example.com"} {
		bad := in
		bad.Email = email
		f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", bad, 400)
	}
	deletionSQL(t, f, "UPDATE mail_channels SET settings_json=JSON_SET(settings_json,'$.enabled',false) WHERE channel_id='ENTERPRISE'")
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets", in, 409)
	deletionSQL(t, f, "UPDATE mail_channels SET settings_json=JSON_SET(settings_json,'$.enabled',true) WHERE channel_id='ENTERPRISE'")
	if _, err := f.store.CreateSupport(ctx, f.users["Bob"].ID, in, nil); err == nil {
		t.Fatal("unavailable encryption did not roll back")
	}
	if deletionCount(t, f, "SELECT COUNT(*) FROM support_tickets") != 0 || deletionCount(t, f, "SELECT COUNT(*) FROM support_messages") != 0 {
		t.Fatal("failed create left ticket data")
	}
	deletionSQL(t, f, "UPDATE identities SET email='player@example.com' WHERE id=?", f.users["Carol"].ID)
	var wg sync.WaitGroup
	responses := make(chan string, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, err := f.store.CreateSupport(ctx, f.users["Bob"].ID, in, f.app.Config.SMTPKey)
			if err != nil {
				responses <- "ERROR:" + err.Error()
				return
			}
			var result map[string]any
			json.Unmarshal(raw, &result)
			responses <- fmt.Sprint(result["ticketId"])
		}()
	}
	wg.Wait()
	close(responses)
	unique := map[string]bool{}
	for response := range responses {
		if strings.HasPrefix(response, "ERROR:") {
			t.Fatal(response)
		}
		unique[response] = true
	}
	if len(unique) != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM support_tickets WHERE requester_id=?", f.users["Carol"].ID) != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM mail_outbox WHERE kind='TICKET_REPLY'") != 1 || deletionCount(t, f, "SELECT COUNT(*) FROM audit_events_next WHERE action='support.create'") != 1 {
		t.Fatal("concurrent retry duplicated business records")
	}
}

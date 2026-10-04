//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
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

func mailSupportFixture(t *testing.T) *socialFixtureV2 {
	f := newSocialFixtureV2(t)
	f.app.Close()
	f.app.Config.SMTPKey = []byte(strings.Repeat("k", 32))
	if err := f.store.Grant(context.Background(), f.users["Alice"].ID, "platform.admin"); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Grant(context.Background(), f.users["Bob"].ID, "support.manage"); err != nil {
		t.Fatal(err)
	}
	for _, channel := range []string{"DM", "ENTERPRISE", "SYSTEM"} {
		f.request(t, "Alice", "PUT", "/api/v1/admin/mail/channels/"+channel, map[string]any{"clientRequestId": "channel-" + channel, "expectedVersion": 0, "settings": notify.Settings{Enabled: true, Host: "smtp.example.com", Port: 465, Security: "TLS", From: strings.ToLower(channel) + "@example.com", Recipients: []string{"operator@example.com"}}}, 200)
	}
	return f
}
func emailSignup(t *testing.T, f *socialFixtureV2, name, email string) store.User {
	t.Helper()
	v := store.EmailVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(identity.Secret())), CodeHash: store.Digest([]byte("123456")), Email: email, GameID: name, Purpose: "register", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewEmailVerification(context.Background(), v, "123456", f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	u, err := f.store.RegisterEmailUser(context.Background(), v, identity.Hash("test-password"))
	if err != nil {
		t.Fatal(err)
	}
	token := identity.Secret()
	if err = f.store.CreateSession(context.Background(), u, store.Digest([]byte(token)), "app", "", time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	f.users[name] = u
	f.tokens[name] = token
	return u
}
func TestMailIdentityGateAndBoundProof(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	original := socialDataV2(f.request(t, "Alice", "GET", "/api/v1/account/me", nil, 200))["user"].(map[string]any)
	if original["gameVerified"] != true {
		t.Fatal("existing account must be verified")
	}
	u := emailSignup(t, f, "NewPlayer", "new@example.com")
	me := socialDataV2(f.request(t, "NewPlayer", "GET", "/api/v1/account/me", nil, 200))["user"].(map[string]any)
	if me["gameVerified"] != false || me["gameId"] != "NewPlayer" {
		t.Fatal("pending account identity lost")
	}
	for _, path := range []string{"/api/v1/chat/messages", "/api/v1/wallet/balance", "/api/v1/admin/mail", "/api/v1/players/" + f.users["Alice"].PlayerRef} {
		r := f.request(t, "NewPlayer", "GET", path, nil, 403)
		if r["error"].(map[string]any)["code"] != "GAME_VERIFICATION_REQUIRED" {
			t.Fatalf("gate missing: %s", path)
		}
	}
	login := socialDataV2(f.request(t, "", "POST", "/api/v1/account/login", map[string]string{"account": "new@example.com", "password": "test-password"}, 200))
	if login["user"].(map[string]any)["userId"] != u.ID {
		t.Fatal("email login changed identity")
	}
	token := identity.Secret()
	code := "567890"
	v := store.GameVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(token)), CodeHash: store.Digest([]byte(token + ":" + code)), Purpose: "game_verify", PlayerUUID: "00000000-0000-4000-8000-000000000099", GameID: "NewPlayer", QQ: "99887766", UserID: u.ID, NodeID: "test", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewGameVerification(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ActivateGameVerification(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	other := emailSignup(t, f, "OtherPlayer", "other@example.com")
	if err := f.store.BindGameIdentity(ctx, other.ID, v); err == nil {
		t.Fatal("proof can cross accounts")
	}
	f.request(t, "NewPlayer", "POST", "/api/v1/account/game-verification", map[string]string{"verificationToken": token, "code": code}, 200)
	f.request(t, "NewPlayer", "GET", "/api/v1/chat/messages", nil, 200)
	after, err := f.store.UserByAlias(ctx, "new@example.com")
	if err != nil || !after.GameVerified || after.ID != u.ID || after.PlayerRef != u.PlayerRef || after.ServerUUID != v.PlayerUUID {
		t.Fatal("binding changed account or failed", err)
	}
	var admissions int
	if err := f.store.DB.QueryRow("SELECT COUNT(*) FROM admission_entries WHERE server_uuid=?", v.PlayerUUID).Scan(&admissions); err != nil {
		t.Fatal(err)
	}
	if admissions != 0 {
		t.Fatal("identity proof granted whitelist")
	}
}
func TestEmailVerificationLimitReplayAndConcurrentConsumption(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	token := identity.Secret()
	v := store.EmailVerification{ID: store.ID("verify_"), TokenHash: store.Digest([]byte(token)), CodeHash: store.Digest([]byte(token + ":123456")), Email: "concurrent@example.com", GameID: "Concurrent", Purpose: "register", ExpiresAt: time.Now().Add(time.Minute)}
	if err := f.store.NewEmailVerification(ctx, v, "123456", f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := f.store.RegisterEmailUser(ctx, v, "hash"); out <- err }()
	}
	wg.Wait()
	close(out)
	success := 0
	for err := range out {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("consumed %d times", success)
	}
	v.ID = store.ID("verify_")
	v.Email = "limit@example.com"
	v.GameID = "Limited"
	v.TokenHash = store.Digest([]byte(identity.Secret()))
	if err := f.store.NewEmailVerification(ctx, v, "123456", f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 5; n++ {
		if _, err := f.store.CheckEmailVerification(ctx, v.TokenHash, "incorrect", "register"); err == nil {
			t.Fatal("wrong code accepted")
		}
	}
	if _, err := f.store.CheckEmailVerification(ctx, v.TokenHash, v.CodeHash, "register"); err == nil {
		t.Fatal("attempt limit bypassed")
	}
	events, err := f.store.SystemMailEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if strings.Contains(e.Cipher, "123456") {
			t.Fatal("plaintext code persisted")
		}
	}
}
func supportTestInbox(t *testing.T, f *socialFixtureV2) store.SupportInbox {
	t.Helper()
	ctx := context.Background()
	password := "local-inbox-test-password"
	_, err := f.store.SaveSupportInbox(ctx, f.users["Alice"].ID, "inbox-setup", 0, notify.InboxSettings{Enabled: true, Host: "imap.example.com", Port: 993, Username: "support@example.com", Address: "support@example.com", Folder: "INBOX", ImportExisting: true}, &password, f.app.Config.SMTPKey)
	if err != nil {
		t.Fatal(err)
	}
	inbox, err := f.store.ClaimSupportInbox(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.SupportInboxPosition(ctx, inbox, 11, 1); err != nil {
		t.Fatal(err)
	}
	return inbox
}
func supportRawMail(sender, replyTo, id, subject, body, extra string) []byte {
	return []byte(fmt.Sprintf("From: %s\r\nReply-To: %s\r\nTo: support@example.com\r\nMessage-ID: <%s>\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n%s\r\n%s", sender, replyTo, id, mime.BEncoding.Encode("UTF-8", subject), extra, body))
}
func TestEmailSupportInboxTicketNumbersThreadingAndRoleIsolation(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	inbox := supportTestInbox(t, f)
	if err := f.store.Grant(ctx, f.users["Bob"].ID, "core.read"); err != nil {
		t.Fatal(err)
	} // support-only role cannot inherit other backstage powers
	raw := supportRawMail("player@example.com", "player@example.com", "first@example.com", "游戏验证遇到问题", "游戏内没有收到验证码，请协助检查。", "")
	created, err := f.store.IngestSupportMail(ctx, inbox, 11, 1, raw, time.Now(), f.app.Config.SMTPKey)
	if err != nil {
		t.Fatal(err)
	}
	ticket := created.TicketID
	if !regexp.MustCompile(`^[0-9]{12}$`).MatchString(ticket) || !strings.HasPrefix(ticket, time.Now().In(time.FixedZone("UTC+8", 8*3600)).Format("20060102")) {
		t.Fatal("incorrect ticket number", ticket)
	}
	again, err := f.store.IngestSupportMail(ctx, inbox, 11, 2, raw, time.Now(), f.app.Config.SMTPKey)
	if err != nil || again.Status != "DUPLICATE" || again.TicketID != ticket {
		t.Fatal("message-id dedupe failed", again, err)
	}
	f.request(t, "Carol", "GET", "/api/v1/support/tickets/"+ticket, nil, 404)
	f.request(t, "Carol", "GET", "/api/v1/admin/support/tickets", nil, 403)
	for _, path := range []string{"/api/v1/admin/accounts", "/api/v1/admin/mail", "/api/v1/admin/core/nodes", "/api/v1/admin/email-settings", "/api/v1/admin/whitelist/applications"} {
		f.request(t, "Bob", "GET", path, nil, 403)
	}
	note := map[string]any{"clientRequestId": "internal-note", "expectedVersion": 1, "body": "PRIVATE OPERATOR NOTE", "visibility": "INTERNAL"}
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets/"+ticket, note, 200)
	reply := map[string]any{"clientRequestId": "public-reply", "expectedVersion": 2, "body": "请重新进入游戏后再试一次。", "visibility": "PUBLIC"}
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets/"+ticket, reply, 200)
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets/"+ticket, reply, 200)
	events, err := f.store.SystemMailEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var outboundID string
	for _, e := range events {
		if e.Reference != ticket {
			continue
		}
		count++
		plain, err := notify.Open(f.app.Config.SMTPKey, e.Cipher)
		if err != nil || strings.Contains(plain, "PRIVATE OPERATOR NOTE") {
			t.Fatal("private note mailed", err)
		}
		if e.ReplyTo != "support@example.com" || e.InReplyTo != "first@example.com" {
			t.Fatal("thread reply address lost", e)
		}
		if e.Kind == "TICKET_CREATED" && e.Channel != "DM" || e.Kind == "TICKET_REPLY" && e.Channel != "ENTERPRISE" {
			t.Fatal("wrong outgoing channel")
		}
		if e.Kind == "TICKET_REPLY" {
			outboundID = e.EventID + "@deuterium-notifications"
		}
	}
	if count != 2 {
		t.Fatalf("expected receipt and one reply, got %d", count)
	}
	customerReply := supportRawMail("player@example.com", "player@example.com", "second@example.com", "回复问题", "我已经尝试，问题仍然存在。", "In-Reply-To: <"+outboundID+">\r\n")
	threaded, err := f.store.IngestSupportMail(ctx, inbox, 11, 3, customerReply, time.Now(), f.app.Config.SMTPKey)
	if err != nil || threaded.TicketID != ticket {
		t.Fatal("reply opened a duplicate ticket", err)
	}
	other := supportRawMail("outsider@example.com", "outsider@example.com", "third@example.com", "其他问题", "不能把陌生发件人的来信并入他人工单。", "In-Reply-To: <"+outboundID+">\r\n")
	isolated, err := f.store.IngestSupportMail(ctx, inbox, 11, 4, other, time.Now(), f.app.Config.SMTPKey)
	if err != nil || isolated.TicketID == ticket {
		t.Fatal("sender crossed ticket boundary", err)
	}
	auto := supportRawMail("auto@example.com", "auto@example.com", "auto@example.com", "自动回复", "我现在不在。", "Auto-Submitted: auto-replied\r\n")
	ignored, err := f.store.IngestSupportMail(ctx, inbox, 11, 5, auto, time.Now(), f.app.Config.SMTPKey)
	if err != nil || ignored.Status != "IGNORED" {
		t.Fatal("auto reply loop", err)
	}
	if _, err := f.store.DB.Exec("UPDATE identities SET display_name='NewSupportName' WHERE id=?", f.users["Bob"].ID); err != nil {
		t.Fatal(err)
	}
	sent := 0
	for n := 0; n < 5; n++ {
		err = f.app.processSystemMail(ctx, func(_ context.Context, c notify.Settings, _ string, _ string, m notify.Message) error {
			sent++
			if m.ReplyTo != "support@example.com" {
				t.Fatal("reply-to lost")
			}
			if c.From == "dm@example.com" && (!strings.Contains(m.HTML, ticket) && !strings.Contains(m.HTML, isolated.TicketID)) {
				t.Fatal("receipt template missing number")
			}
			if c.From == "enterprise@example.com" && (!strings.Contains(m.HTML, "Bob") || strings.Contains(m.HTML, "NewSupportName") || !strings.Contains(m.HTML, "请重新进入游戏后再试一次。") || !strings.Contains(m.HTML, "DEUTERIUM IX") || len(m.Images) != 0 || strings.Contains(m.HTML+m.Text, "PRIVATE OPERATOR NOTE")) {
				t.Fatal("human reply signature, footer or private-note isolation changed")
			}
			return nil
		})
		if err != nil {
			break
		}
	}
	if sent != 3 {
		t.Fatal("wrong outbound count", sent)
	}
	event := strings.TrimSuffix(outboundID, "@deuterium-notifications")
	if _, err = f.store.DB.Exec("UPDATE mail_outbox SET status='FAILED',attempts=8 WHERE event_id=?", event); err != nil {
		t.Fatal(err)
	}
	retry := map[string]string{"clientRequestId": "retry-email"}
	f.request(t, "Bob", "POST", "/api/v1/admin/mail/events/"+event+"/retry", retry, 403)
	f.request(t, "Bob", "POST", "/api/v1/admin/support/tickets/"+isolated.TicketID+"/mail/"+event+"/retry", retry, 404)
	path := "/api/v1/admin/support/tickets/" + ticket + "/mail/" + event + "/retry"
	f.request(t, "Bob", "POST", path, retry, 200)
	f.request(t, "Bob", "POST", path, retry, 200)
	if err = f.app.processSystemMail(ctx, func(_ context.Context, _ notify.Settings, _, id string, m notify.Message) error {
		if id != event || m.InReplyTo != "first@example.com" {
			t.Fatal("retry replaced stable mail reference")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

}

func TestManualEnterpriseMailAddsFooterWithoutInventingTicket(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	body := "您好，\n\n这是客服独立编写的回复正文。"
	if _, err := f.store.QueueManualMail(ctx, f.users["Alice"].ID, "manual-footer", "ENTERPRISE", "reader@example.com", "关于您的问题", body, f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	if err := f.app.processSystemMail(ctx, func(_ context.Context, c notify.Settings, _, _ string, m notify.Message) error {
		if c.From != "enterprise@example.com" || m.ReplyTo != c.From || m.Subject != "关于您的问题" || !strings.HasPrefix(m.Text, body) || !strings.Contains(m.HTML, "Alice") || !strings.Contains(m.HTML, "DEUTERIUM IX") || strings.Contains(m.Text, "工单编号：") || len(m.Images) != 0 {
			t.Fatal("manual correspondence lost its content or footer")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.QueueManualMail(ctx, f.users["Alice"].ID, "manual-dm", "DM", "reader@example.com", "系统通知", body, f.app.Config.SMTPKey); err != nil {
		t.Fatal(err)
	}
	if err := f.app.processSystemMail(ctx, func(_ context.Context, _ notify.Settings, _, _ string, m notify.Message) error {
		if m.Text != body || !strings.Contains(m.HTML, "deuterium-mail-fonts") || strings.Contains(m.HTML, "Alice") || m.ReplyTo != "" {
			t.Fatal("non-support channel received a human-reply footer")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInboxCutoffAttachmentsAndConcurrentTicketNumbers(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	inbox := supportTestInbox(t, f)
	cutoff := time.Now().Add(-time.Minute)
	inbox.AcceptAfter = &cutoff
	old := supportRawMail("old@example.com", "old@example.com", "old@example.com", "旧邮件", "保留在邮箱，不回执。", "")
	result, err := f.store.IngestSupportMail(ctx, inbox, 11, 1, old, cutoff.Add(-time.Hour), f.app.Config.SMTPKey)
	if err != nil || result.Status != "IGNORED" {
		t.Fatal("historical mail imported without opt-in", err)
	}
	raw := []byte("From: files@example.com\r\nMessage-ID: <attachment@example.com>\r\nSubject: Attachment\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=x\r\n\r\n--x\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nDetails\r\n--x\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=proof.txt\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--x--\r\n")
	result, err = f.store.IngestSupportMail(ctx, inbox, 11, 2, raw, time.Now(), f.app.Config.SMTPKey)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := f.store.SupportDetail(ctx, result.TicketID)
	if err != nil {
		t.Fatal(err)
	}
	messages := detail["messages"].([]store.SupportMessage)
	if len(messages) != 1 || len(messages[0].Attachments) != 1 {
		t.Fatal("attachment missing")
	}
	a := messages[0].Attachments[0]
	url := f.http.URL + "/api/v1/admin/support/tickets/" + result.TicketID + "/attachments/" + a.ID
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+f.tokens["Bob"])
	response, err := f.http.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || string(body) != "hello" || !strings.HasPrefix(response.Header.Get("Content-Disposition"), "attachment") || response.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("attachment delivery boundary", err)
	}
	f.request(t, "Carol", "GET", "/api/v1/admin/support/tickets/"+result.TicketID+"/attachments/"+a.ID, nil, 403)
	var wg sync.WaitGroup
	ids := make(chan string, 10)
	failures := make(chan error, 10)
	for n := 0; n < 10; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			raw := supportRawMail("same@example.com", "same@example.com", fmt.Sprintf("parallel-%d@example.com", n), "并发来信", "每封独立来信都需要唯一编号。", "")
			r, e := f.store.IngestSupportMail(ctx, inbox, 11, uint32(n+10), raw, time.Now(), f.app.Config.SMTPKey)
			if e != nil {
				failures <- e
			} else {
				ids <- r.TicketID
			}
		}(n)
	}
	wg.Wait()
	close(ids)
	close(failures)
	for e := range failures {
		t.Error(e)
	}
	seen := map[string]bool{}
	for id := range ids {
		if seen[id] || !regexp.MustCompile(`^[0-9]{12}$`).MatchString(id) {
			t.Fatal("duplicate or invalid number", id)
		}
		seen[id] = true
	}
	if len(seen) != 10 {
		t.Fatal("missing numbered tickets", len(seen))
	}
	// A same-sender subject reference remains useful when mail clients strip References.
	subject := supportRawMail("files@example.com", "files@example.com", "subject-only@example.com", "Re: [工单 "+result.TicketID+"] Attachment", "补充信息。", "")
	thread, err := f.store.IngestSupportMail(ctx, inbox, 11, 30, subject, time.Now(), f.app.Config.SMTPKey)
	if err != nil || thread.TicketID != result.TicketID {
		t.Fatal("subject fallback failed", err)
	}
}
func TestHelpSubmissionIsMailBeforeTicketAndUsesVerifiedReplyTo(t *testing.T) {
	f := mailSupportFixture(t)
	ctx := context.Background()
	inbox := supportTestInbox(t, f)
	emailSignup(t, f, "Pending", "pending@example.com")
	input := map[string]any{"clientRequestId": "help-one", "subject": "通过 Help 请求协助", "category": "ACCOUNT", "body": "这是结构化表单的完整说明。"}
	response := socialDataV2(f.request(t, "Pending", "POST", "/api/v1/support/submissions", input, 200))
	f.request(t, "Pending", "POST", "/api/v1/support/submissions", input, 200)
	if response["ticketId"] != nil || response["status"] != "QUEUED" {
		t.Fatal("Help directly created ticket")
	}
	var count int
	f.store.DB.QueryRow("SELECT COUNT(*) FROM support_tickets").Scan(&count)
	if count != 0 {
		t.Fatal("ticket created before receipt")
	}
	eventID := response["eventId"].(string)
	var cipher string
	if err := f.store.DB.QueryRow("SELECT headers_cipher FROM mail_outbox WHERE event_id=? AND channel_id='SYSTEM' AND recipient='support@example.com' AND reply_to='pending@example.com'", eventID).Scan(&cipher); err != nil {
		t.Fatal(err)
	}
	plain, err := notify.Open(f.app.Config.SMTPKey, cipher)
	if err != nil {
		t.Fatal(err)
	}
	var headers map[string]string
	if json.Unmarshal([]byte(plain), &headers) != nil {
		t.Fatal("missing signed submission headers")
	}
	extra := "X-Deuterium-Submission: " + headers["X-Deuterium-Submission"] + "\r\nX-Deuterium-Proof: " + headers["X-Deuterium-Proof"] + "\r\n"
	raw := supportRawMail("system@example.com", "pending@example.com", eventID+"@deuterium-notifications", "Help 请求", "表单转交。", extra)
	created, err := f.store.IngestSupportMail(ctx, inbox, 11, 1, raw, time.Now(), f.app.Config.SMTPKey)
	if err != nil {
		t.Fatal(err)
	}
	detail := socialDataV2(f.request(t, "Bob", "GET", "/api/v1/admin/support/tickets/"+created.TicketID, nil, 200))
	ticket := detail["ticket"].(map[string]any)
	if ticket["requesterEmail"] != "pending@example.com" || ticket["source"] != "HELP" {
		t.Fatal("system sender shown as customer", ticket)
	}
	forged := supportRawMail("system@example.com", "victim@example.com", "forged@example.com", "伪造转交", "不能伪造玩家身份。", extra)
	bad, err := f.store.IngestSupportMail(ctx, inbox, 11, 2, forged, time.Now(), f.app.Config.SMTPKey)
	if err != nil || bad.Status != "NEEDS_REVIEW" {
		t.Fatal("forged reply-to accepted", bad, err)
	}
}
func TestDisplayNamePreservesGameIdentity(t *testing.T) {
	f := mailSupportFixture(t)
	out := socialDataV2(f.request(t, "Alice", "PATCH", "/api/v1/account/me/profile", map[string]any{"clientRequestId": "nickname", "expectedVersion": 1, "displayName": "中文昵称", "bio": "你好"}, 200))
	if out["displayName"] != "中文昵称" || out["gameId"] != "Alice" {
		t.Fatal("nickname changed game ID")
	}
	other := socialDataV2(f.request(t, "Carol", "GET", "/api/v1/players/player_Alice", nil, 200))
	if other["displayName"] != "中文昵称" || other["qq"] != "10001" {
		t.Fatal("profile lost identity")
	}
}

//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func admissionEmailFixture(t *testing.T, enabled bool) *socialFixtureV2 {
	f := admissionFixture(t)
	f.app.Close()
	f.app.Config.AdmissionApplicationURL = f.http.URL + "/admission/"
	f.request(t, "Alice", "PUT", "/api/v1/admin/email-settings", map[string]any{"clientRequestId": "email-config", "expectedVersion": 0, "enabled": true, "host": "smtp.example.com", "port": 465, "security": "TLS", "username": "", "from": "sender@example.com", "recipients": []string{"admin@example.com"}, "admissionReviewEnabled": enabled}, 200)
	return f
}

func emailApplication(t *testing.T, f *socialFixtureV2, number int, qq string) store.AdmissionApplication {
	t.Helper()
	in := admissionInput()
	in.GameID, in.QQ = fmt.Sprintf("Player%d", number), qq
	a, err := f.store.SubmitAdmission(context.Background(), in, store.AdmissionProfile{UUID: fmt.Sprintf("00000000-0000-4000-8000-%012d", number), Name: in.GameID})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func reviewEmailApplication(t *testing.T, f *socialFixtureV2, a store.AdmissionApplication, decision, reason, key string) map[string]any {
	t.Helper()
	return f.request(t, "Alice", "POST", "/api/v1/admin/whitelist/applications/"+a.ID+"/review", map[string]any{"clientRequestId": key, "expectedVersion": a.Version, "decision": decision, "reason": reason, "qqMemberConfirmed": true, "identityConfirmed": true}, 200)
}

func TestAdmissionEmailAtomicReplayRecipientAndDurableRetry(t *testing.T) {
	f := admissionEmailFixture(t, true)
	a := emailApplication(t, f, 1, "123456789")
	first := reviewEmailApplication(t, f, a, "APPROVE", "PRIVATE ADMIN NOTE", "approve-one")
	again := reviewEmailApplication(t, f, a, "APPROVE", "PRIVATE ADMIN NOTE", "approve-one")
	if socialDataV2(first)["emailStatus"] != "PENDING" || socialDataV2(again)["version"] != socialDataV2(first)["version"] {
		t.Fatal("replay changed result")
	}
	var count int
	var payload string
	if err := f.store.DB.QueryRow(`SELECT COUNT(*),MAX(payload_json) FROM admin_email_outbox_v204 WHERE application_id=?`, a.ID).Scan(&count, &payload); err != nil || count != 1 || strings.Contains(payload, "PRIVATE ADMIN NOTE") {
		t.Fatal("duplicate task or leaked private approval note")
	}
	sends := 0
	var firstID string
	sender := func(_ context.Context, s notify.Settings, password, id string, m notify.Message) error {
		sends++
		if len(s.Recipients) != 1 || s.Recipients[0] != "123456789@qq.com" || !strings.Contains(m.HTML, a.GameID) || strings.Contains(m.HTML+m.Text, "PRIVATE ADMIN NOTE") || strings.Contains(m.HTML, "此邮件为模板测试") {
			t.Fatal("wrong recipient or formal message")
		}
		if sends == 1 {
			firstID = id
			return errors.New("transient failure")
		}
		if id != firstID {
			t.Fatal("retry changed message ID")
		}
		return nil
	}
	if f.app.processEmailV204(context.Background(), sender) == nil {
		t.Fatal("expected delivery retry")
	}
	access, err := f.store.AdmissionCheck(context.Background(), a.UUID)
	if err != nil || !access.Allowed {
		t.Fatal("mail failure rolled back admission")
	}
	if _, err = f.store.DB.Exec(`UPDATE admin_email_settings_v204 SET settings_json=JSON_SET(settings_json,'$.recipients',JSON_ARRAY('changed-admin@example.com')) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.DB.Exec(`UPDATE admin_email_outbox_v204 SET next_attempt_at=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE application_id=?`, a.ID); err != nil {
		t.Fatal(err)
	}
	if err = f.app.processEmailV204(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	f.app.processEmailV204(context.Background(), sender)
	if sends != 2 {
		t.Fatal("accepted mail sent again")
	}

	b := emailApplication(t, f, 2, "987654321")
	if _, err = f.store.DB.Exec(`INSERT INTO admin_email_outbox_v204(event_id,case_version,case_state,next_attempt_at,created_at) VALUES(?,0,'TEST',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, "admission:"+b.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.store.ReviewAdmission(context.Background(), f.users["Alice"].ID, b.ID, store.AdmissionDecision{ClientRequestID: "rollback-mail", ExpectedVersion: b.Version, Decision: "APPROVE", QQMemberConfirmed: true, IdentityConfirmed: true})
	if err == nil {
		t.Fatal("queue collision did not fail approval")
	}
	var state string
	f.store.DB.QueryRow(`SELECT status FROM admission_applications WHERE application_id=?`, b.ID).Scan(&state)
	access, _ = f.store.AdmissionCheck(context.Background(), b.UUID)
	if state != "PENDING" || access.Allowed {
		t.Fatal("approval and email were not atomic")
	}
}

func TestAdmissionRejectedEmailReasonAndApplicantIsolation(t *testing.T) {
	f := admissionEmailFixture(t, true)
	a := emailApplication(t, f, 3, "111112222")
	b := emailApplication(t, f, 4, "333334444")
	reason := "核对 <img src=x onerror=alert(1)> & 信息\n请重新提交。"
	reviewEmailApplication(t, f, a, "REJECT", reason, "reject-one")
	reviewEmailApplication(t, f, b, "REJECT", "另一位玩家的原因", "reject-two")
	seen := map[string]bool{}
	sender := func(_ context.Context, s notify.Settings, _, _ string, m notify.Message) error {
		if len(s.Recipients) != 1 {
			t.Fatal("multiple applicants on one email")
		}
		to := s.Recipients[0]
		seen[to] = true
		if to == "111112222@qq.com" {
			if !strings.Contains(m.Text, reason) || strings.Contains(m.HTML, "<img src=x") || !strings.Contains(m.HTML, "&lt;img") || strings.Contains(m.HTML, b.GameID) {
				t.Fatal("unsafe reason or cross-applicant data")
			}
		} else if to != "333334444@qq.com" || strings.Contains(m.Text, reason) {
			t.Fatal("wrong applicant recipient")
		}
		return nil
	}
	for range 2 {
		if err := f.app.processEmailV204(context.Background(), sender); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 2 {
		t.Fatal("missing applicant mail")
	}
}

func TestAdmissionEmailSwitchSkipPauseAndStaleCancellation(t *testing.T) {
	f := admissionEmailFixture(t, false)
	a := emailApplication(t, f, 5, "555556666")
	r := reviewEmailApplication(t, f, a, "REJECT", "待补充", "disabled-review")
	if socialDataV2(r)["emailStatus"] != "SKIPPED" {
		t.Fatal("disabled channel queued delivery")
	}
	status, _ := f.store.EmailStatusV204(context.Background())
	if status["pending"] != 0 {
		t.Fatal("skipped mail counted as pending")
	}
	setEnabled := func(enabled bool) {
		t.Helper()
		raw := "false"
		if enabled {
			raw = "true"
		}
		if _, err := f.store.DB.Exec(`UPDATE admin_email_settings_v204 SET settings_json=JSON_SET(settings_json,'$.admissionReviewEnabled',JSON_EXTRACT(?, '$')) WHERE id=1`, raw); err != nil {
			t.Fatal(err)
		}
	}
	setEnabled(true)
	b := emailApplication(t, f, 6, "666667777")
	reviewEmailApplication(t, f, b, "APPROVE", "", "enabled-review")
	setEnabled(false)
	sender := func(context.Context, notify.Settings, string, string, notify.Message) error {
		t.Fatal("paused, skipped or stale mail sent")
		return nil
	}
	f.app.processEmailV204(context.Background(), sender)
	var attempts int
	f.store.DB.QueryRow(`SELECT attempts FROM admin_email_outbox_v204 WHERE application_id=?`, b.ID).Scan(&attempts)
	if attempts != 0 {
		t.Fatal("paused mail was claimed")
	}
	_, err := f.store.RevokeAdmission(context.Background(), f.users["Alice"].ID, b.UUID, store.AdmissionRevoke{ClientRequestID: "revoke-before-mail", ExpectedVersion: 1, Reason: "撤销测试"})
	if err != nil {
		t.Fatal(err)
	}
	setEnabled(true)
	if err = f.app.processEmailV204(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	var state string
	f.store.DB.QueryRow(`SELECT status FROM admin_email_outbox_v204 WHERE application_id=?`, b.ID).Scan(&state)
	if state != "CANCELLED" {
		t.Fatal("revoked approval mail not cancelled")
	}
	status, _ = f.store.EmailStatusV204(context.Background())
	if status["pending"] != 0 {
		t.Fatal("cancelled/skipped mail counted as pending")
	}
}

func TestAdmissionTemplatePreviewTestsStayWithAdminRecipients(t *testing.T) {
	f := admissionEmailFixture(t, false)
	f.request(t, "Bob", "GET", "/api/v1/admin/email-settings/templates/APPROVED", nil, 403)
	preview := f.request(t, "Alice", "GET", "/api/v1/admin/email-settings/templates/REJECTED", nil, 200)
	if !strings.Contains(socialDataV2(preview)["html"].(string), "data:image/png;base64,") {
		t.Fatal("browser preview missing image")
	}
	for _, decision := range []string{"APPROVED", "REJECTED"} {
		body := map[string]any{"clientRequestId": "preview-" + decision, "template": decision}
		f.request(t, "Alice", "POST", "/api/v1/admin/email-settings/test", body, 200)
		f.request(t, "Alice", "POST", "/api/v1/admin/email-settings/test", body, 200)
	}
	f.request(t, "Alice", "POST", "/api/v1/admin/email-settings/test", map[string]any{"clientRequestId": "preview-again", "template": "APPROVED"}, 429)
	sends := 0
	sender := func(_ context.Context, s notify.Settings, _, _ string, m notify.Message) error {
		sends++
		if len(s.Recipients) != 1 || s.Recipients[0] != "admin@example.com" || !strings.Contains(m.Subject, "模板测试") || !strings.Contains(m.HTML, "此邮件为模板测试") {
			t.Fatal("test sent as a real applicant email")
		}
		return nil
	}
	for range 2 {
		if err := f.app.processEmailV204(context.Background(), sender); err != nil {
			t.Fatal(err)
		}
	}
	if sends != 2 {
		t.Fatal("template test replay duplicated")
	}
	raw, _ := json.Marshal(socialDataV2(preview))
	if strings.Contains(string(raw), "passwordCipher") {
		t.Fatal("secret in preview")
	}
}

func TestAdmissionEmailCancelsRejectionSupersededByNewApplicationOrManualGrant(t *testing.T) {
	f := admissionEmailFixture(t, true)
	a := emailApplication(t, f, 7, "777778888")
	reviewEmailApplication(t, f, a, "REJECT", "旧申请资料有误", "old-rejection")
	in := admissionInput()
	in.GameID, in.QQ = a.GameID, a.QQ
	if _, err := f.store.SubmitAdmission(context.Background(), in, store.AdmissionProfile{UUID: a.UUID, Name: a.GameID}); err != nil {
		t.Fatal(err)
	}
	b := emailApplication(t, f, 8, "888889999")
	reviewEmailApplication(t, f, b, "REJECT", "旧申请未通过", "manual-rejection")
	_, err := f.store.AddAdmission(context.Background(), f.users["Alice"].ID, store.AdmissionManualAdd{ClientRequestID: "manual-grant", GameID: b.GameID, ExpectedUUID: b.UUID, QQ: b.QQ, Reason: "已另行核实", QQMemberConfirmed: true, IdentityConfirmed: true}, store.AdmissionProfile{UUID: b.UUID, Name: b.GameID})
	if err != nil {
		t.Fatal(err)
	}
	sender := func(context.Context, notify.Settings, string, string, notify.Message) error {
		t.Fatal("outdated rejection sent")
		return nil
	}
	for range 2 {
		if err := f.app.processEmailV204(context.Background(), sender); err != nil {
			t.Fatal(err)
		}
	}
	var total, cancelled int
	if err := f.store.DB.QueryRow(`SELECT COUNT(*),SUM(status='CANCELLED') FROM admin_email_outbox_v204`).Scan(&total, &cancelled); err != nil || total != 2 || cancelled != 2 {
		t.Fatal("old notifications were not cancelled or manual grant created mail")
	}
}

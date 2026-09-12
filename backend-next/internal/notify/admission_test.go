package notify

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestAdmissionTemplatesEscapeDetailsAndSeparatePreview(t *testing.T) {
	input := AdmissionMail{TemplateVersion: AdmissionTemplateVersion, GameID: "NewPlayer", Decision: "REJECTED", Reason: "请核对 <script>alert(1)</script> & 玩家资料。\n第二行", ApplicationURL: "https://example.com/admission/"}
	m, err := RenderAdmission(input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") || !strings.Contains(m.HTML, "&lt;script&gt;") || strings.Contains(m.HTML, "xiwanzi") || !strings.Contains(m.Text, input.Reason) {
		t.Fatal("dynamic reason or player was not rendered safely")
	}
	if strings.Contains(m.HTML, "此邮件为模板测试") || strings.Contains(m.HTML, "示例原因") || strings.Contains(m.Subject, "测试") {
		t.Fatal("formal rejection contains preview copy")
	}
	input.Preview = true
	preview, err := RenderAdmission(input)
	if err != nil || !strings.Contains(preview.HTML, "此邮件为模板测试") || !strings.Contains(preview.Subject, "模板测试") || !strings.Contains(AdmissionPreviewHTML(preview), "data:image/png;base64,") {
		t.Fatal("preview was not clearly marked")
	}
	input.Preview = false
	input.Decision = "APPROVED"
	input.Reason = "PRIVATE ADMIN NOTE"
	approved, err := RenderAdmission(input)
	if err != nil || strings.Contains(approved.HTML+approved.Text, input.Reason) || !strings.Contains(approved.HTML, "RECONSTRUCTION PROGRAM") || !strings.Contains(approved.HTML, "Segoe UI") {
		t.Fatal("approved copy or identity styling changed")
	}
	input.ApplicationURL = "javascript:alert(1)"
	if _, err := RenderAdmission(input); err == nil {
		t.Fatal("unsafe application URL accepted")
	}
}

func TestAdmissionMIMEIncludesTextHTMLAndCIDImage(t *testing.T) {
	m, err := RenderAdmission(AdmissionMail{TemplateVersion: AdmissionTemplateVersion, GameID: "RecipientOne", Decision: "APPROVED", ApplicationURL: "https://example.com/admission/"})
	if err != nil {
		t.Fatal(err)
	}
	s := testSettings()
	s.Recipients = []string{"123456789@qq.com"}
	raw, err := composeMessage(s, "stable-request-id", m)
	if err != nil {
		t.Fatal(err)
	}
	email, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if email.Header.Get("To") != s.Recipients[0] || email.Header.Get("Message-ID") != "<stable-request-id@deuterium-notifications>" {
		t.Fatal("wrong delivery headers")
	}
	_, params, _ := mime.ParseMediaType(email.Header.Get("Content-Type"))
	reader := multipart.NewReader(email.Body, params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	kind, params, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if kind != "multipart/alternative" {
		t.Fatal("missing text alternative")
	}
	alt := multipart.NewReader(part, params["boundary"])
	for _, want := range []struct{ kind, content string }{{"text/plain", m.Text}, {"text/html", m.HTML}} {
		p, err := alt.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		kind, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, p))
		if err != nil || kind != want.kind || string(decoded) != want.content {
			t.Fatalf("broken %s alternative", want.kind)
		}
	}
	img, err := reader.NextPart()
	if err != nil || img.Header.Get("Content-ID") != "<deuterium-ix-emblem>" {
		t.Fatal("CID image missing")
	}
	data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, img))
	if err != nil || !bytes.Equal(data, m.Images[0].Data) {
		t.Fatal("CID image changed")
	}
	if _, err = reader.NextPart(); err != io.EOF {
		t.Fatal("unexpected attachment")
	}
	m.Subject = "title\r\nBcc: stranger@example.com"
	if _, err = composeMessage(s, "stable-id", m); err == nil {
		t.Fatal("header injection accepted")
	}
}

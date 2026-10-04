package notify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportReplyPreservesFreeBodyAndAddsReplyableFooter(t *testing.T) {
	in := SupportReplyMail{Subject: "[工单 202610040427] 游戏登录问题", Body: "您好，\n\n第一段：<script>alert(1)</script>\n1. 检查信息；\n2. 回复结果。", AuthorName: "客服 & <测试>", TicketNumber: "202610040427", ReplyAddress: SupportEmail}
	m, err := RenderSupportReply(in)
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != in.Subject || !strings.HasPrefix(m.Text, "工单编号："+in.TicketNumber+"\n\n"+in.Body+"\n\n") || m.ReplyTo != in.ReplyAddress || strings.Contains(m.HTML, "<script>") || !strings.Contains(m.HTML, "&lt;script&gt;") || !strings.Contains(m.HTML, "客服 &amp; &lt;测试&gt;") {
		t.Fatal("reply body, author or delivery details changed")
	}
	for _, value := range []string{in.TicketNumber, "直接回复本邮件", "© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED.", HelpCenterURL, SupportEmail} {
		if !strings.Contains(m.HTML, value) || !strings.Contains(m.Text, value) {
			t.Fatalf("missing footer detail %q", value)
		}
	}
	if strings.Contains(m.HTML+m.Text, "请勿回复") || strings.Contains(m.HTML+m.Text, "待审查") || len(m.Headers) != 0 {
		t.Fatal("human reply presented as an automated notice")
	}
	m.InReplyTo = "customer@example.com"
	m.References = []string{m.InReplyTo}
	raw, err := composeMessage(testSettings(), "staff-reply", m)
	if err != nil || len(m.Images) != 0 || strings.Contains(string(raw), "Content-ID:") || !strings.Contains(string(raw), "In-Reply-To: <customer@example.com>") || !strings.Contains(string(raw), "Reply-To: "+SupportEmail) {
		t.Fatal("reply MIME or threading invalid", err)
	}
	if strings.Contains(AdmissionPreviewHTML(m), "cid:") {
		t.Fatal("logo missing from preview")
	}
	in.TicketNumber = ""
	in.AuthorName = ""
	general, err := RenderSupportReply(in)
	if err != nil || strings.Contains(general.Text, "工单编号：") {
		t.Fatal("general correspondence invented a ticket", err)
	}
	in.ReplyAddress = "support@example.com\r\nBcc: other@example.com"
	if _, err := RenderSupportReply(in); err == nil {
		t.Fatal("invalid reply address accepted")
	}
}

func TestSupportReplyPreview(t *testing.T) {
	dir := os.Getenv("DEUTERIUM_MAIL_PREVIEW_DIR")
	if dir == "" {
		t.Skip("preview export not requested")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"support-reply":      "您好，\n\n感谢您提供截图。我们已查看您反馈的登录问题，需要再确认以下信息：\n\n1. 出现提示的大致时间；\n2. 当前使用的启动器版本；\n3. 是否能正常进入其他子服。\n\n您可以直接回复这封邮件补充信息，我们会根据这些情况继续排查。",
		"support-reply-long": strings.Repeat("https://example.invalid/details/", 30),
	} {
		m, err := RenderSupportReply(SupportReplyMail{Subject: "[工单 202610040427] 游戏登录遇到问题", Body: body, AuthorName: "xiwanzi", TicketNumber: "202610040427", ReplyAddress: SupportEmail})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name+".html"), []byte(AdmissionPreviewHTML(m)), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

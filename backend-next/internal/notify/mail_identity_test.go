package notify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllMailUsesOfficialTypographyAndEmbeddedBrand(t *testing.T) {
	messages := map[string]Message{}
	for _, purpose := range []string{"register", "password_reset", "password_change"} {
		m, err := RenderVerification(VerificationMail{TemplateVersion: VerificationTemplateVersion, Purpose: purpose, GameID: "xiwanzi", Code: "001234"})
		if err != nil {
			t.Fatal(err)
		}
		messages["verification-"+purpose] = m
	}
	for _, decision := range []string{"APPROVED", "REJECTED"} {
		m, err := RenderAdmission(AdmissionMail{TemplateVersion: AdmissionTemplateVersion, GameID: "xiwanzi", Decision: decision, Reason: "请核对正版游戏 ID 后重新提交申请。", ApplicationURL: "https://apply.deuteriumix.com"})
		if err != nil {
			t.Fatal(err)
		}
		messages["admission-"+strings.ToLower(decision)] = m
	}
	messages["support-received"] = RenderSupportAcknowledgement("202610040427", "游戏登录遇到问题", "进入服务器时提示连接中断，重新登录后仍未恢复。\n已附上提示截图，希望协助确认。")
	reply, err := RenderSupportReply(SupportReplyMail{Subject: "[工单 202610040427] 游戏登录遇到问题", Body: "您好，\n\n感谢您提供截图。我们已查看您反馈的登录问题，需要再确认以下信息：\n\n1. 出现提示的大致时间；\n2. 当前使用的启动器版本；\n3. 是否能正常进入其他子服。\n\n您可以直接回复这封邮件补充信息，我们会根据这些情况继续排查。", AuthorName: "xiwanzi", TicketNumber: "202610040427", ReplyAddress: SupportEmail})
	if err != nil {
		t.Fatal(err)
	}
	messages["support-reply"] = reply
	plain := Message{Subject: "Deuterium 平台介入通知", Text: "案件已有新的处理进展。\n请登录管理页查看编号 202610040427 的记录。\n<script>此处按文字显示</script>", ReplyTo: SupportEmail, InReplyTo: "original@example.com", Headers: map[string]string{"X-Auto-Response-Suppress": "All"}}
	system, err := RenderSystemMessage(plain)
	if err != nil || system.Text != plain.Text || system.ReplyTo != plain.ReplyTo || system.InReplyTo != plain.InReplyTo || system.Headers["X-Auto-Response-Suppress"] != "All" || strings.Contains(system.HTML, "<script>") {
		t.Fatal("system mail metadata or escaping changed", err)
	}
	messages["system-notice"] = system
	for name, m := range messages {
		t.Run(name, func(t *testing.T) {
			for _, s := range []string{"HarmonyOS Sans SC", "Novecentosanswide", "unicode-range:U+0030-0039", "deuterium-mail-fonts", "cid:deuterium-ix-wordmark-v1"} {
				if !strings.Contains(m.HTML, s) {
					t.Fatalf("missing official font or brand: %s", s)
				}
			}
			if len(m.Images) != 2 || len(m.Images[1].Data) == 0 {
				t.Fatal("brand assets missing")
			}
			wrapped, err := RenderSystemMessage(m)
			if err != nil || wrapped.HTML != m.HTML {
				t.Fatal("existing template wrapped twice")
			}
			if dir := os.Getenv("DEUTERIUM_MAIL_PREVIEW_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				preview := m
				if name == "system-notice" {
					preview, err = RenderSystemMessage(Message{Subject: plain.Subject, Text: "案件已有新的处理进展。\n请登录管理页查看编号 202610040427 的记录。"})
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(AdmissionPreviewHTML(preview)), 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

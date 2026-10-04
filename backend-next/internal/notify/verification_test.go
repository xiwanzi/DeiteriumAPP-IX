package notify

import (
	"bytes"
	"github.com/emersion/go-message/mail"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerificationTemplatesHaveDistinctIntentCodeAndBrand(t *testing.T) {
	for _, test := range []struct{ purpose, word, instruction string }{{"register", "注册", "游戏 ID 填写有误"}, {"password_reset", "找回", "您的密码不会因此改变"}, {"password_change", "更改", "检查账号安全"}} {
		t.Run(test.purpose, func(t *testing.T) {
			m, err := RenderVerification(VerificationMail{TemplateVersion: VerificationTemplateVersion, Purpose: test.purpose, GameID: "xiwanzi", Code: "482691"})
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range []string{"您的 Deuterium ID 验证码", "482691", "xiwanzi", test.word, test.instruction, "© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED.", SupportEmail, HelpCenterURL, "10 分钟"} {
				if !strings.Contains(m.HTML, value) || !strings.Contains(m.Text, value) {
					t.Fatalf("missing %q", value)
				}
			}
			if len(m.Images) != 2 || m.Images[0].ContentType != "image/png" || !strings.Contains(m.HTML, "cid:"+m.Images[0].CID) {
				t.Fatal("brand image not embedded")
			}
			raw, err := composeMessage(testSettings(), "verification-test", m)
			if err != nil {
				t.Fatal(err)
			}
			r, err := mail.CreateReader(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			parts := 0
			for {
				p, e := r.NextPart()
				if e == io.EOF {
					break
				}
				if e != nil {
					t.Fatal(e)
				}
				data, e := io.ReadAll(p.Body)
				if e != nil || len(data) == 0 {
					t.Fatal("empty MIME part", e)
				}
				parts++
			}
			if parts < 3 {
				t.Fatal("missing plain text, HTML or logo MIME part")
			}
			if dir := os.Getenv("DEUTERIUM_MAIL_PREVIEW_DIR"); dir != "" {
				if err = os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, "verification-"+test.purpose+".html"), []byte(AdmissionPreviewHTML(m)), 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestVerificationTemplateEscapesIdentityAndRejectsInvalidCodes(t *testing.T) {
	m, err := RenderVerification(VerificationMail{TemplateVersion: VerificationTemplateVersion, Purpose: "register", GameID: "<script>alert(1)</script>", Code: "001234"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") || !strings.Contains(m.HTML, "001234") {
		t.Fatal("identity not escaped or leading zero lost")
	}
	for _, code := range []string{"12345", "1234567", "１２３４５６", "<script>"} {
		if _, err = RenderVerification(VerificationMail{TemplateVersion: VerificationTemplateVersion, Purpose: "register", GameID: "Player", Code: code}); err == nil {
			t.Fatal("invalid code accepted")
		}
	}
	if _, err = RenderVerification(VerificationMail{TemplateVersion: VerificationTemplateVersion, Purpose: "unknown", GameID: "Player", Code: "123456"}); err == nil {
		t.Fatal("unknown purpose accepted")
	}
}

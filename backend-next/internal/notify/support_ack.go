package notify

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

var supportAcknowledgement = template.Must(template.ParseFS(admissionFiles, "templates/support-received-v1.html"))

func RenderSupportAcknowledgement(number, subject, _ string) Message {
	subject = strings.TrimPrefix(subject, "工单已收到 · ")
	data := struct{ Number string }{number}
	var html bytes.Buffer
	_ = supportAcknowledgement.ExecuteTemplate(&html, "support-received-v1.html", data)
	return Message{
		Subject:  "[工单 " + number + "] 已收到：" + subject,
		FromName: "Deuterium IX 客服",
		HTML:     mailHTML(html.String()),
		Text:     fmt.Sprintf("工单已收到\n\n工单编号：%s\n\n您好，\n感谢您联系 Deuterium IX 玩家支持。我们已收到您的来信。\n\n后续进展将通过邮件通知您。如需补充信息，请直接回复本邮件，并保留原邮件主题。\n\n感谢您的耐心与理解。\nDeuterium IX 玩家支持\n\n帮助中心：%s\n客服邮箱：%s\n© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED.", number, HelpCenterURL, SupportEmail),
		Headers:  map[string]string{"Auto-Submitted": "auto-replied", "X-Auto-Response-Suppress": "All"},
	}
}

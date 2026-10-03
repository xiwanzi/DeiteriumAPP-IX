package notify

import (
	"bytes"
	"html/template"
	"strings"
)

var supportAcknowledgement = template.Must(template.New("support-received").Parse(`<!doctype html><html><head><meta name="color-scheme" content="light"><meta name="supported-color-schemes" content="light"></head><body style="margin:0;background:#f3f5f8;color:#202631;font-family:Arial,'Microsoft YaHei',sans-serif"><table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:40px 16px"><table role="presentation" width="560" style="max-width:100%;background:#fff;border-radius:20px;border:1px solid #e4e8ef" cellpadding="0" cellspacing="0"><tr><td style="padding:36px"><p style="font-size:12px;color:#2677ec;letter-spacing:2px">DEUTERIUM IX · SUPPORT</p><h1 style="font-size:27px;line-height:1.5;margin:25px 0 12px">你的来信，我们收到了。</h1><p style="font-size:15px;line-height:1.9;color:#586375">我们已收到你关于「{{.Subject}}」的邮件，正在等待支持人员审查。</p><div style="padding:22px;background:#f1f6ff;border-radius:12px;margin:26px 0"><p style="font-size:12px;color:#66758a;margin:0 0 8px">你的工单编号</p><strong style="font-size:27px;letter-spacing:2px;color:#2677ec">{{.Number}}</strong></div><p style="font-size:13px;color:#738094">问题摘要</p><p style="font-size:14px;line-height:1.9;white-space:pre-wrap;word-break:break-word">{{.Summary}}</p><hr style="border:0;border-top:1px solid #e9edf3;margin:28px 0"><p style="font-size:14px;line-height:1.9;color:#586375">后续进展会通过邮件与你联系。如需补充信息，请直接回复这封邮件，保留邮件主题即可。</p><p style="font-size:12px;color:#929baa;margin-top:24px">Deuterium IX 客服团队</p></td></tr></table></td></tr></table></body></html>`))

func RenderSupportAcknowledgement(number, subject, summary string) Message {
	subject = strings.TrimPrefix(subject, "工单已收到 · ")
	data := struct{ Number, Subject, Summary string }{number, subject, clipText(summary, 500)}
	var html bytes.Buffer
	_ = supportAcknowledgement.Execute(&html, data)
	return Message{Subject: "[工单 " + number + "] 已收到：" + subject, Text: "我们已收到你关于「" + subject + "」的邮件，正在等待支持人员审查。\n\n工单编号：" + number + "\n\n问题摘要：\n" + data.Summary + "\n\n后续进展会通过邮件与你联系。如需补充信息，请直接回复这封邮件并保留主题。\n\nDeuterium IX 客服团队", HTML: html.String(), FromName: "Deuterium IX 客服", Headers: map[string]string{"Auto-Submitted": "auto-replied", "X-Auto-Response-Suppress": "All"}}
}

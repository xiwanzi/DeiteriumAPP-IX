package notify

import (
	"bytes"
	"errors"
	"html/template"
	"net/mail"
	"strings"
)

type SupportReplyMail struct {
	Subject, Body, AuthorName, TicketNumber, ReplyAddress string
}

var supportReplyTemplate = template.Must(template.ParseFS(admissionFiles, "templates/support-reply-v1.html"))

func RenderSupportReply(in SupportReplyMail) (Message, error) {
	address, err := mail.ParseAddress(in.ReplyAddress)
	if strings.TrimSpace(in.Body) == "" || err != nil || address.Address != in.ReplyAddress || strings.ContainsAny(in.ReplyAddress, "\r\n") {
		return Message{}, errors.New("人工回复邮件内容或回复地址无效。")
	}
	in.Body = strings.ReplaceAll(strings.ReplaceAll(in.Body, "\r\n", "\n"), "\r", "\n")
	in.AuthorName = strings.TrimSpace(in.AuthorName)
	var html bytes.Buffer
	if err := supportReplyTemplate.ExecuteTemplate(&html, "support-reply-v1.html", in); err != nil {
		return Message{}, err
	}
	logo, err := admissionFiles.ReadFile("templates/deuterium-ix-emblem.png")
	if err != nil {
		return Message{}, err
	}
	text := ""
	if in.TicketNumber != "" {
		text = "工单编号：" + in.TicketNumber + "\n\n"
	}
	text += in.Body + "\n\n"
	if in.AuthorName != "" {
		text += in.AuthorName + "\n"
	}
	text += "Deuterium IX 玩家支持\n\nDEUTERIUM IX\n"
	text += "您可以直接回复本邮件继续沟通。\n帮助中心：" + HelpCenterURL + "\n客服邮箱：" + in.ReplyAddress + "\n© 2026 DEUTERIUM IX. ALL RIGHTS RESERVED."
	return Message{Subject: in.Subject, Text: text, HTML: html.String(), FromName: "Deuterium IX 玩家支持", ReplyTo: in.ReplyAddress,
		Images: []InlineImage{{CID: "deuterium-ix-emblem-small-v1", Filename: "deuterium-ix.png", ContentType: "image/png", Data: logo}}}, nil
}

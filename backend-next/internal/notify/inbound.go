package notify

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
)

const MaxInboundBytes = 8 << 20

type Attachment struct {
	Name, ContentType string
	Data              []byte
}
type Inbound struct {
	MessageID, From, FromName, ReplyTo, Subject, Text, SubmissionID, Proof string
	References                                                             []string
	Automated                                                              bool
	Attachments                                                            []Attachment
}

func ReadInbound(raw []byte) (out Inbound, err error) {
	if len(raw) > MaxInboundBytes {
		return out, errors.New("邮件超过 8 MiB 接收上限，请在邮箱中查看。")
	}
	r, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return out, errors.New("邮件格式无法解析，请在邮箱中查看。")
	}
	if r == nil {
		return out, errors.New("邮件正文不可读。")
	}
	defer r.Close()
	from, err := r.Header.AddressList("From")
	if err != nil || len(from) != 1 {
		return out, errors.New("发件邮箱格式无效。")
	}
	out.From = strings.ToLower(from[0].Address)
	out.FromName = cleanMailLabel(from[0].Name, 100)
	if reply, err := r.Header.AddressList("Reply-To"); err == nil && len(reply) == 1 {
		out.ReplyTo = strings.ToLower(reply[0].Address)
	}
	if out.ReplyTo == "" {
		out.ReplyTo = out.From
	}
	out.MessageID, _ = r.Header.MessageID()
	if !ValidMessageID(out.MessageID) {
		out.MessageID = ""
	}
	out.Subject, _ = r.Header.Subject()
	out.Subject = cleanMailLabel(out.Subject, 120)
	if out.Subject == "" {
		out.Subject = "无主题邮件"
	}
	for _, header := range []string{"References", "In-Reply-To"} {
		refs, _ := r.Header.MsgIDList(header)
		for _, ref := range refs {
			if ValidMessageID(ref) {
				out.References = append(out.References, ref)
			}
		}
	}
	if len(out.References) > 20 {
		out.References = out.References[len(out.References)-20:]
	}
	auto := strings.ToLower(strings.TrimSpace(r.Header.Get("Auto-Submitted")))
	precedence := strings.ToLower(strings.TrimSpace(r.Header.Get("Precedence")))
	contentType, _, _ := r.Header.ContentType()
	out.Automated = (auto != "" && auto != "no") || precedence == "bulk" || precedence == "list" || precedence == "junk" || strings.HasPrefix(contentType, "multipart/report") || strings.HasPrefix(strings.ToLower(out.From), "mailer-daemon@")
	out.SubmissionID = r.Header.Get("X-Deuterium-Submission")
	out.Proof = r.Header.Get("X-Deuterium-Proof")
	var plain, rich []string
	for count := 0; ; count++ {
		if count > 100 {
			return out, errors.New("邮件正文分段过多。")
		}
		part, e := r.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil && !message.IsUnknownCharset(e) {
			return out, errors.New("邮件附件或正文无法解析。")
		}
		if part == nil {
			break
		}
		data, e := io.ReadAll(io.LimitReader(part.Body, MaxInboundBytes+1))
		if e != nil || len(data) > MaxInboundBytes {
			return out, errors.New("邮件正文或附件超过限制。")
		}
		switch header := part.Header.(type) {
		case *mail.InlineHeader:
			kind, _, _ := header.ContentType()
			if kind == "text/plain" {
				plain = append(plain, string(data))
			} else if kind == "text/html" {
				rich = append(rich, htmlText(data))
			}
		case *mail.AttachmentHeader:
			if len(out.Attachments) >= 10 {
				return out, errors.New("邮件附件超过 10 个，请在邮箱中查看。")
			}
			name, _ := header.Filename()
			name = strings.ReplaceAll(name, "\\", "/")
			name = filepath.Base(name)
			name = cleanMailLabel(name, 180)
			if name == "" || name == "." || name == "/" {
				name = "附件"
			}
			kind, _, _ := header.ContentType()
			if kind == "" {
				kind = mime.TypeByExtension(filepath.Ext(name))
			}
			if kind == "" {
				kind = "application/octet-stream"
			}
			out.Attachments = append(out.Attachments, Attachment{Name: name, ContentType: kind, Data: data})
		}
	}
	if len(plain) > 0 {
		out.Text = strings.Join(plain, "\n\n")
	} else {
		out.Text = strings.Join(rich, "\n\n")
	}
	out.Text = clipText(strings.TrimSpace(out.Text), 12000)
	if out.Text == "" {
		out.Text = "（此邮件没有可读取的正文，请查看附件。）"
	}
	return out, nil
}
func clipText(value string, limit int) string {
	value = strings.ToValidUTF8(value, "�")
	r := []rune(value)
	if len(r) > limit {
		return string(r[:limit-1]) + "…"
	}
	return value
}
func cleanMailLabel(value string, limit int) string {
	return clipText(strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)), " "), limit)
}
func htmlText(data []byte) string {
	z := html.NewTokenizer(bytes.NewReader(data))
	var b strings.Builder
	skip := 0
	for {
		kind := z.Next()
		switch kind {
		case html.ErrorToken:
			return strings.TrimSpace(b.String())
		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if token.Data == "script" || token.Data == "style" {
				if kind == html.StartTagToken {
					skip++
				} else if skip > 0 {
					skip--
				}
			}
			if skip == 0 && (token.Data == "p" || token.Data == "div" || token.Data == "br" || token.Data == "li") {
				b.WriteByte('\n')
			}
		case html.TextToken:
			if skip == 0 {
				b.Write(z.Text())
			}
		}
		if b.Len() > 60000 || !utf8.ValidString(b.String()) {
			return clipText(b.String(), 12000)
		}
	}
}

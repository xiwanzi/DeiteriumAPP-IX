package notify

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type InlineImage struct {
	CID, Filename, ContentType string
	Data                       []byte
}

type Message struct {
	Subject, Text, HTML, FromName string
	Images                        []InlineImage
	ReplyTo, InReplyTo            string
	References                    []string
	Headers                       map[string]string
}

var messageToken = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

func writeEncoded(w io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 0 {
		n := min(76, len(encoded))
		if _, err := io.WriteString(w, encoded[:n]+"\r\n"); err != nil {
			return err
		}
		encoded = encoded[n:]
	}
	return nil
}

func composeMessage(s Settings, id string, m Message) ([]byte, error) {
	if !messageToken.MatchString(id) || strings.IndexFunc(m.Subject+m.FromName, unicode.IsControl) >= 0 {
		return nil, errors.New("邮件标题或编号格式无效。")
	}
	from := s.From
	if m.FromName != "" {
		from = (&mail.Address{Name: m.FromName, Address: s.From}).String()
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@deuterium-notifications>\r\nMIME-Version: 1.0\r\n", from, strings.Join(s.Recipients, ", "), mime.BEncoding.Encode("UTF-8", m.Subject), time.Now().UTC().Format(time.RFC1123Z), id)
	if m.ReplyTo != "" {
		a, err := mail.ParseAddress(m.ReplyTo)
		if err != nil || a.Address != m.ReplyTo || strings.ContainsAny(m.ReplyTo, "\r\n") {
			return nil, errors.New("回复邮箱格式无效。")
		}
		fmt.Fprintf(&b, "Reply-To: %s\r\n", m.ReplyTo)
	}
	if m.InReplyTo != "" {
		if !ValidMessageID(m.InReplyTo) {
			return nil, errors.New("邮件引用编号无效。")
		}
		fmt.Fprintf(&b, "In-Reply-To: <%s>\r\n", m.InReplyTo)
	}
	if len(m.References) > 20 {
		return nil, errors.New("邮件引用过多。")
	}
	if len(m.References) > 0 {
		b.WriteString("References:")
		for _, ref := range m.References {
			if !ValidMessageID(ref) {
				return nil, errors.New("邮件引用编号无效。")
			}
			fmt.Fprintf(&b, " <%s>", ref)
		}
		b.WriteString("\r\n")
	}
	for name, value := range m.Headers {
		if (name != "X-Deuterium-Submission" && name != "X-Deuterium-Proof" && name != "Auto-Submitted" && name != "X-Auto-Response-Suppress") || strings.ContainsAny(value, "\r\n") || len(value) > 200 {
			return nil, errors.New("邮件标记无效。")
		}
		fmt.Fprintf(&b, "%s: %s\r\n", name, value)
	}
	if m.HTML == "" {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
		if err := writeEncoded(&b, []byte(m.Text)); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	var alternative bytes.Buffer
	alt := multipart.NewWriter(&alternative)
	for _, part := range []struct{ kind, body string }{{"text/plain", m.Text}, {"text/html", m.HTML}} {
		w, err := alt.CreatePart(textproto.MIMEHeader{"Content-Type": {part.kind + "; charset=UTF-8"}, "Content-Transfer-Encoding": {"base64"}})
		if err != nil {
			return nil, err
		}
		if err = writeEncoded(w, []byte(part.body)); err != nil {
			return nil, err
		}
	}
	if err := alt.Close(); err != nil {
		return nil, err
	}
	if len(m.Images) == 0 {
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", alt.Boundary())
		b.Write(alternative.Bytes())
		return b.Bytes(), nil
	}
	related := multipart.NewWriter(&b)
	fmt.Fprintf(&b, "Content-Type: multipart/related; boundary=%q\r\n\r\n", related.Boundary())
	w, err := related.CreatePart(textproto.MIMEHeader{"Content-Type": {fmt.Sprintf("multipart/alternative; boundary=%q", alt.Boundary())}})
	if err != nil {
		return nil, err
	}
	if _, err = w.Write(alternative.Bytes()); err != nil {
		return nil, err
	}
	for _, img := range m.Images {
		if !messageToken.MatchString(img.CID) || strings.IndexFunc(img.Filename, unicode.IsControl) >= 0 || img.ContentType != "image/png" {
			return nil, errors.New("邮件内嵌图片格式无效。")
		}
		w, err = related.CreatePart(textproto.MIMEHeader{"Content-Type": {img.ContentType}, "Content-ID": {"<" + img.CID + ">"}, "Content-Disposition": {mime.FormatMediaType("inline", map[string]string{"filename": img.Filename})}, "Content-Transfer-Encoding": {"base64"}})
		if err != nil {
			return nil, err
		}
		if err = writeEncoded(w, img.Data); err != nil {
			return nil, err
		}
	}
	if err = related.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func ValidMessageID(value string) bool {
	return len(value) > 2 && len(value) <= 998 && strings.Contains(value, "@") && strings.IndexFunc(value, func(r rune) bool { return r <= 32 || r >= 127 || strings.ContainsRune("<>", r) }) < 0
}

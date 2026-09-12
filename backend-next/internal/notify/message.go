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
	if m.HTML == "" {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
		if err := writeEncoded(&b, []byte(m.Text)); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	related := multipart.NewWriter(&b)
	fmt.Fprintf(&b, "Content-Type: multipart/related; boundary=%q\r\n\r\n", related.Boundary())
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

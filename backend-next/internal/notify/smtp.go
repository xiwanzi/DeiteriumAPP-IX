package notify

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Settings struct {
	Enabled                bool     `json:"enabled"`
	Host                   string   `json:"host"`
	Port                   int      `json:"port"`
	Security               string   `json:"security"`
	Username               string   `json:"username"`
	From                   string   `json:"from"`
	Recipients             []string `json:"recipients"`
	Version                int64    `json:"version"`
	PasswordConfigured     bool     `json:"passwordConfigured"`
	AdmissionReviewEnabled bool     `json:"admissionReviewEnabled"`
}

func (s Settings) Validate() error {
	invalid := errors.New("请填写有效的 SMTP 主机、端口、发件邮箱与接收邮箱，并选择 TLS 或 STARTTLS。")
	if len(s.Host) > 253 || s.Host == "" || s.Port < 1 || s.Port > 65535 || (s.Security != "TLS" && s.Security != "STARTTLS") || len(s.Username) > 320 || len(s.Recipients) < 1 || len(s.Recipients) > 10 {
		return invalid
	}
	for _, v := range []string{s.Host, s.Username, s.From} {
		if strings.TrimSpace(v) != v || strings.IndexFunc(v, unicode.IsControl) >= 0 {
			return invalid
		}
	}
	if net.ParseIP(s.Host) == nil {
		for _, label := range strings.Split(s.Host, ".") {
			if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return invalid
			}
			for _, r := range label {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
					return invalid
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, v := range append([]string{s.From}, s.Recipients...) {
		a, err := mail.ParseAddress(v)
		if err != nil || a.Address != v || len(v) > 254 || strings.IndexFunc(v, func(r rune) bool { return r > 127 || unicode.IsControl(r) }) >= 0 {
			return invalid
		}
	}
	for _, v := range s.Recipients {
		key := strings.ToLower(v)
		if seen[key] {
			return invalid
		}
		seen[key] = true
	}
	return nil
}

func Seal(key []byte, password string) (string, error) {
	if len(key) != 32 {
		return "", errors.New("邮件密钥尚未配置，请先设置服务器 DEUTERIUM_SMTP_KEY。")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(password), []byte("deuterium-smtp-v204"))), nil
}
func Open(key []byte, secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	fail := errors.New("邮件密码无法解密，请检查服务器邮件密钥或重新保存密码。")
	if len(key) != 32 {
		return "", fail
	}
	raw, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fail
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fail
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(raw) < aead.NonceSize() {
		return "", fail
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("deuterium-smtp-v204"))
	if err != nil {
		return "", fail
	}
	return string(plain), nil
}

// Errors deliberately omit remote SMTP replies: providers can echo credentials
// or addresses. The caller persists only these short diagnostic messages.
func Send(ctx context.Context, s Settings, password, messageID, subject, body string) error {
	return sendWithRoots(ctx, s, password, messageID, subject, body, nil)
}
func sendWithRoots(ctx context.Context, s Settings, password, messageID, subject, body string, roots *x509.CertPool) error {
	return sendMessageWithRoots(ctx, s, password, messageID, Message{Subject: subject, Text: body}, roots)
}
func SendMessage(ctx context.Context, s Settings, password, messageID string, message Message) error {
	return sendMessageWithRoots(ctx, s, password, messageID, message, nil)
}
func sendMessageWithRoots(ctx context.Context, s Settings, password, messageID string, message Message, roots *x509.CertPool) error {
	if err := s.Validate(); err != nil {
		return err
	}
	encoded, err := composeMessage(s, messageID, message)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	address := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	dialer := net.Dialer{Timeout: 8 * time.Second}
	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return errors.New("SMTP 连接失败，请检查主机、端口和网络。")
	}
	defer raw.Close()
	deadline, _ := ctx.Deadline()
	_ = raw.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { raw.Close() })
	defer stop()
	tlsConfig := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12, RootCAs: roots}
	var conn net.Conn = raw
	if s.Security == "TLS" {
		secure := tls.Client(raw, tlsConfig)
		if err = secure.HandshakeContext(ctx); err != nil {
			return errors.New("SMTP TLS 握手失败，请检查证书与加密方式。")
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return errors.New("SMTP 服务响应无效。")
	}
	defer client.Close()
	if s.Security == "STARTTLS" {
		if err = client.StartTLS(tlsConfig); err != nil {
			return errors.New("SMTP STARTTLS 失败，请检查服务端是否支持加密连接。")
		}
	}
	if s.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.Username, password, s.Host)); err != nil {
			return errors.New("SMTP 认证失败，请检查用户名及授权码。")
		}
	}
	if err = client.Mail(s.From); err != nil {
		return errors.New("SMTP 拒绝发件邮箱。")
	}
	for _, recipient := range s.Recipients {
		if err = client.Rcpt(recipient); err != nil {
			return errors.New("SMTP 拒绝接收邮箱，请检查配置。")
		}
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("SMTP 暂时无法接收邮件。")
	}
	if _, err = writer.Write(encoded); err != nil {
		return errors.New("SMTP 发送中断，稍后使用同一邮件编号重试。")
	}
	if err = writer.Close(); err != nil {
		return errors.New("SMTP 未确认接收邮件，稍后使用同一邮件编号重试。")
	}
	// DATA's final success is the delivery acceptance. A lost QUIT must not cause a duplicate.
	_ = client.Quit()
	return nil
}

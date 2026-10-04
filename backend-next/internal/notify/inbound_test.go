package notify

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestInboundMimeReplyHeadersHTMLAndAttachments(t *testing.T) {
	raw := []byte("From: Player <player@example.com>\r\nReply-To: reply@example.com\r\nMessage-ID: <original@example.com>\r\nReferences: <older@example.com>\r\nIn-Reply-To: <latest@example.com>\r\nSubject: =?UTF-8?B?5ri45oiP6Zeu6aKY?=\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=sample\r\n\r\n--sample\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<p>游戏问题</p><script>DO_NOT_EXECUTE()</script><p>详细情况</p>\r\n--sample\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=\"../../proof.txt\"\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--sample--\r\n")
	m, err := ReadInbound(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "游戏问题" || m.From != "player@example.com" || m.ReplyTo != "reply@example.com" || m.MessageID != "original@example.com" || len(m.References) != 2 {
		t.Fatal("mail headers lost", m)
	}
	if strings.Contains(m.Text, "DO_NOT_EXECUTE") || !strings.Contains(m.Text, "详细情况") || len(m.Attachments) != 1 || m.Attachments[0].Name != "proof.txt" || string(m.Attachments[0].Data) != "hello" {
		t.Fatal("mime parsing failed", m)
	}
	auto, err := ReadInbound([]byte("From: bot@example.com\r\nAuto-Submitted: auto-replied\r\n\r\nreply"))
	if err != nil || !auto.Automated {
		t.Fatal("auto reply detection failed", err)
	}
}
func TestTicketReceiptEscapingReplyToAndThreadHeaders(t *testing.T) {
	m := RenderSupportAcknowledgement("202610030042", "<script>alert(1)</script>", "<img src=x>正文")
	m.ReplyTo = "support@example.com"
	m.InReplyTo = "inbound@example.com"
	m.References = []string{m.InReplyTo}
	data, err := composeMessage(testSettings(), "receipt_123", m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") || strings.Contains(m.HTML, "<img src=x>") || !strings.Contains(m.HTML, "202610030042") {
		t.Fatal("unescaped receipt template")
	}
	for _, header := range []string{"Reply-To: support@example.com", "In-Reply-To: <inbound@example.com>", "References: <inbound@example.com>", "Auto-Submitted: auto-replied"} {
		if !strings.Contains(string(data), header) {
			t.Fatal("missing header", header)
		}
	}
	if len(m.Images) != 2 || len(m.Images[0].Data) == 0 || len(m.Images[0].Data) > 40*1024 || !strings.Contains(string(data), "Content-ID: <"+m.Images[0].CID+">") {
		t.Fatal("receipt logo must travel with the email")
	}
	preview := AdmissionPreviewHTML(m)
	if strings.Contains(preview, "cid:") || !strings.Contains(preview, "data:image/png;base64,") || !strings.Contains(m.Text, "直接回复本邮件") {
		t.Fatal("receipt preview or reply instructions missing")
	}
	if dir := os.Getenv("DEUTERIUM_MAIL_PREVIEW_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for name, message := range map[string]Message{
			"support-received":      RenderSupportAcknowledgement("202610040427", "游戏登录遇到问题", "进入服务器时提示连接中断，重新登录后仍未恢复。\n已附上提示截图，希望协助确认。"),
			"support-received-long": RenderSupportAcknowledgement("202610040427", strings.Repeat("LongPlayerSubject", 7), strings.Repeat("https://example.invalid/long-unbroken-details/", 18)),
		} {
			if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(AdmissionPreviewHTML(message)), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	m.ReplyTo = "victim@example.com\r\nBcc: other@example.com"
	if _, err = composeMessage(testSettings(), "receipt_123", m); err == nil {
		t.Fatal("header injection accepted")
	}
}
func TestInboxTLSUsesUIDsReadOnlyAndPeek(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	certTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "loopback-imap"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	raw := "From: player@example.com\r\nMessage-ID: <wire@example.com>\r\nSubject: Test\r\n\r\nHello"
	commands := make(chan []string, 1)
	go func() {
		var seen []string
		defer func() { commands <- seen }()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(8 * time.Second))
		fmt.Fprint(conn, "* OK [CAPABILITY IMAP4rev1] ready\r\n")
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			seen = append(seen, line)
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return
			}
			tag := parts[0]
			switch {
			case strings.Contains(line, " LOGIN "):
				fmt.Fprintf(conn, "%s OK [CAPABILITY IMAP4rev1] logged in\r\n", tag)
			case strings.Contains(line, " CAPABILITY"):
				fmt.Fprintf(conn, "* CAPABILITY IMAP4rev1\r\n%s OK done\r\n", tag)
			case strings.Contains(line, " EXAMINE "):
				fmt.Fprintf(conn, "* FLAGS (\\Seen)\r\n* 1 EXISTS\r\n* OK [UIDVALIDITY 11] valid\r\n* OK [UIDNEXT 42] next\r\n%s OK [READ-ONLY] examined\r\n", tag)
			case strings.Contains(line, " UID SEARCH "):
				fmt.Fprintf(conn, "* SEARCH 41\r\n%s OK searched\r\n", tag)
			case strings.Contains(line, " UID FETCH "):
				fmt.Fprintf(conn, "* 1 FETCH (UID 41 INTERNALDATE \"03-Oct-2026 12:00:00 +0000\" RFC822.SIZE %d BODY[]<0> {%d}\r\n%s)\r\n%s OK fetched\r\n", len(raw), len(raw), raw, tag)
			default:
				fmt.Fprintf(conn, "%s BAD unexpected command\r\n", tag)
			}
		}
	}()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	p, _ := strconv.Atoi(port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got := 0
	err = pollInboxWithRoots(ctx, InboxSettings{Host: "127.0.0.1", Port: p, Username: "support@example.com", Address: "support@example.com", Folder: "INBOX"}, "test", func(valid, next uint32) (uint32, error) {
		if valid != 11 || next != 42 {
			t.Error("bad IMAP UID metadata")
		}
		return 40, nil
	}, func(uid uint32, data []byte, at time.Time) error {
		got++
		if uid != 41 || string(data) != raw || at.IsZero() {
			t.Error("invalid IMAP body or UID")
		}
		return nil
	}, roots)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatal("mail not received")
	}
	seen := <-commands
	joined := strings.Join(seen, "")
	if !strings.Contains(joined, "EXAMINE") || !strings.Contains(joined, "BODY.PEEK[]<0.") || strings.Contains(joined, " STORE ") || strings.Contains(joined, " EXPUNGE") {
		t.Fatal("mailbox flags mutated", joined)
	}
}

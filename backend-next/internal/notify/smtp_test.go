package notify

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func testSettings() Settings {
	return Settings{Enabled: true, Host: "smtp.example.com", Port: 587, Security: "STARTTLS", Username: "admin@example.com", From: "admin@example.com", Recipients: []string{"inbox@example.com"}}
}

func TestSMTPTLSAndSTARTTLSSendUTF8AndAcceptCommittedDataBeforeQuit(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "loopback-smtp"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	for _, security := range []string{"TLS", "STARTTLS"} {
		t.Run(security, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			received := make(chan string, 1)
			failures := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					failures <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				serverTLS := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12}
				if security == "TLS" {
					conn = tls.Server(conn, serverTLS)
				}
				reader := bufio.NewReader(conn)
				fmt.Fprint(conn, "220 loopback SMTP\r\n")
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						failures <- err
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"):
						fmt.Fprint(conn, "250-loopback\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
					case strings.HasPrefix(line, "STARTTLS"):
						fmt.Fprint(conn, "220 start encryption\r\n")
						conn = tls.Server(conn, serverTLS)
						reader = bufio.NewReader(conn)
					case strings.HasPrefix(line, "AUTH PLAIN "):
						encoded := strings.TrimSpace(strings.TrimPrefix(line, "AUTH PLAIN "))
						decoded, _ := base64.StdEncoding.DecodeString(encoded)
						if !strings.Contains(string(decoded), "test-password") {
							failures <- fmt.Errorf("wrong auth")
							return
						}
						fmt.Fprint(conn, "235 accepted\r\n")
					case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
						fmt.Fprint(conn, "250 accepted\r\n")
					case line == "DATA\r\n":
						fmt.Fprint(conn, "354 send body\r\n")
						var message strings.Builder
						for {
							line, err = reader.ReadString('\n')
							if err != nil {
								failures <- err
								return
							}
							if line == ".\r\n" {
								break
							}
							message.WriteString(line)
						}
						fmt.Fprint(conn, "250 stored\r\n")
						received <- message.String()
						return // Drop QUIT after committed DATA.
					default:
						failures <- fmt.Errorf("unexpected command")
						return
					}
				}
			}()
			s := testSettings()
			s.Host = "127.0.0.1"
			s.Port = listener.Addr().(*net.TCPAddr).Port
			s.Security = security
			if err := sendWithRoots(context.Background(), s, "test-password", "same-message", "平台介入更新", "案件状态已更新", roots); err != nil {
				t.Fatal(err)
			}
			select {
			case message := <-received:
				if !strings.Contains(message, "Message-ID: <same-message@deuterium-notifications>") || !strings.Contains(message, base64.StdEncoding.EncodeToString([]byte("案件状态已更新"))) {
					t.Fatal("MIME message missing expected content")
				}
			case err := <-failures:
				t.Fatal(err)
			case <-time.After(time.Second):
				t.Fatal("no accepted message")
			}
		})
	}
}
func TestSecretEncryptionRequiresCorrectKeyAndDetectsTampering(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	secret, err := Seal(key, "smtp-app-password")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(secret, "smtp-app-password") {
		t.Fatal("plaintext persisted")
	}
	plain, err := Open(key, secret)
	if err != nil || plain != "smtp-app-password" {
		t.Fatal("cannot restore secret")
	}
	for _, key := range [][]byte{nil, bytes.Repeat([]byte{8}, 32)} {
		if _, err := Open(key, secret); err == nil {
			t.Fatal("accepted invalid key")
		}
	}
	if _, err := Open(key, secret[:len(secret)-5]); err == nil {
		t.Fatal("accepted truncated ciphertext")
	}
}
func TestSMTPSettingsRejectHeaderInjectionAndUnencryptedMode(t *testing.T) {
	if err := testSettings().Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Settings){func(s *Settings) { s.Host = "https://smtp.example.com" }, func(s *Settings) { s.From = "sender@example.com\r\nBcc: victim@example.com" }, func(s *Settings) { s.Security = "NONE" }, func(s *Settings) { s.Recipients = []string{"invalid"} }, func(s *Settings) { s.Recipients = []string{"a@example.com", "A@example.com"} }} {
		s := testSettings()
		mutate(&s)
		if s.Validate() == nil {
			t.Fatal("accepted invalid settings")
		}
	}
}
func TestSMTPRefusesToSendCredentialsWithoutSTARTTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	commands := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		fmt.Fprint(conn, "220 test SMTP\r\n")
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		fmt.Fprint(conn, "250 test SMTP\r\n")
		next, _ := reader.ReadString('\n')
		commands <- line + next
	}()
	s := testSettings()
	s.Host = "127.0.0.1"
	s.Port = listener.Addr().(*net.TCPAddr).Port
	if err := Send(context.Background(), s, "do-not-leak", "test-message", "test", "test"); err == nil {
		t.Fatal("accepted plaintext SMTP")
	}
	select {
	case sent := <-commands:
		if strings.Contains(sent, "AUTH") || strings.Contains(sent, "do-not-leak") {
			t.Fatal("credentials leaked")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("SMTP did not disconnect")
	}
}

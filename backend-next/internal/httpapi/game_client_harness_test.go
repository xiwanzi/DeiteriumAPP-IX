//go:build integration

package httpapi

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Opt-in real-game integration harness: real Go handlers and random local DB,
// with an isolated HTTPS S3 emulator. It never reads production configuration.
func TestGameClientHarness(t *testing.T) {
	if os.Getenv("DEUTERIUM_GAME_CLIENT_QA") != "1" {
		t.Skip("opt-in game harness")
	}
	out := os.Getenv("DEUTERIUM_GAME_CLIENT_QA_OUTPUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	objects := map[string][]byte{}
	puts := 0
	storage := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case "PUT":
			if _, ok := objects[key]; ok && r.Header.Get("If-None-Match") == "*" {
				w.WriteHeader(412)
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, (2<<20)+1))
			if err != nil || len(data) > 2<<20 {
				w.WriteHeader(400)
				return
			}
			sum := md5.Sum(data)
			if r.Header.Get("Content-Md5") != base64.StdEncoding.EncodeToString(sum[:]) || r.Header.Get("Content-Type") != "image/webp" {
				w.WriteHeader(400)
				return
			}
			objects[key] = data
			puts++
			w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:])+`"`)
			w.WriteHeader(200)
		case "GET", "HEAD":
			data, ok := objects[key]
			if !ok {
				w.WriteHeader(404)
				return
			}
			sum := md5.Sum(data)
			w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:])+`"`)
			w.Header().Set("Content-Type", "image/webp")
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			if r.Method == "GET" {
				_, _ = w.Write(data)
			}
		case "DELETE":
			delete(objects, key)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer storage.Close()
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: storage.TLS.Certificates[0].Certificate[0]})
	if err := os.WriteFile(filepath.Join(out, "storage-ca.pem"), cert, 0600); err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	previous := http.DefaultTransport
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
	http.DefaultTransport = transport
	defer func() { http.DefaultTransport = previous; transport.CloseIdleConnections() }()
	for key, value := range map[string]string{"DEUTERIUM_S3_ENDPOINT": storage.URL, "DEUTERIUM_S3_REGION": "game-test", "DEUTERIUM_S3_BUCKET": "game-chat-test", "DEUTERIUM_S3_PREFIX": "isolated-game/", "DEUTERIUM_S3_ACCESS_KEY": "local-game-key", "DEUTERIUM_S3_SECRET_KEY": "local-game-secret"} {
		t.Setenv(key, value)
	}
	f := newSocialFixtureV2(t)
	ctx := context.Background()
	offlineUUID := func(name string) string {
		sum := md5.Sum([]byte("OfflinePlayer:" + name))
		sum[6] = (sum[6] & 15) | 0x30
		sum[8] = (sum[8] & 63) | 0x80
		h := hex.EncodeToString(sum[:])
		return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
	}
	for old, name := range map[string]string{"Alice": "Dev", "Bob": "ChatNear", "Carol": "ChatFar"} {
		u := f.users[old]
		u.GameID = name
		u.ServerUUID = offlineUUID(name)
		u.GameVerified = true
		if _, err := f.store.DB.Exec(`UPDATE identities SET game_id=?,server_uuid=? WHERE id=?`, name, u.ServerUUID, u.ID); err != nil {
			t.Fatal(err)
		}
		f.users[old] = u
	}
	nodeToken := "local-game-node-01234567890123456789012345"
	f.app.Config.Nodes = []config.Node{{ID: "amiya", Chat: true, TokenSHA256: store.Digest([]byte(nodeToken)), ItemPrefix: "deuterium", InventoryDomain: "survival", CompatibilityProfile: "ix-main-1_21_1"}, {ID: "odyssey", Chat: true, TokenSHA256: store.Digest([]byte(nodeToken)), ItemPrefix: "deuterium", InventoryDomain: "survival", CompatibilityProfile: "ix-main-1_21_1"}}
	done := make(chan struct{})
	var once sync.Once
	finish := identity.Secret()
	normal := f.app.Handler()
	f.http.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__qa__/finish" && r.Method == "POST" && r.Header.Get("X-QA-Finish") == finish {
			once.Do(func() { close(done) })
			w.WriteHeader(204)
			return
		}
		if r.URL.Path == "/__qa__/status" {
			var count int
			_ = f.store.DB.QueryRow(`SELECT COUNT(*) FROM chat_messages_next`).Scan(&count)
			mu.Lock()
			n, p := len(objects), puts
			mu.Unlock()
			writeJSON(w, 200, map[string]any{"publicMessages": count, "objects": n, "puts": p, "amiyaOnline": f.app.Core.Online("amiya"), "odysseyOnline": f.app.Core.Online("odyssey")})
			return
		}
		normal.ServeHTTP(w, r)
	})
	state := map[string]any{"apiBase": f.http.URL, "bridge": "ws" + strings.TrimPrefix(f.http.URL, "http") + "/bridge/v1/connect", "nodeToken": nodeToken, "finish": finish, "tokens": f.tokens, "users": f.users, "storage": storage.URL}
	raw, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "harness.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("isolated game harness ready")
	select {
	case <-done:
	case <-time.After(40 * time.Minute):
		t.Fatal("harness timeout")
	}
	var count int
	if err := f.store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages_next`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	summary := map[string]any{"publicMessages": count, "objects": len(objects), "puts": puts}
	mu.Unlock()
	raw, _ = json.MarshalIndent(summary, "", "  ")
	_ = os.WriteFile(filepath.Join(out, "summary.json"), raw, 0600)
}

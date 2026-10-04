//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/notify"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Loopback-only, opt-in fixture. SMTP is captured in process; production builds
// contain neither these routes nor the synthetic identities.
func TestMailSupportBrowserFixture(t *testing.T) {
	file := os.Getenv("DEUTERIUM_MAIL_BROWSER_FIXTURE")
	if file == "" {
		t.Skip("interactive fixture not requested")
	}
	f := mailSupportFixture(t)
	f.http.Close()
	ctx := context.Background()
	emailSignup(t, f, "Maya", "maya@example.com")
	web, _ := filepath.Abs("../../../web-app/dist")
	api := f.app.Handler()
	static := http.FileServer(http.Dir(web))
	inbox := supportTestInbox(t, f)
	for n, subject := range []string{"游戏内验证码没有收到", "订单领取后背包中没有物品", "想确认白名单审核进度"} {
		result, err := f.store.IngestSupportMail(ctx, inbox, 11, uint32(n+1), supportRawMail("Carol <player@example.com>", "player@example.com", fmt.Sprintf("seed-%d@example.com", n), subject, "你好，今天登录时遇到了这个问题。我已经重新连接服务器，但问题仍然存在，麻烦帮忙查看一下。", ""), time.Now(), f.app.Config.SMTPKey)
		if err != nil {
			t.Fatal(err)
		}
		v := struct{ ID string }{result.TicketID}
		if _, err = f.store.ChangeSupport(ctx, f.users["Bob"].ID, v.ID, store.SupportChange{ClientRequestID: store.ID("request_"), ExpectedVersion: 1, Body: "已收到你的问题。我们正在核查记录，稍后会在这里同步进展。", Visibility: "PUBLIC", Status: "IN_PROGRESS"}, f.app.Config.SMTPKey); err != nil {
			t.Fatal(err)
		}
	}
	f.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/fixture-login":
			name := r.URL.Query().Get("user")
			if name == "" {
				name = "Alice"
			}
			u, ok := f.users[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			token, csrf := identity.Secret(), identity.Secret()
			if err := f.store.CreateSession(ctx, u, store.Digest([]byte(token)), "web", csrf, time.Now().Add(time.Hour), ""); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			f.app.setCookie(w, token, time.Now().Add(time.Hour))
			http.Redirect(w, r, "/admin?section=support", http.StatusSeeOther)
		case r.URL.Path == "/fixture-deliver" && r.Method == "POST":
			messages := []notify.Message{}
			for n := 0; n < 20; n++ {
				if err := f.app.processSystemMail(ctx, func(_ context.Context, _ notify.Settings, _, _ string, m notify.Message) error {
					messages = append(messages, m)
					return nil
				}); err != nil {
					break
				}
			}
			json.NewEncoder(w).Encode(messages)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			api.ServeHTTP(w, r)
		case r.URL.Path == "/web-config.json":
			json.NewEncoder(w).Encode(map[string]any{"mode": "connected", "apiAvailable": true})
		case strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/media/"):
			static.ServeHTTP(w, r)
		default:
			http.ServeFile(w, r, filepath.Join(web, "index.html"))
		}
	}))
	f.app.Config.PublicOrigin = f.http.URL
	data, _ := json.Marshal(map[string]string{"url": f.http.URL})
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(file + ".stop"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}

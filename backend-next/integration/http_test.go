//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/httpapi"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/testdb"
)

type fixture struct {
	store  *store.Store
	app    *httpapi.Server
	http   *httptest.Server
	tokens map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testdb.New(t)
	imported(t, db)
	tokens := map[string]string{}
	nodes := []config.Node{}
	for _, name := range []string{"login", "amiya", "odyssey", "mek"} {
		token := identity.Secret()
		tokens[name] = token
		n := config.Node{ID: name, TokenSHA256: store.Digest([]byte(token)), Chat: true, InventoryDomain: "survival"}
		if name == "amiya" {
			n.ItemPrefix = "deuterium:"
		}
		nodes = append(nodes, n)
	}
	cfg := config.Config{Listen: "127.0.0.1:8080", PublicOrigin: "http://127.0.0.1:8080", Development: true, Nodes: nodes}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	app := httpapi.New(db, cfg)
	server := httptest.NewServer(app.Handler())
	app.Config.PublicOrigin = server.URL
	t.Cleanup(func() { app.Close(); server.Close() })
	return &fixture{store: db, app: app, http: server, tokens: tokens}
}

func (f *fixture) request(t *testing.T, method, path string, payload any, headers http.Header, cookie *http.Cookie) (*http.Response, map[string]any) {
	t.Helper()
	var data []byte
	if payload != nil {
		var err error
		data, err = json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	r, err := http.NewRequest(method, f.http.URL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	r.Header = headers.Clone()
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if payload != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	response, err := f.http.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body map[string]any
	if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return response, body
}
func bearer(token string) http.Header {
	return http.Header{"Authorization": []string{"Bearer " + token}}
}
func (f *fixture) appToken(t *testing.T) string {
	t.Helper()
	response, body := f.request(t, "POST", "/api/v1/account/login", map[string]string{"account": "Alice", "password": password}, nil, nil)
	if response.StatusCode != 200 {
		t.Fatalf("login failed: %v", body)
	}
	return body["data"].(map[string]any)["token"].(string)
}

func TestBrowserSessionOriginCSRFAndAdminIsolation(t *testing.T) {
	f := newFixture(t)
	login := map[string]string{"account": "Alice", "password": password}
	bad := http.Header{"Origin": []string{"https://attacker.example"}}
	response, _ := f.request(t, "POST", "/api/v1/web/session", login, bad, nil)
	if response.StatusCode != 403 || len(response.Cookies()) > 0 {
		t.Fatal("cross-origin login accepted")
	}
	good := http.Header{"Origin": []string{f.http.URL}}
	response, _ = f.request(t, "POST", "/api/v1/account/login", login, good, nil)
	if response.StatusCode != 400 {
		t.Fatal("browser received app token")
	}
	response, body := f.request(t, "POST", "/api/v1/web/session", login, good, nil)
	if response.StatusCode != 200 {
		t.Fatal(body)
	}
	data := body["data"].(map[string]any)
	if _, ok := data["token"]; ok {
		t.Fatal("web response exposed bearer token")
	}
	csrf := data["csrfToken"].(string)
	cookies := response.Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Path != "/" || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("invalid web session cookie")
	}
	cookie := cookies[0]
	response, _ = f.request(t, "GET", "/api/v1/account/me", nil, nil, cookie)
	if response.StatusCode != 200 {
		t.Fatal("web session cannot read common account API")
	}
	response, _ = f.request(t, "GET", "/api/v1/account/me", nil, bearer(cookie.Value), nil)
	if response.StatusCode != 401 {
		t.Fatal("web token used as app token")
	}
	response, _ = f.request(t, "GET", "/api/v1/admin/core/nodes", nil, nil, cookie)
	if response.StatusCode != 403 {
		t.Fatal("migrated player has implicit admin privileges")
	}
	if err := f.store.Grant(context.Background(), "legacy_alice", "core.read"); err != nil {
		t.Fatal(err)
	}
	response, body = f.request(t, "GET", "/api/v1/admin/core/nodes", nil, nil, cookie)
	if response.StatusCode != 200 {
		t.Fatal("explicit permission ineffective")
	}
	encoded, _ := json.Marshal(body)
	if bytes.Contains(encoded, []byte("tokenSha256")) || bytes.Contains(encoded, []byte(f.tokens["amiya"])) {
		t.Fatal("node credential leaked through management API")
	}
	response, _ = f.request(t, "DELETE", "/api/v1/web/session", nil, good, cookie)
	if response.StatusCode != 403 {
		t.Fatal("mutation without CSRF accepted")
	}
	bad.Set("X-CSRF-Token", csrf)
	response, _ = f.request(t, "DELETE", "/api/v1/web/session", nil, bad, cookie)
	if response.StatusCode != 403 {
		t.Fatal("cross-origin mutation accepted")
	}
	good.Set("X-CSRF-Token", csrf)
	response, _ = f.request(t, "DELETE", "/api/v1/web/session", nil, good, cookie)
	if response.StatusCode != 200 {
		t.Fatal("valid logout failed")
	}
	response, _ = f.request(t, "GET", "/api/v1/account/me", nil, nil, cookie)
	if response.StatusCode != 401 {
		t.Fatal("logout did not revoke session")
	}
}

func TestProductionCookieAndInputBounds(t *testing.T) {
	f := newFixture(t)
	f.app.Config.Development = false
	f.app.Config.PublicOrigin = "https://community.example.invalid"
	response, body := f.request(t, "POST", "/api/v1/web/session", map[string]string{"account": "Alice", "password": password}, http.Header{"Origin": []string{f.app.Config.PublicOrigin}}, nil)
	if response.StatusCode != 200 {
		t.Fatal(body)
	}
	cookie := response.Cookies()[0]
	if cookie.Name != "__Host-deuterium_session" || !cookie.Secure || cookie.Domain != "" {
		t.Fatal("production cookie is not host-scoped and secure")
	}
	response, _ = f.request(t, "POST", "/api/v1/account/login", map[string]string{"account": "Alice", "password": strings.Repeat("x", 10000)}, nil, nil)
	if response.StatusCode != 400 {
		t.Fatal("oversized body accepted")
	}
	response, _ = f.request(t, "POST", "/api/v1/account/login", map[string]string{"account": "Alice", "password": password, "serverUuid": aliceUUID}, nil, nil)
	if response.StatusCode != 400 {
		t.Fatal("unexpected identity field accepted")
	}
}

func (f *fixture) socket(t *testing.T, path string, headers http.Header) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, r, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(f.http.URL, "http")+path, &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		if r != nil && r.Body != nil {
			r.Body.Close()
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	return c
}
func socketWrite(t *testing.T, c *websocket.Conn, kind string, payload any, eventID string) {
	t.Helper()
	message := map[string]any{"type": kind, "requestId": store.ID("test_"), "payload": payload}
	if eventID != "" {
		message["eventId"] = eventID
	}
	data, _ := json.Marshal(message)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}
func socketRead(t *testing.T, c *websocket.Conn, wanted string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	for range 100 {
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("waiting for %s: %v", wanted, err)
		}
		var frame map[string]any
		if err = json.Unmarshal(data, &frame); err != nil {
			t.Fatal(err)
		}
		if frame["type"] == wanted {
			return frame["payload"].(map[string]any)
		}
	}
	t.Fatal("too many unexpected frames")
	return nil
}
func (f *fixture) core(t *testing.T, node string) *websocket.Conn {
	t.Helper()
	headers := bearer(f.tokens[node])
	headers.Set("X-Deuterium-Node-ID", node)
	c := f.socket(t, "/bridge/v1/connect", headers)
	socketWrite(t, c, "core.hello", map[string]int{"protocolVersion": 1}, "")
	if hello := socketRead(t, c, "core.welcome"); hello["serverId"] != node {
		t.Fatal("node binding incorrect")
	}
	return c
}

func TestBrowserWebSocketRejectsWrongOriginAndURLTokens(t *testing.T) {
	f := newFixture(t)
	token := f.appToken(t)
	for _, test := range []struct{ path, origin string }{{"/api/v1/chat/ws", "https://attacker.example"}, {"/api/v1/chat/ws?token=forbidden", ""}} {
		h := bearer(token)
		if test.origin != "" {
			h.Set("Origin", test.origin)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c, r, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(f.http.URL, "http")+test.path, &websocket.DialOptions{HTTPHeader: h})
		cancel()
		if c != nil {
			c.CloseNow()
		}
		if r != nil && r.Body != nil {
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
		if err == nil {
			t.Fatal("unsafe WebSocket accepted")
		}
	}
	wrong := bearer(f.tokens["amiya"])
	wrong.Set("X-Deuterium-Node-ID", "odyssey")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, r, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(f.http.URL, "http")+"/bridge/v1/connect", &websocket.DialOptions{HTTPHeader: wrong})
	if c != nil {
		c.CloseNow()
	}
	if r != nil && r.Body != nil {
		r.Body.Close()
	}
	if err == nil {
		t.Fatal("node credential impersonated another node")
	}
}

func TestFourNodeChatReplayDeliveryAndRevocation(t *testing.T) {
	f := newFixture(t)
	token := f.appToken(t)
	app := f.socket(t, "/api/v1/chat/ws", bearer(token))
	cores := map[string]*websocket.Conn{}
	for _, node := range []string{"login", "amiya", "odyssey", "mek"} {
		cores[node] = f.core(t, node)
	}
	chat := map[string]string{"playerUuid": aliceUUID, "gameId": "Alice", "content": "hello from game"}
	socketWrite(t, cores["amiya"], "chat.public.event", chat, "game-message-one")
	if ack := socketRead(t, cores["amiya"], "core.event.result"); ack["status"] != "committed" {
		t.Fatal(ack)
	}
	if event := socketRead(t, app, "chat.message"); event["message"].(map[string]any)["content"] != "hello from game" {
		t.Fatal(event)
	}
	socketWrite(t, cores["amiya"], "chat.public.event", chat, "game-message-one")
	if ack := socketRead(t, cores["amiya"], "core.event.result"); ack["replayed"] != true {
		t.Fatal("replay not identified", ack)
	}
	if count(t, f.store, "chat_messages_next") != 1 || count(t, f.store, "core_chat_deliveries") != 0 {
		t.Fatal("game echo was duplicated or forwarded back to TrChat")
	}
	chat["content"] = "modified"
	socketWrite(t, cores["amiya"], "chat.public.event", chat, "game-message-one")
	if ack := socketRead(t, cores["amiya"], "core.event.result"); ack["status"] != "rejected" {
		t.Fatal("conflicting event accepted", ack)
	}
	socketWrite(t, app, "chat.send", map[string]string{"clientMessageId": "app-message-one", "content": "hello from app"}, "")
	accepted := socketRead(t, app, "chat.send.result")
	if accepted["status"] != "accepted" {
		t.Fatal(accepted)
	}
	messageID := accepted["messageId"].(string)
	for _, node := range []string{"login", "amiya", "odyssey", "mek"} {
		delivery := socketRead(t, cores[node], "chat.app.delivery")
		if delivery["messageId"] != messageID || delivery["suppressRebroadcast"] != true || delivery["senderUuid"] != aliceUUID {
			t.Fatal("wrong game delivery", delivery)
		}
		socketWrite(t, cores[node], "chat.delivery.ack", map[string]string{"messageId": messageID}, "")
		socketRead(t, cores[node], "chat.delivery.ack.result")
		pending, err := f.store.PendingChats(context.Background(), node)
		if err != nil || len(pending) != 0 {
			t.Fatal("ack did not persist", err)
		}
	}
	if err := f.store.AckChat(context.Background(), "unregistered-node", messageID); !errors.Is(err, store.ErrConflict) {
		t.Fatal("unrelated node acknowledged delivery", err)
	}
	for _, c := range cores {
		c.CloseNow()
	}
	deadline := time.Now().Add(3 * time.Second)
	for f.app.Hub.AnyNode(f.app.Config.ChatNodes()) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	socketWrite(t, app, "chat.send", map[string]string{"clientMessageId": "app-message-one", "content": "hello from app"}, "")
	if replay := socketRead(t, app, "chat.send.result"); replay["messageId"] != messageID || replay["status"] != "accepted" {
		t.Fatal("offline retry lost accepted result", replay)
	}
	socketWrite(t, app, "chat.send", map[string]string{"clientMessageId": "new-offline-message", "content": "offline"}, "")
	if failure := socketRead(t, app, "chat.send.result"); failure["status"] != "failed" {
		t.Fatal("new offline message silently accepted", failure)
	}
	if count(t, f.store, "chat_messages_next") != 2 {
		t.Fatal("retries duplicated public messages")
	}
	if err := f.store.Revoke(context.Background(), store.Digest([]byte(token))); err != nil {
		t.Fatal(err)
	}
	socketWrite(t, app, "chat.send", map[string]string{"clientMessageId": "revoked-message", "content": "must not send"}, "")
	if failure := socketRead(t, app, "chat.send.result"); failure["error"].(map[string]any)["code"] != "UNAUTHORIZED" {
		t.Fatal("revoked session still sends", failure)
	}
}

func TestOfflineNodeReceivesDurablePendingChatOnReconnect(t *testing.T) {
	f := newFixture(t)
	token := f.appToken(t)
	app := f.socket(t, "/api/v1/chat/ws", bearer(token))
	f.core(t, "amiya")
	socketWrite(t, app, "chat.send", map[string]string{"clientMessageId": "pending-for-mek", "content": "reconnect test"}, "")
	accepted := socketRead(t, app, "chat.send.result")
	if accepted["status"] != "accepted" {
		t.Fatal(accepted)
	}
	mek := f.core(t, "mek")
	delivery := socketRead(t, mek, "chat.app.delivery")
	if delivery["messageId"] != accepted["messageId"] {
		t.Fatal("offline node did not recover persisted delivery")
	}
	socketWrite(t, mek, "chat.delivery.ack", map[string]any{"messageId": accepted["messageId"]}, "")
	socketRead(t, mek, "chat.delivery.ack.result")
}

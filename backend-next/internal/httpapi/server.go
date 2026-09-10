package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var ErrForbidden = errors.New("forbidden")

type Server struct {
	Store        *store.Store
	Config       config.Config
	Identity     *identity.Service
	Hub          *bridge.Hub
	Core         *bridge.Runtime
	CommerceCore CommerceExecutorV2
	ctx          context.Context
	cancel       context.CancelFunc
	requests     chan struct{}
}

func New(s *store.Store, c config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{Store: s, Config: c, Identity: identity.New(s), Hub: bridge.NewHub(), Core: bridge.NewRuntime(), ctx: ctx, cancel: cancel, requests: make(chan struct{}, 256)}
	server.CommerceCore = &commerceCoreAdapter{server: server}
	nodes := make([]string, 0, len(c.Nodes))
	for _, node := range c.Nodes {
		nodes = append(nodes, node.ID)
	}
	go server.coreWorker(nodes)
	go server.emailWorkerV204()
	return server
}
func (s *Server) Close() { s.cancel(); s.Hub.Wake() }

type requestIDKey struct{}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerReleasesV2(mux)
	s.registerAssetsV2(mux)
	s.registerSocialV2(mux)
	s.registerCatalogV2(mux)
	s.registerCommerceV2(mux)
	s.registerAIV2(mux)
	s.registerAdminAuditV2(mux)
	s.registerAdminEmailV204(mux)
	s.registerAdminAccountsV206(mux)
	s.registerAdminCommerceV204(mux)
	s.registerLauncherIcons(mux)
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { success(w, r, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if s.Store.Ready(ctx) != nil {
			failure(w, r, 503, "NOT_READY", "服务尚未就绪。")
			return
		}
		success(w, r, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("POST /api/v1/account/login", s.appLogin)
	mux.HandleFunc("GET /api/v1/account/me", s.me)
	mux.HandleFunc("POST /api/v1/account/logout", s.logout)
	mux.HandleFunc("POST /api/v1/web/session", s.webLogin)
	mux.HandleFunc("GET /api/v1/web/session", s.webSession)
	mux.HandleFunc("DELETE /api/v1/web/session", s.logout)
	mux.HandleFunc("GET /api/v1/chat/messages", s.messages)
	mux.HandleFunc("POST /api/v1/chat/messages", s.publicSendHTTPV203)
	mux.HandleFunc("GET /api/v1/chat/ws", s.appSocket)
	mux.HandleFunc("GET /bridge/v1/connect", s.coreSocket)
	mux.HandleFunc("GET /api/v1/admin/core/nodes", s.nodes)
	mux.HandleFunc("GET /api/v1/admin/core/items", s.items)
	mux.HandleFunc("GET /api/v1/admin/core/items/{itemRef}/versions/{revision}", s.itemVersionV209)
	s.coreRoutes(mux)
	s.accountRoutes(mux)
	s.walletRoutes(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		failure(w, r, 404, "NOT_FOUND", "接口不存在或尚未实现。")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, store.ID("req_")))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		select {
		case s.requests <- struct{}{}:
			defer func() { <-s.requests }()
		default:
			failure(w, r, 503, "SERVER_BUSY", "请求过多，请稍后重试。")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func requestID(r *http.Request) string { v, _ := r.Context().Value(requestIDKey{}).(string); return v }
func success(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, 200, map[string]any{"requestId": requestID(r), "data": data})
}
func failure(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]any{"requestId": requestID(r), "error": map[string]any{"code": code, "message": message, "details": map[string]string{}}})
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func failError(w http.ResponseWriter, r *http.Request, err error) {
	var coreError *CoreError
	if errors.As(err, &coreError) {
		status := 409
		if coreError.Code == "RESULT_UNKNOWN" || coreError.Code == "ECONOMY_UNAVAILABLE" || coreError.Code == "PLAYER_OFFLINE" {
			status = 503
		}
		failure(w, r, status, coreError.Code, coreError.Message)
		return
	}
	switch {
	case errors.Is(err, store.ErrVerification):
		failure(w, r, 400, "VERIFICATION_INVALID", "验证码无效、已使用、已过期或尝试次数过多。")
	case errors.Is(err, store.ErrUnauthorized):
		failure(w, r, 401, "UNAUTHORIZED", "账号、密码或会话无效。")
	case errors.Is(err, ErrForbidden):
		failure(w, r, 403, "FORBIDDEN", "没有执行此操作的权限。")
	case errors.Is(err, store.ErrRateLimited):
		w.Header().Set("Retry-After", "60")
		failure(w, r, 429, "RATE_LIMITED", "操作过于频繁，请稍后重试。")
	case errors.Is(err, identity.ErrBusy):
		w.Header().Set("Retry-After", "1")
		failure(w, r, 503, "SERVER_BUSY", "服务繁忙，请稍后重试。")
	case errors.Is(err, bridge.ErrNodeOffline):
		failure(w, r, 503, "PLUGIN_BRIDGE_UNAVAILABLE", "游戏服务器连接暂不可用。")
	case errors.Is(err, store.ErrConflict):
		failure(w, r, 409, "CONFLICT", "同一请求标识对应不同内容。")
	case errors.Is(err, bridge.ErrProtocol):
		failure(w, r, 400, "INVALID_REQUEST", "请求格式不正确。")
	default:
		failure(w, r, 503, "SERVICE_UNAVAILABLE", "服务暂不可用，请稍后重试。")
	}
}

func body(w http.ResponseWriter, r *http.Request, target any) error {
	if ct := strings.Split(r.Header.Get("Content-Type"), ";")[0]; strings.TrimSpace(strings.ToLower(ct)) != "application/json" {
		return bridge.ErrProtocol
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return bridge.ErrProtocol
	}
	return bridge.Decode(data, target)
}

func (s *Server) cookieName() string {
	if s.Config.Development {
		return "deuterium_dev_session"
	}
	return "__Host-deuterium_session"
}
func (s *Server) setCookie(w http.ResponseWriter, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if token == "" {
		maxAge = -1
	}
	http.SetCookie(w, &http.Cookie{Name: s.cookieName(), Value: token, Path: "/", Secure: !s.Config.Development, HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: maxAge})
}

func (s *Server) origin(r *http.Request) bool { return r.Header.Get("Origin") == s.Config.PublicOrigin }
func (s *Server) remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "unknown"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "unknown"
	}
	if s.Config.TrustedProxy(ip) {
		if forwarded := net.ParseIP(r.Header.Get("X-Real-IP")); forwarded != nil {
			return forwarded.String()
		}
	}
	return ip.String()
}

func (s *Server) authenticate(r *http.Request) (v store.Session, err error) {
	bearer := r.Header.Get("Authorization")
	cookie, cookieErr := r.Cookie(s.cookieName())
	var token, kind string
	if bearer != "" {
		if cookieErr == nil || !strings.HasPrefix(bearer, "Bearer ") {
			return v, store.ErrUnauthorized
		}
		token = strings.TrimPrefix(bearer, "Bearer ")
		kind = "app"
	} else if cookieErr == nil {
		token = cookie.Value
		kind = "web"
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			return v, ErrForbidden
		}
	} else {
		return v, store.ErrUnauthorized
	}
	if len(token) != 43 {
		return v, store.ErrUnauthorized
	}
	v, err = s.Store.Session(r.Context(), store.Digest([]byte(token)))
	if err != nil {
		return
	}
	if v.Kind != kind {
		return v, store.ErrUnauthorized
	}
	if kind == "web" && r.Method != "GET" && r.Method != "HEAD" {
		if !s.origin(r) || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(v.CSRF)) != 1 {
			return v, ErrForbidden
		}
	}
	return
}

func (s *Server) appLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "" {
		failure(w, r, 400, "USE_WEB_SESSION", "浏览器请使用网页会话接口。")
		return
	}
	s.login(w, r, "app")
}
func (s *Server) webLogin(w http.ResponseWriter, r *http.Request) {
	if !s.origin(r) {
		failError(w, r, ErrForbidden)
		return
	}
	s.login(w, r, "web")
}
func (s *Server) login(w http.ResponseWriter, r *http.Request, kind string) {
	var input struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := body(w, r, &input); err != nil {
		failError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	token, session, err := s.Identity.Login(ctx, input.Account, input.Password, s.remoteIP(r), kind)
	if err != nil {
		if errors.Is(err, store.ErrUnauthorized) {
			failure(w, r, 401, "ACCOUNT_PASSWORD_INVALID", "账号或密码错误。")
			return
		}
		if errors.Is(err, store.ErrRateLimited) {
			w.Header().Set("Retry-After", "900")
			writeJSON(w, 429, map[string]any{"requestId": requestID(r), "error": map[string]any{"code": "LOGIN_LOCKED", "message": "登录尝试过多，请稍后重试。", "details": map[string]string{}, "retryAfterSeconds": 900}})
			return
		}
		failError(w, r, err)
		return
	}
	session.User.Permissions, err = s.Store.PermissionsV2(ctx, session.User.ID)
	if err != nil {
		_ = s.Store.Revoke(ctx, session.TokenHash)
		failError(w, r, err)
		return
	}
	if kind == "web" {
		// Login rotates an existing web session, never leaves a fixed session ID.
		if old, e := r.Cookie(s.cookieName()); e == nil {
			if err = s.Store.Revoke(ctx, store.Digest([]byte(old.Value))); err != nil {
				_ = s.Store.Revoke(ctx, session.TokenHash)
				failError(w, r, err)
				return
			}
		}
		s.setCookie(w, token, session.ExpiresAt)
		success(w, r, map[string]any{"user": session.User, "csrfToken": session.CSRF, "expiresAt": session.ExpiresAt})
	} else {
		success(w, r, map[string]any{"token": token, "user": session.User})
	}
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	v, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	v.User.Permissions, err = s.Store.PermissionsV2(r.Context(), v.User.ID)
	if err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]any{"user": v.User})
}
func (s *Server) webSession(w http.ResponseWriter, r *http.Request) {
	v, err := s.authenticate(r)
	if err == nil && v.Kind != "web" {
		err = ErrForbidden
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	v.User.Permissions, err = s.Store.PermissionsV2(r.Context(), v.User.ID)
	if err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]any{"user": v.User, "csrfToken": v.CSRF, "expiresAt": v.ExpiresAt})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	v, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	if err = s.Store.Revoke(r.Context(), v.TokenHash); err != nil {
		failError(w, r, err)
		return
	}
	if v.Kind == "web" {
		s.setCookie(w, "", time.Unix(1, 0))
	}
	s.Hub.Wake()
	success(w, r, map[string]bool{"loggedOut": true})
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authenticate(r); err != nil {
		failError(w, r, err)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		limit = n
	}
	cursor := int64(0)
	if raw := r.URL.Query().Get("before"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || !strings.HasPrefix(string(decoded), "chat:") {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		cursor, err = strconv.ParseInt(strings.TrimPrefix(string(decoded), "chat:"), 10, 64)
		if err != nil || cursor <= 0 {
			failError(w, r, bridge.ErrProtocol)
			return
		}
	}
	messages, err := s.Store.Messages(r.Context(), cursor, false, limit+1)
	if err != nil {
		failError(w, r, err)
		return
	}
	var next any
	if len(messages) > limit {
		messages = messages[:limit]
		next = base64.RawURLEncoding.EncodeToString([]byte("chat:" + strconv.FormatInt(messages[len(messages)-1].Sequence, 10)))
	}
	writeJSON(w, 200, map[string]any{"requestId": requestID(r), "data": map[string]any{"messages": messages}, "page": map[string]any{"nextCursor": next}})
}

func (s *Server) admin(r *http.Request, permission string) (store.User, error) {
	v, err := s.authenticate(r)
	if err != nil {
		return v.User, err
	}
	ok, err := s.Store.HasPermission(r.Context(), v.User.ID, permission)
	if err != nil {
		return v.User, err
	}
	if !ok {
		return v.User, ErrForbidden
	}
	return v.User, nil
}
func (s *Server) nodes(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "core.read"); err != nil {
		failError(w, r, err)
		return
	}
	nodes := []map[string]any{}
	for _, n := range s.Config.Nodes {
		var runtime map[string]any
		_ = json.Unmarshal(s.Core.Status(n.ID), &runtime)
		nodes = append(nodes, map[string]any{"serverId": n.ID, "online": s.Hub.NodeOnline(n.ID), "chat": n.Chat, "itemPublisher": n.ItemPrefix != "", "claimEnabled": n.ClaimEnabled, "inventoryDomain": n.InventoryDomain, "economyConfigured": n.Economy, "runtime": runtime})
	}
	success(w, r, map[string]any{"nodes": nodes})
}
func (s *Server) items(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "core.read"); err != nil {
		failError(w, r, err)
		return
	}
	ref := r.URL.Query().Get("afterItemRef")
	rev := int64(0)
	if len(ref) > 96 {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if raw := r.URL.Query().Get("afterRevision"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		rev = n
	}
	query, domain := r.URL.Query().Get("q"), r.URL.Query().Get("inventoryDomain")
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 100 || len(domain) > 128 {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	items, err := s.Store.CoreItemsSearchV209(r.Context(), ref, rev, 51, query, domain)
	if err != nil {
		failError(w, r, err)
		return
	}
	var next any
	if len(items) > 50 {
		items = items[:50]
		last := items[49]
		next = map[string]any{"afterItemRef": last.ItemRef, "afterRevision": last.Revision}
	}
	success(w, r, map[string]any{"items": items, "next": next})
}

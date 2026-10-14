package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func sessionDeviceContext(ctx context.Context, r *http.Request, kind, device string) context.Context {
	if kind == "web" {
		ua := r.UserAgent()
		browser, platform := "浏览器", ""
		for _, item := range []struct{ marker, label string }{{"DLauncher/", "启动器社区"}, {"Edg/", "Edge"}, {"Firefox/", "Firefox"}, {"Chrome/", "Chrome"}, {"Safari/", "Safari"}} {
			if strings.Contains(ua, item.marker) {
				browser = item.label
				break
			}
		}
		for _, item := range []struct{ marker, label string }{{"Android", "Android"}, {"iPhone", "iPhone"}, {"iPad", "iPad"}, {"Windows", "Windows"}, {"Macintosh", "macOS"}, {"Linux", "Linux"}} {
			if strings.Contains(ua, item.marker) {
				platform = item.label
				break
			}
		}
		device = strings.TrimSpace(platform + " · " + browser)
		if platform == "" {
			device = browser
		}
	} else if strings.TrimSpace(device) == "" {
		device = "App 设备"
	}
	return store.WithSessionDevice(ctx, device)
}

func accountSessionRoute(r *http.Request) bool {
	if r.URL.Path == "/api/v1/account/sessions" {
		return r.Method == "GET"
	}
	if r.URL.Path == "/api/v1/account/sessions/revoke-others" {
		return r.Method == "POST"
	}
	return r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/v1/account/sessions/") && !strings.Contains(strings.TrimPrefix(r.URL.Path, "/api/v1/account/sessions/"), "/")
}

func (s *Server) registerAccountSessions(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/account/sessions", func(w http.ResponseWriter, r *http.Request) {
		session, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		items, err := s.Store.DeviceSessions(r.Context(), session.User.ID, session.TokenHash)
		if err != nil {
			failError(w, r, err)
			return
		}
		success(w, r, map[string]any{"items": items})
	})
	mux.HandleFunc("DELETE /api/v1/account/sessions/{sessionId}", s.revokeAccountSessions)
	mux.HandleFunc("POST /api/v1/account/sessions/revoke-others", s.revokeAccountSessions)
}

func (s *Server) revokeAccountSessions(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	others := r.Method == "POST"
	if others {
		var input struct{}
		if body(w, r, &input) != nil {
			failError(w, r, bridge.ErrProtocol)
			return
		}
	} else if !store.ValidSocialID(r.PathValue("sessionId")) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	err = s.Store.RevokeDeviceSessions(r.Context(), session.User.ID, session.TokenHash, r.PathValue("sessionId"), others)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	current := !others && r.PathValue("sessionId") == session.ID
	if current && session.Kind == "web" {
		s.setCookie(w, "", time.Unix(1, 0))
	}
	s.Hub.Wake(bridge.UserTopic(session.User.ID))
	success(w, r, map[string]bool{"revoked": true, "current": current})
}

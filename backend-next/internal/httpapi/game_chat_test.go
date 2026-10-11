package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestGameChatTokenCannotEscapeChatScope(t *testing.T) {
	for _, v := range []struct {
		method, path string
		allowed      bool
	}{
		{"GET", "/api/v1/chat/avatars/Bob", true}, {"POST", "/api/v1/chat/avatars/Bob", false},
		{"GET", "/api/v1/chat/messages", true}, {"POST", "/api/v1/chat/messages", true},
		{"POST", "/api/v1/chat/media/uploads/a/complete", true}, {"GET", "/api/v1/chat/media/files/a/access", true},
		{"POST", "/api/v1/chat/conversations/a/messages", true}, {"GET", "/api/v1/chat/ws", false},
		{"GET", "/api/v1/wallet/balance", false}, {"GET", "/api/v1/account/me", false},
		{"POST", "/api/v1/chat/admin", false}, {"DELETE", "/api/v1/chat/messages", false},
		{"POST", "/api/v1/chat/conversations/a/members", false}, {"GET", "/api/v1/chat/media/files/a/other", false},
	} {
		if got := gameChatRoute(httptest.NewRequest(v.method, v.path, nil)); got != v.allowed {
			t.Errorf("%s %s: %v", v.method, v.path, got)
		}
	}
}

package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
)

func TestWebLoginMigrationOrigins(t *testing.T) {
	s := &Server{Config: config.Config{PublicOrigin: "https://47.103.99.34", AdditionalPublicOrigins: []string{"https://chat.deuteriumix.com"}}}
	for _, tc := range []struct {
		origin string
		status int
	}{
		{"https://47.103.99.34", 400},
		{"https://chat.deuteriumix.com", 400},
		{"https://chat.deuteriumix.com.attacker.invalid", 403},
		{"https://attacker.invalid", 403},
		{"http://chat.deuteriumix.com", 403},
		{"null", 403},
		{"", 403},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/v1/web/login", strings.NewReader("{"))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			s.webLogin(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}

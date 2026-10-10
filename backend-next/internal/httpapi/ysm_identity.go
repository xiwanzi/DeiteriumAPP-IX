package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

var ysmProfileUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (s *Server) ysmIdentity(w http.ResponseWriter, r *http.Request) {
	// Separate service credential: never accept website sessions, node tokens,
	// client-supplied role claims, or Minecraft names here.
	expected, err := hex.DecodeString(s.Config.YSMIdentityTokenSHA256)
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	digest := sha256.Sum256([]byte(token))
	if err != nil || len(expected) != 32 || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || len(token) < 32 || subtle.ConstantTimeCompare(expected, digest[:]) != 1 {
		failure(w, r, 401, "UNAUTHENTICATED", "服务身份无效。")
		return
	}
	uuid := r.PathValue("uuid")
	if !ysmProfileUUID.MatchString(uuid) || r.URL.RawQuery != "" {
		failure(w, r, 400, "INVALID_UUID", "需要正版 UUID。")
		return
	}
	identity, err := s.Store.YSMIdentity(r.Context(), uuid)
	if errors.Is(err, sql.ErrNoRows) {
		failure(w, r, 403, "IDENTITY_NOT_ELIGIBLE", "没有对应的已验证有效账号。")
		return
	}
	if err != nil {
		failure(w, r, 503, "IDENTITY_UNAVAILABLE", "身份查询暂不可用。")
		return
	}
	success(w, r, identity)
}

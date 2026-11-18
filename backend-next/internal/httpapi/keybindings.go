package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func officialKeyUUID(id string) bool {
	return gameUUID.MatchString(id) && id[14] == '4' && strings.ContainsRune("89ab", rune(id[19]))
}

// The token only works here and expires with the authenticated game connection or API process.
func (s *Server) keySession(live store.CorePlayerIdentity) string {
	mac := hmac.New(sha256.New, s.keySecret)
	mac.Write([]byte("keybindings\x00" + live.PlayerUUID + "\x00" + live.ServerID + "\x00" + live.SessionEpoch))
	return hex.EncodeToString(mac.Sum(nil))
}
func keyFailure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		failure(w, r, 404, "NOT_FOUND", "该玩家或动作尚未同步。")
	case errors.Is(err, store.ErrConflict):
		failure(w, r, 409, "KEY_CHANGED", "这个键位刚刚发生了变化，请刷新后重试。")
	case errors.Is(err, store.ErrSocialInvalid):
		failure(w, r, 400, "INVALID_KEYBINDING", "键位数据不正确。")
	default:
		failError(w, r, err)
	}
}
func (s *Server) registerKeybindings(mux *http.ServeMux) {
	mux.HandleFunc("POST /bridge/v1/game-chat/keybindings/session", s.issueKeySession)
	mux.HandleFunc("POST /api/v1/game/keybindings/sync", s.syncGameKeys)
	mux.HandleFunc("GET /api/v1/admin/keybindings", s.listGameKeys)
	mux.HandleFunc("GET /api/v1/admin/keybindings/{uuid}", s.detailGameKeys)
	mux.HandleFunc("PUT /api/v1/admin/keybindings/{uuid}", s.editGameKey)
}
func (s *Server) issueKeySession(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	node, valid := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), token)
	if !ok || !valid || !node.Chat || len(s.keySecret) < 32 {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	var in struct {
		UUID  string `json:"playerUuid"`
		Epoch string `json:"sessionEpoch"`
	}
	if socialBodyV2(w, r, &in, 512) != nil || !officialKeyUUID(in.UUID) || in.Epoch == "" {
		keyFailure(w, r, store.ErrSocialInvalid)
		return
	}
	live, err := s.Core.Player("", in.UUID)
	if err != nil || !live.Online || live.ServerID != node.ID || live.SessionEpoch != in.Epoch {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	success(w, r, map[string]string{"playerUuid": live.PlayerUUID, "token": s.keySession(live)})
}
func (s *Server) syncGameKeys(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Deuterium-Player")
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || len(s.keySecret) < 32 || len(token) != 64 || !officialKeyUUID(id) {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	live, err := s.Core.Player("", id)
	if err != nil || !live.Online || live.SessionEpoch == "" || !hmac.Equal([]byte(s.keySession(live)), []byte(token)) {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	var in store.KeySync
	if socialBodyV2(w, r, &in, 2<<20) != nil || in.PlayerUUID != id {
		keyFailure(w, r, store.ErrSocialInvalid)
		return
	}
	out, err := s.Store.SyncKeybindings(r.Context(), id, live.GameID, in)
	if err != nil {
		keyFailure(w, r, err)
		return
	}
	success(w, r, out)
}
func (s *Server) listGameKeys(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		var err error
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			keyFailure(w, r, store.ErrSocialInvalid)
			return
		}
	}
	if len(q) > 192 {
		keyFailure(w, r, store.ErrSocialInvalid)
		return
	}
	items, err := s.Store.KeyProfiles(r.Context(), q, offset)
	if err != nil {
		keyFailure(w, r, err)
		return
	}
	more := len(items) > 30
	if more {
		items = items[:30]
	}
	success(w, r, map[string]any{"items": items, "hasMore": more})
}
func (s *Server) detailGameKeys(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	id := r.PathValue("uuid")
	if !officialKeyUUID(id) {
		keyFailure(w, r, store.ErrSocialInvalid)
		return
	}
	var since int64
	if raw := r.URL.Query().Get("since"); raw != "" {
		value, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || value < 0 {
			keyFailure(w, r, store.ErrSocialInvalid)
			return
		}
		since = value
	}
	out, err := s.Store.KeyProfileDetail(r.Context(), id, since)
	if err != nil {
		keyFailure(w, r, err)
		return
	}
	live, _ := s.Core.Player("", id)
	success(w, r, struct {
		store.KeyState
		Online bool `json:"online"`
	}{out, live.Online})
}
func (s *Server) editGameKey(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	id := r.PathValue("uuid")
	var in struct {
		Action   string `json:"action"`
		Value    string `json:"value"`
		Expected int64  `json:"expectedRevision"`
	}
	if !officialKeyUUID(id) || socialBodyV2(w, r, &in, 2048) != nil {
		keyFailure(w, r, store.ErrSocialInvalid)
		return
	}
	if err = s.Store.SetAdminKey(r.Context(), actor.ID, id, in.Action, in.Value, in.Expected); err != nil {
		keyFailure(w, r, err)
		return
	}
	if live, e := s.Core.Player("", id); e == nil && live.Online {
		_ = s.Core.SendGameChat(r.Context(), live.ServerID, map[string]string{"type": "keybindings-changed", "playerUuid": id, "sessionEpoch": live.SessionEpoch})
	}
	success(w, r, map[string]bool{"saved": true})
}

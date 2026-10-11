package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// A game token is deliberately not a general App session. Keep this allowlist
// exact: adding a future /chat/admin route must not implicitly grant access.
func gameChatRoute(r *http.Request) bool {
	p, m := strings.TrimPrefix(r.URL.Path, "/api/v1/"), r.Method
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		return false
	}
	switch p {
	case "chat/game-messages":
		return m == "GET"
	case "chat/messages":
		return m == "GET" || m == "POST"
	case "chat/player-directory", "chat/presence", "chat/online-players":
		return m == "GET"
	case "chat/stickers", "chat/conversations", "chat/follows":
		return m == "GET" || m == "POST"
	case "chat/media/uploads":
		return m == "POST"
	}
	parts := strings.Split(p, "/")
	if len(parts) == 3 && parts[0] == "chat" {
		return ((parts[1] == "sends" || parts[1] == "avatars") && m == "GET") || ((parts[1] == "stickers" || parts[1] == "follows") && m == "DELETE")
	}
	if len(parts) == 4 && parts[0] == "chat" && parts[1] == "conversations" {
		return (parts[3] == "messages" && (m == "GET" || m == "POST")) || (parts[3] == "read" && m == "POST")
	}
	if len(parts) == 4 && parts[0] == "chat" && parts[1] == "media" && parts[2] == "uploads" {
		return m == "GET"
	}
	if len(parts) == 5 && parts[0] == "chat" && parts[1] == "media" {
		return (parts[2] == "uploads" && parts[4] == "complete" && m == "POST") || (parts[2] == "files" && parts[4] == "access" && m == "GET")
	}
	return false
}

func (s *Server) registerGameChat(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/chat/avatars/{gameId}", s.gameAvatar)
	mux.HandleFunc("POST /bridge/v1/game-chat/team", s.relayGameTeam)
	mux.HandleFunc("POST /bridge/v1/game-chat/media", s.sendGameMedia)
	mux.HandleFunc("GET /api/v1/chat/game-messages", func(w http.ResponseWriter, r *http.Request) {
		session, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		channel := r.URL.Query().Get("channel")
		if channel != "nearby" && channel != "team" {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		messages, err := s.Store.GameChatMessages(r.Context(), session.User.ServerUUID, channel)
		if err != nil {
			failError(w, r, err)
			return
		}
		success(w, r, messages)
	})
	mux.HandleFunc("POST /bridge/v1/game-chat/session", func(w http.ResponseWriter, r *http.Request) {
		node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !ok || !node.Chat || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		var in struct {
			PlayerUUID string `json:"playerUuid"`
		}
		if err := socialBodyV2(w, r, &in, 256); err != nil {
			socialFailureV2(w, r, err)
			return
		}
		// Trust only Core's authenticated live presence, not a client-supplied name.
		live, err := s.Core.Player("", in.PlayerUUID)
		if err != nil || !live.Online || live.ServerID != node.ID {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		var u store.User
		err = s.Store.DB.QueryRowContext(r.Context(), `SELECT id,player_ref,game_id,COALESCE(qq,''),server_uuid,password_hash,status,game_verified FROM identities WHERE server_uuid=? AND status='active' AND game_verified=TRUE`, in.PlayerUUID).Scan(&u.ID, &u.PlayerRef, &u.GameID, &u.QQ, &u.ServerUUID, &u.PasswordHash, &u.Status, &u.GameVerified)
		if err != nil {
			failure(w, r, 403, "GAME_ACCOUNT_REQUIRED", "请先在 App 绑定当前游戏账号以使用云端聊天媒体。")
			return
		}
		// Existing per-account durable limiter bounds session rows and request bursts.
		if err = s.Store.ReserveGameChat(r.Context(), "session:"+u.ID); err != nil {
			failError(w, r, err)
			return
		}
		token := identity.Secret()
		expiry := time.Now().UTC().Add(10 * time.Minute)
		if err = s.Store.CreateSession(r.Context(), u, store.Digest([]byte(token)), "gamechat", node.ID, expiry, ""); err != nil {
			failError(w, r, err)
			return
		}
		success(w, r, map[string]any{"token": token, "expiresAt": expiry, "playerRef": u.PlayerRef, "gameId": u.GameID, "serverId": node.ID})
	})
}

func (s *Server) sendGameMedia(w http.ResponseWriter, r *http.Request) {
	node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if !ok || !node.Chat || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	var in store.GameChatMediaRequest
	if socialBodyV2(w, r, &in, 28000) != nil || !store.ValidSocialID(in.ClientMessageID) || !in.Media.Valid() || (in.Channel != "nearby" && in.Channel != "team") || len(in.Members) < 1 || len(in.Members) > 256 {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	if in.Channel == "team" && (!gameUUID.MatchString(in.TeamID) || in.TeamRevision < 0) {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	live, err := s.Core.Player("", in.PlayerUUID)
	if err != nil || !live.Online || live.ServerID != node.ID {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	sender := false
	for _, id := range in.Members {
		if !gameUUID.MatchString(id) {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		sender = sender || id == in.PlayerUUID
	}
	if !sender {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	message, err := s.Store.GameChatMedia(r.Context(), node.ID, in)
	if err != nil {
		chatMediaFailure(w, r, err)
		return
	}
	if in.Channel == "team" {
		for _, target := range s.Config.ChatNodes() {
			if target != node.ID {
				_ = s.Core.SendGameChat(r.Context(), target, message)
			}
		}
	}
	success(w, r, message)
}

var gameUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (s *Server) relayGameTeam(w http.ResponseWriter, r *http.Request) {
	node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if !ok || !node.Chat || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	var in struct {
		Type      string    `json:"type"`
		MessageID string    `json:"messageId"`
		Channel   string    `json:"channel"`
		ServerID  string    `json:"serverId"`
		Content   string    `json:"content"`
		SentAt    time.Time `json:"sentAt"`
		Sender    struct {
			PlayerRef string `json:"playerRef"`
			GameID    string `json:"gameId"`
		} `json:"sender"`
		TeamID   string   `json:"teamId"`
		Revision int64    `json:"teamRevision"`
		Members  []string `json:"members"`
	}
	if socialBodyV2(w, r, &in, 28000) != nil || in.Type != "message" || in.Channel != "team" || in.ServerID != node.ID || !store.ValidSocialID(in.MessageID) || !store.ValidSocialText(in.Content, 1, 256) || !gameUUID.MatchString(in.TeamID) || len(in.Members) < 1 || len(in.Members) > 256 || in.Revision < 0 || time.Since(in.SentAt) > 15*time.Second || time.Until(in.SentAt) > 5*time.Second {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	live, err := s.Core.Player("", in.Sender.PlayerRef)
	if err != nil || !live.Online || live.ServerID != node.ID || live.GameID != in.Sender.GameID {
		failError(w, r, store.ErrUnauthorized)
		return
	}
	containsSender := false
	for _, id := range in.Members {
		if !gameUUID.MatchString(id) {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		containsSender = containsSender || id == in.Sender.PlayerRef
	}
	if !containsSender {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	if err = s.Store.ReserveGameChat(r.Context(), in.Sender.PlayerRef); err != nil {
		failError(w, r, err)
		return
	}
	payload, _ := json.Marshal(in)
	delivered := 0
	for _, target := range s.Config.ChatNodes() {
		if target != node.ID && s.Core.SendGameChat(r.Context(), target, json.RawMessage(payload)) == nil {
			delivered++
		}
	}
	// Receiving Core rechecks both sender and recipient against its committed FTB snapshot.
	success(w, r, map[string]any{"accepted": true, "nodes": delivered})
}

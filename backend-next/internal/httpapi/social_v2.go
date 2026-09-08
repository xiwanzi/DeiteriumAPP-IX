package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerSocialV2(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/players/{playerRef}", s.socialProfileV2)
	mux.HandleFunc("PATCH /api/v1/account/me/profile", s.socialPatchProfileV2)
	mux.HandleFunc("GET /api/v1/chat/player-directory", s.socialDirectoryV2)
	mux.HandleFunc("GET /api/v1/chat/follows", s.socialFollowsV2)
	mux.HandleFunc("POST /api/v1/chat/follows", s.socialFollowV2)
	mux.HandleFunc("DELETE /api/v1/chat/follows/{playerRef}", s.socialUnfollowV2)
	mux.HandleFunc("GET /api/v1/chat/conversations", s.socialConversationsV2)
	mux.HandleFunc("POST /api/v1/chat/conversations", s.socialCreateConversationV2)
	mux.HandleFunc("GET /api/v1/chat/conversations/{conversationId}/messages", s.socialMessagesV2)
	mux.HandleFunc("POST /api/v1/chat/conversations/{conversationId}/messages", s.socialSendV2)
	mux.HandleFunc("POST /api/v1/chat/conversations/{conversationId}/forwards", s.socialForwardV2)
	mux.HandleFunc("POST /api/v1/chat/conversations/{conversationId}/read", s.socialReadConversationV2)
	mux.HandleFunc("GET /api/v1/announcements", s.socialAnnouncementsV2)
	mux.HandleFunc("GET /api/v1/announcements/{announcementId}", s.socialAnnouncementV2)
	mux.HandleFunc("GET /api/v1/admin/announcements", s.socialAdminAnnouncementsV2)
	mux.HandleFunc("POST /api/v1/admin/announcements", s.socialWriteAnnouncementV2)
	mux.HandleFunc("GET /api/v1/admin/announcements/{announcementId}", s.socialAdminAnnouncementV2)
	mux.HandleFunc("PUT /api/v1/admin/announcements/{announcementId}", s.socialWriteAnnouncementV2)
	mux.HandleFunc("POST /api/v1/admin/announcements/{announcementId}/publish", s.socialPublishAnnouncementV2)
	mux.HandleFunc("POST /api/v1/admin/announcements/{announcementId}/unpublish", s.socialUnpublishAnnouncementV2)
	mux.HandleFunc("GET /api/v1/notifications", s.socialNotificationsV2)
	mux.HandleFunc("POST /api/v1/notifications/read", s.socialReadNotificationsV2)
	mux.HandleFunc("GET /api/v1/notifications/preferences", s.socialPreferencesV2)
	mux.HandleFunc("PATCH /api/v1/notifications/preferences", s.socialPatchPreferencesV2)
}

func socialFailureV2(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrSocialNotFound):
		failure(w, r, 404, "NOT_FOUND", "内容不存在或无权访问。")
	case errors.Is(err, store.ErrSocialInvalid):
		failure(w, r, 400, "INVALID_REQUEST", "请求字段或内容不符合要求。")
	case errors.Is(err, store.ErrSocialVersion):
		failure(w, r, 409, "STATE_VERSION_CONFLICT", "内容已更新，请刷新后重试。")
	case errors.Is(err, store.ErrSocialCapability):
		failure(w, r, 503, "CAPABILITY_UNAVAILABLE", "这项服务暂不可用，请稍后重试。")
	default:
		assetErrorV2(w, r, err)
	}
}
func (s *Server) socialUserV2(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	v, err := s.authenticate(r)
	if err != nil {
		socialFailureV2(w, r, err)
		return v.User, false
	}
	return v.User, true
}
func socialBodyV2(w http.ResponseWriter, r *http.Request, value any, limit int64) error {
	if strings.TrimSpace(strings.ToLower(strings.Split(r.Header.Get("Content-Type"), ";")[0])) != "application/json" {
		return store.ErrSocialInvalid
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return store.ErrSocialInvalid
	}
	if err = bridge.Decode(data, value); err != nil {
		return store.ErrSocialInvalid
	}
	return nil
}

type socialCursorV2 struct {
	Scope  string    `json:"s"`
	Before int64     `json:"b,omitempty"`
	Time   time.Time `json:"t,omitempty"`
	ID     string    `json:"i,omitempty"`
}

func encodeSocialCursorV2(cursor socialCursorV2) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}
func socialPageV2(r *http.Request, scope string) (int, socialCursorV2, error) {
	limit := 20
	var cursor socialCursorV2
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 100 {
			return 0, cursor, store.ErrSocialInvalid
		}
		limit = v
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		if len(raw) > 512 {
			return 0, cursor, store.ErrSocialInvalid
		}
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || bridge.Decode(decoded, &cursor) != nil || cursor.Scope != scope || cursor.Before < 0 {
			return 0, cursor, store.ErrSocialInvalid
		}
	}
	return limit, cursor, nil
}

func (s *Server) socialProfileV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	p, err := s.Store.PublicProfileV2(r.Context(), u.ID, r.PathValue("playerRef"))
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, p)
}
func (s *Server) socialPatchProfileV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input store.SocialProfilePatch
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	p, err := s.Store.PatchProfileV2(r.Context(), u, input)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, p)
}
func (s *Server) socialDirectoryV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	limit, _, err := socialPageV2(r, "directory:"+u.ID)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	if r.URL.Query().Get("limit") == "" {
		limit = 100
	}
	query := r.URL.Query().Get("query")
	if query == "" {
		query = r.URL.Query().Get("q")
	}
	profiles, err := s.Store.PlayerDirectoryV2(r.Context(), u.ID, query, limit)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"players": profiles})
}
func (s *Server) socialFollowsV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	values, err := s.Store.FollowsV2(r.Context(), u.ID)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"players": values})
}
func (s *Server) socialFollowV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input struct {
		PlayerRef string `json:"playerRef"`
	}
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	p, err := s.Store.FollowV2(r.Context(), u, input.PlayerRef, true)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"followed": true, "player": p})
}
func (s *Server) socialUnfollowV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	p, err := s.Store.FollowV2(r.Context(), u, r.PathValue("playerRef"), false)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"followed": false, "player": p})
}
func (s *Server) socialConversationsV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	scope := "conversations:" + u.ID
	limit, cursor, err := socialPageV2(r, scope)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	values, err := s.Store.ConversationsV2(r.Context(), u.ID, cursor.Time, cursor.ID, limit+1)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var next any
	more := len(values) > limit
	if more {
		values = values[:limit]
		last := values[len(values)-1]
		next = encodeSocialCursorV2(socialCursorV2{Scope: scope, Time: last.UpdatedAt, ID: last.ConversationID})
	}
	v2List(w, r, values, next, more)
}
func (s *Server) socialCreateConversationV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input struct {
		ClientRequestID string `json:"clientRequestId"`
		OtherPlayerRef  string `json:"otherPlayerRef"`
	}
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	result, err := s.Store.CreateConversationV2(r.Context(), u, input.ClientRequestID, input.OtherPlayerRef)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	writeJSON(w, 201, map[string]any{"requestId": requestID(r), "data": result, "serverTime": time.Now().UTC()})
}
func (s *Server) socialMessagesV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	conversation := r.PathValue("conversationId")
	scope := "messages:" + u.ID + ":" + conversation
	limit, cursor, err := socialPageV2(r, scope)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	beforeID := r.URL.Query().Get("beforeMessageId")
	if beforeID != "" && !store.ValidSocialID(beforeID) {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	if beforeID != "" && cursor.Before > 0 {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	values, err := s.Store.ConversationMessagesV2(r.Context(), u.ID, conversation, beforeID, cursor.Before, limit+1)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var next any
	more := len(values) > limit
	if more {
		values = values[:limit]
		next = encodeSocialCursorV2(socialCursorV2{Scope: scope, Before: values[len(values)-1].Sequence})
	}
	v2List(w, r, values, next, more)
}
func (s *Server) socialSendV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input store.SocialSendRequest
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	result, err := s.Store.SendDirectV2(r.Context(), u, r.PathValue("conversationId"), input, nil)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	s.Hub.Wake()
	v2Success(w, r, result)
}
func (s *Server) socialForwardV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input store.SocialForwardRequest
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	if input.SourceConversationID != "" && input.SourceConversationID != "public" && !store.ValidSocialID(input.SourceConversationID) {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	result, err := s.Store.SendDirectV2(r.Context(), u, r.PathValue("conversationId"), store.SocialSendRequest{ClientMessageID: input.ClientMessageID}, &input)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	s.Hub.Wake()
	v2Success(w, r, result)
}
func (s *Server) socialReadConversationV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input struct {
		ClientRequestID   string `json:"clientRequestId"`
		LastReadMessageID string `json:"lastReadMessageId"`
	}
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	result, err := s.Store.ReadConversationV2(r.Context(), u.ID, r.PathValue("conversationId"), input.ClientRequestID, input.LastReadMessageID)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) socialNotificationsV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	scope := "notifications:" + u.ID + ":" + r.URL.Query().Get("unreadOnly")
	limit, cursor, err := socialPageV2(r, scope)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	raw := r.URL.Query().Get("unreadOnly")
	if raw != "" && raw != "true" && raw != "false" {
		socialFailureV2(w, r, store.ErrSocialInvalid)
		return
	}
	values, err := s.Store.NotificationsV2(r.Context(), u.ID, cursor.Before, raw == "true", limit+1)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var next any
	more := len(values) > limit
	if more {
		values = values[:limit]
		next = encodeSocialCursorV2(socialCursorV2{Scope: scope, Before: values[len(values)-1].Sequence})
	}
	v2List(w, r, values, next, more)
}
func (s *Server) socialReadNotificationsV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input struct {
		ClientRequestID string   `json:"clientRequestId"`
		NotificationIDs []string `json:"notificationIds"`
	}
	if err := socialBodyV2(w, r, &input, 32<<10); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	result, err := s.Store.ReadNotificationsV2(r.Context(), u.ID, input.ClientRequestID, input.NotificationIDs)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) socialPreferencesV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	result, err := s.Store.NotificationPreferencesV2(r.Context(), u.ID)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) socialPatchPreferencesV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	var input store.SocialPreferencesPatch
	if err := body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	result, err := s.Store.PatchNotificationPreferencesV2(r.Context(), u.ID, input)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) socialAnnouncementsV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.socialUserV2(w, r); !ok {
		return
	}
	s.socialAnnouncementListV2(w, r, false)
}
func (s *Server) socialAdminAnnouncementsV2(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "announcements.manage"); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	s.socialAnnouncementListV2(w, r, true)
}
func (s *Server) socialAnnouncementListV2(w http.ResponseWriter, r *http.Request, admin bool) {
	scope := "announcements"
	if admin {
		scope = "admin-announcements"
	}
	limit, cursor, err := socialPageV2(r, scope)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	values, err := s.Store.AnnouncementsV2(r.Context(), admin, cursor.Before, limit+1)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var next any
	more := len(values) > limit
	if more {
		values = values[:limit]
		next = encodeSocialCursorV2(socialCursorV2{Scope: scope, Before: values[len(values)-1].Sequence})
	}
	v2List(w, r, values, next, more)
}
func (s *Server) socialAnnouncementV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.socialUserV2(w, r); !ok {
		return
	}
	s.socialAnnouncementViewV2(w, r, false)
}
func (s *Server) socialAdminAnnouncementV2(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "announcements.manage"); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	s.socialAnnouncementViewV2(w, r, true)
}
func (s *Server) socialAnnouncementViewV2(w http.ResponseWriter, r *http.Request, admin bool) {
	value, err := s.Store.AnnouncementV2(r.Context(), r.PathValue("announcementId"), admin)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, value)
}
func (s *Server) socialWriteAnnouncementV2(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "announcements.manage")
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var input store.SocialAnnouncementWrite
	if err = socialBodyV2(w, r, &input, 512<<10); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	result, err := s.Store.WriteAnnouncementV2(r.Context(), u.ID, r.PathValue("announcementId"), input)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) socialPublishAnnouncementV2(w http.ResponseWriter, r *http.Request) {
	s.socialChangeAnnouncementV2(w, r, true)
}
func (s *Server) socialUnpublishAnnouncementV2(w http.ResponseWriter, r *http.Request) {
	s.socialChangeAnnouncementV2(w, r, false)
}
func (s *Server) socialChangeAnnouncementV2(w http.ResponseWriter, r *http.Request, publish bool) {
	u, err := s.admin(r, "announcements.manage")
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	var input struct {
		ClientRequestID string `json:"clientRequestId"`
		ExpectedVersion int64  `json:"expectedVersion"`
		Reason          string `json:"reason,omitempty"`
	}
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	value, err := s.Store.ChangeAnnouncementV2(r.Context(), u.ID, r.PathValue("announcementId"), input.ClientRequestID, input.ExpectedVersion, publish, input.Reason)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	s.Hub.Wake()
	v2Success(w, r, value)
}

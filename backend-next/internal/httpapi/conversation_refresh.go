package httpapi

import (
	"context"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"log/slog"
	"net/http"
)

func (s *Server) notifyConversation(ctx context.Context, id string) {
	lo, hi, err := s.Store.ConversationUsers(ctx, id)
	if err != nil {
		// Durable messages still recover on reconnect/low-frequency reconciliation.
		slog.Warn("conversation wakeup deferred", "error", err)
		return
	}
	s.Hub.Wake(bridge.ConversationTopic(lo, id), bridge.ConversationTopic(hi, id))
}
func (s *Server) socialConversationV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.socialUserV2(w, r)
	if !ok {
		return
	}
	value, err := s.Store.ConversationV2(r.Context(), u.ID, r.PathValue("conversationId"))
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, value)
}

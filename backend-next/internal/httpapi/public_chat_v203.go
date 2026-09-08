package httpapi

import (
	"context"
	"encoding/json"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
)

// Sending does not depend on completion of the WebSocket handshake or history reads.
// Both transports share the same authorization, rate limit and durable message key.
func (s *Server) publicSendHTTPV203(w http.ResponseWriter, r *http.Request) {
	session, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	var input store.SocialSendRequest
	if err := socialBodyV2(w, r, &input, 4096); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	payload, err := json.Marshal(input)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v2Success(w, r, s.sendChat(r.Context(), session.TokenHash, payload))
}

func (s *Server) acceptedChatV203(ctx context.Context, clientID, id string) map[string]any {
	result := map[string]any{"clientMessageId": clientID, "status": "accepted", "messageId": id}
	if message, err := s.Store.PublicChatMessageV203(ctx, id); err == nil {
		result["message"] = message
	}
	// A read failure cannot undo a committed acceptance; older clients can fetch history.
	return result
}

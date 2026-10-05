package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var dmMessageID = regexp.MustCompile(`^mail_[A-Za-z0-9_-]{1,59}@deuterium-notifications$`)

// EventBridge's HTTP target sends the configured token in this header. Never
// accept a browser session as authority to mark someone else's email invalid.
func (s *Server) aliyunMailEvent(w http.ResponseWriter, r *http.Request) {
	if s.Config.DMEventToken == "" {
		failure(w, r, 503, "MAIL_EVENTS_DISABLED", "邮件回执尚未配置。")
		return
	}
	want := sha256.Sum256([]byte(s.Config.DMEventToken))
	got := sha256.Sum256([]byte(r.Header.Get("x-eventbridge-signature-token")))
	if subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
		failure(w, r, 401, "UNAUTHORIZED", "回执鉴权失败。")
		return
	}
	var event struct {
		Source string    `json:"source"`
		Type   string    `json:"type"`
		Time   time.Time `json:"time"`
		Data   struct {
			Status    json.RawMessage `json:"status"`
			Recipient string          `json:"rcpt"`
			MessageID string          `json:"msg_id"`
		} `json:"data"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	if err := d.Decode(&event); err != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	status := strings.TrimSpace(string(event.Data.Status))
	if event.Source != "acs:dm" || event.Type != "dm:Deliver:Fail" || (status != `"2"` && status != "2") {
		success(w, r, map[string]bool{"recorded": false})
		return
	}
	id := strings.Trim(strings.TrimSpace(event.Data.MessageID), "<>")
	if !dmMessageID.MatchString(id) {
		success(w, r, map[string]bool{"recorded": false})
		return
	}
	id = strings.TrimSuffix(id, "@deuterium-notifications")
	recorded, err := s.Store.RecordInvalidEmail(r.Context(), id, event.Data.Recipient, "DM_EVENT", event.Time)
	if errors.Is(err, store.ErrSocialInvalid) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]bool{"recorded": recorded})
}

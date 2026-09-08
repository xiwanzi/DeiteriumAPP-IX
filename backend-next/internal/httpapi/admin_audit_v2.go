package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerAdminAuditV2(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/audit-events", s.adminAuditEventsV2)
}

type adminAuditCursorV2 struct {
	UserID string                   `json:"user"`
	Filter store.AdminAuditFilterV2 `json:"filter"`
}

func encodeAdminAuditCursorV2(v adminAuditCursorV2) string {
	raw, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func adminAuditQueryV2(r *http.Request, user string, now time.Time) (int, store.AdminAuditFilterV2, error) {
	limit := 20
	f := store.AdminAuditFilterV2{}
	invalid := func() (int, store.AdminAuditFilterV2, error) { return 0, f, store.ErrSocialInvalid }
	q := r.URL.Query()
	for key, values := range q {
		if len(values) != 1 {
			return invalid()
		}
		switch key {
		case "actorId", "action", "resourceId", "from", "to", "limit", "cursor":
		default:
			return invalid()
		}
	}
	if q.Has("limit") {
		v, err := strconv.Atoi(q.Get("limit"))
		if err != nil || v < 1 || v > 100 {
			return invalid()
		}
		limit = v
	}
	if raw := q.Get("cursor"); raw != "" {
		if len(raw) > 4096 {
			return invalid()
		}
		data, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil {
			return invalid()
		}
		var cursor adminAuditCursorV2
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.DisallowUnknownFields()
		if err = dec.Decode(&cursor); err != nil || cursor.UserID != user || cursor.Filter.BeforeSequence <= 0 {
			return invalid()
		}
		if err = dec.Decode(new(any)); err != io.EOF {
			return invalid()
		}
		f = cursor.Filter
	} else {
		f.To = now.UTC().Truncate(time.Microsecond)
		f.From = f.To.Add(-30 * 24 * time.Hour)
	}
	for _, v := range []struct {
		key   string
		value *string
	}{{"actorId", &f.ActorID}, {"action", &f.Action}, {"resourceId", &f.ResourceID}} {
		if q.Has(v.key) {
			next := q.Get(v.key)
			if f.BeforeSequence > 0 && next != *v.value {
				return invalid()
			}
			*v.value = next
		}
	}
	for _, v := range []struct {
		key   string
		value *time.Time
	}{{"to", &f.To}, {"from", &f.From}} {
		if q.Has(v.key) {
			next, err := time.Parse(time.RFC3339Nano, q.Get(v.key))
			if err != nil || (f.BeforeSequence > 0 && !next.Equal(*v.value)) {
				return invalid()
			}
			*v.value = next.UTC()
		}
	}
	if f.BeforeSequence == 0 && q.Has("to") && !q.Has("from") {
		f.From = f.To.Add(-30 * 24 * time.Hour)
	}
	if !f.Valid() || f.To.After(now) {
		return invalid()
	}
	return limit, f, nil
}

func (s *Server) adminAuditEventsV2(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "audit.read")
	if err != nil {
		failError(w, r, err)
		return
	}
	limit, filter, err := adminAuditQueryV2(r, u.ID, time.Now())
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	events, snapshot, err := s.Store.AuditEventsV2(r.Context(), filter, limit)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	hasMore := len(events) > limit
	var next any
	if hasMore {
		events = events[:limit]
		last := events[len(events)-1]
		filter.SnapshotSequence = snapshot
		filter.BeforeTime = last.CreatedAt
		filter.BeforeSequence, _ = strconv.ParseInt(last.EventID, 10, 64)
		next = encodeAdminAuditCursorV2(adminAuditCursorV2{UserID: u.ID, Filter: filter})
	}
	v2List(w, r, events, next, hasMore)
}

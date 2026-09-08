package store

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type AdminAuditEventV2 struct {
	EventID    string    `json:"eventId"`
	ActorID    string    `json:"actorId"`
	Action     string    `json:"action"`
	ResourceID string    `json:"resourceId"`
	CreatedAt  time.Time `json:"createdAt"`
}

type AdminAuditFilterV2 struct {
	ActorID          string    `json:"actorId"`
	Action           string    `json:"action"`
	ResourceID       string    `json:"resourceId"`
	From             time.Time `json:"from"`
	To               time.Time `json:"to"`
	SnapshotSequence int64     `json:"snapshot"`
	BeforeTime       time.Time `json:"beforeTime"`
	BeforeSequence   int64     `json:"beforeSequence"`
}

func (f AdminAuditFilterV2) Valid() bool {
	for i, value := range []string{f.ActorID, f.Action, f.ResourceID} {
		max := 64
		if i == 2 {
			max = 128
		}
		if len(value) > max || strings.TrimSpace(value) != value {
			return false
		}
		for _, c := range value {
			if c < 33 || c > 126 {
				return false
			}
		}
	}
	if f.From.IsZero() || f.To.IsZero() || !f.To.After(f.From) || f.To.Sub(f.From) > 90*24*time.Hour || f.SnapshotSequence < 0 || f.BeforeSequence < 0 {
		return false
	}
	if f.BeforeSequence > 0 && (f.SnapshotSequence < f.BeforeSequence || f.BeforeTime.Before(f.From) || !f.BeforeTime.Before(f.To)) {
		return false
	}
	return true
}

// AuditEventsV2 reads only the four existing audit columns. It never joins account
// credentials or chat tables. The frozen maximum excludes writes between pages.
func (s *Store) AuditEventsV2(ctx context.Context, filter AdminAuditFilterV2, limit int) ([]AdminAuditEventV2, int64, error) {
	if !filter.Valid() || limit < 1 || limit > 100 {
		return nil, 0, ErrSocialInvalid
	}
	if filter.BeforeSequence == 0 {
		if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_id),0) FROM audit_events_next`).Scan(&filter.SnapshotSequence); err != nil {
			return nil, 0, err
		}
	}
	query := `SELECT sequence_id,actor_id,action,resource_id,created_at FROM audit_events_next WHERE created_at>=? AND created_at<? AND sequence_id<=?`
	args := []any{filter.From.UTC(), filter.To.UTC(), filter.SnapshotSequence}
	for _, v := range []struct{ column, value string }{{"actor_id", filter.ActorID}, {"action", filter.Action}, {"resource_id", filter.ResourceID}} {
		if v.value != "" {
			query += " AND " + v.column + "=BINARY ?"
			args = append(args, v.value)
		}
	}
	if filter.BeforeSequence > 0 {
		query += ` AND (created_at<? OR (created_at=? AND sequence_id<?))`
		args = append(args, filter.BeforeTime.UTC(), filter.BeforeTime.UTC(), filter.BeforeSequence)
	}
	query += ` ORDER BY created_at DESC,sequence_id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	events := []AdminAuditEventV2{}
	for rows.Next() {
		var v AdminAuditEventV2
		var sequence int64
		if err = rows.Scan(&sequence, &v.ActorID, &v.Action, &v.ResourceID, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		v.EventID = strconv.FormatInt(sequence, 10)
		events = append(events, v)
	}
	return events, filter.SnapshotSequence, rows.Err()
}

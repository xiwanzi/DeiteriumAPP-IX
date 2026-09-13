package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type DesktopApplication struct {
	Revision    int64           `json:"revision"`
	Version     string          `json:"version"`
	VersionCode int64           `json:"versionCode"`
	Release     json.RawMessage `json:"release"`
	PublishedAt *time.Time      `json:"publishedAt"`
}

func (s *Store) DesktopApplication(ctx context.Context) (DesktopApplication, error) {
	var value DesktopApplication
	var raw sql.NullString
	var at sql.NullTime
	err := s.DB.QueryRowContext(ctx, "SELECT revision,version_name,version_code,envelope_json,published_at FROM desktop_launcher_application WHERE id=1").Scan(&value.Revision, &value.Version, &value.VersionCode, &raw, &at)
	if raw.Valid {
		value.Release = json.RawMessage(raw.String)
	}
	if at.Valid {
		value.PublishedAt = &at.Time
	}
	return value, err
}

func (s *Store) PublishDesktopApplication(ctx context.Context, actor, request string, expected int64, name string, code int64, envelope json.RawMessage) (json.RawMessage, error) {
	if expected < 0 || code < 1 || len(name) > 32 || len(envelope) > 64000 || !json.Valid(envelope) {
		return nil, ErrSocialInvalid
	}
	input := struct {
		Expected int64
		Version  string
		Code     int64
		Envelope json.RawMessage
	}{expected, name, code, envelope}
	return s.socialMutate(ctx, actor, "desktop-launcher.application", request, input, func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		var revision, previous int64
		if err := tx.QueryRowContext(ctx, "SELECT revision,version_code FROM desktop_launcher_application WHERE id=1 FOR UPDATE").Scan(&revision, &previous); err != nil {
			return nil, err
		}
		if revision != expected {
			return nil, ErrSocialVersion
		}
		if code <= previous {
			return nil, ErrConflict
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		if _, err := tx.ExecContext(ctx, "UPDATE desktop_launcher_application SET revision=revision+1,version_name=?,version_code=?,envelope_json=?,published_at=? WHERE id=1", name, code, string(envelope), now); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,?)", actor, "desktop-launcher.application", name, now); err != nil {
			return nil, err
		}
		return DesktopApplication{Revision: revision + 1, Version: name, VersionCode: code, Release: envelope, PublishedAt: &now}, nil
	})
}

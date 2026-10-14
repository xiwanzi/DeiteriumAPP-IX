package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode"
)

type sessionDeviceKey struct{}

func WithSessionDevice(ctx context.Context, device string) context.Context {
	device = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, device))
	chars := []rune(device)
	if len(chars) > 100 {
		device = string(chars[:100])
	}
	return context.WithValue(ctx, sessionDeviceKey{}, device)
}

type DeviceSession struct {
	ID         string     `json:"sessionId"`
	ClientKind string     `json:"clientKind"`
	DeviceName string     `json:"deviceName"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	Current    bool       `json:"current"`
}

func (s *Store) DeviceSessions(ctx context.Context, user, currentHash string) ([]DeviceSession, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT session_id,client_kind,device_name,created_at,COALESCE(last_seen_at,created_at),expires_at,token_hash=?
	 FROM identity_sessions WHERE user_id=? AND client_kind IN ('app','web') AND revoked_at IS NULL
	 AND (expires_at>UTC_TIMESTAMP(6) OR (expires_at IS NULL AND client_kind='app')) ORDER BY (token_hash=?) DESC,COALESCE(last_seen_at,created_at) DESC LIMIT 200`, currentHash, user, currentHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DeviceSession{}
	for rows.Next() {
		var v DeviceSession
		var expiry sql.NullTime
		if err = rows.Scan(&v.ID, &v.ClientKind, &v.DeviceName, &v.CreatedAt, &v.LastSeenAt, &expiry, &v.Current); err != nil {
			return nil, err
		}
		if expiry.Valid {
			v.ExpiresAt = &expiry.Time
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) RevokeDeviceSessions(ctx context.Context, user, currentHash, target string, others bool) error {
	tx, err := s.beginAccountTx(ctx, user)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the account first, matching login/change-password order, then recheck
	// the caller so a concurrently revoked token cannot manage other sessions.
	var id string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM identities WHERE id=? AND status='active' FOR UPDATE", user).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUnauthorized
		}
		return err
	}
	err = tx.QueryRowContext(ctx, `SELECT user_id FROM identity_sessions WHERE token_hash=? AND user_id=? AND client_kind IN ('app','web') AND revoked_at IS NULL AND (expires_at>UTC_TIMESTAMP(6) OR (expires_at IS NULL AND client_kind='app')) FOR UPDATE`, currentHash, user).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if err != nil {
		return err
	}
	var updated sql.Result
	if others {
		updated, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE user_id=? AND token_hash<>? AND client_kind IN ('app','web') AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>UTC_TIMESTAMP(6))", user, currentHash)
	} else {
		var owner string
		err = tx.QueryRowContext(ctx, "SELECT user_id FROM identity_sessions WHERE session_id=? AND user_id=? AND client_kind IN ('app','web') FOR UPDATE", target, user).Scan(&owner)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSocialNotFound
		}
		if err == nil {
			updated, err = tx.ExecContext(ctx, "UPDATE identity_sessions SET revoked_at=COALESCE(revoked_at,UTC_TIMESTAMP(6)) WHERE session_id=? AND user_id=?", target, user)
		}
	}
	if err != nil {
		return err
	}
	changed, err := updated.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return tx.Commit()
	}
	action, resource := "session.revoke", target
	if others {
		action, resource = "session.revoke-others", user
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", user, action, resource); err != nil {
		return err
	}
	return tx.Commit()
}

// Activity writes are at most once per five minutes per session, not on every
// chat poll. The conditional update handles concurrent requests without a timer.
func (s *Store) TouchSession(ctx context.Context, v Session) {
	if v.Kind == "gamechat" || time.Since(v.LastSeenAt) < 5*time.Minute {
		return
	}
	_, _ = s.DB.ExecContext(ctx, `UPDATE identity_sessions SET last_seen_at=UTC_TIMESTAMP(6) WHERE token_hash=? AND revoked_at IS NULL AND (last_seen_at IS NULL OR last_seen_at<DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 5 MINUTE))`, v.TokenHash)
}

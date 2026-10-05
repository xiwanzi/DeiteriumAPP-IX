package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrEmailAddressInvalid = errors.New("邮箱地址无效")

func checkEmailAddress(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, email string) error {
	var blocked bool
	if err := q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM mail_invalid_addresses WHERE email=? AND blocked_until>UTC_TIMESTAMP(6))", email).Scan(&blocked); err != nil {
		return err
	}
	if blocked {
		return ErrEmailAddressInvalid
	}
	return nil
}

func (s *Store) CheckEmailAddress(ctx context.Context, email string) error {
	email, err := NormalizeEmail(email)
	if err != nil {
		return err
	}
	return checkEmailAddress(ctx, s.DB, email)
}

// Match the provider report to our actual DM outbox before accepting it. A
// report for the same message never extends its original 24-hour deadline.
func (s *Store) RecordInvalidEmail(ctx context.Context, eventID, email, source string, occurredAt time.Time) (bool, error) {
	email, err := NormalizeEmail(email)
	if err != nil || !catalogEnum(source, "SMTP", "DM_EVENT") || occurredAt.IsZero() {
		return false, ErrSocialInvalid
	}
	now := time.Now().UTC()
	if occurredAt.After(now.Add(5 * time.Minute)) {
		return false, ErrSocialInvalid
	}
	if occurredAt.After(now) {
		occurredAt = now
	}
	until := occurredAt.UTC().Add(24 * time.Hour)
	if !until.After(now) {
		return false, nil
	}
	tx, err := s.beginAccountTx(ctx, "")
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Same lock order as NewEmailVerification: mailbox first, then outbox.
	key := Digest([]byte("email:" + email))
	if _, err = tx.ExecContext(ctx, "INSERT INTO game_verification_cooldowns VALUES(?,TIMESTAMP('2000-01-01')) ON DUPLICATE KEY UPDATE cooldown_key=VALUES(cooldown_key)", key); err != nil {
		return false, err
	}
	var recipient, channel, kind string
	var attempts int
	var created time.Time
	err = tx.QueryRowContext(ctx, "SELECT recipient,channel_id,kind,attempts,created_at FROM mail_outbox WHERE event_id=? FOR UPDATE", eventID).Scan(&recipient, &channel, &kind, &attempts, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(recipient, email) || channel != "DM" || attempts == 0 || occurredAt.Before(created.Add(-time.Minute)) {
		return false, nil
	}
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO mail_invalid_addresses(event_id,email,source,reported_at,blocked_until) VALUES(?,?,?,?,?)", eventID, email, source, occurredAt.UTC(), until); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE mail_outbox SET status='FAILED',last_error='邮箱地址无效',lease_until=NULL,payload_cipher=IF(kind LIKE 'CODE_%','',payload_cipher) WHERE event_id=?", eventID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

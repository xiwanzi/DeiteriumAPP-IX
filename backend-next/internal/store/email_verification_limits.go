package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// This read avoids an external lookup during an existing send cooldown.
// NewEmailVerification repeats the checks under locks before committing mail.
func (s *Store) CheckEmailIssueLimits(ctx context.Context, email string) error {
	var next time.Time
	err := s.DB.QueryRowContext(ctx, "SELECT expires_at FROM game_verification_cooldowns WHERE cooldown_key=?", Digest([]byte("email:"+email))).Scan(&next)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if next.After(time.Now().UTC()) {
		return rateLimitedUntil(next).withScope("RECIPIENT")
	}
	var count int
	var earliest sql.NullTime
	if err = s.DB.QueryRowContext(ctx, "SELECT COUNT(*),MIN(created_at) FROM email_verifications WHERE email=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)", email).Scan(&count, &earliest); err != nil {
		return err
	}
	if count >= 20 && earliest.Valid {
		return rateLimitedUntil(earliest.Time.Add(24 * time.Hour)).withScope("RECIPIENT")
	}
	return nil
}

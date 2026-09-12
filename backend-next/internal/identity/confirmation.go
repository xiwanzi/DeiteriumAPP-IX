package identity

import (
	"context"
	"database/sql"
	"errors"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var ErrConfirmationPassword = errors.New("current administrator password is incorrect")

// Verify the authenticated principal, never an account name supplied in a form.
// The returned credential is checked again inside the destructive transaction.
func (s *Service) ConfirmPassword(ctx context.Context, userID, password, ip string) (string, error) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return "", ErrBusy
	}
	accountKey := store.Digest([]byte("confirmation:user:" + userID))
	ipKey := store.Digest([]byte("confirmation:ip:" + ip))
	if err := s.Store.ReserveLogin(ctx, accountKey, ipKey); err != nil {
		return "", err
	}
	var hash, status string
	err := s.Store.DB.QueryRowContext(ctx, "SELECT password_hash,status FROM identities WHERE id=?", userID).Scan(&hash, &status)
	if errors.Is(err, sql.ErrNoRows) || err == nil && status != "active" {
		return "", store.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 64 || len(password) > 256 || !Verify(hash, password) {
		return "", ErrConfirmationPassword
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	_ = s.Store.ClearAccountBudget(ctx, accountKey)
	return hash, nil
}

package identity

import (
	"context"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func SessionExpiry(kind string, now time.Time) time.Time {
	if kind == "web" {
		return now.UTC().Add(30 * 24 * time.Hour)
	}
	return time.Time{}
}

// Registration and login share token generation and the same lifecycle policy.
func (s *Service) IssueSession(ctx context.Context, user store.User, kind string) (string, store.Session, error) {
	return s.issueSession(ctx, user, kind, "")
}

func (s *Service) issueSession(ctx context.Context, user store.User, kind, newHash string) (string, store.Session, error) {
	if kind != "app" && kind != "web" {
		return "", store.Session{}, store.ErrUnauthorized
	}
	start := time.Now()
	defer s.sessionTiming.observe(start)
	token, csrf := Secret(), Secret()
	session := store.Session{User: user, TokenHash: store.Digest([]byte(token)), Kind: kind, CSRF: csrf, ExpiresAt: SessionExpiry(kind, start)}
	if err := s.Store.CreateSession(ctx, user, session.TokenHash, kind, csrf, session.ExpiresAt, newHash); err != nil {
		return "", store.Session{}, err
	}
	return token, session, nil
}

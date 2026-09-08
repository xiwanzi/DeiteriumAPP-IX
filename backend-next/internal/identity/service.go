package identity

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var ErrBusy = errors.New("password verification capacity exhausted")

func (s *Service) HashPassword(ctx context.Context, password string) (string, error) {
	if utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 64 || len(password) > 256 {
		return "", store.ErrUnauthorized
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return "", ErrBusy
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return Hash(password), nil
}

type Service struct {
	Store     *store.Store
	slots     chan struct{}
	dummyHash string
}

func New(s *store.Store) *Service {
	// Public synthetic hash, never an account credential. Unknown accounts still
	// execute the same bounded KDF, without allocating 64 MiB just to start up.
	const dummy = "$argon2id$v=19$m=65536,t=3,p=2$2vFuEp4//iUeRq5II53mPw$qX0Vqkwxq2VJxZUjXYGv5D7meMQzHQ4V1cGb/RIMjBg"
	return &Service{Store: s, slots: make(chan struct{}, 2), dummyHash: dummy}
}

func (s *Service) Login(ctx context.Context, account, password, ip, kind string) (token string, session store.Session, err error) {
	account = strings.ToLower(strings.TrimSpace(account))
	if SystemGameID(account) {
		return "", session, store.ErrUnauthorized
	}
	if (!gamePattern.MatchString(account) && !qqPattern.MatchString(account)) || utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 64 || len(password) > 256 || (kind != "app" && kind != "web") {
		return "", session, store.ErrUnauthorized
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return "", session, ErrBusy
	}
	accountKey := store.Digest([]byte("account:" + account))
	ipKey := store.Digest([]byte("ip:" + ip))
	if err = s.Store.ReserveLogin(ctx, accountKey, ipKey); err != nil {
		return
	}
	user, lookupErr := s.Store.UserByAlias(ctx, account)
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return "", session, lookupErr
	}
	userBudgetKey := store.Digest([]byte("identity:" + user.ID))
	if lookupErr == nil {
		if err = s.Store.ReserveIdentityLogin(ctx, userBudgetKey); err != nil {
			return
		}
	}
	hash := user.PasswordHash
	if lookupErr != nil {
		hash = s.dummyHash
	}
	valid := Verify(hash, password)
	if !valid || lookupErr != nil || user.Status != "active" {
		return "", session, store.ErrUnauthorized
	}
	if err = ctx.Err(); err != nil {
		return
	}
	newHash := ""
	if NeedsUpgrade(hash) {
		newHash = Hash(password)
	}
	token = Secret()
	csrf := Secret()
	expiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	err = s.Store.CreateSession(ctx, user, store.Digest([]byte(token)), kind, csrf, expiry, newHash)
	if errors.Is(err, store.ErrCredentialChanged) {
		// A concurrent first login may have upgraded the same legacy password.
		// Reverify once against the current hash: never bypass a real password reset.
		current, readErr := s.Store.UserByAlias(ctx, account)
		if readErr != nil || current.ID != user.ID || current.Status != "active" || !Verify(current.PasswordHash, password) {
			return "", session, store.ErrUnauthorized
		}
		user = current
		newHash = ""
		if NeedsUpgrade(current.PasswordHash) {
			newHash = Hash(password)
		}
		err = s.Store.CreateSession(ctx, user, store.Digest([]byte(token)), kind, csrf, expiry, newHash)
		if errors.Is(err, store.ErrCredentialChanged) {
			err = store.ErrUnauthorized
		}
	}
	if err != nil {
		return "", session, err
	}
	// Clearing the account budget never clears the source IP budget.
	_ = s.Store.ClearAccountBudget(ctx, accountKey)
	_ = s.Store.ClearAccountBudget(ctx, userBudgetKey)
	session = store.Session{User: user, TokenHash: store.Digest([]byte(token)), Kind: kind, CSRF: csrf, ExpiresAt: expiry}
	return
}

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

var ErrNameLookupUnavailable = errors.New("official name lookup unavailable")
var ErrNameLookupDisabled = errors.New("official name lookup disabled for isolated tests")

var ErrBusy = errors.New("password verification capacity exhausted")

func (s *Service) HashPassword(ctx context.Context, password string) (string, error) {
	if utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 64 || len(password) > 256 {
		return "", store.ErrUnauthorized
	}
	if err := s.acquirePasswordSlot(ctx); err != nil {
		return "", err
	}
	defer func() { <-s.slots }()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return Hash(password), nil
}

type Service struct {
	GameNameResolver func(context.Context, string) (store.AdmissionProfile, error)
	SyncGameName     func(context.Context, store.AdmissionProfile, time.Time) error
	Store            *store.Store
	slots            chan struct{}
	waiting          chan struct{}
	nameLookups      chan struct{}
	dummyHash        string
}

func New(s *store.Store) *Service {
	// Public synthetic hash, never an account credential. Unknown accounts still
	// execute the same bounded KDF, without allocating 64 MiB just to start up.
	const dummy = "$argon2id$v=19$m=65536,t=3,p=2$2vFuEp4//iUeRq5II53mPw$qX0Vqkwxq2VJxZUjXYGv5D7meMQzHQ4V1cGb/RIMjBg"
	return &Service{Store: s, slots: make(chan struct{}, 2), waiting: make(chan struct{}, 32), nameLookups: make(chan struct{}, 8), dummyHash: dummy}
}

func (s *Service) Login(ctx context.Context, account, password, ip, kind string) (token string, session store.Session, err error) {
	account = strings.ToLower(strings.TrimSpace(account))
	if SystemGameID(account) {
		return "", session, store.ErrUnauthorized
	}
	_, emailErr := store.NormalizeEmail(account)
	if (!gamePattern.MatchString(account) && !qqPattern.MatchString(account) && emailErr != nil) || utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 64 || len(password) > 256 || (kind != "app" && kind != "web") {
		return "", session, store.ErrUnauthorized
	}
	accountKey := store.Digest([]byte("account:" + account))
	ipKey := store.Digest([]byte("ip:" + ip))
	if err = s.Store.ReserveLogin(ctx, accountKey, ipKey); err != nil {
		return
	}
	user, lookupErr := s.Store.UserByAlias(ctx, account)
	var canonical *store.AdmissionProfile
	var nameObservedAt time.Time
	var nameLookupErr error
	gameAlias := lookupErr == nil && user.GameVerified && strings.EqualFold(user.GameID, account) && account != user.QQ && !strings.EqualFold(account, user.Email)
	if s.GameNameResolver != nil && store.ValidAdmissionName(account) && (errors.Is(lookupErr, sql.ErrNoRows) || gameAlias) {
		nameObservedAt = time.Now().UTC()
		profile, resolveErr := s.resolveGameName(ctx, account)
		switch {
		case errors.Is(resolveErr, ErrNameLookupDisabled):
		case errors.Is(resolveErr, store.ErrSocialNotFound):
			user = store.User{}
			lookupErr = sql.ErrNoRows
		case resolveErr != nil || !store.ValidAdmissionUUID(profile.UUID) || !strings.EqualFold(profile.Name, account):
			nameLookupErr = ErrNameLookupUnavailable
		default:
			found, findErr := s.Store.UserByUUID(ctx, profile.UUID)
			if findErr == nil {
				user = found
				lookupErr = nil
				canonical = &profile
			} else if errors.Is(findErr, sql.ErrNoRows) {
				user = store.User{}
				lookupErr = sql.ErrNoRows
			} else {
				return "", session, findErr
			}
		}
	}

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
	valid, verifyErr := s.verifyPassword(ctx, hash, password)
	if verifyErr != nil {
		return "", session, verifyErr
	}
	if nameLookupErr != nil {
		return "", session, nameLookupErr
	}
	if !valid || lookupErr != nil || user.Status != "active" {
		return "", session, store.ErrUnauthorized
	}
	if err = ctx.Err(); err != nil {
		return
	}
	if canonical != nil && s.SyncGameName != nil {
		if err = s.SyncGameName(ctx, *canonical, nameObservedAt); err != nil {
			return "", session, err
		}
		refreshed, refreshErr := s.Store.UserByUUID(ctx, user.ServerUUID)
		if refreshErr != nil {
			return "", session, refreshErr
		}
		if refreshed.ID != user.ID || refreshed.PasswordHash != user.PasswordHash || refreshed.Status != "active" {
			return "", session, store.ErrUnauthorized
		}
		user = refreshed
	}
	newHash := ""
	if NeedsUpgrade(hash) {
		if newHash, err = s.HashPassword(ctx, password); err != nil {
			return "", session, err
		}
	}
	token = Secret()
	csrf := Secret()
	expiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	err = s.Store.CreateSession(ctx, user, store.Digest([]byte(token)), kind, csrf, expiry, newHash)
	if errors.Is(err, store.ErrCredentialChanged) {
		// A concurrent first login may have upgraded the same legacy password.
		// Reverify once against the current hash: never bypass a real password reset.
		current, readErr := s.Store.UserByAlias(ctx, account)
		if readErr != nil || current.ID != user.ID || current.Status != "active" {
			return "", session, store.ErrUnauthorized
		}
		valid, verifyErr = s.verifyPassword(ctx, current.PasswordHash, password)
		if verifyErr != nil {
			return "", session, verifyErr
		}
		if !valid {
			return "", session, store.ErrUnauthorized
		}
		user = current
		newHash = ""
		if NeedsUpgrade(current.PasswordHash) {
			if newHash, err = s.HashPassword(ctx, password); err != nil {
				return "", session, err
			}
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

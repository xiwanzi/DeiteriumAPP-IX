package identity

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var ErrNameLookupUnavailable = errors.New("official name lookup unavailable")
var ErrNameLookupDisabled = errors.New("official name lookup disabled for isolated tests")
var ErrNameLookupBusy = errors.New("official name lookup capacity exhausted")

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
	start := time.Now()
	defer s.passwordTiming.observe(start)
	return Hash(password), nil
}

type Service struct {
	GameNameResolver func(context.Context, string) (store.AdmissionProfile, error)
	SyncGameName     func(context.Context, store.AdmissionProfile, time.Time) error
	// Queue-only callback: no network/database work on the login response path.
	RefreshGameName func(store.User)
	Store           *store.Store
	slots           chan struct{}
	waiting         chan struct{}
	dummyHash       string

	loginTiming         timing
	passwordQueueTiming timing
	passwordTiming      timing
	sessionTiming       timing
	passwordRejected    atomic.Uint64
	localLogins         atomic.Uint64
	nameLoginLookups    atomic.Uint64
}

func New(s *store.Store) *Service {
	// Public synthetic hash, never an account credential. Unknown accounts still
	// execute the same bounded KDF, without allocating 64 MiB just to start up.
	const dummy = "$argon2id$v=19$m=65536,t=3,p=2$2vFuEp4//iUeRq5II53mPw$qX0Vqkwxq2VJxZUjXYGv5D7meMQzHQ4V1cGb/RIMjBg"
	return &Service{Store: s, slots: make(chan struct{}, 2), waiting: make(chan struct{}, 128), dummyHash: dummy}
}

func (s *Service) Login(ctx context.Context, account, password, ip, kind string) (token string, session store.Session, err error) {
	start := time.Now()
	defer s.loginTiming.observe(start)
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
	accountReservation, reserveErr := s.Store.ReserveLoginAttempt(ctx, accountKey, ipKey)
	if reserveErr != nil {
		err = reserveErr
		return
	}
	reserved := []store.LoginReservation{accountReservation}
	credentialRejected := false
	defer func() {
		if !credentialRejected {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			defer cancel()
			// Busy, upstream failure and cancellation are not password failures.
			for _, reservation := range reserved {
				_ = s.Store.ReleaseLoginBudget(cleanup, reservation)
			}
		}
	}()
	user, lookupErr := s.Store.UserByAlias(ctx, account)
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return "", session, lookupErr
	}
	// A platform alias and its password authenticate the stable platform account.
	// Official lookup is a fallback for a new/reused name, not a prerequisite for
	// every successful login. Reuse each identity reservation/KDF within a request.
	type checkedCredential struct {
		hash  string
		valid bool
	}
	checked := map[string]checkedCredential{}
	budgeted := map[string]bool{}
	verifyUser := func(candidate store.User) (bool, error) {
		if !budgeted[candidate.ID] {
			reservation, e := s.Store.ReserveIdentityLogin(ctx, store.Digest([]byte("identity:"+candidate.ID)))
			if e != nil {
				return false, e
			}
			reserved = append(reserved, reservation)
			budgeted[candidate.ID] = true
		}
		if previous, ok := checked[candidate.ID]; ok && previous.hash == candidate.PasswordHash {
			return previous.valid && candidate.Status == "active", nil
		}
		valid, e := s.verifyPassword(ctx, candidate.PasswordHash, password)
		if e == nil {
			checked[candidate.ID] = checkedCredential{candidate.PasswordHash, valid}
		}
		return valid && candidate.Status == "active", e
	}
	var canonical *store.AdmissionProfile
	var nameObservedAt time.Time
	gameAlias := lookupErr == nil && (user.GameVerified || user.ProfileUUID != "") && strings.EqualFold(user.GameID, account) && account != user.QQ && !strings.EqualFold(account, user.Email)
	valid := false
	var verifyErr error
	if lookupErr == nil {
		valid, verifyErr = verifyUser(user)
		// Once local credentials were actually rejected, an upstream outage must
		// not erase that failed attempt. A successful fallback to the current
		// name owner releases all reservations below as usual.
		credentialRejected = verifyErr == nil && !valid
		if verifyErr != nil && (!gameAlias || !errors.Is(verifyErr, store.ErrRateLimited)) {
			return "", session, verifyErr
		}
	}
	if !valid && s.GameNameResolver != nil && store.ValidAdmissionName(account) && (errors.Is(lookupErr, sql.ErrNoRows) || gameAlias) {
		s.nameLoginLookups.Add(1)
		nameObservedAt = time.Now().UTC()
		profile, resolveErr := s.resolveGameName(ctx, account)
		if !profile.ObservedAt.IsZero() {
			nameObservedAt = profile.ObservedAt
		}
		switch {
		case errors.Is(resolveErr, ErrNameLookupDisabled):
		case errors.Is(resolveErr, ErrNameLookupBusy):
			return "", session, resolveErr
		case errors.Is(resolveErr, store.ErrSocialNotFound):
			user = store.User{}
			lookupErr = sql.ErrNoRows
		case resolveErr != nil || !store.ValidAdmissionUUID(profile.UUID) || !strings.EqualFold(profile.Name, account):
			return "", session, ErrNameLookupUnavailable
		default:
			found, findErr := s.Store.UserByProfileUUID(ctx, profile.UUID)
			if findErr == nil {
				user = found
				lookupErr = nil
				canonical = &profile
				valid, verifyErr = verifyUser(user)
				if verifyErr == nil && valid {
					credentialRejected = false
				}
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
	hash := user.PasswordHash
	if lookupErr != nil {
		hash = s.dummyHash
		if len(checked) == 0 {
			_, verifyErr = s.verifyPassword(ctx, hash, password)
		}
	}
	if verifyErr != nil {
		return "", session, verifyErr
	}
	if !valid || lookupErr != nil || user.Status != "active" {
		credentialRejected = true
		return "", session, store.ErrUnauthorized
	}
	if err = ctx.Err(); err != nil {
		return
	}
	if canonical != nil {
		if user.GameVerified {
			if s.SyncGameName != nil {
				err = s.SyncGameName(ctx, *canonical, nameObservedAt)
			}
		} else {
			err = s.Store.SyncRegisteredName(ctx, user, *canonical, nameObservedAt)
		}
		if err != nil {
			return "", session, err
		}
		refreshed, refreshErr := s.Store.UserByProfileUUID(ctx, canonical.UUID)
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
	ctx = store.WithLoginAlias(ctx, account)
	token, session, err = s.issueSession(ctx, user, kind, newHash)
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
		token, session, err = s.issueSession(ctx, user, kind, newHash)
		if errors.Is(err, store.ErrCredentialChanged) {
			err = store.ErrUnauthorized
		}
	}
	if err != nil {
		return "", session, err
	}
	if canonical == nil {
		s.localLogins.Add(1)
		if s.RefreshGameName != nil {
			s.RefreshGameName(session.User)
		}
	}
	// Release just this successful reservation, preserving concurrent failures.
	return
}

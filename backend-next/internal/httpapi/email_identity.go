package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func unverifiedAccountRoute(r *http.Request) bool {
	switch r.URL.Path {
	case "/api/v1/account/me", "/api/v1/account/logout", "/api/v1/web/session", "/api/v1/account/game-verification-code", "/api/v1/account/game-verification":
		return true
	}
	return r.URL.Path == "/api/v1/support/submissions"
}
func (s *Server) emailRegistrationCode(w http.ResponseWriter, r *http.Request) {
	if !s.publicAccountOrigin(r) {
		failError(w, r, ErrForbidden)
		return
	}
	var in struct {
		Email    string `json:"email"`
		GameID   string `json:"gameId"`
		Password string `json:"password,omitempty"` // ignored for older clients
	}
	if body(w, r, &in) != nil || !identity.ValidGameID(in.GameID) || identity.SystemGameID(in.GameID) {
		failure(w, r, 400, "GAME_ID_INVALID", "请填写有效的游戏 ID。")
		return
	}
	email, err := store.NormalizeEmail(in.Email)
	if err != nil {
		failure(w, r, 400, "EMAIL_INVALID", "请填写有效的邮箱地址。")
		return
	}
	for _, alias := range []string{email, strings.ToLower(in.GameID)} {
		_, err = s.Store.UserByAlias(r.Context(), alias)
		if err == nil {
			failure(w, r, 409, "ACCOUNT_ALREADY_EXISTS", "该邮箱或游戏 ID 已注册，请登录或找回密码。")
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			failError(w, r, err)
			return
		}
	}
	s.issueEmailCode(w, r, store.EmailVerification{Email: email, GameID: in.GameID, Purpose: "register"})
}
func (s *Server) issueEmailCode(w http.ResponseWriter, r *http.Request, v store.EmailVerification) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
		failError(w, r, err)
		return
	}
	channel, err := s.Store.MailChannel(ctx, "DM")
	if err != nil {
		failError(w, r, err)
		return
	}
	if !channel.Enabled || channel.Validate() != nil || len(s.Config.SMTPKey) != 32 {
		failure(w, r, 503, "MAIL_NOT_READY", "验证码邮件服务暂不可用，请稍后重试。")
		return
	}
	random, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		failError(w, r, err)
		return
	}
	code := fmt.Sprintf("%06d", random.Int64())
	token := identity.Secret()
	v.ID = store.ID("emailv_")
	v.TokenHash = store.Digest([]byte(token))
	v.CodeHash = store.Digest([]byte(token + ":" + code))
	v.ExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	if err = s.Store.NewEmailVerification(ctx, v, code, s.Config.SMTPKey); err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]any{"verificationToken": token, "expiresAt": v.ExpiresAt, "resendAfterSeconds": 60, "delivery": "EMAIL"})
}
func (s *Server) registerEmail(w http.ResponseWriter, r *http.Request, kind string) {
	if (kind == "web" && !s.origin(r)) || (kind == "app" && r.Header.Get("Origin") != "") {
		failError(w, r, ErrForbidden)
		return
	}
	var in struct {
		VerificationToken string `json:"verificationToken"`
		Code              string `json:"code"`
		Password          string `json:"password"`
	}
	if body(w, r, &in) != nil || !validCode(in.VerificationToken, in.Code) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if !validPassword(in.Password) {
		failure(w, r, 400, "PASSWORD_INVALID", "密码需要 8–64 位。")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	v, err := s.Store.CheckEmailVerification(ctx, store.Digest([]byte(in.VerificationToken)), store.Digest([]byte(in.VerificationToken+":"+in.Code)), "register")
	if err != nil {
		failError(w, r, err)
		return
	}
	hash, err := s.Identity.HashPassword(ctx, in.Password)
	if err != nil {
		failError(w, r, err)
		return
	}
	u, err := s.Store.RegisterEmailUser(ctx, v, hash)
	if err != nil {
		failError(w, r, err)
		return
	}
	s.newAccountSession(w, r, ctx, u, kind)
}
func (s *Server) newAccountSession(w http.ResponseWriter, r *http.Request, ctx context.Context, u store.User, kind string) {
	token, csrf := identity.Secret(), identity.Secret()
	expiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err := s.Store.CreateSession(ctx, u, store.Digest([]byte(token)), kind, csrf, expiry, ""); err != nil {
		failError(w, r, err)
		return
	}
	if kind == "web" {
		if old, e := r.Cookie(s.cookieName()); e == nil {
			if err := s.Store.Revoke(ctx, store.Digest([]byte(old.Value))); err != nil {
				_ = s.Store.Revoke(ctx, store.Digest([]byte(token)))
				failError(w, r, err)
				return
			}
		}
		s.setCookie(w, token, expiry)
		success(w, r, map[string]any{"user": u, "csrfToken": csrf, "expiresAt": expiry})
	} else {
		success(w, r, map[string]any{"token": token, "user": u})
	}
}
func (s *Server) gameVerificationCode(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	if session.User.GameVerified {
		failure(w, r, 409, "ALREADY_VERIFIED", "此账号已完成游戏身份验证。")
		return
	}
	var in struct {
		QQ string `json:"qq"`
	}
	if body(w, r, &in) != nil || !identity.ValidQQ(in.QQ) {
		failure(w, r, 400, "QQ_INVALID", "请输入有效的 QQ 号。")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err = s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
		failError(w, r, err)
		return
	}
	if other, err := s.Store.UserByAlias(ctx, in.QQ); err == nil && other.ID != session.User.ID {
		failure(w, r, 409, "IDENTITY_ALREADY_BOUND", "此 QQ 已绑定其他账号。")
		return
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		failError(w, r, err)
		return
	}
	p, err := s.resolveCorePlayer(ctx, session.User.GameID, true)
	if err != nil {
		failError(w, r, err)
		return
	}
	var count int
	if err = s.Store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities WHERE server_uuid=? AND id<>?", p.PlayerUUID, session.User.ID).Scan(&count); err != nil {
		failError(w, r, err)
		return
	}
	if count > 0 {
		failure(w, r, 409, "IDENTITY_ALREADY_BOUND", "该游戏身份已绑定其他账号，请联系支持人员。")
		return
	}
	s.issueGameCode(w, r, ctx, store.GameVerification{Purpose: "game_verify", PlayerUUID: p.PlayerUUID, GameID: p.GameID, QQ: in.QQ, UserID: session.User.ID, NodeID: p.ServerID})
}
func (s *Server) completeGameVerification(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	if session.User.GameVerified {
		success(w, r, map[string]any{"user": session.User})
		return
	}
	var in struct {
		VerificationToken string `json:"verificationToken"`
		Code              string `json:"code"`
	}
	if body(w, r, &in) != nil || !validCode(in.VerificationToken, in.Code) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	v, err := s.Store.CheckGameVerification(r.Context(), store.Digest([]byte(in.VerificationToken)), store.Digest([]byte(in.VerificationToken+":"+in.Code)), "game_verify")
	if err == nil {
		err = s.Store.BindGameIdentity(r.Context(), session.User.ID, v)
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	s.me(w, r)
}

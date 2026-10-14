package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func unverifiedAccountRoute(r *http.Request) bool {
	if r.Method == "POST" && r.URL.Path == "/api/v1/account/game-profile" {
		return true
	}
	if accountSessionRoute(r) {
		return true
	}
	switch r.URL.Path {
	case "/api/v1/account/me", "/api/v1/account/logout", "/api/v1/web/session", "/api/v1/account/game-verification-code", "/api/v1/account/game-verification", "/api/v1/account/password-change-code", "/api/v1/account/password-change":
		return true
	}
	return r.URL.Path == "/api/v1/support/submissions"
}

func (s *Server) passwordChangeCode(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	var in struct{}
	if err = body(w, r, &in); err != nil {
		failError(w, r, err)
		return
	}
	u := session.User
	if u.Email != "" {
		s.issueEmailCode(w, r, store.EmailVerification{Email: u.Email, GameID: u.GameID, UserID: u.ID, Purpose: "password_change"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err = s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
		failError(w, r, err)
		return
	}
	p, err := s.Core.Player("", u.ServerUUID)
	if err != nil {
		failError(w, r, err)
		return
	}
	if p.PlayerUUID == "" {
		failError(w, r, &CoreError{"PLAYER_OFFLINE", "请先使用已绑定的游戏账号进入服务器。"})
		return
	}
	s.issueGameCode(w, r, ctx, store.GameVerification{Purpose: "password_change", PlayerUUID: u.ServerUUID, GameID: p.GameID, QQ: u.QQ, UserID: u.ID, NodeID: p.ServerID})
}
func (s *Server) changeAccountPassword(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	var in struct {
		VerificationToken string `json:"verificationToken"`
		Code              string `json:"code"`
		NewPassword       string `json:"newPassword"`
	}
	if body(w, r, &in) != nil || !validCode(in.VerificationToken, in.Code) || !validPassword(in.NewPassword) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	token, code := store.Digest([]byte(in.VerificationToken)), store.Digest([]byte(in.VerificationToken+":"+in.Code))
	if session.User.Email != "" {
		v, e := s.Store.CheckEmailVerification(ctx, token, code, "password_change")
		if e != nil {
			failError(w, r, e)
			return
		}
		if v.UserID != session.User.ID {
			failError(w, r, store.ErrVerification)
			return
		}
		hash, e := s.Identity.HashPassword(ctx, in.NewPassword)
		if e == nil {
			e = s.Store.ResetEmailPassword(ctx, v, hash)
		}
		err = e
	} else {
		v, e := s.Store.CheckGameVerification(ctx, token, code, "password_change")
		if e != nil {
			failError(w, r, e)
			return
		}
		if v.UserID != session.User.ID {
			failError(w, r, store.ErrVerification)
			return
		}
		hash, e := s.Identity.HashPassword(ctx, in.NewPassword)
		if e == nil {
			e = s.Store.ResetGamePassword(ctx, v, hash)
		}
		err = e
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	if session.Kind == "web" {
		s.setCookie(w, "", time.Unix(1, 0))
	}
	s.Hub.Wake("user:" + session.User.ID)
	success(w, r, map[string]bool{"passwordChanged": true})
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
	v.ResendAt = time.Now().UTC().Add(60 * time.Second)
	v.IssueIP = s.remoteIP(r)
	if err := s.Store.CheckEmailAddress(ctx, v.Email); err != nil {
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
	if err = s.Store.CheckEmailIssueLimits(ctx, v.Email); err != nil {
		failError(w, r, err)
		return
	}
	if v.Purpose == "register" {
		var profile store.AdmissionProfile
		if profile, err = s.checkRegistrationProfile(ctx, v.GameID, v.IssueIP); err != nil {
			registrationProfileFailure(w, r, err)
			return
		}
		v.ProfileChecked = true
		v.ProfileUUID = profile.UUID
		if err = s.Store.CheckProfileReservation(ctx, profile.UUID, v.Email); err != nil {
			catalogFailV2(w, r, err)
			return
		}
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
	success(w, r, map[string]any{"verificationToken": token, "expiresAt": v.ExpiresAt, "resendAfterSeconds": max(1, int(math.Ceil(time.Until(v.ResendAt).Seconds()))), "delivery": "EMAIL"})
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
		DeviceName        string `json:"deviceName,omitempty"`
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
	// The verification record owns the requested ID. A public profile confirms
	// existence only; game ownership remains a separate verification step.
	if v.ProfileUUID == "" {
		var profile store.AdmissionProfile
		if profile, err = s.checkRegistrationProfile(ctx, v.GameID, s.remoteIP(r)); err != nil {
			registrationProfileFailure(w, r, err)
			return
		}
		v.ProfileUUID = profile.UUID
		v.ProfileChecked = true
	}
	hash, err := s.Identity.HashPassword(ctx, in.Password)
	if err != nil {
		failError(w, r, err)
		return
	}
	u, err := s.Store.RegisterEmailUser(ctx, v, hash)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			failure(w, r, 409, "ACCOUNT_ALREADY_EXISTS", "该邮箱或正版账号 ID 已注册，请登录或找回密码。")
			return
		}
		catalogFailV2(w, r, err)
		return
	}
	s.newAccountSession(w, r, sessionDeviceContext(ctx, r, kind, in.DeviceName), u, kind)
}
func (s *Server) newAccountSession(w http.ResponseWriter, r *http.Request, ctx context.Context, u store.User, kind string) {
	token, session, err := s.Identity.IssueSession(ctx, u, kind)
	if err != nil {
		failure(w, r, 503, "ACCOUNT_CREATED_LOGIN_REQUIRED", "账号已创建，暂未完成登录。请使用刚才的邮箱和密码登录。")
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
		s.setCookie(w, token, session.ExpiresAt)
		success(w, r, map[string]any{"user": u, "csrfToken": session.CSRF, "expiresAt": session.ExpiresAt})
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
	if session.User.GameID == "" {
		failure(w, r, 409, "GAME_PROFILE_REQUIRED", "原游戏身份登记已解除，请先在最新版 App 或网页登记自己的正版游戏账号。")
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
	if session.User.ProfileUUID != "" && session.User.ProfileUUID != p.PlayerUUID {
		failure(w, r, 409, "GAME_ID_IDENTITY_MISMATCH", "当前游戏身份与注册时的正版账号不一致，请联系支持人员核对。")
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

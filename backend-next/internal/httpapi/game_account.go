package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"math/big"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *Server) accountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/account/registration-code", s.registrationCode)
	mux.HandleFunc("POST /api/v1/account/password-reset-code", s.passwordResetCode)
	mux.HandleFunc("POST /api/v1/account/register", func(w http.ResponseWriter, r *http.Request) { s.registerGame(w, r, "app") })
	mux.HandleFunc("POST /api/v1/web/register", func(w http.ResponseWriter, r *http.Request) { s.registerGame(w, r, "web") })
	mux.HandleFunc("POST /api/v1/account/password-reset", s.resetGamePassword)
}
func (s *Server) publicAccountOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == "" || s.origin(r)
}
func validPassword(p string) bool {
	return utf8.RuneCountInString(p) >= 8 && utf8.RuneCountInString(p) <= 64 && len(p) <= 256
}
func (s *Server) registrationCode(w http.ResponseWriter, r *http.Request) {
	if !s.publicAccountOrigin(r) {
		failError(w, r, ErrForbidden)
		return
	}
	var in struct {
		GameID string `json:"gameId"`
		QQ     string `json:"qq"`
		// Older clients send this field when requesting a code. Only registerGame validates it.
		Password string `json:"password"`
	}
	if body(w, r, &in) != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if !identity.ValidGameID(in.GameID) || identity.SystemGameID(in.GameID) {
		failure(w, r, 400, "GAME_ID_INVALID", "请输入有效的游戏 ID（1–32 位字母、数字或下划线）。")
		return
	}
	if !identity.ValidQQ(in.QQ) {
		failure(w, r, 400, "QQ_INVALID", "请输入有效的 QQ 号（5–20 位数字）。")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
		failError(w, r, err)
		return
	}
	for _, alias := range []string{strings.ToLower(in.GameID), in.QQ} {
		_, err := s.Store.UserByAlias(ctx, alias)
		if err == nil {
			failure(w, r, 409, "ACCOUNT_ALREADY_EXISTS", "游戏账号或 QQ 已绑定，请直接登录或找回密码。")
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			failError(w, r, err)
			return
		}
	}
	p, err := s.resolveCorePlayer(ctx, in.GameID, true)
	if err != nil {
		failError(w, r, err)
		return
	}
	var n int
	if err = s.Store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM identities WHERE server_uuid=?", p.PlayerUUID).Scan(&n); err != nil {
		failError(w, r, err)
		return
	}
	if n > 0 {
		failure(w, r, 409, "ACCOUNT_ALREADY_EXISTS", "该游戏身份已经注册。")
		return
	}
	s.issueGameCode(w, r, ctx, store.GameVerification{Purpose: "register", PlayerUUID: p.PlayerUUID, GameID: p.GameID, QQ: in.QQ, NodeID: p.ServerID})
}
func (s *Server) passwordResetCode(w http.ResponseWriter, r *http.Request) {
	if !s.publicAccountOrigin(r) {
		failError(w, r, ErrForbidden)
		return
	}
	var in struct {
		Account string `json:"account"`
	}
	if body(w, r, &in) != nil || (!identity.ValidGameID(in.Account) && !identity.ValidQQ(in.Account)) || identity.SystemGameID(in.Account) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
		failError(w, r, err)
		return
	}
	u, err := s.Store.UserByAlias(ctx, strings.ToLower(in.Account))
	if err != nil || u.Status != "active" {
		failure(w, r, 400, "ACCOUNT_NOT_AVAILABLE", "账号不存在或不可用。")
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
	s.issueGameCode(w, r, ctx, store.GameVerification{Purpose: "password_reset", PlayerUUID: u.ServerUUID, GameID: p.GameID, QQ: u.QQ, UserID: u.ID, NodeID: p.ServerID})
}
func (s *Server) issueGameCode(w http.ResponseWriter, r *http.Request, ctx context.Context, v store.GameVerification) {
	random, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		failError(w, r, err)
		return
	}
	token := identity.Secret()
	code := fmt.Sprintf("%06d", random.Int64())
	v.ID = store.ID("verify_")
	v.TokenHash = store.Digest([]byte(token))
	v.CodeHash = store.Digest([]byte(token + ":" + code))
	v.ExpiresAt = time.Now().UTC().Add(10 * time.Minute)
	if err = s.Store.NewGameVerification(ctx, v); err != nil {
		failError(w, r, err)
		return
	}
	op, err := s.coreCall(ctx, "game-verification", v.ID, v.NodeID, "verification.deliver", map[string]string{"playerUuid": v.PlayerUUID, "code": code, "purpose": v.Purpose})
	var result struct {
		Delivered bool `json:"delivered"`
	}
	if err == nil {
		err = CoreResult(op, &result)
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	if !result.Delivered {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if err = s.Store.ActivateGameVerification(ctx, v.ID); err != nil {
		failError(w, r, err)
		return
	}
	success(w, r, map[string]any{"verificationToken": token, "expiresAt": v.ExpiresAt, "resendAfterSeconds": 60})
}
func validCode(token, code string) bool {
	if len(token) != 43 || len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func (s *Server) registerGame(w http.ResponseWriter, r *http.Request, kind string) {
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
	v, err := s.Store.CheckGameVerification(ctx, store.Digest([]byte(in.VerificationToken)), store.Digest([]byte(in.VerificationToken+":"+in.Code)), "register")
	if err != nil {
		failError(w, r, err)
		return
	}
	hash, err := s.Identity.HashPassword(ctx, in.Password)
	if err != nil {
		failError(w, r, err)
		return
	}
	u, err := s.Store.RegisterGameUser(ctx, v, hash)
	if err != nil {
		failError(w, r, err)
		return
	}
	token, csrf := identity.Secret(), identity.Secret()
	expiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err = s.Store.CreateSession(ctx, u, store.Digest([]byte(token)), kind, csrf, expiry, ""); err != nil {
		failError(w, r, err)
		return
	}
	if kind == "web" {
		if old, e := r.Cookie(s.cookieName()); e == nil {
			if err = s.Store.Revoke(ctx, store.Digest([]byte(old.Value))); err != nil {
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
func (s *Server) resetGamePassword(w http.ResponseWriter, r *http.Request) {
	if !s.publicAccountOrigin(r) {
		failError(w, r, ErrForbidden)
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
	v, err := s.Store.CheckGameVerification(ctx, store.Digest([]byte(in.VerificationToken)), store.Digest([]byte(in.VerificationToken+":"+in.Code)), "password_reset")
	if err != nil {
		failError(w, r, err)
		return
	}
	hash, err := s.Identity.HashPassword(ctx, in.NewPassword)
	if err == nil {
		err = s.Store.ResetGamePassword(ctx, v, hash)
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	s.Hub.Wake()
	success(w, r, map[string]bool{"passwordReset": true})
}

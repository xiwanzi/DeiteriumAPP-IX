package httpapi

import (
	"context"
	"crypto/rand"
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
	mux.HandleFunc("POST /api/v1/account/registration-code", s.emailRegistrationCode)
	mux.HandleFunc("POST /api/v1/account/password-reset-code", s.passwordResetCode)
	mux.HandleFunc("POST /api/v1/account/register", func(w http.ResponseWriter, r *http.Request) { s.registerEmail(w, r, "app") })
	mux.HandleFunc("POST /api/v1/web/register", func(w http.ResponseWriter, r *http.Request) { s.registerEmail(w, r, "web") })
	mux.HandleFunc("POST /api/v1/account/game-verification-code", s.gameVerificationCode)
	mux.HandleFunc("POST /api/v1/account/game-verification", s.completeGameVerification)
	mux.HandleFunc("POST /api/v1/account/password-reset", s.resetGamePassword)
	mux.HandleFunc("POST /api/v1/account/password-change-code", s.passwordChangeCode)
	mux.HandleFunc("POST /api/v1/account/password-change", s.changeAccountPassword)
}
func (s *Server) publicAccountOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == "" || s.origin(r)
}
func validPassword(p string) bool {
	return utf8.RuneCountInString(p) >= 8 && utf8.RuneCountInString(p) <= 64 && len(p) <= 256
}
func (s *Server) passwordResetCode(w http.ResponseWriter, r *http.Request) {
	if !s.publicAccountOrigin(r) {
		failError(w, r, ErrForbidden)
		return
	}
	var in struct {
		Account string `json:"account"`
	}
	if body(w, r, &in) != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if strings.Contains(in.Account, "@") {
		u, err := s.Store.UserByAlias(r.Context(), strings.ToLower(strings.TrimSpace(in.Account)))
		if err != nil || u.Status != "active" || u.Email == "" {
			failure(w, r, 400, "ACCOUNT_NOT_AVAILABLE", "账号不存在或不可用。")
			return
		}
		s.issueEmailCode(w, r, store.EmailVerification{Email: u.Email, UserID: u.ID, Purpose: "password_reset"})
		return
	}
	if (!identity.ValidGameID(in.Account) && !identity.ValidQQ(in.Account)) || identity.SystemGameID(in.Account) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	u, err := s.Store.UserByAlias(ctx, strings.ToLower(in.Account))
	if err != nil || u.Status != "active" {
		failure(w, r, 400, "ACCOUNT_NOT_AVAILABLE", "账号不存在或不可用。")
		return
	}
	if u.Email != "" {
		s.issueEmailCode(w, r, store.EmailVerification{Email: u.Email, UserID: u.ID, Purpose: "password_reset"})
		return
	}
	if err := s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
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
	// The installed Core understands register/password_reset; binding is still a
	// registration proof on that wire. The local challenge keeps its own purpose.
	wirePurpose := v.Purpose
	if wirePurpose == "game_verify" || wirePurpose == "profile_claim" {
		wirePurpose = "register"
	}
	if wirePurpose == "password_change" {
		wirePurpose = "password_reset"
	}
	op, err := s.coreCall(ctx, "game-verification", v.ID, v.NodeID, "verification.deliver", map[string]string{"playerUuid": v.PlayerUUID, "code": code, "purpose": wirePurpose})
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
	var emailToken bool
	if err := s.Store.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM email_verifications WHERE token_hash=?)", store.Digest([]byte(in.VerificationToken))).Scan(&emailToken); err != nil {
		failError(w, r, err)
		return
	}
	if emailToken {
		v, err := s.Store.CheckEmailVerification(ctx, store.Digest([]byte(in.VerificationToken)), store.Digest([]byte(in.VerificationToken+":"+in.Code)), "password_reset")
		if err != nil {
			failError(w, r, err)
			return
		}
		hash, err := s.Identity.HashPassword(ctx, in.NewPassword)
		if err == nil {
			err = s.Store.ResetEmailPassword(ctx, v, hash)
		}
		if err != nil {
			failError(w, r, err)
			return
		}
		s.Hub.Wake("accounts.changed")
		success(w, r, map[string]bool{"passwordReset": true})
		return
	}
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
	s.Hub.Wake("accounts.changed")
	success(w, r, map[string]bool{"passwordReset": true})
}

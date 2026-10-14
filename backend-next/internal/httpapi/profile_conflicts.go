package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerProfileConflicts(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/admin/accounts/{userId}/profile-release-code", s.profileReleaseCode)
	mux.HandleFunc("POST /api/v1/admin/accounts/{userId}/profile-release", s.releaseRegisteredProfile)
	mux.HandleFunc("GET /api/v1/admin/accounts/{userId}/profile-resolutions", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		items, err := s.Store.ProfileResolutions(r.Context(), r.PathValue("userId"))
		if err != nil {
			failError(w, r, err)
			return
		}
		success(w, r, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /api/v1/account/game-profile", s.registerReleasedProfile)
	mux.HandleFunc("POST /api/v1/admin/profile-reservations/{caseId}", s.correctProfileReservation)
}

func (s *Server) profileReleaseCode(w http.ResponseWriter, r *http.Request) {
	actor, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var in struct{}
	if body(w, r, &in) != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	current, err := s.Store.AdminGameIdentity(ctx, r.PathValue("userId"))
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	if current.Verified || current.Status == "deleted" || !store.ValidAdmissionUUID(current.UUID) || actor.ID == r.PathValue("userId") {
		failure(w, r, 409, "PROFILE_NOT_RELEASABLE", "只能核验其他账号的未验证游戏登记。")
		return
	}
	if err = s.Store.ReserveVerificationIssue(ctx, s.remoteIP(r)); err != nil {
		failError(w, r, err)
		return
	}
	live, err := s.Core.Player("", current.UUID)
	if err != nil {
		failError(w, r, err)
		return
	}
	if !live.Online || live.PlayerUUID != current.UUID {
		failure(w, r, 503, "PLAYER_OFFLINE", "请让申诉人使用该正版账号进入游戏后接收验证码；暂不能进服时通过工单核实。")
		return
	}
	s.issueGameCode(w, r, ctx, store.GameVerification{Purpose: "profile_claim", PlayerUUID: current.UUID, GameID: live.GameID, UserID: r.PathValue("userId"), NodeID: live.ServerID})
}

func (s *Server) releaseRegisteredProfile(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	allowed, err := s.Store.HasPermission(r.Context(), session.User.ID, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	if !allowed {
		failError(w, r, ErrForbidden)
		return
	}
	var in struct {
		store.ProfileReleaseRequest
		Password          string `json:"password"`
		VerificationToken string `json:"verificationToken,omitempty"`
		Code              string `json:"code,omitempty"`
	}
	if socialBodyV2(w, r, &in, 8192) != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	if in.EvidenceType == "GAME_CODE" && !validCode(in.VerificationToken, in.Code) {
		failError(w, r, store.ErrVerification)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	hash, err := s.Identity.ConfirmPassword(ctx, session.User.ID, in.Password, s.remoteIP(r))
	in.Password = ""
	if errors.Is(err, identity.ErrConfirmationPassword) {
		failure(w, r, 400, "ADMIN_PASSWORD_INVALID", "当前登录的管理员密码不正确。")
		return
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	proof := store.ProfileReleaseProof{AccountDeletionProof: store.AccountDeletionProof{ActorID: session.User.ID, SessionHash: session.TokenHash, CredentialHash: hash}, GameTokenHash: store.Digest([]byte(in.VerificationToken)), GameCodeHash: store.Digest([]byte(in.VerificationToken + ":" + in.Code))}
	result, err := s.Store.ReleaseRegisteredProfile(ctx, r.PathValue("userId"), in.ProfileReleaseRequest, proof)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	s.Hub.Wake("accounts.changed", bridge.UserTopic(r.PathValue("userId")))
	success(w, r, result)
}

func (s *Server) registerReleasedProfile(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	var in struct {
		ClientRequestID string `json:"clientRequestId"`
		GameID          string `json:"gameId"`
	}
	if body(w, r, &in) != nil || !store.ValidSocialID(in.ClientRequestID) || !store.ValidAdmissionName(strings.TrimSpace(in.GameID)) || identity.SystemGameID(in.GameID) {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if result, exists, e := s.Store.RegisteredProfileResult(ctx, session.User.ID, in.ClientRequestID, in.GameID); e != nil {
		catalogFailV2(w, r, e)
		return
	} else if exists {
		success(w, r, result)
		return
	}
	if session.User.GameVerified || session.User.GameID != "" {
		failure(w, r, 409, "PROFILE_ALREADY_REGISTERED", "已有游戏身份登记，不能在此更换绑定。")
		return
	}
	profile, err := s.checkRegistrationProfile(ctx, strings.TrimSpace(in.GameID), s.remoteIP(r))
	if err != nil {
		registrationProfileFailure(w, r, err)
		return
	}
	result, err := s.Store.RegisterReleasedProfile(ctx, session, in.ClientRequestID, profile)
	if errors.Is(err, store.ErrNameConflict) {
		failure(w, r, 409, "GAME_ID_IN_USE", "该游戏身份已被登记，请核对后联系支持人员。")
		return
	}
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	s.Hub.Wake(bridge.UserTopic(session.User.ID), "accounts.changed")
	success(w, r, result)
}

func (s *Server) correctProfileReservation(w http.ResponseWriter, r *http.Request) {
	session, err := s.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	if _, err = s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	var in struct {
		store.ProfileReservationCorrection
		Password string `json:"password"`
	}
	if body(w, r, &in) != nil {
		failError(w, r, bridge.ErrProtocol)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	hash, err := s.Identity.ConfirmPassword(ctx, session.User.ID, in.Password, s.remoteIP(r))
	in.Password = ""
	if errors.Is(err, identity.ErrConfirmationPassword) {
		failure(w, r, 400, "ADMIN_PASSWORD_INVALID", "当前登录的管理员密码不正确。")
		return
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	result, err := s.Store.CorrectProfileReservation(ctx, r.PathValue("caseId"), in.ProfileReservationCorrection, store.AccountDeletionProof{ActorID: session.User.ID, SessionHash: session.TokenHash, CredentialHash: hash})
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	success(w, r, result)
}

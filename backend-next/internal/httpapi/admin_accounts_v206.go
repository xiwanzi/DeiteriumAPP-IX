package httpapi

import (
	"context"
	"errors"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) registerAdminAccountsV206(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/admin/accounts/{userId}/delete", s.deleteAdminAccount)
	mux.HandleFunc("GET /api/v1/account/deletions", s.accountDeletions)
	mux.HandleFunc("GET /api/v1/admin/accounts", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		offset, limit := 0, 30
		var err error
		if raw := r.URL.Query().Get("offset"); raw != "" {
			offset, err = strconv.Atoi(raw)
			if err != nil {
				socialFailureV2(w, r, store.ErrSocialInvalid)
				return
			}
		}
		if raw := r.URL.Query().Get("limit"); raw != "" {
			limit, err = strconv.Atoi(raw)
			if err != nil {
				socialFailureV2(w, r, store.ErrSocialInvalid)
				return
			}
		}
		items, total, err := s.Store.AdminAccountsV206(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("status"), offset, limit)
		if err != nil {
			socialFailureV2(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
	})
	mux.HandleFunc("POST /api/v1/admin/accounts/{userId}", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var input store.AdminAccountChangeV206
		if err = body(w, r, &input); err != nil {
			socialFailureV2(w, r, err)
			return
		}
		result, err := s.Store.ChangeAdminAccountV206(r.Context(), actor.ID, r.PathValue("userId"), input)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
}

func (s *Server) deleteAdminAccount(w http.ResponseWriter, r *http.Request) {
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
	var input struct {
		store.AccountDeletionRequest
		Password string `json:"password"`
	}
	if err = body(w, r, &input); err != nil {
		failError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	hash, err := s.Identity.ConfirmPassword(ctx, session.User.ID, input.Password, s.remoteIP(r))
	input.Password = ""
	if errors.Is(err, identity.ErrConfirmationPassword) {
		failure(w, r, 400, "ADMIN_PASSWORD_INVALID", "当前登录的管理员密码不正确。")
		return
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	result, err := s.Store.DeleteAccount(ctx, store.AccountDeletionProof{ActorID: session.User.ID, SessionHash: session.TokenHash, CredentialHash: hash}, r.PathValue("userId"), input.AccountDeletionRequest)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	v2Success(w, r, result)
}

func (s *Server) accountDeletions(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authenticate(r); err != nil {
		failError(w, r, err)
		return
	}
	after := int64(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		after = n
	}
	items, err := s.Store.AccountDeletions(r.Context(), after, 101)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	more := len(items) > 100
	if more {
		items = items[:100]
	}
	cursor := after
	if len(items) > 0 {
		cursor = items[len(items)-1].Sequence
	}
	v2Success(w, r, map[string]any{"items": items, "cursor": cursor, "hasMore": more})
}

package httpapi

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strconv"
)

func (s *Server) registerAdminAccountsV206(mux *http.ServeMux) {
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

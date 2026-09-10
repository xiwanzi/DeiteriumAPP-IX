package httpapi

import (
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerLauncherIcons(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/app/launcher-icon", s.publicLauncherIcon)
	mux.HandleFunc("GET /api/v1/admin/launcher-icon", s.adminLauncherIcon)
	mux.HandleFunc("PUT /api/v1/admin/launcher-icon", s.saveLauncherIcon)
}

func (s *Server) publicLauncherIcon(w http.ResponseWriter, r *http.Request) {
	v, err := s.Store.LauncherIcon(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, v)
}

func (s *Server) adminLauncherIcon(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "platform.admin"); err != nil {
		failError(w, r, err)
		return
	}
	v, err := s.Store.LauncherIcon(r.Context())
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, map[string]any{"settings": v, "icons": store.LauncherIconOptions()})
}

func (s *Server) saveLauncherIcon(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "platform.admin")
	if err != nil {
		failError(w, r, err)
		return
	}
	var input struct {
		ClientRequestID string `json:"clientRequestId"`
		ExpectedVersion int64  `json:"expectedVersion"`
		IconID          string `json:"iconId"`
	}
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	v, err := s.Store.SaveLauncherIcon(r.Context(), u.ID, input.ClientRequestID, input.ExpectedVersion, input.IconID)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	s.Hub.Wake()
	v2Success(w, r, v)
}

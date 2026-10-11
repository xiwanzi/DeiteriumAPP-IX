package httpapi

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strings"
)

func (s *Server) registerGameTitles(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/titles", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.GameTitles(r.Context(), true)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	change := func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.GameTitleChange
		if err = socialBodyV2(w, r, &in, 8192); err != nil {
			socialFailureV2(w, r, err)
			return
		}
		out, err := s.Store.ChangeGameTitle(r.Context(), actor.ID, r.PathValue("titleId"), in)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	}
	mux.HandleFunc("POST /api/v1/admin/titles", change)
	mux.HandleFunc("POST /api/v1/admin/titles/{titleId}", change)
	mux.HandleFunc("POST /bridge/v1/game-chat/titles/catalog", func(w http.ResponseWriter, r *http.Request) {
		node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !ok || !node.Chat || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		catalog, err := s.Store.GameTitles(r.Context(), false)
		if err != nil {
			failError(w, r, err)
			return
		}
		profiles, err := s.Store.GameTitleProfiles(r.Context())
		if err != nil {
			failError(w, r, err)
			return
		}
		success(w, r, map[string]any{"catalog": catalog, "profiles": profiles})
	})
	reconcile := func(w http.ResponseWriter, r *http.Request) {
		node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !ok || !node.Chat || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		var in store.GameTitleAccess
		if socialBodyV2(w, r, &in, 131072) != nil || !gameUUID.MatchString(in.PlayerUUID) || len(in.AllowedIDs) > 2048 || in.SessionEpoch == "" {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/state") && in.TitleID != "" {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/select") && in.TitleID == "" {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		live, err := s.Core.Player("", in.PlayerUUID)
		if err != nil || !live.Online || live.ServerID != node.ID || live.SessionEpoch != in.SessionEpoch {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		out, err := s.Store.ReconcileGameTitle(r.Context(), live.GameID, in)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		success(w, r, out)
	}
	mux.HandleFunc("POST /bridge/v1/game-chat/titles/state", reconcile)
	mux.HandleFunc("POST /bridge/v1/game-chat/titles/select", reconcile)
}

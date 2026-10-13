package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerPlayerNames(mux *http.ServeMux) {
	mux.HandleFunc("POST /bridge/v1/game-chat/player-names", func(w http.ResponseWriter, r *http.Request) {
		node, ok := s.Config.AuthenticateNode(r.Header.Get("X-Deuterium-Node-ID"), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !ok || !node.Chat || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		var input struct {
			After int64 `json:"after"`
		}
		if e := socialBodyV2(w, r, &input, 256); e != nil || input.After < 0 {
			failure(w, r, 400, "INVALID_CURSOR", "名称同步位置不正确。")
			return
		}
		items, cursor, more, e := s.Store.PlayerNameChanges(r.Context(), input.After)
		if e != nil {
			failError(w, r, e)
			return
		}
		// Keep journal names paired with their original sequence on the Core path too.
		writeJSON(w, 200, map[string]any{"data": map[string]any{"items": items, "cursor": cursor, "hasMore": more}})
	})
	mux.HandleFunc("GET /api/v1/chat/player-names", func(w http.ResponseWriter, r *http.Request) {
		if _, e := s.authenticate(r); e != nil {
			failError(w, r, e)
			return
		}
		after := int64(0)
		var e error
		if raw := r.URL.Query().Get("after"); raw != "" {
			after, e = strconv.ParseInt(raw, 10, 64)
			if e != nil || after < 0 {
				failure(w, r, 400, "INVALID_CURSOR", "名称同步位置不正确。")
				return
			}
		}
		items, cursor, more, e := s.Store.PlayerNameChanges(r.Context(), after)
		if e != nil {
			failError(w, r, e)
			return
		}
		// Changes are an ordered journal; do not project earlier entries to later names.
		writeJSON(w, 200, map[string]any{"data": map[string]any{"items": items, "cursor": cursor, "hasMore": more}})
	})
}
func (s *Server) resolveOfficialUUID(ctx context.Context, uuid string) (store.AdmissionProfile, error) {
	if !store.ValidAdmissionUUID(uuid) {
		return store.AdmissionProfile{}, store.ErrSocialInvalid
	}
	if s.OfficialUUIDResolver != nil {
		p, e := s.OfficialUUIDResolver(ctx, uuid)
		if e != nil {
			return p, e
		}
		if p.UUID != uuid || !store.ValidAdmissionName(p.Name) {
			return store.AdmissionProfile{}, identity.ErrNameLookupUnavailable
		}
		return p, nil
	}
	request, e := http.NewRequestWithContext(ctx, "GET", "https://sessionserver.mojang.com/session/minecraft/profile/"+strings.ReplaceAll(uuid, "-", ""), nil)
	if e != nil {
		return store.AdmissionProfile{}, e
	}
	response, e := admissionProfileClient.Do(request)
	if e != nil {
		return store.AdmissionProfile{}, e
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return store.AdmissionProfile{}, identity.ErrNameLookupUnavailable
	}
	var p struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(&p) != nil || !strings.EqualFold(p.ID, strings.ReplaceAll(uuid, "-", "")) || !store.ValidAdmissionName(p.Name) {
		return store.AdmissionProfile{}, identity.ErrNameLookupUnavailable
	}
	return store.AdmissionProfile{UUID: uuid, Name: p.Name}, nil
}

// Follow verified name-reuse chains so a renamed offline former owner can release
// its stale alias. Public name lookups never serve as proof of a platform password.
func (s *Server) syncOfficialName(ctx context.Context, p store.AdmissionProfile) error {
	return s.syncOfficialNameObserved(ctx, p, time.Now().UTC())
}
func (s *Server) syncOfficialNameObserved(ctx context.Context, p store.AdmissionProfile, observedAt time.Time) error {
	profiles := []store.AdmissionProfile{p}
	seen := map[string]bool{p.UUID: true}
	for at := 0; at < len(profiles); at++ {
		owner, e := s.Store.UserByAlias(ctx, strings.ToLower(profiles[at].Name))
		if errors.Is(e, sql.ErrNoRows) {
			continue
		}
		if e != nil {
			return e
		}
		if owner.ServerUUID == profiles[at].UUID {
			continue
		}
		if !owner.GameVerified || !strings.EqualFold(owner.GameID, profiles[at].Name) || owner.QQ == profiles[at].Name || len(profiles) >= 16 {
			return store.ErrNameConflict
		}
		if seen[owner.ServerUUID] {
			continue
		}
		canonical, e := s.resolveOfficialUUID(ctx, owner.ServerUUID)
		if e != nil {
			return e
		}
		if strings.EqualFold(canonical.Name, profiles[at].Name) {
			return store.ErrNameConflict
		}
		seen[canonical.UUID] = true
		profiles = append(profiles, canonical)
	}
	changes, e := s.Store.SyncVerifiedNamesObserved(ctx, profiles, observedAt)
	if e == nil && len(changes) > 0 {
		s.Hub.Wake(bridge.TopicNames)
	}
	return e
}

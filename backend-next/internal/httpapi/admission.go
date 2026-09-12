package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var admissionProfileClient = &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func (s *Server) resolveAdmissionProfile(ctx context.Context, name string) (store.AdmissionProfile, error) {
	if !store.ValidAdmissionName(name) {
		return store.AdmissionProfile{}, store.ErrSocialInvalid
	}
	if s.AdmissionResolver != nil {
		return s.AdmissionResolver(ctx, name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.minecraftservices.com/minecraft/profile/lookup/name/"+name, nil)
	if err != nil {
		return store.AdmissionProfile{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := admissionProfileClient.Do(req)
	if err != nil {
		return store.AdmissionProfile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 204 {
		return store.AdmissionProfile{}, errAdmissionPlayerMissing
	}
	if resp.StatusCode != 200 {
		return store.AdmissionProfile{}, errors.New("minecraft profile lookup unavailable")
	}
	var profile struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 16384)).Decode(&profile) != nil || len(profile.ID) != 32 || !store.ValidAdmissionName(profile.Name) {
		return store.AdmissionProfile{}, errors.New("invalid minecraft profile response")
	}
	id := strings.ToLower(profile.ID)
	uuid := id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:]
	if !store.ValidAdmissionUUID(uuid) || !strings.EqualFold(profile.Name, name) {
		return store.AdmissionProfile{}, errors.New("minecraft profile mismatch")
	}
	return store.AdmissionProfile{UUID: uuid, Name: profile.Name}, nil
}

var errAdmissionPlayerMissing = errors.New("minecraft profile not found")

func admissionFailure(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrRateLimited) {
		w.Header().Set("Retry-After", "900")
		failure(w, r, 429, "RATE_LIMITED", "操作较频繁，请 15 分钟后再试。")
		return
	}
	if errors.Is(err, errAdmissionPlayerMissing) {
		failure(w, r, 400, "MINECRAFT_PLAYER_NOT_FOUND", "未找到此正版玩家名，请核对 Minecraft Java 版账号名称。")
		return
	}
	catalogFailV2(w, r, err)
}
func (s *Server) publicAdmissionOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return s.Config.AdmissionPublicOrigin != "" && (origin == s.Config.AdmissionPublicOrigin || origin == s.Config.PublicOrigin || origin == "" && r.Header.Get("Sec-Fetch-Site") != "cross-site")
}
func (s *Server) admissionGateway(r *http.Request) bool {
	if r.Header.Get("Origin") != "" {
		return false
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	return len(token) >= 32 && len(token) <= 256 && len(s.Config.AdmissionGatewayTokenSHA256) == 64 && subtle.ConstantTimeCompare([]byte(store.Digest([]byte(token))), []byte(s.Config.AdmissionGatewayTokenSHA256)) == 1
}
func admissionPage(r *http.Request) (int, int, error) {
	offset, limit := 0, 30
	var err error
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, store.ErrSocialInvalid
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, store.ErrSocialInvalid
		}
	}
	if offset < 0 || limit < 1 || limit > 100 {
		return 0, 0, store.ErrSocialInvalid
	}
	return offset, limit, nil
}
func (s *Server) publicAdmissionResult(ctx context.Context, a store.AdmissionApplication) (map[string]any, error) {
	access, err := s.Store.AdmissionCheck(ctx, a.UUID)
	if err != nil {
		return nil, err
	}
	reason := ""
	if a.Status == "REJECTED" {
		reason = a.Reason
	}
	return map[string]any{"applicationId": a.ID, "gameId": a.GameID, "status": a.Status, "createdAt": a.CreatedAt, "reviewedAt": a.ReviewedAt, "reason": reason, "accessAllowed": access.Allowed, "accessStatus": access.Status}, nil
}

func (s *Server) registerAdmission(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admission/config", func(w http.ResponseWriter, r *http.Request) {
		if s.Config.AdmissionPublicOrigin == "" {
			failure(w, r, 503, "ADMISSION_CLOSED", "申请通道尚未开放。")
			return
		}
		v2Success(w, r, map[string]any{"covenantVersion": store.AdmissionCovenantVersion, "groupNumber": "490579956", "groupURL": "https://qm.qq.com/q/HROB9FDSYE", "defaultServer": "amiya", "publicOrigin": s.Config.AdmissionPublicOrigin, "applicationURL": s.Config.AdmissionURL()})
	})
	mux.HandleFunc("POST /api/v1/admission/applications", func(w http.ResponseWriter, r *http.Request) {
		if !s.publicAdmissionOrigin(r) {
			failError(w, r, ErrForbidden)
			return
		}
		var in store.AdmissionSubmission
		if err := body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		in.Normalize()
		if err := in.Validate(); err != nil {
			admissionFailure(w, r, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		// Cheap IP budget also applies to replays without consuming another QQ application slot.
		if err := s.Store.ReserveAdmission(ctx, s.remoteIP(r), "", true); err != nil {
			admissionFailure(w, r, err)
			return
		}
		if old, exists, err := s.Store.ExistingAdmission(ctx, in); err != nil {
			admissionFailure(w, r, err)
			return
		} else if exists {
			result, err := s.publicAdmissionResult(ctx, old)
			if err != nil {
				admissionFailure(w, r, err)
				return
			}
			v2Success(w, r, result)
			return
		}
		if err := s.Store.ReserveAdmission(ctx, s.remoteIP(r), in.QQ, false); err != nil {
			admissionFailure(w, r, err)
			return
		}
		p, err := s.resolveAdmissionProfile(ctx, in.GameID)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		a, err := s.Store.SubmitAdmission(ctx, in, p)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		result, err := s.publicAdmissionResult(ctx, a)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("POST /api/v1/admission/status", func(w http.ResponseWriter, r *http.Request) {
		if !s.publicAdmissionOrigin(r) {
			failError(w, r, ErrForbidden)
			return
		}
		var in struct {
			ReceiptToken string `json:"receiptToken"`
		}
		if err := body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		if err := s.Store.ReserveAdmission(r.Context(), s.remoteIP(r), "", true); err != nil {
			admissionFailure(w, r, err)
			return
		}
		a, err := s.Store.AdmissionReceipt(r.Context(), in.ReceiptToken)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		result, err := s.publicAdmissionResult(r.Context(), a)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("GET /api/v1/admin/whitelist/summary", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		result, err := s.Store.AdmissionSummary(r.Context())
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		result["applicationURL"] = s.Config.AdmissionURL()
		v2Success(w, r, result)
	})
	mux.HandleFunc("GET /api/v1/admin/whitelist/applications", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		offset, limit, err := admissionPage(r)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		items, total, err := s.Store.AdmissionApplications(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("status"), offset, limit)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
	})
	mux.HandleFunc("GET /api/v1/admin/whitelist/entries", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		offset, limit, err := admissionPage(r)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		items, total, err := s.Store.AdmissionEntries(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("status"), offset, limit)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": items, "total": total, "offset": offset, "limit": limit})
	})
	mux.HandleFunc("GET /api/v1/admin/whitelist/resolve", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		p, err := s.resolveAdmissionProfile(r.Context(), strings.TrimSpace(r.URL.Query().Get("gameId")))
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, p)
	})
	mux.HandleFunc("POST /api/v1/admin/whitelist/applications/{applicationId}/review", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.AdmissionDecision
		if err = body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		result, err := s.Store.ReviewAdmission(r.Context(), actor.ID, r.PathValue("applicationId"), in)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("POST /api/v1/admin/whitelist/entries", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.AdmissionManualAdd
		if err = body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		p, err := s.resolveAdmissionProfile(r.Context(), strings.TrimSpace(in.GameID))
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		if p.UUID != in.ExpectedUUID {
			failure(w, r, 409, "PLAYER_IDENTITY_CHANGED", "玩家名对应的身份已变化，请重新查询后确认。")
			return
		}
		result, err := s.Store.AddAdmission(r.Context(), actor.ID, in, p)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("POST /api/v1/admin/whitelist/entries/{uuid}/revoke", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.AdmissionRevoke
		if err = body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		result, err := s.Store.RevokeAdmission(r.Context(), actor.ID, r.PathValue("uuid"), in)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("GET /api/v1/admin/whitelist/entries/{uuid}/history", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		items, err := s.Store.AdmissionHistory(r.Context(), r.PathValue("uuid"))
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /bridge/v1/admission/check", func(w http.ResponseWriter, r *http.Request) {
		if !s.admissionGateway(r) {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		var in struct {
			UUID string `json:"uuid"`
		}
		if err := body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		result, err := s.Store.AdmissionCheck(r.Context(), in.UUID)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, result)
	})
	mux.HandleFunc("POST /bridge/v1/admission/poll", func(w http.ResponseWriter, r *http.Request) {
		if !s.admissionGateway(r) {
			failError(w, r, store.ErrUnauthorized)
			return
		}
		var in store.AdmissionHeartbeat
		if err := body(w, r, &in); err != nil {
			admissionFailure(w, r, err)
			return
		}
		commands, err := s.Store.AdmissionPoll(r.Context(), in)
		if err != nil {
			admissionFailure(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"commands": commands})
	})
}

package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) checkRegistrationProfile(ctx context.Context, name, ip string) (store.AdmissionProfile, error) {
	if !store.ValidAdmissionName(name) {
		return store.AdmissionProfile{}, errAdmissionPlayerMissing
	}
	profile, err := s.Store.CachedMinecraftProfile(ctx, name)
	if !errors.Is(err, sql.ErrNoRows) {
		return profile, err
	}
	if err = s.Store.ReserveProfileLookup(ctx, ip); err != nil {
		return store.AdmissionProfile{}, err
	}
	if profile, err = s.resolveAdmissionProfile(ctx, name); err != nil {
		return profile, err
	}
	return profile, s.Store.CacheMinecraftProfileCheck(ctx, name, profile.UUID)
}

func registrationProfileFailure(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrRateLimited) {
		failError(w, r, err)
	} else if errors.Is(err, errAdmissionPlayerMissing) || errors.Is(err, store.ErrSocialInvalid) {
		failure(w, r, 400, "MINECRAFT_PLAYER_NOT_FOUND", "未找到此正版账号 ID，请核对 Minecraft Java 版玩家名后重新获取验证码。")
	} else {
		failure(w, r, 503, "MINECRAFT_LOOKUP_UNAVAILABLE", "正版账号 ID 查询服务暂不可用，请稍后重新获取验证码。")
	}
}

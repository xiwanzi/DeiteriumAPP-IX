package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) checkRegistrationProfile(ctx context.Context, name string) error {
	if !store.ValidAdmissionName(name) {
		return errAdmissionPlayerMissing
	}
	checked, err := s.Store.MinecraftProfileChecked(ctx, name)
	if err != nil || checked {
		return err
	}
	if _, err = s.resolveAdmissionProfile(ctx, name); err != nil {
		return err
	}
	return s.Store.CacheMinecraftProfileCheck(ctx, name)
}

func registrationProfileFailure(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errAdmissionPlayerMissing) || errors.Is(err, store.ErrSocialInvalid) {
		failure(w, r, 400, "MINECRAFT_PLAYER_NOT_FOUND", "未找到此正版账号 ID，请核对 Minecraft Java 版玩家名后重新获取验证码。")
	} else {
		failure(w, r, 503, "MINECRAFT_LOOKUP_UNAVAILABLE", "正版账号 ID 查询服务暂不可用，请稍后重新获取验证码。")
	}
}

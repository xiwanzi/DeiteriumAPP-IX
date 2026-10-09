package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func chatMediaFailure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrStickerLimit):
		failure(w, r, 409, "STICKER_LIMIT", "贴纸已满（10/10），删除已有贴纸后可继续添加。")
	case errors.Is(err, store.ErrMediaExpired):
		failure(w, r, 410, "MEDIA_EXPIRED", "图片已过期。")
	case errors.Is(err, store.ErrSocialInvalid), errors.Is(err, objectstorage.ErrInvalidObject):
		failure(w, r, 400, "INVALID_MEDIA", "图片未能处理，请重新选择较小的图片。")
	case errors.Is(err, store.ErrSocialNotFound), errors.Is(err, store.ErrAssetForbidden):
		failure(w, r, 404, "MEDIA_NOT_FOUND", "图片不存在或不可用。")
	default:
		assetErrorV2(w, r, err)
	}
}

func chatUploadView(ctx context.Context, u store.ChatMediaUpload) (map[string]any, error) {
	out := map[string]any{"uploadId": u.ID, "media": u.Media, "expiresAt": u.ExpiresAt, "status": u.Media.State}
	if u.Media.State == "READY" {
		return out, nil
	}
	if u.Media.State != "UPLOADING" {
		return nil, store.ErrMediaExpired
	}
	c, err := objectstorage.FromEnvironment()
	if err != nil {
		return nil, err
	}
	main, err := c.Upload(ctx, store.ChatMediaKey(c.Config.Prefix, u.StagedID, false), "image/webp", u.Media.MD5, u.Media.SizeBytes, u.ExpiresAt)
	if err != nil {
		return nil, err
	}
	preview, err := c.Upload(ctx, store.ChatMediaKey(c.Config.Prefix, u.StagedID, true), "image/webp", u.Media.ThumbnailMD5, u.Media.ThumbnailBytes, u.ExpiresAt)
	if err != nil {
		return nil, err
	}
	out["authorization"], out["thumbnailAuthorization"] = main, preview
	return out, nil
}

func (s *Server) registerChatMedia(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/chat/sends/{id}", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		var id string
		err := s.Store.DB.QueryRowContext(r.Context(), `SELECT message_id FROM chat_messages_next WHERE source_id=? AND client_message_id=?`, "app:"+a.User.ID, r.PathValue("id")).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			v2Success(w, r, map[string]any{"status": "absent"})
			return
		}
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		v2Success(w, r, s.acceptedChatV203(r.Context(), r.PathValue("id"), id))
	})
	verification := make(chan struct{}, 1)
	mux.HandleFunc("POST /api/v1/chat/media/uploads", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		var in store.ChatMediaInput
		if err := socialBodyV2(w, r, &in, 4096); err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		// Fail before creating an upload if storage is not configured.
		if _, err := objectstorage.FromEnvironment(); err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		u, err := s.Store.CreateChatMediaUpload(r.Context(), a.User.ID, in)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		out, err := chatUploadView(r.Context(), u)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("GET /api/v1/chat/media/uploads/{id}", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		u, err := s.Store.ChatMediaUpload(r.Context(), a.User.ID, r.PathValue("id"))
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		out, err := chatUploadView(r.Context(), u)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("POST /api/v1/chat/media/uploads/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		u, err := s.Store.ChatMediaUpload(r.Context(), a.User.ID, r.PathValue("id"))
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		if u.Media.State == "READY" {
			out, _ := chatUploadView(r.Context(), u)
			v2Success(w, r, out)
			return
		}
		if u.Media.State != "UPLOADING" {
			chatMediaFailure(w, r, store.ErrMediaExpired)
			return
		}
		select {
		case verification <- struct{}{}:
			defer func() { <-verification }()
		default:
			failure(w, r, 503, "MEDIA_BUSY", "图片正在处理中，请稍后重试。")
			return
		}
		c, err := objectstorage.FromEnvironment()
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		data, err := c.ReadChatObject(r.Context(), store.ChatMediaKey(c.Config.Prefix, u.StagedID, false), u.Media.SizeBytes)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		main, err := objectstorage.VerifyChatImage(data, u.Media.SHA256, u.Media.MD5, false)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		data, err = c.ReadChatObject(r.Context(), store.ChatMediaKey(c.Config.Prefix, u.StagedID, true), u.Media.ThumbnailBytes)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		if _, err = objectstorage.VerifyChatImage(data, u.Media.ThumbnailSHA256, u.Media.ThumbnailMD5, true); err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		u, err = s.Store.CompleteChatMedia(r.Context(), a.User.ID, u.ID, main)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		out, _ := chatUploadView(r.Context(), u)
		v2Success(w, r, out)
	})
	mux.HandleFunc("GET /api/v1/chat/media/files/{id}/access", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		m, deadline, err := s.Store.ChatMediaAccess(r.Context(), a.User.ID, r.PathValue("id"), r.URL.Query().Get("messageId"))
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		c, err := objectstorage.FromEnvironment()
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		url, expires, err := c.DownloadUntil(r.Context(), store.ChatMediaKey(c.Config.Prefix, m.ID, false), deadline)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		preview, _, err := c.DownloadUntil(r.Context(), store.ChatMediaKey(c.Config.Prefix, m.ID, true), deadline)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		v2Success(w, r, map[string]any{"media": m, "url": url, "thumbnailUrl": preview, "urlExpiresAt": expires})
	})
	mux.HandleFunc("GET /api/v1/chat/stickers", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		items, err := s.Store.ChatStickers(r.Context(), a.User.ID)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"stickers": items, "limit": store.ChatStickerLimit})
	})
	mux.HandleFunc("POST /api/v1/chat/stickers", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		var in struct {
			MediaID string `json:"mediaId"`
		}
		if err := socialBodyV2(w, r, &in, 1024); err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		m, err := s.Store.SaveChatSticker(r.Context(), a.User.ID, in.MediaID)
		if err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		v2Success(w, r, m)
	})
	mux.HandleFunc("DELETE /api/v1/chat/stickers/{id}", func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		if err := s.Store.DeleteChatSticker(r.Context(), a.User.ID, r.PathValue("id")); err != nil {
			chatMediaFailure(w, r, err)
			return
		}
		v2Success(w, r, map[string]bool{"deleted": true})
	})
}

func (s *Server) StartChatMediaLifecycle() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
			if c, err := objectstorage.FromEnvironment(); err == nil {
				if _, err = s.Store.RunChatMediaGC(ctx, c, c.Config.Prefix, 32); err != nil && s.ctx.Err() == nil {
					slog.Warn("chat media cleanup deferred")
				}
			}
			cancel()
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

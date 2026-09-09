package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

type assetUploadInputV2 struct {
	ClientRequestID string `json:"clientRequestId"`
	Purpose         string `json:"purpose"`
	BusinessType    string `json:"businessType"`
	BusinessRef     string `json:"businessRef"`
	FileName        string `json:"fileName"`
	ContentType     string `json:"contentType"`
	SizeBytes       int64  `json:"sizeBytes"`
	ContentMD5      string `json:"contentMd5"`
	AltText         string `json:"altText"`
}

func validateAssetInputV2(in assetUploadInputV2) bool {
	md5, err := base64.StdEncoding.DecodeString(in.ContentMD5)
	if in.ClientRequestID == "" || len(in.ClientRequestID) > 128 || utf8.RuneCountInString(in.FileName) < 1 || utf8.RuneCountInString(in.FileName) > 160 || utf8.RuneCountInString(in.AltText) > 200 || len(in.BusinessRef) > 128 || in.SizeBytes < 1 || in.SizeBytes > 20<<20 || err != nil || len(md5) != 16 || base64.StdEncoding.EncodeToString(md5) != in.ContentMD5 {
		return false
	}
	if in.ContentType != "image/png" && in.ContentType != "image/jpeg" && in.ContentType != "image/webp" {
		return false
	}
	expected := map[string]string{"AVATAR": "PROFILE", "STORE_MEDIA": "STORE", "MARKET_PHOTO": "MARKET_LISTING", "COMMISSION_COVER": "COMMISSION", "ANNOUNCEMENT_MEDIA": "ANNOUNCEMENT"}
	if value, ok := expected[in.Purpose]; ok {
		return in.BusinessType == value
	}
	return in.Purpose == "DISPUTE_EVIDENCE" && in.BusinessRef != "" && (in.BusinessType == "ORDER" || in.BusinessType == "COMMISSION" || in.BusinessType == "INTERVENTION")
}

func assetErrorV2(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrAssetBusy):
		failure(w, r, 503, "ASSET_BUSY", "图片正在整理，请稍后重试。")
	case errors.Is(err, store.ErrAssetForbidden):
		failure(w, r, 404, "ASSET_NOT_FOUND", "素材不存在或无权访问。")
	case errors.Is(err, store.ErrAssetUnavailable):
		failure(w, r, 409, "ASSET_NOT_READY", "素材尚未完成验证或已移除。")
	case errors.Is(err, store.ErrAssetInUse):
		failure(w, r, 409, "ASSET_IN_USE", "素材仍在使用中，请先解除业务绑定。")
	case errors.Is(err, objectstorage.ErrUnavailable):
		failure(w, r, 503, "STORAGE_UNAVAILABLE", "图片服务暂不可用，请稍后重试。")
	default:
		failError(w, r, err)
	}
}

func (s *Server) assetSessionViewV2(ctx context.Context, u store.AssetUploadV2) (map[string]any, error) {
	status := u.Status
	if u.Lifecycle == "EXPIRED" || u.Lifecycle == "PURGED" || (u.RetainUntil.Valid && !u.RetainUntil.Time.After(time.Now())) {
		status = "EXPIRED"
	}
	if u.RemovedAt.Valid {
		status = "REJECTED"
	}
	if (status == "AUTHORIZED" || status == "VERIFYING") && time.Now().After(u.ExpiresAt) {
		status = "EXPIRED"
	}
	view := map[string]any{"uploadId": u.UploadID, "assetId": u.AssetID, "purpose": u.Purpose, "status": status, "authorization": nil, "sessionExpiresAt": u.ExpiresAt.UTC(), "asset": nil, "rejectionCode": nil, "retryAfterSeconds": 2}
	if u.Rejection.Valid {
		view["rejectionCode"] = u.Rejection.String
	}
	if status == "READY" {
		asset, err := s.Store.AssetV2(ctx, u.UserID, u.AssetID)
		if err != nil {
			return nil, err
		}
		view["asset"] = asset
	}
	if status == "AUTHORIZED" {
		c, err := objectstorage.FromEnvironment()
		if err != nil {
			return nil, err
		}
		auth, err := c.Upload(ctx, u.ObjectKey, u.ContentType, u.MD5, u.SizeBytes, u.ExpiresAt)
		if err != nil {
			return nil, err
		}
		view["authorization"] = auth
	}
	return view, nil
}

func (s *Server) registerAssetsV2(mux *http.ServeMux) {
	// Full image decoding can allocate pixel buffers: one bounded verification
	// at a time fits the small deployment host and is independent of HTTP load.
	verificationSlots := make(chan struct{}, 1)
	mux.HandleFunc("POST /api/v1/assets/uploads", func(w http.ResponseWriter, r *http.Request) {
		session, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		var in assetUploadInputV2
		if body(w, r, &in) != nil || !validateAssetInputV2(in) {
			failure(w, r, 400, "INVALID_REQUEST", "图片信息不正确，仅支持 PNG、JPEG、WebP，最大20MiB。 ")
			return
		}
		if in.Purpose == "STORE_MEDIA" && in.BusinessRef == "" {
			// Only platform admins can create stores. An avatar uploaded during
			// creation stays owned/unbound until the store is saved successfully.
			if _, err := s.admin(r, "platform.admin"); err != nil {
				failError(w, r, err)
				return
			}
		}
		if in.Purpose == "STORE_MEDIA" && in.BusinessRef != "" {
			var present int
			if err = s.Store.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM catalog_records_v2 WHERE resource_id=? AND kind='store'", in.BusinessRef).Scan(&present); err != nil {
				failError(w, r, err)
				return
			}
			if present != 1 {
				failure(w, r, 404, "STORE_NOT_FOUND", "店铺不存在。 ")
				return
			}
			if err = s.Store.CatalogCanManageV2(r.Context(), session.User.ID, in.BusinessRef, "PRODUCT_EDIT"); err != nil {
				if err = s.Store.CatalogCanManageV2(r.Context(), session.User.ID, in.BusinessRef, "STORE_EDIT"); err != nil {
					failError(w, r, ErrForbidden)
					return
				}
			}
		}
		if in.Purpose == "ANNOUNCEMENT_MEDIA" {
			if _, err = s.admin(r, "announcements.manage"); err != nil {
				failError(w, r, err)
				return
			}
			if in.BusinessRef != "" {
				var present int
				if err = s.Store.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM social_announcements_v2 WHERE announcement_id=?", in.BusinessRef).Scan(&present); err != nil {
					failError(w, r, err)
					return
				}
				if present != 1 {
					failure(w, r, 404, "ANNOUNCEMENT_NOT_FOUND", "公告不存在。 ")
					return
				}
			}
		}
		if in.Purpose == "AVATAR" && in.BusinessRef != "" && in.BusinessRef != session.User.ID && in.BusinessRef != session.User.PlayerRef {
			failError(w, r, ErrForbidden)
			return
		}
		if in.Purpose == "MARKET_PHOTO" && in.BusinessRef != "" {
			var present int
			if err = s.Store.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM catalog_records_v2 WHERE resource_id=? AND kind='listing' AND owner_id=?", in.BusinessRef, session.User.ID).Scan(&present); err != nil {
				failError(w, r, err)
				return
			}
			if present != 1 {
				failError(w, r, ErrForbidden)
				return
			}
		}
		if in.Purpose == "COMMISSION_COVER" && in.BusinessRef != "" {
			commission, lookupError := s.Store.CommerceRecordV2(r.Context(), in.BusinessRef)
			if lookupError != nil {
				catalogFailV2(w, r, lookupError)
				return
			}
			if commission.Kind != "COMMISSION" || commission.OwnerID != session.User.ID {
				failError(w, r, ErrForbidden)
				return
			}
		}
		if in.Purpose == "DISPUTE_EVIDENCE" {
			allowed, permissionError := s.Store.CommerceCanUploadEvidenceV2(r.Context(), session.User.ID, in.BusinessType, in.BusinessRef)
			if permissionError != nil {
				catalogFailV2(w, r, permissionError)
				return
			}
			if !allowed {
				assetErrorV2(w, r, store.ErrAssetForbidden)
				return
			}
		}
		c, err := objectstorage.FromEnvironment()
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		// Replays remain usable at the per-user quota; no new authorization or
		// storage path is minted for an existing clientRequestId.
		var recent int
		err = s.Store.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM asset_uploads_v2 WHERE user_id=? AND created_at>DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 HOUR) AND client_request_id<>?", session.User.ID, in.ClientRequestID).Scan(&recent)
		if err != nil {
			failError(w, r, err)
			return
		}
		if recent >= 100 {
			failure(w, r, 429, "UPLOAD_LIMITED", "图片上传过于频繁，请稍后重试。 ")
			return
		}
		fingerprint, _ := json.Marshal(in)
		id := store.ID("upload_")
		ext := map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp"}[in.ContentType]
		u, err := s.Store.CreateAssetUploadV2(r.Context(), store.AssetUploadV2{UploadID: id, AssetID: store.ID("asset_"), UserID: session.User.ID, ClientRequestID: in.ClientRequestID, Fingerprint: store.Digest(fingerprint), Purpose: in.Purpose, BusinessType: in.BusinessType, BusinessRef: in.BusinessRef, ObjectKey: c.Config.Prefix + "uploads/" + id + "." + ext, ContentType: in.ContentType, MD5: in.ContentMD5, AltText: in.AltText, SizeBytes: in.SizeBytes})
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		view, err := s.assetSessionViewV2(r.Context(), u)
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		v2Success(w, r, view)
	})
	read := func(w http.ResponseWriter, r *http.Request) {
		session, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		if r.Method == "POST" {
			var in struct {
				ClientRequestID string `json:"clientRequestId"`
			}
			if body(w, r, &in) != nil || in.ClientRequestID == "" || len(in.ClientRequestID) > 128 {
				failure(w, r, 400, "INVALID_REQUEST", "请求标识不正确。 ")
				return
			}
		}
		u, err := s.Store.AssetUploadV2(r.Context(), session.User.ID, r.PathValue("uploadId"))
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		view, err := s.assetSessionViewV2(r.Context(), u)
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		v2Success(w, r, view)
	}
	mux.HandleFunc("GET /api/v1/assets/uploads/{uploadId}", read)
	mux.HandleFunc("POST /api/v1/assets/uploads/{uploadId}/renew", read)
	mux.HandleFunc("POST /api/v1/assets/uploads/{uploadId}/complete", func(w http.ResponseWriter, r *http.Request) {
		session, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string `json:"clientRequestId"`
			OSSRequestID    string `json:"ossRequestId"`
		}
		if body(w, r, &in) != nil || in.ClientRequestID == "" || len(in.ClientRequestID) > 128 || len(in.OSSRequestID) > 128 {
			failure(w, r, 400, "INVALID_REQUEST", "请求标识不正确。 ")
			return
		}
		u, err := s.Store.AssetUploadV2(r.Context(), session.User.ID, r.PathValue("uploadId"))
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		if u.Status != "READY" && u.Status != "REJECTED" && time.Now().Before(u.ExpiresAt) {
			select {
			case verificationSlots <- struct{}{}:
				defer func() { <-verificationSlots }()
			default:
				failure(w, r, 503, "VERIFY_BUSY", "图片正在排队验证，请稍后重试。 ")
				return
			}
			claimed, err := s.Store.BeginAssetVerificationV2(r.Context(), u)
			if err != nil {
				failError(w, r, err)
				return
			}
			if claimed {
				ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
				defer cancel()
				c, verifyErr := objectstorage.FromEnvironment()
				var verified objectstorage.VerifiedImage
				if verifyErr == nil {
					verified, verifyErr = c.Verify(ctx, u.ObjectKey, u.ContentType, u.MD5, u.SizeBytes)
				}
				finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
				defer finishCancel()
				if err = s.Store.FinishAssetVerificationV2(finishCtx, u, verified, verifyErr); err != nil {
					failError(w, r, err)
					return
				}
				if verifyErr != nil && !errors.Is(verifyErr, objectstorage.ErrInvalidObject) {
					assetErrorV2(w, r, verifyErr)
					return
				}
			}
		}
		u, err = s.Store.AssetUploadV2(r.Context(), session.User.ID, u.UploadID)
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		view, err := s.assetSessionViewV2(r.Context(), u)
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		v2Success(w, r, view)
	})
	mux.HandleFunc("GET /api/v1/assets/{assetId}", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		asset, err := s.Store.AssetV2(r.Context(), v.User.ID, r.PathValue("assetId"))
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		v2Success(w, r, asset)
	})
	mux.HandleFunc("POST /api/v1/assets/{assetId}/remove", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.authenticate(r)
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string `json:"clientRequestId"`
		}
		if body(w, r, &in) != nil || strings.TrimSpace(in.ClientRequestID) == "" || len(in.ClientRequestID) > 128 {
			failure(w, r, 400, "INVALID_REQUEST", "请求标识不正确。 ")
			return
		}
		if err = s.Store.RemoveAssetV2(r.Context(), v.User.ID, r.PathValue("assetId")); err != nil {
			assetErrorV2(w, r, err)
			return
		}
		v2Success(w, r, map[string]bool{"removed": true})
	})
}

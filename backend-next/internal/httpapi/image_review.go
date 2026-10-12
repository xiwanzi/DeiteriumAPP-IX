package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/imagereview"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// ImageReviewHandler intentionally does NOT call New: it starts no business,
// mail, game, storage-GC or migration workers in the existing backend.
func ImageReviewHandler(source *store.Store, c config.Config, review *imagereview.Service) http.Handler {
	s := &Server{Store: source, Config: c}
	mux := http.NewServeMux()
	prefix := "/api/v1/admin/image-review"
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		_, e := s.admin(r, "platform.admin")
		if e != nil {
			failError(w, r, e)
			return false
		}
		return true
	}
	fail := func(w http.ResponseWriter, r *http.Request, e error) {
		if errors.Is(e, imagereview.ErrNotFound) {
			failure(w, r, 404, "REVIEW_NOT_FOUND", "审核记录不存在或已经删除。")
			return
		}
		failure(w, r, 503, "REVIEW_UNAVAILABLE", "图片审核暂不可用，请稍后重试。")
	}
	mux.HandleFunc("GET "+prefix, func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		page := 0
		if raw := r.URL.Query().Get("page"); raw != "" {
			var e error
			page, e = strconv.Atoi(raw)
			if e != nil || page < 0 || page > 10000 {
				failure(w, r, 400, "INVALID_PAGE", "页码不正确。")
				return
			}
		}
		items, more, e := review.Repo.List(r.Context(), page)
		if e != nil {
			fail(w, r, e)
			return
		}
		summary, e := review.Repo.Summary(r.Context())
		if e != nil {
			fail(w, r, e)
			return
		}
		v2Success(w, r, map[string]any{"items": items, "hasMore": more, "page": page, "summary": summary})
	})
	mux.HandleFunc("GET "+prefix+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		id := r.PathValue("id")
		if !imagereview.ValidID(id) {
			fail(w, r, imagereview.ErrNotFound)
			return
		}
		i, e := review.Repo.Item(r.Context(), id)
		if e != nil {
			fail(w, r, e)
			return
		}
		if i.State != "FLAGGED" {
			fail(w, r, imagereview.ErrNotFound)
			return
		}
		// Update recent message attribution on demand, then read durable snapshots.
		if e = review.RefreshEvents(r.Context(), i); e != nil {
			failure(w, r, 503, "REVIEW_EVENTS_UNAVAILABLE", "发送记录暂未读取成功，请稍后重试。")
			return
		}
		i.Events, i.EventsTruncated, e = review.Repo.Events(r.Context(), id)
		if e != nil {
			fail(w, r, e)
			return
		}
		v2Success(w, r, i)
	})
	mux.HandleFunc("GET "+prefix+"/{id}/image", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		id := r.PathValue("id")
		if !imagereview.ValidID(id) {
			fail(w, r, imagereview.ErrNotFound)
			return
		}
		i, e := review.Repo.Item(r.Context(), id)
		if e != nil {
			fail(w, r, e)
			return
		}
		if i.State != "FLAGGED" || i.EvidenceKey == "" {
			fail(w, r, imagereview.ErrNotFound)
			return
		}
		location, e := review.Objects.URL(r.Context(), i.EvidenceKey)
		if e != nil {
			fail(w, r, e)
			return
		}
		// Browser fetches OSS directly. This service never proxies full image bytes.
		http.Redirect(w, r, location, http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("DELETE "+prefix+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		id := r.PathValue("id")
		if !imagereview.ValidID(id) {
			fail(w, r, imagereview.ErrNotFound)
			return
		}
		i, e := review.Repo.BeginDelete(r.Context(), id)
		if e != nil {
			fail(w, r, e)
			return
		}
		cleanupCtx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if e = review.Cleanup(cleanupCtx, i); e != nil {
			// Durable DELETING is picked up independently even if the client disconnects.
			v2Success(w, r, map[string]any{"id": id, "state": "DELETING"})
			return
		}
		v2Success(w, r, map[string]any{"id": id, "state": "DELETED"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if source.DB.PingContext(ctx) != nil {
			failure(w, r, 503, "NOT_READY", "not ready")
			return
		}
		if _, e := review.Repo.Summary(ctx); e != nil {
			failure(w, r, 503, "NOT_READY", "not ready")
			return
		}
		success(w, r, map[string]string{"status": "ready"})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, store.ID("req_")))
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// Only web sessions are accepted; this also excludes gamechat bearer handling.
		if strings.HasPrefix(r.URL.Path, prefix) && r.Header.Get("Authorization") != "" {
			failure(w, r, 401, "UNAUTHORIZED", "请通过网页后台登录。")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

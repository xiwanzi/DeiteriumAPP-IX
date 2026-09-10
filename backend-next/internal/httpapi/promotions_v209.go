package httpapi

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
	"strconv"
)

func (s *Server) itemVersionV209(w http.ResponseWriter, r *http.Request) {
	if _, e := s.admin(r, "core.read"); e != nil {
		failError(w, r, e)
		return
	}
	rev, e := strconv.ParseInt(r.PathValue("revision"), 10, 64)
	if e != nil || rev < 1 || len(r.PathValue("itemRef")) > 96 {
		catalogFailV2(w, r, &store.CatalogErrorV2{Status: 400, Code: "INVALID_REQUEST", Message: "物品版本格式不正确。"})
		return
	}
	item, archived, e := s.Store.CoreItem(r.Context(), r.PathValue("itemRef"), rev)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2Success(w, r, store.CoreItemListing{ItemVersion: item, Archived: archived})
}

func (s *Server) registerPromotionsV209(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/store/coupons", s.couponsV209(false))
	mux.HandleFunc("GET /api/v1/store/coupons/attention", s.couponAttentionV209)
	mux.HandleFunc("POST /api/v1/store/coupons/attention", s.acknowledgeCouponAttentionV209)
	mux.HandleFunc("GET /api/v1/admin/coupons", s.couponsV209(true))
	mux.HandleFunc("POST /api/v1/admin/coupons", s.saveCouponV209)
	mux.HandleFunc("POST /api/v1/admin/coupons/publish", s.publishCouponDraftsV209)
	mux.HandleFunc("PUT /api/v1/admin/coupons/{couponId}", s.saveCouponV209)
}

func (s *Server) couponsV209(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		if admin {
			if _, e := s.admin(r, "platform.admin"); e != nil {
				failError(w, r, e)
				return
			}
		}
		limit, e := catalogLimitV2(r)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		rows, next, more, e := s.Store.CouponsFilteredV209(r.Context(), u.User.ID, admin, r.URL.Query().Get("q"), r.URL.Query().Get("cursor"), limit, r.URL.Query().Get("status"))
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		views, e := s.Store.CouponViewsV209(r.Context(), rows, admin)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		var cursor any
		if more {
			cursor = next
		}
		v2List(w, r, views, cursor, more)
	}
}

func (s *Server) saveCouponV209(w http.ResponseWriter, r *http.Request) {
	u, e := s.admin(r, "platform.admin")
	if e != nil {
		failError(w, r, e)
		return
	}
	input, e := catalogReadV2(w, r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	id := r.PathValue("couponId")
	key, version, content, e := catalogFlatMutationV2(input, id == "")
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	d, e := s.Store.SaveCouponV209(r.Context(), u.ID, id, key, version, content)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	status := 200
	if id == "" {
		status = 201
	}
	catalogSuccessV2(w, r, status, store.CouponViewV209(d, true))
}

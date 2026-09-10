package httpapi

import (
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) couponAttentionV209(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	limit, e := catalogLimitV2(r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	views, next, more, e := s.Store.CouponAttentionV209(r.Context(), u.User.ID, r.URL.Query().Get("cursor"), limit)
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

func (s *Server) acknowledgeCouponAttentionV209(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	input, e := catalogReadV2(w, r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	ids, ok := input["couponIds"].([]any)
	viewed, valid := input["viewed"].(bool)
	if !ok || !valid || len(input) != 2 {
		catalogFailV2(w, r, &store.CatalogErrorV2{Status: 400, Code: "INVALID_REQUEST", Message: "优惠提醒确认格式不正确。"})
		return
	}
	refs := []string{}
	for _, id := range ids {
		ref, ok := id.(string)
		if !ok {
			catalogFailV2(w, r, &store.CatalogErrorV2{Status: 400, Code: "INVALID_REQUEST", Message: "优惠券标识格式不正确。"})
			return
		}
		refs = append(refs, ref)
	}
	if e := s.Store.AcknowledgeCouponAttentionV209(r.Context(), u.User.ID, refs, viewed); e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2Success(w, r, map[string]any{"acknowledged": true})
}

package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) publishCouponDraftsV209(w http.ResponseWriter, r *http.Request) {
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
	key, ok := input["clientRequestId"].(string)
	items, valid := input["coupons"].([]any)
	bad := func() {
		catalogFailV2(w, r, &store.CatalogErrorV2{Status: 400, Code: "INVALID_REQUEST", Message: "请选择有效的优惠券草稿。"})
	}
	if !ok || !valid || len(input) != 2 {
		bad()
		return
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok || len(row) != 2 || row["couponId"] == nil || row["expectedVersion"] == nil {
			bad()
			return
		}
	}
	raw, e := json.Marshal(items)
	if e != nil {
		bad()
		return
	}
	var selected []store.CouponDraftSelectionV209
	if e = json.Unmarshal(raw, &selected); e != nil {
		bad()
		return
	}
	batch, e := s.Store.PublishCouponDraftsV209(r.Context(), u.ID, key, selected)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	views, e := s.Store.CouponViewsV209(r.Context(), batch.Coupons, true)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2Success(w, r, map[string]any{"releaseBatchId": batch.BatchID, "releasedAt": batch.ReleasedAt, "coupons": views})
}

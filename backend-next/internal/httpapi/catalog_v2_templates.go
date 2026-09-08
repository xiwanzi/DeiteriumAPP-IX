package httpapi

import (
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerCatalogTemplatesV2(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/merchant/stores/{storeId}/delivery-templates", s.catalogTemplateWriteV2(true))
	mux.HandleFunc("GET /api/v1/merchant/stores/{storeId}/delivery-templates/{templateRef}", s.catalogTemplateReadV2)
	mux.HandleFunc("PUT /api/v1/merchant/stores/{storeId}/delivery-templates/{templateRef}", s.catalogTemplateWriteV2(false))
	mux.HandleFunc("POST /api/v1/merchant/stores/{storeId}/delivery-templates/{templateRef}/disable", s.catalogTemplateDisableV2)
}
func (s *Server) catalogTemplateReadV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	d, e := s.Store.CatalogGetRecordV2(r.Context(), u.User.ID, r.PathValue("templateRef"), "delivery_template", true)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	if d.StoreID != r.PathValue("storeId") {
		catalogFailV2(w, r, &store.CatalogErrorV2{Status: 404, Code: "NOT_FOUND", Message: "交付模板不属于此店铺。"})
		return
	}
	v2Success(w, r, store.CatalogTemplateViewV2(d, true))
}
func (s *Server) catalogTemplateWriteV2(create bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		input, e := catalogReadV2(w, r)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		key, version, content, e := catalogFlatMutationV2(input, create)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		nodes := map[string]store.CatalogNodePolicyV2{}
		for _, n := range s.Config.Nodes {
			nodes[n.ID] = store.CatalogNodePolicyV2{InventoryDomain: n.InventoryDomain, ClaimEnabled: n.ClaimEnabled}
		}
		d, e := s.Store.CatalogWriteTemplateV2(r.Context(), u.User.ID, r.PathValue("storeId"), r.PathValue("templateRef"), key, version, content, nodes)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		status := 200
		if create {
			status = 201
		}
		catalogSuccessV2(w, r, status, store.CatalogTemplateViewV2(d, true))
	}
}
func (s *Server) catalogTemplateDisableV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	input, e := catalogReadV2(w, r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	key, version, _, e := store.CatalogMutationInputV2(input, false, true, "")
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	d, e := s.Store.CatalogDisableTemplateV2(r.Context(), u.User.ID, r.PathValue("storeId"), r.PathValue("templateRef"), key, version)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2Success(w, r, store.CatalogTemplateViewV2(d, true))
}

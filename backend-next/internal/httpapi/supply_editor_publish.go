package httpapi

import (
	"net/http"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

const supplyEditorPublicationPath = "/internal/v1/supply-editor/publish"

func (s *Server) registerSupplyEditorPublication(mux *http.ServeMux) {
	mux.HandleFunc("POST "+supplyEditorPublicationPath, s.publishSupplyEditorCatalog)
}

func (s *Server) publishSupplyEditorCatalog(w http.ResponseWriter, r *http.Request) {
	node, ok := s.supplyEditorNode(w, r)
	if !ok {
		return
	}
	var input store.SupplyEditorPublication
	if socialBodyV2(w, r, &input, 2<<20) != nil {
		failure(w, r, 400, "INVALID_SUPPLY_CATALOG", "卡池目录无效或超过大小限制。")
		return
	}
	deliveries := make(map[string]bool)
	for _, target := range s.Config.Nodes {
		if target.ClaimEnabled && target.InventoryDomain == node.InventoryDomain {
			deliveries[target.ID] = true
		}
	}
	out, err := s.Store.PublishSupplyEditor(r.Context(), node.ID, node.ItemPrefix, deliveries, input)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	v2Success(w, r, out)
}

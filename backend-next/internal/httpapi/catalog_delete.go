package httpapi

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
)

func (s *Server) deleteCatalogEntry(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		if kind == "coupon" {
			if _, e := s.admin(r, "platform.admin"); e != nil {
				failError(w, r, e)
				return
			}
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
		result, e := s.Store.DeleteCatalogEntry(r.Context(), u.User.ID, r.PathValue(kind+"Id"), kind, key, version)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		v2Success(w, r, result)
	}
}

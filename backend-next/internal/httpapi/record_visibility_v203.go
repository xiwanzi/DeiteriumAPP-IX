package httpapi

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
)

func (s *Server) hideRecordV203(kind, idParam string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		input, err := catalogReadV2(w, r)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		key, version, err := store.CommerceActionInputV2(input, "hide")
		if err == nil {
			err = s.Store.HideRecordV203(r.Context(), session.User.ID, kind, r.PathValue(idParam), key, version)
		}
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, map[string]bool{"hidden": true})
	}
}

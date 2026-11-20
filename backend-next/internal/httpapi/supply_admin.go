package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerSupplyAdmin(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/supply/pools", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		items, err := s.Store.SupplyPools(r.Context(), true)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": items})
	})
	change := func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.SupplyPoolChange
		if socialBodyV2(w, r, &in, 262144) != nil {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		out, err := s.Store.ChangeSupplyPool(r.Context(), actor.ID, r.PathValue("poolId"), in)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	}
	mux.HandleFunc("POST /api/v1/admin/supply/pools", change)
	mux.HandleFunc("POST /api/v1/admin/supply/pools/{poolId}", change)
	mux.HandleFunc("GET /api/v1/admin/supply/pools/{poolId}/versions", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		items, err := s.Store.SupplyPoolHistory(r.Context(), r.PathValue("poolId"))
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": items})
	})
	mux.HandleFunc("GET /api/v1/admin/supply/pools/{poolId}/versions/{revision}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		rev, err := strconv.ParseInt(r.PathValue("revision"), 10, 64)
		if err != nil || rev < 1 {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		d, err := s.Store.SupplyPoolRevision(r.Context(), r.PathValue("poolId"), rev)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, d)
	})
	mux.HandleFunc("GET /api/v1/admin/supply/settings", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.SupplySettings(r.Context())
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("POST /api/v1/admin/supply/settings", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string               `json:"clientRequestId"`
			Settings        store.SupplySettings `json:"settings"`
		}
		if socialBodyV2(w, r, &in, 4096) != nil {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		out, err := s.Store.SaveSupplySettings(r.Context(), actor.ID, in.ClientRequestID, in.Settings)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("GET /api/v1/admin/supply/players", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.SupplyPlayerSearch(r.Context(), r.URL.Query().Get("q"))
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, map[string]any{"items": out})
	})
	mux.HandleFunc("GET /api/v1/admin/supply/players/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.SupplyAccount(r.Context(), r.PathValue("uuid"))
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("POST /api/v1/admin/supply/tickets", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in store.SupplyAdjustment
		if socialBodyV2(w, r, &in, 8192) != nil {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		out, err := s.Store.AdjustSupplyTickets(r.Context(), actor.ID, in, true)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	for _, kind := range []string{"orders", "ledger"} {
		mux.HandleFunc("GET /api/v1/admin/supply/"+kind, func(w http.ResponseWriter, r *http.Request) {
			if _, err := s.admin(r, "platform.admin"); err != nil {
				failError(w, r, err)
				return
			}
			f, err := supplyFilter(r)
			if err != nil {
				socialFailureV2(w, r, err)
				return
			}
			var out map[string]any
			if kind == "orders" {
				out, err = s.Store.SupplyOrders(r.Context(), f)
			} else {
				out, err = s.Store.SupplyLedger(r.Context(), f)
			}
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			v2Success(w, r, out)
		})
	}
	mux.HandleFunc("GET /api/v1/admin/supply/orders/{orderId}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		out, err := s.Store.SupplyOrder(r.Context(), r.PathValue("orderId"), "")
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
	mux.HandleFunc("POST /api/v1/admin/supply/deliveries/{deliveryId}/retry", func(w http.ResponseWriter, r *http.Request) {
		actor, err := s.admin(r, "platform.admin")
		if err != nil {
			failError(w, r, err)
			return
		}
		var in struct {
			ClientRequestID string `json:"clientRequestId"`
			Reason          string `json:"reason"`
		}
		if socialBodyV2(w, r, &in, 4096) != nil {
			socialFailureV2(w, r, store.ErrSocialInvalid)
			return
		}
		out, err := s.Store.RetrySupplyDelivery(r.Context(), actor.ID, in.ClientRequestID, r.PathValue("deliveryId"), in.Reason)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, out)
	})
}

func supplyFilter(r *http.Request) (store.SupplyFilter, error) {
	q := r.URL.Query()
	f := store.SupplyFilter{PlayerUUID: q.Get("playerUuid"), PoolID: q.Get("poolId"), Scheme: q.Get("scheme"), Currency: q.Get("currency"), State: q.Get("state"), Limit: 25}
	for key, target := range map[string]*int64{"before": &f.Before, "snapshot": &f.Snapshot} {
		if q.Get(key) != "" {
			v, err := strconv.ParseInt(q.Get(key), 10, 64)
			if err != nil {
				return f, store.ErrSocialInvalid
			}
			*target = v
		}
	}
	if q.Get("limit") != "" {
		n, err := strconv.Atoi(q.Get("limit"))
		if err != nil {
			return f, store.ErrSocialInvalid
		}
		f.Limit = n
	}
	for key, target := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if q.Get(key) != "" {
			v, err := time.Parse(time.RFC3339, q.Get(key))
			if err != nil {
				return f, store.ErrSocialInvalid
			}
			*target = &v
		}
	}
	return f, nil
}

func supplyRawObject(raw json.RawMessage) (map[string]any, error) {
	var value map[string]any
	err := json.Unmarshal(raw, &value)
	return value, err
}

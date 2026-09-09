package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerAdminCommerceV204(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/players", s.adminPlayersV204)
	mux.HandleFunc("GET /api/v1/admin/orders", s.adminCommerceListV204("orders"))
	mux.HandleFunc("GET /api/v1/admin/products", s.adminCommerceListV204("products"))
	mux.HandleFunc("GET /api/v1/admin/orders/{resourceId}", s.adminCommerceDetailV204("orders"))
	mux.HandleFunc("GET /api/v1/admin/products/{resourceId}", s.adminCommerceDetailV204("products"))
	mux.HandleFunc("GET /api/v1/admin/interventions/summary", s.adminInterventionSummaryV204)
}

type adminPlayersCursorV204 struct {
	Actor string `json:"actor"`
	Query string `json:"q"`
	After string `json:"after"`
}

func encodeCursorV204(value any) string {
	raw, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(raw)
}
func decodeCursorV204(raw string, value any) error {
	if len(raw) > 4096 {
		return bridge.ErrProtocol
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	return bridge.Decode(data, value)
}
func (s *Server) adminPlayersV204(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "audit.read")
	if err != nil {
		failError(w, r, err)
		return
	}
	q := r.URL.Query()
	for k, v := range q {
		if len(v) != 1 || (k != "q" && k != "cursor") {
			failError(w, r, bridge.ErrProtocol)
			return
		}
	}
	c := adminPlayersCursorV204{Actor: u.ID, Query: strings.TrimSpace(q.Get("q"))}
	if raw := q.Get("cursor"); raw != "" {
		if decodeCursorV204(raw, &c) != nil || c.Actor != u.ID || (q.Has("q") && q.Get("q") != c.Query) {
			failError(w, r, bridge.ErrProtocol)
			return
		}
	}
	players, err := s.Store.AdminPlayersV204(r.Context(), u.ID, c.Query, c.After, 26)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	more := len(players) > 25
	var next any
	if more {
		players = players[:25]
		c.After = players[24].UUID
		next = encodeCursorV204(c)
	}
	v2List(w, r, players, next, more)
}

type adminCommerceCursorV204 struct {
	Actor      string                        `json:"actor"`
	Collection string                        `json:"collection"`
	Filter     store.AdminCommerceFilterV204 `json:"filter"`
}

func adminCommerceQueryV204(r *http.Request, actor, collection string) (adminCommerceCursorV204, int, error) {
	c := adminCommerceCursorV204{Actor: actor, Collection: collection}
	limit := 25
	invalid := func() (adminCommerceCursorV204, int, error) { return c, 0, bridge.ErrProtocol }
	q := r.URL.Query()
	for k, v := range q {
		if len(v) != 1 {
			return invalid()
		}
		switch k {
		case "playerRef", "q", "kind", "channel", "status", "cursor", "limit":
		default:
			return invalid()
		}
	}
	if raw := q.Get("cursor"); raw != "" {
		if decodeCursorV204(raw, &c) != nil || c.Actor != actor || c.Collection != collection || c.Filter.BeforeSequence <= 0 {
			return invalid()
		}
	}
	for _, v := range []struct {
		key   string
		value *string
	}{{"playerRef", &c.Filter.PlayerRef}, {"q", &c.Filter.Query}, {"kind", &c.Filter.Kind}, {"channel", &c.Filter.Channel}, {"status", &c.Filter.Status}} {
		if q.Has(v.key) {
			if c.Filter.BeforeSequence > 0 && *v.value != q.Get(v.key) {
				return invalid()
			}
			*v.value = q.Get(v.key)
		}
	}
	if !c.Filter.Valid() {
		return invalid()
	}
	if c.Filter.Kind != "" && (collection != "products" || (c.Filter.Kind != "product" && c.Filter.Kind != "listing")) {
		return invalid()
	}
	if c.Filter.Channel != "" && (collection != "orders" || (c.Filter.Channel != "OFFICIAL_STORE" && c.Filter.Channel != "PLAYER_MARKET")) {
		return invalid()
	}
	if q.Has("limit") {
		n, e := strconv.Atoi(q.Get("limit"))
		if e != nil || n < 1 || n > 50 {
			return invalid()
		}
		limit = n
	}
	return c, limit, nil
}
func (s *Server) adminCommerceListV204(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "audit.read")
		if err != nil {
			failError(w, r, err)
			return
		}
		cursor, limit, err := adminCommerceQueryV204(r, u.ID, collection)
		if err != nil {
			failError(w, r, err)
			return
		}
		values := []map[string]any{}
		var next any
		more := false
		if collection == "orders" {
			records, err := s.Store.AdminOrdersV204(r.Context(), u.ID, cursor.Filter, limit+1)
			if err != nil {
				socialFailureV2(w, r, err)
				return
			}
			more = len(records) > limit
			if more {
				records = records[:limit]
				last := records[len(records)-1]
				cursor.Filter.BeforeTime = last.CreatedAt
				cursor.Filter.BeforeSequence = last.Sequence
				next = encodeCursorV204(cursor)
			}
			for _, d := range records {
				values = append(values, map[string]any{"orderId": d.ID, "channel": d.Channel, "status": d.State, "fundsStatus": d.FundsState, "amount": d.Amount, "settledAmount": d.SettledAmount, "refundedAmount": d.RefundedAmount, "createdAt": d.CreatedAt, "updatedAt": d.UpdatedAt, "buyer": d.Body["buyer"], "seller": d.Body["seller"], "items": d.Body["items"], "storeId": d.StoreID, "interventionCaseId": d.InterventionCaseID})
			}
		} else {
			records, err := s.Store.AdminProductsV204(r.Context(), u.ID, cursor.Filter, limit+1)
			if err != nil {
				socialFailureV2(w, r, err)
				return
			}
			more = len(records) > limit
			if more {
				records = records[:limit]
				last := records[len(records)-1]
				cursor.Filter.BeforeTime = last.CreatedAt
				cursor.Filter.BeforeSequence = last.Sequence
				next = encodeCursorV204(cursor)
			}
			for _, d := range records {
				values = append(values, map[string]any{"productId": d.ID, "kind": d.Kind, "title": d.Body["title"], "price": d.Body["price"], "stock": d.Stock, "state": d.State, "storeId": d.StoreID, "createdAt": d.CreatedAt, "updatedAt": d.UpdatedAt})
			}
		}
		v2List(w, r, values, next, more)
	}
}
func (s *Server) adminCommerceDetailV204(collection string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.admin(r, "audit.read")
		if err != nil {
			failError(w, r, err)
			return
		}
		id := r.PathValue("resourceId")
		if !store.CatalogReferenceV2(id) {
			failError(w, r, bridge.ErrProtocol)
			return
		}
		var view map[string]any
		if collection == "orders" {
			view, err = s.Store.AdminCommerceViewV204(r.Context(), u.ID, id)
			if err == nil && view["orderId"] != id {
				failure(w, r, 404, "NOT_FOUND", "订单不存在。")
				return
			}
		} else {
			view, err = s.Store.AdminProductV204(r.Context(), u.ID, id)
		}
		if err != nil {
			socialFailureV2(w, r, err)
			return
		}
		v2Success(w, r, view)
	}
}
func (s *Server) adminInterventionSummaryV204(w http.ResponseWriter, r *http.Request) {
	if _, err := s.admin(r, "intervention.manage"); err != nil {
		failError(w, r, err)
		return
	}
	var pending, submitted int
	err := s.Store.DB.QueryRowContext(r.Context(), `SELECT COUNT(*),COALESCE(SUM(state='SUBMITTED'),0) FROM commerce_interventions_v2 WHERE state NOT IN ('RESOLVED','WITHDRAWN')`).Scan(&pending, &submitted)
	if err != nil {
		failError(w, r, err)
		return
	}
	v2Success(w, r, map[string]any{"pending": pending, "submitted": submitted})
}

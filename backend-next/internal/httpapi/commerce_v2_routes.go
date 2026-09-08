package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func (s *Server) registerCommerceV2(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/orders/{orderId}/hide", s.hideRecordV203("ORDER", "orderId"))
	mux.HandleFunc("POST /api/v1/commissions/{commissionId}/hide", s.hideRecordV203("COMMISSION", "commissionId"))
	mux.HandleFunc("POST /api/v1/market/listings/{listingId}/hide", s.hideRecordV203("LISTING", "listingId"))
	mux.HandleFunc("POST /api/v1/store/orders", s.commerceCreateOrderV2("OFFICIAL_STORE"))
	mux.HandleFunc("POST /api/v1/market/orders", s.commerceCreateOrderV2("PLAYER_MARKET"))
	mux.HandleFunc("GET /api/v1/orders", s.commerceListHTTPV2("ORDER", false, false))
	mux.HandleFunc("GET /api/v1/commissions", s.commerceListHTTPV2("COMMISSION", true, false))
	mux.HandleFunc("GET /api/v1/commissions/me", s.commerceListHTTPV2("COMMISSION", false, false))
	mux.HandleFunc("POST /api/v1/commissions", s.commerceCreateCommissionV2)
	mux.HandleFunc("GET /api/v1/merchant/orders", s.commerceListHTTPV2("ORDER", false, true))
	mux.HandleFunc("GET /api/v1/merchant/orders/{orderId}", s.commerceGetHTTPV2("ORDER", "view"))
	mux.HandleFunc("POST /api/v1/merchant/orders/{orderId}/delivery-retry", s.commerceActionHTTPV2("ORDER", "delivery-retry"))
	mux.HandleFunc("GET /api/v1/operations/{operationId}", s.commerceOperationHTTPV2)
	mux.HandleFunc("GET /api/v1/operations/by-client-request", s.commerceOperationHTTPV2)
	for _, group := range []struct {
		path, id, kind string
		actions        []string
	}{
		{"orders", "orderId", "ORDER", []string{"ship", "start-work", "complete-work", "confirm"}},
		{"commissions", "commissionId", "COMMISSION", []string{"accept", "cancel", "complete", "confirm"}},
	} {
		base := "/api/v1/" + group.path + "/{" + group.id + "}"
		mux.HandleFunc("GET "+base, s.commerceGetHTTPV2(group.kind, "view"))
		mux.HandleFunc("GET "+base+"/snapshot", s.commerceGetHTTPV2(group.kind, "snapshot"))
		mux.HandleFunc("GET "+base+"/refunds/{refundId}", s.commerceGetHTTPV2(group.kind, "refund"))
		mux.HandleFunc("POST "+base+"/refunds", s.commerceActionHTTPV2(group.kind, "refund"))
		mux.HandleFunc("POST "+base+"/refunds/{refundId}/resolve", s.commerceActionHTTPV2(group.kind, "resolve-refund"))
		mux.HandleFunc("POST "+base+"/refunds/{refundId}/withdraw", s.commerceActionHTTPV2(group.kind, "withdraw-refund"))
		for _, action := range group.actions {
			mux.HandleFunc("POST "+base+"/"+action, s.commerceActionHTTPV2(group.kind, action))
		}
	}
	mux.HandleFunc("GET /api/v1/orders/{orderId}/mailbox", s.commerceGetHTTPV2("ORDER", "mailbox"))
	mux.HandleFunc("POST /api/v1/orders/{orderId}/interventions", s.commerceCaseCreateHTTPV2("ORDER"))
	mux.HandleFunc("POST /api/v1/commissions/{commissionId}/interventions", s.commerceCaseCreateHTTPV2("COMMISSION"))
	mux.HandleFunc("GET /api/v1/interventions/{caseId}", s.commerceCaseGetHTTPV2(false))
	mux.HandleFunc("GET /api/v1/admin/interventions/{caseId}", s.commerceCaseGetHTTPV2(true))
	mux.HandleFunc("GET /api/v1/admin/interventions", s.commerceCaseListHTTPV2)
	for _, action := range []string{"evidence", "withdraw"} {
		mux.HandleFunc("POST /api/v1/interventions/{caseId}/"+action, s.commerceCaseActionHTTPV2(action, false))
	}
	for _, action := range []string{"assign", "request-evidence", "resolve"} {
		mux.HandleFunc("POST /api/v1/admin/interventions/{caseId}/"+action, s.commerceCaseActionHTTPV2(action, true))
	}
}

func (s *Server) commerceAvailableHTTPV2(command string) bool {
	return s.CommerceCore != nil && s.CommerceCore.Available(command)
}

func commerceReferenceHTTPV2(r *http.Request, kind string) string {
	if kind == "COMMISSION" {
		return r.PathValue("commissionId")
	}
	return r.PathValue("orderId")
}

func (s *Server) commerceCreateOrderV2(channel string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		input, err := catalogReadV2(w, r)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		key, quote, version, err := store.CommerceCreateOrderInputV2(input)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		nodes := map[string]store.CatalogNodePolicyV2{}
		for _, n := range s.Config.Nodes {
			nodes[n.ID] = store.CatalogNodePolicyV2{InventoryDomain: n.InventoryDomain, ClaimEnabled: n.ClaimEnabled}
		}
		available := s.commerceAvailableHTTPV2(CommerceReserveV2)
		if channel == "OFFICIAL_STORE" {
			available = available && s.commerceAvailableHTTPV2("mailbox.create") && s.commerceAvailableHTTPV2("mailbox.revoke")
		}
		mutation, err := s.Store.PrepareOrderV2(r.Context(), u.User.ID, key, quote, version, channel, available, nodes)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		s.commerceMutationHTTPV2(w, r, u.User.ID, mutation, true, true)
	}
}

func (s *Server) commerceCreateCommissionV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	input, err := catalogReadV2(w, r)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	key := store.CommerceStringV2(input, "clientRequestId")
	content, ok := store.CommerceContentV2(input, "content")
	if !ok || len(input) != 2 || !store.CatalogReferenceV2(key) {
		catalogFailV2(w, r, bridge.ErrProtocol)
		return
	}
	if err = store.ValidateCommissionContentV2(content); err != nil {
		catalogFailV2(w, r, err)
		return
	}
	mutation, err := s.Store.PrepareCommissionV2(r.Context(), u.User.ID, key, content, s.commerceAvailableHTTPV2(CommerceReserveV2))
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	s.commerceMutationHTTPV2(w, r, u.User.ID, mutation, true, true)
}

func (s *Server) commerceActionHTTPV2(kind, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		id := commerceReferenceHTTPV2(r, kind)
		if !store.CatalogReferenceV2(id) {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		input, err := catalogReadV2(w, r)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		key, version, err := store.CommerceActionInputV2(input, action)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		var mutation store.CommerceMutationV2
		bundle := false
		switch action {
		case "ship", "start-work", "complete-work", "complete":
			mutation, err = s.Store.CommerceFulfillmentV2(r.Context(), u.User.ID, id, kind, key, action, version, input)
		case "accept":
			mutation, err = s.Store.PrepareCommissionAcceptV2(r.Context(), u.User.ID, id, key, version, s.commerceAvailableHTTPV2(CommerceBindV2))
		case "confirm":
			bundle = true
			mutation, err = s.Store.PrepareCommerceSettlementV2(r.Context(), u.User.ID, id, kind, key, version, s.commerceAvailableHTTPV2(CommerceSettleV2), false, time.Now().UTC())
		case "cancel":
			bundle = true
			mutation, err = s.Store.PrepareCommissionCancelV2(r.Context(), u.User.ID, id, key, version, s.commerceAvailableHTTPV2(CommerceRefundV2))
		case "refund":
			available := s.commerceAvailableHTTPV2(CommerceRefundV2)
			if kind == "ORDER" {
				resource, lookupError := s.Store.CommerceRecordV2(r.Context(), id)
				if lookupError != nil {
					catalogFailV2(w, r, lookupError)
					return
				}
				if resource.Channel == "OFFICIAL_STORE" {
					available = available && s.commerceAvailableHTTPV2("mailbox.revoke")
				}
			}
			mutation, err = s.Store.PrepareCommerceRefundV2(r.Context(), u.User.ID, id, kind, key, version, input, available)
		case "resolve-refund":
			mutation, err = s.Store.ResolveCommerceRefundV2(r.Context(), u.User.ID, id, kind, r.PathValue("refundId"), key, version, store.CommerceStringV2(input, "decision"), store.CommerceStringV2(input, "reason"), s.commerceAvailableHTTPV2(CommerceRefundV2))
		case "withdraw-refund":
			mutation, err = s.Store.WithdrawCommerceRefundV2(r.Context(), u.User.ID, id, kind, r.PathValue("refundId"), key, version)
		case "delivery-retry":
			bundle = true
			mutation, err = s.Store.PrepareDeliveryRetryV2(r.Context(), u.User.ID, id, key, version, store.CommerceStringV2(input, "reason"), s.commerceAvailableHTTPV2("mailbox.create"))
		default:
			err = bridge.ErrProtocol
		}
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		s.commerceMutationHTTPV2(w, r, u.User.ID, mutation, bundle, false)
	}
}

func (s *Server) commerceMutationHTTPV2(w http.ResponseWriter, r *http.Request, user string, mutation store.CommerceMutationV2, bundle, created bool) {
	var operation *store.CommerceOperationV2
	if mutation.OperationID != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		op, err := s.RunCommerceOperationV2(ctx, mutation.OperationID, true)
		cancel()
		if err != nil {
			op, err = s.Store.CommerceOperationV2(r.Context(), mutation.OperationID)
		}
		if err != nil {
			writeJSON(w, 503, map[string]any{"requestId": requestID(r), "error": map[string]any{"code": "RESULT_UNKNOWN", "message": "请求已受理，请查询原操作确认结果。", "details": map[string]string{"operationId": mutation.OperationID}}})
			return
		}
		operation = &op
	}
	view, err := s.Store.CommerceViewV2(r.Context(), user, mutation.ResourceID, false)
	if err != nil && (!bundle || operation == nil) {
		catalogFailV2(w, r, err)
		return
	}
	status := http.StatusOK
	if created && !mutation.Replayed {
		status = http.StatusCreated
	}
	if operation != nil && (operation.State == "PROCESSING" || operation.State == "UNKNOWN") {
		status = http.StatusAccepted
	}
	if !bundle {
		catalogSuccessV2(w, r, status, view)
		return
	}
	var operationView any
	if operation != nil {
		operationView = store.CommerceOperationViewV2(*operation, mutation.Kind)
	}
	resourceKey := "order"
	if mutation.Kind == "COMMISSION" {
		resourceKey = "commission"
	}
	var resourceView any = view
	if err != nil {
		resourceView = nil
	}
	catalogSuccessV2(w, r, status, map[string]any{"operation": operationView, resourceKey: resourceView})
}

func (s *Server) commerceOperationHTTPV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	id, key, kind := r.PathValue("operationId"), r.URL.Query().Get("clientRequestId"), r.URL.Query().Get("kind")
	if (id != "" && !store.CatalogReferenceV2(id)) || (id == "" && (!store.CatalogReferenceV2(key) || !commerceOneOfV2(kind, "STORE_PURCHASE", "MARKET_PURCHASE", "COMMISSION_PUBLISH", "COMMISSION_ACCEPT", "REFUND", "SETTLEMENT"))) {
		catalogFailV2(w, r, bridge.ErrProtocol)
		return
	}
	op, resource, err := s.Store.CommerceVisibleOperationV2(r.Context(), u.User.ID, id, key, kind)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	current, err := s.RunCommerceOperationV2(ctx, op.ID, false)
	cancel()
	if err == nil {
		op = current
	}
	v2Success(w, r, store.CommerceOperationViewV2(op, resource.Kind))
}

func commerceOneOfV2(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func (s *Server) commerceGetHTTPV2(kind, viewKind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		id := commerceReferenceHTTPV2(r, kind)
		if !store.CatalogReferenceV2(id) {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/merchant/") {
			resource, err := s.Store.CommerceRecordV2(r.Context(), id)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			if resource.Kind != "ORDER" || resource.Channel != "OFFICIAL_STORE" {
				failure(w, r, 404, "NOT_FOUND", "店铺订单不存在。")
				return
			}
			if err = s.Store.CatalogCanManageV2(r.Context(), u.User.ID, resource.StoreID, "ORDER_MANAGE"); err != nil {
				catalogFailV2(w, r, err)
				return
			}
		}
		// Authorize before reading a pending operation or a mailbox receipt.
		public := kind == "COMMISSION" && viewKind == "view"
		view, err := s.Store.CommerceViewV2(r.Context(), u.User.ID, id, public)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		if (kind == "ORDER" && view["orderId"] != id) || (kind == "COMMISSION" && view["commissionId"] != id) {
			failure(w, r, 404, "NOT_FOUND", "交易不存在或无权访问。")
			return
		}
		if kind == "ORDER" && view["channel"] == "OFFICIAL_STORE" && (viewKind == "view" || viewKind == "mailbox") {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			_ = s.RefreshCommerceMailboxV2(ctx, id)
			cancel()
			view, err = s.Store.CommerceViewV2(r.Context(), u.User.ID, id, false)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
		}
		switch viewKind {
		case "view":
			v2Success(w, r, view)
		case "snapshot":
			snapshot, err := s.Store.CommerceSnapshotV2(r.Context(), u.User.ID, id, false)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			v2Success(w, r, snapshot)
		case "refund":
			refund, err := s.Store.CommerceRefundViewV2(r.Context(), u.User.ID, id, r.PathValue("refundId"))
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			v2Success(w, r, refund)
		case "mailbox":
			if view["channel"] != "OFFICIAL_STORE" {
				failure(w, r, 404, "NOT_FOUND", "此交易没有游戏邮箱交付。")
				return
			}
			v2Success(w, r, view)
		}
	}
}

func (s *Server) commerceListHTTPV2(kind string, public, merchant bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		f := store.CommerceFilterV2{Kind: kind, Public: public, Channel: q.Get("channel"), Role: q.Get("role"), State: q.Get("status"), Query: strings.TrimSpace(q.Get("q")), Urgency: q.Get("urgency"), Sort: defaultValue(q.Get("sort"), "NEWEST")}
		if f.Query == "" {
			f.Query = strings.TrimSpace(q.Get("query"))
		}
		if f.Channel != "" && !commerceOneOfV2(f.Channel, "OFFICIAL_STORE", "PLAYER_MARKET") {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		if utf8.RuneCountInString(f.Query) > 80 {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		if kind == "ORDER" {
			if raw, exists := q["hasRefund"]; exists {
				if len(raw) != 1 || (raw[0] != "true" && raw[0] != "false") {
					catalogFailV2(w, r, bridge.ErrProtocol)
					return
				}
				flag := raw[0] == "true"
				f.HasRefund = &flag
			}
			if f.Role != "" && !commerceOneOfV2(f.Role, "BUYER", "SELLER") {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
			if f.State != "" && !commerceOneOfV2(f.State, "UNPAID", "PAYMENT_PROCESSING", "AWAITING_SHIPMENT", "SHIPPED", "WORK_COMPLETED", "CONFIRMED", "AWAITING_CLAIM", "CLAIMED", "REFUNDED", "CANCELLED", "REFUNDING", "SETTLING") {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		} else {
			if f.Urgency != "" && !commerceOneOfV2(f.Urgency, "NORMAL", "SOON", "URGENT") {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
			if f.Role == "PUBLISHER" {
				f.Role = "OWNER"
			}
			if f.Role != "" && !commerceOneOfV2(f.Role, "OWNER", "WORKER", "PUBLISHED", "ACCEPTED") {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
			if public && (f.State != "" && f.State != "OPEN") {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
			if f.State != "" && !commerceOneOfV2(f.State, "UNPAID", "FUNDING", "OPEN", "ACTIVE", "COMPLETED", "CONFIRMED", "CANCELLED", "REFUNDED", "REFUNDING", "SETTLING") {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		}
		if !commerceOneOfV2(f.Sort, "NEWEST", "REWARD_DESC", "REWARD_ASC") || (kind == "ORDER" && f.Sort != "NEWEST") {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		if merchant {
			f.StoreID = q.Get("storeId")
			f.Channel = "OFFICIAL_STORE"
			if !store.CatalogReferenceV2(f.StoreID) {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		}
		encoded, _ := json.Marshal(f)
		scope := "commerce:" + r.URL.Path + ":" + u.User.ID + ":" + store.Digest(encoded)
		limit, err := commerceLimitHTTPV2(r)
		if err != nil {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		f.Limit = limit + 1
		f.Before, f.BeforeAmount, err = store.CommerceParseCursorV2(scope, q.Get("cursor"), f.Sort)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		items, err := s.Store.CommerceListV2(r.Context(), u.User.ID, f)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		var next any
		if len(items) > limit {
			items = items[:limit]
			next = store.CommerceCursorV2(scope, items[len(items)-1], f.Sort)
		}
		views := []map[string]any{}
		for _, item := range items {
			view, err := s.Store.CommerceViewV2(r.Context(), u.User.ID, item.ID, public)
			if err != nil {
				catalogFailV2(w, r, err)
				return
			}
			views = append(views, view)
		}
		v2List(w, r, views, next, next != nil)
	}
}

func commerceLimitHTTPV2(r *http.Request) (int, error) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return 0, bridge.ErrProtocol
		}
		limit = n
	}
	return limit, nil
}

func (s *Server) commerceCaseCreateHTTPV2(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		input, err := catalogReadV2(w, r)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		key, version, err := store.CommerceInterventionInputV2(input, "create")
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		mutation, err := s.Store.CreateCommerceInterventionV2(r.Context(), u.User.ID, commerceReferenceHTTPV2(r, kind), kind, key, version, input)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		view, err := s.Store.CommerceCaseViewV2(r.Context(), u.User.ID, mutation.ResourceID, false)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		status := http.StatusCreated
		if mutation.Replayed {
			status = http.StatusOK
		}
		catalogSuccessV2(w, r, status, view)
	}
}

func (s *Server) commerceCaseGetHTTPV2(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		if admin {
			if _, err := s.admin(r, "intervention.manage"); err != nil {
				catalogFailV2(w, r, err)
				return
			}
		}
		view, err := s.Store.CommerceCaseViewV2(r.Context(), u.User.ID, r.PathValue("caseId"), admin)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, view)
	}
}

func (s *Server) commerceCaseActionHTTPV2(action string, admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		if admin {
			if _, err := s.admin(r, "intervention.manage"); err != nil {
				catalogFailV2(w, r, err)
				return
			}
		}
		input, err := catalogReadV2(w, r)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		key, version, err := store.CommerceInterventionInputV2(input, action)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		available := s.commerceAvailableHTTPV2(CommerceRefundV2) && s.commerceAvailableHTTPV2(CommerceSettleV2)
		mutation, err := s.Store.CommerceCaseActionV2(r.Context(), u.User.ID, r.PathValue("caseId"), key, action, version, input, available)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		if mutation.OperationID != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
			_, _ = s.RunCommerceOperationV2(ctx, mutation.OperationID, true)
			cancel()
		}
		view, err := s.Store.CommerceCaseViewV2(r.Context(), u.User.ID, mutation.ResourceID, admin)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		v2Success(w, r, view)
	}
}

func (s *Server) commerceCaseListHTTPV2(w http.ResponseWriter, r *http.Request) {
	u, err := s.admin(r, "intervention.manage")
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	q := r.URL.Query()
	f := store.CommerceCaseFilterV2{State: q.Get("status"), TransactionID: q.Get("transactionId")}
	if raw, exists := q["assignedToMe"]; exists {
		if len(raw) != 1 || (raw[0] != "true" && raw[0] != "false") {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		f.AssignedToMe = raw[0] == "true"
	}
	if (f.State != "" && !commerceOneOfV2(f.State, "SUBMITTED", "IN_REVIEW", "WAITING_EVIDENCE", "RESOLVING", "RESOLVED", "WITHDRAWN")) || (f.TransactionID != "" && !store.CatalogReferenceV2(f.TransactionID)) {
		catalogFailV2(w, r, bridge.ErrProtocol)
		return
	}
	encoded, _ := json.Marshal(f)
	scope := "commerce-cases:" + u.ID + ":" + store.Digest(encoded)
	limit, cursor, err := socialPageV2(r, scope)
	if err != nil {
		catalogFailV2(w, r, bridge.ErrProtocol)
		return
	}
	f.Limit = limit + 1
	f.Before = cursor.Before
	items, err := s.Store.CommerceCaseListV2(r.Context(), u.ID, f)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	var next any
	if len(items) > limit {
		items = items[:limit]
		next = encodeSocialCursorV2(socialCursorV2{Scope: scope, Before: items[len(items)-1].Sequence})
	}
	views := []map[string]any{}
	for _, item := range items {
		view, err := s.Store.CommerceCaseViewV2(r.Context(), u.ID, item.ID, true)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		views = append(views, view)
	}
	v2List(w, r, views, next, next != nil)
}

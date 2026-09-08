package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/bridge"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Catalog bodies can contain complete product descriptions. They still have a
// hard byte/depth bound, reject duplicate object keys, and never accept HTML as
// executable content or client-provided sellers, prices in orders, or balances.
func catalogReadV2(w http.ResponseWriter, r *http.Request) (store.CatalogObjectV2, error) {
	if strings.TrimSpace(strings.ToLower(strings.Split(r.Header.Get("Content-Type"), ";")[0])) != "application/json" {
		return nil, bridge.ErrProtocol
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512<<10)
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	v, e := catalogJSONValueV2(dec, 0)
	if e != nil {
		return nil, bridge.ErrProtocol
	}
	if _, e = dec.Token(); e != io.EOF {
		return nil, bridge.ErrProtocol
	}
	o, ok := v.(map[string]any)
	if !ok {
		return nil, bridge.ErrProtocol
	}
	return store.CatalogObjectV2(o), nil
}
func catalogJSONValueV2(d *json.Decoder, depth int) (any, error) {
	if depth > 32 {
		return nil, bridge.ErrProtocol
	}
	t, e := d.Token()
	if e != nil {
		return nil, e
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return t, nil
	}
	switch delim {
	case '{':
		o := map[string]any{}
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return nil, e
			}
			k, ok := key.(string)
			if !ok {
				return nil, bridge.ErrProtocol
			}
			if _, exists := o[k]; exists {
				return nil, bridge.ErrProtocol
			}
			v, e := catalogJSONValueV2(d, depth+1)
			if e != nil {
				return nil, e
			}
			o[k] = v
		}
		end, e := d.Token()
		if e != nil || end != json.Delim('}') {
			return nil, bridge.ErrProtocol
		}
		return o, nil
	case '[':
		a := []any{}
		for d.More() {
			if len(a) >= 1000 {
				return nil, bridge.ErrProtocol
			}
			v, e := catalogJSONValueV2(d, depth+1)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		end, e := d.Token()
		if e != nil || end != json.Delim(']') {
			return nil, bridge.ErrProtocol
		}
		return a, nil
	default:
		return nil, bridge.ErrProtocol
	}
}
func catalogFailV2(w http.ResponseWriter, r *http.Request, e error) {
	var problem *store.CatalogErrorV2
	if errors.As(e, &problem) {
		failure(w, r, problem.Status, problem.Code, problem.Message)
		return
	}
	if errors.Is(e, store.ErrAssetUnavailable) || errors.Is(e, store.ErrAssetForbidden) || errors.Is(e, store.ErrAssetInUse) || errors.Is(e, store.ErrAssetBusy) || errors.Is(e, objectstorage.ErrUnavailable) {
		assetErrorV2(w, r, e)
		return
	}
	failError(w, r, e)
}
func catalogSuccessV2(w http.ResponseWriter, r *http.Request, status int, data any) {
	if status == http.StatusOK {
		v2Success(w, r, data)
		return
	}
	writeJSON(w, status, map[string]any{"requestId": requestID(r), "data": data, "serverTime": time.Now().UTC()})
}
func catalogHTTPIntV2(v any) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, e := n.Int64()
	return i, e == nil
}
func catalogHTTPStringV2(o store.CatalogObjectV2, k string) string { s, _ := o[k].(string); return s }
func catalogFlatMutationV2(input store.CatalogObjectV2, create bool) (string, int64, store.CatalogObjectV2, error) {
	key := catalogHTTPStringV2(input, "clientRequestId")
	if !store.CatalogReferenceV2(key) {
		return "", 0, nil, bridge.ErrProtocol
	}
	version := int64(0)
	if create {
		if _, exists := input["expectedVersion"]; exists {
			return "", 0, nil, bridge.ErrProtocol
		}
	} else {
		var ok bool
		version, ok = catalogHTTPIntV2(input["expectedVersion"])
		if !ok || version < 1 || version > 2147483647 {
			return "", 0, nil, bridge.ErrProtocol
		}
	}
	content := store.CatalogObjectV2{}
	for k, v := range input {
		if k != "clientRequestId" && k != "expectedVersion" {
			content[k] = v
		}
	}
	return key, version, content, nil
}
func (s *Server) catalogAuthV2(w http.ResponseWriter, r *http.Request) (store.Session, bool) {
	session, e := s.authenticate(r)
	if e != nil {
		catalogFailV2(w, r, e)
		return session, false
	}
	return session, true
}
func (s *Server) catalogRenderV2(w http.ResponseWriter, r *http.Request, user string, d store.CatalogRecordV2, management bool, status int) {
	view, e := s.Store.CatalogViewV2(r.Context(), user, d, management)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	catalogSuccessV2(w, r, status, view)
}

func (s *Server) registerCatalogV2(mux *http.ServeMux) {
	s.registerCatalogTemplatesV2(mux)
	mux.HandleFunc("GET /api/v1/store/stores", s.catalogListV2("store", false))
	mux.HandleFunc("GET /api/v1/store/stores/{storeId}", s.catalogGetV2("store", false))
	mux.HandleFunc("GET /api/v1/store/stores/{storeId}/homepage", s.catalogHomepageV2(false))
	mux.HandleFunc("GET /api/v1/store/brands", s.catalogListV2("brand", false))
	mux.HandleFunc("GET /api/v1/store/categories", s.catalogListV2("category", false))
	mux.HandleFunc("GET /api/v1/store/products", s.catalogListV2("product", false))
	mux.HandleFunc("GET /api/v1/store/products/{productId}", s.catalogGetV2("product", false))
	mux.HandleFunc("GET /api/v1/store/cart", s.catalogCartReadV2)
	mux.HandleFunc("PUT /api/v1/store/cart/items/{productId}", s.catalogCartWriteV2(false))
	mux.HandleFunc("POST /api/v1/store/cart/items/{productId}/remove", s.catalogCartWriteV2(true))
	mux.HandleFunc("POST /api/v1/checkout/quotes", s.catalogQuoteV2)
	mux.HandleFunc("GET /api/v1/market/categories", s.catalogCategoriesV2)
	mux.HandleFunc("GET /api/v1/market/listings", s.catalogListV2("listing", false))
	mux.HandleFunc("POST /api/v1/market/listings", s.catalogContentWriteV2("listing", true, false))
	mux.HandleFunc("GET /api/v1/market/listings/{listingId}", s.catalogGetV2("listing", false))
	mux.HandleFunc("PUT /api/v1/market/listings/{listingId}", s.catalogContentWriteV2("listing", false, false))
	mux.HandleFunc("GET /api/v1/market/me/listings", s.catalogListV2("listing", true))
	mux.HandleFunc("POST /api/v1/market/listings/{listingId}/unlist", s.catalogActionV2("listing", "unlist"))
	mux.HandleFunc("POST /api/v1/market/listings/{listingId}/republish", s.catalogContentWriteV2("listing", false, true))
	mux.HandleFunc("GET /api/v1/merchant/me", s.catalogMerchantV2)
	mux.HandleFunc("GET /api/v1/merchant/stores/{storeId}", s.catalogGetV2("store", true))
	mux.HandleFunc("PUT /api/v1/merchant/stores/{storeId}", s.catalogFlatWriteV2("store", false))
	mux.HandleFunc("GET /api/v1/merchant/stores/{storeId}/homepage", s.catalogHomepageV2(true))
	mux.HandleFunc("PUT /api/v1/merchant/stores/{storeId}/homepage", s.catalogHomepageWriteV2)
	for plural, kind := range map[string]string{"brands": "brand", "categories": "category"} {
		mux.HandleFunc("GET /api/v1/merchant/stores/{storeId}/"+plural, s.catalogListV2(kind, true))
		mux.HandleFunc("POST /api/v1/merchant/stores/{storeId}/"+plural, s.catalogFlatWriteV2(kind, true))
		mux.HandleFunc("PUT /api/v1/merchant/stores/{storeId}/"+plural+"/{entryId}", s.catalogFlatWriteV2(kind, false))
	}
	mux.HandleFunc("GET /api/v1/merchant/stores/{storeId}/delivery-templates", s.catalogTemplatesV2)
	mux.HandleFunc("GET /api/v1/merchant/stores/{storeId}/products", s.catalogListV2("product", true))
	mux.HandleFunc("POST /api/v1/merchant/stores/{storeId}/products", s.catalogContentWriteV2("product", true, false))
	mux.HandleFunc("GET /api/v1/merchant/products/{productId}", s.catalogGetV2("product", true))
	mux.HandleFunc("PUT /api/v1/merchant/products/{productId}", s.catalogContentWriteV2("product", false, false))
	for _, action := range []string{"publish", "unlist", "archive", "stock-adjustments"} {
		mux.HandleFunc("POST /api/v1/merchant/products/{productId}/"+action, s.catalogActionV2("product", action))
	}
	// Explicit admin bootstrap replaces preseeded "official" demo stores.
	mux.HandleFunc("POST /api/v1/admin/stores", s.catalogCreateStoreV2)
	mux.HandleFunc("GET /api/v1/admin/stores/{storeId}/members", s.catalogMembersReadV2)
	mux.HandleFunc("POST /api/v1/admin/stores/{storeId}/members", s.catalogMembersWriteV2(false))
	mux.HandleFunc("POST /api/v1/admin/stores/{storeId}/members/{playerRef}/revoke", s.catalogMembersWriteV2(true))
}
func catalogPathIDV2(r *http.Request, kind string) string {
	switch kind {
	case "store":
		return r.PathValue("storeId")
	case "product":
		return r.PathValue("productId")
	case "listing":
		return r.PathValue("listingId")
	default:
		return r.PathValue("entryId")
	}
}
func (s *Server) catalogGetV2(kind string, management bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		d, e := s.Store.CatalogGetRecordV2(r.Context(), u.User.ID, catalogPathIDV2(r, kind), kind, management)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		s.catalogRenderV2(w, r, u.User.ID, d, management, 200)
	}
}
func catalogLimitV2(r *http.Request) (int, error) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 || n > 100 {
			return 0, bridge.ErrProtocol
		}
		limit = n
	}
	return limit, nil
}
func (s *Server) catalogListV2(kind string, management bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		limit, e := catalogLimitV2(r)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		q := r.URL.Query()
		storeID := r.PathValue("storeId")
		if storeID == "" {
			storeID = q.Get("storeId")
		}
		f := store.CatalogFilterV2{Kind: kind, StoreID: storeID, BrandID: q.Get("brandId"), CategoryID: q.Get("categoryId"), SellerRef: q.Get("sellerPlayerRef"), Query: q.Get("q"), Management: management, Limit: limit + 1}
		if kind == "listing" {
			f.CategoryID = q.Get("categoryCode")
			if f.CategoryID != "" && !catalogAllowedCategoryV2(f.CategoryID) {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
			if management {
				if active, exists := q["active"]; exists {
					if len(active) != 1 || (active[0] != "true" && active[0] != "false") {
						catalogFailV2(w, r, bridge.ErrProtocol)
						return
					}
					f.State = "ACTIVE"
					if active[0] == "false" {
						f.State = "UNLISTED"
					}
				}
			}
		} else if management {
			f.State = q.Get("visibility")
			if f.State != "" && f.State != "DRAFT" && f.State != "ACTIVE" && f.State != "UNLISTED" && f.State != "ARCHIVED" {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		}
		for _, ref := range []string{f.StoreID, f.BrandID, f.CategoryID, f.SellerRef} {
			if ref != "" && !store.CatalogReferenceV2(ref) {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		}
		if utf8.RuneCountInString(f.Query) > 80 {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		scope := r.URL.Path + "|" + f.StoreID + "|" + f.BrandID + "|" + f.CategoryID + "|" + f.SellerRef + "|" + f.Query + "|" + f.State + "|" + u.User.ID
		f.Before, e = store.CatalogParseCursorV2(scope, q.Get("cursor"))
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		rows, e := s.Store.CatalogListV2(r.Context(), u.User.ID, f)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}
		views := []any{}
		for _, d := range rows {
			v, e := s.Store.CatalogViewV2(r.Context(), u.User.ID, d, management)
			if e != nil {
				catalogFailV2(w, r, e)
				return
			}
			views = append(views, v)
		}
		var cursor any
		if more {
			cursor = store.CatalogCursorV2(scope, rows[len(rows)-1].Sequence)
		}
		v2List(w, r, views, cursor, more)
	}
}
func (s *Server) catalogContentWriteV2(kind string, create, republish bool) http.HandlerFunc {
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
		key, version, content, e := store.CatalogMutationInputV2(input, true, !create, "")
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		var d store.CatalogRecordV2
		if create {
			d, e = s.Store.CatalogCreateV2(r.Context(), u.User.ID, kind, r.PathValue("storeId"), key, content)
		} else {
			d, e = s.Store.CatalogEditV2(r.Context(), u.User.ID, catalogPathIDV2(r, kind), kind, key, version, content, republish)
		}
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		status := 200
		if create {
			status = 201
		}
		s.catalogRenderV2(w, r, u.User.ID, d, kind == "product", status)
	}
}
func (s *Server) catalogFlatWriteV2(kind string, create bool) http.HandlerFunc {
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
		var d store.CatalogRecordV2
		if create {
			d, e = s.Store.CatalogCreateV2(r.Context(), u.User.ID, kind, r.PathValue("storeId"), key, content)
		} else {
			current, e2 := s.Store.CatalogGetRecordV2(r.Context(), u.User.ID, catalogPathIDV2(r, kind), kind, true)
			if e2 != nil {
				catalogFailV2(w, r, e2)
				return
			}
			if kind != "store" && current.StoreID != r.PathValue("storeId") {
				catalogFailV2(w, r, &store.CatalogErrorV2{Status: 404, Code: "NOT_FOUND", Message: "内容不属于此店铺。"})
				return
			}
			d, e = s.Store.CatalogEditV2(r.Context(), u.User.ID, current.ID, kind, key, version, content, false)
		}
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		status := 200
		if create {
			status = 201
		}
		s.catalogRenderV2(w, r, u.User.ID, d, true, status)
	}
}
func (s *Server) catalogCreateStoreV2(w http.ResponseWriter, r *http.Request) {
	u, e := s.admin(r, "platform.admin")
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	input, e := catalogReadV2(w, r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	key, _, content, e := catalogFlatMutationV2(input, true)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	d, e := s.Store.CatalogCreateV2(r.Context(), u.ID, "store", "", key, content)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	s.catalogRenderV2(w, r, u.ID, d, true, 201)
}
func (s *Server) catalogActionV2(kind, action string) http.HandlerFunc {
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
		extra := ""
		if kind == "product" && action != "publish" {
			extra = "reason"
		}
		if action == "stock-adjustments" {
			extra += " delta"
		}
		key, version, _, e := store.CatalogMutationInputV2(input, false, true, extra)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		reason := catalogHTTPStringV2(input, "reason")
		if kind == "product" && action != "publish" && (utf8.RuneCountInString(strings.TrimSpace(reason)) < 2 || utf8.RuneCountInString(reason) > 500) {
			catalogFailV2(w, r, bridge.ErrProtocol)
			return
		}
		var delta int64
		if action == "stock-adjustments" {
			var ok bool
			delta, ok = catalogHTTPIntV2(input["delta"])
			if !ok {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		}
		d, e := s.Store.CatalogActionV2(r.Context(), u.User.ID, catalogPathIDV2(r, kind), kind, key, version, action, reason, delta)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		s.catalogRenderV2(w, r, u.User.ID, d, kind == "product", 200)
	}
}
func (s *Server) catalogHomepageV2(management bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.catalogAuthV2(w, r)
		if !ok {
			return
		}
		d, e := s.Store.CatalogHomepageV2(r.Context(), u.User.ID, r.PathValue("storeId"), management)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		s.catalogRenderV2(w, r, u.User.ID, d, management, 200)
	}
}
func (s *Server) catalogHomepageWriteV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	input, e := catalogReadV2(w, r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	key, version, content, e := catalogFlatMutationV2(input, false)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	storeID := r.PathValue("storeId")
	d, e := s.Store.CatalogWriteHomepageV2(r.Context(), u.User.ID, storeID, key, version, content)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	s.catalogRenderV2(w, r, u.User.ID, d, true, 200)
}
func (s *Server) catalogMerchantV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	result, e := s.Store.CatalogMerchantV2(r.Context(), u.User)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2Success(w, r, result)
}
func (s *Server) catalogTemplatesV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	limit, e := catalogLimitV2(r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	scope := "delivery-templates:" + r.PathValue("storeId") + ":" + u.User.ID
	before, e := store.CatalogParseCursorV2(scope, r.URL.Query().Get("cursor"))
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	rows, e := s.Store.CatalogListV2(r.Context(), u.User.ID, store.CatalogFilterV2{Kind: "delivery_template", StoreID: r.PathValue("storeId"), Management: true, Limit: limit + 1, Before: before})
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	out := []any{}
	for _, d := range rows {
		out = append(out, store.CatalogTemplateViewV2(d, false))
	}
	var cursor any
	if more {
		cursor = store.CatalogCursorV2(scope, rows[len(rows)-1].Sequence)
	}
	v2List(w, r, out, cursor, more)
}
func catalogAllowedCategoryV2(s string) bool {
	switch s {
	case "MATERIALS", "EQUIPMENT", "SUPPLIES", "DECORATION", "CONSTRUCTION", "OTHER":
		return true
	}
	return false
}
func (s *Server) catalogCategoriesV2(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.catalogAuthV2(w, r); !ok {
		return
	}
	codes := []string{"MATERIALS", "EQUIPMENT", "SUPPLIES", "DECORATION", "CONSTRUCTION", "OTHER"}
	names := []string{"建材", "装备", "补给", "装饰", "建筑服务", "其他"}
	descriptions := []string{"建筑材料与基础方块", "装备、工具及其配件", "食物、药剂与日常消耗品", "装饰方块与陈设物品", "按约定地点和工期提供建筑服务", "其他允许交易的物品与服务"}
	out := []any{}
	for i, code := range codes {
		out = append(out, map[string]any{"code": code, "name": names[i], "description": descriptions[i], "sortOrder": i})
	}
	v2List(w, r, out, nil, false)
}
func (s *Server) catalogCartReadV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	cart, e := s.Store.CatalogCartV2(r.Context(), u.User.ID)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2Success(w, r, cart)
}
func (s *Server) catalogCartWriteV2(remove bool) http.HandlerFunc {
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
		extra := ""
		if !remove {
			extra = "quantity"
		}
		key, version, _, e := store.CatalogMutationInputV2(input, false, true, extra)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		var quantity int64
		if !remove {
			var ok bool
			quantity, ok = catalogHTTPIntV2(input["quantity"])
			if !ok {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
		}
		cart, e := s.Store.CatalogCartChangeV2(r.Context(), u.User.ID, r.PathValue("productId"), key, version, quantity, remove)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		v2Success(w, r, cart)
	}
}
func (s *Server) catalogQuoteV2(w http.ResponseWriter, r *http.Request) {
	u, ok := s.catalogAuthV2(w, r)
	if !ok {
		return
	}
	input, e := catalogReadV2(w, r)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	quote, e := s.Store.CatalogQuoteV2(r.Context(), u.User.ID, r.Header.Get("Idempotency-Key"), input)
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	catalogSuccessV2(w, r, 201, quote)
}
func (s *Server) catalogMembersReadV2(w http.ResponseWriter, r *http.Request) {
	if _, e := s.admin(r, "store.members.manage"); e != nil {
		catalogFailV2(w, r, e)
		return
	}
	result, e := s.Store.CatalogMembersV2(r.Context(), r.PathValue("storeId"))
	if e != nil {
		catalogFailV2(w, r, e)
		return
	}
	v2List(w, r, result, nil, false)
}
func (s *Server) catalogMembersWriteV2(revoke bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, e := s.admin(r, "store.members.manage")
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		input, e := catalogReadV2(w, r)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		extra := "playerRef permissions"
		if revoke {
			extra = ""
		}
		key, version, _, e := store.CatalogMutationInputV2(input, false, true, extra)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		playerRef := r.PathValue("playerRef")
		permissions := []string{}
		if !revoke {
			playerRef = catalogHTTPStringV2(input, "playerRef")
			array, ok := input["permissions"].([]any)
			if !ok {
				catalogFailV2(w, r, bridge.ErrProtocol)
				return
			}
			for _, v := range array {
				p, ok := v.(string)
				if !ok {
					catalogFailV2(w, r, bridge.ErrProtocol)
					return
				}
				permissions = append(permissions, p)
			}
		}
		result, e := s.Store.CatalogSetMemberV2(r.Context(), u.ID, r.PathValue("storeId"), key, playerRef, version, permissions, revoke)
		if e != nil {
			catalogFailV2(w, r, e)
			return
		}
		if revoke {
			v2Success(w, r, map[string]bool{"removed": true})
		} else {
			catalogSuccessV2(w, r, 201, result)
		}
	}
}

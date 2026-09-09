package httpapi

import (
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"net/http"
)

func (g *aiGatewayV2) quoteV207(w http.ResponseWriter, r *http.Request) {
	u, err := g.server.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	var input store.AIPurchaseInputV206
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	available := g.server.commerceAvailableHTTPV2(CommerceReserveV2) && g.server.commerceAvailableHTTPV2(CommerceSettleV2) && g.server.commerceAvailableHTTPV2(CommerceRefundV2)
	quote, err := g.server.Store.AIQuoteV207(r.Context(), u.User.ID, input, g.config.Enabled && g.config.PaidEnabled && g.configError == nil, available)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	v2Success(w, r, quote.PublicV207())
}

func (g *aiGatewayV2) purchaseV206(w http.ResponseWriter, r *http.Request) {
	// Replays remain recoverable through operations even after sales are disabled.
	u, err := g.server.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	var input store.AIPurchaseInputV206
	if err = body(w, r, &input); err != nil {
		socialFailureV2(w, r, err)
		return
	}
	available := g.server.commerceAvailableHTTPV2(CommerceReserveV2) && g.server.commerceAvailableHTTPV2(CommerceSettleV2) && g.server.commerceAvailableHTTPV2(CommerceRefundV2)
	mutation, err := g.server.Store.PrepareAIPurchaseV206(r.Context(), u.User.ID, input, g.config.Enabled && g.config.PaidEnabled && g.configError == nil, available)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	g.server.commerceMutationHTTPV2(w, r, u.User.ID, mutation, true, true)
}
func (g *aiGatewayV2) purchaseGetV206(w http.ResponseWriter, r *http.Request) {
	u, err := g.server.authenticate(r)
	if err != nil {
		failError(w, r, err)
		return
	}
	d, err := g.server.Store.CommerceRecordV2(r.Context(), r.PathValue("purchaseId"))
	if err != nil || d.OwnerID != u.User.ID || !store.IsAIOrderV206(d) {
		failure(w, r, 404, "AI_PURCHASE_NOT_FOUND", "未找到该套餐购买记录。")
		return
	}
	view, err := g.server.Store.CommerceViewV2(r.Context(), u.User.ID, d.ID, false)
	if err != nil {
		catalogFailV2(w, r, err)
		return
	}
	v2Success(w, r, view)
}

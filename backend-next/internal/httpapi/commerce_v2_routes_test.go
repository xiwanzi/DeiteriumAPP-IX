package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommerceRoutesAuthenticateBeforeUsingBodiesOrReferences(t *testing.T) {
	server := &Server{}
	mux := http.NewServeMux()
	server.registerCommerceV2(mux)
	for _, request := range []struct{ method, path string }{
		{"POST", "/api/v1/store/orders"}, {"POST", "/api/v1/market/orders"},
		{"GET", "/api/v1/orders"}, {"GET", "/api/v1/orders/unseen"}, {"GET", "/api/v1/orders/unseen/snapshot"},
		{"POST", "/api/v1/orders/unseen/confirm"}, {"POST", "/api/v1/orders/unseen/ship"},
		{"POST", "/api/v1/orders/unseen/refunds"}, {"POST", "/api/v1/orders/unseen/refunds/refund/resolve"},
		{"GET", "/api/v1/operations/by-client-request?clientRequestId=x&kind=REFUND"}, {"GET", "/api/v1/operations/unseen"},
		{"GET", "/api/v1/commissions"}, {"GET", "/api/v1/commissions/me"}, {"POST", "/api/v1/commissions"},
		{"POST", "/api/v1/commissions/unseen/accept"}, {"POST", "/api/v1/commissions/unseen/complete"},
		{"GET", "/api/v1/merchant/orders?storeId=unseen"}, {"POST", "/api/v1/merchant/orders/unseen/delivery-retry"},
		{"POST", "/api/v1/orders/unseen/interventions"}, {"GET", "/api/v1/interventions/unseen"},
		{"POST", "/api/v1/interventions/unseen/evidence"}, {"GET", "/api/v1/admin/interventions"},
		{"POST", "/api/v1/admin/interventions/unseen/resolve"},
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(request.method, request.path, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s returned %d", request.method, request.path, w.Code)
		}
	}
}

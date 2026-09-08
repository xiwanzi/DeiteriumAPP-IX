//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommerceHTTPRejectsOfflinePaymentAndUntrustedFinancialFields(t *testing.T) {
	f := newCatalogFixture(t)
	mux := http.NewServeMux()
	f.server.registerCommerceV2(mux)
	for _, test := range []struct {
		path, body string
		want       int
	}{
		{"/api/v1/store/orders", `{"clientRequestId":"offline-shop","quoteId":"real-quote-ref","expectedQuoteVersion":1}`, 503},
		{"/api/v1/market/orders", `{"clientRequestId":"offline-market","quoteId":"real-quote-ref","expectedQuoteVersion":1}`, 503},
		{"/api/v1/market/orders", `{"clientRequestId":"bad-payee","quoteId":"real-quote-ref","expectedQuoteVersion":1,"payerUuid":"client-chosen","amount":"0.01"}`, 400},
		{"/api/v1/market/orders", `{"clientRequestId":"duplicate","quoteId":"one","quoteId":"two","expectedQuoteVersion":1}`, 400},
		{"/api/v1/orders/own-order/refunds/refund-id/resolve", `{"clientRequestId":"no-version","decision":"APPROVE","reason":""}`, 400},
		{"/api/v1/orders/own-order/refunds/refund-id/resolve", `{"clientRequestId":"empty-rejection","expectedVersion":1,"decision":"REJECT","reason":"  "}`, 400},
		{"/api/v1/commissions", `{"clientRequestId":"offline-commission","content":{"title":"测试委托","description":"完成指定区域照明布置","location":"主世界出生点","urgency":"NORMAL","reward":"1.00","workHours":24,"coverAssetId":"asset_ready"}}`, 503},
	} {
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("Authorization", "Bearer "+f.buyerToken)
		request.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != test.want {
			t.Errorf("%s returned %d: %s", test.path, w.Code, w.Body.String())
		}
	}
	for _, table := range []string{"commerce_requests_v2", "commerce_resources_v2", "commerce_operations_v2", "core_operations"} {
		var count int
		if err := f.s.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("invalid or offline requests persisted %d rows in %s", count, table)
		}
	}
}

func TestCommerceHTTPValidatesListScopeAndHidesAdministratorRoutes(t *testing.T) {
	f := newCatalogFixture(t)
	mux := http.NewServeMux()
	f.server.registerCommerceV2(mux)
	for _, path := range []string{
		"/api/v1/commissions?status=ACTIVE", "/api/v1/commissions?urgency=SECRET", "/api/v1/commissions?sort=INVALID",
		"/api/v1/orders?role=anyone", "/api/v1/orders?hasRefund=maybe", "/api/v1/orders?limit=101",
		"/api/v1/merchant/orders", "/api/v1/operations/by-client-request?clientRequestId=one&kind=SERVER_COMMAND",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+f.buyerToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != 400 {
			t.Errorf("%s returned %d: %s", path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/admin/interventions", "/api/v1/admin/interventions/unseen"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+f.buyerToken)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != 403 {
			t.Errorf("unauthorized administrator route %s returned %d: %s", path, w.Code, w.Body.String())
		}
	}
}

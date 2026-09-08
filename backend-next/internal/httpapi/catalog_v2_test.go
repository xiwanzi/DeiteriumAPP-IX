package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogJSONRejectsDuplicateKeysDepthAndTrailingData(t *testing.T) {
	for _, body := range []string{`{"clientRequestId":"one","clientRequestId":"two"}`, `{"content":{"price":"1","price":"2"}}`, `{} {}`, `[]`, strings.Repeat(`{"nested":`, 35) + `{}` + strings.Repeat(`}`, 35)} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if _, e := catalogReadV2(httptest.NewRecorder(), r); e == nil {
			t.Fatalf("accepted ambiguous input %q", body)
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"content":{"stock":0,"description":"真实描述"}}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	if _, e := catalogReadV2(httptest.NewRecorder(), r); e != nil {
		t.Fatal(e)
	}
}
func TestCatalogRoutesRequireAuthenticationBeforeBusinessWork(t *testing.T) {
	s := &Server{}
	mux := http.NewServeMux()
	s.registerCatalogV2(mux)
	s.registerCommerceV2(mux)
	for _, route := range []struct{ method, path string }{{"GET", "/api/v1/store/products"}, {"GET", "/api/v1/market/listings"}, {"POST", "/api/v1/market/listings"}, {"POST", "/api/v1/store/orders"}, {"POST", "/api/v1/admin/stores"}, {"GET", "/api/v1/merchant/me"}} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))
		if w.Code != 401 {
			t.Fatalf("%s %s returned %d", route.method, route.path, w.Code)
		}
	}
}

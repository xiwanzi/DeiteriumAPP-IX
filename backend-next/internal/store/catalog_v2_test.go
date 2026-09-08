package store

import (
	"encoding/json"
	"strings"
	"testing"
)

func validListingCatalogTest() CatalogObjectV2 {
	return CatalogObjectV2{"title": "石材", "subtitle": "建筑材料", "description": "一组石材，按约定交付。", "categoryCode": "MATERIALS", "price": "12.30", "stock": 1, "photoAssetIds": []any{"asset_one"}, "contactQq": "123456789", "deliveryMethods": []any{"PICKUP"}, "pickupLocation": "主城仓库", "workHours": 24}
}
func copyCatalogTest(o CatalogObjectV2) CatalogObjectV2 {
	b, _ := json.Marshal(o)
	var out CatalogObjectV2
	_ = json.Unmarshal(b, &out)
	return out
}
func TestCatalogValidationRejectsClientAuthorityAndInvalidDelivery(t *testing.T) {
	good := validListingCatalogTest()
	if e := ValidateCatalogContentV2("listing", good); e != nil {
		t.Fatal(e)
	}
	mutations := []func(CatalogObjectV2){func(o CatalogObjectV2) { o["sellerPlayerRef"] = "other" }, func(o CatalogObjectV2) { delete(o, "description") }, func(o CatalogObjectV2) { o["stock"] = 0 }, func(o CatalogObjectV2) { o["stock"] = 1.5 }, func(o CatalogObjectV2) { o["photoAssetIds"] = []any{"a", "a"} }, func(o CatalogObjectV2) { o["price"] = "-1" }, func(o CatalogObjectV2) { o["price"] = "1.001" }, func(o CatalogObjectV2) { o["price"] = "NaN" }, func(o CatalogObjectV2) { o["deliveryMethods"] = []any{"WORKSITE"} }, func(o CatalogObjectV2) { o["categoryCode"] = "CONSTRUCTION" }, func(o CatalogObjectV2) { o["pickupLocation"] = "" }, func(o CatalogObjectV2) { o["description"] = strings.Repeat("长", 10001) }}
	for i, change := range mutations {
		v := copyCatalogTest(good)
		change(v)
		if ValidateCatalogContentV2("listing", v) == nil {
			t.Fatalf("accepted invalid mutation %d", i)
		}
	}
	build := copyCatalogTest(good)
	build["categoryCode"] = "CONSTRUCTION"
	build["deliveryMethods"] = []any{"WORKSITE"}
	if e := ValidateCatalogContentV2("listing", build); e != nil {
		t.Fatal(e)
	}
}
func TestCatalogAmountsAreExactAndQuotesRejectForgedTotals(t *testing.T) {
	for _, s := range []string{"0", "0.00", "-2", "1e3", "01.20", "1.234", strings.Repeat("9", 20)} {
		if _, ok := catalogMoney(s); ok {
			t.Fatalf("accepted invalid amount %q", s)
		}
	}
	for input, want := range map[string]string{"0.01": "0.01", "19.9": "19.90", "90071992547409.91": "90071992547409.91"} {
		n, ok := catalogMoney(input)
		if !ok || catalogMoneyString(n) != want {
			t.Fatalf("lost decimal accuracy for %s", input)
		}
	}
	quote := CatalogObjectV2{"channel": "OFFICIAL_STORE", "items": []any{map[string]any{"productId": "product_one", "quantity": 1, "expectedProductVersion": 1}}, "delivery": map[string]any{"method": "MAILBOX", "location": "", "projectName": ""}}
	if e := CatalogQuoteInputV2(quote); e != nil {
		t.Fatal(e)
	}
	quote["totalAmount"] = "0.01"
	if CatalogQuoteInputV2(quote) == nil {
		t.Fatal("accepted client price")
	}
	delete(quote, "totalAmount")
	quote["items"] = append(quote["items"].([]any), quote["items"].([]any)[0])
	if CatalogQuoteInputV2(quote) == nil {
		t.Fatal("duplicate product could bypass per-order limit")
	}
}
func TestCatalogCursorIsBoundToViewerAndFilters(t *testing.T) {
	v := CatalogCursorV2("viewer:alice/category:MATERIALS", 12)
	n, e := CatalogParseCursorV2("viewer:alice/category:MATERIALS", v)
	if e != nil || n != 12 {
		t.Fatal(n, e)
	}
	for _, scope := range []string{"viewer:bob/category:MATERIALS", "viewer:alice/category:OTHER"} {
		if _, e := CatalogParseCursorV2(scope, v); e == nil {
			t.Fatal("accepted cursor from another result set")
		}
	}
}

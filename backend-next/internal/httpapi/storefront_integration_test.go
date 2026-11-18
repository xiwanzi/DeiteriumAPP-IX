//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func TestStorefrontRecommendations(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	base, content := catalogProductFixture(t, f)
	content["inventoryPolicy"] = "UNLIMITED"
	content["stock"] = 0
	products := []store.CatalogRecordV2{}
	for i := 0; i < 10; i++ {
		products = append(products, quoteReviewProductV2(t, f, content, base.StoreID, fmt.Sprintf("featured-%d", i)))
	}
	for index, state := range []string{"HELD", "SETTLED", "PAID", "REFUNDED", "UNKNOWN", "HELD"} {
		quantity := []int{3, 2, 1, 5, 5, 5}[index]
		product := products[index]
		key := fmt.Sprintf("order-%d", index)
		quote := quoteReviewAcceptedV2(t, f, key+"-quote", []store.CatalogRecordV2{product}, []int{quantity})
		order, err := f.s.PrepareOrderV2(ctx, f.buyer.ID, key, fmt.Sprint(quote["quoteId"]), 1, "OFFICIAL_STORE", true, map[string]store.CatalogNodePolicyV2{"amiya": {InventoryDomain: "survival", ClaimEnabled: true}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.s.DB.Exec("UPDATE commerce_resources_v2 SET funds_state=?, state='COMPLETED',created_at=UTC_TIMESTAMP()-INTERVAL ? DAY WHERE resource_id=?", state, index/5*40, order.ResourceID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.s.DB.Exec("UPDATE catalog_records_v2 SET available_stock=0 WHERE resource_id=?", base.ID); err != nil {
		t.Fatal(err)
	}
	available, err := f.s.CatalogListV2(ctx, f.buyer.ID, store.CatalogFilterV2{Kind: "product", InStock: true, Limit: 100})
	if err != nil || len(available) != len(products) {
		t.Fatalf("available products: %d, error: %v", len(available), err)
	}
	for _, record := range available {
		if record.ID == base.ID {
			t.Fatal("sold-out product in the available catalogue")
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		rows, err := f.s.CatalogRecommendations(ctx)
		if err != nil || len(rows) != 6 {
			t.Fatalf("recommendations: %v %v", len(rows), err)
		}
		seen := map[string]bool{}
		for pos, row := range rows {
			if seen[row.ID] || row.ID == base.ID {
				t.Fatal("duplicate or sold out recommendation", row.ID)
			}
			if pos%2 == 0 && row.ID != products[pos/2].ID {
				t.Fatalf("incorrect popular product at %d", pos)
			}
			seen[row.ID] = true
		}
	}
}

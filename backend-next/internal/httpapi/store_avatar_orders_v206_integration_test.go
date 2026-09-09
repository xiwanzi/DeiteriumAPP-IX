//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"testing"
)

func TestStoreAvatarIsOptionalForExistingOrderReadsV206(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	product, _ := catalogProductFixture(t, f)
	profile, err := f.s.CatalogGetRecordV2(ctx, f.admin.ID, product.StoreID, "store", true)
	if err != nil {
		t.Fatal(err)
	}
	content := catalogObjectClone(profile.Body)
	content["logoAssetId"] = nil
	profile, err = f.s.CatalogEditV2(ctx, f.admin.ID, profile.ID, "store", "clear-logo", profile.Version, content, false)
	if err != nil {
		t.Fatal(err)
	}
	order := commerceOfficialTest(t, f, product, "order-without-store-logo")
	check := func(expected string) {
		t.Helper()
		view, err := f.s.CommerceViewV2(ctx, f.buyer.ID, order.ResourceID, false)
		if err != nil {
			t.Fatalf("avatar blocked order read: %v", err)
		}
		raw, _ := json.Marshal(view)
		var result struct {
			Seller struct {
				Avatar *struct {
					AssetID string `json:"assetId"`
				} `json:"avatar"`
			} `json:"seller"`
		}
		if err = json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		if expected == "" && result.Seller.Avatar != nil {
			t.Fatal("unavailable avatar leaked into order")
		}
		if expected != "" && (result.Seller.Avatar == nil || result.Seller.Avatar.AssetID != expected) {
			t.Fatal("configured avatar was not projected")
		}
	}
	check("")
	asset := catalogTestAsset(t, f.s, f.admin.ID, "STORE_MEDIA")
	content["logoAssetId"] = asset
	_, err = f.s.CatalogEditV2(ctx, f.admin.ID, profile.ID, "store", "set-logo", profile.Version, content, false)
	if err != nil {
		t.Fatal(err)
	}
	check(asset)
	// A concurrent logo replacement can detach an asset after the profile read.
	if _, err = f.s.DB.ExecContext(ctx, "DELETE FROM asset_bindings_v2 WHERE business_type='STORE_PROFILE' AND business_ref=? AND asset_id=?", profile.ID, asset); err != nil {
		t.Fatal(err)
	}
	check("")
}

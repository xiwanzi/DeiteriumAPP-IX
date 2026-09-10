package store

import (
	"context"
	"math/big"
	"testing"
	"time"
)

func TestPromotionFreeDeliveryEvidenceNeverReleasesRewardsOnAdapterFailureV209(t *testing.T) {
	for _, status := range []string{"CREATED", "CLAIMING", "CLAIMED", "UNKNOWN"} {
		d := CommerceRecordV2{Channel: "OFFICIAL_STORE", Kind: "ORDER", Amount: "0.00", State: "CLAIMED", Body: CatalogObjectV2{"mailId": "real_mail", "mailboxState": status}}
		e := (&Store{}).commerceFailOperationV2(context.Background(), nil, &d, CommerceOperationV2{Action: "reserve", StepIndex: 0}, time.Now())
		if e != nil || d.FundsState != "HELD" || d.State == "CANCELLED" {
			t.Fatal(status, d.State, d.FundsState, e)
		}
	}
}

func promotionTestCouponV209(kind string, stack bool) CatalogObjectV2 {
	return CatalogObjectV2{"name": "测试券", "type": kind, "benefit": "PERCENT", "amountOff": "0.00", "discountRate": int64(8000), "minimumSpend": "0.00", "maxDiscount": "0.00", "stackWithProductDiscount": stack, "audience": "ALL", "playerRefs": []any{}, "storeIds": []any{}, "productIds": []any{}, "startsAt": "2026-01-01T00:00:00Z", "endsAt": "2027-01-01T00:00:00Z", "active": true}
}

func TestPromotionBestSingleUnitAndNonStackComparisonV209(t *testing.T) {
	lines := []promotionLineV209{{"a", "shop", 5, big.NewInt(10000), big.NewInt(5000)}, {"b", "shop", 2, big.NewInt(9000), big.NewInt(9000)}}
	for _, test := range []struct {
		kind  string
		stack bool
		want  int64
		item  string
	}{{"ITEM", false, 1800, "b"}, {"ITEM", true, 1800, "b"}, {"ORDER", false, 0, ""}, {"ORDER", true, 8600, ""}} {
		saved, item := promotionSavingV209(promotionTestCouponV209(test.kind, test.stack), lines)
		if saved.Int64() != test.want || item != test.item {
			t.Fatalf("%+v: %s/%s", test, saved, item)
		}
	}
	// Replacing a product's existing discount must never increase the total.
	weak := promotionTestCouponV209("ITEM", false)
	weak["productIds"] = []any{"a"}
	if saved, _ := promotionSavingV209(weak, lines); saved.Sign() != 0 {
		t.Fatal("weaker coupon replaced product sale")
	}
}

func TestPromotionThresholdScopeCapRoundingAndFreeV209(t *testing.T) {
	lines := []promotionLineV209{{"a", "one", 2, big.NewInt(10000), big.NewInt(9000)}, {"b", "two", 1, big.NewInt(50000), big.NewInt(50000)}}
	c := promotionTestCouponV209("ORDER", true)
	c["storeIds"] = []any{"one"}
	c["maxDiscount"] = "20.00"
	if saved, _ := promotionSavingV209(c, lines); saved.Int64() != 2000 {
		t.Fatal(saved)
	}
	c["minimumSpend"] = "200.00"
	if saved, _ := promotionSavingV209(c, lines); saved.Sign() != 0 {
		t.Fatal("out-of-scope items counted toward threshold")
	}
	c["stackWithProductDiscount"] = false
	if saved, _ := promotionSavingV209(c, lines); saved.Sign() != 0 {
		t.Fatal("equal coupon consumed")
	}
	c["minimumSpend"] = "0.00"
	c["benefit"] = "FIXED"
	c["amountOff"] = "300.00"
	if saved, _ := promotionSavingV209(c, lines); saved.Int64() != 18000 {
		t.Fatal("fixed discount must stop at scoped subtotal", saved)
	}
	if got := promotionRateV209(big.NewInt(5), 5000).Int64(); got != 3 {
		t.Fatal("half cent not rounded once", got)
	}
	c["productIds"] = []any{"b"}
	if saved, _ := promotionSavingV209(c, lines); saved.Sign() != 0 {
		t.Fatal("store/product scopes were unioned")
	}
}

func TestPromotionSelectsGreatestSavingAndDoesNotExposeAudienceV209(t *testing.T) {
	products := map[string]CatalogRecordV2{"p": {ID: "p", StoreID: "shop", Published: CatalogObjectV2{"price": "100.00", "discountRate": 8000}}}
	c1, c2 := promotionTestCouponV209("ITEM", false), promotionTestCouponV209("ORDER", true)
	c1["discountRate"] = int64(5000)
	c1["audience"] = "PLAYERS"
	c1["playerRefs"] = []any{"private-other-player"}
	c2["discountRate"] = int64(9000)
	quote, e := promotionTotalsV209(products, map[string]int64{"p": 2}, []CatalogRecordV2{{ID: "item", Version: 1, Body: c1}, {ID: "order", Version: 1, Body: c2}})
	if e != nil || quote["totalAmount"] != "130.00" || quote["productDiscount"] != "40.00" || quote["couponDiscount"] != "30.00" {
		t.Fatal(quote, e)
	}
	applied, _ := catalogObject(quote["coupon"])
	terms, _ := catalogObject(applied["terms"])
	if applied["couponId"] != "item" || terms["playerRefs"] != nil || terms["audience"] != nil {
		t.Fatal("coupon selection or audience leak", applied)
	}
}

func TestPromotionLimitCalendarBoundariesV209(t *testing.T) {
	limits := CatalogObjectV2{"dailyTime": "00:00", "weeklyDay": int64(1), "weeklyTime": "04:30", "monthlyDay": int64(31), "monthlyTime": "01:00"}
	for _, test := range []struct{ period, now, want string }{
		{"daily", "2026-09-09T15:59:59Z", "2026-09-08T16:00:00Z"},
		{"daily", "2026-09-09T16:00:00Z", "2026-09-09T16:00:00Z"},
		{"weekly", "2026-09-06T20:29:59Z", "2026-08-30T20:30:00Z"},
		{"weekly", "2026-09-06T20:30:00Z", "2026-09-06T20:30:00Z"},
		{"monthly", "2026-02-27T16:59:59Z", "2026-01-30T17:00:00Z"},
		{"monthly", "2026-02-27T17:00:00Z", "2026-02-27T17:00:00Z"},
		{"monthly", "2028-02-28T17:00:00Z", "2028-02-28T17:00:00Z"},
		{"monthly", "2026-01-01T00:00:00Z", "2025-12-30T17:00:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, test.now)
		if got := promotionPeriodStartV209(test.period, limits, now).Format(time.RFC3339); got != test.want {
			t.Errorf("%+v got %s", test, got)
		}
	}
}

func TestPromotionValidationRejectsInvalidAmountsTargetsAndDatesV209(t *testing.T) {
	for _, patch := range []CatalogObjectV2{{"discountRate": 0}, {"minimumSpend": "-1"}, {"amountOff": "1.001"}, {"endsAt": "2025-01-01T00:00:00Z"}, {"audience": "PLAYERS"}, {"playerRefs": []any{"leaked"}}, {"unexpected": true}} {
		c := promotionTestCouponV209("ORDER", false)
		for k, v := range patch {
			c[k] = v
		}
		if validateCouponV209(c) == nil {
			t.Fatal("accepted invalid coupon", patch)
		}
	}
	if validateProductPromotionV209(CatalogObjectV2{"purchaseLimits": CatalogObjectV2{"weeklyDay": 8}}) == nil {
		t.Fatal("invalid weekly reset")
	}
	if validateProductPromotionV209(CatalogObjectV2{"purchaseLimits": CatalogObjectV2{"dailyTime": "25:00"}}) == nil {
		t.Fatal("invalid reset time")
	}
}

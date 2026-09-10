package store

import (
	"math/big"
	"sort"
	"time"
)

// Rates are the payable share in basis points. Round once per unit to cents.
func promotionRateV209(amount *big.Int, rate int64) *big.Int {
	n := new(big.Int).Mul(amount, big.NewInt(rate))
	return n.Quo(n.Add(n, big.NewInt(5000)), big.NewInt(10000))
}

func promotionPriceV209(content CatalogObjectV2) (*big.Int, error) {
	price, ok := catalogMoney(content["price"])
	if !ok {
		return nil, catalogInvalid()
	}
	rate := catalogNumber(content, "discountRate")
	if rate == 0 {
		rate = 10000
	}
	return promotionRateV209(price, rate), nil
}

func validateProductPromotionV209(o CatalogObjectV2) error {
	if rate, has := o["discountRate"]; has && !catalogRange(rate, 1, 10000) {
		return catalogInvalid()
	}
	if credits, has := o["deliveryCredits"]; has && !catalogRange(credits, 0, 1000000000) {
		return catalogInvalid()
	}
	if value, has := o["purchaseLimits"]; has {
		limits, ok := catalogObject(value)
		if !ok || !catalogFields(limits, "", "lifetime daily weekly monthly dailyTime weeklyDay weeklyTime monthlyDay monthlyTime") {
			return catalogInvalid()
		}
		for _, k := range []string{"lifetime", "daily", "weekly", "monthly"} {
			if n, exists := limits[k]; exists && !catalogRange(n, 0, 999999) {
				return catalogInvalid()
			}
		}
		for _, k := range []string{"dailyTime", "weeklyTime", "monthlyTime"} {
			if value, exists := limits[k]; exists {
				s, ok := value.(string)
				if !ok {
					return catalogInvalid()
				}
				if _, err := time.Parse("15:04", s); err != nil || len(s) != 5 {
					return catalogInvalid()
				}
			}
		}
		if n, exists := limits["weeklyDay"]; exists && !catalogRange(n, 1, 7) {
			return catalogInvalid()
		}
		if n, exists := limits["monthlyDay"]; exists && !catalogRange(n, 1, 31) {
			return catalogInvalid()
		}
	}
	return nil
}

type promotionLineV209 struct {
	ID, StoreID     string
	Quantity        int64
	Original, Price *big.Int
}

func promotionMatchesV209(coupon CatalogObjectV2, line promotionLineV209) bool {
	for key, id := range map[string]string{"storeIds": line.StoreID, "productIds": line.ID} {
		ids := catalogIDs(coupon, key)
		if len(ids) > 0 {
			found := false
			for _, candidate := range ids {
				if candidate == id {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func promotionReductionV209(coupon CatalogObjectV2, basis *big.Int) *big.Int {
	var saved *big.Int
	if coupon["benefit"] == "FIXED" {
		saved, _ = commerceAmountV2(catalogString(coupon, "amountOff"))
	} else {
		saved = new(big.Int).Sub(basis, promotionRateV209(basis, catalogNumber(coupon, "discountRate")))
		if cap, e := commerceAmountV2(catalogString(coupon, "maxDiscount")); e == nil && cap.Sign() > 0 && saved.Cmp(cap) > 0 {
			saved = cap
		}
	}
	if saved == nil {
		return new(big.Int)
	}
	if saved.Cmp(basis) > 0 {
		return new(big.Int).Set(basis)
	}
	return saved
}

// Returns the extra saving beyond existing product discounts, and the single
// selected unit for ITEM coupons. Ineligible products retain their own discount.
func promotionSavingV209(coupon CatalogObjectV2, lines []promotionLineV209) (*big.Int, string) {
	stack, _ := coupon["stackWithProductDiscount"].(bool)
	threshold, _ := commerceAmountV2(catalogString(coupon, "minimumSpend"))
	if threshold == nil {
		threshold = new(big.Int)
	}
	best, itemID := new(big.Int), ""
	original, discounted := new(big.Int), new(big.Int)
	for _, line := range lines {
		if !promotionMatchesV209(coupon, line) {
			continue
		}
		if coupon["type"] == "ITEM" {
			basis := line.Original
			if stack {
				basis = line.Price
			}
			if basis.Cmp(threshold) < 0 {
				continue
			}
			payable := new(big.Int).Sub(basis, promotionReductionV209(coupon, basis))
			extra := new(big.Int).Sub(line.Price, payable)
			if extra.Cmp(best) > 0 || (extra.Sign() > 0 && extra.Cmp(best) == 0 && line.ID < itemID) {
				best, itemID = extra, line.ID
			}
		} else {
			original.Add(original, new(big.Int).Mul(line.Original, big.NewInt(line.Quantity)))
			discounted.Add(discounted, new(big.Int).Mul(line.Price, big.NewInt(line.Quantity)))
		}
	}
	if coupon["type"] == "ITEM" {
		return best, itemID
	}
	basis := original
	if stack {
		basis = discounted
	}
	if basis.Cmp(threshold) < 0 {
		return new(big.Int), ""
	}
	payable := new(big.Int).Sub(basis, promotionReductionV209(coupon, basis))
	extra := new(big.Int).Sub(discounted, payable)
	if extra.Sign() < 0 {
		extra.SetInt64(0)
	}
	return extra, ""
}

func promotionTotalsV209(products map[string]CatalogRecordV2, quantities map[string]int64, coupons []CatalogRecordV2) (CatalogObjectV2, error) {
	lines := []promotionLineV209{}
	original, discounted := new(big.Int), new(big.Int)
	for _, id := range commerceSortedKeysV2(quantities) {
		p := products[id]
		price, e := promotionPriceV209(p.Published)
		if e != nil {
			return nil, e
		}
		base, _ := catalogMoney(p.Published["price"])
		q := quantities[id]
		lines = append(lines, promotionLineV209{id, p.StoreID, q, base, price})
		original.Add(original, new(big.Int).Mul(base, big.NewInt(q)))
		discounted.Add(discounted, new(big.Int).Mul(price, big.NewInt(q)))
	}
	// Stable tie break preserves a later-expiring coupon for a future purchase.
	if e := commerceWithinExecutionLimitV2(catalogMoneyString(original)); e != nil {
		return nil, e
	}
	sort.Slice(coupons, func(i, j int) bool {
		a, b := catalogString(coupons[i].Body, "endsAt"), catalogString(coupons[j].Body, "endsAt")
		if a != b {
			return a < b
		}
		return coupons[i].ID < coupons[j].ID
	})
	best := new(big.Int)
	var applied any
	for _, c := range coupons {
		saved, item := promotionSavingV209(c.Body, lines)
		if saved.Cmp(best) <= 0 {
			continue
		}
		best = saved
		terms := promotionCopyV209(c.Body)
		delete(terms, "playerRefs")
		delete(terms, "audience")
		productTitle := ""
		if p, ok := products[item]; ok {
			productTitle = catalogString(p.Published, "title")
		}
		applied = CatalogObjectV2{"couponId": c.ID, "version": c.Version, "name": c.Body["name"], "type": c.Body["type"], "productId": item, "productTitle": productTitle, "discountAmount": catalogMoneyString(saved), "stackWithProductDiscount": c.Body["stackWithProductDiscount"], "terms": terms}
	}
	return CatalogObjectV2{"originalTotal": catalogMoneyString(original), "productDiscount": catalogMoneyString(new(big.Int).Sub(original, discounted)), "couponDiscount": catalogMoneyString(best), "discountTotal": catalogMoneyString(new(big.Int).Sub(original, new(big.Int).Sub(discounted, best))), "totalAmount": catalogMoneyString(new(big.Int).Sub(discounted, best)), "coupon": applied}, nil
}

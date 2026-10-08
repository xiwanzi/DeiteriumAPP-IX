package store

import (
	"context"
	"database/sql"
	"math/big"
	"sort"
	"time"
)

func checkoutStoresV221(products map[string]CatalogRecordV2) []string {
	seen := map[string]bool{}
	for _, p := range products {
		seen[p.StoreID] = true
	}
	stores := make([]string, 0, len(seen))
	for id := range seen {
		stores = append(stores, id)
	}
	sort.Strings(stores)
	return stores
}

// Maximum-weight assignment: one coupon per order and one order per coupon.
// Extra zero-weight columns represent not using a coupon. All values are cents.
// Sorted stores/coupons and strict comparisons make equal-cost choices stable.
func checkoutCouponAssignmentV221(savings [][]int64) []int {
	n := len(savings)
	if n == 0 {
		return nil
	}
	coupons := len(savings[0])
	m := coupons + n
	u, v := make([]int64, n+1), make([]int64, m+1)
	p, way := make([]int, m+1), make([]int, m+1)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv, used := make([]int64, m+1), make([]bool, m+1)
		for j := 1; j <= m; j++ {
			minv[j] = 1 << 62
		}
		for {
			used[j0] = true
			i0, j1, delta := p[j0], 0, int64(1<<62)
			for j := 1; j <= m; j++ {
				if used[j] {
					continue
				}
				cost := int64(0)
				if j <= coupons {
					cost = -savings[i0-1][j-1]
				}
				cur := cost - u[i0] - v[j]
				if cur < minv[j] {
					minv[j] = cur
					way[j] = j0
				}
				if minv[j] < delta {
					delta = minv[j]
					j1 = j
				}
			}
			for j := 0; j <= m; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}
	assignment := make([]int, n)
	for i := range assignment {
		assignment[i] = -1
	}
	for j := 1; j <= coupons; j++ {
		if p[j] > 0 && savings[p[j]-1][j-1] > 0 {
			assignment[p[j]-1] = j - 1
		}
	}
	return assignment
}

func (s *Store) catalogCheckoutQuoteV221(ctx context.Context, tx *sql.Tx, owner User, input CatalogObjectV2, products map[string]CatalogRecordV2, quantities map[string]int64, lines []any, delivery CatalogObjectV2, now time.Time) (CatalogObjectV2, error) {
	stores := checkoutStoresV221(products)
	coupons, e := promotionCouponsTxV209(ctx, tx, owner, now)
	if e != nil {
		return nil, e
	}
	sort.Slice(coupons, func(i, j int) bool {
		a, b := catalogString(coupons[i].Body, "endsAt"), catalogString(coupons[j].Body, "endsAt")
		if a != b {
			return a < b
		}
		return coupons[i].ID < coupons[j].ID
	})
	byStore := map[string]map[string]CatalogRecordV2{}
	quantitiesByStore := map[string]map[string]int64{}
	savings := make([][]int64, len(stores))
	for i, store := range stores {
		ps, qs := map[string]CatalogRecordV2{}, map[string]int64{}
		pricingLines := []promotionLineV209{}
		for _, id := range commerceSortedKeysV2(quantities) {
			p := products[id]
			if p.StoreID != store {
				continue
			}
			ps[id], qs[id] = p, quantities[id]
			price, e := promotionPriceV209(p.Published)
			if e != nil {
				return nil, e
			}
			original, ok := catalogMoney(p.Published["price"])
			if !ok {
				return nil, catalogInvalid()
			}
			pricingLines = append(pricingLines, promotionLineV209{id, p.StoreID, quantities[id], original, price})
		}
		if _, e := commerceDeliveryContentsV2(ctx, tx, ps, qs, nil); e != nil {
			return nil, e
		}
		byStore[store], quantitiesByStore[store] = ps, qs
		savings[i] = make([]int64, len(coupons))
		for j, coupon := range coupons {
			saved, _ := promotionSavingV209(coupon.Body, pricingLines)
			if !saved.IsInt64() {
				return nil, catalogInvalid()
			}
			savings[i][j] = saved.Int64()
		}
	}
	assignment := checkoutCouponAssignmentV221(savings)
	groups := []any{}
	totals := map[string]*big.Int{}
	for _, k := range []string{"totalAmount", "originalTotal", "productDiscount", "couponDiscount", "discountTotal"} {
		totals[k] = new(big.Int)
	}
	for i, store := range stores {
		selected := []CatalogRecordV2{}
		if assignment[i] >= 0 {
			selected = append(selected, coupons[assignment[i]])
		}
		pricing, e := promotionTotalsV209(byStore[store], quantitiesByStore[store], selected)
		if e != nil {
			return nil, e
		}
		shop, e := catalogRecordTx(ctx, tx, store, "store", false)
		if e != nil {
			return nil, e
		}
		groupLines := []any{}
		for _, v := range lines {
			line, _ := catalogObject(v)
			if products[catalogString(line, "productId")].StoreID == store {
				groupLines = append(groupLines, v)
			}
		}
		group := CatalogObjectV2{"storeId": store, "storeName": shop.Body["name"], "channel": "OFFICIAL_STORE", "items": groupLines, "delivery": delivery, "currency": "CREDIT", "version": 1}
		for k, v := range pricing {
			group[k] = v
		}
		if input["source"] != "DIRECT" {
			cart, e := promotionCartSnapshotV209(ctx, tx, owner.ID, quantitiesByStore[store])
			if e != nil {
				return nil, e
			}
			group["cartItems"] = cart
		}
		for k, total := range totals {
			n, e := commerceAmountV2(catalogString(pricing, k))
			if e != nil {
				return nil, e
			}
			total.Add(total, n)
		}
		groups = append(groups, group)
	}
	if e := commerceWithinExecutionLimitV2(catalogMoneyString(totals["originalTotal"])); e != nil {
		return nil, e
	}
	quote := CatalogObjectV2{"quoteId": ID("quote_"), "channel": "OFFICIAL_STORE", "items": lines, "currency": "CREDIT", "expiresAt": now.Add(5 * time.Minute), "delivery": delivery, "version": 1, "coupon": nil, "storeName": "多店合并付款", "storeOrders": groups, "warnings": []string{"按店铺分别生成订单和游戏邮件；每店独立计算优惠，付款时重新校验库存及交付条件。"}}
	for k, total := range totals {
		quote[k] = catalogMoneyString(total)
	}
	if _, e := tx.ExecContext(ctx, "INSERT INTO catalog_quotes_v2 VALUES(?,?,?,?,?,?)", quote["quoteId"], owner.ID, "OFFICIAL_STORE", catalogJSON(quote), now, now.Add(5*time.Minute)); e != nil {
		return nil, e
	}
	return quote, nil
}

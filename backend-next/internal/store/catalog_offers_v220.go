package store

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"
)

// One eligibility read for the whole page. Pricing is the same authoritative
// function used for quotes; this preview neither reserves nor consumes a coupon.
func (s *Store) CatalogOfferPreviewsV220(ctx context.Context, viewer string, products []CatalogRecordV2) (map[string]CatalogObjectV2, error) {
	out := map[string]CatalogObjectV2{}
	if len(products) == 0 {
		return out, nil
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var owner User
	var verified bool
	e = tx.QueryRowContext(ctx, "SELECT id,player_ref,COALESCE(server_uuid,''),status,game_verified FROM identities WHERE id=?", viewer).Scan(&owner.ID, &owner.PlayerRef, &owner.ServerUUID, &owner.Status, &verified)
	if e != nil {
		return nil, e
	}
	if owner.Status != "active" {
		return nil, ErrUnauthorized
	}
	if !verified || owner.ServerUUID == "" {
		return out, nil
	}
	now := time.Now().UTC()
	coupons, e := promotionCouponsTxV209(ctx, tx, owner, now)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	if len(coupons) == 0 {
		return out, nil
	}
	for _, product := range products {
		if product.Kind != "product" || product.State != "ACTIVE" || product.Published == nil {
			continue
		}
		totals, e := promotionTotalsV209(map[string]CatalogRecordV2{product.ID: product}, map[string]int64{product.ID: 1}, coupons)
		if e != nil {
			// A product that cannot form a purchasable quote gets no preview;
			// it must not hide unrelated items in this catalog page.
			continue
		}
		coupon, ok := catalogObject(totals["coupon"])
		if !ok {
			continue
		}
		terms, _ := catalogObject(coupon["terms"])
		out[product.ID] = CatalogObjectV2{"couponId": coupon["couponId"], "couponName": coupon["name"], "condition": catalogCouponConditionV220(terms), "finalPrice": totals["totalAmount"], "saving": totals["couponDiscount"], "stack": coupon["stackWithProductDiscount"], "expiresAt": terms["endsAt"], "asOf": now.Format(time.RFC3339)}
	}
	return out, nil
}

func catalogCouponConditionV220(coupon CatalogObjectV2) string {
	minimum := catalogString(coupon, "minimumSpend")
	condition := ""
	minimumAmount, _ := commerceAmountV2(minimum)
	if minimumAmount != nil && minimumAmount.Sign() > 0 {
		condition = "满" + minimum + " "
	}
	if coupon["benefit"] == "FIXED" {
		return condition + "减" + catalogString(coupon, "amountOff")
	}
	rate := big.NewRat(catalogNumber(coupon, "discountRate"), 1000)
	value := rate.FloatString(3)
	for len(value) > 0 && value[len(value)-1] == '0' {
		value = value[:len(value)-1]
	}
	if len(value) > 0 && value[len(value)-1] == '.' {
		value = value[:len(value)-1]
	}
	condition += value + "折"
	cap := catalogString(coupon, "maxDiscount")
	capAmount, _ := commerceAmountV2(cap)
	if capAmount != nil && capAmount.Sign() > 0 {
		condition += fmt.Sprintf(" · 最高减%s", cap)
	}
	return condition
}

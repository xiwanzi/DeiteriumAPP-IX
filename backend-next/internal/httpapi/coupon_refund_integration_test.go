//go:build integration

package httpapi

import (
	"context"
	"testing"
	"time"
)

func TestRefundReturnsCouponOnlyAfterProofAndOldReplayKeepsReuse(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	core := newCommerceTestCore()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	p, _ := catalogProductFixture(t, f)
	c := promotionSaveTestV209(t, f, promotionCouponTestV209())
	if e := f.s.AcknowledgeCouponAttentionV209(ctx, f.buyer.ID, []string{c.ID}, true); e != nil {
		t.Fatal(e)
	}
	d := commerceRunTest(t, f, promotionPrepareTestV209(t, f, promotionQuoteTestV209(t, f, p, "first", 1, "DIRECT"), "first-pay"))
	m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, commerceRefundInput("", d.Version), true)
	if e != nil {
		t.Fatal(e)
	}
	checkCount := func(want int) {
		t.Helper()
		rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 100)
		if e != nil || len(rows) != want {
			t.Fatal("usable coupons", len(rows), want, e)
		}
	}
	checkCount(0)
	core.unknownOnce = "wallet.escrow.refund"
	op, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
	if e != nil || op.State != "UNKNOWN" {
		t.Fatal(op, e)
	}
	checkCount(0)
	d = commerceRunTest(t, f, m)
	if d.State != "REFUNDED" {
		t.Fatal(d)
	}
	checkCount(1)
	attention, _, _, e := f.s.CouponAttentionV209(ctx, f.buyer.ID, "", 100)
	if e != nil || len(attention) != 0 {
		t.Fatal("refund repeated arrival reminder", attention, e)
	}
	// Reuse on a new order, then replay the original refund and historical migration.
	commerceRunTest(t, f, promotionPrepareTestV209(t, f, promotionQuoteTestV209(t, f, p, "reused", 1, "DIRECT"), "second-pay"))
	checkCount(0)
	commerceRunTest(t, f, m)
	if _, e = f.s.DB.Exec("DELETE FROM schema_migrations_next WHERE version='026_refunded_coupon_release.sql'"); e != nil {
		t.Fatal(e)
	}
	if e = f.s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	checkCount(0)
	if core.calls["wallet.escrow.refund"] != 1 {
		t.Fatal("refund sent twice", core.calls)
	}
}

func TestRefundedCouponStillHonorsDisabledDeletedAndExpiredRules(t *testing.T) {
	for _, rule := range []string{"disabled", "deleted", "expired", "failed-refund"} {
		t.Run(rule, func(t *testing.T) {
			f := newCatalogFixture(t)
			ctx := context.Background()
			core := newCommerceTestCore()
			core.mailMode = "revoked"
			f.server.CommerceCore = core
			p, _ := catalogProductFixture(t, f)
			c := promotionSaveTestV209(t, f, promotionCouponTestV209())
			d := commerceRunTest(t, f, promotionPrepareTestV209(t, f, promotionQuoteTestV209(t, f, p, "q", 1, "DIRECT"), "pay"))
			if rule == "expired" {
				c.Body["endsAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
			} else if rule != "failed-refund" {
				c.Body["active"] = false
			}
			var e error
			if rule != "failed-refund" {
				c, e = f.s.SaveCouponV209(ctx, f.admin.ID, c.ID, "change-rule", c.Version, c.Body)
				if e != nil {
					t.Fatal(e)
				}
			}
			if rule == "deleted" {
				if _, e = f.s.DeleteCatalogEntry(ctx, f.admin.ID, c.ID, "coupon", "delete", c.Version); e != nil {
					t.Fatal(e)
				}
			}
			m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, commerceRefundInput("", d.Version), true)
			if e != nil {
				t.Fatal(e)
			}
			if rule == "failed-refund" {
				core.failOnce = "mailbox.revoke"
				op, e := f.server.RunCommerceOperationV2(ctx, m.OperationID, true)
				if e != nil || op.State != "FAILED" {
					t.Fatal(op, e)
				}
			} else {
				commerceRunTest(t, f, m)
			}
			rows, _, _, e := f.s.CouponsV209(ctx, f.buyer.ID, false, "", "", 100)
			if e != nil || len(rows) != 0 {
				t.Fatal("unavailable coupon returned as usable", rows, e)
			}
			count := catalogTestCount(t, f.s, "promotion_redemptions_v209")
			if rule == "failed-refund" && count != 1 || rule != "failed-refund" && count != 0 {
				t.Fatal("incorrect redemption release", count)
			}
		})
	}
}

func TestHistoricalRefundMigrationReleasesOnlyCompleteRefunds(t *testing.T) {
	f := newCatalogFixture(t)
	ctx := context.Background()
	core := newCommerceTestCore()
	core.mailMode = "revoked"
	f.server.CommerceCore = core
	p, _ := catalogProductFixture(t, f)
	c := promotionSaveTestV209(t, f, promotionCouponTestV209())
	d := commerceRunTest(t, f, promotionPrepareTestV209(t, f, promotionQuoteTestV209(t, f, p, "q", 1, "DIRECT"), "pay"))
	m, e := f.s.PrepareCommerceRefundV2(ctx, f.buyer.ID, d.ID, "ORDER", "refund", d.Version, commerceRefundInput("", d.Version), true)
	if e != nil {
		t.Fatal(e)
	}
	d = commerceRunTest(t, f, m)
	if _, e = f.s.DB.Exec("INSERT INTO promotion_redemptions_v209(coupon_id,owner_uuid,resource_id,redeemed_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", c.ID, f.buyer.ServerUUID, d.ID); e != nil {
		t.Fatal(e)
	}
	reapply := func() {
		t.Helper()
		if _, e := f.s.DB.Exec("DELETE FROM schema_migrations_next WHERE version='026_refunded_coupon_release.sql'"); e != nil {
			t.Fatal(e)
		}
		if e := f.s.Migrate(ctx); e != nil {
			t.Fatal(e)
		}
	}
	for _, patch := range []string{"funds_state='UNKNOWN'", "pending_operation_id='pending_test'", "refunded_amount='0.01'", "settled_amount='1.00'"} {
		if _, e = f.s.DB.Exec("UPDATE commerce_resources_v2 SET "+patch+" WHERE resource_id=?", d.ID); e != nil {
			t.Fatal(e)
		}
		reapply()
		if catalogTestCount(t, f.s, "promotion_redemptions_v209") != 1 {
			t.Fatal("migration released incomplete refund", patch)
		}
		if _, e = f.s.DB.Exec("UPDATE commerce_resources_v2 SET funds_state='REFUNDED',pending_operation_id=NULL,refunded_amount=amount,settled_amount='0.00' WHERE resource_id=?", d.ID); e != nil {
			t.Fatal(e)
		}
	}
	reapply()
	reapply()
	if catalogTestCount(t, f.s, "promotion_redemptions_v209") != 0 || catalogTestCount(t, f.s, "commerce_resources_v2") != 1 || catalogTestCount(t, f.s, "promotion_coupon_batches_v209") != 1 {
		t.Fatal("migration lost history or failed to return coupon")
	}
}

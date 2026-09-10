import test from "node:test";
import assert from "node:assert/strict";
import { salePrice, activeCoupons, couponScope, couponConditions } from "./promotions.js";

test("display price preserves cents and large integers without floating point", () => {
  assert.equal(salePrice("0.05", 5000), "0.03");
  assert.equal(salePrice("11.10", 8000), "8.88");
  assert.equal(salePrice("900719925474.99", 8500), "765611936653.74");
});
test("coupon list drops expired coupons exactly at the server deadline", () => {
  const coupon = { active: true, startsAt: "2026-09-10T00:00:00Z", endsAt: "2026-09-11T00:00:00Z" };
  assert.equal(activeCoupons([coupon], Date.parse("2026-09-10T23:59:59Z")).length, 1);
  assert.equal(activeCoupons([coupon], Date.parse(coupon.endsAt)).length, 0);
  assert.equal(activeCoupons([{ ...coupon, active: false }], Date.parse(coupon.startsAt)).length, 0);
});
test("scope and cap summaries remain explicit for restricted coupons", () => {
  const coupon = { storeIds: ["eos"], productIds: ["gift"], benefit: "PERCENT", minimumSpend: "100.00", maxDiscount: "20.00" };
  assert.equal(couponScope(coupon, [{ storeId: "eos", name: "EOS Lab旗舰店" }], [{ productId: "gift", title: "探索补给" }]), "EOS Lab旗舰店 · 探索补给");
  assert.equal(couponConditions(coupon), "满 100 信用点可用 · 最多减 20");
});

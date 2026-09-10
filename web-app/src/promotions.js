export const rateLabel = (rate) => `${Number((Number(rate || 10000) / 1000).toFixed(2))} 折`;
export function salePrice(price, rate = 10000) {
  const [whole, fraction = ""] = String(price || "0").split(".");
  if (!/^\d+$/.test(whole) || !/^\d{0,2}$/.test(fraction)) return "0.00";
  const cents = BigInt(whole) * 100n + BigInt((fraction + "00").slice(0, 2));
  const result = (cents * BigInt(Math.round(Number(rate))) + 5000n) / 10000n;
  return `${result / 100n}.${String(result % 100n).padStart(2, "0")}`;
}
export function couponBenefit(coupon) {
  return coupon.benefit === "FIXED" ? `减 ${Number(coupon.amountOff)}` : rateLabel(coupon.discountRate);
}
export function couponConditions(coupon) {
  const threshold = Number(coupon.minimumSpend) > 0 ? `满 ${Number(coupon.minimumSpend)} 信用点可用` : "无门槛";
  const cap = coupon.benefit === "PERCENT" && Number(coupon.maxDiscount) > 0 ? ` · 最多减 ${Number(coupon.maxDiscount)}` : "";
  return threshold + cap;
}
export function couponScope(coupon, stores = [], products = []) {
  const names = (ids, rows, key, title) => ids.map((id) => rows.find((row) => row[key] === id)?.[title]).filter(Boolean);
  const shop = coupon.storeIds?.length ? names(coupon.storeIds, stores, "storeId", "name").join("、") || `${coupon.storeIds.length} 家指定店铺` : "全部店铺";
  const goods = coupon.productIds?.length ? names(coupon.productIds, products, "productId", "title").join("、") || `${coupon.productIds.length} 件指定商品` : "全部商品";
  return `${shop} · ${goods}`;
}
export function couponDate(value) {
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat("zh-CN", { timeZone: "Asia/Shanghai", month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit", hour12: false }).format(date) : "待设置";
}
export function activeCoupons(coupons, now = Date.now()) {
  return coupons.filter((coupon) => coupon.active && Date.parse(coupon.startsAt) <= now && Date.parse(coupon.endsAt) > now);
}

export function couponStatus(coupon, now = Date.now()) {
  if (coupon.publicationState === "DRAFT") return "草稿";
  if (!coupon.active) return "已停用";
  if (Date.parse(coupon.endsAt) <= now) return "已结束";
  return Date.parse(coupon.startsAt) > now ? "未开始" : "进行中";
}

export function couponPublishSelection(coupons) {
  if (!coupons.length || coupons.length > 100) throw new Error("每批请选择 1–100 张草稿。");
  const seen = new Set();
  return coupons.map((coupon) => {
    if (coupon.publicationState !== "DRAFT" || !coupon.couponId || !Number.isSafeInteger(coupon.version) || coupon.version < 1 || seen.has(coupon.couponId)) throw new Error("所选草稿已变化，请刷新后重新选择。");
    seen.add(coupon.couponId);
    return { couponId: coupon.couponId, expectedVersion: coupon.version };
  }).sort((a, b) => a.couponId.localeCompare(b.couponId));
}

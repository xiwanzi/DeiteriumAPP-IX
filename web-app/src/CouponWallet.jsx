import React, { useEffect, useRef, useState } from "react";
import { Ticket, Check } from "lucide-react";
import { Button } from "./components.jsx";
import { activeCoupons, couponBenefit, couponConditions, couponDate, couponScope } from "./promotions.js";

export function CouponCard({ coupon, stores = [], products = [], preview = false }) {
  return <article className="coupon-card">
    <div className="coupon-card-top"><Ticket size={23} /><span>{coupon.type === "ITEM" ? "单品优惠" : "整单优惠"}</span></div>
    <h3>{coupon.name || "你的专属优惠"}</h3>
    <div className="coupon-value">{couponBenefit(coupon)}</div>
    <div className="coupon-conditions">{couponConditions(coupon)}</div>
    <p className="coupon-scope">{coupon.scopeDescription || couponScope(coupon, stores, products)}</p>
    <div className="coupon-card-footer"><span>{coupon.stackWithProductDiscount ? "可与商品折扣同享" : "与商品折扣自动择优"}{coupon.type === "ITEM" ? " · 仅限一件" : ""}</span><span>有效至 {couponDate(coupon.endsAt)} · 北京时间</span><span className="coupon-auto"><Check size={13} />{preview ? "结算时自动择优使用" : "无需领取，结算自动使用"}</span></div>
  </article>;
}

export default function CouponWallet({ client }) {
  const [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""), [now, setNow] = useState(Date.now());
  const generation = useRef(0), offset = useRef(0);
  const load = async (more = false) => {
    const current = ++generation.current; setBusy(true); setError("");
    try {
      const r = await client.request(`/api/v1/store/coupons?limit=30${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`);
      if (current !== generation.current) return;
      const server = Date.parse(r.serverTime); if (Number.isFinite(server)) offset.current = server - Date.now();
      setNow(Date.now() + offset.current); setItems((old) => more ? [...old, ...r.data] : r.data); setCursor(r.page?.nextCursor);
    } catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => {
    load(); const timer = setInterval(() => setNow(Date.now() + offset.current), 1000);
    const focus = () => { if (!document.hidden) load(); }; document.addEventListener("visibilitychange", focus);
    return () => { generation.current++; clearInterval(timer); document.removeEventListener("visibilitychange", focus); };
  }, [client]);
  const visible = activeCoupons(items, now);
  return <div className="coupon-wallet"><p className="description">每笔结算会自动选用一张最优惠的券。</p>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="coupon-grid">{visible.map((coupon) => <CouponCard key={coupon.couponId} coupon={coupon} />)}</div>
    {!visible.length && <div className="coupon-empty"><Ticket size={38} /><h3>{busy ? "正在查看优惠" : "期待下一份优惠"}</h3><p>{busy ? "正在获取你的可用优惠券。" : "暂无可用优惠券。店铺发放后，会自动出现在这里。"}</p></div>}
    <div className="button-row"><Button secondary disabled={busy} onClick={() => load()}>刷新</Button>{cursor && <Button secondary disabled={busy} onClick={() => load(true)}>查看更多</Button>}</div>
  </div>;
}

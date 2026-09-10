import React from "react";
import { Field, Toggle } from "./components.jsx";
import { credit } from "./format.js";
import { rateLabel, salePrice } from "./promotions.js";

export function ProductDiscountFields({ value, setValue }) {
  const rate = value.discountRate || 10000;
  return <div>
    <Toggle label="商品折扣" description="发布后直接显示折后价，优惠券会按叠加规则自动比较。" checked={rate < 10000} onChange={(enabled) => setValue((v) => ({ ...v, discountRate: enabled ? 9000 : 10000 }))} />
    {rate < 10000 && <Field label="商品折扣（折）" type="number" min="0.01" max="9.99" step="0.01" required value={rate / 1000} onChange={(e) => setValue((v) => ({ ...v, discountRate: Math.round(Number(e.target.value) * 1000) }))} hint="例如 8.5 表示原价的 85%。" />}
    <div className="price-preview-strip"><span>展示售价</span><strong>{credit(salePrice(value.price, rate))}</strong>{rate < 10000 && <><del className="sale-original">{credit(value.price || "0")}</del><span className="badge">{rateLabel(rate)}</span></>}<small>信用点</small></div>
  </div>;
}

export function ProductLimitFields({ value, setValue }) {
  const limits = { lifetime: 0, daily: 0, weekly: 0, monthly: 0, dailyTime: "00:00", weeklyDay: 1, weeklyTime: "00:00", monthlyDay: 1, monthlyTime: "00:00", ...value.purchaseLimits };
  const update = (patch) => setValue((v) => ({ ...v, purchaseLimits: { ...limits, ...patch } }));
  return <>
    <p className="muted">按每位玩家统计，可同时开启多项。刷新时间统一使用北京时间。</p>
    <Field label="每笔订单最多购买" type="number" required min={1} max={999} value={value.limitPerOrder} onChange={(e) => setValue((v) => ({ ...v, limitPerOrder: Number(e.target.value) }))} />
    {[["lifetime", "累计限购", "限制每位玩家累计购买的数量。"], ["daily", "每日限购", "默认每天 00:00 刷新。"], ["weekly", "每周限购", "在指定星期与时间刷新。"], ["monthly", "每月限购", "在指定日期与时间刷新，短月份按最后一天。"]].map(([key, label, description]) => <div className="limit-row" key={key}>
      <Toggle label={label} description={description} checked={limits[key] > 0} onChange={(enabled) => update({ [key]: enabled ? 1 : 0 })} />
      {limits[key] > 0 && <div className="limit-reset">
        <Field label="每位玩家限购数量" type="number" required min={1} max={999999} value={limits[key]} onChange={(e) => update({ [key]: Number(e.target.value) })} />
        {key === "weekly" && <Field label="每周刷新日"><select value={limits.weeklyDay} onChange={(e) => update({ weeklyDay: Number(e.target.value) })}>{["一", "二", "三", "四", "五", "六", "日"].map((day, i) => <option key={day} value={i + 1}>星期{day}</option>)}</select></Field>}
        {key === "monthly" && <Field label="每月刷新日"><select value={limits.monthlyDay} onChange={(e) => update({ monthlyDay: Number(e.target.value) })}>{Array.from({ length: 31 }, (_, i) => <option key={i} value={i + 1}>{i + 1} 日</option>)}</select></Field>}
        {key !== "lifetime" && <Field label="刷新时间" type="time" required value={limits[key + "Time"]} onChange={(e) => update({ [key + "Time"]: e.target.value })} />}
      </div>}
    </div>)}
  </>;
}

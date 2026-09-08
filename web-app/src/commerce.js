import { dateKey, money } from "./domain.js";
import { players } from "./data.js";

export function transactionRows(state) {
  const orders = state.orders.map((o) => {
    const paid = !["待付款", "已取消"].includes(o.status);
    const refunded = o.refundedAmount ?? (o.status === "已退款" ? o.amount : 0);
    const settled =
      o.settledAmount ??
      (["已完成", "已领取", "已验收"].includes(o.status)
        ? o.amount - refunded
        : 0);
    return {
      ...o,
      kind: "ORDER",
      paidAmount: paid ? o.amount : 0,
      refunded,
      settled,
      held: paid ? Math.max(0, o.amount - refunded - settled) : 0,
      number: o.number || `DT${o.id.slice(0, 8).toUpperCase()}`,
    };
  });
  const commissions = state.commissions.map((c) => ({
    id: c.id,
    number: `WT${c.id.slice(0, 8).toUpperCase()}`,
    kind: "COMMISSION",
    title: c.title,
    channel: "委托",
    buyer: c.owner,
    seller: c.worker || null,
    time: c.createdAt || "2026-09-07T06:00:00Z",
    amount: c.price,
    paidAmount: c.price,
    refunded: c.status === "CANCELLED" ? c.price : 0,
    settled: c.status === "CONFIRMED" ? c.price : 0,
    held: ["OPEN", "ACTIVE", "COMPLETED"].includes(c.status) ? c.price : 0,
    status: {
      OPEN: "待接取",
      ACTIVE: "进行中",
      COMPLETED: "待验收",
      CONFIRMED: "已完成",
      CANCELLED: "已退款",
    }[c.status],
  }));
  const transfers = (state.transfers || []).map((t) => ({
    ...t,
    kind: "TRANSFER",
    channel: "转账",
    number: `ZZ${t.id.slice(0, 8).toUpperCase()}`,
    title: t.note || "信用点转账",
    buyer: t.from,
    seller: t.to,
    paidAmount: t.amount,
    refunded: 0,
    settled: t.amount,
    held: 0,
    status: "已完成",
  }));
  return [...orders, ...commissions, ...transfers].sort((a, b) =>
    b.time.localeCompare(a.time),
  );
}
export function filterTransactions(
  rows,
  {
    player = "",
    role = "全部身份",
    channel = "全部渠道",
    status = "全部状态",
    from = "",
    to = "",
    query = "",
  } = {},
) {
  const q = query.trim().toLowerCase();
  return rows.filter(
    (t) =>
      (!player ||
        (role === "卖出 / 收入"
          ? t.seller === player
          : role === "买入 / 支出"
            ? t.buyer === player
            : t.buyer === player || t.seller === player)) &&
      (channel === "全部渠道" || t.channel === channel) &&
      (status === "全部状态" || t.status === status) &&
      (!from || dateKey(t.time) >= from) &&
      (!to || dateKey(t.time) <= to) &&
      (!q ||
        `${t.title} ${t.number} ${t.id} ${t.buyer} ${t.seller}`
          .toLowerCase()
          .includes(q)),
  );
}
export function salesSummary(rows, player) {
  const sales = rows.filter(
    (r) => r.channel === "玩家市场" && r.seller === player && r.paidAmount > 0,
  );
  const sum = (k) => sales.reduce((n, r) => n + r[k], 0);
  return {
    gross: sum("paidAmount"),
    refund: sum("refunded"),
    sold: sum("paidAmount") - sum("refunded"),
    settled: sum("settled"),
    held: sum("held"),
    orderCount: sales.length,
  };
}
export function allPlayerSales(state, filters = {}) {
  const rows = filterTransactions(transactionRows(state), {
    from: filters.from,
    to: filters.to,
  });
  const q = (filters.query || "").trim().toLowerCase();
  return players
    .filter((p) => `${p.name} ${p.qq}`.toLowerCase().includes(q))
    .map((p) => ({ ...p, ...salesSummary(rows, p.id) }))
    .sort((a, b) => b.sold - a.sold);
}
export function exportTransactions(rows) {
  const escape = (value) => {
    let text = String(value ?? "");
    if (/^[\s]*[=+@-]/.test(text) || /^[\t\r\n]/.test(text)) text = "'" + text;
    return '"' + text.replaceAll('"', '""') + '"';
  };
  const headings = [
    "交易号",
    "渠道",
    "标题",
    "付款方",
    "收款方",
    "成交原额",
    "退款",
    "已结算",
    "担保中",
    "状态",
    "时间",
  ];
  return (
    "\ufeff" +
    [
      headings,
      ...rows.map((r) => [
        r.number,
        r.channel,
        r.title,
        r.buyer,
        r.seller,
        money(r.paidAmount),
        money(r.refunded),
        money(r.settled),
        money(r.held),
        r.status,
        r.time,
      ]),
    ]
      .map((r) => r.map(escape).join(","))
      .join("\r\n")
  );
}
export function downloadCsv(rows) {
  const url = URL.createObjectURL(
    new Blob([exportTransactions(rows)], { type: "text/csv;charset=utf-8" }),
  );
  const link = document.createElement("a");
  link.href = url;
  link.download = `deuterium-transactions-${dateKey(new Date())}.csv`;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

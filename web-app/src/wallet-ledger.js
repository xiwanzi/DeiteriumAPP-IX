import { credit } from "./format.js";

export const businessLabels = { TRANSFER: "玩家转账", GAME: "游戏内收支", OFFICIAL_STORE: "官方商城", MARKET_ORDER: "玩家市场", COMMISSION: "委托" };
export const statusLabels = { success: "已完成", failed: "失败", processing: "处理中", unknown: "待核对" };

const dayLength = 86400000;
const beijingOffset = 8 * 3600000;
const clock = new Intl.DateTimeFormat("zh-CN", { timeZone: "Asia/Shanghai", hour: "2-digit", minute: "2-digit", hour12: false });

export function walletDay(value) {
  if (value == null || value === "") return "";
  const instant = new Date(value).getTime();
  return Number.isFinite(instant) ? new Date(instant + beijingOffset).toISOString().slice(0, 10) : "";
}

export function walletTime(value) {
  return walletDay(value) ? clock.format(new Date(value)) : "—";
}

export function ledgerQuery({ direction, kind, day }, today = walletDay(Date.now())) {
  const query = {};
  if (direction) query.direction = direction;
  if (kind) query.businessType = kind;
  if (day) {
    const start = Date.parse(`${day}T00:00:00+08:00`);
    query.from = new Date(start).toISOString();
    if (day < today) query.to = new Date(start + dayLength).toISOString();
  }
  return query;
}

export function recordAmount(record) {
  const amount = credit(record.amount);
  if (amount === "—") return amount;
  const sign = {income:"+", expense:"−"}[record.direction] || "";
  return sign + amount;
}

export function recordTitle(record) {
  const player = record.otherPlayer?.displayName || record.otherPlayer?.gameId;
  if (player && (record.businessType === "TRANSFER" || !record.businessType)) {
    return record.direction === "income" ? `收到 ${player} 的转账` : `转给 ${player}`;
  }
  return record.title || businessLabels[record.businessType] || record.note || "信用点交易";
}

export const poolKinds = { STANDARD: "常驻池", LIMITED: "轮换限定", EVENT: "独立活动" };
export const ticketKinds = { STANDARD: "常驻券", LIMITED: "限定券" };
export const deliveryStates = { PENDING: "等待交付", PREPARING: "准备物品", ISSUING: "正在写入背包", INVENTORY: "已放入背包", MAIL_PENDING: "等待寄送邮箱", MAIL: "已寄送邮箱", REVIEW: "等待保存回执核对" };

export function newPool(kind = "LIMITED") {
  return { id: "", version: 0, publishedRevision: 0, enabled: false, definition: {
    name: "", title: "", subtitle: "", description: "", kind, seriesId: "", startsAt: null, endsAt: null, visibleAt: null, sort: 0,
    art: { hero: "", rail: "", detail: "", fallback: "", theme: "violet", heroX: .5, heroY: .5 },
    eventGoldPpm: kind === "EVENT" ? 5000 : 0, eventPity: kind === "EVENT" ? 80 : 0, rewards: [],
  } };
}

export function ruleText(kind, scheme, definition = {}) {
  if (kind === "EVENT") return `每抽 ${Number(definition.eventGoldPpm || 0) / 10000}% · ${definition.eventPity || "—"} 抽必出金 · 100% UP`;
  if (kind === "STANDARD") return "每抽 0.5% · 80 抽必出金";
  return scheme === "120" ? "每抽 1.2% · 每轮随机 100～120 抽保底 · 100% UP" : "每抽 0.5% · 80 抽必出金 · 100% UP";
}

export function scheduleInput(iso) {
  if (!iso) return "";
  const value = new Date(iso);
  return Number.isFinite(value.getTime()) ? new Date(value.getTime() + 8 * 3600000).toISOString().slice(0, 16) : "";
}

export function scheduleISO(value) {
  if (!value) return null;
  const result = new Date(`${value}:00+08:00`);
  if (!Number.isFinite(result.getTime())) throw new Error("日期时间不正确");
  return result.toISOString();
}

export function creditMinor(text) {
  if (!/^\d{1,7}(\.\d{1,2})?$/.test(String(text))) throw new Error("兑换价格须为正数或 0，最多两位小数");
  const [whole, fraction = ""] = String(text).split(".");
  const amount = Number(whole) * 100 + Number(fraction.padEnd(2, "0"));
  if (amount > 100000000) throw new Error("兑换价格超过上限");
  return amount;
}

export function creditText(minor) { return (Number(minor || 0) / 100).toFixed(2); }
export function dateText(value) { return value ? new Date(value).toLocaleString("zh-CN", { timeZone: "Asia/Singapore", hour12: false }) : "长期"; }

export function poolIssues(d) {
  const issues = [];
  if (!d.name?.trim() || !d.title?.trim()) issues.push("填写卡池名称和页面标题");
  if (!d.art?.hero || !d.art?.rail) issues.push("选择首页宣传图和侧栏封面");
  if (d.kind !== "STANDARD" && (!d.startsAt || !d.endsAt)) issues.push("设置开放和结束时间");
  if (d.startsAt && d.endsAt && new Date(d.endsAt) <= new Date(d.startsAt)) issues.push("结束时间必须晚于开放时间");
  if (d.visibleAt && d.startsAt && new Date(d.visibleAt) > new Date(d.startsAt)) issues.push("预告时间不能晚于开放时间");
  if (!d.rewards?.some(r => r.stars === 6) || !d.rewards?.some(r => r.stars < 6)) issues.push("同时配置金色与普通品质奖品");
  for (const r of d.rewards || []) {
    if (!r.itemRef || !r.itemRevision || !r.payloadSha256 || !r.servers?.length) issues.push(`${r.name || "奖品"} 尚未选择有效的物品库版本`);
    if (d.kind !== "STANDARD" && r.stars === 6 && !r.up) issues.push(`${r.name} 必须标为 UP`);
    if (!r.modelKey && !r.art?.fallback) issues.push(`${r.name || "奖品"} 需要展示模型或备用图片`);
  }
  return [...new Set(issues)];
}

export function rewardFromItem(item) {
  return { key: `weapon_${crypto.randomUUID().replaceAll("-", "").slice(0, 16)}`, name: item.displayName, gunId: "", modelKey: "", stars: 6, weight: 1, up: true,
    itemRef: item.itemRef, itemRevision: item.revision, payloadSha256: item.payloadSha256,
    inventoryDomain: item.inventoryDomain, servers: [...(item.compatibleServerIds || [])],
    art: { hero: "", rail: "", detail: "", fallback: "", theme: "", heroX: .5, heroY: .5 } };
}

export function csvCell(value) {
  let text = String(value ?? "");
  if (/^[=+@-]/.test(text)) text = `'${text}`;
  return `"${text.replaceAll('"', '""')}"`;
}

export function downloadCSV(name, rows) {
  const blob = new Blob(["\ufeff", rows.map(row => row.map(csvCell).join(",")).join("\r\n")], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob), link = document.createElement("a"); link.href = url; link.download = name; link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

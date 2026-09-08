export function id() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  const bytes = globalThis.crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export const money = (cents) => (cents / 100).toLocaleString("zh-CN", {
  minimumFractionDigits: 2, maximumFractionDigits: 2,
});

// API money stays decimal text so large values never pass through binary floats.
export function credit(value) {
  if (value === undefined || value === null || !/^-?\d+(\.\d{1,2})?$/.test(String(value))) return "—";
  const [whole, fraction = ""] = String(value).split(".");
  return `${whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}.${fraction.padEnd(2, "0")}`;
}

export function transferAmount(value) {
  if (!/^\d{1,7}(\.\d{1,2})?$/.test(value)) throw new Error("请输入有效金额，最多两位小数。");
  const [whole, fraction = ""] = value.split(".");
  if (BigInt(whole) * 100n + BigInt(fraction.padEnd(2, "0")) <= 0n) throw new Error("转账金额必须大于 0。");
  return `${BigInt(whole)}.${fraction.padEnd(2, "0")}`;
}

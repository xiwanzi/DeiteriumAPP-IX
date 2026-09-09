export const fundsLabels = { UNPAID: "尚未付款", PROCESSING: "资金处理中", PAID: "付款已确认", HELD: "报酬或货款已托管", SETTLING: "结算中", SETTLED: "已结算", REFUNDING: "退款中", REFUNDED: "已退款", INTERVENTION_HOLD: "等待平台处理", UNKNOWN: "资金结果待核对" };
export function businessResource(data) {
  if (data?.order?.orderId) return { type: "ORDER", value: data.order };
  if (data?.commission?.commissionId) return { type: "COMMISSION", value: data.commission };
  if (data?.orderId) return { type: "ORDER", value: data };
  if (data?.commissionId) return { type: "COMMISSION", value: data };
  return null;
}
export function businessOperation(data) { return data?.operation?.operationId ? data.operation : data?.operationId ? data : null; }
export function fundsPending(value) { return Boolean(value?.pendingOperationId) || ["PROCESSING", "SETTLING", "REFUNDING", "UNKNOWN"].includes(value?.fundsStatus); }
export function paymentConfirmed(value) { return !fundsPending(value) && ["PAID", "HELD", "SETTLED"].includes(value?.fundsStatus); }
export function resourcePath(type, resourceId) { return `/api/v1/${type === "COMMISSION" ? "commissions" : "orders"}/${encodeURIComponent(resourceId)}`; }
export function resourceId(value) { return value?.orderId || value?.commissionId; }
export function savedBusiness(userId, storage = globalThis.sessionStorage) {
  try { const values = JSON.parse(storage.getItem(`deuterium-business:${userId}`)); return values && typeof values === "object" ? values : {}; }
  catch { return {}; }
}
export function saveBusiness(userId, entry, storage = globalThis.sessionStorage) {
  const values = savedBusiness(userId, storage); values[entry.body.clientRequestId] = entry;
  storage.setItem(`deuterium-business:${userId}`, JSON.stringify(values));
}
export function clearBusiness(userId, clientRequestId, storage = globalThis.sessionStorage) {
  const values = savedBusiness(userId, storage); delete values[clientRequestId]; storage.setItem(`deuterium-business:${userId}`, JSON.stringify(values));
}
export async function findBusiness(client, entry) {
  const path = entry.operationId ? `/api/v1/operations/${encodeURIComponent(entry.operationId)}` : `/api/v1/operations/by-client-request?clientRequestId=${encodeURIComponent(entry.body.clientRequestId)}&kind=${encodeURIComponent(entry.kind)}`;
  let response;
  try { response = await client.request(path); }
  catch (error) { error.originalOperationNotFound = !entry.operationId && error.status === 404; throw error; }
  const operation = businessOperation(response.data);
  if (!operation) throw new Error("服务尚未返回这笔业务的处理结果。");
  const type = entry.resourceType || operation.resourceType, reference = entry.resourceId || operation.resourceId;
  let resource = null;
  if (reference && ["ORDER", "COMMISSION"].includes(type)) { try { const r = await client.request(resourcePath(type, reference)); resource = businessResource(r.data); } catch (error) { error.knownOperation = operation; throw error; } }
  return { operation, resource };
}
export function canReplayBusiness(entry, userId, origin) {
  return !entry.operationId && entry.userId === userId && entry.origin === origin && /^https?:\/\//.test(origin || "") && ({ STORE_PURCHASE: "/api/v1/store/orders", MARKET_PURCHASE: "/api/v1/market/orders", AI_PURCHASE: "/api/v1/ai/purchases", COMMISSION_PUBLISH: "/api/v1/commissions" })[entry.kind] === entry.path && Boolean(entry.body?.clientRequestId);
}
export async function replayUnreceivedBusiness(client, entry, userId, origin) {
  if (!canReplayBusiness(entry, userId, origin)) throw new Error("当前账号、网站或原处理记录不允许重放这笔请求，请刷新订单核对。");
  try { return await findBusiness(client, entry); }
  catch (error) { if (!error.originalOperationNotFound) throw error; }
  const response = await client.request(entry.path, { method: "POST", body: entry.body });
  return { operation: businessOperation(response.data), resource: businessResource(response.data) };
}

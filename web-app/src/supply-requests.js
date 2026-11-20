const prefix = "deuterium.supply.pending.";
const invalidRequest = "待确认请求记录无法读取，请先核对操作流水";

export function pendingSupplyRequest(scope) {
  try {
    const raw = sessionStorage.getItem(prefix + scope);
    if (!raw) return null;
    const pending = JSON.parse(raw);
    if (!pending || typeof pending.path !== "string" || typeof pending.fingerprint !== "string" || !pending.body?.clientRequestId) return { invalid: true };
    return pending;
  } catch { return { invalid: true }; }
}

export async function mutateSupply(client, scope, path, body) {
  const fingerprint = JSON.stringify({ path, body });
  let pending = pendingSupplyRequest(scope);
  if (pending?.invalid) throw new Error(invalidRequest);
  if (pending && pending.fingerprint !== fingerprint) throw new Error("上一笔请求结果尚未确认，请先恢复原请求，再提交新修改");
  if (!pending) {
    pending = { fingerprint, path, body: { ...body, clientRequestId: crypto.randomUUID() } };
    sessionStorage.setItem(prefix + scope, JSON.stringify(pending));
  }
  return retrySupply(client, scope);
}

export async function retrySupply(client, scope) {
  const pending = pendingSupplyRequest(scope);
  if (!pending) throw new Error("没有待确认请求");
  if (pending.invalid) throw new Error(invalidRequest);
  try {
    const result = await client.request(pending.path, { method: "POST", body: pending.body });
    sessionStorage.removeItem(prefix + scope);
    return result.data;
  } catch (error) {
    if ([400, 409, 422].includes(error.status) && error.code !== "RESULT_UNKNOWN") sessionStorage.removeItem(prefix + scope);
    throw error;
  }
}

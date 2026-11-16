import test from "node:test";
import assert from "node:assert/strict";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";

function storage(t) {
  const saved = Object.getOwnPropertyDescriptor(globalThis, "sessionStorage");
  const entries = new Map();
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: {
    getItem: key => entries.get(key) ?? null,
    setItem: (key, value) => entries.set(key, value),
    removeItem: key => entries.delete(key),
  } });
  t.after(() => { if (saved) Object.defineProperty(globalThis, "sessionStorage", saved); else delete globalThis.sessionStorage; });
  return entries;
}

test("lost supply reply recovers the same request and original values", async t => {
  storage(t);
  const calls = [];
  const client = { request: async (path, options) => {
    calls.push({ path, body: options.body });
    if (calls.length === 1) throw new Error("connection lost after commit");
    return { data: { limited: 17 } };
  } };
  await assert.rejects(mutateSupply(client, "player", "/api/v1/admin/supply/tickets", { amount: 17 }), /connection lost/);
  assert.ok(pendingSupplyRequest("player"));
  assert.deepEqual(await retrySupply(client, "player"), { limited: 17 });
  assert.deepEqual(calls[0], calls[1]);
  assert.equal(pendingSupplyRequest("player"), null);
});

test("new adjustment cannot replace a supply request with an unknown result", async t => {
  storage(t);
  let calls = 0;
  const client = { request: async () => { calls++; throw new Error("offline"); } };
  await assert.rejects(mutateSupply(client, "player", "/tickets", { amount: 17 }));
  await assert.rejects(mutateSupply(client, "player", "/tickets", { amount: 20 }), /恢复原请求/);
  assert.equal(calls, 1);
  assert.equal(pendingSupplyRequest("player").body.amount, 17);
});

test("definite rejected supply mutation permits a corrected request", async t => {
  storage(t);
  const client = { request: async () => { throw Object.assign(new Error("version changed"), { status: 409 }); } };
  await assert.rejects(mutateSupply(client, "pool", "/pool", { expectedVersion: 1 }));
  assert.equal(pendingSupplyRequest("pool"), null);
});

test("corrupt supply journal remains visible without sending a replacement", async t => {
  const entries = storage(t);
  entries.set("deuterium.supply.pending.pool", "{corrupt");
  let calls = 0;
  const client = { request: async () => { calls++; } };
  assert.deepEqual(pendingSupplyRequest("pool"), { invalid: true });
  await assert.rejects(retrySupply(client, "pool"), /核对操作流水/);
  await assert.rejects(mutateSupply(client, "pool", "/pool", {}), /核对操作流水/);
  assert.equal(calls, 0);
  assert.equal(entries.get("deuterium.supply.pending.pool"), "{corrupt");
});

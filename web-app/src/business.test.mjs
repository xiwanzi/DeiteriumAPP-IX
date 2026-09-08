import test from "node:test";
import assert from "node:assert/strict";
import { businessOperation, businessResource, paymentConfirmed, fundsPending, saveBusiness, savedBusiness, clearBusiness, findBusiness, canReplayBusiness, replayUnreceivedBusiness } from "./business.js";
import { DeuteriumClient } from "./api.js";

test("paid-looking resource stays pending while its authoritative operation is unknown", () => {
  assert.equal(paymentConfirmed({ fundsStatus: "HELD", pendingOperationId: "operation-bind" }), false);
  assert.equal(paymentConfirmed({ fundsStatus: "UNKNOWN" }), false);
  assert.equal(paymentConfirmed({ fundsStatus: "UNPAID" }), false);
  assert.equal(paymentConfirmed({ fundsStatus: "HELD", pendingOperationId: null }), true);
  assert.equal(fundsPending({ fundsStatus: "REFUNDING" }), true);
  assert.equal(fundsPending({ fundsStatus: "SETTLING" }), true);
});
test("operation envelopes cannot be mistaken for a completed order", () => {
  const op = { operationId: "op", status: "UNKNOWN", resourceType: "ORDER", resourceId: "order" };
  assert.deepEqual(businessOperation({ operation: op }), op);
  assert.equal(businessResource({ operation: op }), null);
  assert.deepEqual(businessResource({ commission: { commissionId: "commission" } }), { type: "COMMISSION", value: { commissionId: "commission" } });
});
test("business recovery journals are isolated by account and preserve exact payment requests", () => {
  const values = new Map(), storage = { getItem: (key) => values.get(key), setItem: (key, value) => values.set(key, value) };
  const entry = { path: "/api/v1/store/orders", kind: "STORE_PURCHASE", body: { clientRequestId: "same-id", quoteId: "quote", expectedQuoteVersion: 1 } };
  saveBusiness("alice", entry, storage);
  assert.deepEqual(savedBusiness("alice", storage)["same-id"], entry);
  assert.deepEqual(savedBusiness("bob", storage), {});
  clearBusiness("bob", "same-id", storage); assert.ok(savedBusiness("alice", storage)["same-id"]);
  clearBusiness("alice", "same-id", storage); assert.deepEqual(savedBusiness("alice", storage), {});
});
test("unknown payment recovery only reads the original operation and its bound resource", async () => {
  const paths = [], client = { request: async (path) => {
    paths.push(path);
    return path.includes("/operations/") ? { data: { operationId: "operation", status: "COMPLETED", resourceType: "ORDER", resourceId: "order" } } : { data: { orderId: "order", fundsStatus: "HELD", pendingOperationId: null } };
  } };
  const result = await findBusiness(client, { body: { clientRequestId: "original-id" }, kind: "STORE_PURCHASE" });
  assert.equal(paths[0], "/api/v1/operations/by-client-request?clientRequestId=original-id&kind=STORE_PURCHASE");
  assert.equal(paths[1], "/api/v1/orders/order"); assert.equal(paymentConfirmed(result.resource.value), true);
});
test("separate quote intentions can use separate HTTP idempotency keys without changing quote body", async () => {
  const calls = [], client = new DeuteriumClient({ fetchImpl: async (path, options) => { calls.push(options); return new Response(JSON.stringify({ data: {} })); } });
  const body = { channel: "OFFICIAL_STORE", items: [], delivery: { method: "MAILBOX" } };
  await client.request("/api/v1/checkout/quotes", { method: "POST", body, idempotencyKey: "first-purchase" });
  await client.request("/api/v1/checkout/quotes", { method: "POST", body, idempotencyKey: "second-purchase" });
  assert.notEqual(calls[0].headers["Idempotency-Key"], calls[1].headers["Idempotency-Key"]);
  assert.equal(calls[0].body, calls[1].body);
});

const scopedEntry = { userId: "alice", origin: "https://example.test", path: "/api/v1/store/orders", kind: "STORE_PURCHASE", body: { clientRequestId: "original-key", quoteId: "original-quote", expectedQuoteVersion: 3 } };
test("only explicit original-key 404 can replay the exact original payment", async () => {
  const calls = [], client = { request: async (path, options) => {
    calls.push({ path, options }); if (!options) throw Object.assign(new Error("missing original operation"), { status: 404 });
    return { data: { operation: { operationId: "now-known", status: "UNKNOWN" }, order: { orderId: "order", fundsStatus: "UNKNOWN" } } };
  } };
  const result = await replayUnreceivedBusiness(client, scopedEntry, "alice", scopedEntry.origin);
  assert.equal(calls.length, 2); assert.match(calls[0].path, /by-client-request/); assert.equal(calls[1].path, scopedEntry.path);
  assert.deepEqual(calls[1].options, { method: "POST", body: scopedEntry.body }); assert.equal(result.operation.operationId, "now-known");
});
test("known operation, changed account or changed origin cannot replay a payment", async () => {
  const client = { request: async () => { throw new Error("unexpected network access"); } };
  for (const [entry, user, origin] of [[{ ...scopedEntry, operationId: "known" }, "alice", scopedEntry.origin], [scopedEntry, "bob", scopedEntry.origin], [scopedEntry, "alice", "https://changed.test"], [{ ...scopedEntry, path: "/api/v1/wallet/transfers" }, "alice", scopedEntry.origin]]) {
    assert.equal(canReplayBusiness(entry, user, origin), false); await assert.rejects(replayUnreceivedBusiness(client, entry, user, origin), /不允许重放/);
  }
});
test("network uncertainty and a resource 404 after a known operation never authorize a POST", async () => {
  for (const mode of ["network", "forbidden", "resource-missing"]) {
    const calls = [], client = { request: async (path, options) => {
      calls.push({ path, options });
      if (mode === "resource-missing" && path.includes("/operations/")) return { data: { operationId: "known", status: "UNKNOWN", resourceType: "ORDER", resourceId: "hidden-order" } };
      throw Object.assign(new Error(mode), { status: mode === "resource-missing" ? 404 : mode === "forbidden" ? 403 : 0 });
    } };
    await assert.rejects(replayUnreceivedBusiness(client, scopedEntry, "alice", scopedEntry.origin), (error) => mode !== "resource-missing" || error.knownOperation.operationId === "known");
    assert.ok(calls.every((call) => !call.options));
  }
});
test("an original operation appearing before retry is reused without another POST", async () => {
  const calls = [], client = { request: async (path, options) => { calls.push({ path, options }); return path.includes("/operations/") ? { data: { operationId: "already-created", resourceType: "ORDER", resourceId: "order", status: "COMPLETED" } } : { data: { orderId: "order", fundsStatus: "HELD" } }; } };
  const result = await replayUnreceivedBusiness(client, scopedEntry, "alice", scopedEntry.origin);
  assert.equal(result.operation.operationId, "already-created"); assert.equal(calls.length, 2); assert.ok(calls.every((call) => !call.options));
});

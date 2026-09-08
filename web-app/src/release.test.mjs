import test from "node:test";
import assert from "node:assert/strict";
import { DeuteriumClient } from "./api.js";
import { credit, transferAmount } from "./format.js";
import { publicConfig, backendOrigin } from "../gateway.mjs";

test("release configuration cannot activate a demo login without an upstream", () => {
  assert.deepEqual(publicConfig(null), { mode: "connected", version: "2.0.0", configured: false });
  assert.equal(publicConfig(backendOrigin("http://127.0.0.1:8080")).mode, "connected");
});

test("decimal credit preserves financial precision and missing balance is never zero", () => {
  assert.equal(credit("9007199254740993.25"), "9,007,199,254,740,993.25");
  assert.equal(credit(null), "—");
  assert.equal(credit(undefined), "—");
  assert.equal(credit("0"), "0.00");
  assert.equal(transferAmount("00068.5"), "68.50");
  assert.equal(transferAmount("0.01"), "0.01");
  for (const value of ["0", "-1", "NaN", "0.001", "1e3", "10000000", "1,000"])
    assert.throws(() => transferAmount(value));
});

test("HTTP 202 financial uncertainty remains an error rather than an accepted payment", async () => {
  const client = new DeuteriumClient({ fetchImpl: async () => new Response(JSON.stringify({ error: { code: "TRANSFER_RESULT_UNKNOWN", message: "待核对" } }), { status: 202 }) });
  await assert.rejects(client.transfer({ clientRequestId: "one-operation" }), (error) => error.code === "TRANSFER_RESULT_UNKNOWN" && error.status === 202);
});

test("wallet requests keep CSRF, decimal amounts, confirmed recipient refs and stable request IDs", async () => {
  const calls = [];
  const client = new DeuteriumClient({ fetchImpl: async (path, options) => {
    calls.push({ path, ...options });
    return new Response(JSON.stringify({ data: { csrfToken: "csrf-memory-only", transfer: { status: "unknown" } } }));
  } });
  await client.restore();
  await client.recipients("Player & QQ");
  const operation = { clientRequestId: "one-operation", recipientPlayerRef: "player-confirmed", amount: "0.10", note: "备注" };
  await client.transfer(operation);
  await client.transfer(operation);
  assert.equal(calls[1].path, "/api/v1/wallet/recipients/search?query=Player%20%26%20QQ&type=auto");
  assert.equal(calls[2].headers["X-CSRF-Token"], "csrf-memory-only");
  assert.equal(calls[2].body, calls[3].body);
  assert.equal(JSON.parse(calls[2].body).amount, "0.10");
  await client.transferResult("opaque/reference");
  assert.equal(calls.at(-1).path, "/api/v1/wallet/transfers/opaque%2Freference");
});

test("registration and reset preserve the game verification token contract", async () => {
  const calls = [];
  const client = new DeuteriumClient({ fetchImpl: async (path, options) => {
    calls.push({ path, body: JSON.parse(options.body) });
    return new Response(JSON.stringify({ data: {} }));
  } });
  await client.requestVerification("register", { gameId: "Player", qq: "123456", password: "test-password" });
  assert.deepEqual(calls[0].body, { gameId: "Player", qq: "123456", password: "test-password" });
  await client.completeVerification("reset", { verificationToken: "opaque-token", code: "123456", password: "new-test-password" });
  assert.deepEqual(calls[1].body, { verificationToken: "opaque-token", code: "123456", newPassword: "new-test-password" });
});

import test from "node:test";
import assert from "node:assert/strict";
import { DeuteriumClient } from "./api.js";
import { adminNavigation } from "./admin-navigation.js";

test("YSM entry belongs only to platform administrators", () => {
  for (const permissions of [[], ["core.read"], ["store.manage"], ["audit.read"]]) assert.equal(adminNavigation(permissions).some(([p]) => p.includes("section=ysm")), false);
  assert.equal(adminNavigation(["platform.admin"]).filter(([p]) => p.includes("section=ysm")).length, 1);
});

test("model upload preserves binary, Chinese ID, session CSRF and same-origin credentials", async () => {
  let sent;
  const client = new DeuteriumClient({ fetchImpl: async (path, options) => {
    sent = { path, ...options };
    return new Response(JSON.stringify({ data: path.endsWith("/session") ? { csrfToken: "web-csrf" } : { confirmationToken: "opaque-ticket" } }), { status: 200 });
  } });
  await client.restore();
  const file = new Blob([new Uint8Array([0, 255, 42])]);
  await client.request("/api/v1/admin/ysm/models/stage", { method: "POST", body: file, ysmUpload: { modelId: "测试/模型.ysm", contentType: "application/vnd.deuteriumysm.model", operation: "replace", revision: 11 } });
  assert.equal(sent.body, file);
  assert.equal(sent.credentials, "same-origin");
  assert.equal(sent.headers["X-CSRF-Token"], "web-csrf");
  assert.equal(decodeURIComponent(sent.headers["X-Model-Id"]), "测试/模型.ysm");
  assert.equal(sent.headers["X-Expected-Revision"], "11");
  assert.equal(sent.headers["Content-Type"], "application/vnd.deuteriumysm.model");
  await assert.rejects(client.request("/api/v1/other", { method: "POST", body: file, ysmUpload: {} }));
});

test("uncertain YSM confirmation is never retried automatically", async () => {
  let calls = 0;
  const client = new DeuteriumClient({ fetchImpl: async () => { calls++; throw new Error("disconnected"); } });
  await assert.rejects(client.request("/api/v1/admin/ysm/models/confirm", { method: "POST", body: { confirmationToken: "opaque" } }), /不要重复提交/);
  assert.equal(calls, 1);
});

test("Cloud model upload preserves bytes, revision and Chinese metadata", async () => {
  let sent;
  const client = new DeuteriumClient({ fetchImpl: async (path, options) => {
    sent = { path, ...options };
    return new Response(JSON.stringify({ data: path.endsWith("/session") ? { csrfToken: "web-csrf" } : { revision: 8 } }));
  } });
  await client.restore();
  const file = new Blob([new Uint8Array([255, 0, 7])]);
  await client.request("/api/v1/admin/ysm/cloud/upload", { method: "POST", body: file, idempotencyKey: "stable-key", ysmCloudUpload: {
    assetId: "asset_a", name: "验收模型", format: "ysm", sha256: "a".repeat(64), revision: 7,
    metadata: { modelId: "角色/测试", category: "角色", note: "更新贴图" },
  } });
  assert.equal(sent.body, file); assert.equal(sent.headers["X-CSRF-Token"], "web-csrf");
  assert.equal(sent.headers["X-Asset-Base-Revision"], "7"); assert.equal(sent.headers["Idempotency-Key"], "stable-key");
  assert.equal(decodeURIComponent(sent.headers["X-Asset-Name"]), "验收模型");
  assert.equal(JSON.parse(decodeURIComponent(sent.headers["X-Deuterium-Model-Metadata"])).modelId, "角色/测试");
  await assert.rejects(client.request("/api/v1/other", { method: "POST", body: file, ysmCloudUpload: {} }));
});

test("Cloud mutations with unknown result are never retried", async () => {
  let calls = 0;
  const client = new DeuteriumClient({ fetchImpl: async () => { calls++; throw new Error("disconnected"); } });
  await assert.rejects(client.request("/api/v1/admin/ysm/cloud/models/asset/rollback", { method: "POST", body: { expectedVersion: 7, version: 2 } }), /不要重复提交/);
  assert.equal(calls, 1);
});

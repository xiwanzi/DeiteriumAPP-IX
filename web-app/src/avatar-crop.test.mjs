import test from "node:test";
import assert from "node:assert/strict";
import { avatarCrop } from "./avatar-crop.js";
import { canReplayBusiness, findBusiness } from "./business.js";

test("portrait and landscape avatars fill the circular frame without empty edges", () => {
  for (const [w, h] of [[1600, 900], [900, 1600], [512, 512]]) for (const zoom of [1, 1.5, 4]) for (const point of [[0, 0], [w, h], [w / 2, h / 2]]) {
    const crop = avatarCrop(w, h, zoom, ...point);
    assert.ok(crop.x >= 0 && crop.y >= 0 && crop.x + crop.size <= w && crop.y + crop.size <= h);
    assert.equal(crop.size, Math.min(w, h) / zoom);
  }
  assert.throws(() => avatarCrop(0, 20));
});
test("Saki recovery binds original account, origin and request path", async () => {
  const e = { userId: "alice", origin: "https://example.test", kind: "AI_PURCHASE", path: "/api/v1/ai/purchases", body: { clientRequestId: "once", planId: "plan_pro", expectedPlanVersion: 2 } };
  assert.equal(canReplayBusiness(e, "alice", e.origin), true);
  assert.equal(canReplayBusiness(e, "bob", e.origin), false);
  assert.equal(canReplayBusiness(e, "alice", "https://other.test"), false);
  const paths = [];
  await findBusiness({ request: async (path) => { paths.push(path); return { data: { operationId: "op_one", status: "UNKNOWN" } }; } }, { ...e, operationId: "op_one" });
  assert.deepEqual(paths, ["/api/v1/operations/op_one"]);
});

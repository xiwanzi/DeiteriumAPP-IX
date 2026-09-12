import { test } from "node:test";
import assert from "node:assert/strict";
import { redactErasedAccounts } from "./account-erasure.js";
import { DeuteriumClient } from "./api.js";

test("erasure preserves financial facts and the other party while masking delayed identity data", () => {
  const old = { amount: "19999.00", seller: { playerRef: "erased", displayName: "Old", contactQq: "12345", avatar: { assetId: "avatar" } }, buyer: { playerRef: "new", displayName: "Old" } };
  const view = redactErasedAccounts(old, new Set(["erased"]));
  assert.equal(view.amount, old.amount);
  assert.equal(view.seller.displayName, "已注销用户"); assert.equal(view.seller.contactQq, ""); assert.equal(view.seller.avatar, null);
  assert.deepEqual(view.buyer, old.buyer); assert.equal(old.seller.displayName, "Old");
});
test("erasure removes copied forward content without changing unrelated text", () => {
  const view = redactErasedAccounts({ content: "private", forwarded: { content: "private", availability: "AVAILABLE", sender: { playerRef: "erased", gameId: "Old" } }, reply: { content: "keep", sender: { playerRef: "live" } } }, new Set(["erased"]));
  assert.equal(view.content, "原消息不可见"); assert.equal(view.forwarded.content, ""); assert.equal(view.forwarded.availability, "UNAVAILABLE"); assert.equal(view.reply.content, "keep");
});
test("deletion synchronization follows every page and sanitizes later responses", async () => {
  const observed = [];
  const client = new DeuteriumClient({ fetchImpl: async (path) => {
    const data = path.includes("after=0") ? { items: [{ sequence: 1, playerRef: "old" }], cursor: 1, hasMore: true } : path.includes("after=1") ? { items: [{ sequence: 2, playerRef: "other" }], cursor: 2, hasMore: false } : { playerRef: "old", gameId: "Old", qq: "12345" };
    return new Response(JSON.stringify({ data }), { status: 200, headers: { "content-type": "application/json" } });
  } });
  const unsubscribe = client.onAccountDeletions((refs) => observed.push(...refs));
  await client.syncAccountDeletions(); unsubscribe();
  assert.deepEqual(observed, ["old", "other"]); assert.equal(client.deletionCursor, 2);
  assert.equal((await client.profile("old")).data.gameId, "已注销用户");
});

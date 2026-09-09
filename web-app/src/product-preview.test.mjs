import test from "node:test";
import assert from "node:assert/strict";
import { centeredCrop, productImageFrame, mailContentError } from "./product-preview.js";
import { normalizeMessage } from "./api.js";

test("App previews reflect fixed Android dp frames at multiple phone widths", () => {
  assert.deepEqual(productImageFrame("detail", 393), [353, 260]);
  assert.deepEqual(productImageFrame("grid", 360), [154, 170]);
  assert.deepEqual(productImageFrame("grid", 430), [189, 170]);
  assert.deepEqual(productImageFrame("poster", 430), [308, 405]);
  assert.deepEqual(productImageFrame("bag"), [80, 100]);
});
test("center crop reports source pixels, symmetric loss, and retained area", () => {
  assert.deepEqual(centeredCrop(1200, 800, 400, 400), { x: 200, y: 0, width: 800, height: 800, retained: 2 / 3 });
  const portrait = centeredCrop(600, 1200, 300, 200);
  assert.equal(portrait.width, 600);
  assert.equal(portrait.height, 400);
  assert.equal(portrait.y, 400);
  assert.equal(centeredCrop(0, 1, 3, 4), null);
  assert.equal(centeredCrop(1, 1, Infinity, 4), null);
});
test("game mail validates Core UTF-8 limits, plain text and blank legacy defaults", () => {
  assert.equal(mailContentError("", ""), "");
  assert.equal(mailContentError("中".repeat(80), "中".repeat(1000)), "");
  assert.equal(mailContentError("你好", "第一行\n第二行"), "");
  assert.match(mailContentError("🙂".repeat(61), ""), /标题过长/);
  assert.match(mailContentError("", "🙂".repeat(751)), /正文过长/);
  assert.match(mailContentError("第一行\n第二行", ""), /控制字符/);
  assert.match(mailContentError("", "正文\t内容"), /控制字符/);
});
test("public and direct message normalization retains signed avatar identity", () => {
  const sender = { playerRef: "player_one", gameId: "青梧", avatar: { assetId: "avatar_one", url: "https://assets.example/avatar?signature=short-lived" } };
  for (const conversationId of [undefined, "conversation_one"]) {
    const message = normalizeMessage({ messageId: "message_one", conversationId, sender, content: "你好", sentAt: "2026-09-09T01:00:00Z" }, { playerRef: "player_two" });
    assert.deepEqual(message.senderProfile, sender);
    assert.equal(message.sender, "player_one");
  }
});

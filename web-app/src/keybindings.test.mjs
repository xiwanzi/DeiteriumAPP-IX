import test from "node:test";
import assert from "node:assert/strict";
import { keyClashes, keyName, keyStatus } from "./keybindings.js";

test("key display supports modified, unbound, mouse and scan codes", () => {
  assert.equal(keyName("key.keyboard.enter:ALT"), "Alt + Enter");
  assert.equal(keyName("key.mouse.4"), "鼠标 4");
  assert.equal(keyName("scancode.123"), "扫描码 123");
  assert.equal(keyName("key.keyboard.unknown"), "未绑定");
});
test("key conflicts follow scopes and modifiers", () => {
  const rows = {
    one: { label: "聊天", value: "key.keyboard.t", context: "game" },
    two: { label: "轮盘", value: "key.keyboard.g:ALT", context: "game" },
    gui: { label: "菜单", value: "key.keyboard.g", context: "gui" },
  };
  assert.deepEqual(keyClashes(rows, "one", "key.keyboard.g"), ["轮盘"]);
  assert.deepEqual(keyClashes(rows, "one", "key.keyboard.g:SHIFT"), []);
  assert.deepEqual(keyClashes(rows, "one", "key.keyboard.left.alt"), ["轮盘"]);
  assert.deepEqual(keyClashes(rows, "one", "key.keyboard.unknown"), []);
});
test("the administrator sees acknowledgement separately from the requested binding", () => {
  assert.equal(keyStatus({ value: "A", observed: "A" }), "已生效");
  assert.equal(keyStatus({ value: "B", observed: "A" }), "待生效");
  assert.equal(
    keyStatus({ value: "B", observed: "A", missing: true }),
    "当前不可用",
  );
  assert.equal(
    keyStatus({ value: "B", observed: "A", conflict: true }),
    "修改冲突",
  );
});

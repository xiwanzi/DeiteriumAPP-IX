import test from "node:test";
import assert from "node:assert/strict";
import { productInventory } from "./productInventory.js";

test("unlimited products ignore the draft zero stock but retain per-order limits", () => {
  assert.deepEqual(productInventory({ availableStock: null, content: { inventoryPolicy: "UNLIMITED", stock: 0, limitPerOrder: 64 } }), { unlimited: true, stock: Infinity, max: 64 });
});
test("live finite stock zero overrides nonzero draft stock", () => {
  assert.equal(productInventory({ availableStock: 0, content: { inventoryPolicy: "FINITE", stock: 100, limitPerOrder: 64 } }).max, 0);
  assert.equal(productInventory({ availableStock: 3, content: { inventoryPolicy: "FINITE", stock: 100, limitPerOrder: 64 } }).max, 3);
});
test("unknown finite stock fails closed and market products never become unlimited", () => {
  assert.equal(productInventory({ content: { inventoryPolicy: "FINITE" } }).max, 0);
  assert.equal(productInventory({ stock: 0, inventoryPolicy: "UNLIMITED" }, false).max, 0);
});

import test from "node:test";
import assert from "node:assert/strict";
import {
  initialState,
  transition as run,
  amount,
  isBanned,
  visibleListings,
  visibleCommissions,
  dateKey,
} from "./domain.js";

test("bill dates use the same local calendar day as displayed timestamps", () => {
  const local = new Date(2026, 8, 8, 0, 15);
  assert.equal(dateKey(local.toISOString()), "2026-09-08");
});

test("amounts use exact cents and reject invalid or excessive values", () => {
  assert.equal(amount("0.01"), 1);
  assert.equal(amount("9999999.99"), 999999999);
  for (const v of ["-1", "0", "1.001", "1e3", "10000000", "NaN"])
    assert.throws(() => amount(v));
});
test("insufficient funds leave stock, balance and orders unchanged", () => {
  const s = initialState();
  s.balances.mori = 1;
  const copy = structuredClone(s);
  assert.throws(
    () => run(s, { type: "BUY", id: "l1", key: "purchase-1" }),
    /不足/,
  );
  assert.deepEqual(s, copy);
});
test("same purchase key does not charge twice and orders keep an immutable snapshot", () => {
  const s = initialState();
  const next = run(s, { type: "BUY", id: "l1", key: "purchase-1" });
  const twice = run(next, { type: "BUY", id: "l1", key: "purchase-1" });
  assert.deepEqual(next, twice);
  assert.equal(next.orders.length, 1);
  assert.equal(next.balances.mori, s.balances.mori - 12800);
  next.listings[0].title = "后来修改的标题";
  assert.notEqual(next.orders[0].snapshot.title, next.listings[0].title);
});
test("own purchases and sold-out listings cannot be bought", () => {
  let s = initialState();
  assert.throws(() => run(s, { type: "BUY", id: "l6" }), /自己/);
  s = run(s, { type: "BUY", id: "l2" });
  assert.equal(
    visibleListings(s).some((p) => p.id === "l2"),
    false,
  );
  assert.throws(() => run(s, { type: "BUY", id: "l2" }), /售罄/);
});
test("cart enforces inventory and checkout atomically consumes all lines once", () => {
  let s = initialState();
  s = run(s, { type: "CART", id: "iphone", delta: 1 });
  s = run(s, { type: "CART", id: "ipad", delta: 1 });
  const before = s.balances.mori;
  s = run(s, { type: "CHECKOUT", key: "checkout-1" });
  assert.equal(s.balances.mori, before - 899900 - 479900);
  assert.equal(s.products.find((p) => p.id === "iphone").stock, 11);
  assert.equal(s.orders[0].lines.length, 2);
  assert.equal(Object.keys(s.cart.mori).length, 0);
  assert.deepEqual(run(s, { type: "CHECKOUT", key: "checkout-1" }), s);
});
test("commission accept removes it from public view and rejects a second worker", () => {
  let s = run(initialState(), { type: "ACCEPT", id: "c1" });
  assert.equal(
    visibleCommissions(s).some((c) => c.id === "c1"),
    false,
  );
  s = run(s, { type: "LOGIN", user: "aster" });
  assert.throws(() => run(s, { type: "ACCEPT", id: "c1" }), /接取/);
});
test("commission needs work completion and owner confirmation; reward only releases once", () => {
  let s = run(initialState(), { type: "ACCEPT", id: "c1" });
  s = run(s, { type: "LOGIN", user: "luna" });
  assert.throws(() => run(s, { type: "COMMISSION_CONFIRM", id: "c1" }), /验收/);
  s = run(s, { type: "LOGIN", user: "mori" });
  s = run(s, { type: "COMMISSION_COMPLETE", id: "c1" });
  s = run(s, { type: "LOGIN", user: "luna" });
  const before = s.balances.mori;
  s = run(s, { type: "COMMISSION_CONFIRM", id: "c1" });
  assert.equal(s.balances.mori, before + 80000);
  assert.throws(() => run(s, { type: "COMMISSION_CONFIRM", id: "c1" }));
});
test("prepay commission can cancel once while open, returning the original amount", () => {
  let s = initialState();
  const before = s.balances.mori;
  s = run(s, {
    type: "COMMISSION_PUBLISH",
    data: {
      title: "新庭院",
      description: "设计一座花园",
      location: "主城",
      hours: 48,
      price: "250.30",
    },
  });
  const cid = s.commissions[0].id;
  assert.equal(s.balances.mori, before - 25030);
  s = run(s, { type: "COMMISSION_CANCEL", id: cid });
  assert.equal(s.balances.mori, before);
  assert.throws(() => run(s, { type: "COMMISSION_CANCEL", id: cid }));
});
test("transfers conserve available balances and refuse the same recipient as sender", () => {
  let s = initialState();
  const total = Object.values(s.balances).reduce((a, b) => a + b, 0);
  s = run(s, { type: "TRANSFER", to: "luna", amount: "500.01" });
  assert.equal(
    Object.values(s.balances).reduce((a, b) => a + b, 0),
    total,
  );
  assert.throws(() => run(s, { type: "TRANSFER", to: "mori", amount: "1" }));
});
test("idempotency is account/action scoped and rejects changed transfer content", () => {
  let s = run(initialState(), {
    type: "TRANSFER",
    to: "luna",
    amount: "100",
    key: "same",
  });
  assert.throws(
    () => run(s, { type: "TRANSFER", to: "luna", amount: "200", key: "same" }),
    /幂等/,
  );
  s = run(s, { type: "LOGIN", user: "aster" });
  const before = s.balances.aster;
  s = run(s, { type: "TRANSFER", to: "luna", amount: "100", key: "same" });
  assert.equal(s.balances.aster, before - 10000);
});
test("admin ban blocks login and old sessions without deleting ongoing transactions", () => {
  let s = run(initialState(), { type: "BUY", id: "l1" });
  s = run(s, { type: "BAN", player: "oak", reason: "交易争议", days: 7 });
  assert.ok(isBanned(s, "oak"));
  assert.equal(s.orders.length, 1);
  assert.throws(() => run(s, { type: "LOGIN", user: "oak" }), /封禁/);
  const old = { ...s, user: "oak" };
  assert.throws(
    () => run(old, { type: "MESSAGE", channel: "public", text: "旧会话" }),
    /不可用/,
  );
  assert.equal(
    visibleListings(s).some((p) => p.owner === "oak"),
    false,
  );
  s = run(s, { type: "UNBAN", id: s.bans[0].id, reason: "核实解除" });
  assert.ok(!isBanned(s, "oak"));
  assert.equal(s.audit.length, 2);
});
test("regular players cannot access admin actions; self and admin bans rejected", () => {
  let s = initialState();
  assert.throws(() =>
    run(s, { type: "BAN", player: "mori", reason: "测试", days: 1 }),
  );
  s = run(s, { type: "LOGIN", user: "luna" });
  assert.throws(
    () => run(s, { type: "BAN", player: "oak", reason: "测试", days: 1 }),
    /权限/,
  );
  assert.throws(
    () =>
      run(s, {
        type: "PRODUCT_SAVE",
        id: "iphone",
        title: "任意",
        price: "1",
        stock: 9,
      }),
    /权限/,
  );
});
test("favorites and direct message sending are scoped to the current account", () => {
  let s = run(initialState(), { type: "FAVORITE", id: "l1" });
  assert.deepEqual(s.favorites, ["mori:l1"]);
  s = run(s, { type: "LOGIN", user: "aster" });
  s = run(s, { type: "FAVORITE", id: "l1" });
  assert.deepEqual(s.favorites, ["mori:l1", "aster:l1"]);
  assert.throws(
    () =>
      run(s, { type: "MESSAGE", channel: "dm:luna:mori", text: "不应发送" }),
    /无权/,
  );
});
test("new official products are immediately visible while existing order prices stay fixed", () => {
  let s = run(initialState(), { type: "CART", id: "ipad", delta: 1 });
  s = run(s, { type: "CHECKOUT" });
  s = run(s, {
    type: "PRODUCT_SAVE",
    id: "ipad",
    title: "新价格",
    price: "100",
    stock: 10,
  });
  assert.equal(s.orders[0].lines[0].price, 479900);
  s = run(s, {
    type: "PRODUCT_CREATE",
    data: {
      title: "官方新商品",
      subtitle: "新到好物",
      price: "200",
      stock: 10,
      image: "store_ipad.webp",
      brand: "Apple",
    },
  });
  assert.equal(s.products[0].title, "官方新商品");
  assert.equal(s.products[0].price, 20000);
});

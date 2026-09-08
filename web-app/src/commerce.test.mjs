import test from "node:test";
import assert from "node:assert/strict";
import { initialState, transition } from "./domain.js";
import { upgradeDemo } from "./demo-v2.js";
import {
  transactionRows,
  filterTransactions,
  salesSummary,
  allPlayerSales,
  exportTransactions,
} from "./commerce.js";

test("seller totals exclude unpaid orders, subtract partial/full refunds and keep held money separate", () => {
  const s = upgradeDemo(initialState()),
    rows = transactionRows(s);
  assert.deepEqual(salesSummary(rows, "aster"), {
    gross: 180000,
    refund: 0,
    sold: 180000,
    settled: 180000,
    held: 0,
    orderCount: 1,
  });
  assert.deepEqual(salesSummary(rows, "luna"), {
    gross: 77600,
    refund: 19600,
    sold: 58000,
    settled: 58000,
    held: 0,
    orderCount: 2,
  });
  assert.equal(salesSummary(rows, "oak").held, 51200);
  assert.equal(
    allPlayerSales(s).reduce((n, r) => n + r.sold, 0),
    325200,
  );
});
test("each player drilldown includes only that player and preserves role/channel/date filters", () => {
  const rows = transactionRows(upgradeDemo(initialState()));
  const seller = filterTransactions(rows, {
    player: "luna",
    role: "卖出 / 收入",
    channel: "玩家市场",
  });
  assert.equal(seller.length, 2);
  assert.ok(seller.every((t) => t.seller === "luna"));
  const buyer = filterTransactions(rows, {
    player: "luna",
    role: "买入 / 支出",
    channel: "玩家市场",
  });
  assert.equal(buyer.length, 1);
  assert.equal(filterTransactions(rows, { query: "DT202609070002" }).length, 1);
  assert.equal(filterTransactions(rows, { from: "2099-01-01" }).length, 0);
  assert.equal(
    filterTransactions(rows, { status: "已退款", channel: "玩家市场" }).length,
    1,
  );
});
test("transfers appear as separate ledger entries without increasing product sales", () => {
  let s = upgradeDemo(initialState());
  const before = salesSummary(transactionRows(s), "luna").sold;
  s = transition(s, {
    type: "TRANSFER",
    to: "luna",
    amount: "200.01",
    note: "庭院材料",
  });
  const entries = filterTransactions(transactionRows(s), {
    player: "luna",
    channel: "转账",
  });
  assert.equal(entries.length, 1);
  assert.equal(entries[0].paidAmount, 20001);
  assert.equal(salesSummary(transactionRows(s), "luna").sold, before);
});
test("CSV export preserves filtered rows and neutralizes spreadsheet formulas", () => {
  const rows = filterTransactions(
    transactionRows(upgradeDemo(initialState())),
    { query: "DT202609070002" },
  );
  rows[0].title = '=HYPERLINK("https://invalid.test")';
  const csv = exportTransactions(rows);
  assert.ok(csv.startsWith("\ufeff"));
  assert.equal(csv.split("\r\n").length, 2);
  assert.ok(csv.includes("\"'=HYPERLINK"));
  assert.ok(csv.includes('"退款"'));
  assert.ok(csv.includes('"100.00"'));
});
test("demo migration preserves previous orders and only adds reference history once", () => {
  const old = initialState();
  old.orders.push({ id: "previous-user-order" });
  const next = upgradeDemo(old);
  assert.ok(next.orders.some((o) => o.id === "previous-user-order"));
  assert.equal(upgradeDemo(next).orders.length, next.orders.length);
});
test("announcement edits remain private drafts until explicitly published again", () => {
  let s = upgradeDemo(initialState());
  s = transition(s, {
    type: "ANNOUNCEMENT_CREATE",
    data: {
      title: "新公告",
      summary: "一份摘要",
      body: "原始正文",
      pinned: false,
      cover: "street-daylight.png",
    },
  });
  const id = s.announcements[0].id;
  assert.equal(s.announcements[0].published, null);
  s = transition(s, { type: "ANNOUNCEMENT_PUBLISH", id, expectedVersion: 1 });
  assert.equal(s.announcements[0].published.body, "原始正文");
  s = transition(s, {
    type: "ANNOUNCEMENT_SAVE",
    id,
    expectedVersion: 2,
    data: {
      title: "新标题",
      summary: "一份摘要",
      body: "尚未公开的修改",
      pinned: true,
      cover: "street-daylight.png",
    },
  });
  assert.equal(s.announcements[0].published.body, "原始正文");
  assert.equal(s.announcements[0].hasDraftChanges, true);
  assert.throws(
    () =>
      transition(s, { type: "ANNOUNCEMENT_PUBLISH", id, expectedVersion: 2 }),
    /更新/,
  );
  s = transition(s, { type: "ANNOUNCEMENT_PUBLISH", id, expectedVersion: 3 });
  assert.equal(s.announcements[0].published.body, "尚未公开的修改");
  s = transition(s, { type: "ANNOUNCEMENT_WITHDRAW", id, expectedVersion: 4 });
  assert.equal(s.announcements[0].status, "WITHDRAWN");
  assert.ok(s.announcements[0].published);
});
test("ordinary demo players cannot create, publish or withdraw official announcements", () => {
  let s = upgradeDemo(initialState());
  s = transition(s, { type: "LOGIN", user: "luna" });
  assert.throws(
    () =>
      transition(s, {
        type: "ANNOUNCEMENT_CREATE",
        data: { title: "私自发布", body: "正文", summary: "摘要" },
      }),
    /权限/,
  );
  assert.throws(
    () =>
      transition(s, {
        type: "ANNOUNCEMENT_WITHDRAW",
        id: "a1",
        expectedVersion: 1,
      }),
    /权限/,
  );
});
test("chat drafts and read positions are scoped to user and channel", () => {
  let s = transition(initialState(), {
    type: "CHAT_DRAFT",
    channel: "dm:luna:mori",
    text: "尚未发出的消息",
  });
  s = transition(s, { type: "CHAT_READ", channel: "public", id: "m3" });
  assert.equal(s.chatDrafts["mori:dm:luna:mori"], "尚未发出的消息");
  assert.equal(s.readPositions["mori:public"], "m3");
  assert.equal(s.readPositions["luna:public"], undefined);
});

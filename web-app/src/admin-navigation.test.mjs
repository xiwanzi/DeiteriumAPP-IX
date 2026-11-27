import test from "node:test";
import assert from "node:assert/strict";
import {
  OVERVIEW_NAV_ITEM,
  adminNavigation,
  adminNavigationGroups,
  findAdminNavItem,
  matchAdminSearch,
} from "./admin-navigation.js";

test("platform admin receives all 18 modules grouped into 5 operational domains plus overview", () => {
  const { groups, flatItems, hasOverview, supportOnly } = adminNavigationGroups(["platform.admin"]);
  assert.equal(supportOnly, false);
  assert.equal(hasOverview, true);
  assert.equal(groups.length, 5);
  assert.equal(flatItems.length, 18);
  assert.equal(adminNavigation(["platform.admin"]).length, 18);
  // Every flat adminNavigation path appears in grouped navigation
  const groupedPaths = new Set(flatItems.map((i) => i.to));
  for (const [to] of adminNavigation(["platform.admin"])) {
    assert.equal(groupedPaths.has(to), true, `Missing ${to} in grouped navigation`);
  }
});

test("support-only agent sees only support ticket desk without overview clutter", () => {
  const { groups, flatItems, hasOverview, supportOnly } = adminNavigationGroups(["support.manage"]);
  assert.equal(supportOnly, true);
  assert.equal(hasOverview, false);
  assert.equal(groups.length, 1);
  assert.equal(flatItems.length, 1);
  assert.equal(flatItems[0].to, "/admin?section=support");
});

test("command palette search supports Chinese labels, English keys, and pinyin initials", () => {
  const { flatItems } = adminNavigationGroups(["platform.admin"]);
  const whitelist = flatItems.find((i) => i.section === "whitelist");
  const supply = flatItems.find((i) => i.section === "supply");
  const titles = flatItems.find((i) => i.section === "titles");

  assert.equal(matchAdminSearch(whitelist, "bmd"), true);
  assert.equal(matchAdminSearch(whitelist, "白名单"), true);
  assert.equal(matchAdminSearch(whitelist, "whitelist"), true);
  assert.equal(matchAdminSearch(supply, "卡池"), true);
  assert.equal(matchAdminSearch(supply, "gacha"), true);
  assert.equal(matchAdminSearch(titles, "tx"), true);
  assert.equal(matchAdminSearch(titles, "不存在的功能"), false);
});

test("findAdminNavItem resolves overview and grouped items accurately", () => {
  const overview = findAdminNavItem(OVERVIEW_NAV_ITEM.to, ["platform.admin"]);
  assert.equal(overview.item?.section, "overview");
  assert.equal(overview.group, null);

  const core = findAdminNavItem("/admin?section=core", ["platform.admin"]);
  assert.equal(core.item?.label, "Core 管理");
  assert.equal(core.group?.label, "系统与审计");
});

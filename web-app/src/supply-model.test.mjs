import test from "node:test";
import assert from "node:assert/strict";
import { creditMinor, creditText, csvCell, isRewardBound, isSamplePool, newPool, poolIssues, rewardFromItem, ruleText, scheduleInput, scheduleISO, unboundRewardCount } from "./supply-model.js";

test("supply rules disclose fixed 0.5% and 1.2% without increasing odds", () => {
  assert.match(ruleText("LIMITED", "80"), /0\.5%/);
  assert.match(ruleText("LIMITED", "120"), /1\.2%.*100～120/);
  assert.doesNotMatch(ruleText("LIMITED", "120"), /递增|1\.5%/);
});
test("supply schedules always round trip in UTC+8", () => {
  assert.equal(scheduleISO("2026-11-17T08:00"), "2026-11-17T00:00:00.000Z");
  assert.equal(scheduleInput("2026-11-17T00:00:00Z"), "2026-11-17T08:00");
});
test("credit exchange prices use exact minor units", () => {
  assert.equal(creditMinor("0.29"), 29); assert.equal(creditMinor("100"), 10000); assert.equal(creditText(29), "0.29");
  for (const value of ["-1", "1.001", "1e3", "", "1000001"]) assert.throws(() => creditMinor(value));
});
test("pool publication checklist catches missing content and invalid schedule", () => {
  const d = newPool().definition; assert.ok(poolIssues(d).length >= 4);
  d.startsAt = "2026-11-17T01:00:00Z"; d.endsAt = "2026-11-17T00:00:00Z";
  assert.ok(poolIssues(d).includes("结束时间必须晚于开放时间"));
});
test("CSV export does not execute spreadsheet formulas", () => {
  assert.equal(csvCell("=1+1"), '"\'=1+1"'); assert.equal(csvCell('a"b'), '"a""b"');
});
test("sample pool allows unbound preview publishing while tracking unbound count", () => {
  assert.equal(isSamplePool("sp_sample_up_191"), true);
  assert.equal(isSamplePool("sp_custom_1"), false);
  const d = newPool("STANDARD").definition;
  d.name = "常驻补给"; d.title = "雨夜鸣金";
  d.art.hero = "builtin-hero-standard"; d.art.rail = "builtin-rail-standard";
  d.rewards = [
    { key: "r6", name: "六星", stars: 6, up: false, weight: 100, modelKey: "gilded-rain", art: {} },
    { key: "r4", name: "四星", stars: 4, up: false, weight: 100, modelKey: "tacz-ak47", art: {} },
  ];
  assert.equal(unboundRewardCount(d), 2);
  assert.equal(isRewardBound(d.rewards[0]), false);
  assert.equal(poolIssues(d, "sp_sample_standard").length, 0);
  assert.ok(poolIssues(d, "sp_custom").some(msg => msg.includes("物品库版本")));
  d.sections = [{ title: "", body: "missing title" }];
  assert.ok(poolIssues(d, "sp_sample_standard").some(msg => msg.includes("自定义详情分段")));
});
test("rewardFromItem extracts TaCZ gunId and displayId from imported stackData", () => {
  const item = { itemRef: "amiya:gacha_1", revision: 2, displayName: "AWM", payloadSha256: "a".repeat(64), inventoryDomain: "survival", compatibleServerIds: ["amiya"] };
  const built = rewardFromItem(item, { stars: 5, weight: 250, stackData: '{id:"tacz:modern_kinetic_gun",components:{"minecraft:custom_data":{GunId:"tacz:ai_awp",GunDisplayId:"tacz:ai_awp_default"}}}' });
  assert.equal(built.stars, 5);
  assert.equal(built.weight, 250);
  assert.equal(built.gunId, "tacz:ai_awp");
  assert.equal(built.displayId, "tacz:ai_awp_default");
  assert.equal(built.modelKey, "tacz-ai_awp");
});

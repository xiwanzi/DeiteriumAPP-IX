import test from "node:test";
import assert from "node:assert/strict";
import { creditMinor, creditText, csvCell, newPool, poolIssues, ruleText, scheduleInput, scheduleISO } from "./supply-model.js";

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

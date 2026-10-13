import test from "node:test";
import assert from "node:assert/strict";
import { applyPlayerNames, rememberPlayerNames } from "./player-names.js";
import { redactErasedAccounts } from "./account-erasure.js";

test("same-UUID rename changes structured labels, not money, nicknames, or source snapshots", () => {
  const names = new Map(); rememberPlayerNames(names, [{ sequence: 2, uuid: "uuid-a", playerRef: "player_a", gameId: "NewName" }]);
  const value = { amount: "9007199254740993.25", buyer: { kind: "PLAYER", playerRef: "player_a", displayName: "OldName" }, user: { playerRef: "player_a", gameId: "OldName", displayName: "中文昵称" }, message: { sender: "player_a", senderName: "OldName", text: "OldName is part of this message" }, snapshot: { sender: { playerRef: "player_a", gameId: "OldName" } } };
  const copy = structuredClone(value), result = applyPlayerNames(value, names);
  assert.deepEqual(value, copy); assert.equal(result.buyer.displayName, "NewName"); assert.equal(result.user.gameId, "NewName"); assert.equal(result.user.displayName, "中文昵称");
  assert.equal(result.amount, value.amount); assert.equal(result.message.senderName, "NewName"); assert.equal(result.message.text, value.message.text); assert.deepEqual(result.snapshot, value.snapshot);
});
test("late deltas cannot roll back a name and reused spelling never redirects another identity", () => {
  const names = new Map();rememberPlayerNames(names,[{sequence:5,uuid:"uuid-a",playerRef:"player_a",gameId:"NewName"}]);
  assert.equal(rememberPlayerNames(names,[{sequence:3,uuid:"uuid-a",playerRef:"player_a",gameId:"OldName"}]),false);
  const values=[{playerRef:"player_a",gameId:"OldName"},{playerRef:"player_b",gameId:"OldName"}];
  assert.deepEqual(applyPlayerNames(values,names),[{playerRef:"player_a",gameId:"NewName"},{playerRef:"player_b",gameId:"OldName"}]);
  assert.equal(redactErasedAccounts(applyPlayerNames(values[0],names),new Set(["player_a"])).gameId,"已注销用户");
});

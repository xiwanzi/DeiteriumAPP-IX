import test from "node:test";
import assert from "node:assert/strict";
import { launcherPublication, matchesLauncherPublication } from "./launcher-publication.js";

const payload = { product: "DLauncher", platform: "windows-x64", version: "0.6.10", versionCode: 6010, updaterProtocol: 1,
  notes: "fixture", publishedAt: "2026-10-15T00:00:00Z", package: { url: "https://fixture.test/package.zip", size: 100, sha256: "a".repeat(64) } };
const signed = p => ({ payload: Buffer.from(JSON.stringify(p)).toString("base64"), signature: "fixture" });
const legacy = signed(payload);
const incremental = signed({ ...payload, updaterProtocol: 2, manifest: { url: "https://fixture.test/manifest.json", size: 50, sha256: "b".repeat(64) } });

test("old publication files remain selectable", () => {
  const selected = launcherPublication(legacy);
  assert.equal(selected.release.version, "0.6.10");
  assert.equal(selected.incremental, null);
  assert.ok(matchesLauncherPublication({ release: legacy }, selected));
});

test("incremental publication keeps both envelopes for unknown-result recovery", () => {
  const selected = launcherPublication({ schemaVersion: 2, legacy, incremental });
  assert.ok(matchesLauncherPublication({ release: legacy, incrementalRelease: incremental }, selected));
  assert.equal(matchesLauncherPublication({ release: legacy }, selected), false);
  assert.equal(matchesLauncherPublication({ release: legacy, incrementalRelease: { ...incremental, signature: "different" } }, selected), false);
});

test("mismatched or missing legacy compatibility cannot be selected", () => {
  assert.throws(() => launcherPublication({ schemaVersion: 2, legacy, incremental: signed({ ...payload, updaterProtocol: 2, version: "0.6.11", manifest: {} }) }));
  assert.throws(() => launcherPublication(incremental));
  assert.throws(() => launcherPublication({ schemaVersion: 2, legacy }));
});

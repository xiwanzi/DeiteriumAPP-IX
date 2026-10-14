export function unpackLauncherRelease(envelope) {
  if (!envelope || typeof envelope.payload !== "string" || typeof envelope.signature !== "string") throw new Error("请选择构建生成的版本清单。");
  return JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(envelope.payload), c => c.charCodeAt(0))));
}

export function launcherPublication(envelope) {
  const bundled = envelope?.schemaVersion === 2;
  const legacy = bundled ? envelope.legacy : envelope;
  const incremental = bundled ? envelope.incremental : null;
  const release = unpackLauncherRelease(legacy);
  if (release.updaterProtocol !== 1) throw new Error("版本文件缺少旧版兼容更新包。");
  if (bundled) {
    const next = unpackLauncherRelease(incremental);
    if (next.updaterProtocol !== 2 || !next.manifest ||
        ["product", "platform", "version", "versionCode", "notes", "publishedAt"].some(key => next[key] !== release[key]) ||
        ["url", "size", "sha256"].some(key => next.package?.[key] !== release.package?.[key])) throw new Error("增量清单与兼容更新包不是同一版本。");
  }
  return { envelope, legacy, incremental, release };
}

export function matchesLauncherPublication(current, selected) {
  const same = (a, b) => !!a && !!b && a.payload === b.payload && a.signature === b.signature;
  return same(current?.release, selected.legacy) && (!selected.incremental || same(current?.incrementalRelease, selected.incremental));
}

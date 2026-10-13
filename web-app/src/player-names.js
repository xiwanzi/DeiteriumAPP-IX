const archived = new Set(["snapshot", "snapshotJson", "raw", "signedPayload", "content", "body", "settings", "configuration"]);

// Only stable identity references select a name. Never replace free text or use
// an old spelling to guess who owns a conversation or financial record.
export function applyPlayerNames(value, names) {
  if (!value || typeof value !== "object" || !names.size) return value;
  if (Array.isArray(value)) return value.map(item => applyPlayerNames(item, names));
  const result = Object.fromEntries(Object.entries(value).map(([key, item]) => [key, archived.has(key) ? item : applyPlayerNames(item, names)]));
  const ref = [value.playerRef, value.playerUuid, value.serverUuid, value.profileUuid, value.uuid, value.senderRef, typeof value.sender === "string" ? value.sender : null, value.id].find(ref => typeof ref === "string" && names.has(ref));
  const name = names.get(ref)?.gameId;
  if (name && !value.deleted) {
    for (const key of ["gameId", "senderName", "playerName"]) if (key in result) result[key] = name;
    if (value.id === ref && "name" in result) result.name = name;
    if (value.kind === "PLAYER") { result.gameId = name; if ("displayName" in result) result.displayName = name; }
  }
  for (const role of ["buyer", "seller", "owner", "worker", "author", "recipient"]) {
    const current = names.get(value[`${role}Ref`])?.gameId;
    if (!current) continue;
    if (typeof result[`${role}Name`] === "string") result[`${role}Name`] = current;
    if (typeof result[role] === "string") result[role] = current;
  }
  return result;
}

export function rememberPlayerNames(names, changes) {
  let changed = false;
  for (const entry of changes) {
    if (!entry.playerRef || !entry.uuid || !/^[A-Za-z0-9_]{1,32}$/.test(entry.gameId) || !Number.isSafeInteger(entry.sequence)) continue;
    if ((names.get(entry.playerRef)?.sequence ?? -1) >= entry.sequence) continue;
    names.set(entry.playerRef, entry); names.set(entry.uuid, entry); changed = true;
  }
  return changed;
}

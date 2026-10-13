const viewChildren = ["user", "player", "profile", "otherPlayer", "sender", "senderProfile", "reply", "forwarded", "message", "lastMessage", "messages", "records", "items", "data", "buyer", "seller", "owner", "worker", "applicant", "respondent", "counterparty", "recipient", "author"];

// Only stable identity references select a name. Never replace free text or use
// an old spelling to guess who owns a conversation or financial record.
export function applyPlayerNames(value, names) {
  if (!value || typeof value !== "object" || !names.size) return value;
  if (Array.isArray(value)) { const next = value.map(item => applyPlayerNames(item, names)); return next.some((item,index) => item !== value[index]) ? next : value; }
  let result = value;
  const set = (key, item) => { if (result[key] === item) return; if (result === value) result = { ...value }; result[key] = item; };
  for (const key of viewChildren) if (value[key] && typeof value[key] === "object") set(key, applyPlayerNames(value[key], names));
  const ref = [value.playerRef, value.playerUuid, value.serverUuid, value.profileUuid, value.uuid, value.senderRef, typeof value.sender === "string" ? value.sender : null, value.id].find(ref => typeof ref === "string" && names.has(ref));
  const name = names.get(ref)?.gameId;
  if (name && !value.deleted) {
    for (const key of ["gameId", "senderName", "playerName"]) if (key in result) set(key, name);
    if (value.id === ref && "name" in result) set("name", name);
    if (value.kind === "PLAYER") { set("gameId", name); if ("displayName" in result) set("displayName", name); }
  }
  for (const role of ["buyer", "seller", "owner", "worker", "author", "recipient"]) {
    const current = names.get(value[`${role}Ref`])?.gameId;
    if (!current) continue;
    if (typeof result[`${role}Name`] === "string") set(`${role}Name`, current);
    if (typeof result[role] === "string") set(role, current);
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

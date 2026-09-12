export function redactErasedAccounts(value, refs) {
  if (!value || typeof value !== "object" || !refs.size) return value;
  if (Array.isArray(value)) return value.map((item) => redactErasedAccounts(item, refs));
  const result = Object.fromEntries(Object.entries(value).map(([key, item]) => [key, redactErasedAccounts(item, refs)]));
  if (refs.has(value.playerRef)) {
    for (const key of ["gameId", "displayName", "name"]) if (key in result) result[key] = "已注销用户";
    for (const key of ["qq", "bio", "contactQq"]) if (key in result) result[key] = "";
    if ("avatar" in result) result.avatar = null;
    result.deleted = true;
    if ("registered" in result) result.registered = false;
  }
  if ("availability" in value && refs.has(value.sender?.playerRef)) { result.availability = "UNAVAILABLE"; result.content = ""; }
  if (refs.has(value.forwarded?.sender?.playerRef)) result.content = "原消息不可见";
  return result;
}

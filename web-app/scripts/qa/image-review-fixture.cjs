async (page) => {
  const now = new Date().toISOString();
  const make = (id, name, score, scores, summary) => ({ id: id.repeat(64), sourceKind: "CHAT", sourceId: `upload_${id}`, mediaId: `media_${id}`, state: "FLAGGED", score, gameId: name, displayName: name === "Amiya" ? "阿米娅" : name, uploaderId: `user_${id}`, playerRef: `player_${id}`, purpose: "CHAT_MEDIA", uploadedAt: now, reviewedAt: now, policyVersion: "static-v1-20261012", model: "deepseek-flash", sizeBytes: 32450, result: { scores, summary, reasons: [summary, "此为隔离的界面验收数据，不来自真实玩家。"] }, events: [{ id: `msg_${id}`, senderId: `user_${id}`, gameId: name, displayName: name, channel: "world", sentAt: now }] });
  let items = [make("a", "Amiya", 89, { r18: 0, politics: 0, promotion: 89 }, "图片含有联系方式及站外导流文字，建议人工复核。"), make("b", "青梧", 76, { r18: 0, politics: 76, promotion: 0 }, "图片中出现明确的现实政治宣传内容。"), make("c", "LongPlayerName16", 51, { r18: 0, politics: 0, promotion: 51 }, "疑似含有群组邀请信息，建议核对上下文。")];
  await page.route("**/web-config.json", route => route.fulfill({ json: { mode: "connected", version: "2.0.37", configured: true } }));
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname, method = request.method();
    const reply = data => route.fulfill({ json: { data, page: { hasMore: false, nextCursor: null } } });
    if (path === "/api/v1/web/session") return reply({ user: { userId: "fixture-admin", playerRef: "player_admin", gameId: "ReviewAdmin", displayName: "管理员", gameVerified: true, permissions: ["platform.admin"] }, csrfToken: "fixture-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/account/deletions") return reply({ items: [], cursor: 0, hasMore: false });
    if (path === "/api/v1/admin/interventions/summary") return reply({ pending: 0 });
    if (path === "/api/v1/notifications/unread-count") return reply({ count: 0 });
    if (path === "/api/v1/notifications") return reply([]);
    if (path === "/api/v1/chat/conversations") return reply([]);
    if (path === "/api/v1/admin/image-review") return reply({ items, page: 0, hasMore: false, summary: { flagged: items.length, clear: 126, pending: 2, retrying: 0, skipped: 4, unavailable: 0, deleting: 0, lastScanAt: now, lastWorkAt: now, model: "deepseek-flash", modelLabel: "DeepSeek V4.1 Flash", threshold: 50 } });
    if (path.endsWith("/image")) return route.fulfill({ contentType: "image/svg+xml", body: `<svg xmlns="http://www.w3.org/2000/svg" width="800" height="520"><rect width="800" height="520" fill="#eef1f3"/><rect x="50" y="50" width="700" height="420" rx="18" fill="#dfe6e8"/><path d="M50 365L220 155L390 335L490 220L750 440H50" fill="#9fadb0"/><circle cx="600" cy="140" r="42" fill="#f8f6ed"/><rect x="110" y="360" width="580" height="70" rx="12" fill="#fff"/><text x="400" y="405" text-anchor="middle" fill="#333" font-size="24" font-family="sans-serif">图片审核 · 隔离验收样本</text></svg>` });
    if (path.startsWith("/api/v1/admin/image-review/")) {
      const id = path.split("/").pop(), item = items.find(item => item.id === id);
      if (method === "DELETE") { if (request.headers()["x-csrf-token"] !== "fixture-csrf") throw new Error("CSRF token missing"); items = items.filter(item => item.id !== id); return reply({ id, state: "DELETED" }); }
      if (item) return reply(item);
    }
    return reply({});
  });
  await page.setViewportSize({ width: 1440, height: 1050 });
  await page.goto("http://127.0.0.1:5266/admin?section=image-review");
  await page.getByRole("heading", { name: "图片审核", exact: true }).waitFor();
  return { mode: "isolated fixture; no production API requests", cards: await page.locator(".review-card").count() };
}

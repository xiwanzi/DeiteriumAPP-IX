async (page) => {
  const origin = "http://127.0.0.1:5219", passed = [], reads = [], errors = []; let permissions = ["audit.read"];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", (error) => errors.push(error.message)); await page.unroute("**/api/v1/**");
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request(), path = "/api/v1" + request.url().split("/api/v1")[1].split("?")[0], query = Object.fromEntries((request.url().split("?")[1] || "").split("&").filter(Boolean).map((pair) => pair.split("=").map(decodeURIComponent)));
    const json = (data, page = { nextCursor: null }) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ data, page }) });
    if (path === "/api/v1/web/session") return json({ user: { userId: "audit-admin", playerRef: "audit-admin-ref", gameId: "AuditAdmin", qq: "100001", permissions }, csrfToken: "audit-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/admin/audit-events") {
      reads.push({ method: request.method(), query });
      if (query.actorId === "nobody") return json([]);
      if (query.cursor) return json([{ eventId: "9007199254740993", actorId: "local-cli", action: "permission.grant", resourceId: "user-one:audit.read", createdAt: "2026-09-08T03:00:00Z" }]);
      return json([{ eventId: "9007199254740994", actorId: "actual-actor", action: "catalog.mutate", resourceId: "catalog:product", createdAt: "2026-09-08T03:01:00Z" }], { nextCursor: "frozen-query-cursor", hasMore: true });
    }
    return route.fulfill({ status: 404, contentType: "application/json", body: JSON.stringify({ error: { code: "NOT_FOUND", message: "未找到" } }) });
  });
  await page.goto(origin + "/admin"); await page.getByText("catalog.mutate", { exact: true }).waitFor();
  check(await page.getByRole("heading", { name: "管理审计", exact: true }).count() === 1, "audit.read exposes only its granted audit section");
  check(await page.getByRole("button", { name: "Core 管理", exact: true }).count() === 0, "audit privilege does not expose Core management tab");
  await page.getByRole("button", { name: "加载更多记录", exact: true }).click(); await page.getByText("permission.grant", { exact: true }).waitFor();
  check(reads[1].query.cursor === "frozen-query-cursor" && Object.keys(reads[1].query).sort().join(",") === "cursor,limit", "pagination preserves server frozen cursor without changing filters");
  check(await page.locator("tbody tr").count() === 2, "event IDs beyond JavaScript safe integer stay distinct");
  await page.getByRole("textbox", { name: "操作者 ID", exact: true }).fill("nobody"); await page.getByRole("button", { name: "查询记录", exact: true }).click(); await page.getByRole("heading", { name: "没有符合条件的记录", exact: true }).waitFor();
  check(reads.at(-1).query.actorId === "nobody" && !reads.at(-1).query.cursor, "new filter resets snapshot and renders truthful empty state");
  await page.getByLabel("起始时间", { exact: true }).fill("2025-01-01T12:00"); const count = reads.length; await page.getByRole("button", { name: "查询记录", exact: true }).click(); await page.getByRole("alert").filter({ hasText: "最多 90 天" }).waitFor();
  check(reads.length === count, "overlong audit interval is rejected before network access");
  await page.setViewportSize({ width: 390, height: 844 }); check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), "audit form fits mobile viewport");
  check(reads.every((read) => read.method === "GET"), "audit interface never performs a mutation");
  permissions = []; await page.reload(); await page.getByRole("heading", { name: "商店管理", exact: true }).waitFor();
  check(await page.getByRole("button", { name: "管理审计", exact: true }).count() === 0, "ordinary account does not receive audit entry");
  check(errors.length === 0, "audit interface has no JavaScript runtime exception");
  return { mode: "isolated audit contract responses; no real audit mutations", count: passed.length, passed, errors };
}

async (page) => {
  const passed = [], writes = [], origin = "http://127.0.0.1:5219", now = "2026-09-08T03:00:00Z"; let mode = "missing";
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  await page.unroute("**/api/v1/**"); await page.route("**/api/v1/**", async (route) => {
    const request = route.request(), path = "/api/v1" + request.url().split("/api/v1")[1].split("?")[0];
    const json = (data, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify({ data, page: { nextCursor: null } }) });
    const fail = (status, code) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify({ error: { code, message: code } }) });
    if (path === "/api/v1/web/session") return json({ user: { userId: "recovery-buyer", playerRef: "recovery-ref", gameId: "buyer", qq: "10001", permissions: [] }, csrfToken: "recovery-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/orders") return json([]);
    if (path.startsWith("/api/v1/operations/")) {
      if (mode === "network") return route.abort("connectionfailed");
      if (mode === "resource-missing") return json({ operationId: "known-operation", status: "UNKNOWN", resourceType: "ORDER", resourceId: "missing-resource" });
      return fail(404, "NOT_FOUND");
    }
    if (path === "/api/v1/orders/missing-resource") return fail(404, "NOT_FOUND");
    if (path === "/api/v1/store/orders") {
      writes.push(request.postDataJSON()); if (mode === "expired") return fail(409, "QUOTE_EXPIRED");
      return json({ operation: { operationId: "known-operation", status: "UNKNOWN", resourceType: "ORDER", resourceId: "order-created" }, order: { orderId: "order-created", fundsStatus: "UNKNOWN", pendingOperationId: "known-operation", amount: "10.00", createdAt: now } }, 202);
    }
    return fail(404, "NOT_FOUND");
  });
  const open = async (kind, alteredOrigin = origin) => {
    mode = kind; await page.goto(origin + "/orders");
    await page.evaluate(({ origin, kind }) => { sessionStorage.setItem("deuterium-business:recovery-buyer", JSON.stringify({ [kind]: { userId: "recovery-buyer", origin, kind: "STORE_PURCHASE", path: "/api/v1/store/orders", body: { clientRequestId: kind, quoteId: "original-quote", expectedQuoteVersion: 3 } } })); }, { origin: alteredOrigin, kind });
    await page.reload(); await page.getByRole("button", { name: "查看第 1 笔", exact: true }).click();
  };
  await open("missing"); await page.getByRole("button", { name: "重试原请求", exact: true }).waitFor(); await page.getByRole("button", { name: "重试原请求", exact: true }).click(); await page.getByText("资金结果待核对", { exact: true }).waitFor();
  check(writes.length === 1 && JSON.stringify(writes[0]) === '{"clientRequestId":"missing","quoteId":"original-quote","expectedQuoteVersion":3}', "explicit absent-operation retry keeps the exact original quote and request ID");
  await page.getByRole("button", { name: "刷新处理结果", exact: true }).click(); await page.getByRole("alert").filter({ hasText: "NOT_FOUND" }).waitFor();
  check(await page.getByRole("button", { name: "重试原请求", exact: true }).count() === 0, "a known operation returning 404 never enables another POST");
  check(await page.evaluate(() => Object.values(JSON.parse(sessionStorage.getItem("deuterium-business:recovery-buyer")))[0].operationId) === "known-operation", "authoritative operation identity is durable across browser reloads");
  await open("resource-missing"); await page.getByRole("alert").filter({ hasText: "NOT_FOUND" }).waitFor(); check(await page.getByRole("button", { name: "重试原请求", exact: true }).count() === 0, "a resource 404 after a valid operation is not proof of an absent payment");
  await open("network"); await page.getByRole("alert").waitFor(); check(await page.getByRole("button", { name: "重试原请求", exact: true }).count() === 0, "network uncertainty does not authorize original POST replay");
  await open("missing", "https://other-origin.test"); await page.getByRole("alert").waitFor(); check(await page.getByRole("button", { name: "重试原请求", exact: true }).count() === 0, "journal from another origin cannot replay even with a 404");
  await open("expired"); await page.getByRole("button", { name: "重试原请求", exact: true }).click(); await page.getByRole("heading", { name: "报价已过期", exact: true }).waitFor();
  check(await page.getByRole("button", { name: "重试原请求", exact: true }).count() === 0, "expired quote ends pending recovery and asks for a new confirmation");
  check(await page.evaluate(() => Object.keys(JSON.parse(sessionStorage.getItem("deuterium-business:recovery-buyer"))).length) === 0, "expired quote removes only its original pending record");
  check(writes.length === 2, "only the two explicitly selected missing-operation retries created POST requests");
  return { mode: "isolated payment-recovery contract; no real financial operation", count: passed.length, passed };
}

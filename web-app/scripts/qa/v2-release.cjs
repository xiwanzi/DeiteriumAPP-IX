async (page) => {
  const passed = [], errors = [], operations = [], base = "http://127.0.0.1:5219";
  const check = (value, name) => { if (!value) throw new Error(name); passed.push(name); };
  page.on("pageerror", error => errors.push(error.message));
  await page.unroute("**/api/v1/**");
  await page.evaluate(() => { localStorage.setItem("deuterium-web-theme", "light"); sessionStorage.removeItem("deuterium-transfer:qa-user"); });
  await page.goto(base + "/login");
  await page.getByRole("heading", { name: "暂时无法连接后端" }).waitFor();
  check(!/体验账号|界面演示/.test(await page.locator("body").innerText()), "unconfigured upstream never offers demo authentication");
  let signedIn = false, transferAttempts = 0, paid = false, walletFails = false;
  const user = { userId: "qa-user", playerRef: "player-qa", gameId: "QAPlayer", qq: "100001", identityStatus: "bound", permissions: ["core.read"] };
  await page.route("**/api/v1/**", async route => {
    const request = route.request(), path = "/api/v1" + request.url().split("/api/v1")[1].split("?")[0];
    const reply = (data, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(data) });
    const session = { data: { user, csrfToken: "qa-memory-csrf", expiresAt: "2030-01-01T00:00:00Z" } };
    if (path === "/api/v1/web/session") {
      if (request.method() === "POST") {
        const body = request.postDataJSON();
        if (body.password !== "qa-test-only-password") return reply({ error: { code: "INVALID_CREDENTIALS", message: "账号或密码错误" } }, 401);
        signedIn = true; return reply(session);
      }
      if (request.method() === "DELETE") { signedIn = false; return reply({ data: { loggedOut: true } }); }
      return signedIn ? reply(session) : reply({ error: { code: "UNAUTHORIZED", message: "请登录" } }, 401);
    }
    if (!signedIn) return reply({ error: { code: "UNAUTHORIZED", message: "请登录" } }, 401);
    if (path === "/api/v1/chat/messages") return reply({ data: { messages: [] }, page: { nextCursor: null } });
    if (path.startsWith("/api/v1/wallet/balance")) return walletFails ? reply({ error: { code: "PLUGIN_BRIDGE_UNAVAILABLE", message: "游戏节点暂时离线" } }, 503) : reply({ data: { balance: { amount: paid ? "1249.90" : "1250.00", currency: "CREDIT", fresh: true, refreshedAt: "2026-09-08T01:00:00Z" } } });
    if (path === "/api/v1/wallet/records") return reply({ data: { records: paid ? [{ recordId: "record-qa", direction: "expense", otherPlayer: { gameId: "QARecipient" }, amount: "0.10", status: "success", note: "浏览器契约验证", occurredAt: "2026-09-08T01:00:00Z" }] : [] }, page: { nextCursor: null } });
    if (path === "/api/v1/wallet/recipients/search") return reply({ data: { candidates: [user, { playerRef: "player-recipient", gameId: "QARecipient", confirmedAt: "2026-09-08T01:00:00Z" }] } });
    if (path === "/api/v1/wallet/transfers") {
      operations.push(request.postDataJSON());
      check(request.headers()["x-csrf-token"] === "qa-memory-csrf", "transfer carries current CSRF token");
      if (++transferAttempts === 1) return route.abort("connectionfailed");
      paid = true;
      return reply({ data: { transfer: { transferId: "transfer-qa", ...request.postDataJSON(), status: "success", recipient: { gameId: "QARecipient" } } } });
    }
    if (path === "/api/v1/admin/core/nodes") return reply({ data: { nodes: [] } });
    if (path === "/api/v1/admin/core/items") return reply({ data: { items: [], next: null } });
    if (path === "/api/v1/announcements") return reply({ data: [{ announcementId: "qa-ann", title: "QA公告", summary: "仅浏览器契约响应", pinned: false, contentBlocks: [{ blockId: "qa-block", type: "PARAGRAPH", text: "<script>window.qaExecuted=true</script>" }] }], page: { nextCursor: null } });
    if (path === "/api/v1/notifications" || path === "/api/v1/store/products" || path === "/api/v1/market/listings" || path === "/api/v1/commissions") return reply({ data: [], page: { nextCursor: null } });
    return reply({ error: { code: "NOT_IMPLEMENTED", message: "服务暂未开放" } }, 501);
  });
  await page.routeWebSocket("**/api/v1/chat/ws", ws => { ws.onMessage(raw => { const frame = JSON.parse(raw); ws.send(JSON.stringify({ type: "chat.send.result", requestId: frame.requestId, payload: { status: "accepted", messageId: "qa-message" } })); }); });
  await page.goto(base + "/preview/admin");
  await page.getByRole("textbox", { name: "游戏 ID / QQ" }).waitFor();
  check(page.url().endsWith("/admin"), "historical preview URL redirects to authenticated route");
  check(!/体验账号|演示/.test(await page.locator("body").innerText()), "login offers no synthetic identities or preview link");
  await page.getByRole("textbox", { name: "游戏 ID / QQ" }).fill("QAPlayer");
  await page.getByRole("textbox", { name: "密码", exact: true }).fill("incorrect");
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByRole("alert").filter({ hasText: "账号或密码错误" }).waitFor();
  check(true, "server credential rejection stays on login");
  await page.getByRole("textbox", { name: "密码", exact: true }).fill("qa-test-only-password");
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await page.getByText("实时连接正常", { exact: true }).waitFor();
  check(await page.evaluate(() => !JSON.stringify({ ...localStorage }).includes("qa-test-only-password") && !JSON.stringify({ ...localStorage }).includes("qa-memory-csrf")), "credentials and CSRF stay out of persistent browser storage");
  await page.getByRole("link", { name: "钱包", exact: true }).click();
  await page.getByText("1,250.00", { exact: true }).waitFor();
  await page.getByRole("heading", { name: "还没有账单" }).waitFor();
  check(true, "wallet reads server decimal balance and empty records");
  await page.getByRole("button", { name: "转账", exact: true }).click();
  await page.getByRole("textbox", { name: "收款玩家" }).fill("QARecipient");
  await page.getByRole("button", { name: "查找玩家" }).click();
  await page.getByRole("button", { name: "QARecipient", exact: true }).click();
  check(await page.getByRole("dialog").getByRole("button", { name: "QAPlayer", exact: true }).count() === 0, "self is removed from transfer recipients");
  await page.getByRole("textbox", { name: "金额（信用点）" }).fill("0.10");
  await page.getByRole("textbox", { name: "备注" }).fill("浏览器契约验证");
  await page.getByRole("button", { name: "下一步" }).click();
  check(operations.length === 0, "transfer waits for explicit amount and recipient confirmation");
  await page.getByRole("button", { name: "确认转账" }).click();
  await page.getByRole("alert").filter({ hasText: "无法连接服务" }).waitFor();
  check(await page.getByRole("heading", { name: "转账成功" }).count() === 0, "lost response never reports successful transfer");
  await page.reload();
  await page.getByRole("button", { name: "查看转账", exact: true }).waitFor();
  await page.getByRole("button", { name: "查看转账", exact: true }).click();
  await page.getByRole("button", { name: "核对转账结果", exact: true }).click();
  await page.getByRole("heading", { name: "转账成功" }).waitFor();
  check(JSON.stringify(operations[0]) === JSON.stringify(operations[1]), "refresh and unknown-result retry retain one immutable transfer request");
  check(operations[0].amount === "0.10" && operations[0].recipientPlayerRef === "player-recipient", "transfer sends exact decimal amount and confirmed player reference");
  check(await page.evaluate(() => sessionStorage.getItem("deuterium-transfer:qa-user") === null), "successful server result clears the pending transfer journal");
  await page.getByRole("button", { name: "关闭弹窗", exact: true }).click();
  await page.getByText("1,249.90", { exact: true }).waitFor();
  check(true, "successful transfer reloads server wallet balance");
  await page.getByRole("link", { name: "商城", exact: true }).click();
  await page.getByRole("heading", { name: "还没有上架商品" }).waitFor();
  check(!/iPhone|MacBook/.test(await page.locator("body").innerText()), "shop has no manufactured seed products");
  await page.getByRole("link", { name: "委托大厅", exact: true }).click();
  await page.getByRole("heading", { name: "当前没有待接取委托" }).waitFor();
  check(true, "commissions start with an actual empty service list");
  await page.goto(base + "/announcements");
  await page.getByRole("button", { name: /QA公告/ }).click();
  check((await page.getByRole("dialog").innerText()).includes("<script>window.qaExecuted=true</script>"), "announcement text is rendered as text, not executable HTML");
  check(await page.evaluate(() => !window.qaExecuted), "announcement content does not execute script");
  await page.getByRole("button", { name: "关闭弹窗", exact: true }).click();
  for (const width of [360, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    for (const path of ["/wallet", "/information", "/me", "/"]) {
      await page.goto(base + path);
      await page.locator(".app-shell").waitFor();
      check(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `no page overflow at ${width}px on ${path}`);
    }
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(base + "/wallet");
  await page.getByText("1,249.90", { exact: true }).waitFor();
  await page.screenshot({ path: "output/playwright/v2-release-wallet.png", animations: "disabled" });
  await page.getByRole("button", { name: "切换深色模式" }).click();
  await page.reload();
  check(await page.evaluate(() => document.documentElement.dataset.theme === "dark"), "selected theme survives reload");
  await page.getByText("1,249.90", { exact: true }).waitFor();
  walletFails = true;
  await page.getByRole("button", { name: "刷新余额", exact: true }).click();
  await page.getByRole("alert").filter({ hasText: "游戏节点暂时离线" }).waitFor();
  check(await page.getByText(/上次已知余额/).count() === 1, "offline refresh labels the retained balance as stale");
  check(await page.getByRole("button", { name: "转账", exact: true }).isDisabled(), "offline wallet cannot submit a fresh transfer");
  await page.goto(base + "/me");
  await page.getByRole("button", { name: "退出登录", exact: true }).click();
  await page.getByRole("textbox", { name: "游戏 ID / QQ" }).waitFor();
  check(true, "logout returns to real login");
  await page.screenshot({ path: "output/playwright/v2-release-login.png", animations: "disabled" });
  check(errors.length === 0, "browser flows have no JavaScript runtime exception");
  return { mode: "isolated browser contract responses; not live backend proof", count: passed.length, passed, errors };
}

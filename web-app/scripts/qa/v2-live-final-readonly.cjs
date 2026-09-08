async (page) => {
  const origin = "https://47.103.99.34", passed = [], errors = [], blockedWrites = [], apiReads = [], expectedScript = "/assets/index-tMukH1-X.js", transferId = "transfer_acad3a87845dcba74310068ae5ab734d791a7de1", messageId = "msg_41ec45fa0ca4e1db827ab93ac9906f73", messageText = "【2.0.0 联通验收】消息转发测试，请忽略。";
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  // Remove all earlier local HTML/asset overlays. These checks load the deployed bundle.
  await page.unroute(origin + "/**"); await page.unroute("**/api/v1/**");
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route(origin + "/api/v1/**", async (route) => {
    const request = route.request();
    if (!["GET", "HEAD", "OPTIONS"].includes(request.method())) { blockedWrites.push({ method: request.method(), path: request.url().slice(origin.length).split("?")[0] }); return route.abort("blockedbyclient"); }
    return route.continue();
  });
  const get = async (path) => { const response = await page.request.get(origin + path); apiReads.push({ path, status: response.status() }); check(response.status() === 200, `live GET ${path} is 200`); return (await response.json()).data; };
  const documentResponse = await page.goto(origin + "/orders"); await page.locator(".app-shell").waitFor();
  check(documentResponse.status() === 200, "orders loads directly from the deployed HTTPS document");
  const scripts = await page.locator("script[src]").evaluateAll((nodes) => nodes.map((node) => node.getAttribute("src")));
  check(scripts.includes(expectedScript), "current page loads deployed 4de1ee7 JavaScript index-tMukH1-X.js without a local overlay");
  check(await page.getByRole("heading", { name: "还没有订单", exact: true }).isVisible(), "deployed order page shows truthful empty state");
  const orders = await get("/api/v1/orders?limit=20"); check(Array.isArray(orders) && orders.length === 0, "order empty state matches actual 013 data");
  await page.goto(origin + "/commissions"); await page.getByRole("heading", { name: "当前没有委托", exact: true }).waitFor();
  const commissions = await get("/api/v1/commissions?limit=20"); check(Array.isArray(commissions) && commissions.length === 0, "commission hall empty state matches actual 013 data");
  for (const name of ["我发布的", "我接取的"]) { await page.getByRole("button", { name, exact: true }).click(); await page.getByRole("heading", { name: "当前没有委托", exact: true }).waitFor(); }
  check(true, "publisher and worker commission views also render actual empty state");
  await page.goto(origin + "/merchant"); await page.getByRole("heading", { name: "商店管理", exact: true }).waitFor();
  const merchantResponse = await page.request.get(origin + "/api/v1/merchant/me"), merchant = await merchantResponse.json(); apiReads.push({ path: "/api/v1/merchant/me", status: merchantResponse.status() });
  if (merchantResponse.status() === 403 && merchant.error?.code === "STORE_NOT_CONFIGURED") { await page.getByRole("alert").waitFor(); check((await page.getByRole("alert").innerText()) === merchant.error.message && await page.getByRole("button", { name: "创建商店", exact: true }).isVisible(), "unconfigured merchant account renders the actual no-store message and authorised create entry"); }
  else { check(merchantResponse.status() === 200 && Array.isArray(merchant.data?.storeIds), "merchant management uses actual configured stores"); if (!merchant.data.storeIds.length) check(await page.getByRole("combobox", { name: "当前商店", exact: true }).count() === 0, "empty merchant list does not invent a store"); }
  await page.goto(origin + "/admin"); await page.getByRole("button", { name: "平台介入", exact: true }).click(); await page.getByRole("heading", { name: "暂无介入案件", exact: true }).waitFor();
  const cases = await get("/api/v1/admin/interventions?limit=20"); check(Array.isArray(cases) && cases.length === 0, "platform intervention empty state matches actual case list");
  await page.getByRole("button", { name: "管理审计", exact: true }).click(); await page.locator("tbody tr").first().waitFor();
  const audit = await get("/api/v1/admin/audit-events?limit=20"); check(Array.isArray(audit) && audit.length > 0, "deployed 017 audit returns existing real records");
  check(audit.every((event) => Object.keys(event).sort().join(",") === "action,actorId,createdAt,eventId,resourceId"), "live audit only returns its five approved projection fields");
  check((await page.locator("tbody").innerText()).includes(audit[0].action), "audit table renders the actual latest recorded action");
  await page.screenshot({ path: "output/playwright/v2-final-live-audit.png", animations: "disabled" });
  await page.goto(origin + "/wallet"); await page.locator(".balance-number").waitFor();
  const wallet = await get("/api/v1/wallet/balance"), transfer = (await get("/api/v1/wallet/transfers/" + transferId)).transfer;
  check(transfer?.transferId === transferId && transfer.status === "success" && transfer.amount === "1.00", "previous authorized 1.00 transfer remains the same successful transaction");
  await page.locator("tbody tr").filter({ hasText: "luoyinwuchen1" }).first().waitFor();
  check((await page.locator("tbody tr").filter({ hasText: "luoyinwuchen1" }).first().innerText()).includes("1.00"), "wallet table renders the original recipient and exact 1.00 amount");
  const amount = wallet.balance?.amount ?? wallet.amount;
  check(typeof amount === "string" && (await page.locator(".balance-number").innerText()).replaceAll(",", "").trim() === amount, "wallet display matches the current authoritative balance string");
  await page.goto(origin + "/information"); await page.getByText("实时连接正常", { exact: true }).waitFor(); await page.locator(".bubble").filter({ hasText: messageText }).waitFor();
  const history = await get("/api/v1/chat/messages?limit=100"), matches = history.messages.filter((message) => message.messageId === messageId);
  check(matches.length === 1 && matches[0].content === messageText, "original authorized public message remains once under its original server ID");
  check(await page.locator(".bubble").filter({ hasText: messageText }).count() === 1, "deployed public chat renders the original verification message in one message bubble");
  await page.screenshot({ path: "output/playwright/v2-final-live-chat.png", animations: "disabled" });
  await page.setViewportSize({ width: 390, height: 844 }); await page.goto(origin + "/orders"); await page.getByRole("heading", { name: "还没有订单", exact: true }).waitFor();
  check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), "deployed orders fit 390px viewport");
  check(blockedWrites.length === 0, "read-only page navigation attempted no business HTTP mutations");
  check(errors.length === 0, "final live combination has no JavaScript runtime exceptions");
  return { mode: "actual deployed Web 4de1ee7 with Go 00fbdac reported by root; no local document or asset overlays", at: new Date().toISOString(), script: expectedScript, count: passed.length, passed, apiReads, blockedWrites, errors, originalTransferId: transferId, originalMessageId: messageId, auditRecordCount: audit.length };
}

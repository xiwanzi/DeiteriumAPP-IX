async (page) => {
  const passed = [], writes = [], reads = [], errors = [], origin = "http://127.0.0.1:5219", now = "2026-09-08T03:00:00Z";
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  const party = (name) => ({ displayName: name, playerRef: `player-${name}`, kind: "PLAYER" });
  let caseValue = null;
  const order = { orderId: "order-evidence", status: "SHIPPED", fundsStatus: "HELD", amount: "15.00", buyer: party("buyer"), seller: party("seller"), items: [{ productId: "product-evidence", title: "需要核对的订单", quantity: 1, unitPrice: "15.00" }], availableActions: ["REQUEST_INTERVENTION"], createdAt: now, version: 7, refundAttemptsUsed: 1 };
  page.on("pageerror", (error) => errors.push(error.message)); await page.unroute("**/api/v1/**");
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request(), path = "/api/v1" + request.url().split("/api/v1")[1].split("?")[0]; reads.push(path);
    const json = (data) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ data, page: { nextCursor: null } }) });
    if (path === "/api/v1/web/session") return json({ user: { userId: "evidence-buyer", playerRef: "player-buyer", gameId: "buyer", qq: "10001", permissions: [] }, csrfToken: "evidence-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/orders") return json([order]);
    if (path === "/api/v1/orders/order-evidence") return json(order);
    if (path === "/api/v1/chat/conversations") return json([{ conversationId: "private-deal", otherPlayer: { gameId: "seller" } }]);
    if (path === "/api/v1/chat/messages") return json({ messages: [{ messageId: "public-evidence", sender: { displayName: "buyer" }, content: "公开交付时间说明", sentAt: now }] });
    if (path === "/api/v1/chat/conversations/private-deal/messages") return json([{ messageId: "private-evidence", sender: { displayName: "seller" }, content: "双方确认的交付约定", sentAt: now }]);
    if (path === "/api/v1/orders/order-evidence/interventions") {
      const body = request.postDataJSON(); writes.push(body);
      caseValue = { caseId: "case-evidence", status: "SUBMITTED", applicant: party("buyer"), respondent: party("seller"), description: body.description, desiredResolution: body.desiredResolution, updatedAt: now, version: 1, evidenceAssets: [], evidenceEntries: [{ actorPlayerRef: "player-buyer", createdAt: now, description: body.description, relatedMessages: [{ messageId: "private-evidence", senderPlayerRef: "player-seller", content: "服务器冻结的交付约定" }] }], snapshot: { capturedAt: now, order } };
      order.availableActions = []; order.interventionCaseId = caseValue.caseId; return json(caseValue);
    }
    if (path === "/api/v1/interventions/case-evidence") return json(caseValue);
    return route.fulfill({ status: 404, contentType: "application/json", body: JSON.stringify({ error: { code: "NOT_FOUND", message: "未找到" } }) });
  });
  await page.setViewportSize({ width: 1440, height: 1000 }); await page.goto(origin + "/orders?order=order-evidence"); await page.getByRole("button", { name: "申请平台介入", exact: true }).click();
  check(!reads.some((path) => path.startsWith("/api/v1/chat/")), "case form does not fetch unrelated chat until the person opens selection");
  await page.getByRole("textbox", { name: "详细说明", exact: true }).fill("请核对双方确认的交付约定及当前交付进度。"); await page.getByRole("button", { name: "选择相关聊天", exact: true }).click(); await page.getByText("公开交付时间说明", { exact: false }).waitFor();
  check(await page.getByText("提交后，所选消息的正文快照会随案件提供给交易双方及处理管理员。请选择与本次争议相关的内容。", { exact: true }).isVisible(), "evidence disclosure explains the intended recipients");
  await page.getByRole("combobox", { name: "资料来源", exact: true }).selectOption("private-deal"); await page.getByRole("dialog").last().getByRole("checkbox").check();
  await page.getByRole("button", { name: /收起聊天资料/ }).click(); await page.getByRole("dialog").last().getByRole("button", { name: "确认提交", exact: true }).click(); await page.getByText("服务器冻结的交付约定", { exact: true }).waitFor();
  check(writes.length === 1 && JSON.stringify(writes[0].relatedMessageIds) === '["private-evidence"]', "case submission references only the explicitly chosen accessible message");
  check(Object.keys(writes[0]).sort().join(",") === "clientRequestId,description,desiredResolution,evidenceAssetIds,expectedVersion,reasonCode,relatedMessageIds", "client cannot provide forged message contents or authors");
  check(writes[0].expectedVersion === 7, "case submission preserves the authoritative transaction version");
  check(await page.getByText("服务器冻结的交付约定", { exact: true }).count() === 1, "case view displays server evidence snapshot rather than local selected text");
  await page.setViewportSize({ width: 390, height: 844 }); check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), "case evidence view fits mobile viewport");
  check(errors.length === 0, "evidence selection has no JavaScript runtime exception");
  return { mode: "isolated evidence contract responses; no real private messages shared", count: passed.length, passed, errors };
}

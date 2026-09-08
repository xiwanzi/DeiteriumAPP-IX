async (page) => {
  const origin = "http://127.0.0.1:5219", passed = [], errors = [], requests = [];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", (error) => errors.push(error.message)); await page.unroute("**/api/v1/**");
  await page.evaluate(() => { sessionStorage.removeItem("deuterium-ai-pending:qa-ai"); localStorage.setItem("deuterium-web-theme", "light"); });
  let admin = false, request = null, complete = false, resetCount = 0;
  const now = "2026-09-08T03:00:00Z", sources = [{ title: "结构引用", url: "https://example.com/cited", origin: "annotation" }, { title: "正文返回链接", url: "https://example.com/text-link", origin: "provider_text" }, { title: "不安全链接", url: "javascript:alert(1)", origin: "provider_text" }];
  const quota = () => ({ used: request ? 1 : 0, limit: 20, remaining: request ? 19 : 20, windowHours: 24, unlimited: admin, reserved: 0, resetsAt: "2026-09-09T03:00:00Z" });
  const messages = () => !request ? [] : [{ messageId: "ai-user", conversationId: "ai-conversation", role: "user", content: request.content, createdAt: now, status: "completed" }, { messageId: "ai-assistant", conversationId: "ai-conversation", role: "assistant", content: complete ? "这是完整回答。" : "这是", createdAt: now, status: complete ? "completed" : "incomplete", sources: complete ? sources : [], searchUsed: complete }];
  await page.route("**/api/v1/**", async (route) => {
    const path = "/api/v1" + route.request().url().split("/api/v1")[1].split("?")[0];
    const json = (data, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify({ data }) });
    if (path === "/api/v1/web/session") return json({ user: { userId: "qa-ai", playerRef: "player-ai", gameId: "QA_AI", qq: "100001", permissions: admin ? ["platform.admin"] : [] }, csrfToken: "qa-ai-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/chat/messages") return json({ messages: [] });
    if (path === "/api/v1/chat/conversations") return json([]);
    if (path === "/api/v1/ai/me") return json({ assistantName: "小祥", quota: quota(), conversation: { conversationId: "ai-conversation", active: true, startedAt: now, updatedAt: now }, pendingRequest: request && !complete ? { ...request, userMessageId: "ai-user", assistantMessageId: "ai-assistant", status: "incomplete", createdAt: now, updatedAt: now, retryAfterSeconds: 0 } : null });
    if (path === "/api/v1/ai/messages") return json({ messages: messages() });
    if (path === "/api/v1/ai/plans") return json({ plans: [{ planId: "free", name: "免费计划", price: "0.00", active: true, purchasable: false }, { planId: "pro", name: "Pro", price: "9999999.00", active: false, purchasable: false }] });
    if (path === "/api/v1/ai/conversation/reset") { resetCount++; request = null; complete = false; return json({ conversation: { conversationId: "reset-conversation" } }); }
    if (path === "/api/v1/ai/chat/stream") {
      const body = route.request().postDataJSON(); requests.push(body); request = body;
      check(route.request().headers()["x-csrf-token"] === "qa-ai-csrf", "AI stream carries current session CSRF");
      const frame = (type, data) => `event: ${type}\ndata: ${JSON.stringify(data)}\n\n`;
      let stream = frame("meta", { conversationId: "ai-conversation", userMessageId: "ai-user", assistantMessageId: "ai-assistant", userMessage: messages()[0], quota: quota() }) + frame("delta", { content: "这是" });
      if (requests.length > 1) { complete = true; stream += frame("delta", { content: "完整回答。" }) + frame("sources", { sources }) + frame("done", { message: messages()[1], quota: quota() }); }
      return route.fulfill({ status: 200, contentType: "text/event-stream", body: stream });
    }
    return json({}, 404);
  });
  await page.routeWebSocket("**/api/v1/chat/ws", () => {});
  await page.goto(origin + "/information"); await page.locator(".pc-contact").filter({ hasText: "小祥 AI" }).click();
  await page.getByText(/24 小时额度 · 20 \/ 20/).first().waitFor();
  await page.getByRole("textbox", { name: "消息内容" }).fill("请回答测试问题"); await page.getByRole("button", { name: "发送", exact: true }).click();
  await page.getByRole("alert").filter({ hasText: /回复连接已中断/ }).first().waitFor();
  check(await page.locator(".pc-message .bubble").filter({ hasText: "这是完整回答。" }).count() === 0, "EOF without done never displays a fabricated completed answer");
  await page.reload(); await page.locator(".pc-contact").filter({ hasText: "小祥 AI" }).click(); await page.getByRole("button", { name: "AI 选项", exact: true }).click();
  await page.getByRole("button", { name: "继续查看", exact: true }).click();
  await page.locator(".pc-message .bubble").filter({ hasText: "这是完整回答。" }).waitFor();
  check(requests.length === 2 && JSON.stringify(requests[0]) === JSON.stringify(requests[1]), "reload recovery replays the same AI clientMessageId and content");
  check(await page.locator(".ai-sources a").count() === 2, "only real HTTP(S) source shapes are rendered");
  check(await page.getByRole("link", { name: "正文链接 · 正文返回链接", exact: true }).count() === 1, "provider text URL is labelled as a text link, not a verified citation");
  check(await page.getByText("已使用联网搜索", { exact: true }).count() === 1, "search badge follows the actual done.searchUsed flag");
  check(await page.evaluate(() => sessionStorage.getItem("deuterium-ai-pending:qa-ai") === null), "completed AI response clears the recovery journal");
  await page.getByRole("button", { name: "AI 选项", exact: true }).click(); await page.getByRole("button", { name: "额度与计划", exact: true }).click();
  check((await page.getByRole("dialog").innerText()).includes("免费额度：20 次 / 24 小时"), "free plan displays the server's twenty-per-day quota");
  check((await page.getByRole("dialog").innerText()).includes("9,999,999.00"), "future paid plan shows the requested placeholder price");
  check(await page.getByRole("button", { name: "暂不提供购买", exact: true }).count() === 2 && await page.getByRole("button", { name: "暂不提供购买", exact: true }).last().isDisabled(), "paid purchase controls remain disabled");
  await page.getByRole("button", { name: "关闭弹窗", exact: true }).click();
  await page.getByRole("textbox", { name: "消息内容" }).fill("/new"); await page.getByRole("button", { name: "发送", exact: true }).click();
  await page.locator(".pc-message .bubble").filter({ hasText: "这是完整回答。" }).waitFor({ state: "hidden" });
  check(resetCount === 1 && requests.length === 2, "/new resets the backend conversation without making another provider request");
  admin = true; await page.reload(); await page.locator(".pc-contact").filter({ hasText: "小祥 AI" }).click(); await page.getByText(/管理员 · 不限次数/).first().waitFor(); check(true, "platform administrator quota is displayed as unlimited");
  for (const width of [360, 390, 768, 1440]) { await page.setViewportSize({ width, height: 900 }); check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `AI conversation fits ${width}px`); }
  await page.screenshot({ path: "output/playwright/v2-ai-contract.png", animations: "disabled" });
  check(errors.length === 0, "AI browser flows have no JavaScript runtime exceptions");
  return { mode: "isolated AI transport contract responses; not a provider call", count: passed.length, passed, errors };
}

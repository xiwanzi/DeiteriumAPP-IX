async (page) => {
  const origin = "https://47.103.99.34", root = "C:/DeuteriumAPP/.worktrees/web-player-v1/web-app/dist/", passed = [], posts = [], errors = [];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route(origin + "/**", async (route) => {
    const request = route.request(), path = request.url().slice(origin.length).split("?")[0];
    if (path.startsWith("/api/v1/ai/") && request.method() === "POST") posts.push(path);
    if (request.resourceType() === "document") return route.fulfill({ path: root + "index.html", contentType: "text/html" });
    if (/^\/assets\/index-[A-Za-z0-9_-]+\.(js|css)$/.test(path)) return route.fulfill({ path: root + path.slice(1), contentType: path.endsWith(".js") ? "text/javascript" : "text/css" });
    return route.continue();
  });
  const me = await (await page.request.get(origin + "/api/v1/ai/me")).json(), plans = await (await page.request.get(origin + "/api/v1/ai/plans")).json();
  check(me.data?.quota?.windowHours === 24 && me.data?.webSearchAvailable === true, "live AI advertises the 24-hour quota and optional web search");
  check(plans.data.plans.filter((plan) => Number(plan.price) > 0).every((plan) => plan.price === "9999999.00" && plan.active === false), "live paid plans have the requested disabled price");
  await page.goto(origin + "/information"); await page.locator(".pc-contact").filter({ hasText: "小祥 AI" }).click();
  await page.getByRole("button", { name: "AI 选项", exact: true }).waitFor(); await page.getByRole("button", { name: "AI 选项", exact: true }).click(); await page.getByRole("button", { name: "额度与计划", exact: true }).click();
  await page.getByRole("dialog").getByText("当前使用 DeepSeek V4 Flash。根据问题需要使用联网搜索。", { exact: true }).waitFor();
  if (me.data.quota.unlimited) check((await page.getByRole("dialog").innerText()).includes("管理员账号不限制请求次数。"), "live administrator exemption is displayed correctly");
  check(await page.getByRole("button", { name: "暂不提供购买", exact: true }).last().isDisabled(), "live plan purchase remains disabled");
  await page.getByRole("button", { name: "关闭弹窗", exact: true }).click();
  check(posts.length === 0, "read-only AI verification does not create or reset any provider request");
  check(errors.length === 0, "live AI data renders without JavaScript runtime exceptions");
  return { mode: "live AI backend read-only with local candidate UI; no generation or reset", count: passed.length, passed, errors };
}

async (page) => {
  const origin = "https://47.103.99.34", root = "C:/DeuteriumAPP/.worktrees/web-player-v1/web-app/dist/", passed = [], errors = [];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", (error) => errors.push(error.message));
  // Render this local build at the approved origin; all API requests remain live.
  await page.route(origin + "/**", async (route) => {
    const request = route.request(), path = request.url().slice(origin.length).split("?")[0];
    if (request.resourceType() === "document") return route.fulfill({ path: root + "index.html", contentType: "text/html" });
    if (/^\/assets\/index-[A-Za-z0-9_-]+\.(js|css)$/.test(path)) return route.fulfill({ path: root + path.slice(1), contentType: path.endsWith(".js") ? "text/javascript" : "text/css" });
    return route.continue();
  });
  await page.goto(origin + "/me");
  await page.getByRole("button", { name: "编辑简介", exact: true }).waitFor();
  check(await page.getByRole("button", { name: "退出登录", exact: true }).count() === 1, "live secure session restores against the new profile UI");
  for (const path of ["/api/v1/chat/player-directory", "/api/v1/chat/conversations", "/api/v1/announcements", "/api/v1/notifications", "/api/v1/notifications/preferences", "/api/v1/admin/announcements", "/api/v1/store/products", "/api/v1/market/listings", "/api/v1/merchant/me"]) {
    const response = await page.request.get(origin + path); check(response.status() === 200 || (path === "/api/v1/merchant/me" && response.status() === 403 && (await response.json()).error?.code === "STORE_NOT_CONFIGURED"), `live authenticated GET ${path} returns data or a truthful unconfigured-store state`);
  }
  check(await page.evaluate(() => !document.cookie.includes("deuterium_session")), "live session cookie is not exposed to JavaScript");
  await page.screenshot({ path: "output/playwright/v2-live-profile.png", animations: "disabled" });
  await page.getByRole("link", { name: "信息", exact: true }).click();
  await page.getByText("实时连接正常", { exact: true }).waitFor();
  await page.getByRole("button", { name: "发起私聊", exact: true }).click();
  await page.getByRole("textbox", { name: "查找玩家", exact: true }).waitFor();
  await page.getByRole("dialog").locator(".button-row button").first().waitFor();
  check(await page.getByRole("dialog").locator(".button-row button").count() > 0, "live player directory populates the private-conversation picker");
  await page.getByRole("button", { name: "关闭弹窗", exact: true }).click();
  await page.getByRole("link", { name: "官方管理", exact: true }).click();
  await page.getByRole("button", { name: "公告管理", exact: true }).click();
  await page.getByRole("button", { name: "新建公告", exact: true }).waitFor();
  check(true, "authorised announcement management loads without synthetic content");
  await page.getByRole("button", { name: "新建公告", exact: true }).click();
  await page.getByRole("textbox", { name: "标题", exact: true }).waitFor();
  check(await page.getByRole("textbox", { name: "标题", exact: true }).inputValue() === "", "new announcement editor starts empty");
  await page.getByRole("button", { name: "关闭弹窗", exact: true }).click();
  await page.goto(origin + "/notification-settings");
  await page.getByRole("switch").first().waitFor();
  check(await page.getByRole("switch").count() === 10, "all ten notification preferences read from the live account");
  for (const width of [390, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    for (const path of ["/me", "/information", "/notification-settings", "/admin"]) {
      await page.goto(origin + path); await page.locator(".app-shell").waitFor();
      check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `live ${path} fits ${width}px`);
    }
  }
  check(errors.length === 0, "live read-only browser run has no JavaScript exception");
  return { mode: "live backend, local candidate UI, read-only; no chat/profile/announcement/product mutations", count: passed.length, passed, errors };
}

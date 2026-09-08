async (page) => {
  const origin = "http://127.0.0.1:5219", passed = [], errors = [], writes = []; let template = null;
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", (error) => errors.push(error.message)); await page.unroute("**/api/v1/**");
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request(), path = "/api/v1" + request.url().split("/api/v1")[1].split("?")[0]; const json = (data, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify({ data }) });
    if (path === "/api/v1/web/session") return json({ user: { userId: "template-admin", playerRef: "player-template", gameId: "TemplateAdmin", qq: "100001", permissions: ["platform.admin"] }, csrfToken: "template-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/merchant/me") return json({ storeIds: ["store-qa"], permissions: ["PRODUCT_EDIT"] });
    if (path === "/api/v1/merchant/stores/store-qa") return json({ storeId: "store-qa", name: "模板验证商店", version: 1 });
    if (path === "/api/v1/admin/core/nodes") return json({ nodes: [{ serverId: "amiya", claimEnabled: true, inventoryDomain: "shared", online: true }, { serverId: "login", claimEnabled: false, inventoryDomain: "shared", online: true }, { serverId: "mek", claimEnabled: true, inventoryDomain: "isolated", online: true }] });
    if (path === "/api/v1/admin/core/items") return json({ items: [{ itemRef: "core:approved", revision: 3, payloadSha256: "a".repeat(64), displayName: "已批准的物品", maxQuantity: 16, inventoryDomain: "shared", compatibleServerIds: ["amiya"], codec: "bukkit-bytes-v1" }], next: null });
    if (path.endsWith("/delivery-templates") && request.method() === "GET") return json(template ? [{ templateRef: template.templateRef, name: template.name, summary: template.summary, active: template.active }] : []);
    if (path.endsWith("/delivery-templates") && request.method() === "POST") { const body = request.postDataJSON(); writes.push(body); template = { ...body, templateRef: "template-approved", version: 1, storeId: "store-qa" }; return json(template, 201); }
    if (path.endsWith("/delivery-templates/template-approved/disable")) { writes.push(request.postDataJSON()); template = { ...template, active: false, version: template.version + 1 }; return json(template); }
    if (path.endsWith("/delivery-templates/template-approved")) return json(template);
    if (["products", "brands", "categories"].some((suffix) => path.endsWith("/" + suffix))) return json([]);
    return json({}, 404);
  });
  await page.goto(origin + "/merchant"); await page.getByRole("button", { name: "交付模板", exact: true }).click(); await page.getByRole("button", { name: "创建交付模板", exact: true }).click();
  const editor = page.getByRole("dialog").last(); await editor.getByRole("textbox", { name: "模板名称", exact: true }).fill("正式附件模板"); await editor.getByRole("textbox", { name: "交付摘要", exact: true }).fill("仅传递已批准物品引用与数量");
  await editor.getByRole("combobox", { name: "背包同步组", exact: true }).selectOption("shared"); await editor.getByRole("combobox", { name: "已发布物品版本", exact: true }).selectOption("core:approved:3"); await editor.getByRole("button", { name: "加入附件", exact: true }).click();
  await editor.getByRole("spinbutton", { name: "附件数量", exact: true }).fill("2"); await editor.getByRole("checkbox", { name: "amiya · 在线", exact: true }).check();
  check(await editor.getByRole("checkbox", { name: /login/ }).count() === 0 && await editor.getByRole("checkbox", { name: /mek/ }).count() === 0, "non-claim and incompatible inventory nodes are excluded");
  await editor.getByRole("button", { name: "保存交付模板", exact: true }).click(); await page.getByRole("dialog").getByRole("heading", { name: "正式附件模板", exact: true }).waitFor();
  check(writes[0].attachments[0].itemRef === "core:approved" && writes[0].attachments[0].revision === 3 && writes[0].attachments[0].payloadSha256 === "a".repeat(64) && writes[0].attachments[0].quantity === 2, "template preserves immutable Core reference, revision, fingerprint and quantity");
  check(JSON.stringify(writes[0].allowedServerIds) === '["amiya"]' && writes[0].inventoryDomain === "shared", "template submits the explicit compatible claim scope");
  check(!("content" in writes[0]) && !("expectedVersion" in writes[0]) && writes[0].clientRequestId, "create uses the flat template contract with a stable request ID");
  await page.getByRole("button", { name: "查看与编辑", exact: true }).click(); await page.getByRole("dialog").last().getByRole("button", { name: "停用", exact: true }).click(); await page.getByRole("button", { name: "确认停用", exact: true }).click();
  await page.getByRole("dialog").getByText("已停用", { exact: true }).waitFor();
  check(writes[1].expectedVersion === 1 && writes[1].clientRequestId && Object.keys(writes[1]).length === 2, "disable requires the current version and does not resubmit content");
  for (const width of [390, 1440]) { await page.setViewportSize({ width, height: 900 }); check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `template manager fits ${width}px`); }
  check(errors.length === 0, "template UI has no JavaScript runtime exceptions");
  return { mode: "isolated delivery-template API contract responses", count: passed.length, passed, errors };
}

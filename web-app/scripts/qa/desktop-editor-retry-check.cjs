async (page) => {
  const passed = [], writes = [], publishes = [];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("request", request => { if (request.method() === "PUT" && request.url().endsWith("/products/product_one")) writes.push(request.postDataJSON()); });
  await page.route("**/api/v1/merchant/products/product_one/publish", route => {
    publishes.push(route.request().postDataJSON());
    return route.fulfill({status:publishes.length === 1 ? 503 : 200,contentType:"application/json",body:JSON.stringify(publishes.length === 1 ? {error:{code:"UNAVAILABLE",message:"发布暂未确认，请重试原请求。"}} : {data:{productId:"product_one",version:3,visibility:"ACTIVE"}})});
  });
  await page.getByRole("button", {name:"编辑 / 预览",exact:true}).first().click();
  check(await page.getByRole("textbox",{name:"第 1 张图片说明",exact:true}).inputValue() === "石材", "saved gallery alt text is retained when opening an editor");
  await page.getByRole("textbox", {name:"邮件标题",exact:true}).fill("新邮件标题");
  await page.getByRole("button", {name:"将第 2 张设为封面",exact:true}).click();
  await page.getByRole("button", {name:"保存并发布",exact:true}).click();
  await page.getByRole("alert").filter({hasText:"草稿已保存。发布暂未确认"}).waitFor();
  check(writes.length === 1 && writes[0].content.coverAssetId === "asset_two" && writes[0].content.mailTitle === "新邮件标题", "save persists new cover and mail content before publication");
  await page.getByRole("button", {name:"保存并发布",exact:true}).click();
  await page.getByRole("dialog").waitFor({state:"hidden"});
  check(writes.length === 1 && publishes.length === 2 && JSON.stringify(publishes[0]) === JSON.stringify(publishes[1]), "retry after uncertain publication reuses the same version and request ID without another draft write");
  await page.routeWebSocket("**/api/v1/chat/ws", () => {});
  await page.goto("http://127.0.0.1:5246/information");
  await page.waitForFunction(() => [...document.querySelectorAll(".detail-member img.avatar")].some(img => img.complete && img.naturalWidth > 0));
  check(await page.locator(".detail-member img.avatar").count() === 1, "recent public-chat participants use the resolved profile avatar too");
  await page.screenshot({path:"output/playwright/chat-avatar-light.png"});
  return {passed};
}

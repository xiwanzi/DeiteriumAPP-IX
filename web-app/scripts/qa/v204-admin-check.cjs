async (page) => {
  const passed=[],errors=[],writes=[];
  try {
  const check=(value,label)=>{if(!value)throw new Error(label);passed.push(label);};
  page.on("pageerror",error=>errors.push(error.message));page.on("request",request=>{if(["POST","PUT","DELETE"].includes(request.method()))writes.push({path:request.url(),body:request.postDataJSON()});});
  await page.setViewportSize({width:1440,height:1000});
  await page.getByRole("button",{name:"管理审计",exact:true}).click();await page.getByRole("heading",{name:"交易与玩家审计"}).waitFor();await page.locator("tbody tr").first().waitFor();
  check(await page.getByText("游戏内支出",{exact:true}).count()>=1,"game activity appears in full-server ledger");
  await page.screenshot({path:"output/playwright/v204-audit-light.png",fullPage:true,animations:"disabled"});
  await page.getByRole("button",{name:"玩家查询",exact:true}).click();await page.getByRole("textbox",{name:"搜索玩家"}).waitFor();await page.locator("tbody tr").first().waitFor();
  await page.locator("tbody tr").first().getByRole("button",{name:"订单",exact:true}).click();await page.getByText("当前查看此玩家的记录",{exact:true}).waitFor();await page.locator("tbody tr").first().waitFor();
  check(await page.getByText("青梧",{exact:true}).count()>0,"player scope is visible when drilling into order history");
  await page.getByRole("button",{name:"详情",exact:true}).click();await page.getByRole("dialog").getByText("只读审计",{exact:true}).waitFor();check(await page.getByRole("dialog").getByText("主城仓库",{exact:true}).count()===1,"audited order exposes delivery details without write actions");await page.getByRole("button",{name:"关闭弹窗",exact:true}).click();
  await page.getByRole("button",{name:"商品审计",exact:true}).click();await page.getByRole("table").getByText("已下架",{exact:true}).waitFor();check(await page.getByText("石材建造礼包",{exact:true}).count()===1,"unlisted product remains in audit history");
  await page.getByRole("button",{name:"管理操作",exact:true}).click();await page.getByText("announcement.delete",{exact:true}).waitFor();check(true,"existing management action audit remains accessible");
  await page.getByRole("button",{name:"公告管理",exact:true}).click();await page.getByRole("button",{name:"永久删除",exact:true}).click();await page.getByRole("dialog").waitFor();
  check(await page.getByRole("dialog").getByText(/无法恢复/).count()===1,"permanent announcement deletion explains irreversible effect");
  await page.getByRole("button",{name:"保留公告",exact:true}).click();check(writes.filter(r=>r.path.endsWith("/delete")).length===0,"cancelling permanent delete does not mutate announcement");
  await page.getByRole("button",{name:"永久删除",exact:true}).click();await page.getByRole("button",{name:"确认永久删除",exact:true}).click();await page.getByRole("heading",{name:"还没有公告",exact:true}).waitFor();check(writes.filter(r=>r.path.endsWith("/delete")).length===1,"confirmed permanent delete sends one versioned request");
  await page.getByRole("button",{name:"邮件提醒",exact:true}).click();await page.getByRole("heading",{name:"SMTP 配置",exact:true}).waitFor();await page.getByRole("checkbox",{name:"启用平台介入邮件提醒"}).check();await page.getByRole("textbox",{name:"接收邮箱",exact:true}).fill("new-admin@example.com");
  await page.getByRole("button",{name:"公告管理",exact:true}).click();check(await page.getByRole("heading",{name:"SMTP 配置",exact:true}).count()===1,"unsaved SMTP edits survive cancelled navigation");
  await page.getByRole("button",{name:"保存配置",exact:true}).click();await page.getByText("邮件配置已保存。新的介入状态将按此配置提醒。",{exact:true}).waitFor();
  check(await page.getByRole("button",{name:"发送测试邮件",exact:true}).isEnabled(),"saved SMTP configuration can immediately send a test");
  await page.getByRole("button",{name:"发送测试邮件",exact:true}).click();await page.getByText("已被邮件服务器接收",{exact:true}).waitFor();check(writes.find(r=>r.path.endsWith("/email-settings")).body.password===undefined,"saving recipients preserves hidden SMTP password");
  await page.screenshot({path:"output/playwright/v204-email-light.png",fullPage:true,animations:"disabled"});
  await page.getByRole("button",{name:"切换深色模式",exact:true}).click();await page.screenshot({path:"output/playwright/v204-email-dark.png",fullPage:true,animations:"disabled"});
  await page.getByRole("button",{name:"商店管理",exact:true}).click();await page.getByRole("button",{name:"进入商店管理",exact:true}).click();await page.getByRole("heading",{name:"主城物资商店",exact:true}).waitFor();await page.getByRole("button",{name:"编辑",exact:true}).click();
  const dialog=page.getByRole("dialog");await dialog.getByRole("textbox",{name:"商品标题",exact:true}).fill("改良石材礼包");await page.mouse.click(5,5);check(await dialog.count()===1,"product backdrop cannot discard an edit");
  await dialog.getByRole("button",{name:"关闭弹窗",exact:true}).click();check(await dialog.getByRole("textbox",{name:"商品标题",exact:true}).inputValue()==="改良石材礼包","close confirmation preserves edited product title when cancelled");
  await dialog.locator(".editor-actions").scrollIntoViewIfNeeded();await page.screenshot({path:"output/playwright/v204-product-dark.png",fullPage:true,animations:"disabled"});
  await dialog.getByRole("button",{name:"保存并发布",exact:true}).click();await dialog.waitFor({state:"hidden"});check(writes.some(r=>r.path.endsWith("/products/product_one")&&r.body.content?.title==="改良石材礼包")&&writes.some(r=>r.path.endsWith("/products/product_one/publish")),"save-and-publish persists the draft and publishes the returned version");
  await page.setViewportSize({width:390,height:844});await page.getByRole("button",{name:"新增商品",exact:true}).click();await page.getByRole("dialog").waitFor();check(await page.getByRole("combobox",{name:"品牌",exact:true}).inputValue()==="brand_one"&&await page.getByRole("combobox",{name:"分类",exact:true}).inputValue()==="category_one","single brand and category choices are preselected");
  check(await page.getByRole("combobox",{name:"游戏内邮箱交付模板",exact:true}).inputValue()==="template_one","single delivery template is preselected");
  check(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),"merchant editor fits mobile viewport");await page.screenshot({path:"output/playwright/v204-product-mobile.png",fullPage:true,animations:"disabled"});
  await page.getByRole("button",{name:"关闭弹窗",exact:true}).click();
  check(errors.length===0,"new admin flows have no JavaScript runtime errors");
  const result={mode:"isolated browser contract fixtures; backend authority tested separately",count:passed.length,passed,errors};
  await page.evaluate(value=>{window.__v204Check=value},result);
  return result;
  } catch(error) { const result={count:passed.length,passed,errors,error:String(error)};await page.evaluate(value=>{window.__v204Check=value},result);return result; }
}

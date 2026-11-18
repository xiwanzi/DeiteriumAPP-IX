async page => {
  const origin = new URL(page.url()).origin;
  const passed = [], errors = [], posts = [], queries = [];
  const check = (value,name) => { if (!value) throw new Error(name); passed.push(name); };
  page.on("pageerror",error=>errors.push(error.message));
  const user={userId:"wallet-browser-test",playerRef:"player_self",gameId:"TestPlayer",gameVerified:true,permissions:[]};
  const peer={gameId:"QingHe",playerRef:"player_peer"}, oldPeer={gameId:"OldName",playerRef:"player_old"};
  const now=Date.now(), day=new Date(now+28800000).toISOString().slice(0,10), time=new Date(now).toISOString();
  const expense={recordId:"econ_3",direction:"expense",amount:"12.30",businessType:"OFFICIAL_STORE",title:"商城交易",note:"花田路灯",status:"success",beforeBalance:"1246.86",afterBalance:"1234.56",occurredAt:time};
  const income={...expense,recordId:"econ_2",amount:"88.20",direction:"income",businessType:"TRANSFER",title:"玩家转账",note:"车站灯笼",otherPlayer:peer};
  const prior={...expense,recordId:"econ_1",note:"旧流水",occurredAt:"2025-01-02T09:00:00Z"};
  let balanceDown=false,missingStats=false,amount="1234.56",failMore=false,delayMore=null,delaySearch=null,lose=true,lookupDown=true,detailsDown=false;
  const send=(route,data,pageInfo)=>route.fulfill({json:{data,page:pageInfo,serverTime:time}});
  const fail=(route,code,message,status=503)=>route.fulfill({status,json:{error:{code,message}}});
  await page.routeWebSocket("**/api/v1/chat/ws",()=>{});
  await page.route("**/web-config.json",r=>r.fulfill({json:{configured:true,mode:"connected",version:"test"}}));
  await page.route("**/api/v1/**", async route=>{
    const req=route.request(), url=new URL(req.url()), path=url.pathname;
    if(path.endsWith("/web/session"))return send(route,{user,csrfToken:"wallet-test-only",expiresAt:"2030-01-01T00:00:00Z"});
    if(path.startsWith("/api/v1/wallet/balance")) return balanceDown ? fail(route,"PLUGIN_BRIDGE_UNAVAILABLE","余额暂不可用") : send(route,{balance:{amount,availableAmount:amount,heldAmount:"40.00",today:missingStats?null:{date:day,income:"88.20",expense:"24.60"},fresh:true,serverNow:time}});
    if(path.endsWith("/wallet/records")){
      const params=url.searchParams;queries.push(Object.fromEntries(params));
      if(params.get("cursor")==="page2"){
        if(delayMore){const hold=delayMore;delayMore=null;await hold();}
        if(failMore){failMore=false;return fail(route,"TEMPORARY","账单续页暂不可用");}
        return send(route,{records:[expense,prior]},{hasMore:false});
      }
      if(params.has("from") && params.get("from").startsWith("2025-01-01"))return send(route,{records:[prior]});
      if(params.get("businessType")==="COMMISSION")return send(route,{records:[]});
      if(params.get("direction")==="income")return send(route,{records:[income]});
      return send(route,{records:[expense]},{hasMore:true,nextCursor:"page2"});
    }
    if(path.includes("/wallet/records/"))return detailsDown?fail(route,"TEMPORARY","账单详情暂不可用"):send(route,{...expense,status:"SUCCESS",businessRef:"order_one"});
    if(path.endsWith("/wallet/recipients/search")){
      if(url.searchParams.get("query")==="old"){if(delaySearch){const hold=delaySearch;delaySearch=null;await hold();}return send(route,{candidates:[oldPeer]});}
      return send(route,{candidates:[user,{...peer,expiresAt:new Date(now+(url.searchParams.get("query")==="expired"?-60000:300000)).toISOString()}]});
    }
    if(path.endsWith("/wallet/transfers")){
      posts.push(req.postDataJSON());
      if(lose){lose=false;return route.abort("connectionfailed");}
      return send(route,{transfer:{...posts.at(-1),transferId:"tr_wallet",status:"processing",recipient:peer}});
    }
    if(path.endsWith("/wallet/transfers/tr_wallet")){
      if(lookupDown){lookupDown=false;return fail(route,"PLUGIN_BRIDGE_UNAVAILABLE","结果暂时无法读取");}
      amount="1234.46"; return send(route,{transfer:{transferId:"tr_wallet",amount:"0.10",status:"success",recipient:peer}});
    }
    if(path.endsWith("/store/cart"))return send(route,{version:1,items:[]});
    if(path.includes("/players/"))return send(route,user);
    if(path.endsWith("/account/deletions")||path.endsWith("/chat/player-names"))return send(route,{items:[],cursor:0,hasMore:false});
    return send(route,[]);
  });
  const button=name=>page.getByRole("button",{name,exact:true});
  await page.setViewportSize({width:1440,height:900});await page.goto(origin+"/wallet");await page.locator(".ledger-row").first().waitFor();
  check(await page.locator(".balance-number").innerText()==="1,234.56" && await page.locator(".wallet-held strong").innerText()==="40.00","available balance and held funds stay distinct");
  check(await button("查看当日收入账单").innerText().then(t=>t.includes("88.20")),"daily totals come from the balance service");
  let releaseMore;
  const moreStarted=new Promise(started=>{delayMore=()=>new Promise(resolve=>{releaseMore=resolve;started();});});
  await button("加载更多账单").click();await moreStarted;
  await button("查看当日收入账单").click();await page.locator(".ledger-row").filter({hasText:"QingHe"}).waitFor();
  const late=page.waitForResponse(r=>r.url().includes("cursor=page2"));releaseMore();await late;
  check(queries.some(q=>q.direction==="income"&&q.from===new Date(day+"T00:00:00+08:00").toISOString()) && await page.locator(".ledger-row").count()===1,"today drilldown uses Beijing midnight and ignores a late previous page");
  await page.getByLabel("交易类型",{exact:true}).selectOption("COMMISSION");await page.getByText("没有符合条件的账单",{exact:true}).waitFor();
  check(queries.at(-1).businessType==="COMMISSION"&&queries.at(-1).direction==="income","business filters apply to the full account ledger");
  await button("重置筛选").click();await page.getByLabel("账单日期").fill("2025-01-02");await page.locator(".ledger-row").filter({hasText:"旧流水"}).waitFor();
  check(queries.at(-1).from==="2025-01-01T16:00:00.000Z"&&queries.at(-1).to==="2025-01-02T16:00:00.000Z","past-day filter uses an exclusive next midnight");
  await button("重置筛选").click();await button("加载更多账单").waitFor();failMore=true;await button("加载更多账单").click();await button("重试读取").click();
  await page.locator(".ledger-row").filter({hasText:"旧流水"}).waitFor();check(await page.locator(".ledger-row").count()===2,"retry appends the same cursor and deduplicates record IDs");
  detailsDown=true;await page.locator(".ledger-row").first().click();await page.getByRole("alert").getByText("账单详情暂不可用").waitFor();detailsDown=false;await button("重试").click();
  const dialog=page.getByRole("dialog");await dialog.getByText("1,246.86",{exact:true}).waitFor();
  check(await dialog.getByText("1,234.56",{exact:true}).isVisible()&&await dialog.getByText("econ_3",{exact:true}).isVisible(),"detail shows exact before/after balances and record ID");
  await button("关闭弹窗").click();balanceDown=true;await button("刷新余额").click();await page.getByText(/上次已知余额/).waitFor();
  check(await button("转账").isDisabled()&&await page.locator(".balance-number").innerText()==="1,234.56","failed refresh retains the last known amount and blocks a fresh transfer");
  balanceDown=false;missingStats=true;await button("重试").click();await page.waitForFunction(()=>document.querySelector(".wallet-today-rows button").disabled);
  check(await button("查看当日收入账单").innerText().then(t=>t.includes("—")),"missing daily totals are not shown as zero");
  await button("转账").click();const query=page.getByRole("textbox",{name:"收款玩家"});
  let releaseSearch;
  const searchStarted=new Promise(started=>{delaySearch=()=>new Promise(resolve=>{releaseSearch=resolve;started();});});
  await query.fill("old");await button("查找玩家").click();await searchStarted;await query.fill("QingHe");await button("查找玩家").click();await button("QingHe").waitFor();
  const oldResponse=page.waitForResponse(r=>r.url().includes("query=old"));releaseSearch();await oldResponse;
  check(await button("OldName").count()===0&&await button(user.gameId).count()===0,"old search results and the sender are excluded");
  await query.fill("expired");await button("查找玩家").click();await button("QingHe").click();
  await page.getByRole("textbox",{name:"金额（信用点）"}).fill("1.00");await button("下一步").click();await query.waitFor();
  check(posts.length===0 && await page.getByRole("alert").innerText().then(t=>t.includes("过期")),"expired recipient selection returns to search without submitting");
  await query.fill("QingHe");await button("查找玩家").click();await button("QingHe").click();const money=page.getByRole("textbox",{name:"金额（信用点）"});
  await money.fill("0");await button("下一步").click();await page.getByRole("alert").waitFor();check(posts.length===0,"invalid amount cannot proceed");
  await money.fill("0.10");await page.getByRole("textbox",{name:"备注"}).fill("修路");await button("下一步").click();await button("返回修改").click();
  check(await money.inputValue()==="0.10"&&await page.getByRole("textbox",{name:"备注"}).inputValue()==="修路","editing confirmation restores the draft and payee");
  await button("下一步").click();check(await dialog.getByText("QingHe",{exact:true}).isVisible()&&posts.length===0,"review shows the recipient and does not send funds");
  await button("确认转账").click();await dialog.getByRole("alert").waitFor();await page.reload();await button("查看转账").click();await button("核对转账结果").click();
  await page.waitForFunction(()=>JSON.parse(sessionStorage.getItem("deuterium-transfer:wallet-browser-test"))?.result?.transferId==="tr_wallet");
  check(posts.length===2&&JSON.stringify(posts[0])===JSON.stringify(posts[1]),"lost acknowledgement reuses the original confirmed request after reload");
  await button("核对转账结果").click();await dialog.getByText("结果暂时无法读取").waitFor();
  check(await button("核对转账结果").isVisible()&&await button("返回").count()===0,"failed result lookup keeps an existing transfer recoverable");
  await button("核对转账结果").click();await page.getByRole("heading",{name:"转账成功"}).waitFor();await button("关闭弹窗").click();await page.getByText("1,234.46",{exact:true}).waitFor();
  check(posts.length===2&&await page.evaluate(()=>!sessionStorage.getItem("deuterium-transfer:wallet-browser-test")),"confirmed recovery refreshes the balance without an extra transfer");
  amount="9007199254740993.67";missingStats=false;
  await page.setViewportSize({width:320,height:700});await button("刷新余额").click();await page.getByText("9,007,199,254,740,993.67",{exact:true}).waitFor();
  check(await page.evaluate(()=>document.documentElement.scrollWidth===innerWidth),"large decimal balance fits a 320px viewport without rounding");
  await button("筛选账单").click();await page.waitForFunction(()=>document.activeElement?.getAttribute("aria-label")==="账单日期");
  check(await page.getByLabel("交易类型",{exact:true}).isVisible(),"wallet shortcut opens the mobile filters and focuses the date control");
  check(errors.length===0,"no browser runtime errors");return {passed,errors};
}

async page => {
  const origin = new URL(page.url()).origin, passed = [], errors = [], payments = [], prices = [], searches = [];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", e => errors.push(e.message));
  const user = {userId:"storefront-test", playerRef:"test-player", displayName:"晴川", gameId:"FixturePlayer", gameVerified:true, permissions:[]};
  const categories = [{categoryId:"home",name:"家具"},{categoryId:"beds",parentCategoryId:"home",name:"床"},{categoryId:"lamps",name:"灯具"}];
  const photos = [[320,80],[80,320]].map(([width,height],i)=>({assetId:"asset_"+i,url:"data:image/svg+xml,"+encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><rect width="100%" height="100%" fill="#bedce5"/><rect x="2" y="2" width="${width-4}" height="${height-4}" fill="none" stroke="navy" stroke-width="4"/></svg>`)}));
  const products = ["蓝蘑菇床","木床","南瓜灯","台灯","阅读灯","地灯","壁灯","落地灯"].map((title,index)=>({
    productId:"product_"+index,version:1,availableStock:index===7?0:null,effectivePrice:"12.00",images:photos,
    content:{title,price:"15.00",discountRate:8000,inventoryPolicy:index===7?"FINITE":"UNLIMITED",limitPerOrder:8,categoryId:index<2?"beds":"lamps"}
  }));
  let batch=0, cart=[], loseAdd=false, expire=false, failQuote=false, losePayment=false;
  const adds=[], writes=new Map();
  let cartVersion=1, holdRead=null;
  const snapshot=()=>({items:cart.map(item=>({...item})),version:cartVersion});
  await page.routeWebSocket("**/api/v1/chat/ws",()=>{});
  await page.route("**/web-config.json",route=>route.fulfill({json:{mode:"connected",configured:true,version:"test"}}));
  await page.route("**/api/v1/**",async route=>{
    const req=route.request(), url=new URL(req.url()), path=url.pathname;
    const reply=(data,more=null)=>route.fulfill({json:{data,page:{hasMore:!!more,nextCursor:more},serverTime:new Date().toISOString()}});
    if(path.endsWith("/web/session")) return reply({user,csrfToken:"fixture",expiresAt:"2030-01-01T00:00:00Z"});
    if(path.endsWith("/store/categories")) return url.searchParams.has("cursor")?reply(categories.slice(1)):reply(categories.slice(0,1),"next");
    if(path.endsWith("/store/recommendations")) {const offset=batch++%2;return reply(products.slice(offset,offset+6));}
    if(path.endsWith("/store/products")){
      searches.push(url.search);const group=url.searchParams.get("categoryGroup"), child=url.searchParams.get("subcategory"), query=url.searchParams.get("q")||"";
      return reply(products.filter(p=>(!url.searchParams.get("inStock")||p.availableStock!==0)&&(!group||p.content.categoryId===(group==="家具"?"beds":"lamps"))&&(!child||p.content.categoryId==="beds")&&p.content.title.includes(query)));
    }
    if(path.includes("/store/products/"))return reply(products.find(p=>path.endsWith(p.productId)));
    if(path.endsWith("/store/cart")){
      const value=snapshot();
      if(holdRead){const hold=holdRead;holdRead=null;await hold();}
      return reply(value);
    }
    if(path.includes("/store/cart/items/")){
      const body=req.postDataJSON(),productId=path.split("/")[6]; adds.push(body);
      if(writes.has(body.clientRequestId))return reply(writes.get(body.clientRequestId));
      if(body.expectedVersion!==cartVersion)return route.fulfill({status:409,json:{error:{code:"STATE_VERSION_CONFLICT",message:"购物袋已变更。"}}});
      cart=cart.filter(item=>item.productId!==productId);
      if(!path.endsWith("/remove"))cart.push({productId,quantity:body.quantity,productVersion:1});
      cartVersion++; writes.set(body.clientRequestId,snapshot());
      if(loseAdd){loseAdd=false;return route.fulfill({status:503,json:{error:{code:"TEMPORARY",message:"加购回执丢失"}}});}
      return reply(snapshot());
    }
    if(path.endsWith("/checkout/quotes")){
      const body=req.postDataJSON();prices.push(body);
      if(failQuote){failQuote=false;return route.fulfill({status:503,json:{error:{code:"TEMPORARY",message:"金额暂不可用"}}});}
      const items=body.items.map(item=>({...item,title:products.find(p=>p.productId===item.productId).content.title,unitPrice:"12.00"}));
      const expiresAt=new Date(Date.now()+(expire?-60000:300000)).toISOString();expire=false;
      return reply({items,totalAmount:(items.reduce((total,item)=>total+item.quantity*12,0)-1).toFixed(2),quoteId:"q"+prices.length,version:1,expiresAt});
    }
    if(path.endsWith("/store/orders")) {
      payments.push(req.postDataJSON());
      if(losePayment){losePayment=false;return route.fulfill({status:503,json:{error:{code:"TEMPORARY",message:"结果暂未返回"}}});}
      const quote=prices[Number(req.postDataJSON().quoteId.slice(1))-1];
      if(quote.source==="CART"){
        cart=cart.map(item=>({...item,quantity:item.quantity-(quote.items.find(line=>line.productId===item.productId)?.quantity||0)})).filter(item=>item.quantity>0);
        cartVersion++;
      }
      return reply({operation:{status:"COMPLETED",operationId:"op",resourceType:"ORDER",resourceId:"order"},order:{orderId:"order",orderNo:"T1",amount:"11.00",fundsStatus:"PAID"}});
    }
    if(path.includes("/operations/"))return route.fulfill({status:404,json:{error:{code:"NOT_FOUND",message:"尚未收到原请求"}}});
    if(path.endsWith("/wallet/balance"))return reply({balance:{amount:"120.00",fresh:true}});
    if(path.endsWith("/wallet/records"))return reply({records:[]});
    if(path.endsWith("/chat/stickers"))return reply({stickers:[]});
    if(path.endsWith("/chat/messages"))return reply({messages:[]});
    if(path.endsWith("/chat/follows"))return reply({players:[]});
    if(path.includes("/players/"))return reply(user);
    if(path.endsWith("/account/deletions")||path.endsWith("/chat/player-names"))return reply({items:[],cursor:0,hasMore:false});
    return reply([]);
  });
  await page.setViewportSize({width:1440,height:900});await page.goto(origin+"/");
  const button=name=>page.getByRole("button",{name,exact:true});
  const picks=page.locator(".shop-pick .shop-title"), tiles=page.locator(".shop-tile"), search=page.getByRole("textbox",{name:"搜索官方商城"});
  await picks.first().waitFor();const first=await picks.first().innerText();
  const badge=page.locator(".bag-count:visible");
  check(await badge.count()===0,"an empty cart has no badge");
  await button("换一批").click();await page.waitForFunction(first=>document.querySelector(".shop-pick .shop-title").textContent!==first,first);
  check(await picks.count()===6,"refresh replaces the recommendation shelf");
  await button("家具").click();await button("床").click();await search.fill("蓝");
  await page.waitForFunction(()=>document.querySelectorAll(".shop-tile").length===1);
  check(searches.some(q=>{const p=new URLSearchParams(q);return p.get("subcategory")==="床"&&p.get("categoryGroup")==="家具"&&p.get("q")==="蓝"}),"search and two category levels query the complete catalogue together");
  check(await picks.count()===0,"recommendations give way to the selected catalogue");
  await button("清除筛选").click();await picks.first().waitFor();
  await page.keyboard.press("/");check(await search.evaluate(el=>el===document.activeElement),"store search shortcut focuses the product field");
  await page.getByRole("checkbox",{name:"仅看有货"}).check();await page.waitForFunction(()=>document.querySelectorAll(".shop-tile").length===7);
  check(searches.some(q=>new URLSearchParams(q).get("inStock")==="true"),"stock filter is applied by the catalogue service");
  await page.getByRole("checkbox",{name:"仅看有货"}).uncheck();
  await button("紧凑列表").click();check(await tiles.first().evaluate(el=>el.getBoundingClientRect().height<100),"compact view exposes products in single-line rows");
  loseAdd=true;await button("加购 蓝蘑菇床").click();await page.getByText("加购回执丢失",{exact:true}).waitFor();
  await button("加购 蓝蘑菇床").click();await page.locator(".shop-cart-notice").getByText("已加入购物袋",{exact:true}).waitFor();
  check(adds.length===2&&JSON.stringify(adds[0])===JSON.stringify(adds[1]),"retrying a lost add acknowledgement preserves quantity and request ID");
  check(cart[0]?.quantity===1 && !await page.getByRole("dialog").count(),"one click adds from the list without opening a product or checkout");
  check(await badge.innerText()==="1","a confirmed quick add updates the header badge");
  check(await page.locator(".topbar .shop-cart-notice").isVisible(),"add confirmation stays in the header, clear of catalogue controls");
  await button("商品网格").click();
  const contained = selector => page.locator(selector).first().evaluate(el=>{const box=el.getBoundingClientRect(), parent=el.parentElement.getBoundingClientRect();return getComputedStyle(el).objectFit==="contain"&&box.left>=parent.left-1&&box.right<=parent.right+1&&box.bottom<=parent.bottom+1;});
  check(await contained(".shop-tile img")&&await contained(".shop-pick img"),"catalogue and recommended artwork stay inside their slots");
  await tiles.first().click();await button("查看第 2 张商品图").click();
  check(await contained(".product-main-image img")&&await contained(".product-thumbnails img"),"the portrait detail and landscape thumbnail both display the whole asset");
  await button("加入购物袋").click();await page.getByText("已加入购物袋",{exact:true}).waitFor();
  check(await badge.innerText()==="2","detail add counts units of the same product");
  expire=true;await button("立即购买").click();await button("更新价格").waitFor();check(!await button("确认付款").count(),"expired totals cannot be paid");
  failQuote=true;await button("更新价格").click();await button("重新计算金额").click();await button("确认付款").waitFor();
  check(prices.length===3&&payments.length===0&&await page.locator(".checkout-amount strong").innerText()==="11.00","automatic calculation, refresh and retry only show the server total");
  await button("关闭弹窗").click();await button("打开购物袋").click();await page.locator(".cart-product-image").waitFor();check(await contained(".cart-product-image"),"cart thumbnails contain the complete image");
  let releaseRead;
  const gotRead=new Promise(started=>{holdRead=()=>new Promise(resolve=>{releaseRead=resolve;started();});});
  await page.evaluate(()=>window.dispatchEvent(new Event("focus")));await gotRead;
  await button("增加蓝蘑菇床数量").click();await page.waitForFunction(()=>document.querySelector(".bag-count")?.textContent==="3");
  const oldResponse=page.waitForResponse(response=>new URL(response.url()).pathname==="/api/v1/store/cart");
  releaseRead();await oldResponse;await page.evaluate(()=>new Promise(requestAnimationFrame));
  check(await badge.innerText()==="3" && await page.locator(".bag-amount strong").innerText()==="36.00","an older read cannot overwrite the updated badge or subtotal");
  await page.getByRole("spinbutton",{name:"蓝蘑菇床数量"}).fill("9");await page.getByRole("spinbutton",{name:"蓝蘑菇床数量"}).press("Enter");
  check(await page.getByText("请输入 1 至 8 的整数").isVisible()&&cart[0].quantity===3,"invalid quantities preserve the confirmed cart");
  await button("减少蓝蘑菇床数量").click();await page.waitForFunction(()=>document.querySelector(".bag-count")?.textContent==="2");
  const desktopGap=await page.evaluate(()=>document.querySelector(".bag-summary").getBoundingClientRect().left-document.querySelector(".bag-items").getBoundingClientRect().right);
  check(desktopGap>=16,"desktop checkout is separated from the items");
  await page.setViewportSize({width:320,height:700});
  check(await page.evaluate(()=>{const rect=s=>document.querySelector(s).getBoundingClientRect();return rect(".bag-summary").top-rect(".bag-items").bottom>=16&&rect(".cart-stepper").right<rect(".bag-remove").left&&document.documentElement.scrollWidth<=innerWidth;}),"narrow checkout and quantity controls have distinct space");
  await page.setViewportSize({width:1440,height:900});await button("去结算").click();await button("确认付款").waitFor();
  check(prices.at(-1).source==="CART"&&payments.length===0,"shopping bag opens straight into the priced order");
  check(await contained(".checkout-line img"),"checkout uses an uncropped thumbnail");
  losePayment=true;await button("确认付款").click();await button("继续处理").waitFor();
  check(payments.length===1 && !await page.getByRole("heading",{name:"付款已确认"}).count(),"lost acknowledgement does not start another payment");
  cart.push({productId:"product_1",quantity:1,productVersion:1});cartVersion++;
  await button("继续处理").click();await page.getByRole("heading",{name:"付款已确认"}).waitFor();
  check(payments.length===2&&JSON.stringify(payments[0])===JSON.stringify(payments[1]),"explicit recovery uses the same order request and accepted quote");
  await button("关闭弹窗").click();
  await page.waitForFunction(()=>document.querySelector(".bag-count")?.textContent==="1");
  check(await page.locator(".bag-row h2").innerText()==="木床","paid cart refresh preserves an item added after checkout");
  await button("将木床移出购物袋").click();await page.getByText("购物袋还是空的",{exact:true}).waitFor();
  check(await badge.count()===0,"removing the last item clears the badge");
  await button("搜索商城商品").click();
  await page.waitForFunction(()=>document.activeElement===document.querySelector(".shop-search input"));
  check(new URL(page.url()).pathname==="/","cart search opens the catalogue with the input focused");
  products[0].content.limitPerOrder=150;
  cart=[{productId:"product_0",quantity:103,productVersion:1}];cartVersion++;
  await page.reload();await page.waitForFunction(()=>document.querySelector(".bag-count")?.textContent==="99+");
  check(await button("打开购物袋").getAttribute("aria-description")==="购物袋中有 103 件商品","reload hydrates the cart and keeps an accessible uncapped count");
  const transition = name => page.evaluate(async name=>{
    const sidebar=document.querySelector("#app-navigation"),main=document.querySelector(".main-shell");
    const start=sidebar.getBoundingClientRect().width,samples=[];
    sidebar.querySelector(`nav a[aria-label="${name}"]`).click();
    await new Promise(resolve=>{const began=performance.now();const frame=()=>{samples.push([sidebar.getBoundingClientRect().width,parseFloat(getComputedStyle(main).marginLeft)]);if(performance.now()-began<380)requestAnimationFrame(frame);else resolve();};requestAnimationFrame(frame);});
    const end=samples.at(-1)[0], low=Math.min(start,end)+1, high=Math.max(start,end)-1;
    return samples.some(([width,margin])=>width>low&&width<high&&Math.abs(width-margin)<2);
  },name);
  for (const name of ["信息","钱包","信息","商城"]) check(await transition(name), "navigation and content animate together toward "+name);
  if (await button("切换深色模式").isVisible()) await button("切换深色模式").click();
  check(await page.locator(".brand img").evaluate(el=>parseFloat(getComputedStyle(el).borderTopLeftRadius)>0),"dark sidebar logo has rounded corners");
  await page.emulateMedia({reducedMotion:"reduce"});
  await page.getByRole("link",{name:"信息",exact:true}).click();
  check(await page.locator(".chat-rail-actions .bag-count").innerText()==="99+","chat rail uses the shared red badge");
  check(await page.locator("#app-navigation").evaluate(el=>el.getAnimations().length===0),"reduced motion skips the sidebar transition");
  await page.emulateMedia({reducedMotion:"no-preference"});
  await page.setViewportSize({width:390,height:844});await page.goto(origin+"/");await picks.first().waitFor();
  check(await page.locator(".shop-products").evaluate(el=>getComputedStyle(el).gridTemplateColumns.split(" ").length===2)&&await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),"mobile uses two product columns and fits the viewport");
  check(await badge.innerText()==="99+","mobile top bar restores the cart badge");
  for(const name of ["信息","商城"]){
    await button("打开导航菜单").click();
    const moving=await page.evaluate(async label=>{
      document.querySelector(`#app-navigation nav a[aria-label="${label}"]`).click();
      await new Promise(requestAnimationFrame);
      return document.querySelector("#main-content").getAnimations().some(motion=>motion.effect.getKeyframes().some(frame=>Number(frame.opacity)===.45));
    },name);
    check(moving,"mobile page transition runs toward "+name);
  }
  products[3].effectivePrice="9007199254740993.67";
  cart=[{productId:"product_3",quantity:3,productVersion:1}];cartVersion++;
  await page.setViewportSize({width:320,height:700});await page.reload();await tiles.first().waitFor();
  check(await page.locator(".shop-tile-footer").evaluateAll(nodes=>nodes.every(node=>node.querySelector(".shop-price").getBoundingClientRect().right<=node.querySelector(".shop-add").getBoundingClientRect().left-3)),"long prices have their own space beside quick add at 320px");
  await button("打开购物袋").click();await page.getByText("27,021,597,764,222,981.01",{exact:true}).waitFor();
  check(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),"a subtotal beyond Number precision fits the narrow summary");
  await button("去结算").click();await button("确认付款").click();await page.getByRole("heading",{name:"付款已确认"}).waitFor();
  await page.waitForFunction(()=>!document.querySelector(".bag-count"));
  check(!cart.length,"checkout clearing the bag removes the badge before dismissing the receipt");
  await button("关闭弹窗").click();await page.getByText("购物袋还是空的",{exact:true}).waitFor();
  check(errors.length===0,"store path has no browser errors");
  return {passed,errors};
}

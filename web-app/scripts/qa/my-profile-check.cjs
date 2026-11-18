async page=>{
  const origin=new URL(page.url()).origin, passed=[], errors=[], patches=[], uploads=[], sentCodes=[], changes=[], revokes=[];
  const check=(ok,label)=>{if(!ok)throw new Error(label);passed.push(label);};
  const button=name=>page.getByRole("button",{name,exact:true}), link=name=>page.getByRole("link",{name,exact:true});
  page.on("pageerror",error=>errors.push(error.message));
  const user={userId:"profile-fixture-self",playerRef:"player_profile",gameId:"HaruRiver",displayName:"晴川",gameVerified:true,email:"haru@example.invalid",qq:"871234567",permissions:[]};
  let profile={...user,bio:"车站旁的花又开了。",version:1,avatar:null}, available=false, loseSave=false, conflict=false, loggedOut=false;
  let prefs={version:1,enabled:true,showPreviews:true,wallet:true}, devices=[{sessionId:"self",current:true,clientKind:"web",deviceName:"当前浏览器"},{sessionId:"phone",current:false,clientKind:"app",deviceName:"Android 手机"}].map(item=>({...item,lastSeenAt:new Date().toISOString()}));
  const imageUrl="data:image/svg+xml,"+encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"><rect width="32" height="32" fill="teal"/></svg>');
  const file={name:"avatar.png",mimeType:"image/png",buffer:Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jFuQAAAAASUVORK5CYII=","base64")};
  const saved=new Map();
  await page.routeWebSocket("**/api/v1/chat/ws",()=>{});
  await page.route("**/web-config.json",route=>route.fulfill({json:{configured:true,mode:"connected",version:"test"}}));
  await page.route("**/api/v1/**",async route=>{
    const req=route.request(),path=new URL(req.url()).pathname,method=req.method();
    const ok=data=>route.fulfill({json:{data,serverTime:new Date().toISOString()}});
    const fail=(status,code,message)=>route.fulfill({status,json:{error:{code,message}}});
    if(path.endsWith("/web/session")&&method==="DELETE"){loggedOut=true;return ok({});}
    if(path.endsWith("/web/session"))return loggedOut?fail(401,"UNAUTHORIZED","请登录"):ok({user,csrfToken:"profile-test-only",expiresAt:"2030-01-01T00:00:00Z"});
    if(path.includes("/players/"))return available?ok(profile):fail(503,"TEMPORARY","资料暂时无法读取");
    if(path.endsWith("/account/me/profile")){
      const body=req.postDataJSON();patches.push(body);
      if(saved.has(body.clientRequestId))return ok(saved.get(body.clientRequestId));
      if(conflict){conflict=false;profile={...profile,bio:"另一设备的新简介",version:profile.version+1};}
      if(body.expectedVersion!==profile.version)return fail(409,"STATE_VERSION_CONFLICT","版本已更新");
      profile={...profile,version:profile.version+1,...("bio" in body?{bio:body.bio}:{}),...(body.displayName?{displayName:body.displayName}:{}),...(body.avatarAssetId?{avatar:{assetId:body.avatarAssetId,url:imageUrl}}:{})};
      saved.set(body.clientRequestId,profile);
      if(loseSave){loseSave=false;return route.abort("connectionfailed");}
      return ok(profile);
    }
    if(path.endsWith("/assets/uploads")){uploads.push(req.postDataJSON());return ok({status:"READY",asset:{assetId:"asset_test",url:imageUrl}});}
    if(path.endsWith("/account/sessions"))return ok({items:devices});
    if(path.endsWith("/account/sessions/phone") && method==="DELETE"){revokes.push(path);devices=devices.filter(d=>d.current);return ok({});}
    if(path.endsWith("/account/password-change-code")){sentCodes.push(method);return ok({verificationToken:"fixture-code-token",delivery:"EMAIL",resendAfterSeconds:60});}
    if(path.endsWith("/account/password-change")){changes.push(req.postDataJSON());return ok({});}
    if(path.endsWith("/notifications/preferences")){
      if(method==="PATCH"){const body=req.postDataJSON();if(body.expectedVersion!==prefs.version)return fail(409,"STATE_VERSION_CONFLICT","版本已更新");prefs={...prefs,wallet:body.wallet,version:prefs.version+1};}
      return ok(prefs);
    }
    if(path.endsWith("/store/cart"))return ok({version:1,items:[{productId:"p",quantity:3}]});
    if(path.endsWith("/account/deletions")||path.endsWith("/chat/player-names"))return ok({items:[],cursor:0,hasMore:false});
    if(path.endsWith("/wallet/records"))return ok({records:[]});
    if(path.endsWith("/wallet/balance"))return ok({balance:{amount:"120.00",fresh:true}});
    if(path.endsWith("/merchant/me"))return ok({storeIds:[]});
    if(path.endsWith("/support/contact"))return ok({email:"help@example.invalid"});
    return ok([]);
  });
  await page.setViewportSize({width:1440,height:900});await page.goto(origin+"/me");await button("重试").waitFor();
  check(await button("编辑资料").isDisabled()&&await link("查看钱包").isVisible()&&await button("登录设备").isEnabled()&&await button("退出登录").isEnabled(),"unavailable profile preserves services and account actions");
  available=true;await button("重试").click();await page.locator(".me-bio").waitFor();
  check(await page.locator(".me-identity").evaluate(el=>getComputedStyle(el).backgroundImage.includes("gradient") && getComputedStyle(el,"::before").pointerEvents==="none") && await page.locator(".me-verified").isVisible(),"theme-aware card background leaves identity controls accessible");
  const account=page.getByRole("region",{name:"账号信息",exact:true});
  check(!(await account.innerText()).includes(user.email)&&!(await account.innerText()).includes(user.qq),"contact values are masked initially");
  await button("显示联系方式").click();check((await account.innerText()).includes(user.email)&&(await account.innerText()).includes(user.qq),"contact reveal uses the account values");
  await button("隐藏联系方式").click();
  await page.evaluate(()=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async text=>{window.fixtureCopied=text;}}}));
  await button("复制 Deuterium ID").click();check(await page.evaluate(()=>window.fixtureCopied)==="profile-fixture-self","copy uses the full account ID");
  await button("编辑资料").click();await page.getByRole("textbox",{name:"昵称",exact:true}).fill("林间风");conflict=true;await button("保存").click();
  await page.getByRole("alert").filter({hasText:"其他设备"}).waitFor();
  check(await page.getByRole("textbox",{name:"昵称",exact:true}).inputValue()==="林间风"&&await page.getByRole("textbox",{name:"个人简介",exact:true}).inputValue()==="另一设备的新简介","version conflict preserves the draft and updates untouched fields");
  loseSave=true;await button("保存").click();await page.getByRole("alert").filter({hasText:"无法连接"}).waitFor();await button("保存").click();
  await page.locator(".me-name h2").filter({hasText:"林间风"}).waitFor();
  check(patches.length===3&&JSON.stringify(patches[1])===JSON.stringify(patches[2])&&!("bio" in patches[1]),"lost acknowledgement retries the identical nickname change");
  check(await page.locator(".sidebar-profile strong").innerText()==="林间风","saved nickname updates the sidebar");
  await page.locator('.me-identity input[type=file]').setInputFiles({name:"bad.txt",mimeType:"text/plain",buffer:Buffer.from("bad")});
  await page.getByRole("alert").filter({hasText:"20 MB"}).waitFor();
  check(await button("使用新头像").isDisabled()&&uploads.length===0,"invalid avatars cannot be submitted");await button("关闭弹窗").click();
  await page.locator('.me-identity input[type=file]').setInputFiles(file);await page.getByRole("img",{name:"新头像预览"}).waitFor();
  await button("关闭弹窗").click();check(uploads.length===0,"cancelling the avatar preview sends nothing");
  await page.locator('.me-identity input[type=file]').setInputFiles(file);loseSave=true;await button("使用新头像").click();await page.getByRole("alert").filter({hasText:"无法连接"}).waitFor();await button("使用新头像").click();await page.getByText("头像已更新",{exact:true}).waitFor();
  check(JSON.stringify(patches.at(-1))===JSON.stringify(patches.at(-2)) && uploads.length===1,"retrying avatar attachment reuses the uploaded asset and mutation ID");
  check(uploads.length===1&&uploads[0].purpose==="AVATAR"&&uploads[0].businessRef===user.userId&&await page.locator(".sidebar-profile img.avatar").count()===1,"avatar attaches to the current user and updates both views");
  await button("登录设备").click();await page.getByText("当前设备",{exact:true}).waitFor();await page.getByRole("dialog").getByRole("button",{name:"退出",exact:true}).click();
  check(revokes.length===0,"viewing or selecting a device does not revoke it");await button("确认退出").click();await page.getByText("已退出所选设备。",{exact:true}).waitFor();
  check(devices.length===1&&devices[0].current,"device revocation keeps the current session");await button("关闭弹窗").click();
  await button("更改密码").click();check(sentCodes.length===0&&changes.length===0,"opening password settings does not send a code");
  await button("发送验证码").click();await page.getByRole("textbox",{name:"邮箱验证码"}).fill("123456");
  await page.getByLabel("新密码",{exact:true}).fill("fixture-pass-A");await page.getByLabel("再次输入新密码",{exact:true}).fill("fixture-pass-B");await button("确认更改密码").click();
  check(await page.getByText("两次输入的密码不一致。",{exact:true}).isVisible()&&changes.length===0,"mismatched passwords remain local");await button("关闭弹窗").click();
  await link("通知设置").click();await page.getByRole("switch",{name:/钱包/}).click();await page.waitForFunction(()=>document.querySelectorAll(".me-pref-group").length===3&&!document.querySelector("fieldset:disabled")&&Array.from(document.querySelectorAll(".toggle-row")).some(row=>row.querySelector("strong")?.textContent==="钱包"&&!row.querySelector("input").checked));
  check(prefs.wallet===false&&prefs.enabled,"changing the wallet notification leaves the master preference intact");await button("返回我的").click();
  check(await link("购物袋").getAttribute("aria-description")==="共 3 件商品","cart shortcut shares the existing cart count");
  for(const [label,path] of [["查看钱包","/wallet"],["我的订单","/orders"],["我的商店","/merchant"],["邮件支持","/support"]]){
    await link(label).click();check(new URL(page.url()).pathname===path,"service link reaches "+path);await page.goBack();await page.locator(".me-name h2").waitFor();
  }
  profile={...profile,displayName:"很长的玩家昵称".repeat(3),bio:"沿着主城的小路散步。".repeat(17),version:profile.version+1};await button("刷新资料").click();
  await page.locator(".me-name h2").filter({hasText:profile.displayName}).waitFor();
  for(const width of [1840,1024,768,390,320]){
    await page.setViewportSize({width,height:844});
    check(await page.evaluate(()=>{const box=s=>document.querySelector(s).getBoundingClientRect();return document.documentElement.scrollWidth===innerWidth&&box(".me-avatar-button > .avatar").right<=box(".me-intro").left-2;}),"avatar, long names and bio fit at "+width+"px");
  }
  if(await button("切换深色模式").isVisible())await button("切换深色模式").click();
  check(await page.evaluate(()=>getComputedStyle(document.querySelector(".me-identity")).borderRadius!=="0px"),"dark identity card keeps its rounded outline");
  await button("退出登录").click();await page.getByRole("button",{name:"登录",exact:true}).first().waitFor();
  check(loggedOut,"logout reaches the authenticated session endpoint");
  check(errors.length===0,"no browser runtime errors");return {passed,errors};
}

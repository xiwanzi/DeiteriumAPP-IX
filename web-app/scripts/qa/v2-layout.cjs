async(page)=>{
 const issues=[],results=[],errors=[];page.on('pageerror',e=>errors.push(e.message));
 const shot=async(name)=>{await page.evaluate(async()=>{await Promise.all([...document.images].map(i=>i.decode().catch(()=>{})));await Promise.all(document.getAnimations().map(a=>a.finished.catch(()=>{})));});const box=await page.evaluate(()=>({width:innerWidth,scroll:document.documentElement.scrollWidth,height:innerHeight,pageHeight:document.documentElement.scrollHeight,broken:[...document.images].filter(i=>i.complete&&!i.naturalWidth).map(i=>i.src)}));if(box.scroll>box.width+1)issues.push(name+' horizontal overflow');if(box.broken.length)issues.push(name+' missing media');results.push({name,...box});await page.screenshot({path:`output/playwright/${name}.png`});};
 for(const width of [1440,768,390,360]){
  await page.setViewportSize({width,height:width===1440?1000:844});
  await page.goto('http://127.0.0.1:5180/login');await page.getByRole('textbox',{name:'游戏 ID / QQ'}).waitFor();await shot(`v2-login-${width}`);
  if(width===1440||width===390){await page.getByRole('button',{name:'创建账号',exact:true}).click();await shot(`v2-register-${width}`);await page.getByRole('button',{name:'返回登录',exact:true}).click();await page.getByRole('button',{name:'忘记密码？'}).click();await shot(`v2-reset-${width}`);}
  for(const route of ['/','/market','/commissions','/wallet','/me']){await page.goto('http://127.0.0.1:5180'+route);await page.locator('main').waitFor();await shot(`v2-${route.slice(1)||'store'}-${width}`);}
  await page.goto('http://127.0.0.1:5180/information');await page.getByRole('textbox',{name:'搜索会话'}).waitFor();await shot(`v2-chat-list-${width}`);if(width<768){await page.locator('.pc-contact').filter({hasText:'公共聊天'}).click();await shot(`v2-chat-thread-${width}`);}
  await page.goto('http://127.0.0.1:5180/admin');await page.getByRole('button',{name:'玩家销售',exact:true}).click();await shot(`v2-admin-sales-${width}`);await page.getByRole('button',{name:'交易明细',exact:true}).click();await shot(`v2-admin-transactions-${width}`);await page.getByRole('button',{name:'公告管理',exact:true}).click();await shot(`v2-admin-announcements-${width}`);
 }
 await page.setViewportSize({width:1440,height:1000});await page.getByRole('button',{name:'切换深色模式'}).click();
 for(const route of ['/login','/information','/admin']){await page.goto('http://127.0.0.1:5180'+route);if(route==='/admin')await page.getByRole('button',{name:'玩家销售',exact:true}).click();await shot(`v2-${route.slice(1)}-dark`);}
 await page.getByRole('button',{name:'切换浅色模式'}).click();await page.goto('http://127.0.0.1:5180/login');
 if(issues.length||errors.length)throw new Error(JSON.stringify({issues,errors}));return {checks:results.length,results,issues,errors};
}

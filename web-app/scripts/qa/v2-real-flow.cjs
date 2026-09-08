async(page)=>{
 const passed=[],errors=[];page.on('pageerror',e=>errors.push(e.message));const check=(x,name)=>{if(!x)throw new Error(name);passed.push(name);};
 await page.goto('http://127.0.0.1:5182/information');
 if(await page.getByRole('textbox',{name:'游戏 ID / QQ'}).count()){
  await page.getByRole('textbox',{name:'游戏 ID / QQ'}).fill('WebMori');await page.getByRole('textbox',{name:'密码',exact:true}).fill('migration-password-123');await page.getByRole('button',{name:'登录',exact:true}).click();
 }
 await page.getByText('实时连接正常',{exact:true}).waitFor();
 check(await page.locator('.pc-message .bubble').filter({hasText:'Core 模拟节点'}).count()===1,'Go/Core public history received');
 const cookies=await page.context().cookies();const sessionCookie=cookies.find(c=>c.name==='deuterium_dev_session');check(Boolean(sessionCookie?.httpOnly),'development session is HttpOnly');check(sessionCookie?.sameSite==='Lax','session cookie SameSite Lax');
 check(await page.evaluate(()=>!document.cookie.includes('deuterium_dev_session')),'session not readable by JavaScript');
 check(await page.evaluate(()=>!JSON.stringify({...localStorage}).includes('csrfToken')&&!JSON.stringify({...localStorage}).includes('migration-password-123')),'credentials and CSRF absent from localStorage');
 const content='真实链路验收 '+Date.now();await page.getByRole('textbox',{name:'消息内容'}).fill(content);await page.getByRole('button',{name:'发送',exact:true}).click();
 await page.locator('.pc-message .bubble').filter({hasText:content}).waitFor();await page.locator('.message-delivery').filter({hasText:'已发送'}).first().waitFor();
 check(await page.locator('.pc-message .bubble').filter({hasText:content}).count()===1,'WebSocket send acknowledgement and event deduplicated');
 await page.reload();await page.getByText('实时连接正常',{exact:true}).waitFor();await page.locator('.pc-message .bubble').filter({hasText:content}).waitFor();check(true,'Cookie restores identity and SQL chat history after refresh');
 const csrfRejected=await page.request.delete('http://127.0.0.1:5182/api/v1/web/session',{headers:{Origin:'http://127.0.0.1:5182','X-CSRF-Token':'incorrect-test-csrf'}});check(csrfRejected.status()===403,'wrong CSRF rejected by actual backend');
 const originRejected=await page.request.post('http://127.0.0.1:5182/api/v1/web/session',{headers:{Origin:'https://untrusted.example'},data:{account:'WebMori',password:'migration-password-123'}});check(originRejected.status()===403,'untrusted login Origin rejected');
 await page.getByRole('link',{name:'官方管理',exact:true}).click();await page.getByText('webtest',{exact:true}).first().waitFor();await page.getByText('测试材料包 · 元数据示例',{exact:true}).waitFor();check(true,'authorised Core nodes and item version read from Go backend');
 await page.getByRole('link',{name:'钱包',exact:true}).click();check(await page.getByRole('heading',{name:'钱包服务尚未接入'}).count()===1,'unimplemented wallet never falls back to demo money');
 await page.getByRole('link',{name:'信息',exact:true}).click();await page.getByText('实时连接正常',{exact:true}).waitFor();
 await page.evaluate(()=>{window.testRevokedClosed=false;window.testExtraWs=new WebSocket('ws://127.0.0.1:5182/api/v1/chat/ws');window.testExtraWs.onclose=()=>window.testRevokedClosed=true;});await page.waitForFunction(()=>window.testExtraWs.readyState===1);
 await page.getByRole('link',{name:'我的',exact:true}).click();await page.getByRole('button',{name:'退出登录',exact:true}).click();await page.getByRole('textbox',{name:'游戏 ID / QQ'}).waitFor();await page.waitForFunction(()=>window.testRevokedClosed,{},{timeout:8000});check(true,'logout revokes actual session and closes an existing WebSocket');
 const after=await page.request.get('http://127.0.0.1:5182/api/v1/web/session');check(after.status()===401,'revoked session rejected');
 await page.getByRole('textbox',{name:'游戏 ID / QQ'}).fill('WebMori');await page.getByRole('textbox',{name:'密码',exact:true}).fill('incorrect-test-password');await page.getByRole('button',{name:'登录',exact:true}).click();await page.getByRole('alert').waitFor();check((await page.getByRole('alert').textContent()).includes('账号或密码错误'),'real credential error displayed inline');
 await page.getByRole('button',{name:'忘记密码？'}).click();await page.getByRole('textbox',{name:'游戏 ID',exact:true}).fill('WebMori');await page.getByRole('button',{name:'获取重置验证码',exact:true}).click();await page.getByRole('alert').waitFor();check((await page.getByRole('alert').textContent()).includes('尚未接入'),'unsupported reset uses actual capability error');
 await page.getByRole('button',{name:'返回登录',exact:true}).click();await page.getByRole('textbox',{name:'游戏 ID / QQ'}).fill('WebMori');await page.getByRole('textbox',{name:'密码',exact:true}).fill('migration-password-123');await page.getByRole('button',{name:'登录',exact:true}).click();await page.getByText('实时连接正常',{exact:true}).waitFor();
 await page.screenshot({path:'output/playwright/v2-connected-chat-final.png'});check(errors.length===0,'no runtime exception through live authentication/chat/logout');
 return {count:passed.length,passed,errors};
}

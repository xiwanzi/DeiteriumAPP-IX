async(page)=>{
 await page.goto('http://127.0.0.1:5182/information');await page.getByText('实时连接正常',{exact:true}).waitFor();
 const nodes=await page.request.get('http://127.0.0.1:5182/api/v1/admin/core/nodes');if((await nodes.json()).data.nodes.some(n=>n.online))throw new Error('Core simulator should be disconnected');
 await page.evaluate(()=>{window.testSentIds=[];window.originalSocketSend=WebSocket.prototype.send;WebSocket.prototype.send=function(raw){try{const f=JSON.parse(raw);if(f.type==='chat.send')window.testSentIds.push(f.payload.clientMessageId);}catch{}return window.originalSocketSend.call(this,raw);};window.testOfflineContent='离线恢复验收 '+Date.now();});
 const content=await page.evaluate(()=>window.testOfflineContent);await page.getByRole('textbox',{name:'消息内容'}).fill(content);await page.getByRole('button',{name:'发送',exact:true}).click();await page.locator('.message-failed').filter({hasText:content}).waitFor();
 const row=page.locator('.pc-message').filter({hasText:content});const failed=(await row.textContent()).includes('服务器连接暂不可用');const preserved=await page.getByRole('textbox',{name:'消息内容'}).inputValue()===content;if(!failed||!preserved)throw new Error('Failed send incorrectly acknowledged or draft lost');
 return {offlineCoreDetected:true,explicitFailure:failed,draftPreserved:preserved};
}

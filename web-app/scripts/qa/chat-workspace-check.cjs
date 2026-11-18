async (page) => {
  const passed = [], errors = [], sent = [], sockets = [];
  let pendingAck;
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  page.on("pageerror", error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem("deuterium-web-theme", "light"));
  const now = new Date(), stamp = minutes => new Date(now.getTime() - minutes * 60000).toISOString();
  const self = { userId: "fixture-self", playerRef: "player_self", gameId: "SelfPlayer", displayName: "晴川", gameVerified: true, permissions: [] };
  const peers = [
    {playerRef:"player_lin",gameId:"Lin",displayName:"林间风",bio:"",avatar:null,registered:true},
    {playerRef:"player_island",gameId:"Island",displayName:"小岛",bio:"",avatar:null,registered:true},
    {playerRef:"player_blue",gameId:"Blue",displayName:"青禾",bio:"",avatar:null,registered:true},
  ];
  const message = (messageId, sender, content, ago, extra={}) => ({messageId, sender, content, sentAt:stamp(ago), ...extra});
  const publicMessages = [
    message("public_1",peers[0],"有人在主城吗？河边的新步道已经修好啦。",15),
    message("public_2",peers[0],"从车站出来往右走，沿着路灯就能看到。",14),
    message("public_3",peers[1],"刚到！傍晚的光线真好看。",12),
    message("public_4",self,"我也过来看看，顺便带点灯笼。",11),
    message("public_5",self,"桥边还需要补几盏吗？",10),
    message("public_6",peers[2],"需要的，靠近花田那一侧还留了位置。",8),
    message("public_7",peers[1],"等大家都到了拍张合照吧。",7),
    message("public_8",peers[1],"今天的落日一定很漂亮。",6),
    message("public_9",self,"好，马上到。",5),
  ];
  const privateMessages = [message("private_1",peers[0],"周末一起把车站旁边的小花园补完？",20,{conversationId:"conv_lin"})];
  const conversations = () => peers.map((p,i)=>({conversationId:["conv_lin","conv_island","conv_blue"][i],otherPlayer:p,lastMessage:i===0?privateMessages.at(-1):message(`summary_${i}`,p,["","材料已经整理好了，谢谢！","周末一起去探索吧"][i],22+i),unreadCount:i===2?3:0,updatedAt:stamp(20+i),version:1}));
  await page.routeWebSocket("**/api/v1/chat/ws", socket => {
    sockets.push(socket);
    socket.onMessage(async raw => {
      const frame=JSON.parse(raw); if(frame.type!=="chat.send")return;
      sent.push({channel:"public",...frame.payload});
      if (pendingAck) await pendingAck;
      const accepted=message(`sent_${sent.length}`,self,frame.payload.content,0,{clientMessageId:frame.payload.clientMessageId});
      publicMessages.push(accepted);
      socket.send(JSON.stringify({type:"chat.send.result",requestId:frame.requestId,payload:{status:"accepted",messageId:accepted.messageId}}));
      socket.send(JSON.stringify({type:"chat.message",payload:{message:accepted}}));
    });
  });
  await page.route("**/web-config.json", route => route.fulfill({json:{mode:"connected",version:"2.0.44",configured:true}}));
  await page.route("**/api/v1/**", async route => {
    const request=route.request(), path=new URL(request.url()).pathname;
    const reply = data => route.fulfill({json:{data,page:{hasMore:false,nextCursor:null},serverTime:now.toISOString()}});
    if(path==="/api/v1/web/session")return reply({user:self,csrfToken:"isolated-fixture-csrf",expiresAt:"2030-01-01T00:00:00Z"});
    if(path.startsWith("/api/v1/players/"))return reply({...([self,...peers].find(p=>p.playerRef===path.split("/").at(-1))||peers[0]),version:1});
    if(path==="/api/v1/account/deletions"||path==="/api/v1/chat/player-names")return reply({items:[],cursor:0,hasMore:false});
    if(path==="/api/v1/chat/conversations")return reply(conversations());
    if(path.endsWith("/messages")&&path.includes("/conversations/")) {
      if(request.method()==="POST") {
        const body=request.postDataJSON();sent.push({channel:"conv_lin",...body});
        const accepted=message(`sent_${sent.length}`,self,body.content,0,{conversationId:"conv_lin",clientMessageId:body.clientMessageId});
        privateMessages.push(accepted);return reply(accepted);
      }
      return reply(path.includes("conv_lin")?privateMessages.slice().reverse():[]);
    }
    if(path.endsWith("/read"))return reply({unreadCount:0});
    if(path.startsWith("/api/v1/chat/conversations/"))return reply(conversations().find(c=>path.endsWith(c.conversationId)));
    if(path==="/api/v1/chat/messages") {
      if (new URL(request.url()).searchParams.has('before')) return reply({messages:[message('older_fixture',peers[1],'早上的第一条记录。',60)]});
      return route.fulfill({json:{data:{messages:publicMessages.slice().reverse()},page:{hasMore:true,nextCursor:'older_fixture'},serverTime:now.toISOString()}});
    }
    if(path==="/api/v1/chat/stickers")return reply({stickers:[]});
    if(path==="/api/v1/notifications")return reply([]);
    if(path==="/api/v1/chat/follows"||path==="/api/v1/chat/player-directory")return reply({players:peers});
    return reply({});
  });
  await page.setViewportSize({width:2480,height:1284});
  await page.goto("http://127.0.0.1:5286/information");
  await page.locator(".bubble").filter({hasText:"好，马上到。"}).waitFor();
  const editor=page.getByRole("textbox",{name:"消息内容",exact:true});
  const contact=name=>page.locator(".pc-contact").filter({has:page.getByText(name,{exact:true})});
  const layout=await page.evaluate(()=>{
    const box=s=>document.querySelector(s).getBoundingClientRect();
    return {rail:box('.sidebar').width,list:box('.pc-conversations').width,header:box('.pc-thread-header').width,headerHeight:box('.pc-thread-header').height,composer:box('.chat-composer-capsule').width,avatar:box('.pc-contact .conversation-symbol').width,topbar:getComputedStyle(document.querySelector('.topbar')).display,overflow:document.documentElement.scrollWidth>innerWidth};
  });
  check(layout.rail===80&&layout.list===500&&layout.header===1120&&layout.composer===layout.header&&!layout.overflow&&layout.topbar==='none',"desktop at normal browser zoom matches the reference rail, list and reading column");
  const messageFont = () => page.locator('.bubble').first().evaluate(el=>parseFloat(getComputedStyle(el).fontSize));
  check(layout.headerHeight===62.5 && layout.avatar===57.5 && await messageFont()===25, "desktop typography, header and avatars match the 125% reference");
  await page.setViewportSize({width:1840,height:922});
  const wideFont = await messageFont();
  await page.getByRole('button',{name:'展开会话详情'}).click();
  check(wideFont>20 && await messageFont()<wideFont, "opening details adapts text to the remaining thread width");
  await page.getByRole('button',{name:'关闭会话详情'}).click();
  check(await page.locator('.pc-details').count()===0,"conversation details start collapsed");
  check(await page.locator('.public-thread .group-continues > .avatar-button').first().evaluate(el=>getComputedStyle(el).visibility)==='hidden',"group avatar appears at the end of consecutive messages");
  await page.screenshot({path:"output/playwright/chat-workspace-light.png"});
  await page.getByRole("button",{name:"切换深色模式",exact:true}).click();
  await page.screenshot({path:"output/playwright/chat-workspace-dark.png"});
  await page.getByRole("button",{name:"未读",exact:true}).click();
  check(await page.locator('.pc-contact').count()===1&&await contact('青禾').count()===1,"unread filter uses actual unread counts");
  await page.getByRole("button",{name:"私聊",exact:true}).click();
  check(await page.locator('.pc-contact').count()===3,"private filter excludes public chat and AI");
  await page.getByRole("textbox",{name:"搜索会话",exact:true}).fill('林间');
  check(await page.locator('.pc-contact').count()===1,"conversation search combines with the selected filter");
  await page.getByRole("button",{name:"清除会话搜索",exact:true}).click();
  await page.getByRole("button",{name:"全部",exact:true}).click();
  await editor.fill('公共频道草稿');
  await contact('林间风').click();
  await page.locator('.bubble').filter({hasText:'周末一起把车站旁边的小花园补完？'}).waitFor();
  check(await editor.inputValue()==='',"public draft does not leak into private conversation");
  await editor.fill('私聊第一行');await editor.press('Shift+Enter');await editor.type('第二行');
  const privateDraft=await editor.inputValue();
  check(privateDraft==='私聊第一行\n第二行',"Shift Enter grows the private composer without sending");
  await contact('公共聊天').click();
  check(await editor.inputValue()==='公共频道草稿',"switching back restores the public draft");
  await editor.fill('');
  await page.locator('.pc-message').filter({has:page.getByRole('button',{name:'回复消息 public_6',exact:true})}).hover();
  await page.getByRole('button',{name:'回复消息 public_6',exact:true}).click();
  await editor.fill('回复测试');await editor.press('Enter');
  await page.locator('.bubble').filter({hasText:'回复测试'}).waitFor();
  check(sent.at(-1).replyToMessageId==='public_6'&&sent.at(-1).channel==='public',"reply sends the original referenced message ID to the selected public channel");
  check(await editor.inputValue()==='',"successful send clears the submitted draft");
  check(await page.getByLabel('已发送',{exact:true}).count()>0,"accepted messages display sent status without invented read receipts");
  await editor.focus();
  check(await editor.evaluate(el=>getComputedStyle(el).outlineStyle==='none'&&getComputedStyle(el).boxShadow==='none'),"composer has no blue focus outline or inset ring");
  const composerBefore=await page.locator('.chat-composer-capsule').boundingBox();
  const motion=await page.evaluate(async()=>{
    document.querySelector('[aria-label="选择表情"]').click();
    const samples=[];
    await new Promise(resolve=>{const start=performance.now();function frame(){const el=document.querySelector('.chat-expression-viewport'),style=getComputedStyle(el);samples.push({at:performance.now()-start,opacity:Number(style.opacity),transform:style.transform});if(performance.now()-start<320)requestAnimationFrame(frame);else resolve();}requestAnimationFrame(frame);});
    return samples;
  });
  check(motion.some(s=>s.opacity>0&&s.opacity<1)&&motion.at(-1).opacity===1,"emoji popover has a real intermediate fade and scale animation");
  const composerAfter=await page.locator('.chat-composer-capsule').boundingBox(), menu=await page.locator('.chat-expression-viewport.open').boundingBox();
  check(Math.abs(composerBefore.y-composerAfter.y)<1&&menu.y+menu.height<=composerAfter.y&&menu.width<=480,"emoji menu floats above the input without moving its position");
  let vectorEmojiCount=0;
  for(const name of ['表情','手势','心情','自然','美食','活动']) {
    await page.getByRole('tab',{name,exact:true}).click();
    await page.waitForFunction(()=>{const buttons=[...document.querySelectorAll('.chat-emoji-grid button')];return buttons.length>0&&buttons.every(b=>{const img=b.querySelector('img');return img&&img.complete&&img.naturalWidth>0&&img.src.endsWith('.svg');});});
    vectorEmojiCount+=await page.locator('.chat-emoji-grid img').count();
  }
  check(vectorEmojiCount===259,"all 259 selectable emoji load local SVG artwork");
  check(await page.locator('.chat-emoji-categories svg').count()===7,"category controls use scalable SVG icons");
  await page.getByRole('tab',{name:'心情',exact:true}).click();
  await page.screenshot({path:'output/playwright/chat-workspace-emoji-dark.png',animations:'disabled'});
  await page.getByRole('button',{name:'切换键盘',exact:true}).click();
  await page.waitForFunction(()=>getComputedStyle(document.querySelector('.chat-expression-viewport')).visibility==='hidden');
  check(await editor.evaluate(el=>el===document.activeElement),"closing expressions restores keyboard focus");
  await editor.fill('前后');await editor.evaluate(el=>el.setSelectionRange(1,1));
  await page.getByRole('button',{name:'选择表情',exact:true}).click();
  await page.getByRole('button',{name:'❤️‍🔥',exact:true}).click();
  check(await editor.inputValue()==='前❤️‍🔥后',"emoji inserts at the existing caret without replacing surrounding text");
  await page.getByRole('button',{name:'删除表情',exact:true}).click();
  check(await editor.inputValue()==='前后',"emoji delete removes the entire joined glyph");
  await page.getByRole('heading',{name:'公共聊天',exact:true}).click();
  check(await page.locator('.chat-expression-viewport.open').count()===0,"clicking outside closes the emoji popover");
  await editor.fill('输入法组合中');const sendsBeforeComposition=sent.length;
  await editor.dispatchEvent('keydown',{key:'Enter',code:'Enter',isComposing:true});
  check(sent.length===sendsBeforeComposition&&await editor.inputValue()==='输入法组合中',"IME confirmation Enter does not accidentally send");
  await page.locator('.chat-composer-capsule input[type=file]').setInputFiles('web-app/public/media/product_ryzen9950_1.png');
  await page.getByRole('dialog',{name:'发送图片',exact:true}).waitFor();
  await page.getByRole('button',{name:'取消',exact:true}).click();
  await page.getByRole('dialog',{name:'发送图片',exact:true}).waitFor({state:'hidden'});
  check(await editor.inputValue()==='输入法组合中',"cancelling an image preview preserves the text draft");
  await editor.fill('');
  await contact('林间风').click();
  check(await editor.inputValue()===privateDraft,"private multiline draft is restored independently");
  await page.getByRole('button',{name:'选择表情',exact:true}).click();
  check(await page.locator('.chat-emoji-grid button').count()>10&&await page.getByRole('tab',{name:/贴纸/}).count()===0,"private chats use the complete emoji picker without exposing unsupported sticker sending");
  await page.getByRole('button',{name:'切换键盘',exact:true}).click();
  await editor.press('Enter');await page.locator('.bubble').filter({hasText:'私聊第一行'}).waitFor();
  check(sent.at(-1).channel==='conv_lin'&&sent.at(-1).content===privateDraft,"private send preserves multiline content and conversation destination");
  await page.getByRole('button',{name:'展开会话详情',exact:true}).click();
  check(await page.locator('.pc-details').isVisible(),"details remain available on demand");
  await page.getByRole('button',{name:'关闭会话详情',exact:true}).click();
  await contact('公共聊天').click();
  for(const width of [1280,1024,768,390]) {
    await page.setViewportSize({width,height:850});
    if(width===390&&await contact('公共聊天').isVisible())await contact('公共聊天').click();
    if(width===390)await page.waitForFunction(()=>document.querySelector('.sidebar').getBoundingClientRect().right<=1);
    check(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`no horizontal overflow at ${width}px`);
    const bounds=await editor.boundingBox();check(bounds&&bounds.x>=0&&bounds.x+bounds.width<=width&&bounds.y+bounds.height<=850,`composer stays visible at ${width}px`);
  }
  await page.screenshot({path:"output/playwright/chat-workspace-mobile-dark.png",animations:'disabled'});
  check(await messageFont()===16 && (await editor.boundingBox()).width>=220, "390px phones preserve 16px text and room to type beside the send controls");
  await page.getByRole('button',{name:'返回会话列表',exact:true}).click();
  check(await contact('公共聊天').isVisible(),"mobile back returns to the conversation list");
  await page.getByRole('button',{name:'打开导航菜单',exact:true}).click();
  check(await page.getByRole('link',{name:'商城',exact:true}).isVisible(),"mobile navigation still exposes existing features");
  await page.getByRole('button',{name:'关闭导航菜单',exact:true}).click({position:{x:300,y:400}});
  await page.getByRole('button',{name:'切换浅色模式',exact:true}).click();
  await contact('公共聊天').click();
  await page.screenshot({path:"output/playwright/chat-workspace-mobile-light.png",animations:'disabled'});
  await page.emulateMedia({reducedMotion:'reduce'});
  await page.getByRole('button',{name:'选择表情',exact:true}).click();
  check(await page.locator('.chat-expression-viewport').evaluate(el=>getComputedStyle(el).transitionDuration==='0s'),"reduced-motion preference removes popover transitions");
  await page.getByRole('button',{name:'切换键盘',exact:true}).click();
  await page.emulateMedia({reducedMotion:'no-preference'});
  await page.setViewportSize({width:1840,height:922});
  await page.getByRole('button',{name:'切换深色模式',exact:true}).click();
  let releaseAck;
  pendingAck = new Promise(resolve => { releaseAck = resolve; });
  await editor.fill('等待回执的消息');
  await editor.press('Enter');
  await page.getByLabel('发送中',{exact:true}).waitFor();
  await contact('林间风').click();
  await editor.fill('留在私聊的新草稿');
  pendingAck = null;
  releaseAck();
  await contact('公共聊天').getByText('你：等待回执的消息',{exact:true}).waitFor();
  check(await editor.inputValue()==='留在私聊的新草稿','late acknowledgement clears only the original public draft');
  await contact('公共聊天').click();
  check(await editor.inputValue()==='', 'acknowledged background draft is cleared when reopening its thread');

  pendingAck = new Promise(resolve => { releaseAck = resolve; });
  await editor.fill('正在发送的第一句');
  await editor.press('Enter');
  await page.getByLabel('发送中',{exact:true}).waitFor();
  await editor.fill('已经开始写第二句');
  pendingAck = null;
  releaseAck();
  await page.getByLabel('发送中',{exact:true}).waitFor({state:'hidden'});
  check(await editor.inputValue()==='已经开始写第二句','typing the next draft survives the previous send completing');
  await editor.fill('');

  const incoming = (messageId, content, extra={}) => sockets.at(-1).send(JSON.stringify({type:'chat.message',payload:{message:{
    ...message(messageId,peers[0],content,0),sentAt:new Date().toISOString(),...extra,
  }}}));
  incoming('quoted_fixture','线路编号 [IX-42] 已记下。',{reply:{messageId:'public_2',sender:peers[0],content:publicMessages[1].content}});
  await page.locator('[data-message-id="quoted_fixture"]').waitFor();
  await page.getByRole('button',{name:'搜索当前会话消息'}).click();
  await page.getByRole('textbox',{name:'搜索已加载消息'}).fill('[ix-42]');
  check(await page.locator('.bubble mark').innerText()==='[IX-42]','search highlights literal punctuation and preserves the original case');
  await page.keyboard.press('Escape');
  check(await page.getByRole('button',{name:'搜索当前会话消息'}).evaluate(el=>el===document.activeElement),'closing search returns keyboard focus to the search control');
  await page.setViewportSize({width:390,height:700});
  await page.getByRole('button',{name:/^查看原消息/}).click();
  check(await page.locator('[data-message-id="public_2"][data-located]').count()===1,'tapping a quotation locates its loaded original');
  const jump = page.getByRole('button',{name:'回到最新消息',exact:true});
  await jump.waitFor();
  const readingPosition = await page.locator('.pc-message-scroll').evaluate(el=>el.scrollTop);
  await page.getByRole('button',{name:'搜索当前会话消息'}).click();
  await page.getByRole('textbox',{name:'搜索已加载消息'}).fill('不存在的线路');
  await page.getByText('没有匹配的消息',{exact:true}).waitFor();
  await page.keyboard.press('Escape');
  await jump.waitFor();
  check(Math.abs(await page.locator('.pc-message-scroll').evaluate(el=>el.scrollTop)-readingPosition)<2,'closing a filtered search restores the previous history position');
  incoming('read_while_away','花田收工，明天见。');
  await jump.getByText('有新消息').waitFor();
  check(Math.abs(await page.locator('.pc-message-scroll').evaluate(el=>el.scrollTop)-readingPosition)<2,'incoming messages leave the current reading position in place');
  await jump.click();
  await jump.waitFor({state:'hidden'});
  check(await page.locator('.pc-message-scroll').evaluate(el=>el.scrollHeight-el.clientHeight-el.scrollTop<2),'jump lands below the newest message');
  await page.getByRole('button',{name:'消息操作 read_while_away',exact:true}).click();
  const actions = page.getByRole('dialog',{name:'消息操作'});
  check(await actions.getByRole('button',{name:'回复消息 read_while_away',exact:true}).isVisible(),'touch layout exposes actions for the chosen message in a dialog');
  await page.screenshot({path:'output/playwright/chat-review-mobile-actions.png',animations:'disabled'});
  await actions.getByRole('button',{name:'回复消息 read_while_away',exact:true}).click();
  await page.locator('.reply-preview').waitFor();
  check(await page.locator('.reply-preview').innerText().then(text=>text.includes('花田收工，明天见。')),'touch reply shows the correct excerpt above the composer');
  await page.getByRole('button',{name:'选择表情',exact:true}).click();
  await page.keyboard.press('Escape');
  check(await page.locator('.reply-preview').isVisible(),'dismissing the emoji panel preserves the reply');
  await page.screenshot({path:'output/playwright/chat-review-mobile-reply.png',animations:'disabled'});
  await page.getByRole('button',{name:'展开会话详情'}).click();
  check(await page.getByRole('button',{name:'关闭会话详情'}).isVisible(),'mobile details opens with a reachable close control');
  await page.keyboard.press('Escape');
  incoming('long_word', 'abcdefghijklmnopqrstuvwxyz'.repeat(9));
  await page.setViewportSize({width:320,height:570});
  const longBubble=page.locator('[data-message-id="long_word"] .bubble');
  await longBubble.waitFor();
  check(await longBubble.evaluate(el=>el.scrollWidth<=el.clientWidth+1 && el.getBoundingClientRect().right<=innerWidth),'unbroken text wraps inside the bubble at 320px');
  await page.getByRole('button',{name:'返回会话列表'}).click();
  await contact('林间风').click();
  check(await editor.evaluate(el=>document.activeElement!==el),'opening a conversation on a phone keeps the keyboard closed');
  await page.setViewportSize({width:1024,height:640});
  await contact('公共聊天').click();
  const history = page.locator('.pc-message-scroll'), firstMessage = page.locator('[data-message-id="public_1"]');
  await history.evaluate(el=>{el.scrollTop=0;});
  const anchorTop = (await firstMessage.boundingBox()).y;
  await page.getByRole('button',{name:'查看更早的消息'}).click();
  await page.locator('[data-message-id="older_fixture"]').waitFor();
  await page.waitForFunction(top=>Math.abs(document.querySelector('[data-message-id="public_1"]').getBoundingClientRect().top-top)<2,anchorTop);
  check(true,'loading an older page preserves the visible message anchor');
  await editor.fill('断线期间的草稿');
  const sentBeforeDisconnect = sent.length;
  sockets.at(-1).close({code:1000,reason:'fixture reconnect'});
  const reconnect = page.locator('.chat-connection-banner').getByRole('button',{name:'重新连接'});
  await reconnect.waitFor();
  await editor.press('Enter');
  check(sent.length===sentBeforeDisconnect && await editor.inputValue()==='断线期间的草稿','offline Enter preserves the draft without submitting');
  await reconnect.click();
  await reconnect.waitFor({state:'hidden'});
  check(await page.getByRole('button',{name:'发送消息',exact:true}).isEnabled(),'manual reconnect restores the composer');
  check(errors.length===0,`no React errors: ${errors.join('; ')}`);
  return {passed,errors,vectorEmojiCount,motionFrames:motion.length,publicSends:sent.filter(m=>m.channel==='public').length,privateSends:sent.filter(m=>m.channel!=='public').length,network:'All API and WebSocket traffic intercepted with synthetic fixtures'};
}

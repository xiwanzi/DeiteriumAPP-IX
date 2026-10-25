import {OfficialAccountSession,AccountError} from './account-session.js';

const client=new OfficialAccountSession({apiOrigin:location.hostname==='help.deuteriumix.com'?'https://www.deuteriumix.com':''});
const trigger=document.querySelector('[data-account-entry]');
const reduce=matchMedia('(prefers-reduced-motion: reduce)');
const escape=value=>String(value??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const icon=name=>`<svg viewBox="0 0 24 24" aria-hidden="true">${{
  user:'<circle cx="12" cy="8" r="3.5"/><path d="M5 21v-2a7 7 0 0 1 14 0v2"/>',
  mail:'<rect x="2" y="5" width="20" height="15" rx="1"/><path d="m2 6 10 8L22 6"/>',
  lock:'<rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V6a4 4 0 0 1 8 0v4m-4 5v2"/>',
  close:'<path d="m6 6 12 12M6 18 18 6"/>',
  eye:'<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>'
}[name]||''}</svg>`;
const dialog=document.createElement('dialog');
dialog.className='d9-auth';dialog.id='d9AccountDialog';dialog.setAttribute('aria-labelledby','d9-auth-title');dialog.setAttribute('data-lenis-prevent','');
document.body.append(dialog);
const legalOrigin=document.documentElement.dataset.accountLegalOrigin==='https://www.deuteriumix.com'?'https://www.deuteriumix.com':location.origin;
const emblem=document.querySelector('.header-emblem img')?.getAttribute('src')||'assets/emblem-original.png';
let screen='login',busy=false,codeBusy=false,challenge=null,challengeIdentity='',resendAt=0,cooldownTimer=null,dialogGeneration=0,codeGeneration=0,closing=false,returnFocus=null;
let currentUser=null,sessionWaiters=[],lastRestoredAt=0;
const actionCooldowns=new Map();
const noCodeDelivery=new Set(['MINECRAFT_PLAYER_NOT_FOUND','MINECRAFT_LOOKUP_UNAVAILABLE','MAIL_NOT_READY','EMAIL_INVALID','GAME_ID_INVALID','ACCOUNT_ALREADY_EXISTS','FORBIDDEN','INVALID_REQUEST']);
function cooldownKey(form,mode=screen){return mode==='register'?`register:${form.elements.email.value.trim().toLowerCase()}`:`${mode}:${form.elements.account.value.trim().toLowerCase()}`;}
function updateLoginButton(){if(screen!=='login'||busy)return;const form=dialog.querySelector('form'),button=form?.querySelector('.d9-auth-primary');if(!button)return;const remaining=Math.max(0,Math.ceil(((actionCooldowns.get(cooldownKey(form))?.until||0)-Date.now())/1000));button.disabled=remaining>0;button.textContent=remaining?`${remaining} 秒后重试`:'登录';}
const channel=typeof BroadcastChannel!=='undefined'?new BroadcastChannel('deuterium-official-account'):null;

function field({name,label,type='text',value='',placeholder=label,required=true,minlength,maxlength=254,autocomplete='off',symbol='user',trailing=''}) {
  return `<label class="d9-auth-field">${icon(symbol)}<input name="${name}" aria-label="${label}" type="${type}" placeholder="${placeholder}" value="${escape(value)}" ${required?'required':''} ${minlength?`minlength="${minlength}"`:''} maxlength="${maxlength}" autocomplete="${autocomplete}">${trailing}</label>`;
}
function policyURL(kind){const url=new URL(`${document.documentElement.lang==='en'?'/en':''}/${kind}/`,legalOrigin);url.searchParams.set('returnTo',location.href);url.searchParams.set('fromLogin','1');return escape(url.href);}
const agreements=()=>`<label class="d9-auth-agreement"><input name="agree" type="checkbox" required><span>我已阅读并同意 <a href="${policyURL('terms')}" target="_blank" rel="noopener">《用户协议》</a> 和 <a href="${policyURL('privacy')}" target="_blank" rel="noopener">《隐私政策》</a></span></label>`;
const errorBox=()=>'<p class="d9-auth-error" id="d9-auth-error" role="alert" hidden></p>';
function showError(error) {
  const el=dialog.querySelector('#d9-auth-error');if(!el)return;
  el.textContent=error.message||'请求未完成，请稍后重试。';el.hidden=false;
}
function open(mode='login',{account='',message=''}={}) {
  if(closing)return;
  const first=!dialog.open;
  if(first){returnFocus=document.activeElement;const menu=document.querySelector('#menuBtn');if(menu?.getAttribute('aria-expanded')==='true')menu.click();}
  const preserveChallenge=mode===screen&&!first;
  if(!preserveChallenge){challenge=null;challengeIdentity='';resendAt=0;clearInterval(cooldownTimer);}
  screen=mode;busy=false;codeBusy=false;++codeGeneration;const generation=++dialogGeneration;
  const heading={login:'Deuterium ID 登录',register:'注册 Deuterium ID',reset:'找回密码',account:'账号信息'}[screen];
  let contents='';
  if(screen==='account') {
    const session=client.snapshot();if(!session)return open('login');
    const user=session.user;
    contents=`<div class="d9-auth-profile"><div class="d9-auth-avatar">${icon('user')}</div><strong>${escape(user.displayName||user.gameId||'玩家')}</strong><span>已登录</span></div><dl class="d9-auth-details"><div><dt>游戏内 ID</dt><dd>${escape(user.gameId||'暂无')}</dd></div><div><dt>邮箱</dt><dd>${escape(user.email||'暂无')}</dd></div></dl>${errorBox()}<button class="d9-auth-primary" data-auth-action="logout" type="button">退出登录</button>`;
  }else {
    const email=screen==='register'?field({name:'email',label:'邮箱',type:'email',autocomplete:'email',symbol:'mail'}):'';
    const accountField=field({name:screen==='register'?'gameId':'account',label:screen==='register'?'正版账号ID':'邮箱 / 正版账号ID / QQ',value:account,autocomplete:'username',maxlength:screen==='register'?32:254});
    const code=screen==='login'?'':field({name:'code',label:'验证码',placeholder:'请输入验证码',symbol:'lock',maxlength:6,autocomplete:'one-time-code',trailing:'<button type="button" class="d9-auth-code" data-auth-action="send-code">获取验证码</button>'});
    const password=field({name:'password',label:screen==='login'?'密码':screen==='reset'?'设置新密码':'设置密码',type:'password',symbol:'lock',minlength:screen==='login'?undefined:8,maxlength:64,placeholder:screen==='login'?'请输入密码':'设置密码（8–64 位）',autocomplete:screen==='login'?'current-password':'new-password',trailing:`<button type="button" class="d9-auth-eye" data-auth-action="toggle-password" aria-label="显示密码">${icon('eye')}</button>`});
    const confirm=screen==='login'?'':field({name:'confirm',label:'确认密码',placeholder:'再次输入密码',type:'password',symbol:'lock',minlength:8,maxlength:64,autocomplete:'new-password'});
    contents=`<form id="d9-auth-form">${accountField}${email}${code}${password}${confirm}${screen==='login'?'<div class="d9-auth-secondary"><button type="button" data-auth-action="reset">忘记密码？</button></div>':''}${agreements()}${errorBox()}<button class="d9-auth-primary" type="submit">${screen==='login'?'登录':screen==='register'?'注册并登录':'保存新密码'}</button><div class="d9-auth-switch">${screen==='login'?'没有账号？<button type="button" data-auth-action="register">立即注册</button>':'已有账号？<button type="button" data-auth-action="login">返回登录</button>'}</div></form>`;
  }
  dialog.innerHTML=`<div class="d9-auth-brand"><img alt="" src="${escape(emblem)}"><b>DEUTERIUM IX</b><button type="button" class="d9-auth-close" data-auth-action="close" aria-label="关闭登录窗口">${icon('close')}</button></div><div class="d9-auth-body"><h2 id="d9-auth-title" class="${screen==='login'?'d9-auth-sr':''}">${heading}</h2>${message?`<p class="d9-auth-message" role="status">${escape(message)}</p>`:''}${contents}</div>`;
  dialog.dataset.screen=screen;
  document.documentElement.classList.add('d9-auth-open');
  if(first)dialog.showModal();else if(!reduce.matches)dialog.querySelector('.d9-auth-body').animate([{opacity:.3,transform:'translateY(5px)'},{opacity:1,transform:'translateY(0)'}],{duration:170,easing:'ease-out'});
  dialog.scrollTop=0;
  requestAnimationFrame(()=>{if(generation===dialogGeneration)dialog.querySelector('.d9-auth-close')?.focus({preventScroll:true});});
  const form=dialog.querySelector('form');
  form?.addEventListener('submit',submit);
  form?.addEventListener('input',event=>{
    const error=dialog.querySelector('#d9-auth-error');if(error)error.hidden=true;
    if(['gameId','email','account'].includes(event.target.name)&&challengeIdentity!==identityKey(form)){
      ++codeGeneration;codeBusy=false;challenge=null;challengeIdentity='';resendAt=0;updateCodeButton();
    }
    updateCodeButton();
  });
  clearInterval(cooldownTimer);cooldownTimer=setInterval(updateCodeButton,500);updateCodeButton();
}
async function close({cancel=true}={}) {
  if(!dialog.open||closing)return;
  closing=true;++dialogGeneration;++codeGeneration;clearInterval(cooldownTimer);
  if(!reduce.matches)await dialog.animate([{opacity:1,transform:'translateY(0) scale(1)'},{opacity:0,transform:'translateY(7px) scale(.99)'}],{duration:130,easing:'ease-in'}).finished.catch(()=>{});
  dialog.close();dialog.innerHTML='';document.documentElement.classList.remove('d9-auth-open');closing=false;busy=false;codeBusy=false;challenge=null;
  if(cancel)sessionWaiters.splice(0).forEach(resolve=>resolve(null));
  returnFocus?.isConnected&&returnFocus.focus?.({preventScroll:true});
}
function identityKey(form) { return screen==='register'?`${form.elements.gameId.value.trim()}\n${form.elements.email.value.trim().toLowerCase()}`:form.elements.account.value.trim(); }
function updateCodeButton() {
  updateLoginButton();
  const button=dialog.querySelector('[data-auth-action="send-code"]');if(!button)return;
  resendAt=Math.max(actionCooldowns.get(cooldownKey(dialog.querySelector('form')))?.until||0,actionCooldowns.get('code:*')?.until||0);
  const remaining=Math.max(0,Math.ceil((resendAt-Date.now())/1000));
  button.disabled=busy||codeBusy||remaining>0;
  button.textContent=remaining?`${remaining} 秒后重发`:'获取验证码';
}
function setBusy(value) {
  busy=value;dialog.setAttribute('aria-busy',String(value));
  dialog.querySelectorAll('input,button:not(.d9-auth-close)').forEach(el=>{el.disabled=value;});
  const button=dialog.querySelector('.d9-auth-primary');
  if(button)button.textContent=value?'正在处理…':screen==='register'?'注册并登录':screen==='reset'?'保存新密码':screen==='account'?'退出登录':'登录';
  if(!value)updateCodeButton();
}
async function sendCode() {
  if(busy||codeBusy||resendAt>Date.now())return;
  const form=dialog.querySelector('form');if(!form)return;
  const fields=screen==='register'?['gameId','email']:['account'];
  if(fields.some(name=>!form.elements[name].reportValidity()))return;
  if(screen==='register'&&!/^[A-Za-z0-9_]{3,16}$/.test(form.elements.gameId.value.trim()))return showError(new Error('请填写 Minecraft Java 版玩家名：3–16 位字母、数字或下划线。'));
  const key=identityKey(form),generation=dialogGeneration,requestGeneration=++codeGeneration,type=screen==='register'?'register':'reset';
  const recipient=cooldownKey(form),requestedAt=Date.now();
  actionCooldowns.set(recipient,{until:requestedAt+60000,requestGeneration});
  challengeIdentity=key;codeBusy=true;resendAt=Date.now()+60000;updateCodeButton();
  clearInterval(cooldownTimer);cooldownTimer=setInterval(updateCodeButton,500);
  try {
    const data=await client.requestCode(type,Object.fromEntries(new FormData(form)));
    if(actionCooldowns.get(recipient)?.requestGeneration===requestGeneration)actionCooldowns.set(recipient,{until:Date.now()+Math.max(1,Number(data.resendAfterSeconds)||60)*1000,requestGeneration});
    if(generation!==dialogGeneration||requestGeneration!==codeGeneration||identityKey(form)!==key)return;
    challenge=data;challengeIdentity=key;resendAt=Date.now()+Math.max(1,Number(data.resendAfterSeconds)||60)*1000;
    clearInterval(cooldownTimer);cooldownTimer=setInterval(updateCodeButton,500);
    form.elements.code.focus();
  }catch(error){
    if(actionCooldowns.get(recipient)?.requestGeneration===requestGeneration){
      if(error.status===429){const entry={until:Date.now()+Math.max(1,Number(error.retryAfter)||60)*1000,requestGeneration};actionCooldowns.set(recipient,entry);if(error.retryScope==='SOURCE')actionCooldowns.set('code:*',entry);}
      else if(noCodeDelivery.has(error.code)||error.status===400||error.status===403||error.status===409)actionCooldowns.delete(recipient);
    }
    if(generation===dialogGeneration&&requestGeneration===codeGeneration&&identityKey(form)===key)showError(noCodeDelivery.has(error.code)||error.status>=400&&error.status<500?error:new Error('本次发送结果暂未确认，请查看邮箱；倒计时结束后可重试。'));
  }
  finally{if(generation===dialogGeneration&&requestGeneration===codeGeneration)codeBusy=false;updateCodeButton();}
}
async function submit(event) {
  event.preventDefault();if(busy||codeBusy)return;
  const form=event.currentTarget;if(!form.reportValidity())return;
  const input=Object.fromEntries(new FormData(form));
  if(screen==='login'&&(actionCooldowns.get(cooldownKey(form))?.until||0)>Date.now())return;
  if(screen!=='login') {
    if(!challenge||challengeIdentity!==identityKey(form))return showError(new Error('请先获取当前账号的验证码。'));
    if(Date.parse(challenge.expiresAt)<=Date.now())return showError(new Error('验证码已过期，请重新获取。'));
    if(!/^\d{6}$/.test(input.code))return showError(new Error('请输入 6 位验证码。'));
    if(Array.from(input.password).length<8||Array.from(input.password).length>64)return showError(new Error('密码需要 8–64 位。'));
    if(input.password!==input.confirm)return showError(new Error('两次输入的密码不一致。'));
  }
  const generation=dialogGeneration,mode=screen;setBusy(true);
  try {
    if(mode==='login')await client.login(input.account,input.password);
    else if(mode==='register')await client.register({verificationToken:challenge.verificationToken,code:input.code,password:input.password});
    else await client.resetPassword({verificationToken:challenge.verificationToken,code:input.code,password:input.password});
    channel?.postMessage({type:'changed'});
    if(generation!==dialogGeneration)return;
    if(mode==='reset'){setBusy(false);open('login',{account:input.account,message:'密码已更新，请使用新密码登录。'});}
    else {
      const session=client.snapshot();
      sessionWaiters.splice(0).forEach(resolve=>resolve(session));
      await close({cancel:false});
    }
  }catch(error){if(mode==='login'&&error.status===429)actionCooldowns.set(`login:${input.account.trim().toLowerCase()}`,{until:Date.now()+Math.max(1,Number(error.retryAfter)||900)*1000});if(generation===dialogGeneration)showError(mode==='register'&&['NETWORK_ERROR','INVALID_RESPONSE','SERVICE_UNAVAILABLE'].includes(error.code)?new Error('注册结果暂未确认，请尝试使用刚才的邮箱和密码登录。'):error);}
  finally{input.password='';input.confirm='';if(generation===dialogGeneration)setBusy(false);}
}
async function logout() {
  if(busy)return;const generation=dialogGeneration;setBusy(true);
  try{await client.logout();channel?.postMessage({type:'changed'});if(generation===dialogGeneration)await close();}
  catch(error){if(generation===dialogGeneration)showError(error);}
  finally{if(generation===dialogGeneration)setBusy(false);}
}
let backdropPointerDown=false;
function onBackdrop(event){const r=dialog.getBoundingClientRect();return event.target===dialog&&(event.clientX<r.left||event.clientX>r.right||event.clientY<r.top||event.clientY>r.bottom);}
dialog.addEventListener('pointerdown',event=>{backdropPointerDown=event.button===0&&onBackdrop(event);});
dialog.addEventListener('pointercancel',()=>{backdropPointerDown=false;});
dialog.addEventListener('click',event=>{
  const backdropClick=backdropPointerDown&&onBackdrop(event);backdropPointerDown=false;
  const action=event.target.closest('[data-auth-action]')?.dataset.authAction;
  if(action==='close')close();
  else if(['login','register','reset'].includes(action)&&!busy&&!codeBusy) {
    const account=dialog.querySelector('[name="account"]')?.value||dialog.querySelector('[name="gameId"]')?.value||'';
    open(action,{account});
  }else if(action==='send-code')sendCode();
  else if(action==='logout')logout();
  else if(action==='toggle-password') {
    const input=dialog.querySelector('[name="password"]');input.type=input.type==='password'?'text':'password';
    event.target.closest('button').setAttribute('aria-label',input.type==='password'?'显示密码':'隐藏密码');
  }else if(backdropClick)close();
});
dialog.addEventListener('cancel',event=>{event.preventDefault();close();});
dialog.addEventListener('keydown',event=>event.stopPropagation());

function publishSession() {
  const session=client.snapshot();currentUser=session?.user||null;
  if(trigger){
    trigger.querySelector('[data-account-label]').textContent=currentUser?(currentUser.displayName||currentUser.gameId||'账号'):'用户登录';
    trigger.setAttribute('aria-label',currentUser?'查看账号信息':'用户登录');
    trigger.classList.toggle('is-signed-in',!!currentUser);
  }
  // A late restore may update a pristine dialog, but never discard typed input.
  if(dialog.open&&!busy) {
    if(currentUser&&screen==='login'&&![...dialog.querySelectorAll('input:not([type=checkbox])')].some(input=>input.value))open('account');
    else if(!currentUser&&screen==='account')open('login');
  }
  window.dispatchEvent(new CustomEvent('deuterium:authchange',{detail:session}));
}
client.addEventListener('change',publishSession);
async function refresh() { lastRestoredAt=Date.now();return client.restore(); }
function refreshIfStale() { return Date.now()-lastRestoredAt<30000?Promise.resolve(client.snapshot()):refresh(); }
const ready=refresh().catch(()=>null);
function requireLogin() {
  return refresh().then(session=>{
    if(session)return session;
    return new Promise(resolve=>{sessionWaiters.push(resolve);open('login');});
  });
}
window.DeuteriumAuth=Object.freeze({
  ready, getSession:()=>client.snapshot(), refresh,
  requireLogin,
  openLogin:()=>open(client.snapshot()?'account':'login'),
  request:(path,options)=>client.authenticatedRequest(path,options),
  logout:async()=>{await client.logout();channel?.postMessage({type:'changed'});}
});
trigger?.addEventListener('click',()=>{open(client.snapshot()?'account':'login');refreshIfStale().catch(()=>{});});
channel?.addEventListener('message',event=>{if(event.data?.type==='changed')refresh().catch(()=>{});});
document.addEventListener('visibilitychange',()=>{if(!document.hidden)refreshIfStale().catch(()=>{});});
window.addEventListener('focus',()=>{refreshIfStale().catch(()=>{});});
window.addEventListener('pageshow',event=>{if(event.persisted)refreshIfStale().catch(()=>{});});
publishSession();

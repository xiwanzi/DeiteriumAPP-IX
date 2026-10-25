// Official-site account client. Session authority stays in the existing Go service.
// The HttpOnly cookie persists for seven days; no credentials or login flag are stored here.
export class AccountError extends Error {
  constructor(message, code='REQUEST_FAILED', status=0, retryAfter=0, retryScope='') {
    super(message);this.name='AccountError';this.code=code;this.status=status;this.retryAfter=retryAfter;this.retryScope=retryScope;
  }
}

export class OfficialAccountSession extends EventTarget {
  #session=null;
  #restore=null;
  #revision=0;
  #expiryTimer=null;
  #apiOrigin='';
  constructor({fetchImpl=globalThis.fetch.bind(globalThis),now=()=>Date.now(),apiOrigin=''}={}) {
    super();
    if(!['','https://www.deuteriumix.com'].includes(apiOrigin))throw new AccountError('账号服务来源不正确。','INVALID_ORIGIN');
    this.#apiOrigin=apiOrigin;this.fetch=fetchImpl;this.now=now;
  }
  snapshot() {
    if(this.#session&&Date.parse(this.#session.expiresAt)<=this.now())this.#set(null);
    return this.#session?{user:structuredClone(this.#session.user),expiresAt:this.#session.expiresAt}:null;
  }
  #set(session) {
    clearTimeout(this.#expiryTimer);
    if(session) {
      if(!session.user?.userId||!session.csrfToken||!Number.isFinite(Date.parse(session.expiresAt))||Date.parse(session.expiresAt)<=this.now()) {
        throw new AccountError('登录状态暂时无法读取，请重新登录。','INVALID_SESSION');
      }
      this.#session={user:{...session.user},csrfToken:session.csrfToken,expiresAt:session.expiresAt};
      this.#expiryTimer=setTimeout(()=>this.#set(null),Math.min(Date.parse(session.expiresAt)-this.now(),2147483647));
      this.#expiryTimer?.unref?.();
    }else this.#session=null;
    this.dispatchEvent(new Event('change'));
  }
  async #request(path,{method='GET',body,headers={},signal}={}) {
    const requestRevision=this.#revision;
    if(!path.startsWith('/api/v1/')||path.startsWith('//')||path.includes('://')||/[\r\n]/.test(path))throw new AccountError('请求地址不正确。','INVALID_PATH');
    const requestHeaders=new Headers(headers);
    requestHeaders.set('Accept','application/json');
    if(body!==undefined)requestHeaders.set('Content-Type','application/json');
    if(!['GET','HEAD'].includes(method)&&this.#session)requestHeaders.set('X-CSRF-Token',this.#session.csrfToken);
    let response;
    try {
      response=await this.fetch(this.#apiOrigin+path,{method,body:body===undefined?undefined:JSON.stringify(body),headers:requestHeaders,credentials:this.#apiOrigin?'include':'same-origin',cache:'no-store',signal:signal||AbortSignal.timeout(15000)});
    }catch(error){throw new AccountError(error.name==='TimeoutError'?'连接超时，请稍后重试。':'暂时无法连接账号服务，请稍后重试。','NETWORK_ERROR');}
    let value;
    try{value=await response.json();}catch{throw new AccountError('账号服务暂不可用，请稍后重试。','INVALID_RESPONSE',response.status);}
    if(!response.ok||value.error) {
      const error=value.error||{};
      if(response.status===401&&error.code!=='ACCOUNT_PASSWORD_INVALID'&&requestRevision===this.#revision)this.#set(null);
      throw new AccountError(error.message||'请求未完成，请稍后重试。',error.code||'REQUEST_FAILED',response.status,Number(error.retryAfterSeconds||response.headers.get('Retry-After')||0),error.retryScope||'');
    }
    return value.data;
  }
  async restore() {
    if(this.#restore)return this.#restore;
    const revision=this.#revision;
    const promise=(async()=>{
      try {
        const data=await this.#request('/api/v1/web/session');
        if(revision===this.#revision)this.#set(data);
      }catch(error) {
        if(revision!==this.#revision)return this.snapshot();
        if(error.status===401)this.#set(null);else throw error;
      }
      return this.snapshot();
    })();
    this.#restore=promise;
    try{return await promise;}finally{if(this.#restore===promise)this.#restore=null;}
  }
  async login(account,password) {
    const revision=++this.#revision;
    const data=await this.#request('/api/v1/web/session',{method:'POST',body:{account:account.trim(),password}});
    if(revision===this.#revision)this.#set(data);
    return this.snapshot();
  }
  async requestCode(type,{gameId,email,account}) {
    if(!['register','reset'].includes(type))throw new AccountError('验证码用途不正确。','INVALID_PURPOSE');
    return this.#request(`/api/v1/account/${type==='register'?'registration-code':'password-reset-code'}`,{
      method:'POST',body:type==='register'?{gameId:gameId.trim(),email:email.trim()}:{account:account.trim()}
    });
  }
  async register({verificationToken,code,password}) {
    const revision=++this.#revision;
    const data=await this.#request('/api/v1/web/register',{method:'POST',body:{verificationToken,code:code.trim(),password}});
    if(revision===this.#revision)this.#set(data);
    return this.snapshot();
  }
  async resetPassword({verificationToken,code,password}) {
    const data=await this.#request('/api/v1/account/password-reset',{method:'POST',body:{verificationToken,code:code.trim(),newPassword:password}});
    ++this.#revision;this.#set(null);return data;
  }
  async logout() {
    await this.restore();
    if(!this.#session)return;
    ++this.#revision;
    await this.#request('/api/v1/web/session',{method:'DELETE'});
    this.#set(null);
  }
  async authenticatedRequest(path,options={}) {
    const expectedUser=this.snapshot()?.user.userId;
    await this.restore();
    if(!this.snapshot())throw new AccountError('请先登录。','LOGIN_REQUIRED',401);
    if(expectedUser&&this.#session.user.userId!==expectedUser)throw new AccountError('登录账号已变化，请重新确认后提交。','ACCOUNT_CHANGED',409);
    return this.#request(path,options);
  }
}

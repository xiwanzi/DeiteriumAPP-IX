const englishErrors={"EMAIL_ADDRESS_INVALID":"This email address is invalid. Check it before requesting another code.","ACCOUNT_PASSWORD_INVALID":"The account or password is incorrect.","UNAUTHORIZED":"Please sign in again.","FORBIDDEN":"You cannot perform this action.","ACCOUNT_ALREADY_EXISTS":"This email or Minecraft username is already registered. Sign in or reset your password.","ACCOUNT_NOT_FOUND":"No account was found. Check your email or username.","EMAIL_INVALID":"Enter a valid email address.","GAME_ID_INVALID":"Enter a valid Minecraft: Java Edition username.","MINECRAFT_PLAYER_NOT_FOUND":"This Minecraft username could not be found. Check your Java Edition username and request a new code.","MINECRAFT_LOOKUP_UNAVAILABLE":"The Minecraft profile service is unavailable. Try requesting a code again later.","MAIL_NOT_READY":"Verification email is unavailable. Please try again later.","VERIFICATION_INVALID":"The code is incorrect, expired, already used, or has exceeded its attempt limit.","PASSWORD_INVALID":"Your password must contain 8–64 characters.","ACCOUNT_CREATED_LOGIN_REQUIRED":"Your account was created, but sign-in did not finish. Sign in with your email and password.","INVALID_REQUEST":"Check the information you entered and try again.","LOGIN_LOCKED":"Too many sign-in attempts. Please wait before trying again.","RATE_LIMITED":"Too many requests. Please wait before trying again.","SERVER_BUSY":"The service is busy. Please try again later.","SERVICE_UNAVAILABLE":"The service is unavailable. Please try again later.","GAME_VERIFICATION_REQUIRED":"Complete game identity verification to use this service.","MAIL_ADDRESS_REJECTED":"This email address was rejected. Check it and try another address.","MAIL_RECIPIENT_BLOCKED":"This email address is temporarily blocked after a delivery failure. Check the address before retrying.","MAIL_DELIVERY_FAILED":"Email delivery failed. Check your address and try again later."};
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
    if(!['','https://www.deuteriumix.com'].includes(apiOrigin))throw new AccountError('Invalid account service origin.','INVALID_ORIGIN');
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
        throw new AccountError('Your session could not be restored. Sign in again.','INVALID_SESSION');
      }
      this.#session={user:{...session.user},csrfToken:session.csrfToken,expiresAt:session.expiresAt};
      this.#expiryTimer=setTimeout(()=>this.#set(null),Math.min(Date.parse(session.expiresAt)-this.now(),2147483647));
      this.#expiryTimer?.unref?.();
    }else this.#session=null;
    this.dispatchEvent(new Event('change'));
  }
  async #request(path,{method='GET',body,headers={},signal}={}) {
    const requestRevision=this.#revision;
    if(!path.startsWith('/api/v1/')||path.startsWith('//')||path.includes('://')||/[\r\n]/.test(path))throw new AccountError('Invalid request URL.','INVALID_PATH');
    const requestHeaders=new Headers(headers);
    requestHeaders.set('Accept','application/json');
    if(body!==undefined)requestHeaders.set('Content-Type','application/json');
    if(!['GET','HEAD'].includes(method)&&this.#session)requestHeaders.set('X-CSRF-Token',this.#session.csrfToken);
    let response;
    try {
      response=await this.fetch(this.#apiOrigin+path,{method,body:body===undefined?undefined:JSON.stringify(body),headers:requestHeaders,credentials:this.#apiOrigin?'include':'same-origin',cache:'no-store',signal:signal||AbortSignal.timeout(15000)});
    }catch(error){throw new AccountError(error.name==='TimeoutError'?'The connection timed out. Please try again.':'The account service is unavailable. Please try again.','NETWORK_ERROR');}
    let value;
    try{value=await response.json();}catch{throw new AccountError('The account service is unavailable. Please try again.','INVALID_RESPONSE',response.status);}
    if(!response.ok||value.error) {
      const error=value.error||{};
      if(response.status===401&&error.code!=='ACCOUNT_PASSWORD_INVALID'&&requestRevision===this.#revision)this.#set(null);
      throw new AccountError((response.status===429&&Number(error.retryAfterSeconds||response.headers.get('Retry-After'))>0?`Please wait ${Number(error.retryAfterSeconds||response.headers.get('Retry-After'))} seconds before trying again.`:englishErrors[error.code]||'The request could not be completed. Please try again.'),error.code||'REQUEST_FAILED',response.status,Number(error.retryAfterSeconds||response.headers.get('Retry-After')||0),error.retryScope||'');
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
    if(!['register','reset'].includes(type))throw new AccountError('Invalid verification code request.','INVALID_PURPOSE');
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
    if(!this.snapshot())throw new AccountError('Please sign in first.','LOGIN_REQUIRED',401);
    if(expectedUser&&this.#session.user.userId!==expectedUser)throw new AccountError('The signed-in account has changed. Review your details before submitting.','ACCOUNT_CHANGED',409);
    return this.#request(path,options);
  }
}

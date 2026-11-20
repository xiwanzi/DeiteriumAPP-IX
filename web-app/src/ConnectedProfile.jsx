import { usePlayerNameState } from "./use-player-names.js";
import React, { useContext, useEffect, useRef, useState } from "react";
import { Bell, ArrowLeft, Camera, Check, ChevronRight, Copy, Eye, EyeOff, Headphones, KeyRound, LogOut, MessageCircle, Monitor, PenLine, ReceiptText, RefreshCw, ShieldCheck, ShoppingBag, Store, WalletCards } from "lucide-react";
import { Avatar, Badge, Button, Empty, Field, Modal, Toggle } from "./components.jsx";
import { id } from "./format.js";
import DeviceSessions from "./DeviceSessions.jsx";
import { AvatarEditor, ProfileEditor } from "./ProfileEditors.jsx";
import { ShoppingCartContext } from "./ShoppingCartContext.jsx";
import "./my-profile.css";

const services = [
  ["/wallet","查看钱包",WalletCards,"wallet"], ["/orders","我的订单",ReceiptText,"orders"],
  ["/cart","购物袋",ShoppingBag,"bag"], ["/merchant","我的商店",Store,"store"],
];

function MeLink({ to, navigate, children, ...props }) {
  return <a href={to} {...props} onClick={event=>{
    if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.button!==0) return;
    event.preventDefault(); navigate(to);
  }}>{children}</a>;
}

function contactValue(value, reveal, email = false) {
  if (!value) return "未提供";
  const text = String(value);
  if (reveal) return text;
  const at = email ? text.lastIndexOf("@") : -1;
  if (at>0) return text.slice(0,1)+"•••"+text.slice(at);
  return text.length>4 ? text.slice(0,2)+"•••"+text.slice(-2) : "••••";
}

export default function ConnectedProfile({ client, user, onLogout, onPasswordChanged, navigate, onProfileUpdated, initialProfile, upload }) {
  const cached = initialProfile && initialProfile.playerRef===user.playerRef && Number.isSafeInteger(initialProfile.version) ? initialProfile : null;
  const [profile, setProfile] = usePlayerNameState(client, cached);
  const [reading, setReading] = useState(true), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const [panel, setPanel] = useState(""), [file, setFile] = useState(null), [reveal, setReveal] = useState(false);
  const [copyState, setCopyState] = useState(""), [loggingOut, setLoggingOut] = useState(false);
  const fileInput = useRef(null), generation = useRef(0), active = useRef(true), latest = useRef(cached);
  const cart = useContext(ShoppingCartContext);
  const accept = value => {
    if (!active.current) return;
    if (!value || value.playerRef!==user.playerRef || !Number.isSafeInteger(value.version) || value.version<1) throw new Error("资料未完整返回，请刷新重试。");
    if (latest.current?.version>value.version) return;
    latest.current = value; setProfile(value); onProfileUpdated?.(value);
  };
  const load = async () => {
    const current=++generation.current;
    setReading(true); setError("");
    try {
      const response=await client.profile(user.playerRef);
      if (current===generation.current) accept(response.data);
    } catch(e) { if (current===generation.current) setError(e.message); }
    finally { if (current===generation.current) setReading(false); }
  };
  useEffect(()=>{
    active.current=true; load();
    return ()=>{active.current=false;generation.current++;};
  },[client,user.playerRef]);
  useEffect(()=>{
    if (!copyState) return;
    const timer=setTimeout(()=>setCopyState(""),3000);
    return ()=>clearTimeout(timer);
  },[copyState]);
  const copyId=async()=>{
    try { await navigator.clipboard.writeText(user.userId); setCopyState("已复制"); }
    catch { setCopyState("请选中编号复制"); }
  };
  const editProfile=()=>{setNotice("");setPanel("edit");};
  const saved=(value)=>{accept(value);setError("");setNotice(panel==="avatar"?"头像已更新":"个人资料已保存");};
  const name=profile?.displayName || user.displayName || profile?.gameId || user.gameId;
  const qq=profile?.qq || user.qq;
  return <section className="my-profile" aria-label="我的">
    <header className="me-page-heading"><h1>我的</h1><Button secondary disabled={reading} onClick={load} aria-label="刷新资料"><RefreshCw size={16} className={reading?"spin":undefined}/>刷新</Button></header>
    <section className="me-identity" aria-label="个人资料">
      <div className="me-identity-body"><button type="button" className="me-avatar-button" disabled={!profile} title="更换头像" aria-label="更换头像" onClick={()=>fileInput.current?.click()}>
        <Avatar user={profile || user} size="large"/><span><Camera size={15}/></span></button>
        <div className="me-intro"><div className="me-name"><h2>{name}</h2>{user.gameVerified===true && <span className="me-verified"><ShieldCheck size={14}/>游戏身份已验证</span>}</div>
          <p className="me-game-id">游戏 ID <strong>{profile?.gameId || user.gameId || "—"}</strong></p>
          {profile?.bio && <p className="me-bio">{profile.bio}</p>}
          <Button secondary disabled={!profile} onClick={editProfile}><PenLine size={15}/>编辑资料</Button>
        </div>
      </div>
      <input ref={fileInput} hidden type="file" accept="image/png,image/jpeg,image/webp" onChange={event=>{const next=event.target.files?.[0];event.target.value="";if(next){setFile(next);setNotice("");setPanel("avatar");}}}/>
    </section>
    {reading && !profile && <p className="me-status" role="status">正在读取个人资料…</p>}
    {error && <div className="me-alert" role="alert"><span>{error}</span><button type="button" disabled={reading} onClick={load}>重试</button></div>}
    {notice && <p className="me-status" role="status"><Check size={15}/>{notice}</p>}
    <nav className="me-services" aria-label="常用服务">{services.map(([to,label,Icon,tone])=>
      <MeLink key={to} to={to} navigate={navigate} aria-label={label} aria-description={to==="/cart" && cart?.count>0?`共 ${cart.count} 件商品`:undefined}>
        <span className={`me-service-icon ${tone}`}><Icon size={21}/>{to==="/cart" && cart?.count>0 && <b aria-hidden="true">{cart.count>99?"99+":cart.count}</b>}</span>
        <strong>{label}</strong><ChevronRight size={15} className="me-service-arrow"/>
      </MeLink>)}</nav>
    <div className="me-panels">
      <section className="me-panel me-settings" aria-label="账号与设置"><header><h2>账号与设置</h2></header>
        <button type="button" onClick={()=>setPanel("devices")}><Monitor size={18}/>登录设备<ChevronRight size={15}/></button>
        <button type="button" onClick={()=>setPanel("password")}><KeyRound size={18}/>更改密码<ChevronRight size={15}/></button>
        <MeLink to="/notification-settings" navigate={navigate}><Bell size={18}/>通知设置<ChevronRight size={15}/></MeLink>
        <MeLink to="/support" navigate={navigate}><Headphones size={18}/>邮件支持<ChevronRight size={15}/></MeLink>
      </section>
      <section className="me-panel me-account" aria-label="账号信息"><header><h2>账号信息</h2>
        {(user.email || qq) && <button type="button" className="me-subtle" aria-label={reveal?"隐藏联系方式":"显示联系方式"} aria-pressed={reveal} onClick={()=>setReveal(old=>!old)}>{reveal?<EyeOff size={16}/>:<Eye size={16}/>}<span>{reveal?"隐藏":"显示"}</span></button>}</header>
        <dl><div><dt>Deuterium ID</dt><dd><code>{user.userId}</code><button type="button" className="me-copy" aria-label="复制 Deuterium ID" title="复制 Deuterium ID" onClick={copyId}>{copyState==="已复制"?<Check size={15}/>:<Copy size={15}/>}</button></dd></div>
          <div><dt>邮箱</dt><dd>{contactValue(user.email,reveal,true)}</dd></div>
          <div><dt>QQ</dt><dd>{contactValue(qq,reveal)}</dd></div></dl>
        {copyState && <p className="me-status" role="status">{copyState}</p>}
      </section>
    </div>
    <footer className="me-footer"><button type="button" disabled={loggingOut} onClick={async()=>{setLoggingOut(true);try{await onLogout();}catch(e){if(active.current)setError(e.message);}finally{if(active.current)setLoggingOut(false);}}}><LogOut size={16}/>{loggingOut?"正在退出…":"退出登录"}</button></footer>
    {panel==="edit" && profile && <ProfileEditor client={client} profile={profile} accept={accept} close={()=>setPanel("")} onSaved={saved}/>}
    {panel==="avatar" && profile && file && <AvatarEditor client={client} user={user} profile={profile} firstFile={file} accept={accept} onSaved={saved} upload={upload} close={()=>{setPanel("");setFile(null);}}/>}
    {panel==="password" && <Modal title="更改密码" className="me-dialog" close={()=>setPanel("")} dismissOnBackdrop={false}><PasswordChangeForm client={client} user={user} onChanged={onPasswordChanged}/></Modal>}
    {panel==="devices" && <Modal title="登录设备" className="me-dialog" close={()=>setPanel("")}><DeviceSessions client={client}/></Modal>}
  </section>;
}

export function PlayerProfile({ client, user, playerRef, onMessage }) {
  const [profile, setProfile] = usePlayerNameState(client, null), [error, setError] = useState(""), [busy, setBusy] = useState(false);
  useEffect(() => { let active = true; setProfile(null); client.profile(playerRef).then((r) => { if (active && !client.erasedPlayerRefs.has(playerRef)) setProfile(r.data); }).catch((e) => { if (active) setError(e.message); }); const unsubscribe = client.onAccountDeletions((refs) => { if (refs.has(playerRef)) { setProfile(null); setError("该账号已注销"); } }); return () => { active = false; unsubscribe(); }; }, [client, playerRef]);
  const follow = async () => { setBusy(true); setError(""); try { const r = profile.followed ? await client.request(`/api/v1/chat/follows/${encodeURIComponent(playerRef)}`, { method: "DELETE" }) : await client.request("/api/v1/chat/follows", { method: "POST", body: { playerRef } }); setProfile({ ...profile, followed: r.data.followed }); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  if (!profile) return <Empty title={error ? "暂时无法读取资料" : "正在读取资料"} text={error || "请稍候…"} />;
  return <><div className="player-detail"><Avatar user={profile} size="large" /><h2>{profile.displayName || profile.gameId}</h2><Badge tone="neutral">{profile.online ? "游戏在线" : profile.lastSeenAt ? `上次在线 ${new Date(profile.lastSeenAt).toLocaleString("zh-CN")}` : "玩家"}</Badge></div>
    <p className="description" style={{ whiteSpace: "pre-wrap" }}>{profile.bio || "还没有填写个人简介。"}</p><p>游戏 ID · {profile.gameId}</p><p>QQ · {profile.qq}</p>
    {user.playerRef !== playerRef && <div className="button-row"><Button disabled={busy} onClick={async () => { setBusy(true); try { await onMessage(profile); } catch (e) { setError(e.message); } finally { setBusy(false); } }}><MessageCircle size={16} />私聊</Button><Button secondary disabled={busy} onClick={follow}>{profile.followed ? "取消特别关心" : "特别关心"}</Button></div>}
    {error && <p className="auth-error" role="alert">{error}</p>}
  </>;
}

const notificationGroups = [
  ["提醒与预览",[["enabled","消息提醒","接收账号的业务提醒"],["showPreviews","显示通知摘要","在系统通知中显示消息内容"]]],
  ["聊天",[["directMessages","私聊消息","其他玩家发来的消息"],["mentions","有人提及我","公共聊天或私聊中的提及"],["followedPlayers","特别关心","关注玩家发布的新消息"]]],
  ["交易与社区",[["wallet","钱包","信用点变化与转账"],["marketOrders","市场订单","购买、发货与退款"],["commissions","委托","接取、完成与验收"],["announcements","官方公告","社区的重要消息"],["appUpdates","应用更新","应用与资源版本变化"]]],
];

export function ConnectedNotificationSettings({ client, onBack }) {
  const [value,setValue]=useState(null), [busy,setBusy]=useState(false), [error,setError]=useState("");
  const lock = useRef(false), active = useRef(true);
  const load = async () => {
    if (lock.current) return;
    lock.current=true; setBusy(true); setError("");
    try { const r=await client.request("/api/v1/notifications/preferences"); if(active.current)setValue(r.data); }
    catch(e){if(active.current)setError(e.message);}
    finally{lock.current=false;if(active.current)setBusy(false);}
  };
  useEffect(()=>{active.current=true;load();return ()=>{active.current=false;};},[client]);
  const change = async (key,checked) => {
    if(lock.current || !value) return;
    lock.current=true;setBusy(true);setError("");
    try {
      const r=await client.request("/api/v1/notifications/preferences",{method:"PATCH",body:{clientRequestId:id(),expectedVersion:value.version,[key]:checked}});
      if(active.current)setValue(r.data);
    } catch(e){
      if(active.current)setError(e.message);
      if(e.status===409 && active.current){
        try {const latest=await client.request("/api/v1/notifications/preferences");if(active.current)setValue(latest.data);}
        catch(readError){if(active.current)setError(readError.message);}
      }
    } finally {lock.current=false;if(active.current)setBusy(false);}
  };
  return <section className="me-preferences">
    {onBack && <button className="me-subtle" type="button" onClick={onBack}><ArrowLeft size={16}/>返回我的</button>}
    <header className="me-page-heading"><h1>通知设置</h1><Button secondary onClick={load} disabled={busy}><RefreshCw size={16}/>刷新</Button></header>
    {error && <p className="me-alert" role="alert">{error}</p>}
    {!value && busy && <p role="status" className="me-status">正在读取通知偏好…</p>}
    {value && <div className="me-pref-grid">{notificationGroups.map(([group,topics])=><fieldset key={group} className="me-pref-group" disabled={busy} aria-label={group}><h2>{group}</h2>
      {topics.map(([key,label,description])=><Toggle key={key} label={label} description={description} checked={Boolean(value[key])} onChange={checked=>change(key,checked)}/>)}
    </fieldset>)}</div>}
  </section>;
}

function PasswordChangeForm({client,user,onChanged}){
  const [verification,setVerification]=useState(null),[code,setCode]=useState(""),[password,setPassword]=useState(""),[confirm,setConfirm]=useState("");
  const [busy,setBusy]=useState(false),[wait,setWait]=useState(0),[error,setError]=useState("");
  const lock=useRef(false);
  useEffect(()=>{if(!wait)return;const timer=setTimeout(()=>setWait(n=>n-1),1000);return()=>clearTimeout(timer);},[wait]);
  const send=async()=>{
    if(lock.current || wait)return;
    lock.current=true;setBusy(true);setError("");
    try {
      const r=await client.request("/api/v1/account/password-change-code",{method:"POST",body:{}});
      setVerification(r.data);setCode("");setWait(r.data.resendAfterSeconds??60);
    } catch(e){setError(e.message);} finally{lock.current=false;setBusy(false);}
  };
  const save=async event=>{
    event.preventDefault();if(lock.current || !verification)return;
    if(password!==confirm){setError("两次输入的密码不一致。");return;}
    lock.current=true;setBusy(true);setError("");
    try {
      await client.request("/api/v1/account/password-change",{method:"POST",body:{verificationToken:verification.verificationToken,code,newPassword:password}});
      setPassword("");setConfirm("");onChanged?.();
    } catch(e){setError(e.message);} finally{lock.current=false;setBusy(false);}
  };
  return <form className="me-password-form" onSubmit={save}>
    <p>{verification ? (verification.delivery==="EMAIL"?"验证码已发送至邮箱。":"请在游戏内查看验证码。") : user.email ? `验证码将发送至 ${contactValue(user.email,false,true)}。` : "请进入游戏接收验证码。"}</p>
    <div><Button secondary disabled={busy || wait>0} onClick={send}>{wait?`${wait} 秒后可重发`:"发送验证码"}</Button></div>
    {verification && <>
      <Field label={verification.delivery==="EMAIL"?"邮箱验证码":"游戏验证码"} value={code} onChange={e=>setCode(e.target.value.replace(/\D/g,"").slice(0,6))} required maxLength={6} inputMode="numeric" autoComplete="one-time-code"/>
      <Field label="新密码" type="password" value={password} onChange={e=>setPassword(e.target.value)} required minLength={8} maxLength={64} autoComplete="new-password"/>
      <Field label="再次输入新密码" type="password" value={confirm} onChange={e=>setConfirm(e.target.value)} required minLength={8} maxLength={64} autoComplete="new-password"/>
      <p className="muted">更改后，所有设备都需要使用新密码登录。</p>
    </>}
    {error&&<p className="auth-error" role="alert">{error}</p>}
    {verification && <footer><Button type="submit" disabled={busy || code.length!==6 || !password || !confirm}>确认更改密码</Button></footer>}
  </form>;
}

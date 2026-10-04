import React, { useEffect, useState } from "react";
import { ArrowRight, Check, LogOut, Mail, ShieldCheck, Sun, Moon } from "lucide-react";
import { Button, Field, media } from "./components.jsx";
import launcherIcon from "./assets/launcher-icons/default.svg";

import "./mail-support.css";

export default function GameVerification({ client, user, onVerified, onLogout, dark, toggleTheme }) {
  const [qq, setQQ] = useState(""), [code, setCode] = useState(""), [token, setToken] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""), [wait, setWait] = useState(0), [done, setDone] = useState(false), [support, setSupport] = useState(false);
  useEffect(() => { document.title = "验证游戏身份 · Deuterium ID"; }, []);
  useEffect(() => { if (!wait) return; const timer = setTimeout(() => setWait(n => n - 1), 1000); return () => clearTimeout(timer); }, [wait]);
  const send = async () => {
    if (busy || wait) return; setBusy(true); setError("");
    try { const r = await client.request("/api/v1/account/game-verification-code", { method: "POST", body: { qq } }); setToken(r.data.verificationToken); setCode(""); setWait(r.data.resendAfterSeconds || 60); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const submit = async e => {
    e.preventDefault(); if (busy) return; setBusy(true); setError("");
    try { await client.request("/api/v1/account/game-verification", { method: "POST", body: { verificationToken: token, code } }); setDone(true); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  useEffect(() => { if (!done) return; const timer = setTimeout(() => onVerified(), window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 550); return () => clearTimeout(timer); }, [done]);
  return <div className="auth-page verification-page">
    <header className="auth-header"><a className="auth-brand" href="/verify"><img src={launcherIcon} alt="" /><span>Deuterium <small>ID</small></span></a><div><button className="icon-button" aria-label="切换主题" onClick={toggleTheme}>{dark ? <Sun size={18} /> : <Moon size={18} />}</button><Button secondary onClick={onLogout}><LogOut size={16} />退出账号</Button></div></header>
    {support ? <div className="verification-support"><Button secondary onClick={() => setSupport(false)}>返回身份验证</Button><SupportContact client={client} user={user} /></div> : <div className="auth-layout">
      <section className="auth-story"><img className="auth-story-image" src={media("street-evening.png")} alt="Deuterium 游戏实景" /><div className="auth-story-shade" /><div className="auth-story-copy"><span>YOUR NEXT CHAPTER</span><h2>账号已就绪。<br />下一站，你的世界。</h2><p>完成游戏身份验证，<br />让社区里的你与游戏里的你相遇。</p></div></section>
      <main className="auth-form-section" id="main-content"><div className={`auth-form-wrap verification-content ${done ? "verified" : ""}`}>
        <div className="auth-emblem">{done ? <Check size={34} /> : <ShieldCheck size={34} />}</div><p className="eyebrow">DEUTERIUM ID · 02 / 02</p>
        <h1>{done ? "验证完成。" : "连接你的游戏身份。"}</h1><p className="auth-intro">{done ? "正在进入社区…" : "账号已创建并安全登录。完成这一步，即可使用 App 和玩家社区。"}</p>
        {!done && <><div className="verification-identity"><strong>{user.gameId}</strong><span><Mail size={14} />{user.email}</span></div>
          <form className="auth-form" onSubmit={submit}><Field label="QQ 号" value={qq} onChange={e => { setQQ(e.target.value.replace(/\D/g, "").slice(0,20)); setToken(null); }} inputMode="numeric" autoComplete="off" pattern="[0-9]{5,20}" required maxLength={20} hint="用于联系与身份核对。" />
            <p className="auth-verification-note">请使用 <strong>{user.gameId}</strong> 进入游戏，验证码会通过游戏内私聊发送。</p>
            <Button secondary disabled={busy || wait > 0 || !/^[0-9]{5,20}$/.test(qq)} onClick={send}>{wait ? `${wait} 秒后可重发` : token ? "重新发送游戏验证码" : "发送游戏验证码"}</Button>
            {token && <Field label="游戏内验证码" value={code} onChange={e => setCode(e.target.value.replace(/\D/g, "").slice(0,6))} inputMode="numeric" autoComplete="one-time-code" required maxLength={6} />}
            {error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" className="full auth-submit" disabled={busy || !token || code.length !== 6}>{busy ? "正在验证…" : "完成验证"}<ArrowRight size={18} /></Button>
          </form><p className="auth-register">暂时无法验证？<button onClick={() => setSupport(true)}>通过邮件联系我们</button></p></>}
      </div></main>
    </div>}
  </div>;
}

export function SupportContact({client,user}){
 const [address,setAddress]=useState(""),[error,setError]=useState("");
 useEffect(()=>{let alive=true;client.request("/api/v1/support/contact",{sessionRequired:false}).then(r=>{if(alive)setAddress(r.data.email||"")}).catch(e=>{if(alive)setError(e.message)});return()=>{alive=false}},[client]);
 return <section className="panel support-contact"><Mail size={32}/><h1>邮件联系支持</h1><p>请通过邮件说明问题类型、游戏 ID 和详细情况。收到来信后，我们会回复你的工单编号，并通过邮件持续跟进。</p>{address?<><strong>{address}</strong><a className="button" href={`mailto:${address}?subject=${encodeURIComponent("Deuterium 支持请求")}&body=${encodeURIComponent(`游戏 ID：${user?.gameId||""}\n问题类型：\n详细情况：\n`)}`}>打开邮件应用<ArrowRight size={16}/></a></>:<p role="status">{error||"玩家支持邮箱尚未配置。"}</p>}</section>
}

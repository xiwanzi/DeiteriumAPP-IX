import launcherIcon from "./assets/launcher-icons/default.svg";
import React, { useState, useEffect, useRef } from "react";
import {
  ArrowRight,
  ArrowLeft,
  Eye,
  EyeOff,
  UserRound,
  LockKeyhole,
  ChevronDown,
  Check,
  LoaderCircle,
  Sun,
  Moon,
  ExternalLink,
  MapPin,
} from "lucide-react";
import { Button, Field, media } from "./components.jsx";


export default function AuthScreen({
  onLogin,
  onVerify,
  onComplete,
  mode = "connected",
  blocked,
  statusMessage,
  dark = false,
  toggleTheme,

}) {
  const [screen, setScreen] = useState("login"),
    [account, setAccount] = useState(""),
    [password, setPassword] = useState(""),
    [visible, setVisible] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [scene, setScene] = useState(0),
    [verification, setVerification] = useState(null),
    [success, setSuccess] = useState("");
  const [email, setEmail] = useState("");
  const cooldowns = useRef(new Map());
  const recipient = `${screen}:${screen === 'register' ? email.trim().toLowerCase() : account.trim().toLowerCase()}`;
  const recipientRef = useRef(recipient); recipientRef.current = recipient;
  const noDelivery = new Set(['MINECRAFT_PLAYER_NOT_FOUND','MINECRAFT_LOOKUP_UNAVAILABLE','MAIL_NOT_READY','EMAIL_INVALID','GAME_ID_INVALID','ACCOUNT_ALREADY_EXISTS','FORBIDDEN','INVALID_REQUEST']);
  const [codeBusy, setCodeBusy] = useState(false), [resendAt, setResendAt] = useState(0), [remaining, setRemaining] = useState(0);
  const codeRequest = useRef(0);
  useEffect(() => {
    const tick = () => { const until=Math.max(cooldowns.current.get(recipientRef.current)?.until || 0,recipientRef.current.startsWith('login:')?0:cooldowns.current.get('code:*')?.until||0); setResendAt(until); setRemaining(Math.max(0, Math.ceil((until - Date.now()) / 1000))); };
    tick(); const timer = setInterval(tick, 250);
    return () => clearInterval(timer);
  }, [recipient]);
  useEffect(() => () => { codeRequest.current++; }, []);
  const invalidateCode = (nextRecipient=recipient) => { codeRequest.current++; setCodeBusy(false); const until=Math.max(cooldowns.current.get(nextRecipient)?.until||0,cooldowns.current.get('code:*')?.until||0);setResendAt(until);setRemaining(Math.max(0,Math.ceil((until-Date.now())/1000)));setVerification(null); setSuccess(""); setError(""); };
  useEffect(() => {
    document.title =
      (screen === "login"
        ? "登录"
        : screen === "register"
          ? "创建账号"
          : "找回密码") + " · Deuterium ID";
  }, [screen]);
  const scenes = [
    ["street-evening.png", "主城街道 · 夜幕时分"],
    ["street-daylight.png", "主城花桥 · 阳光正好"],
    ["plaza-sunset.png", "中心广场 · 每一次相遇"],
  ];
  const change = (s) => {
    invalidateCode();
    setResendAt(0);setRemaining(0);
    setScreen(s);
    setVerification(null);
    setSuccess("");
    setError("");
    setPassword("");
  };
  const submit = async (e) => {
    e.preventDefault();
    if (busy || codeBusy || (!verification && resendAt > Date.now())) return;
    if (screen !== "login" && !verification && onVerify) {
      const fields = Object.fromEntries(new FormData(e.currentTarget));
      if(screen==='register'&&!/^[A-Za-z0-9_]{3,16}$/.test(fields.gameId.trim())){setError('请填写 Minecraft Java 版玩家名：3–16 位字母、数字或下划线。');return;}
      const requestId = ++codeRequest.current, requestedAt = Date.now();
      const requestedRecipient=recipient;
      cooldowns.current.set(requestedRecipient,{until:requestedAt+60000,requestId});
      setCodeBusy(true); setResendAt(requestedAt + 60000); setRemaining(60); setError("");
      try {
        const result = await onVerify(screen, fields);
        if(cooldowns.current.get(requestedRecipient)?.requestId===requestId)cooldowns.current.set(requestedRecipient,{until:Date.now()+Math.max(1,Number(result.data?.resendAfterSeconds)||60)*1000,requestId});
        if (requestId !== codeRequest.current) return;
        if (!result.data?.verificationToken) throw new Error("未取得验证码凭证，请稍后重试。");
        setVerification(result.data);
        setResendAt(Math.max(requestedAt + 60000, Date.now() + Math.max(1, Number(result.data.resendAfterSeconds) || 60) * 1000));
        setSuccess("");
      } catch (error) {
        if(cooldowns.current.get(requestedRecipient)?.requestId===requestId){
          if(error.status===429){const entry={until:Date.now()+Math.max(1,Number(error.retryAfter)||60)*1000,requestId};cooldowns.current.set(requestedRecipient,entry);if(error.retryScope==='SOURCE')cooldowns.current.set('code:*',entry);}
          else if(noDelivery.has(error.code)||[400,403,409].includes(error.status))cooldowns.current.delete(requestedRecipient);
        }
        if (requestId === codeRequest.current) { setError(noDelivery.has(error.code)||error.status>=400&&error.status<500 ? error.message : "本次发送结果暂未确认，请查看邮箱；倒计时结束后可重试。"); }
      } finally { if (requestId === codeRequest.current) setCodeBusy(false);const until=Math.max(cooldowns.current.get(recipientRef.current)?.until||0,cooldowns.current.get('code:*')?.until||0);setResendAt(until);setRemaining(Math.max(0,Math.ceil((until-Date.now())/1000))); }
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (screen === "login")
        await onLogin({ account: account.trim(), password });
      else {
        const f = Object.fromEntries(new FormData(e.currentTarget));
        if (verification && onComplete) {
          if(Array.from(f.password).length<8||Array.from(f.password).length>64)throw new Error('密码需要 8–64 位。');
          await onComplete(screen, { ...f, verificationToken: verification.verificationToken });
          change("login");
          setSuccess(screen === "register" ? "账号已创建，请登录。" : "密码已重置，请使用新密码登录。");
        }
        else
          throw new Error(
            "身份验证服务暂不可用，请稍后重试。",
          );
      }
    } catch (e) {
      if(screen==='login'&&e.status===429){const seconds=Math.max(1,Number(e.retryAfter)||900);const until=Date.now()+seconds*1000;cooldowns.current.set(recipient,{until});setResendAt(until);setRemaining(seconds);}
      setError(screen==='register'&&['NETWORK_ERROR','INVALID_RESPONSE','SERVICE_UNAVAILABLE'].includes(e.code)?'注册结果暂未确认，请尝试使用刚才的邮箱和密码登录。':e.message || "连接暂时失败，请稍后重试");
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="auth-page">
      <header className="auth-header">
        <a href="/" className="auth-brand">
          <img src={launcherIcon} alt="Deuterium App 图标" />
          <span>
            Deuterium <small>ID</small>
          </span>
        </a>
        <div>
          <span>
            统一身份 · Deuterium ID
          </span>
          <button
            className="icon-button"
            aria-label={dark ? "切换浅色模式" : "切换深色模式"}
            onClick={toggleTheme}
          >
            {dark ? <Sun size={18} /> : <Moon size={18} />}
          </button>
        </div>
      </header>
      <div className="auth-layout">
        <section className="auth-story">
          <img
            className="auth-story-image"
            src={media(scenes[scene][0])}
            alt="Deuterium 服务器现有实景"
          />
          <div className="auth-story-shade" />
          <div className="auth-story-top">
            <span className="auth-world">
              <i /> DEUTERIUM COMMUNITY
            </span>
            <span>一个账号，连接所有热爱。</span>
          </div>
          <div className="auth-story-copy">
            <span>很高兴，再次遇见你。</span>
            <h2>
              你的世界，
              <br />
              不止在游戏里。
            </h2>
            <p>
              发现好物，交换灵感。
              <br />
              和熟悉的朋友，继续创造新的风景。
            </p>
            <div className="auth-feature-line">
              <span>商城</span>
              <i />
              <span>市场</span>
              <i />
              <span>委托</span>
              <i />
              <span>社区</span>
            </div>
          </div>
          <footer>
            <span>
              <MapPin size={14} />
              {scenes[scene][1]}
            </span>
            <div>
              {scenes.map((s, i) => (
                <button
                  key={s[0]}
                  aria-label={`查看${s[1]}`}
                  aria-pressed={scene === i}
                  className={scene === i ? "selected" : ""}
                  onClick={() => setScene(i)}
                />
              ))}
            </div>
          </footer>
        </section>
        <main id="main-content" className="auth-form-section">
          <div className="auth-form-wrap">
            {screen !== "login" && (
              <button className="auth-back" onClick={() => change("login")}>
                <ArrowLeft size={15} />
                返回登录
              </button>
            )}
            <div className="auth-emblem">
              <img src={launcherIcon} alt="Deuterium" />
            </div>
            <p className="eyebrow">ONE ID. YOUR WHOLE WORLD.</p>
            <h1>
              {screen === "login"
                ? "欢迎回来。"
                : screen === "register"
                  ? "开始你的新旅程。"
                  : "找回你的账号。"}
            </h1>
            {(statusMessage || success) && (
              <div className="auth-verification-note" role="status">
                {success || statusMessage}
              </div>
            )}
            <form className="auth-form" onSubmit={submit} onChange={(e) => { if (screen !== "login" && ["gameId", "email"].includes(e.target.name)) invalidateCode(`${screen}:${(screen==='register'?e.currentTarget.elements.email.value:e.currentTarget.elements.gameId.value).trim().toLowerCase()}`); }}>
              <div className="auth-input-group">
                <label htmlFor="auth-account">
                  {screen === "register" ? "游戏 ID" : "邮箱 / 游戏 ID / QQ"}
                </label>
                <div>
                  <UserRound size={17} />
                  <input
                    id="auth-account"
                    name="gameId"
                    autoComplete="username"
                    value={account}
                    onChange={(e) => setAccount(e.target.value)}
                    placeholder={
                      screen === "register"
                        ? "输入正版游戏 ID"
                        : "输入邮箱、游戏 ID 或 QQ"
                    }
                    required
                    maxLength={254}
                    readOnly={Boolean(verification)}
                  />
                </div>
              </div>
              {screen === "register" && (
                <Field
                  label="邮箱"
                  name="email"
                  value={email}
                  onChange={(e)=>setEmail(e.target.value)}
                  autoComplete="email"
                  placeholder="输入邮箱地址"
                  type="email"
                  maxLength={254}
                  readOnly={Boolean(verification)}
                  required
                />
              )}
              {screen === "login" ? (
                <>
                  <div className="auth-input-group">
                    <div className="auth-password-label">
                      <label htmlFor="auth-password">密码</label>
                      <button type="button" onClick={() => change("reset")}>
                        忘记密码？
                      </button>
                    </div>
                    <div>
                      <LockKeyhole size={17} />
                      <input
                        id="auth-password"
                        autoComplete="current-password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        type={visible ? "text" : "password"}
                        placeholder="输入你的密码"
                        required
                        maxLength={128}
                      />
                      <button
                        type="button"
                        aria-label={visible ? "隐藏密码" : "显示密码"}
                        aria-pressed={visible}
                        onClick={() => setVisible(!visible)}
                      >
                        {visible ? <EyeOff size={17} /> : <Eye size={17} />}
                      </button>
                    </div>
                  </div>
                </>
              ) : null}
              {verification && screen === "register" && <Field label="密码" name="password" type="password" autoComplete="new-password" required minLength={8} maxLength={64} />}
              {verification && screen !== "login" && <>
                <Field label="验证码" name="code" inputMode="numeric" autoComplete="one-time-code" required maxLength={12} />
                {screen === "reset" && <Field label="新密码" name="password" type="password" autoComplete="new-password" required minLength={8} maxLength={64} />}
              </>}
              {(error || blocked) && (
                <div className="auth-error" role="alert">
                  {error || "此账号已封禁，暂时无法登录，请联系管理员处理。"}
                </div>
              )}
              <Button
                type="submit"
                className="full auth-submit"
                disabled={busy || codeBusy || (!verification && remaining > 0)}
              >
                {!verification && remaining > 0 ? `${remaining} 秒后${screen==='login'?'重试':'重发'}` : busy ? (
                  <>
                    <LoaderCircle className="spin" size={17} />
                    正在连接…
                  </>
                ) : (
                  <>
                    {screen === "login"
                      ? "登录"
                      : verification ? (screen === "register" ? "创建账号" : "保存新密码") : "获取验证码"}
                    <ArrowRight size={18} />
                  </>
                )}
              </Button>
              {verification && <button type="button" className="text-button" disabled={busy} onClick={()=>invalidateCode()}>修改信息或重新获取验证码</button>}
            </form>
            {screen === "login" && (
              <p className="auth-register">
                还没有 Deuterium ID？
                <button onClick={() => change("register")}>
                  创建账号 <ArrowUpRightSmall />
                </button>
              </p>
            )}
          </div>
          <footer className="auth-form-footer">
            <span>DEUTERIUM ID</span>
            <span>你的身份，始终如一。</span>
          </footer>
        </main>
      </div>
    </div>
  );
}
function ArrowUpRightSmall() {
  return (
    <svg
      width="13"
      height="13"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
    >
      <path d="M7 17 17 7M7 7h10v10" />
    </svg>
  );
}

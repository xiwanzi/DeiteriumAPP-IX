import launcherIcon from "./assets/launcher-icons/default.svg";
import React, { useState, useEffect } from "react";
import {
  ArrowRight,
  ArrowLeft,
  Eye,
  EyeOff,
  UserRound,
  LockKeyhole,
  ShieldCheck,
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
    setScreen(s);
    setVerification(null);
    setSuccess("");
    setError("");
    setPassword("");
  };
  const submit = async (e) => {
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      if (screen === "login")
        await onLogin({ account: account.trim(), password });
      else {
        const f = Object.fromEntries(new FormData(e.currentTarget));
        if (verification && onComplete) {
          await onComplete(screen, { ...f, verificationToken: verification.verificationToken });
          change("login");
          setSuccess(screen === "register" ? "账号已创建，请登录。" : "密码已重置，请使用新密码登录。");
        } else if (onVerify) {
          const result = await onVerify(screen, f);
          if (!result.data?.verificationToken) throw new Error("未取得验证码凭证，请稍后重试。");
          setVerification(result.data);
          setSuccess("验证码已发送，请在游戏内查看。");
        }
        else
          throw new Error(
            "游戏内身份验证尚未接入。请使用已有账号登录，或联系管理员。",
          );
      }
    } catch (e) {
      setError(e.message || "连接暂时失败，请稍后重试");
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
            <p className="auth-intro">
              {screen === "login"
                ? "使用你的游戏 ID 或 QQ 登录 Deuterium。"
                : screen === "register"
                  ? "通过游戏内验证，把你的玩家身份与账号关联。"
                  : "验证游戏内身份后，为你的账号设置新密码。"}
            </p>
            {(statusMessage || success) && (
              <div className="auth-verification-note" role="status">
                {statusMessage || success}
              </div>
            )}
            <form className="auth-form" onSubmit={submit}>
              <div className="auth-input-group">
                <label htmlFor="auth-account">
                  {screen === "login" ? "游戏 ID / QQ" : "游戏 ID"}
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
                      screen === "login"
                        ? "输入游戏 ID 或 QQ"
                        : "你的完整游戏 ID"
                    }
                    required
                    maxLength={64}
                    readOnly={Boolean(verification)}
                  />
                </div>
              </div>
              {screen === "register" && (
                <Field
                  label="QQ"
                  name="qq"
                  placeholder="用于登录与身份核对"
                  inputMode="numeric"
                  required
                  pattern="[0-9]{5,20}"
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
              ) : (
                <div className="auth-verification-note">
                  <ShieldCheck size={20} />
                  <span>
                    请先进入游戏，验证码会通过游戏内私聊发送给你。
                  </span>
                </div>
              )}
              {screen === "register" && <Field label="密码" name="password" type="password" autoComplete="new-password" required minLength={8} maxLength={64} />}
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
                disabled={busy}
              >
                {busy ? (
                  <>
                    <LoaderCircle className="spin" size={17} />
                    正在连接…
                  </>
                ) : (
                  <>
                    {screen === "login"
                      ? "登录"
                      : verification ? (screen === "register" ? "创建账号" : "保存新密码") : verification ? (screen === "register" ? "创建账号" : "保存新密码") : screen === "register" ? "获取注册验证码" : "获取重置验证码"}
                    <ArrowRight size={18} />
                  </>
                )}
              </Button>
            </form>
            {screen === "login" && (
              <p className="auth-register">
                还没有 Deuterium ID？
                <button onClick={() => change("register")}>
                  创建账号 <ArrowUpRightSmall />
                </button>
              </p>
            )}
            <div className="auth-trust">
              <ShieldCheck size={15} />
              <span>与 App 共用账号 · 密码不会保存在网页中</span>
            </div>
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

import React, { useEffect, useRef, useState } from "react";
import { Button, Field } from "./components.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";
import "./account-sessions.css";

export default function ProfileConflictForm({ client, account, onSaved }) {
  const [email, setEmail] = useState(""), [type, setType] = useState("GAME_CODE"), [reference, setReference] = useState(""), [reason, setReason] = useState("");
  const [password, setPassword] = useState(""), [confirmed, setConfirmed] = useState(false), [token, setToken] = useState(""), [code, setCode] = useState("");
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [uncertain, setUncertain] = useState(false), [wait, setWait] = useState(0), [history, setHistory] = useState([]);
  const [correction, setCorrection] = useState(null);
  const pending = useRef(null), saving = useRef(false);
  useUnsavedChanges([email, type, reference, reason, confirmed], busy);
  useEffect(() => { let alive = true; client.request(`/api/v1/admin/accounts/${encodeURIComponent(account.userId)}/profile-resolutions`).then(r => { if (alive) setHistory(r.data.items); }).catch(e => { if (alive) setError(e.message); }); return () => { alive = false; }; }, [client, account.userId]);
  useEffect(() => { if (!wait) return; const timer = setTimeout(() => setWait(n => n - 1), 1000); return () => clearTimeout(timer); }, [wait]);
  const sendCode = async () => {
    if (saving.current || wait) return;
    saving.current = true; setBusy(true); setError("");
    try { const r = await client.request(`/api/v1/admin/accounts/${encodeURIComponent(account.userId)}/profile-release-code`, { method: "POST", body: {} }); setToken(r.data.verificationToken); setCode(""); setWait(r.data.resendAfterSeconds || 60); }
    catch (e) { setError(e.message); }
    finally { saving.current = false; setBusy(false); }
  };
  const save = async e => {
    e.preventDefault(); if (saving.current || !confirmed || !password) return;
    const input = { expectedVersion: account.version, claimantEmail: email.trim(), evidenceType: type, evidenceReference: reference.trim(), reason: reason.trim(), ownershipConfirmed: confirmed };
    const fingerprint = JSON.stringify(input);
    if (!pending.current || !uncertain && pending.current.fingerprint !== fingerprint) pending.current = { fingerprint, body: { ...input, clientRequestId: id() } };
    const body = { ...pending.current.body, password, ...(type === "GAME_CODE" ? { verificationToken: token, code } : {}) };
    saving.current = true; setBusy(true); setError(""); setPassword("");
    try { const r = await client.request(`/api/v1/admin/accounts/${encodeURIComponent(account.userId)}/profile-release`, { method: "POST", body }); await onSaved(r.data); }
    catch (e) { setError(e.message); setUncertain(e.status >= 500 || ["NETWORK_ERROR", "INVALID_RESPONSE"].includes(e.code)); }
    finally { saving.current = false; setBusy(false); }
  };
  return <div className="settings-stack">
    <p>原账号：<strong>{account.gameId || "待登记游戏身份"}</strong> · {account.email}</p>
    {!!account.gameId && !account.gameVerified && <form className="settings-stack" onSubmit={save}>
      <p className="muted">核实真正持有人后，解除此账号的未验证登记。原账号保留邮箱、密码和历史资料，之后用邮箱登录。此操作不授予游戏权限。</p>
      <Field label="真正持有人的邮箱" hint="持有人需验证此邮箱后继续注册。请核对工单中的地址。" type="email" required autoComplete="off" maxLength={254} disabled={busy || uncertain} value={email} onChange={e => setEmail(e.target.value)} />
      <Field label="所有权证明"><select disabled={busy || uncertain} value={type} onChange={e => setType(e.target.value)}><option value="GAME_CODE">可信游戏会话验证码</option><option value="SUPPORT_REVIEW">客服已完成人工核验</option></select></Field>
      {type === "GAME_CODE" ? <><p className="muted">让申诉人使用该正版账号进入游戏，验证码通过游戏内私聊发送。由申诉人通过工单提供验证码。</p><Button secondary disabled={busy || wait > 0 || uncertain} onClick={sendCode}>{wait ? `${wait} 秒后可重发` : "发送游戏验证码"}</Button><Field label="游戏验证码" inputMode="numeric" autoComplete="one-time-code" required maxLength={6} pattern="[0-9]{6}" value={code} disabled={busy} onChange={e => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))} /></> : <p className="muted">用于暂时无法进服的申诉。需有可靠的所有权核验记录；公开 UUID、游戏名和截图不能单独作为依据。证据不足时请继续处理工单。</p>}
      <Field label="工单编号与核验记录" required minLength={3} maxLength={240} disabled={busy || uncertain} value={reference} onChange={e => setReference(e.target.value)} />
      <Field label="处理原因"><textarea required minLength={3} maxLength={500} rows={3} disabled={busy || uncertain} value={reason} onChange={e => setReason(e.target.value)} /></Field>
      <label className="checkbox-field"><input type="checkbox" required checked={confirmed} disabled={busy || uncertain} onChange={e => setConfirmed(e.target.checked)} />已核实游戏账号所有权与认领邮箱</label>
      <Field label="当前管理员密码" type="password" required autoComplete="current-password" maxLength={256} disabled={busy} value={password} onChange={e => setPassword(e.target.value)} />
      {uncertain && <p role="status">处理结果尚未确认，请重新输入管理员密码，确认原请求的结果。</p>}
      <Button danger type="submit" disabled={busy || !confirmed || !password || type === "GAME_CODE" && (!token || code.length !== 6)}>{busy ? "正在处理…" : uncertain ? "确认处理结果" : "解除登记并预留给持有人"}</Button>
    </form>}
    {error && <p className="auth-error" role="alert">{error}</p>}
    {!!history.length && <div className="profile-resolution-history"><h3>处理记录</h3>{history.map(item => <article key={item.caseId}><strong>{item.previousGameId} · {item.claimedUserId ? "已继续注册" : item.pending ? "等待持有人注册" : "已结束"}</strong><p>{item.currentClaimantEmail}</p><p>{item.evidenceReference} · {item.reason}</p><small>{new Date(item.createdAt).toLocaleString("zh-CN")} · {item.actorId} · {item.evidenceType === "GAME_CODE" ? "游戏验证码" : "人工核验"}</small>{!!item.emailCorrections?.length && <details><summary>邮箱更正记录（{item.emailCorrections.length}）</summary>{item.emailCorrections.map(edit => <div key={edit.editId}><p>{edit.previousEmail} → {edit.nextEmail}</p><p>{edit.reason}</p><small>{new Date(edit.createdAt).toLocaleString("zh-CN")} · {edit.actorId}</small></div>)}</details>}{item.pending && <div><Button secondary disabled={busy || !!correction} onClick={() => setCorrection(item)}>更正认领邮箱</Button></div>}</article>)}</div>}
    {correction && <CorrectClaimEmail client={client} item={correction} onCancel={() => setCorrection(null)} onSaved={async () => { const r = await client.request(`/api/v1/admin/accounts/${encodeURIComponent(account.userId)}/profile-resolutions`); setHistory(r.data.items); setCorrection(null); }} />}
  </div>;
}

function CorrectClaimEmail({ client, item, onCancel, onSaved }) {
  const [email, setEmail] = useState(item.currentClaimantEmail), [reason, setReason] = useState(""), [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [uncertain, setUncertain] = useState(false);
  const pending = useRef(null), saving = useRef(false);
  useUnsavedChanges([email, reason], busy);
  const save = async e => {
    e.preventDefault(); if (saving.current) return;
    const data = { expectedEmail: item.currentClaimantEmail, claimantEmail: email.trim(), reason: reason.trim() }, fingerprint = JSON.stringify(data);
    if (!pending.current || !uncertain && fingerprint !== pending.current.fingerprint) pending.current = { fingerprint, body: { ...data, clientRequestId: id() } };
    const body = { ...pending.current.body, password };
    saving.current = true; setBusy(true); setPassword(""); setError("");
    try { await client.request(`/api/v1/admin/profile-reservations/${encodeURIComponent(item.caseId)}`, { method: "POST", body }); await onSaved(); }
    catch (e) { setError(e.message); setUncertain(e.status >= 500 || ["NETWORK_ERROR", "INVALID_RESPONSE"].includes(e.code)); }
    finally { saving.current = false; setBusy(false); }
  };
  return <form className="device-confirm settings-stack" onSubmit={save}><h3>更正认领邮箱</h3><p className="muted">仅能在持有人注册前更正，修改原因会保留在审计记录中。</p><Field label="确认后的持有人邮箱" type="email" required value={email} disabled={busy || uncertain} maxLength={254} onChange={e => setEmail(e.target.value)} /><Field label="更正原因" required minLength={3} maxLength={500} disabled={busy || uncertain} value={reason} onChange={e => setReason(e.target.value)} /><Field label="当前管理员密码" type="password" required autoComplete="current-password" value={password} disabled={busy} onChange={e => setPassword(e.target.value)} />{error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button secondary disabled={busy} onClick={onCancel}>取消</Button><Button type="submit" disabled={busy || !password}>{busy ? "正在保存…" : uncertain ? "确认更正结果" : "保存更正"}</Button></div></form>;
}

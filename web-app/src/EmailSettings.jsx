import React, { useEffect, useRef, useState } from "react";
import { Mail, RefreshCw, Save, Send } from "lucide-react";
import { Badge, Button, Field, PageHead } from "./components.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

export default function EmailSettings({ client }) {
  const [data, setData] = useState(null), [error, setError] = useState("");
  const load = async () => { try { const r = await client.request("/api/v1/admin/email-settings"); setData(r.data); setError(""); } catch (e) { setError(e.message); } };
  useEffect(() => { load(); }, []);
  return <>
    <PageHead eyebrow="PLATFORM NOTIFICATIONS" title="邮件提醒" subtitle="平台介入有新申请或状态更新时，及时通知负责处理的人。"><Button secondary onClick={load}><RefreshCw size={16} />刷新送达状态</Button></PageHead>
    {error && <p className="notice-box" role="alert">{error}</p>}
    {data ? <EmailForm client={client} initial={data.settings} delivery={data.delivery} secretReady={data.secretStorageReady} onSaved={load} /> : !error && <p role="status">正在读取邮件配置…</p>}
  </>;
}

function EmailForm({ client, initial, delivery, secretReady, onSaved }) {
  const [value, setValue] = useState(initial), [password, setPassword] = useState(""), [replacePassword, setReplacePassword] = useState(false), [recipients, setRecipients] = useState(initial.recipients.join("\n")), [busy, setBusy] = useState(false), [error, setError] = useState(""), [message, setMessage] = useState("");
  const request = useRef(null), testRequest = useRef(null);
  const guard = useUnsavedChanges({ value, password, replacePassword, recipients }, busy);
  const field = (key, label, props = {}) => <Field label={label} value={value[key]} onChange={(e) => setValue((old) => ({ ...old, [key]: props.type === "number" ? Number(e.target.value) : e.target.value }))} {...props} />;
  const save = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError(""); setMessage("");
    const { enabled, host, port, security, username, from } = value;
    const body = { enabled, host: host.trim(), port, security, username: username.trim(), from: from.trim(), recipients: recipients.split(/[\n,;，；]+/).map((v) => v.trim()).filter(Boolean), expectedVersion: value.version, ...((!initial.passwordConfigured || replacePassword) && password ? { password } : {}) };
    const fingerprint = JSON.stringify(body); if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, body: { ...body, clientRequestId: id() } };
    try { const r = await client.request("/api/v1/admin/email-settings", { method: "PUT", body: request.current.body }); setValue(r.data); setPassword(""); setReplacePassword(false); request.current = null; guard.markSaved({value:r.data,password:"",replacePassword:false,recipients}); setMessage("邮件配置已保存。新的介入状态将按此配置提醒。"); await onSaved(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const test = async () => {
    if (busy) return; setBusy(true); setError(""); setMessage(""); testRequest.current ||= id();
    try { await client.request("/api/v1/admin/email-settings/test", { method: "POST", body: { clientRequestId: testRequest.current } }); testRequest.current = null; setMessage("测试邮件已排队。请稍后刷新送达状态，并检查接收邮箱。"); await onSaved(); } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const labels = { PENDING: "等待发送", SENDING: "正在发送", RETRY: "失败，等待重试", SENT: "已被邮件服务器接收" };
  return <div className="email-layout">
    <form className="panel admin-form" onSubmit={save}>
      <h2><Mail size={21} />SMTP 配置</h2>
      <fieldset disabled={busy}>
        <label className="checkbox-row"><input type="checkbox" checked={value.enabled} onChange={(e) => setValue({ ...value, enabled: e.target.checked })} />启用平台介入邮件提醒</label>
        <div className="form-grid">{field("host", "SMTP 主机", { required: true, maxLength: 253, placeholder: "smtp.example.com", autoComplete: "off" })}{field("port", "端口", { required: true, type: "number", min: 1, max: 65535 })}</div>
        <Field label="连接加密"><select value={value.security} onChange={(e) => setValue({ ...value, security: e.target.value, port: e.target.value === "TLS" ? 465 : 587 })}><option value="STARTTLS">STARTTLS · 常用端口 587</option><option value="TLS">TLS · 常用端口 465</option></select></Field>
        {field("username", "SMTP 用户名", { maxLength: 320, autoComplete: "off", placeholder: "通常为发件邮箱" })}
        {initial.passwordConfigured && <label className="checkbox-row"><input type="checkbox" checked={replacePassword} onChange={(e) => setReplacePassword(e.target.checked)} />更换已保存的授权码</label>}
        {(!initial.passwordConfigured || replacePassword) && <Field label="密码 / 邮箱授权码" hint="保存后不再显示；可使用邮箱服务提供的 SMTP 专用授权码。" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} maxLength={2048} required={Boolean(value.username) && (!initial.passwordConfigured || replacePassword)} />}
        {!secretReady && <p className="notice-box">邮件密码存储尚未准备好，需要先完成服务器邮件密钥配置。</p>}
        {field("from", "发件邮箱", { required: true, type: "email", maxLength: 254 })}
        <Field label="接收邮箱" hint="最多 10 个，每行一个；也可以用逗号分隔。"><textarea required rows={4} value={recipients} onChange={(e) => setRecipients(e.target.value)} placeholder="admin@example.com" /></Field>
      </fieldset>
      {error && <p className="auth-error" role="alert">{error}</p>}{message && <p className="notice-box" role="status">{message}</p>}
      <div className="editor-actions"><Button type="submit" disabled={busy}><Save size={16} />{busy ? "正在处理…" : "保存配置"}</Button><Button secondary disabled={busy || !initial.enabled || guard.dirty} onClick={test}><Send size={16} />发送测试邮件</Button></div>
    </form>
    <section className="email-delivery"><div className="panel admin-form"><h2>送达状态</h2><div className="admin-metrics"><div><strong>{delivery.pending}</strong><span>待发送</span></div><div><strong>{delivery.retrying}</strong><span>等待重试</span></div></div><p className="muted">连接或认证失败会自动重试。邮件只包含案件编号、状态和管理入口。</p></div>
      <div className="panel">{delivery.events.length ? delivery.events.map((event) => <article className="delivery-event" key={event.eventId}><Badge tone={event.status === "SENT" ? "sage" : event.status === "RETRY" ? "red" : "amber"}>{labels[event.status]}</Badge><strong>{event.caseId || "配置测试邮件"}</strong><small>{new Date(event.createdAt).toLocaleString("zh-CN")} · 尝试 {event.attempts} 次</small>{event.lastError && <p className="auth-error">{event.lastError}</p>}</article>) : <p className="muted admin-form">还没有邮件发送记录。</p>}</div>
    </section>
  </div>;
}

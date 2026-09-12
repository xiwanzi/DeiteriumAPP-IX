import React, { useEffect, useRef, useState } from "react";
import { Mail, RefreshCw, Save, Send, Eye } from "lucide-react";
import { Badge, Button, Field, Modal, PageHead } from "./components.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

const kinds = { TEST: "连接测试", INTERVENTION: "平台介入提醒", ADMISSION_APPROVED: "白名单审核通过", ADMISSION_REJECTED: "白名单审核拒绝", TEST_APPROVED: "通过邮件模板测试", TEST_REJECTED: "拒绝邮件模板测试" };
const statuses = { PENDING: "等待发送", SENDING: "正在发送", RETRY: "失败，等待重试", SENT: "已被邮件服务器接收", SKIPPED: "提醒关闭，已跳过", CANCELLED: "旧通知已取消" };

export default function EmailSettings({ client }) {
  const [data, setData] = useState(null), [error, setError] = useState("");
  const active = useRef(true), loading = useRef(false);
  const load = async () => {
    if (loading.current) return;
    loading.current = true;
    try { const r = await client.request("/api/v1/admin/email-settings"); if (active.current) { setData(old => old && old.settings.version > r.data.settings.version ? { ...r.data, settings: old.settings } : r.data); setError(""); } }
    catch (e) { if (active.current) setError(e.message); }
    finally { loading.current = false; }
  };
  useEffect(() => { active.current = true; load(); const timer = setInterval(load, 5000); return () => { active.current = false; clearInterval(timer); }; }, [client]);
  const saved = async settings => { if (settings) setData(old => ({ ...old, settings })); await load(); };
  return <>
    <PageHead eyebrow="PLATFORM NOTIFICATIONS" title="邮件提醒" subtitle="管理平台提醒与白名单审核结果邮件。"><Button secondary onClick={load}><RefreshCw size={16} />刷新投递状态</Button></PageHead>
    {error && <p className="notice-box" role="alert">{error}</p>}
    {data ? <EmailForm client={client} initial={data.settings} delivery={data.delivery} secretReady={data.secretStorageReady} onSaved={saved} /> : !error && <p role="status">正在读取邮件配置…</p>}
  </>;
}

function EmailForm({ client, initial, delivery, secretReady, onSaved }) {
  const [value, setValue] = useState(initial), [password, setPassword] = useState(""), [replacePassword, setReplacePassword] = useState(false), [recipients, setRecipients] = useState(initial.recipients.join("\n"));
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [message, setMessage] = useState(""), [preview, setPreview] = useState(null);
  const request = useRef(null), testRequest = useRef(null), previewGeneration = useRef(0);
  const guard = useUnsavedChanges({ value, password, replacePassword, recipients }, busy);
  useEffect(() => {
    if (!guard.dirty && !busy) {
      const nextRecipients = initial.recipients.join("\n");
      setValue(initial); setRecipients(nextRecipients);
      guard.markSaved({ value: initial, password, replacePassword, recipients: nextRecipients });
    }
  }, [initial]);
  useEffect(() => () => { previewGeneration.current++; }, []);
  const field = (key, label, props = {}) => <Field label={label} value={value[key]} onChange={(e) => {
    const next = props.type === "number" ? Number(e.target.value) : e.target.value;
    setValue((old) => ({ ...old, [key]: next, ...(key === "port" && [465, 587].includes(next) ? { security: next === 465 ? "TLS" : "STARTTLS" } : {}) }));
  }} {...props} />;
  const save = async event => {
    event.preventDefault(); if (busy) return; setBusy(true); setError(""); setMessage("");
    const { enabled, host, port, security, username, from, admissionReviewEnabled } = value;
    const body = { enabled, admissionReviewEnabled: Boolean(admissionReviewEnabled), host: host.trim(), port, security, username: username.trim(), from: from.trim(), recipients: recipients.split(/[\n,;，；]+/).map(v => v.trim()).filter(Boolean), expectedVersion: value.version, ...((!initial.passwordConfigured || replacePassword) && password ? { password } : {}) };
    const fingerprint = JSON.stringify(body);
    if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, body: { ...body, clientRequestId: id() } };
    try {
      const r = await client.request("/api/v1/admin/email-settings", { method: "PUT", body: request.current.body });
      setValue(r.data); setPassword(""); setReplacePassword(false); request.current = null;
      guard.markSaved({ value: r.data, password: "", replacePassword: false, recipients });
      setMessage("邮件配置已保存。"); await onSaved(r.data);
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const test = async (template = "CONNECTION") => {
    if (busy || guard.dirty) return; setBusy(true); setError(""); setMessage("");
    if (testRequest.current?.template !== template) testRequest.current = { template, key: id() };
    try {
      await client.request("/api/v1/admin/email-settings/test", { method: "POST", body: { clientRequestId: testRequest.current.key, template } });
      testRequest.current = null; setMessage("测试邮件已排队，将发送到下方配置的管理提醒 / 测试收件邮箱。"); await onSaved();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const showPreview = async (decision) => {
    const generation = ++previewGeneration.current;
    setPreview({ decision, loading: true });
    try { const r = await client.request(`/api/v1/admin/email-settings/templates/${decision}`); if (generation === previewGeneration.current) setPreview({ decision, ...r.data }); }
    catch (e) { if (generation === previewGeneration.current) setPreview({ decision, error: e.message }); }
  };
  const testDisabled = busy || !initial.enabled || guard.dirty;
  return <div className="email-layout">
    <div className="email-settings-stack">
      <form className="panel admin-form" onSubmit={save}>
        <h2><Mail size={21} />SMTP 配置</h2>
        <fieldset disabled={busy}>
          <label className="checkbox-row"><input type="checkbox" checked={value.enabled} onChange={e => setValue({ ...value, enabled: e.target.checked })} />启用邮件发送</label>
          <p className="muted">关闭后暂停全部待发送邮件，恢复后继续投递。</p>
          <div className="form-grid">{field("host", "SMTP 主机", { required: true, maxLength: 253, placeholder: "smtp.example.com", autoComplete: "off" })}{field("port", "端口", { required: true, type: "number", min: 1, max: 65535 })}</div>
          <Field label="连接加密"><select value={value.security} onChange={e => setValue({ ...value, security: e.target.value, port: e.target.value === "TLS" ? 465 : 587 })}><option value="STARTTLS">STARTTLS · 常用端口 587</option><option value="TLS">TLS · 常用端口 465</option></select></Field>
          {field("username", "SMTP 用户名", { maxLength: 320, autoComplete: "off", placeholder: "通常为发件邮箱" })}
          {initial.passwordConfigured && <label className="checkbox-row"><input type="checkbox" checked={replacePassword} onChange={e => setReplacePassword(e.target.checked)} />更换已保存的授权码</label>}
          {(!initial.passwordConfigured || replacePassword) && <Field label="密码 / 邮箱授权码" hint="保存后不再显示，可使用邮箱提供的 SMTP 专用授权码。" type="password" autoComplete="new-password" value={password} onChange={e => setPassword(e.target.value)} maxLength={2048} required={Boolean(value.username) && (!initial.passwordConfigured || replacePassword)} />}
          {!secretReady && <p className="notice-box">邮件密码存储尚未准备好，需要先完成服务器邮件密钥配置。</p>}
          {field("from", "发件邮箱", { required: true, type: "email", maxLength: 254 })}
          <Field label="管理提醒 / 测试收件邮箱" hint="平台介入提醒和模板测试发送至此列表，最多 10 个，每行一个。"><textarea required rows={3} value={recipients} onChange={e => setRecipients(e.target.value)} placeholder="admin@example.com" /></Field>
          <div className="email-admission-switch"><label className="checkbox-row"><input type="checkbox" checked={Boolean(value.admissionReviewEnabled)} onChange={e => setValue({ ...value, admissionReviewEnabled: e.target.checked })} />发送白名单审核结果邮件</label><p className="muted">通过或拒绝后，发送至申请人填写的 QQ 号码对应邮箱。关闭时的新审核记录会跳过，已排队的审核邮件暂缓。</p></div>
        </fieldset>
        {error && <p className="auth-error" role="alert">{error}</p>}{message && <p className="notice-box" role="status">{message}</p>}
        <div className="editor-actions"><Button type="submit" disabled={busy}><Save size={16} />{busy ? "正在处理…" : "保存配置"}</Button><Button secondary disabled={testDisabled} onClick={() => test()}><Send size={16} />发送连接测试</Button></div>
      </form>
      <section className="panel admin-form email-templates"><h2>白名单审核邮件</h2><p className="muted">正式通知使用申请人的游戏 ID 和本次审核结果；拒绝原因由审核人填写。</p>
        {[{ decision: "APPROVED", title: "通行许可已签发", text: "申请通过后，向玩家发送通行凭证与欢迎信息。" }, { decision: "REJECTED", title: "申请暂未通过", text: "说明真实拒绝原因，并提供返回申请站的入口。" }].map(template => <article className="email-template" key={template.decision}><strong>{template.title}</strong><p className="muted">{template.text}</p><div className="button-row"><Button secondary onClick={() => showPreview(template.decision)}><Eye size={15} />预览{template.decision === "APPROVED" ? "通过" : "拒绝"}邮件</Button><Button secondary disabled={testDisabled} onClick={() => test(template.decision)}><Send size={15} />测试{template.decision === "APPROVED" ? "通过" : "拒绝"}邮件</Button></div></article>)}
        {guard.dirty && <p className="muted">保存当前配置后，即可发送测试邮件。</p>}
      </section>
    </div>
    <section className="email-delivery"><div className="panel admin-form"><h2>投递状态</h2><div className="admin-metrics"><div><strong>{delivery.pending}</strong><span>待发送</span></div><div><strong>{delivery.retrying}</strong><span>等待重试</span></div></div><p className="muted">失败后自动重试。已被后续申请或资格变化取代的审核通知会取消。以下显示最近 20 封邮件。</p></div>
      <div className="panel">{delivery.events.length ? delivery.events.map(event => <article className="delivery-event" key={event.eventId}><Badge tone={event.status === "SENT" ? "sage" : event.status === "RETRY" ? "red" : ["SKIPPED", "CANCELLED"].includes(event.status) ? "neutral" : "amber"}>{statuses[event.status] || event.status}</Badge><strong>{kinds[event.kind] || "邮件提醒"}{event.gameId ? ` · ${event.gameId}` : ""}</strong><small>{event.recipient ? `收件人：${event.recipient}` : "历史记录未保存收件地址"}</small>{(event.applicationId || event.caseId) && <small>{event.applicationId || event.caseId}</small>}<small>{new Date(event.createdAt).toLocaleString("zh-CN")} · 尝试 {event.attempts} 次</small>{event.sentAt && <small>接收时间：{new Date(event.sentAt).toLocaleString("zh-CN")}</small>}{event.lastError && <p className={event.status === "RETRY" ? "auth-error" : "muted"}>{event.lastError}</p>}</article>) : <p className="muted admin-form">还没有邮件发送记录。</p>}</div>
    </section>
    {preview && <Modal title={preview.decision === "APPROVED" ? "通过邮件预览" : "拒绝邮件预览"} wide close={() => { previewGeneration.current++; setPreview(null); }}>{preview.loading ? <p role="status">正在读取邮件模板…</p> : preview.error ? <p className="auth-error" role="alert">{preview.error}</p> : <><p className="muted">{preview.subject}</p><iframe className="email-template-preview" title="邮件模板效果" sandbox="" referrerPolicy="no-referrer" srcDoc={preview.html} /></>}</Modal>}
  </div>;
}

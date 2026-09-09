import React, { useEffect, useRef, useState } from "react";
import { Plus, Save, BookOpen, Sparkles, Trash2 } from "lucide-react";
import { Badge, Button, Field, PageHead, Tabs, Toggle } from "./components.jsx";
import { id, credit } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

export default function AiSettings({ client }) {
  const [value, setValue] = useState(null), [section, setSection] = useState("套餐与开放"), [error, setError] = useState(""), [saved, setSaved] = useState(""), [busy, setBusy] = useState(false), [provider, setProvider] = useState(false);
  const pending = useRef(null), changes = useUnsavedChanges(value, busy);
  const load = async () => { setError(""); const r = await client.request("/api/v1/admin/ai-settings"); setValue(r.data.settings); changes.markSaved(r.data.settings); setProvider(r.data.providerConfigured); };
  useEffect(() => { load().catch((e) => setError(e.message)); }, [client]);
  const set = (key, v) => { setSaved(""); setValue((old) => ({ ...old, [key]: v })); };
  const plan = (index, key, v) => set("plans", value.plans.map((p, i) => i === index ? { ...p, [key]: v } : p));
  const entry = (index, key, v) => set("knowledge", value.knowledge.map((k, i) => i === index ? { ...k, [key]: v } : k));
  const save = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError(""); setSaved("");
    const settings = { ...value, plans: value.plans.map((p) => ({ ...p, price: p.code === "free" ? "0.00" : p.price })) }, fingerprint = JSON.stringify(settings);
    if (pending.current?.fingerprint !== fingerprint) pending.current = { fingerprint, body: { clientRequestId: id(), expectedVersion: value.version, settings } };
    try { await client.request("/api/v1/admin/ai-settings", { method: "PUT", body: pending.current.body }); await load(); setSaved("设置已保存，将用于之后的新对话请求和新订单。"); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <div className="ai-settings"><PageHead eyebrow="SAKI AI" title="小祥设置" subtitle="管理套餐、回复风格与知识，让小祥更了解社区。"><Button secondary disabled={busy} onClick={() => { if (changes.dirty && !window.confirm("重新读取会放弃未保存的修改，继续？")) return; load().catch((e) => setError(e.message)); }}>重新读取</Button></PageHead>
    {error && <p role="alert" className="auth-error">{error}</p>}{!value && <Button secondary onClick={load}>重新读取</Button>}
    {value && <form onSubmit={save}><fieldset disabled={busy} className="settings-fieldset">
      <div className="settings-banner"><Sparkles size={26} /><div><strong>{value.assistantName || "小祥"}</strong><p>{provider ? "已连接服务器配置的 AI 服务" : "请先在服务器配置 AI 服务凭据"}</p></div><Badge tone={value.enabled ? "sage" : "neutral"}>{value.enabled ? "服务已开启" : "服务已关闭"}</Badge></div>
      <Tabs values={["套餐与开放", "回复与参数", "系统提示词", "知识库"]} value={section} onChange={setSection} label="小祥设置分类" />
      {section === "套餐与开放" && <div className="settings-stack">
        <section className="settings-panel"><h2>服务与购买</h2><Toggle label="开放 AI 服务" description="控制小祥是否接受新的对话请求" checked={value.enabled} onChange={(v) => set("enabled", v)} /><Toggle label="开放套餐购买" description="开启后，玩家可使用信用点购买已启用的套餐" checked={value.paidPlansEnabled} onChange={(v) => set("paidPlansEnabled", v)} /><Toggle label="管理员不限额度" description="平台管理员使用 AI 时不消耗额度" checked={value.adminQuotaExempt} onChange={(v) => set("adminQuotaExempt", v)} /></section>
        <div className="section-head"><div><h2>套餐</h2><p className="muted">按有效天数购买，不自动续费；已购订单的权益保留。</p></div><Button secondary disabled={value.plans.length >= 20} onClick={() => { const code = `custom_${id().replaceAll("-", "").slice(0, 12)}`; set("plans", [...value.plans, { planId: `plan_${code}`, code, name: "新套餐", description: "", price: "10.00", currency: "CREDIT", quotaPerWindow: 50, windowHours: 24, durationDays: 30, modelTier: "flash", active: false, purchasable: false, version: 0 }]); }}><Plus size={16} />添加套餐</Button></div>
        <div className="plan-editor-grid">{value.plans.map((p, index) => <section className="settings-panel" key={p.planId}><div className="section-head"><h3>{p.name}</h3><Badge tone="neutral">{p.code === "free" ? "免费" : p.active && value.paidPlansEnabled ? "可购买" : "未开放"}</Badge></div>
          <Field label="套餐名称" value={p.name} maxLength={80} required onChange={(e) => plan(index, "name", e.target.value)} /><Field label="套餐说明"><textarea value={p.description} rows={2} maxLength={1000} onChange={(e) => plan(index, "description", e.target.value)} /></Field>
          <div className="form-grid"><Field label="价格 · 信用点" type="number" min={p.code === "free" ? 0 : 0.01} max="1000000000000" step="0.01" disabled={p.code === "free"} required value={p.price} onChange={(e) => plan(index, "price", e.target.value)} /><Field label="有效天数" type="number" min={p.code === "free" ? 0 : 1} max={3650} disabled={p.code === "free"} required value={p.durationDays} onChange={(e) => plan(index, "durationDays", Number(e.target.value))} /></div>
          <div className="form-grid"><Field label="每时段额度 · 次" type="number" min={1} max={100000} required value={p.quotaPerWindow} onChange={(e) => plan(index, "quotaPerWindow", Number(e.target.value))} /><Field label="重置时段 · 小时" type="number" min={1} max={168} required value={p.windowHours} onChange={(e) => plan(index, "windowHours", Number(e.target.value))} /></div>
          {p.code !== "free" && <Toggle label="启用此套餐" description={`${credit(p.price)} 信用点 / ${p.durationDays} 天`} checked={p.active} onChange={(v) => plan(index, "active", v)} />}
        </section>)}</div>
      </div>}
      {section === "回复与参数" && <section className="settings-panel settings-stack"><h2>回复与生成</h2><div className="form-grid"><Field label="助手显示名称" value={value.assistantName} required maxLength={80} onChange={(e) => set("assistantName", e.target.value)} /><Field label="模型名称" value={value.model} required maxLength={120} onChange={(e) => set("model", e.target.value)} /></div>
        <Toggle label="联网查找资料" description="按问题需要查询新信息，并展示来源" checked={value.webSearch} onChange={(v) => set("webSearch", v)} />
        <div className="form-grid"><Field label="思考程度"><select value={value.reasoningEffort} onChange={(e) => set("reasoningEffort", e.target.value)}><option value="none">关闭</option><option value="low">轻量</option><option value="high">深入</option></select></Field><Field label="回复随机性" hint="0 更稳定，2 更有变化" type="number" min={0} max={2} step={0.1} value={value.temperature} required onChange={(e) => set("temperature", Number(e.target.value))} /></div>
        <div className="form-grid">{[["maxInputChars", "单次输入字数", 1, 2000], ["maxContextMessages", "保留上下文条数", 2, 40], ["maxOutputTokens", "最大输出 Token", 256, 16384], ["timeoutSeconds", "回复超时 · 秒", 5, 300], ["maxConcurrent", "同时回复数", 1, 16]].map(([key, label, min, max]) => <Field key={key} label={label} type="number" min={min} max={max} required value={value[key]} onChange={(e) => set(key, Number(e.target.value))} />)}</div>
      </section>}
      {section === "系统提示词" && <section className="settings-panel settings-stack"><h2>小祥如何回答</h2><p className="muted">设定语气、角色和回答要求。知识内容可单独放入知识库。</p><Field label="系统提示词" hint={`${value.systemPrompt.length} / 32000 字`}><textarea className="prompt-editor" rows={18} value={value.systemPrompt} maxLength={32000} onChange={(e) => set("systemPrompt", e.target.value)} /></Field></section>}
      {section === "知识库" && <div className="settings-stack"><div className="section-head"><div><h2>参考知识</h2><p className="muted">启用的内容会随问题提供给小祥，适合社区规则、常见问题与活动说明。</p></div><Button secondary disabled={value.knowledge.length >= 50} onClick={() => set("knowledge", [...value.knowledge, { id: id(), title: "新知识", content: "", enabled: true }])}><Plus size={16} />添加知识</Button></div>
        {!value.knowledge.length && <div className="settings-empty"><BookOpen size={34} /><h3>从一条常见问题开始</h3><p>添加准确的资料，帮助小祥给出更贴近社区的回答。</p></div>}
        {value.knowledge.map((k, index) => <section key={k.id} className="settings-panel settings-stack"><Field label="标题" value={k.title} maxLength={120} required onChange={(e) => entry(index, "title", e.target.value)} /><Field label="内容"><textarea rows={6} maxLength={12000} required value={k.content} onChange={(e) => entry(index, "content", e.target.value)} /></Field><div className="knowledge-actions"><Toggle label="参与回答" checked={k.enabled} onChange={(v) => entry(index, "enabled", v)} /><Button secondary danger onClick={() => set("knowledge", value.knowledge.filter((_, i) => i !== index))}><Trash2 size={15} />移除</Button></div></section>)}
      </div>}
    </fieldset><div className="settings-save"><span role="status">{saved || "更改完成后统一保存"}</span><Button type="submit" disabled={busy}><Save size={16} />{busy ? "正在保存…" : "保存设置"}</Button></div></form>}
  </div>;
}

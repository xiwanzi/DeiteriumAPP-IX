import React, { useEffect, useRef, useState } from "react";
import { RefreshCw, Plus, Search, Layers, History, Users, PersonStanding, Activity, Settings2, ArrowUpRight } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, Tabs } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";
import "./ysm-cloud.css";

const prefix = "/api/v1/admin/ysm/cloud/";
const views = [["models", "模型库", Layers], ["releases", "发布记录", History], ["players", "玩家外观", Users], ["npcs", "NPC 外观", PersonStanding], ["activity", "变更记录", Activity], ["connection", "连接与存储", Settings2]];
const descriptions = { models: "发布、更新和管理服务器中的模型。", releases: "每次发布都有记录，历史版本可以恢复。", players: "查看已验证玩家的模型选择。", npcs: "管理 Citizens NPC 的模型、贴图和动作。", activity: "追踪模型和 NPC 外观的变更。", connection: "查看模型服务、分发方式和游戏节点。" };
const bytes = (n = 0) => n < 1024 ? `${n} B` : n < 1048576 ? `${(n / 1024).toFixed(1)} KB` : `${(n / 1048576).toFixed(1)} MB`;
const when = v => { if (!v) return "—"; const d = new Date(typeof v === "number" ? v * (v < 1e12 ? 1000 : 1) : /^\d{4}-\d\d-\d\d /.test(v) ? v.replace(" ", "T") + "Z" : v); return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString("zh-CN", { hour12: false }); };
const digest = async data => [...new Uint8Array(await crypto.subtle.digest("SHA-256", typeof data === "string" ? new TextEncoder().encode(data) : await data.arrayBuffer()))].map(x => x.toString(16).padStart(2, "0")).join("");
const seg = encodeURIComponent, listKey = { releases: "releases", players: "players", npcs: "npcs", activity: "events" };

export default function YsmCloudManagement({ client }) {
  const [view, setView] = useState("models"), [data, setData] = useState(null), [connection, setConnection] = useState(null), [lists, setLists] = useState({});
  const [loading, setLoading] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState(""), [notice, setNotice] = useState(""), [updated, setUpdated] = useState(null);
  const [query, setQuery] = useState(""), [filter, setFilter] = useState("已发布"), [limit, setLimit] = useState(24), [editor, setEditor] = useState(null), [detail, setDetail] = useState(null), [confirmation, setConfirmation] = useState(null);
  const alive = useRef(true), acting = useRef(false), reading = useRef(false), currentView = useRef(view), modalOpen = useRef(false);
  currentView.current = view; modalOpen.current = !!(editor || detail || confirmation);
  useUnsavedChanges(null, busy);
  const get = async path => (await client.request(prefix + path)).data;
  const post = async (path, body, options = {}) => (await client.request(prefix + path, { method: "POST", body, signal: AbortSignal.timeout(310000), ...options })).data;
  const load = async () => {
    if (reading.current || acting.current) return;
    reading.current = true; setLoading(true); const tab = currentView.current;
    try {
      const config = await get("connection"); if (!alive.current) return; setConnection(config);
      const next = await get("overview"); if (!alive.current) return; setData(next);
      if (listKey[tab]) { const result = await get(tab); if (!alive.current) return; setLists(old => ({ ...old, [tab]: result[listKey[tab]] || [] })); }
      setUpdated(new Date()); setError("");
    } catch (e) { if (alive.current) setError(e.message); }
    finally { reading.current = false; if (alive.current) setLoading(false); }
  };
  useEffect(() => { alive.current = true; load(); const timer = setInterval(() => { if (!document.hidden && !modalOpen.current) load(); }, 15000); return () => { alive.current = false; clearInterval(timer); }; }, [client]);
  useEffect(() => { setQuery(""); setLimit(24); load(); }, [view]);
  const run = async work => {
    if (acting.current) return; acting.current = true; setBusy(true); setError(""); setNotice("");
    let succeeded = false;
    try { await work(); succeeded = true; } catch (e) { if (alive.current) setError(e.message); }
    finally { acting.current = false; if (alive.current) { setBusy(false); if (succeeded) await load(); } }
  };
  const models = data?.models || [], matching = models.filter(m => (filter === "全部" || m.status === (filter === "已发布" ? "published" : "archived")) && `${m.name} ${m.modelId}`.toLowerCase().includes(query.toLowerCase()));
  const rows = (lists[view] || []).filter(r => JSON.stringify(r).toLowerCase().includes(query.toLowerCase()));
  const close = setter => () => { if (!acting.current) setter(null); };
  const modelName = id => models.find(m => m.assetId === id)?.name || id || "默认外观";
  const openDetail = m => run(async () => { const result = await get(`models/${seg(m.assetId)}/versions`); if (alive.current) setDetail({ ...m, versions: result.versions }); });
  const publish = form => run(async () => {
    const model = editor.model, sha = await digest(form.file);
    if (model?.sha256 === sha) { setNotice("文件内容没有变化，无需重新发布。"); setEditor(null); return; }
    const result = await post("upload", form.file, { idempotencyKey: crypto.randomUUID(), ysmCloudUpload: {
      assetId: model?.assetId || "d9_" + (await digest(form.id)).slice(0, 40), name: form.name,
      format: form.file.name.toLowerCase().endsWith(".zip") ? "zip" : "ysm", sha256: sha, revision: model?.version || 0,
      metadata: { modelId: form.id, category: form.id.includes("/") ? form.id.slice(0, form.id.lastIndexOf("/")) : "", note: form.note },
    } });
    if (alive.current) { setEditor(null); setNotice(`${form.name} 的 v${result.revision} 已发布。`); }
  });
  return <section className="ysm-cloud">
    <div className="ysm-cloud-heading"><div><span className="ysm-kicker">DEUTERIUM · MODEL STUDIO</span><h1>模型中心</h1><p>一次发布，共享给每位玩家。</p></div><div className="ysm-cloud-actions"><span className="muted">{updated ? `更新于 ${updated.toLocaleTimeString("zh-CN", { hour12: false })}` : "等待连接"}</span><Button secondary disabled={busy || loading} onClick={load} aria-label="刷新模型中心"><RefreshCw size={16} /></Button></div></div>
    <nav className="ysm-cloud-nav" aria-label="模型中心导航">{views.map(([key, label, Icon]) => <button key={key} type="button" disabled={busy || loading} aria-current={view === key ? "page" : undefined} onClick={() => setView(key)}><Icon size={17} />{label}{key === "models" && data && <span>{models.length}</span>}</button>)}</nav>
    {error && <p className="error-banner" role="alert">{error}</p>}{notice && <p className="success-banner" role="status">{notice}</p>}
    <div className="ysm-cloud-metrics">{[["已发布模型", data ? models.filter(m => m.status === "published").length : "—", "按模型独立更新"], ["模型资源", data ? bytes(models.reduce((n, m) => n + m.bytes, 0)) : "—", "相同文件共享存储"], ["已验证玩家", data?.verifiedPlayers ?? "—", "使用游戏身份验证"], ["NPC 外观", data?.npcCount ?? "—", "保存后等待节点应用"]].map(([label, value, hint]) => <div key={label}><span>{label}</span><strong>{value}</strong><small>{hint}</small></div>)}</div>
    <div className="ysm-cloud-section-head"><div><h2>{views.find(v => v[0] === view)[1]}</h2><p>{descriptions[view]}</p></div>{view === "models" && <Button disabled={busy || !data} onClick={() => { setError(""); setEditor({ kind: "upload" }); }}><Plus size={17} /> 发布模型</Button>}</div>
    {view !== "connection" && <div className="ysm-cloud-toolbar">{view === "models" && <Tabs values={["已发布", "已下架", "全部"]} value={filter} onChange={v => { setFilter(v); setLimit(24); }} />}<label className="ysm-cloud-search"><Search size={17} /><input aria-label="搜索模型中心" value={query} placeholder={view === "models" ? "搜索名称或模型 ID" : "搜索当前列表"} onChange={e => { setQuery(e.target.value); setLimit(24); }} /></label></div>}
    {view === "models" && <><div className="ysm-cloud-models">{matching.slice(0, limit).map(m => <button className="ysm-cloud-model" key={m.assetId} disabled={busy} onClick={() => openDetail(m)} aria-label={`查看 ${m.name}`}><span className="ysm-model-art"><span>{m.format.toUpperCase()}</span><Badge tone={m.status === "published" ? "sage" : "neutral"}>{m.status === "published" ? "已发布" : "已下架"}</Badge><strong aria-hidden="true">{[...m.name].slice(0, 2).join("")}</strong></span><span className="ysm-model-description"><strong>{m.name}</strong><small>{m.modelId}</small><span>v{m.version} · {bytes(m.bytes)}<ArrowUpRight size={16} /></span></span></button>)}</div>{!matching.length && <Empty title={loading ? "正在读取模型" : "这里还没有模型"} text="发布一个模型，或调整搜索和筛选条件。" />}{matching.length > limit && <Button secondary onClick={() => setLimit(limit + 24)}>继续显示 · 还剩 {matching.length - limit} 个</Button>}</>}
    {view === "releases" && <RecordList rows={rows} loading={loading} render={r => <><div><strong>{r.name}</strong><small>{r.modelId}</small></div><Badge>v{r.version}</Badge><span>{bytes(r.bytes)}</span><time>{when(r.createdAt)}</time></>} />}
    {view === "players" && <RecordList rows={rows} loading={loading} render={r => <><div><strong>{r.identity}</strong><small>{r.uuid || "尚无外观记录"}</small></div><span>{modelName(r.assetId)}{r.version ? ` · v${r.version}` : ""}<small>{r.texture || "默认贴图"}</small></span><time>{when(r.updatedAt)}</time></>} />}
    {view === "npcs" && <RecordList rows={rows} loading={loading} empty="还没有 NPC 上报" render={r => <><div><strong>{r.name} · #{r.citizensId}</strong><small>{r.nodeId} · {r.modelId || "默认外观"}</small></div><Badge tone={r.revision === r.appliedRevision ? "sage" : "amber"}>{r.revision === r.appliedRevision ? "节点已应用" : "等待节点应用"}</Badge><Button secondary disabled={busy} onClick={() => { setError(""); setEditor({ kind: "npc", npc: r }); }}>编辑外观</Button></>} />}
    {view === "activity" && <RecordList rows={rows} loading={loading} render={r => <><div><strong>{{ publish: "发布模型", rollback: "恢复版本", archive: "下架模型", restore: "重新上架", "npc.update": "更新 NPC" }[r.operation] || r.operation}</strong><small>{r.target}</small></div><span>{r.actor}</span><time>{when(r.createdAt)}</time></>} />}
    {view === "connection" && <><div className="ysm-cloud-settings"><article><h3>模型服务</h3><p>{connection?.origin || "尚未配置服务地址"}</p><Badge tone={data ? "sage" : "amber"}>{data ? "已连接" : "等待连接"}</Badge><dl><dt>实例</dt><dd>{data?.instanceId || "—"}</dd><dt>管理凭据</dt><dd>{connection?.tokenConfigured ? "已配置" : "未配置"}</dd></dl><Button secondary disabled={busy || !connection} onClick={() => { setError(""); setEditor({ kind: "connection" }); }}>修改连接</Button></article><article><h3>模型分发</h3><p>{data?.storage?.s3Enabled ? "对象存储分发已启用" : "尚未启用对象存储分发"}</p><dl><dt>下载票据</dt><dd>{data?.storage?.tickets ?? "—"}</dd><dt>复用对象</dt><dd>{data?.storage?.cacheHits ?? "—"}</dd><dt>直传兜底</dt><dd>{data?.storage?.fallbackTransfers ?? "—"} 次</dd></dl><small>计数自服务本次启动开始。</small></article></div><h3>游戏节点</h3><RecordList rows={data?.nodes || []} loading={loading} empty="还没有节点连接" render={r => <><div><strong>{r.name}</strong><small>{r.id} · {r.version}</small></div><Badge tone={Date.now() / 1000 - r.lastSeen < 90 ? "sage" : "amber"}>{Date.now() / 1000 - r.lastSeen < 90 ? "在线" : "离线"}</Badge><span>{r.players} 位玩家 · {r.npcs} 个 NPC</span><time>{when(r.lastSeen)}</time></>} /></>}
    {detail && <Modal title={detail.name} close={close(setDetail)}><p className="muted">{detail.modelId}</p><dl className="ysm-cloud-details"><dt>当前版本</dt><dd>v{detail.version}</dd><dt>大小</dt><dd>{bytes(detail.bytes)}</dd><dt>内容摘要</dt><dd>{detail.sha256}</dd></dl><h3>历史版本</h3><div className="ysm-cloud-records">{detail.versions.map(v => <div key={v.version}><div><strong>v{v.version}</strong><small>{when(v.createdAt)}</small></div><span>{bytes(v.bytes)}</span>{v.version === detail.version ? <Badge>当前版本</Badge> : <Button secondary disabled={busy} onClick={() => setConfirmation({ title: `恢复到 v${v.version}`, text: "旧内容将发布为一个新版本，历史记录会保留。", path: `models/${seg(detail.assetId)}/rollback`, body: { expectedVersion: detail.version, version: v.version } })}>恢复此版本</Button>}</div>)}</div><div className="ysm-cloud-actions"><Button secondary disabled={busy} onClick={() => setConfirmation({ title: detail.status === "published" ? "下架模型" : "重新上架", text: detail.status === "published" ? "模型将从可用目录移除，历史版本会保留。" : "模型将重新出现在可用目录中。", path: `models/${seg(detail.assetId)}/${detail.status === "published" ? "archive" : "restore"}`, body: { expectedVersion: detail.version } })}>{detail.status === "published" ? "下架模型" : "重新上架"}</Button><Button disabled={busy} onClick={() => { setEditor({ kind: "upload", model: detail }); setDetail(null); }}>发布新版本</Button></div></Modal>}
    {editor?.kind === "upload" && <UploadEditor model={editor.model} busy={busy} error={error} close={close(setEditor)} submit={publish} />}
    {editor?.kind === "npc" && <NpcEditor npc={editor.npc} models={models} busy={busy} error={error} close={close(setEditor)} submit={form => run(async () => { const n = editor.npc; await post(`npcs/${seg(n.nodeId)}/${seg(n.uuid)}`, { ...form, expectedRevision: n.revision }); setEditor(null); setNotice("外观已保存，等待游戏节点确认应用。"); })} />}
    {editor?.kind === "connection" && <ConnectionEditor connection={connection} busy={busy} error={error} close={close(setEditor)} submit={form => run(async () => { await post("connection", { ...form, expectedRevision: connection.revision }); setEditor(null); setNotice("服务连接已验证并保存。"); })} />}
    {confirmation && <Modal title={confirmation.title} close={close(setConfirmation)}><p>{confirmation.text}</p>{error && <p className="error-banner" role="alert">{error}</p>}<div className="ysm-cloud-actions"><Button secondary disabled={busy} onClick={close(setConfirmation)}>取消</Button><Button disabled={busy} onClick={() => run(async () => { await post(confirmation.path, confirmation.body); setConfirmation(null); setDetail(null); setNotice("模型状态已更新。"); })}>{busy ? "正在处理…" : "确认"}</Button></div></Modal>}
  </section>;
}

function RecordList({ rows, loading, render, empty = "暂无记录" }) {
  const [limit, setLimit] = useState(50);
  return rows.length ? <><div className="ysm-cloud-records">{rows.slice(0, limit).map((r, i) => <div key={`${r.assetId || r.uuid || r.id || r.identity || "row"}-${r.version || i}`}>{render(r)}</div>)}</div>{rows.length > limit && <Button secondary onClick={() => setLimit(limit + 50)}>显示更多</Button>}</> : <Empty title={loading ? "正在读取…" : empty} text={loading ? "请稍候。" : "服务上报后会显示真实记录。"} />;
}
function EditorActions({ busy, close, title = "保存" }) { return <div className="ysm-cloud-actions"><Button secondary disabled={busy} onClick={close}>取消</Button><Button type="submit" disabled={busy}>{busy ? "正在处理…" : title}</Button></div>; }
function UploadEditor({ model, busy, error, close, submit }) {
  const [form, setForm] = useState({ id: model?.modelId || "", name: model?.name || "", note: "", file: null }), [validation, setValidation] = useState("");
  useUnsavedChanges({ ...form, file: form.file?.name }, busy);
  const change = (key, value) => setForm(old => ({ ...old, [key]: value }));
  return <Modal title={model ? `发布 ${model.name} 的新版本` : "发布模型"} close={close} guardClose><form onSubmit={e => { e.preventDefault(); if (!form.file || !/\.(ysm|zip)$/i.test(form.file.name) || !form.file.size || form.file.size > 128 * 1024 * 1024) { setValidation("请选择不超过 128 MB 的 .ysm 或模型目录 ZIP。"); return; } const id = form.id.trim(); if (!id || id.startsWith("/") || id.includes("\\") || id.split("/").some(p => !p || p === "." || p === "..")) { setValidation("模型 ID 必须是有效的相对名称。"); return; } setValidation(""); submit({ ...form, id, name: form.name.trim() }); }}><p className="muted">相同内容不会重复发布；更新保持模型 ID 不变。</p><Field label="模型文件" hint={form.file ? bytes(form.file.size) : "支持 .ysm、模型目录 ZIP，最大 128 MB"} type="file" accept=".ysm,.zip" required disabled={busy} onChange={e => change("file", e.target.files[0])} /><Field label="显示名称" value={form.name} required maxLength={128} disabled={busy} onChange={e => change("name", e.target.value)} /><Field label="模型 ID" value={form.id} required maxLength={256} disabled={busy || !!model} placeholder="例如：Arknights/amiya" onChange={e => change("id", e.target.value)} /><Field label="发布说明"><textarea rows={3} maxLength={2000} disabled={busy} value={form.note} onChange={e => change("note", e.target.value)} /></Field>{(error || validation) && <p className="error-banner" role="alert">{validation || error}</p>}{busy && <p role="status">正在校验、上传并发布，请保持页面打开…</p>}<EditorActions busy={busy} close={close} title="校验并发布" /></form></Modal>;
}
function NpcEditor({ npc, models, busy, error, close, submit }) {
  const [form, setForm] = useState({ modelId: npc.modelId, textureId: npc.textureId, actionId: npc.actionId, actionLocked: npc.actionLocked }); useUnsavedChanges(form, busy);
  const change = (key, value) => setForm(old => ({ ...old, [key]: value }));
  return <Modal title={`${npc.name} · 外观设置`} close={close} guardClose><form onSubmit={e => { e.preventDefault(); submit(form); }}><Field label="模型"><select value={form.modelId} disabled={busy} onChange={e => change("modelId", e.target.value)}><option value="">默认外观</option>{form.modelId && !models.some(m => m.modelId === form.modelId && m.status === "published") && <option value={form.modelId}>{form.modelId}（当前不可用）</option>}{models.filter(m => m.status === "published").map(m => <option key={m.modelId} value={m.modelId}>{m.name}</option>)}</select></Field><Field label="贴图" hint="留空使用默认贴图" value={form.textureId} maxLength={256} disabled={busy} onChange={e => change("textureId", e.target.value)} /><Field label="动作" hint="留空停止自定义动作" value={form.actionId} maxLength={256} disabled={busy} onChange={e => change("actionId", e.target.value)} /><label><input type="checkbox" checked={form.actionLocked} disabled={busy} onChange={e => change("actionLocked", e.target.checked)} />保持指定动作</label>{error && <p className="error-banner" role="alert">{error}</p>}<EditorActions busy={busy} close={close} title="保存外观" /></form></Modal>;
}
function ConnectionEditor({ connection, busy, error, close, submit }) {
  const [form, setForm] = useState({ origin: connection.origin, token: "" }); useUnsavedChanges(form, busy);
  return <Modal title="模型服务连接" close={close} guardClose><form onSubmit={e => { e.preventDefault(); submit(form); }}><Field label="服务地址" type="url" required value={form.origin} disabled={busy} onChange={e => setForm({ ...form, origin: e.target.value })} /><Field label="管理令牌" type="password" autoComplete="new-password" hint={connection.tokenConfigured ? "留空将保留当前令牌。" : "仅保存到管理网关，页面不会回显。"} value={form.token} disabled={busy} onChange={e => setForm({ ...form, token: e.target.value })} />{error && <p className="error-banner" role="alert">{error}</p>}<EditorActions busy={busy} close={close} title="验证并保存" /></form></Modal>;
}

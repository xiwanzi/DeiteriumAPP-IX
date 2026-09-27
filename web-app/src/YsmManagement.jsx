import React, { useEffect, useRef, useState } from "react";
import { RefreshCw, Plus, Search, Server, Download, FileBox, ArrowRight } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";
import "./ysm.css";

const prefix = "/api/v1/admin/ysm/";
const names = { VERIFY_S3: "验证存储", DRY_RUN: "预览目录差异", PUBLISH: "增量发布", CLUSTER_RELOAD: "集群重载", MODEL_ADD: "新增模型", MODEL_REPLACE: "替换模型", MODEL_DELETE: "删除模型", GC_DRY_RUN: "回收预览", GC: "空间回收" };
const phases = { READY: "就绪", OFFLINE: "离线", SYNCING: "同步中", STARTING: "启动中", DEGRADED: "需要处理", QUEUED: "排队中", RUNNING: "进行中", SUCCEEDED: "成功", FAILED: "失败", IDLE: "尚未开始", NOT_RUN: "尚未运行", PREVIEW: "预览就绪", VERIFYING: "正在验证", COPYING: "迁移与校验中", PREPARED: "校验完成", COMPLETED: "已完成" };
const effects = { VERIFY_S3: "验证存储的读写、公共读取和删除权限，并清理测试对象。", DRY_RUN: "扫描源目录，查看增改删差异。", PUBLISH: "上传有变化的对象，校验后发布新的模型版本。", CLUSTER_RELOAD: "请求在线节点重载模型，并刷新玩家模型目录。", GC_DRY_RUN: "计算超过保留期限、可回收的对象。", GC: "按最近一次回收预览删除过期对象，保留仍被使用的模型和版本。" };
const bytes = (n = 0) => n < 1024 ? `${n} B` : n < 1048576 ? `${(n / 1024).toFixed(1)} KiB` : `${(n / 1048576).toFixed(1)} MiB`;
const when = (v) => v ? new Date(v).toLocaleString("zh-CN", { hour12: false }) : "—";
const status = (value) => <Badge tone={["READY", "SUCCEEDED", "COMPLETED"].includes(value) ? "sage" : ["FAILED", "DEGRADED"].includes(value) ? "amber" : "neutral"}>{phases[value] || value || "—"}</Badge>;

export default function YsmManagement({ client }) {
  const [view, setView] = useState("模型库"), [data, setData] = useState({}), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const [loading, setLoading] = useState(true), [busy, setBusy] = useState(false), [updated, setUpdated] = useState(null);
  const [query, setQuery] = useState(""), [category, setCategory] = useState(""), [page, setPage] = useState(0);
  const [selected, setSelected] = useState(null), [upload, setUpload] = useState(null), [editing, setEditing] = useState(false), [pending, setPending] = useState(null);
  const alive = useRef(true), refreshing = useRef(false), acting = useRef(false);
  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  useUnsavedChanges(null, busy);
  const get = async (path) => (await client.request(prefix + path)).data;
  const post = async (path, body, extra = {}) => (await client.request(prefix + path, { method: "POST", body, signal: AbortSignal.timeout(310000), ...extra })).data;
  const load = async (quiet = false) => {
    if (refreshing.current) return;
    refreshing.current = true;
    if (!quiet) { setLoading(true); setError(""); }
    const keys = quiet ? ["cluster", "tasks", "migration", "gc", "catalog", "diff", "audit"] : ["config", "catalog", "cluster", "tasks", "diff", "audit", "migration", "gc", "dashboard"];
    const next = {}, failures = [];
    try {
      // Stay within the bounded upstream worker pool, including background polling.
      for (let i = 0; i < keys.length; i += 2) {
        const batch = keys.slice(i, i + 2);
        const results = await Promise.allSettled(batch.map(get));
        results.forEach((r, j) => r.status === "fulfilled" ? next[batch[j]] = r.value : failures.push(r.reason.message));
        if (!alive.current) return;
      }
      setData((old) => ({ ...old, ...next }));
      if (failures.length) setError(failures[0] + " 未更新的数据保留上次结果。");
      if (!failures.length) setUpdated(new Date());
    } finally { refreshing.current = false; if (alive.current) setLoading(false); }
  };
  useEffect(() => { load(); const timer = setInterval(() => { if (!document.hidden && !acting.current) load(true); }, 10000); return () => clearInterval(timer); }, [client]);
  const run = async (fn) => {
    if (acting.current) return;
    acting.current = true; setBusy(true); setError(""); setNotice("");
    try { await fn(); } catch (e) { if (alive.current) setError(e.message); }
    finally { acting.current = false; if (alive.current) setBusy(false); }
  };
  const operation = (name, retryTaskId = null) => run(async () => {
    const cluster = await get("cluster");
    const expectedRevision = ["PUBLISH", "DRY_RUN", "GC", "GC_DRY_RUN"].includes(name) ? cluster.modelRevision : cluster.configRevision;
    const preview = await post("operations/preview", { operation: retryTaskId ? null : { operation: name, expectedRevision, expectedConfigRevision: cluster.configRevision }, retryTaskId });
    setPending({ ...preview, path: "operations/confirm", title: retryTaskId ? "重试后台任务" : names[name], lines: [retryTaskId ? `重新执行任务 ${retryTaskId}。` : effects[name], ...(name === "GC" ? [`最近回收预览：${data.gc?.objects || 0} 个对象，${bytes(data.gc?.reclaimableBytes)}。`] : [])] });
  });
  const remove = (model) => run(async () => {
    const cluster = await get("cluster");
    const preview = await post("models/delete/preview", { modelId: model.modelId, expectedRevision: cluster.modelRevision });
    setSelected(null);
    setPending({ ...preview, path: "models/confirm", title: "删除模型", danger: true, lines: [model.modelId, "从新的模型目录移除此模型，并通知节点同步。使用它的玩家和 NPC 会按原规则回退。"] });
  });
  const stage = (form) => run(async () => {
    const contentType = /\.ysm$/i.test(form.file?.name || "") ? "application/vnd.deuteriumysm.model" : /\.zip$/i.test(form.file?.name || "") ? "application/zip" : "";
    if (!contentType || !form.id.trim()) throw new Error("请填写模型 ID，并选择 .ysm 或 .zip 文件。");
    const maximum = Math.min(data.config?.config?.security?.maxUploadBytes || 512 * 1048576, 512 * 1048576);
    if (!form.file.size || form.file.size > maximum) throw new Error(`文件大小应在 1 B 到 ${bytes(maximum)} 之间。`);
    const cluster = await get("cluster");
    const preview = await post("models/stage", form.file, { ysmUpload: { contentType, modelId: form.id.trim(), operation: form.operation, revision: cluster.modelRevision } });
    setPending({ ...preview, path: "models/confirm", title: form.operation === "add" ? "发布新模型" : "替换模型", lines: [preview.modelId, `已校验 ${preview.fileCount} 个文件，上传 ${bytes(preview.uploadedBytes)}，展开 ${bytes(preview.extractedBytes)}。`, "确认后发布新版本，并协调节点同步。"] });
  });
  const previewConfig = (config, credentials) => run(async () => {
    const preview = await post("config/preview", { expectedRevision: config.configRevision, config, credentials });
    setPending({ ...preview, path: "config/confirm", title: "保存 YSM 配置", lines: [`修改字段：${preview.changedFields.join("、")}`, preview.s3VerificationRequired ? "先验证新存储并迁移完整目录，成功后切换；失败时保留原配置。" : "保存后由 YSM 向各节点分发配置。", "监听地址、端口或节点凭据的修改可能需要配套重启和连接配置。"] });
  });
  const confirm = () => run(async () => {
    const current = pending;
    // Consume the UI action even if the HTTP result is unknown. Never auto-resubmit.
    setPending(null);
    let result;
    try { result = await post(current.path, { confirmationToken: current.confirmationToken }); }
    catch (e) { setView("任务与审计"); await load(); throw e; }
    setUpload(null); setEditing(false);
    setNotice(result.taskId ? `已提交任务 ${result.taskId}，请在“任务与审计”查看结果。` : `配置已保存，版本 #${result.configRevision}。`);
    if (result.taskId) setView("任务与审计");
    await load();
  });
  const models = data.catalog?.models || [], categories = [...new Set(models.map((m) => m.directory).filter(Boolean))].sort();
  const filtered = models.filter((m) => (!category || m.directory === category) && `${m.modelId} ${m.directory}`.toLowerCase().includes(query.toLowerCase()));
  const pages = Math.max(1, Math.ceil(filtered.length / 24)), currentPage = Math.min(page, pages - 1);
  const nodes = data.cluster?.nodes || [];
  return <section className="ysm-workspace" aria-busy={busy}>
    <PageHead eyebrow="YSM MODEL LIBRARY" title="模型管理" subtitle="模型、节点和发布进度，在同一处管理。">
      <Button secondary disabled={loading || busy} onClick={() => load()}><RefreshCw size={16} />刷新</Button>
      <Button disabled={busy || !data.catalog} onClick={() => { setError(""); setUpload({ operation: "add", id: "" }); }}><Plus size={16} />新增模型</Button>
    </PageHead>
    {error && <div className="ysm-feedback error" role="alert">{error}</div>}
    {notice && <div className="ysm-feedback" role="status">{notice}</div>}
    <div className="ysm-metrics">
      <div><span>模型总数</span><strong>{data.catalog?.modelCount ?? "—"}</strong><small>{bytes(data.catalog?.totalBytes)} · {data.catalog?.objectCount ?? "—"} 个文件</small></div>
      <div><span>模型版本</span><strong>{data.catalog ? `#${data.catalog.revision}` : "—"}</strong><small>最近同步 {when(data.catalog?.lastSyncTime)}</small></div>
      <div><span>在线节点</span><strong>{data.cluster ? `${nodes.filter((n) => n.online).length} / ${nodes.length}` : "—"}</strong><small>{nodes.filter((n) => n.online && n.phase === "READY" && !n.lastErrorCode).length} 个节点就绪</small></div>
    </div>
    <Tabs values={["模型库", "节点状态", "任务与审计", "存储与配置"]} value={view} onChange={setView} label="YSM 管理栏目" />
    {loading && !data.catalog && <p role="status">正在读取 YSM 管理数据…</p>}
    {!loading && !data.catalog && <Empty title="YSM 管理暂不可用" text="恢复连接后即可读取现有模型库。"><Button secondary onClick={() => load()}>重新连接</Button></Empty>}
    {data.catalog && view === "模型库" && <>
      <div className="ysm-toolbar"><label className="ysm-search"><Search size={18} /><input aria-label="搜索模型" placeholder="搜索模型 ID 或目录" value={query} onChange={(e) => { setQuery(e.target.value); setPage(0); }} /></label>
        <select aria-label="模型目录" value={category} onChange={(e) => { setCategory(e.target.value); setPage(0); }}><option value="">全部目录</option>{categories.map((v) => <option key={v}>{v}</option>)}</select><span>{filtered.length} 个模型</span></div>
      <div className="ysm-model-grid">{filtered.slice(currentPage * 24, currentPage * 24 + 24).map((m) => <button className="ysm-model" key={m.modelId} onClick={() => setSelected(m)}>
        <ModelCover model={m} /><span className="ysm-model-copy"><strong>{m.modelId}</strong><small>{m.directory || "根目录"}</small><span>{bytes(m.size)} · {m.objectCount} 个文件 <ArrowRight size={14} /></span></span>
      </button>)}</div>
      {!filtered.length && <Empty title="没有匹配的模型" text="试试其他关键词，或选择全部目录。" />}
      <div className="ysm-pagination"><Button secondary disabled={!currentPage} onClick={() => setPage(currentPage - 1)}>上一页</Button><span>{currentPage + 1} / {pages}</span><Button secondary disabled={currentPage + 1 >= pages} onClick={() => setPage(currentPage + 1)}>下一页</Button></div>
    </>}
    {view === "节点状态" && <><div className="ysm-toolbar"><p>查看各节点当前版本与同步进度。</p><Button secondary disabled={busy || !data.cluster} onClick={() => operation("CLUSTER_RELOAD")}>集群重载</Button></div>
      <div className="ysm-node-grid">{nodes.map((n) => <article className="ysm-panel" key={n.nodeId}><div className="ysm-row"><Server size={20} /><h3>{n.nodeId}</h3>{status(n.online ? n.phase : "OFFLINE")}</div><p>模型 #{n.modelRevision} · 配置 #{n.configRevision}</p><progress max="1000" value={n.progressPermille || 0} aria-label={`${n.nodeId} 同步进度`} /><small>最近心跳 {when(n.lastHeartbeat)}</small>{n.lastErrorCode && <p className="ysm-error">{n.lastErrorCode}</p>}</article>)}</div>{!nodes.length && <Empty title="尚未发现节点" text="节点连接后会出现在这里。" />}</>}
    {view === "任务与审计" && <>
      <div className="ysm-toolbar"><div className="ysm-actions"><Button secondary disabled={busy || !data.cluster} onClick={() => operation("DRY_RUN")}>预览目录差异</Button><Button disabled={busy || !data.cluster} onClick={() => operation("PUBLISH")}>增量发布</Button></div><a className="button secondary" href={`${prefix}diagnostics.zip`} download><Download size={16} />下载诊断包</a></div>
      <div className="ysm-panel"><h2>后台任务</h2>{(data.tasks || []).slice(0, 50).map((t) => <div className="ysm-task" key={t.taskId}><div><strong>{names[t.operation] || t.operation}</strong><small>{t.summary || t.errorCode || t.taskId}</small><small>{when(t.updatedAt)} · {t.taskId}</small></div>{status(t.status)}{t.status === "FAILED" && <Button secondary disabled={busy} onClick={() => operation(t.operation, t.taskId)}>重试</Button>}</div>)}{!data.tasks?.length && <p className="muted">暂无任务记录</p>}</div>
      <div className="ysm-panel"><h2>目录差异</h2>{data.diff?.available && data.diff.diff ? <><p>新增 {data.diff.diff.addedObjects || 0} · 修改 {data.diff.diff.modifiedObjects || 0} · 删除 {data.diff.diff.deletedObjects || 0}</p><ul className="ysm-paths">{[["新增", "addedPaths"], ["修改", "modifiedPaths"], ["删除", "deletedPaths"]].flatMap(([label, key]) => (data.diff.diff[key] || []).slice(0, 50).map((p) => <li key={`${key}:${p}`}>{label} · {p}</li>))}</ul></> : <p className="muted">执行目录预览后，可在这里查看变化。</p>}</div>
      <div className="ysm-panel"><h2>YSM 操作审计</h2>{(data.audit?.entries || []).slice(0, 60).map((a, i) => <div className="ysm-task" key={i}><div><strong>{a.action}</strong><small>{when(a.time)} · {a.actor} · 版本 #{a.revision}</small></div><span>{a.result}</span></div>)}{!data.audit?.entries?.length && <p className="muted">暂无记录</p>}<p className="ysm-footnote">通过本后台提交的操作者与请求记录，可在“管理审计”查看。</p></div>
    </>}
    {view === "存储与配置" && <>
      <div className="ysm-panel"><div className="ysm-row"><h2>模型存储</h2><Badge>配置 #{data.config?.config?.configRevision ?? "—"}</Badge></div><dl className="ysm-facts"><dt>存储桶</dt><dd>{data.config?.config?.s3?.bucket || "—"}</dd><dt>Endpoint</dt><dd>{data.config?.config?.s3?.endpoint || "—"}</dd><dt>公共读取地址</dt><dd>{data.config?.config?.s3?.publicBaseUrl || "—"}</dd></dl><div className="ysm-actions"><Button disabled={busy || !data.config} onClick={() => setEditing(true)}>编辑配置</Button><Button secondary disabled={busy || !data.config} onClick={() => operation("VERIFY_S3")}>验证存储</Button></div></div>
      <div className="ysm-panel"><h2>存储迁移</h2>{status(data.migration?.phase)}<p>{data.migration?.sourceBucket ? `${data.migration.sourceBucket} → ${data.migration.targetBucket}，已校验 ${data.migration.completedObjects} / ${data.migration.totalObjects} 个对象` : "更换存储时显示迁移与校验进度。"}</p>{data.migration?.errorCode && <p className="ysm-error">{data.migration.errorCode}</p>}</div>
      <div className="ysm-panel"><h2>空间回收</h2>{status(data.gc?.phase)}<p>{data.gc?.objects || 0} 个对象 · 可回收 {bytes(data.gc?.reclaimableBytes)}</p><div className="ysm-actions"><Button secondary disabled={busy || !data.config} onClick={() => operation("GC_DRY_RUN")}>预览可回收内容</Button><Button danger disabled={busy || data.gc?.phase !== "PREVIEW"} onClick={() => operation("GC")}>执行回收</Button></div></div>
    </>}
    <p className="ysm-footnote">{updated ? `状态更新于 ${updated.toLocaleTimeString("zh-CN", { hour12: false })}` : "等待服务数据"}</p>
    {selected && <Modal title="模型详情" close={() => setSelected(null)}><ModelCover model={selected} /><h3 className="ysm-break">{selected.modelId}</h3><dl className="ysm-facts"><dt>目录</dt><dd>{selected.directory || "根目录"}</dd><dt>文件</dt><dd>{selected.objectCount} 个 · {bytes(selected.size)}</dd><dt>校验</dt><dd>{selected.verification}</dd><dt>SHA-256</dt><dd>{selected.sha256}</dd></dl><div className="ysm-actions"><Button disabled={busy} onClick={() => { setUpload({ operation: "replace", id: selected.modelId }); setSelected(null); }}>替换模型</Button><Button danger disabled={busy} onClick={() => remove(selected)}>删除模型</Button></div></Modal>}
    {upload && <UploadEditor initial={upload} busy={busy} error={error} close={() => !busy && setUpload(null)} submit={stage} />}
    {editing && data.config && <ConfigEditor initial={data.config.config} busy={busy} error={error} close={() => !busy && setEditing(false)} submit={previewConfig} />}
    {pending && <Confirmation pending={pending} busy={busy} close={() => !busy && setPending(null)} confirm={confirm} />}
  </section>;
}

function ModelCover({ model }) {
  const [failed, setFailed] = useState(false);
  const safe = typeof model.coverUrl === "string" && /^https:\/\//i.test(model.coverUrl);
  return <span className="ysm-cover">{safe && !failed ? <img src={model.coverUrl} alt="" loading="lazy" referrerPolicy="no-referrer" onError={() => setFailed(true)} /> : <FileBox size={40} strokeWidth={1.2} />}</span>;
}

function Confirmation({ pending, busy, close, confirm }) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => { const id = setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(id); }, []);
  const expired = now >= pending.expiresAtEpochSecond * 1000;
  return <Modal title={pending.title} close={close}><ul className="ysm-confirm-lines">{pending.lines.map((line, i) => <li key={i}>{line}</li>)}</ul><p>{expired ? "确认已过期，请取消并重新预览。" : `确认有效至 ${when(pending.expiresAtEpochSecond * 1000)}`}</p><div className="ysm-actions"><Button secondary disabled={busy} onClick={close}>取消</Button><Button danger={pending.danger} disabled={busy || expired} onClick={confirm}>确认执行</Button></div></Modal>;
}

function UploadEditor({ initial, busy, error, close, submit }) {
  const [form, setForm] = useState({ ...initial, file: null });
  useUnsavedChanges({ id: form.id, file: form.file?.name }, busy);
  return <Modal title={form.operation === "add" ? "新增模型" : "替换模型"} close={close} guardClose>
    {error && <p role="alert" className="ysm-error">{error}</p>}
    <Field label="模型 ID" hint="使用原模型目录中的完整 ID，支持中文。" value={form.id} disabled={busy || form.operation === "replace"} onChange={(e) => setForm({ ...form, id: e.target.value })} />
    <Field label="模型文件" hint="支持 .ysm 或 ZIP 模型包。上传校验后还需确认发布。"><input type="file" accept=".ysm,.zip" disabled={busy} onChange={(e) => { const file = e.target.files[0]; setForm({ ...form, file, id: form.id || file?.name.replace(/\.zip$/i, "") || "" }); }} /></Field>
    <div className="ysm-actions"><Button secondary disabled={busy} onClick={close}>取消</Button><Button disabled={busy || !form.file || !form.id.trim()} onClick={() => submit(form)}>{busy ? "正在上传与校验…" : "上传并预览"}</Button></div>
  </Modal>;
}

const configFields = [
  ["s3", "endpoint", "Endpoint", "url"], ["s3", "region", "区域", "text"], ["s3", "bucket", "存储桶", "text"], ["s3", "publicBaseUrl", "公共读取地址", "url"],
  ["distribution", "downloadConcurrency", "并发下载数", "number"], ["distribution", "requestTimeoutMs", "单请求超时（毫秒）", "number"], ["distribution", "gcGraceSeconds", "回收宽限期（秒）", "number"], ["distribution", "gcKeepRevisions", "保留版本数", "number"],
  ["fallback", "bytesPerSecond", "兜底总速率（字节/秒）", "number"], ["fallback", "maxConcurrentDownloads", "兜底并发数", "number"], ["fallback", "timeoutSeconds", "兜底超时（秒）", "number"], ["fallback", "maxSessionBytes", "每会话上限（字节）", "number"],
];
function ConfigEditor({ initial, busy, error, close, submit }) {
  const [draft, setDraft] = useState(JSON.stringify(initial, null, 2));
  const [credentials, setCredentials] = useState({ newAccessKey: "", newSecretKey: "", newCoordinatorSecret: "", clearAccessKey: false, clearSecretKey: false, clearCoordinatorSecret: false });
  const [localError, setLocalError] = useState("");
  useUnsavedChanges({ draft, credentials }, busy);
  let parsed;
  try { const value = JSON.parse(draft); if (value?.s3 && value?.distribution && value?.fallback) parsed = value; } catch {}
  const change = (group, key, value) => { const next = structuredClone(parsed); next[group][key] = value; setDraft(JSON.stringify(next, null, 2)); };
  return <Modal title="YSM 存储与配置" close={close} wide guardClose>
    {(error || localError) && <p role="alert" className="ysm-error">{localError || error}</p>}
    <p className="muted">配置版本 #{initial.configRevision}。修改先预览，再确认保存；现有密钥留空保留。</p>
    <fieldset className="ysm-config-fields" disabled={busy || !parsed}>
      {configFields.map(([group, key, label, type]) => <Field key={`${group}.${key}`} label={label} type={type} value={parsed?.[group]?.[key] ?? ""} onChange={(e) => change(group, key, type === "number" ? Number(e.target.value) : e.target.value)} />)}
      <label><input type="checkbox" checked={parsed?.s3?.pathStyleAccess || false} onChange={(e) => change("s3", "pathStyleAccess", e.target.checked)} /> 使用 path-style 访问</label>
      <label><input type="checkbox" checked={parsed?.distribution?.gcEnabled || false} onChange={(e) => change("distribution", "gcEnabled", e.target.checked)} /> 自动回收过期对象</label>
    </fieldset>
    <details className="ysm-panel"><summary>更新存储和节点凭据</summary><p className="muted">密钥仅用于这次配置提交，不会在页面或诊断包中回显。</p>{[["newAccessKey", "clearAccessKey", "Access Key"], ["newSecretKey", "clearSecretKey", "Secret Key"], ["newCoordinatorSecret", "clearCoordinatorSecret", "节点控制凭据"]].map(([key, clear, label]) => <div key={key}><Field label={label} type="password" autoComplete="new-password" disabled={busy || credentials[clear]} value={credentials[key]} onChange={(e) => setCredentials({ ...credentials, [key]: e.target.value })} /><label><input type="checkbox" disabled={busy} checked={credentials[clear]} onChange={(e) => setCredentials({ ...credentials, [clear]: e.target.checked, [key]: "" })} /> 清除现有{label}</label></div>)}</details>
    <details className="ysm-panel"><summary>高级配置</summary><p className="muted">保留原配置结构。修改监听参数后需同步更新管理连接。</p><Field label="完整配置 JSON"><textarea className="ysm-json" rows="16" disabled={busy} value={draft} spellCheck="false" onChange={(e) => setDraft(e.target.value)} /></Field></details>
    <div className="ysm-actions"><Button secondary disabled={busy} onClick={close}>取消</Button><Button disabled={busy} onClick={() => { if (!parsed) { setLocalError("配置 JSON 格式不正确。"); return; } setLocalError(""); submit(parsed, credentials); }}>{busy ? "正在处理…" : "预览配置变更"}</Button></div>
  </Modal>;
}

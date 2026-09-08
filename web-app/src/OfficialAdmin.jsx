import React, { useEffect, useRef, useState } from "react";
import { Plus, Upload, ArrowUp, ArrowDown, Trash2 } from "lucide-react";
import CoreAdmin from "./CoreAdmin.jsx";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { ContentBlocks } from "./RemoteCollection.jsx";
import { id } from "./format.js";
import { uploadAsset } from "./assets.js";
import InterventionManagement from "./Interventions.jsx";
import AuditManagement from "./AuditManagement.jsx";

export default function OfficialAdmin({ client, user, navigate }) {
  const permissions = user.permissions || [], all = permissions.includes("platform.admin"),
    tabs = [...(all || permissions.includes("core.read") ? ["Core 管理"] : []), ...(all || permissions.includes("announcements.manage") ? ["公告管理"] : []), ...(all || permissions.includes("intervention.manage") ? ["平台介入"] : []), ...(all || permissions.includes("audit.read") ? ["管理审计"] : []), "商店管理"],
    [tab, setTab] = useState(tabs[0]);
  return <><Tabs values={tabs} value={tab} onChange={setTab} />
    {tab === "Core 管理" ? <CoreAdmin client={client} /> : tab === "公告管理" ? <AnnouncementManagement client={client} /> : tab === "平台介入" ? <InterventionManagement client={client} user={user} /> : tab === "管理审计" ? <AuditManagement client={client} /> : <><PageHead eyebrow="MERCHANT WORKSPACE" title="商店管理" subtitle="管理你的商店、商品与玩家发布。" /><Button onClick={() => navigate("/merchant")}>进入商店管理</Button></>}
  </>;
}

function AnnouncementManagement({ client }) {
  const [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false), [editor, setEditor] = useState(null), [withdraw, setWithdraw] = useState(null), [reason, setReason] = useState("");
  const actionKeys = useRef({});
  const load = async (more = false) => {
    setBusy(true); setError("");
    try { const r = await client.request(`/api/v1/admin/announcements?limit=30${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); setItems((old) => more ? [...old, ...r.data] : r.data); setCursor(r.page?.nextCursor); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  useEffect(() => { load(); }, []);
  const action = async (item, actionName, explanation = "") => {
    setBusy(true); setError(""); const scope = `${item.announcementId}:${item.version}:${actionName}:${explanation}`;
    actionKeys.current[scope] ||= id();
    try {
      await client.request(`/api/v1/admin/announcements/${encodeURIComponent(item.announcementId)}/${actionName}`, { method: "POST", body: { clientRequestId: actionKeys.current[scope], expectedVersion: item.version, ...(actionName === "unpublish" ? { reason: explanation } : {}) } });
      setWithdraw(null); await load();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <>
    <PageHead eyebrow="OFFICIAL ANNOUNCEMENTS" title="公告管理" subtitle="保存草稿，确认后再向社区发布。"><Button onClick={() => setEditor({})}><Plus size={16} />新建公告</Button></PageHead>
    {error && <p className="notice-box" role="alert">{error}</p>}
    <div className="announcement-admin-list">{items.map((item) => <article key={item.announcementId}>
      {item.cover?.url && <div className="admin-ann-cover"><img src={item.cover.url} alt={item.title} /></div>}
      <div className="admin-ann-copy"><Badge tone={item.status === "PUBLISHED" ? "sage" : "neutral"}>{{ DRAFT: "草稿", PUBLISHED: "已发布", WITHDRAWN: "已撤下" }[item.status] || "公告"}</Badge>{item.pinned && <Badge>置顶</Badge>}{item.hasUnpublishedChanges && item.status === "PUBLISHED" && <Badge tone="amber">有未发布修改</Badge>}<h3>{item.title}</h3><p>{item.summary}</p></div>
      <div className="button-row"><Button secondary disabled={busy} onClick={() => setEditor(item)}>编辑</Button><Button disabled={busy} onClick={() => action(item, "publish")}>{item.status === "PUBLISHED" ? "发布更新" : "发布"}</Button>{item.status === "PUBLISHED" && <Button secondary disabled={busy} onClick={() => { setWithdraw(item); setReason(""); }}>撤下</Button>}</div>
    </article>)}</div>
    {!busy && !items.length && <Empty title="还没有公告" text="创建并发布后，玩家就能在 App 和网页看到。" />}
    {cursor && <Button secondary disabled={busy} onClick={() => load(true)}>加载更多</Button>}
    {editor && <Modal title={editor.announcementId ? "编辑公告" : "新建公告"} close={() => setEditor(null)} wide><AnnouncementForm client={client} initial={editor} onSaved={async () => { setEditor(null); await load(); }} /></Modal>}
    {withdraw && <Modal title="撤下公告" close={() => setWithdraw(null)}><form onSubmit={(e) => { e.preventDefault(); action(withdraw, "unpublish", reason); }}><p>撤下“{withdraw.title}”后，玩家将无法查看此公告。</p><Field label="撤下原因"><textarea required maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} /></Field>{error && <p role="alert" className="auth-error">{error}</p>}<Button type="submit" disabled={busy}>确认撤下</Button></form></Modal>}
  </>;
}

function AnnouncementForm({ client, initial, onSaved }) {
  const [title, setTitle] = useState(initial.title || ""), [summary, setSummary] = useState(initial.summary || ""),
    [blocks, setBlocks] = useState(initial.contentBlocks || [{ blockId: id(), type: "PARAGRAPH", text: "" }]),
    [cover, setCover] = useState(initial.cover || null), [media, setMedia] = useState(initial.media || []),
    [pinned, setPinned] = useState(initial.pinned || false), [priority, setPriority] = useState(initial.priority || "NORMAL"),
    [preview, setPreview] = useState(false), [busy, setBusy] = useState(false), [progress, setProgress] = useState(null), [error, setError] = useState("");
  const request = useRef(null), coverInput = useRef(null), imageInput = useRef(null);
  const update = (index, patch) => setBlocks((old) => old.map((block, n) => n === index ? { ...block, ...patch } : block));
  const move = (index, delta) => setBlocks((old) => { const next = [...old], target = index + delta; if (target < 0 || target >= next.length) return old; [next[index], next[target]] = [next[target], next[index]]; return next; });
  const upload = async (file, asCover) => {
    if (!file || busy) return; setBusy(true); setError("");
    try { const asset = await uploadAsset(client, file, "ANNOUNCEMENT_MEDIA", "ANNOUNCEMENT", initial.announcementId || "", (value, label) => setProgress({ value, label })); setMedia((old) => [...old, asset]); if (asCover) setCover(asset); else setBlocks((old) => [...old, { blockId: id(), type: "IMAGE", assetId: asset.assetId, altText: file.name }]); }
    catch (e) { setError(e.message); } finally { setBusy(false); setProgress(null); if (coverInput.current) coverInput.current.value = ""; if (imageInput.current) imageInput.current.value = ""; }
  };
  const save = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    const input = { title, summary, contentBlocks: blocks, coverAssetId: cover?.assetId || null, pinned, priority, ...(initial.announcementId ? { expectedVersion: initial.version } : {}) }, fingerprint = JSON.stringify(input);
    if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, body: { clientRequestId: id(), ...input } };
    try { await client.request(initial.announcementId ? `/api/v1/admin/announcements/${encodeURIComponent(initial.announcementId)}` : "/api/v1/admin/announcements", { method: initial.announcementId ? "PUT" : "POST", body: request.current.body }); await onSaved(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={save}><fieldset disabled={busy} style={{ padding: 0, border: 0, margin: 0, minWidth: 0 }}>
    <Field label="标题" required maxLength={100} value={title} onChange={(e) => setTitle(e.target.value)} /><Field label="摘要"><textarea required maxLength={300} rows={2} value={summary} onChange={(e) => setSummary(e.target.value)} /></Field>
    <div className="button-row"><Button secondary onClick={() => coverInput.current?.click()}><Upload size={16} />{cover ? "更换封面" : "上传封面"}</Button>{cover && <Button secondary onClick={() => setCover(null)}>移除封面</Button>}<Button secondary onClick={() => setPreview(!preview)}>{preview ? "继续编辑" : "预览正文"}</Button></div>
    <input hidden ref={coverInput} type="file" accept="image/png,image/jpeg,image/webp" onChange={(e) => upload(e.target.files?.[0], true)} /><input hidden ref={imageInput} type="file" accept="image/png,image/jpeg,image/webp" onChange={(e) => upload(e.target.files?.[0], false)} />
    {cover?.url && <img src={cover.url} alt="公告封面" style={{ width: "100%", maxHeight: 250, objectFit: "cover", borderRadius: 12, marginBottom: 20 }} />}
    {preview ? <div className="panel" style={{ padding: 20 }}><h2>{title}</h2><p>{summary}</p><ContentBlocks blocks={blocks} media={media} /></div> : blocks.map((block, index) => <div key={block.blockId} className="panel" style={{ padding: 18, margin: "12px 0" }}>
      <div className="button-row"><Badge tone="neutral">{{ HEADING: "小标题", PARAGRAPH: "段落", IMAGE: "图片", KEY_VALUE_LIST: "信息列表" }[block.type]}</Badge><Button secondary aria-label={`上移第 ${index + 1} 段`} onClick={() => move(index, -1)}><ArrowUp size={14} /></Button><Button secondary aria-label={`下移第 ${index + 1} 段`} onClick={() => move(index, 1)}><ArrowDown size={14} /></Button><Button secondary disabled={blocks.length < 2} aria-label={`删除第 ${index + 1} 段`} onClick={() => setBlocks((old) => old.filter((_, n) => n !== index))}><Trash2 size={14} /></Button></div>
      {block.type === "HEADING" && <Field label="小标题" required maxLength={100} value={block.heading || ""} onChange={(e) => update(index, { heading: e.target.value })} />}
      {block.type === "PARAGRAPH" && <Field label="段落内容"><textarea required rows={4} maxLength={3000} value={block.text || ""} onChange={(e) => update(index, { text: e.target.value })} /></Field>}
      {block.type === "IMAGE" && <><ContentBlocks blocks={[block]} media={media} /><Field label="图片说明" maxLength={200} value={block.altText || ""} onChange={(e) => update(index, { altText: e.target.value })} /></>}
      {block.type === "KEY_VALUE_LIST" && block.rows?.map((row, n) => <div key={n}><Field label="名称" required maxLength={80} value={row.label} onChange={(e) => update(index, { rows: block.rows.map((value, i) => i === n ? { ...value, label: e.target.value } : value) })} /><Field label="内容" required maxLength={500} value={row.value} onChange={(e) => update(index, { rows: block.rows.map((value, i) => i === n ? { ...value, value: e.target.value } : value) })} /></div>)}
    </div>)}
    {!preview && <div className="button-row"><Button secondary disabled={blocks.length >= 100} onClick={() => setBlocks((old) => [...old, { blockId: id(), type: "PARAGRAPH", text: "" }])}>添加段落</Button><Button secondary disabled={blocks.length >= 100} onClick={() => setBlocks((old) => [...old, { blockId: id(), type: "HEADING", heading: "" }])}>添加小标题</Button><Button secondary disabled={blocks.length >= 100} onClick={() => imageInput.current?.click()}>添加图片</Button></div>}
    <Field label="重要程度"><select value={priority} onChange={(e) => setPriority(e.target.value)}><option value="NORMAL">普通</option><option value="IMPORTANT">重要</option></select></Field><label className="checkbox-row"><input type="checkbox" checked={pinned} onChange={(e) => setPinned(e.target.checked)} />置顶公告</label>
    </fieldset>{progress && <p role="status">{progress.label} · {progress.value}%</p>}{error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button type="submit" disabled={busy}>{busy ? "正在处理…" : "保存草稿"}</Button></div>
  </form>;
}

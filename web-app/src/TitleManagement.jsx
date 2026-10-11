import React, { useEffect, useRef, useState } from "react";
import { Button, Field, Modal, PageHead, Empty, Badge } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";

export function TitleText({ text }) {
  let color = "#A9B3B8";
  return <>{String(text || "").split(/(#[0-9a-fA-F]{6})/g).map((part, i) => {
    if (/^#[0-9a-fA-F]{6}$/.test(part)) { color = part; return null; }
    return <span key={i} style={{ color }}>{part}</span>;
  })}</>;
}
const states = { DRAFT: "草稿", PUBLISHED: "已发布", DISABLED: "已停用" };
const blank = { internalId: "", displayName: "", description: "", public: true, weight: 0, status: "DRAFT", version: 0 };
export default function TitleManagement({ client }) {
  const [items, setItems] = useState([]), [loading, setLoading] = useState(true), [error, setError] = useState("");
  const [edit, setEdit] = useState(null), [busy, setBusy] = useState(false), [remove, setRemove] = useState(null), [query, setQuery] = useState("");
  const alive = useRef(true), acting = useRef(false), retry = useRef(null);
  useUnsavedChanges(!!edit, busy);
  const load = async () => { const r = await client.request("/api/v1/admin/titles"); if (alive.current) setItems(r.data.items); };
  useEffect(() => { alive.current = true; load().catch(e => { if (alive.current) setError(e.message); }).finally(() => { if (alive.current) setLoading(false); }); return () => { alive.current = false; }; }, [client]);
  const save = async (value) => {
    if (acting.current) return;
    acting.current = true; setBusy(true); setError("");
    const data = { internalId: value.internalId, displayName: value.displayName, description: value.description || "", public: !!value.public, weight: Number(value.weight), status: value.status, expectedVersion: value.version };
    const key = JSON.stringify([value.id, data]);
    if (retry.current?.key !== key) retry.current = { key, id: crypto.randomUUID() };
    try {
      await client.request(`/api/v1/admin/titles${value.id ? `/${value.id}` : ""}`, { method: "POST", body: { ...data, clientRequestId: retry.current.id } });
      retry.current = null;
      if (alive.current) { setEdit(null); setRemove(null); }
      await load();
    } catch (e) { if (alive.current) setError(e.message); }
    finally { acting.current = false; if (alive.current) setBusy(false); }
  };
  const shown = items.filter(t => `${t.internalId} ${t.displayName} ${t.id}`.toLowerCase().includes(query.toLowerCase()));
  return <>
    <PageHead title="头衔管理" subtitle="修改显示名会同步到正在佩戴该头衔的玩家。权重越高越靠前，同权重后发布的优先。">
      <Button secondary disabled={busy} onClick={() => load().then(() => setError("")).catch(e => setError(e.message))}>刷新</Button>
      <Button onClick={() => { setEdit({ ...blank }); setError(""); }}>创建头衔</Button>
    </PageHead>
    {error && <p role="alert" className="form-error">{error}</p>}
    <Field label="搜索头衔"><input value={query} onChange={e => setQuery(e.target.value)} placeholder="显示名、内部 ID 或头衔 ID" /></Field>
    {loading ? <p>正在加载…</p> : !shown.length ? <Empty title="没有匹配的头衔" /> : <div className="table-wrap"><table><thead><tr><th>显示名</th><th>内部 ID / 权限</th><th>权重</th><th>状态</th><th>可见性</th><th>操作</th></tr></thead><tbody>
      {shown.map(t => <tr key={t.id}><td><span style={{ display: "inline-block", background: "#17181c", padding: "8px 12px", borderRadius: 8 }}><TitleText text={t.displayName} /></span></td><td>{t.internalId}<br /><small>{t.id === "title_default" ? "所有玩家默认可用" : `deuterium.title.${t.internalId}`}</small></td><td>{t.weight}</td><td><Badge>{states[t.status]}</Badge></td><td>{t.public ? "公开" : "未公开"}</td><td><Button secondary onClick={() => { setEdit({ ...t }); setError(""); }}>管理</Button></td></tr>)}
    </tbody></table></div>}
    {edit && <Modal title={edit.id ? "管理头衔" : "创建头衔"} close={() => { if (!busy) setEdit(null); }}>
      <form onSubmit={e => { e.preventDefault(); save(edit); }}>
        {error && <p role="alert" className="form-error">{error}</p>}
        {edit.id && <p>头衔 ID：{edit.id}</p>}
        <Field label="内部 ID"><input required disabled={!!edit.id || busy} pattern="[a-z0-9][a-z0-9_-]{0,47}" maxLength={48} value={edit.internalId} onChange={e => setEdit({ ...edit, internalId: e.target.value })} /></Field>
        <p className="muted">{edit.id === "title_default" ? "默认头衔无需权限。" : `权限：deuterium.title.${edit.internalId || "xxx"} · 创建后内部 ID 不可修改。`}</p>
        <Field label="显示名"><input required disabled={busy} maxLength={512} value={edit.displayName} onChange={e => setEdit({ ...edit, displayName: e.target.value })} placeholder="#d9afd9参#c3bddc考#adcbde这#97d9e1样" /></Field>
        <p>支持 #RRGGBB 逐字配色，最多 32 个可见字符。</p>
        <div aria-label="头衔效果预览" style={{ background: "#17181c", padding: 24, marginBottom: 20, borderRadius: 12, fontSize: 22 }}><TitleText text={edit.displayName || "溯行者"} /></div>
        <Field label="介绍及获取方式"><textarea rows={4} maxLength={1000} disabled={busy} value={edit.description || ""} onChange={e => setEdit({ ...edit, description: e.target.value })} placeholder="介绍这个头衔，并说明如何获得" /></Field>
        <label className="checkbox-row"><input type="checkbox" disabled={busy} checked={!!edit.public} onChange={e => setEdit({ ...edit, public: e.target.checked })} />公开展示</label><p className="muted">不公开的头衔不会出现在全部列表，有权限的玩家仍可在已拥有中选择。</p>
        <Field label="权重"><input type="number" required min={-1000000} max={1000000} step={1} disabled={busy} value={edit.weight} onChange={e => setEdit({ ...edit, weight: e.target.value })} /></Field>
        <Field label="状态"><select disabled={busy || edit.id === "title_default"} value={edit.status} onChange={e => setEdit({ ...edit, status: e.target.value })}>{Object.entries(states).map(([v, name]) => <option key={v} value={v}>{name}</option>)}</select></Field>
        <div className="button-row">{edit.id && edit.id !== "title_default" && <Button danger disabled={busy} onClick={() => setRemove(edit)}>删除</Button>}<Button secondary disabled={busy} onClick={() => setEdit(null)}>取消</Button><Button type="submit" disabled={busy}>{busy ? "保存中…" : "保存"}</Button></div>
      </form>
    </Modal>}
    {remove && <Modal title="删除头衔" close={() => { if (!busy) setRemove(null); }}><p>正在佩戴的玩家将回退到此前仍可用的头衔。内部 ID「{remove.internalId}」删除后不可复用。</p><div className="button-row"><Button secondary disabled={busy} onClick={() => setRemove(null)}>取消</Button><Button danger disabled={busy} onClick={() => save({ ...remove, status: "DELETED" })}>确认删除</Button></div></Modal>}
  </>;
}

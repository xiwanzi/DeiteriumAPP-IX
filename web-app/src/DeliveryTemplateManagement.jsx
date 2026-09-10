import React, { useEffect, useRef, useState } from "react";
import { Plus, Search, Trash2, Package } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, Toggle } from "./components.jsx";
import { id } from "./format.js";
import SearchPicker from "./SearchPicker.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";

export default function DeliveryTemplateManagement({ client, storeId, onChanged }) {
  const [templates, setTemplates] = useState([]), [cursor, setCursor] = useState(null), [query, setQuery] = useState(""),
    [error, setError] = useState(""), [busy, setBusy] = useState(false), [editor, setEditor] = useState(null), [disabling, setDisabling] = useState(null);
  const actionKeys = useRef({}), generation = useRef(0), base = `/api/v1/merchant/stores/${encodeURIComponent(storeId)}/delivery-templates`;
  const load = async (more = false) => {
    const current = ++generation.current; setBusy(true); setError("");
    try {
      const r = await client.request(`${base}?limit=18&q=${encodeURIComponent(query)}${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`);
      if (current !== generation.current) return;
      setTemplates((old) => more ? [...old, ...r.data] : r.data); setCursor(r.page?.nextCursor);
    } catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { const timer = setTimeout(load, 220); return () => { clearTimeout(timer); generation.current++; }; }, [storeId, query]);
  const edit = async (template) => {
    setBusy(true); setError("");
    try { const r = await client.request(base + "/" + encodeURIComponent(template.templateRef)); setEditor(r.data); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const disable = async () => {
    setBusy(true); setError(""); const scope = `${disabling.templateRef}:${disabling.version}`; actionKeys.current[scope] ||= id();
    try { await client.request(`${base}/${encodeURIComponent(disabling.templateRef)}/disable`, { method: "POST", body: { clientRequestId: actionKeys.current[scope], expectedVersion: disabling.version } }); setDisabling(null); await load(); await onChanged?.(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <>
    <p className="description">把多种物品组合成一个交付模板，每种物品可单独设置数量。</p>
    <div className="template-toolbar"><label className="picker-search"><Search size={18} /><input aria-label="搜索交付模板" placeholder="搜索交付模板" value={query} onChange={(e) => setQuery(e.target.value)} /></label><Button onClick={() => setEditor({})}><Plus size={16} />新建模板</Button></div>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="foundation-grid">{templates.map((template) => <article key={template.templateRef} className="foundation-card"><Badge tone={template.active ? "sage" : "neutral"}>{template.active ? "已启用" : "已停用"}</Badge><h3>{template.name}</h3><p>{template.summary}</p><p className="muted">{template.attachmentCount ?? 0} 种物品 · {template.inventoryDomain}</p><Button secondary onClick={() => edit(template)} disabled={busy}>编辑模板</Button></article>)}</div>
    {!busy && !templates.length && <Empty title={query ? "未找到交付模板" : "组合你的第一份交付"} text={query ? "换个名称再试试。" : "从物品库选择一个或多个物品，保存后即可关联商品。"} />}
    {cursor && <Button secondary disabled={busy} onClick={() => load(true)}>加载更多模板</Button>}
    {editor && <Modal title={editor.templateRef ? "编辑交付模板" : "创建交付模板"} close={() => setEditor(null)} wide guardClose dismissOnBackdrop={false}><TemplateForm client={client} base={base} initial={editor} onSaved={async () => { setEditor(null); setQuery(""); if(!query)await load(); await onChanged?.(); }} onDisable={editor.templateRef && editor.active ? () => { setDisabling(editor); setEditor(null); } : null} /></Modal>}
    {disabling && <Modal title="停用交付模板" close={() => setDisabling(null)}><p>停用“{disabling.name}”后，引用它的商品无法接受新订单。已经支付的订单继续按购买时的内容交付。</p>{error && <p className="auth-error" role="alert">{error}</p>}<Button disabled={busy} onClick={disable}>确认停用</Button></Modal>}
  </>;
}

function TemplateForm({ client, base, initial, onSaved, onDisable }) {
  const [name, setName] = useState(initial.name || ""), [summary, setSummary] = useState(initial.summary || ""), [domain, setDomain] = useState(initial.inventoryDomain || ""),
    [servers, setServers] = useState(initial.allowedServerIds || []), [attachments, setAttachments] = useState(initial.attachments || []), [active, setActive] = useState(initial.active ?? true),
    [metadata, setMetadata] = useState({}), [nodes, setNodes] = useState([]), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const mutation = useRef(null);
  const guard = useUnsavedChanges({ name, summary, domain, servers, attachments, active }, busy);
  const key = (item) => `${item.itemRef}:${item.revision}`;
  useEffect(() => {
    let live = true;
    client.nodes().then((r) => { if (live) setNodes(r.data.nodes); }).catch((e) => { if (live) setError(e.message); });
    Promise.all((initial.attachments || []).map(async (a) => {
      const r = await client.request(`/api/v1/admin/core/items/${encodeURIComponent(a.itemRef)}/versions/${a.revision}`); return r.data;
    })).then((items) => { if (live) setMetadata(Object.fromEntries(items.map((v) => [key(v), v]))); }).catch((e) => { if (live) setError(e.message); });
    return () => { live = false; };
  }, []);
  const domains = [...new Set(nodes.filter((node) => node.claimEnabled).map((node) => node.inventoryDomain))];
  const allowed = nodes.filter((node) => node.claimEnabled && node.inventoryDomain === domain && attachments.every((a) => !metadata[key(a)] || metadata[key(a)].compatibleServerIds?.includes(node.serverId)));
  const choose = (selected) => {
    const refs = new Set();
    if (selected.some((v) => refs.has(v.itemRef) || !refs.add(v.itemRef))) { setError("同一物品请选择一个版本。若要更换版本，请先移除原版本。"); return; }
    const nextDomain = domain || selected[0]?.inventoryDomain || "";
    if (selected.some((v) => v.inventoryDomain && v.inventoryDomain !== nextDomain)) { setError("同一个模板需要选择属于相同背包同步组的物品。"); return; }
    setDomain(nextDomain); setError("");
    setMetadata((old) => ({ ...old, ...Object.fromEntries(selected.map((v) => [key(v), v])) }));
    setAttachments(selected.map((v) => ({ itemRef: v.itemRef, revision: v.revision, quantity: attachments.find((a) => key(a) === key(v))?.quantity || 1, payloadSha256: v.payloadSha256 })));
  };
  const save = async (e) => {
    e.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      if (!servers.length) throw new Error("请至少选择一个允许领取的服务器。");
      if (servers.some((server) => !allowed.some((node) => node.serverId === server))) throw new Error("已选服务器不适用于当前附件，请重新选择。");
      const input = { name, summary, inventoryDomain: domain, allowedServerIds: servers, attachments, active, ...(initial.templateRef ? { expectedVersion: initial.version } : {}) }, fingerprint = JSON.stringify(input);
      if (mutation.current?.fingerprint !== fingerprint) mutation.current = { fingerprint, body: { clientRequestId: id(), ...input } };
      await client.request(base + (initial.templateRef ? "/" + encodeURIComponent(initial.templateRef) : ""), { method: initial.templateRef ? "PUT" : "POST", body: mutation.current.body });
      guard.markSaved({ name, summary, domain, servers, attachments, active }); await onSaved();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={save} className="template-form">
    <fieldset disabled={busy} style={{ border: 0, margin: 0, padding: 0, minWidth: 0 }}>
      <div className="form-grid"><Field label="模板名称" required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} /><Field label="背包同步组"><select required value={domain} onChange={(e) => { setDomain(e.target.value); setServers([]); }}><option value="">选择背包同步组</option>{domains.map((value) => <option key={value} value={value}>{value}</option>)}</select></Field></div>
      <Field label="交付摘要"><textarea required maxLength={1000} rows={3} value={summary} onChange={(e) => setSummary(e.target.value)} /></Field>
      <div className="field"><label>交付物品</label><SearchPicker label="从物品库添加" placeholder="搜索物品名称或物品编号" multiple max={32} showChips={false} values={attachments.map((a) => ({ ...a, ...metadata[key(a)] }))} getKey={key} getTitle={(v) => v.displayName || v.itemRef} getDescription={(v) => `${v.itemRef} · 版本 ${v.revision}${v.archived ? " · 已归档" : ""}`}
        unavailable={(v) => v.archived || (domain && v.inventoryDomain !== domain)} onChange={choose}
        load={async (query, cursor) => { const r = await client.items(cursor, query, domain); return { items: r.data.items, nextCursor: r.data.next }; }} /></div>
      <div className="template-attachments">{attachments.map((a, index) => { const item = metadata[key(a)]; return <div className="template-attachment" key={key(a)}><div><strong>{item?.displayName || a.itemRef}</strong><small>{a.itemRef} · 版本 {a.revision}</small></div><Field label="数量" type="number" min={1} max={Math.min(99999, item?.maxQuantity || 99999)} required value={a.quantity} onChange={(e) => setAttachments((old) => old.map((v, i) => i === index ? { ...v, quantity: Number(e.target.value) } : v))} /><button type="button" className="icon-button" aria-label={`移除 ${item?.displayName || a.itemRef}`} onClick={() => setAttachments((old) => old.filter((_, i) => i !== index))}><Trash2 size={17} /></button></div>; })}</div>
      {!attachments.length && <p className="notice-box"><Package size={16} /> 可以选择多种物品。仅发放信用点时可留空，并在商品编辑页填写附带信用点。</p>}
      <div className="field"><label>允许领取的服务器</label><div className="template-scope">{allowed.map((node) => <label className="checkbox-row" key={node.serverId}><input type="checkbox" checked={servers.includes(node.serverId)} onChange={(e) => setServers(e.target.checked ? [...servers, node.serverId] : servers.filter((server) => server !== node.serverId))} />{node.serverId}<small>{node.online ? "在线" : "离线"}</small></label>)}</div>{!allowed.length && <p className="muted">先选择背包同步组，再查看可以领取的服务器。</p>}</div>
      <Toggle label="启用模板" description="启用后可用于新商品；停用会停止接受关联商品的新订单。" checked={active} onChange={setActive} />
    </fieldset>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="editor-actions">{onDisable && <Button secondary disabled={busy} onClick={onDisable}>停用模板</Button>}<Button type="submit" disabled={busy}>保存交付模板</Button></div>
  </form>;
}

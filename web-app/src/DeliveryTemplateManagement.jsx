import React, { useEffect, useRef, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Badge, Button, Empty, Field, Modal } from "./components.jsx";
import { id } from "./format.js";

export default function DeliveryTemplateManagement({ client, storeId, onChanged }) {
  const [templates, setTemplates] = useState([]), [error, setError] = useState(""), [busy, setBusy] = useState(false), [editor, setEditor] = useState(null), [disabling, setDisabling] = useState(null);
  const actionKeys = useRef({}), base = `/api/v1/merchant/stores/${encodeURIComponent(storeId)}/delivery-templates`;
  const load = async () => { setBusy(true); setError(""); try { const r = await client.request(base + "?limit=100"); setTemplates(r.data); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  useEffect(() => { load(); }, [storeId]);
  const edit = async (template) => { setBusy(true); setError(""); try { const r = await client.request(base + "/" + encodeURIComponent(template.templateRef)); setEditor(r.data); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  const disable = async () => { setBusy(true); setError(""); const scope = `${disabling.templateRef}:${disabling.version}`; actionKeys.current[scope] ||= id(); try { await client.request(base + "/" + encodeURIComponent(disabling.templateRef) + "/disable", { method: "POST", body: { clientRequestId: actionKeys.current[scope], expectedVersion: disabling.version } }); setDisabling(null); await load(); await onChanged?.(); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  return <><p className="description">选择 Core 已发布的物品版本，设定附件数量与允许领取的服务器。</p><div className="button-row"><Button onClick={() => setEditor({})}><Plus size={16} />创建交付模板</Button><Button secondary onClick={load} disabled={busy}>刷新</Button></div>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="foundation-grid">{templates.map((template) => <div key={template.templateRef} className="foundation-card"><Badge tone={template.active ? "sage" : "neutral"}>{template.active ? "已启用" : "已停用"}</Badge><h3>{template.name}</h3><p>{template.summary}</p><Button secondary onClick={() => edit(template)} disabled={busy}>查看与编辑</Button></div>)}</div>
    {!busy && !templates.length && <Empty title="还没有交付模板" text="先选择真实的 Core 物品版本，再创建可用于商品的交付模板。" />}
    {editor && <Modal title={editor.templateRef ? "编辑交付模板" : "创建交付模板"} close={() => setEditor(null)} wide><TemplateForm client={client} base={base} initial={editor} onSaved={async () => { setEditor(null); await load(); await onChanged?.(); }} onDisable={editor.templateRef && editor.active ? () => { setDisabling(editor); setEditor(null); } : null} /></Modal>}
    {disabling && <Modal title="停用交付模板" close={() => setDisabling(null)}><p>停用“{disabling.name}”后，引用它的商品将无法通过新的交付校验。已经冻结的订单快照保留。</p>{error && <p className="auth-error" role="alert">{error}</p>}<Button disabled={busy} onClick={disable}>确认停用</Button></Modal>}
  </>;
}

function TemplateForm({ client, base, initial, onSaved, onDisable }) {
  const [name, setName] = useState(initial.name || ""), [summary, setSummary] = useState(initial.summary || ""), [domain, setDomain] = useState(initial.inventoryDomain || ""),
    [servers, setServers] = useState(initial.allowedServerIds || []), [attachments, setAttachments] = useState(initial.attachments || []), [active, setActive] = useState(initial.active ?? true),
    [items, setItems] = useState([]), [nodes, setNodes] = useState([]), [cursor, setCursor] = useState(null), [choice, setChoice] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const mutation = useRef(null);
  const load = async () => { setBusy(true); setError(""); try { const [n, i] = await Promise.all([client.nodes(), client.items()]); setNodes(n.data.nodes); setItems(i.data.items); setCursor(i.data.next); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  useEffect(() => { load(); }, []);
  const key = (item) => `${item.itemRef}:${item.revision}`;
  const domains = [...new Set(nodes.filter((node) => node.claimEnabled).map((node) => node.inventoryDomain))];
  const choices = items.filter((item) => (!domain || item.inventoryDomain === domain) && !attachments.some((attached) => attached.itemRef === item.itemRef));
  const allowed = nodes.filter((node) => node.claimEnabled && node.inventoryDomain === domain && attachments.every((attachment) => {
    const item = items.find((value) => value.itemRef === attachment.itemRef && value.revision === attachment.revision);
    return !item || item.compatibleServerIds?.includes(node.serverId);
  }));
  const add = () => {
    const item = choices.find((value) => key(value) === choice); if (!item) return;
    if (!item.inventoryDomain || !item.payloadSha256) { setError("这个物品版本的交付信息尚未完整同步，请刷新后重试。"); return; }
    if (!domain) setDomain(item.inventoryDomain);
    setAttachments([...attachments, { itemRef: item.itemRef, revision: item.revision, quantity: 1, payloadSha256: item.payloadSha256 }]); setChoice("");
  };
  const save = async (e) => {
    e.preventDefault(); setBusy(true); setError("");
    try {
      if (!attachments.length || !servers.length) throw new Error("请至少选择一个附件和一个允许领取的服务器。");
      if (servers.some((server) => !allowed.some((node) => node.serverId === server))) throw new Error("已选服务器不适用于当前附件，请重新选择。");
      const input = { name, summary, inventoryDomain: domain, allowedServerIds: servers, attachments, active, ...(initial.templateRef ? { expectedVersion: initial.version } : {}) }, fingerprint = JSON.stringify(input);
      if (mutation.current?.fingerprint !== fingerprint) mutation.current = { fingerprint, body: { clientRequestId: id(), ...input } };
      await client.request(base + (initial.templateRef ? "/" + encodeURIComponent(initial.templateRef) : ""), { method: initial.templateRef ? "PUT" : "POST", body: mutation.current.body }); await onSaved();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={save}><fieldset disabled={busy} style={{ border: 0, margin: 0, padding: 0, minWidth: 0 }}>
    <Field label="模板名称" required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} /><Field label="交付摘要"><textarea required maxLength={1000} rows={3} value={summary} onChange={(e) => setSummary(e.target.value)} /></Field>
    <Field label="背包同步组"><select required value={domain} onChange={(e) => { setDomain(e.target.value); setServers([]); }}><option value="">选择背包同步组</option>{domains.map((value) => <option key={value} value={value}>{value}</option>)}</select></Field>
    <Field label="已发布物品版本"><select value={choice} onChange={(e) => setChoice(e.target.value)}><option value="">选择物品版本</option>{choices.map((item) => <option key={key(item)} value={key(item)}>{item.displayName} · 版本 {item.revision}</option>)}</select></Field><Button secondary disabled={!choice || attachments.length >= 32} onClick={add}>加入附件</Button>
    {!items.length && <p className="notice-box">Core 尚未提供可用的物品目录。同步完成后，可以在这里选择真实物品。</p>}
    {cursor && <Button secondary onClick={async () => { setBusy(true); try { const r = await client.items(cursor); setItems((old) => [...old, ...r.data.items]); setCursor(r.data.next); } catch (e) { setError(e.message); } finally { setBusy(false); } }}>加载更多物品版本</Button>}
    {attachments.map((attachment, index) => { const item = items.find((value) => value.itemRef === attachment.itemRef && value.revision === attachment.revision); return <div className="panel" style={{ padding: 14, marginTop: 12 }} key={attachment.itemRef}><h4>{item?.displayName || attachment.itemRef} · 版本 {attachment.revision}</h4><Field label="附件数量" type="number" min={1} max={Math.min(99999, item?.maxQuantity || 99999)} required value={attachment.quantity} onChange={(e) => setAttachments((old) => old.map((value, n) => n === index ? { ...value, quantity: Number(e.target.value) } : value))} /><Button secondary aria-label={`移除附件 ${attachment.itemRef}`} onClick={() => setAttachments((old) => old.filter((_, n) => n !== index))}><Trash2 size={14} />移除</Button></div>; })}
    <div className="field"><label>允许领取的服务器</label>{allowed.map((node) => <label className="checkbox-row" key={node.serverId}><input type="checkbox" checked={servers.includes(node.serverId)} onChange={(e) => setServers(e.target.checked ? [...servers, node.serverId] : servers.filter((server) => server !== node.serverId))} />{node.serverId} · {node.online ? "在线" : "离线"}</label>)}{!allowed.length && <p className="muted">没有与当前背包组及附件兼容的可领取服务器。</p>}</div>
    <label className="checkbox-row"><input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />启用此交付模板</label>
  </fieldset>{error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button type="submit" disabled={busy}>保存交付模板</Button>{onDisable && <Button secondary disabled={busy} onClick={onDisable}>停用</Button>}</div></form>;
}

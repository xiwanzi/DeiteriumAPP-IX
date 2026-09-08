import React, { useEffect, useRef, useState } from "react";
import { Badge, Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { MediaPicker } from "./MediaPicker.jsx";
import { credit, id } from "./format.js";
import { fundsLabels } from "./business.js";
import EvidenceMessagePicker from "./EvidenceMessagePicker.jsx";

const states = { SUBMITTED: "待受理", IN_REVIEW: "处理中", WAITING_EVIDENCE: "待补充资料", RESOLVING: "资金执行中", RESOLVED: "已处理", WITHDRAWN: "已撤回" };

export default function InterventionManagement({ client, user }) {
  const [status, setStatus] = useState(""), [mine, setMine] = useState(false), [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""), [selected, setSelected] = useState(null);
  const generation = useRef(0);
  const load = async (more = false) => { const current = ++generation.current; setBusy(true); setError(""); try { const r = await client.request(`/api/v1/admin/interventions?limit=30${status ? `&status=${status}` : ""}${mine ? "&assignedToMe=true" : ""}${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); if (current === generation.current) { setItems((old) => more ? [...old, ...r.data] : r.data); setCursor(r.page?.nextCursor); } } catch (e) { if (current === generation.current) setError(e.message); } finally { if (current === generation.current) setBusy(false); } };
  useEffect(() => { setItems([]); load(); return () => { generation.current++; }; }, [status, mine]);
  return <><PageHead eyebrow="PLATFORM SUPPORT" title="平台介入" subtitle="依据成交快照与当事人提交的资料处理争议。"><Button secondary disabled={busy} onClick={() => load()}>刷新</Button></PageHead>
    <div className="filter-header"><Field label="案件状态"><select value={status} onChange={(e) => setStatus(e.target.value)}><option value="">全部状态</option>{Object.entries(states).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></Field><label className="checkbox-row"><input type="checkbox" checked={mine} onChange={(e) => setMine(e.target.checked)} />只看由我处理的案件</label></div>
    {error && <p className="notice-box" role="alert">{error}</p>}
    <div className="foundation-grid">{items.map((item) => <button className="foundation-card" key={item.caseId} style={{ textAlign: "left", color: "inherit" }} onClick={() => setSelected(item.caseId)}><Badge>{states[item.status] || item.status}</Badge><h3>{item.applicant?.displayName} 与 {item.respondent?.displayName || "交易方"}</h3><p>{item.description}</p><small>{new Date(item.updatedAt).toLocaleString("zh-CN")}</small></button>)}</div>
    {!busy && !items.length && <Empty title="暂无介入案件" text="收到申请后会在这里显示。" />}{cursor && <Button secondary disabled={busy} onClick={() => load(true)}>加载更多</Button>}
    {selected && <Modal title="处理平台介入" close={() => setSelected(null)} wide><InterventionPanel client={client} user={user} reference={selected} admin onChanged={() => load()} /></Modal>}
  </>;
}

export function InterventionPanel({ client, user, reference, admin = false, onChanged }) {
  const [value, setValue] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false), [action, setAction] = useState(null), [reason, setReason] = useState(""), [decision, setDecision] = useState("CONTINUE_FULFILLMENT"), [amount, setAmount] = useState(""), [evidence, setEvidence] = useState([]), [messages, setMessages] = useState([]);
  const pending = useRef(null), current = useRef(value), alive = useRef(true); current.current = value;
  const base = `/api/v1/${admin ? "admin/" : ""}interventions/${encodeURIComponent(reference)}`;
  const load = async () => { try { const r = await client.request(base); if (alive.current) { setValue(r.data); setError(""); } } catch (e) { if (alive.current) setError(e.message); } };
  useEffect(() => { alive.current = true; load(); const timer = setInterval(() => { if (!document.hidden && current.current?.status === "RESOLVING") load(); }, 5000); return () => { alive.current = false; clearInterval(timer); }; }, [reference, admin]);
  const choose = (next) => { setAction(next); setReason(""); setEvidence([]); setMessages([]); setAmount(""); pending.current = null; };
  const submit = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      const body = { expectedVersion: value.version };
      if (action === "evidence") { if (!evidence.length) throw new Error("请添加至少一张补充资料图片。"); body.description = reason; body.evidenceAssetIds = evidence.map((asset) => asset.assetId); body.relatedMessageIds = messages.map((message) => message.messageId); }
      else if (action !== "assign") body.reason = reason;
      if (action === "resolve") { body.decision = decision; if (decision === "PARTIAL_REFUND") body.refundAmount = amount.trim(); }
      const fingerprint = JSON.stringify(body); if (pending.current?.fingerprint !== fingerprint) pending.current = { fingerprint, body: { clientRequestId: id(), ...body } };
      const r = await client.request(base + "/" + action, { method: "POST", body: pending.current.body }); if (!r.data.caseId) throw new Error("平台处理结果尚未确认，请刷新记录。"); setValue(r.data); setAction(null); await onChanged?.();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  if (!value) return <Empty title={error ? "暂时无法读取记录" : "正在读取记录"} text={error || "请稍候…"}><Button secondary onClick={load}>刷新</Button></Empty>;
  const closed = ["RESOLVED", "WITHDRAWN"].includes(value.status), processing = value.status === "RESOLVING", snapshot = value.snapshot?.order || value.snapshot?.commission;
  return <><div className="button-row"><Badge tone={closed ? "sage" : "neutral"}>{states[value.status] || value.status}</Badge><Button secondary disabled={busy} onClick={load}>刷新记录</Button></div><h3>{value.applicant?.displayName} 与 {value.respondent?.displayName || "交易方"}</h3><p style={{ whiteSpace: "pre-wrap" }}>{value.description}</p>
    <dl className="detail-list"><div><dt>期望处理</dt><dd>{{ FULL_REFUND: "全额退款", PARTIAL_REFUND: "部分退款", CONTINUE_FULFILLMENT: "继续履约", OTHER: "其他" }[value.desiredResolution] || value.desiredResolution}</dd></div>{value.requestedRefundAmount && <div><dt>申请退款金额</dt><dd>{credit(value.requestedRefundAmount)} 信用点</dd></div>}<div><dt>更新时间</dt><dd>{new Date(value.updatedAt).toLocaleString("zh-CN")}</dd></div></dl>
    {processing && <p className="notice-box">资金指令仍在执行或核对。处理记录会继续更新，确认前不能再次裁决。</p>}
    {value.resolution && <p className="notice-box">处理说明：{value.resolution}</p>}
    {snapshot && <details className="panel" style={{ padding: 18, marginTop: 18 }}><summary>查看成交快照</summary><p className="muted">采集于 {new Date(value.snapshot.capturedAt).toLocaleString("zh-CN")}</p><p>{snapshot.content?.title || snapshot.items?.map((item) => item.title).join("、")}</p><p className="price">{credit(snapshot.amount ?? snapshot.content?.reward)}<small>信用点</small></p><p>{fundsLabels[snapshot.fundsStatus] || snapshot.fundsStatus}</p>{snapshot.items?.map((item) => <p key={item.productId}>{item.title} × {item.quantity} · {credit(item.unitPrice)} 信用点</p>)}{value.snapshot.eventLog?.map((event) => <p key={event.eventId}>{new Date(event.occurredAt).toLocaleString("zh-CN")} · {event.summary}</p>)}</details>}
    {(value.evidence || value.evidenceAssets || []).map((asset) => asset.url && <img key={asset.assetId} src={asset.url} alt={asset.altText || "案件资料"} style={{ width: "100%", maxHeight: 400, objectFit: "contain", borderRadius: 10, marginTop: 12 }} />)}
    {value.completionAssets?.length > 0 && <div className="panel" style={{ padding: 16, marginTop: 12 }}><h3>成交快照中的完成资料</h3>{value.completionAssets.map((asset) => asset.url && <img key={asset.assetId} src={asset.url} alt={asset.altText || "完成资料"} style={{ maxWidth: "100%", maxHeight: 400, objectFit: "contain" }} />)}</div>}
    {value.evidenceEntries?.map((entry, index) => <div className="panel" key={entry.entryId || index} style={{ padding: 16, marginTop: 12 }}><small className="muted">{entry.createdAt && new Date(entry.createdAt).toLocaleString("zh-CN")} · {entry.actorPlayerRef}</small><p style={{ whiteSpace: "pre-wrap" }}>{entry.description}</p>{(entry.assets || []).map((asset) => <img key={asset.assetId} src={asset.url} alt={asset.altText || "补充资料"} style={{ maxWidth: "100%", maxHeight: 300 }} />)}{entry.relatedMessages?.map((message) => <blockquote key={message.messageId} style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}><small>{message.senderPlayerRef}</small><p>{message.content}</p></blockquote>)}</div>)}
    {error && <p className="auth-error" role="alert">{error}</p>}
    {!closed && <div className="button-row">{admin ? <><Button secondary disabled={busy || processing} onClick={() => choose("assign")}>由我处理</Button><Button secondary disabled={busy || processing} onClick={() => choose("request-evidence")}>要求补充资料</Button><Button disabled={busy || processing} onClick={() => choose("resolve")}>提交处理决定</Button></> : <><Button secondary disabled={busy || processing} onClick={() => choose("evidence")}>补充资料</Button>{value.applicant?.playerRef === user?.playerRef && <Button secondary disabled={busy || processing} onClick={() => choose("withdraw")}>撤回申请</Button>}</>}</div>}
    {action && <Modal title={{ assign: "受理案件", "request-evidence": "要求补充资料", resolve: "处理决定", evidence: "补充资料", withdraw: "撤回申请" }[action]} close={() => setAction(null)} wide={action === "resolve" || action === "evidence"}><form onSubmit={submit}>
      {action === "assign" && <p>确认由当前账号负责处理这起案件？</p>}
      {action === "resolve" && <><Field label="处理决定"><select value={decision} onChange={(e) => setDecision(e.target.value)}><option value="CONTINUE_FULFILLMENT">继续履约</option><option value="FULL_REFUND">全额退款</option><option value="PARTIAL_REFUND">部分退款</option><option value="RELEASE_TO_PAYEE">结算给收款方</option><option value="REQUIRE_MANUAL_RECOVERY">保留人工核对</option><option value="NO_ACTION">无需资金调整</option></select></Field>{decision === "PARTIAL_REFUND" && <Field label="退款金额" required inputMode="decimal" maxLength={20} value={amount} onChange={(e) => setAmount(e.target.value)} />}</>}
      {action !== "assign" && <Field label={action === "resolve" ? "处理依据" : "说明"}><textarea required minLength={action === "resolve" ? 10 : 2} maxLength={action === "resolve" || action === "evidence" ? 3000 : 500} rows={5} value={reason} onChange={(e) => setReason(e.target.value)} /></Field>}
      {action === "evidence" && <MediaPicker client={client} images={evidence} onChange={setEvidence} purpose="DISPUTE_EVIDENCE" businessType="INTERVENTION" businessRef={reference} max={10} onBusy={setBusy} />}
      {action === "evidence" && <EvidenceMessagePicker client={client} selected={messages} onChange={setMessages} />}
      {error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>{busy ? "正在提交…" : "确认提交"}</Button>
    </form></Modal>}
  </>;
}

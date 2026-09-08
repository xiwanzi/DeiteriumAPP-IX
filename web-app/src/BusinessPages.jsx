import React, { useEffect, useRef, useState } from "react";
import { Plus, RefreshCw } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { MediaPicker } from "./MediaPicker.jsx";
import { CreationProgress } from "./PurchaseFlow.jsx";
import { businessOperation, businessResource, clearBusiness, fundsLabels, fundsPending, resourceId, resourcePath, saveBusiness, savedBusiness } from "./business.js";
import { credit, id } from "./format.js";
import { InterventionPanel } from "./Interventions.jsx";
import EvidenceMessagePicker from "./EvidenceMessagePicker.jsx";

const orderStatus = { PAYMENT_PROCESSING: "付款处理中", AWAITING_SHIPMENT: "待交付", SHIPPED: "已发货", WORK_COMPLETED: "待验收", CONFIRMED: "已完成", AWAITING_CLAIM: "等待邮箱领取", CLAIMED: "已领取", REFUNDED: "已退款", CANCELLED: "已取消" };
const commissionStatus = { FUNDING: "预付处理中", OPEN: "待接取", ACTIVE: "进行中", COMPLETED: "等待验收", CONFIRMED: "已完成", CANCELLED: "已取消" };
const actionLabels = { SHIP: "确认发货", START_WORK: "开始施工", COMPLETE_WORK: "提交工程完成", CONFIRM_RECEIPT: "确认收货", CONFIRM_ACCEPTANCE: "确认验收", ACCEPT: "接取委托", CANCEL: "取消委托", COMPLETE: "提交完成", CONFIRM: "确认完成", REQUEST_REFUND: "申请退款", WITHDRAW_REFUND: "撤回退款", RESOLVE_REFUND: "处理退款", REQUEST_INTERVENTION: "申请平台介入", VIEW_MAILBOX: "查看邮箱交付" };
const refundStatus = { REQUESTED: "等待处理", PROCESSING: "退款处理中", APPROVED: "已同意", REJECTED: "已拒绝", WITHDRAWN: "已撤回", REFUNDED: "已退款" };

export function BusinessDetail({ client, user, type, reference, initial, onChanged }) {
  const [value, setValue] = useState(initial || null), [error, setError] = useState(""), [busy, setBusy] = useState(false), [action, setAction] = useState(null), [mailbox, setMailbox] = useState(false), [intervention, setIntervention] = useState(null), [snapshot, setSnapshot] = useState(null);
  const alive = useRef(true), current = useRef(value), loading = useRef(false); current.current = value;
  const load = async () => {
    if (loading.current) return; loading.current = true;
    try {
      let operationError;
      if (current.current?.pendingOperationId) { try { await client.request(`/api/v1/operations/${encodeURIComponent(current.current.pendingOperationId)}`); } catch (e) { if (e.status === 401) throw e; operationError = e; } }
      const result = await client.request(resourcePath(type, reference)); if (alive.current) { const parsed = businessResource(result.data); if (!parsed) throw new Error("业务详情格式不正确。"); setValue(parsed.value); setError(""); }
      if (operationError && alive.current && fundsPending(businessResource(result.data)?.value)) setError(operationError.message);
    } catch (e) { if (alive.current) setError(e.message); } finally { loading.current = false; }
  };
  useEffect(() => { alive.current = true; load(); const timer = setInterval(() => { if (!document.hidden && current.current && fundsPending(current.current)) load(); }, 5000); return () => { alive.current = false; clearInterval(timer); }; }, [reference]);
  if (!value) return <Empty title={error ? "暂时无法读取详情" : "正在读取详情"} text={error || "请稍候…"}><Button secondary onClick={load}>重新读取</Button></Empty>;
  const pending = fundsPending(value), label = type === "COMMISSION" ? value.status === "ACTIVE" && pending ? "正在确认接取" : commissionStatus[value.status] : value.construction && value.status === "SHIPPED" ? "施工中" : orderStatus[value.status];
  const updated = async (data) => { const resource = businessResource(data); if (resource) setValue(resource.value); setAction(null); await load(); await onChanged?.(); };
  return <>
    <div className="button-row"><Badge tone={pending ? "neutral" : "blue"}>{label || value.status}</Badge><Badge tone="neutral">{fundsLabels[value.fundsStatus] || value.fundsStatus}</Badge><Button secondary disabled={busy} onClick={load}><RefreshCw size={15} />刷新</Button></div>
    <h2>{type === "COMMISSION" ? value.content?.title : value.items?.map((item) => item.title).join("、") || value.orderNo}</h2>
    <p className="price">{credit(value.amount ?? value.content?.reward)}<small>信用点</small></p>
    {pending && <p className="notice-box">资金结果仍在核对。页面会查询原处理记录，确认前暂不开放新的资金操作。</p>}
    {value.interventionCaseId && <p className="notice-box">这笔业务有平台处理记录。<Button secondary onClick={() => setIntervention(value.interventionCaseId)}>查看处理记录</Button></p>}
    <dl className="detail-list">
      <div><dt>{type === "COMMISSION" ? "发布者" : "买方"}</dt><dd>{(value.owner || value.buyer)?.displayName}</dd></div>
      <div><dt>{type === "COMMISSION" ? "接取者" : "卖方"}</dt><dd>{(value.worker || value.seller)?.displayName || "尚未接取"}</dd></div>
      <div><dt>创建时间</dt><dd>{new Date(value.createdAt).toLocaleString("zh-CN")}</dd></div>
      {(value.workDueAt || value.autoConfirmAt) && <div><dt>{type === "COMMISSION" ? "履约截止" : "自动确认时间"}</dt><dd>{new Date(value.workDueAt || value.autoConfirmAt).toLocaleString("zh-CN")}</dd></div>}
      {value.acceptanceDueAt && <div><dt>验收截止</dt><dd>{new Date(value.acceptanceDueAt).toLocaleString("zh-CN")}</dd></div>}
      {value.pausedRemainingSeconds !== null && value.pausedRemainingSeconds !== undefined && <div><dt>计时状态</dt><dd>已暂停，剩余约 {Math.ceil(value.pausedRemainingSeconds / 3600)} 小时</dd></div>}
      {(value.delivery?.location || value.content?.location) && <div><dt>地点</dt><dd>{value.delivery?.location || value.content?.location}</dd></div>}
      {(value.seller?.contactQq || value.worker?.contactQq) && <div><dt>联系 QQ</dt><dd>{value.seller?.contactQq || value.worker?.contactQq}</dd></div>}
    </dl>
    {type === "COMMISSION" ? <p style={{ whiteSpace: "pre-wrap" }}>{value.content?.description}</p> : value.items?.map((item) => <div className="panel" style={{ padding: 16, marginTop: 12 }} key={item.productId}><h3>{item.title} × {item.quantity}</h3><p>{item.subtitle}</p><p style={{ whiteSpace: "pre-wrap" }}>{item.description}</p><small>{credit(item.unitPrice)} 信用点 / 件</small></div>)}
    {value.completionDescription && <div className="panel" style={{ padding: 16, marginTop: 16 }}><h3>完成说明</h3><p style={{ whiteSpace: "pre-wrap" }}>{value.completionDescription}</p>{value.completionAssets?.map((asset) => asset.url && <img key={asset.assetId} src={asset.url} alt={asset.altText || "完成资料"} style={{ maxWidth: "100%", maxHeight: 400, objectFit: "contain" }} />)}</div>}
    <div className="button-row"><Button secondary disabled={busy} onClick={async () => { setBusy(true); try { const r = await client.request(resourcePath(type, reference) + "/snapshot"); setSnapshot(r.data); } catch (e) { setError(e.message); } finally { setBusy(false); } }}>查看成交约定</Button></div>
    {value.refund && <div className="notice-box"><h3>退款 · {refundStatus[value.refund.status] || value.refund.status}</h3><p>{value.refund.reason}</p>{value.refund.rejectionReason && <p>拒绝原因：{value.refund.rejectionReason}</p>}<p>已使用 {value.refundAttemptsUsed} 次退款申请机会</p></div>}
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="button-row">{(value.availableActions || []).filter((key) => actionLabels[key]).map((key) => <Button key={key} secondary={["REQUEST_REFUND", "WITHDRAW_REFUND", "VIEW_MAILBOX", "REQUEST_INTERVENTION"].includes(key)} disabled={busy || pending && key !== "VIEW_MAILBOX"} onClick={async () => {
      if (key === "VIEW_MAILBOX") { setBusy(true); try { const result = await client.request(`/api/v1/orders/${encodeURIComponent(reference)}/mailbox`); const resource = businessResource(result.data); if (resource) setValue(resource.value); setMailbox(true); } catch (e) { setError(e.message); } finally { setBusy(false); } }
      else setAction(key);
    }}>{actionLabels[key]}</Button>)}</div>
    {action && <Modal title={actionLabels[action]} close={() => setAction(null)} wide={action === "REQUEST_INTERVENTION"}><BusinessAction client={client} user={user} type={type} value={value} action={action} onUpdated={updated} onIntervention={(caseId) => { setAction(null); setIntervention(caseId); load(); }} /></Modal>}
    {mailbox && <Modal title="游戏内邮箱交付" close={() => setMailbox(false)}><p>当前订单状态：{orderStatus[value.status] || value.status}。</p><p>请进入支持领取的游戏服务器，打开游戏内邮箱查看与领取。是否可以退款以服务器最新领取记录为准。</p></Modal>}
    {intervention && <Modal title="平台处理记录" close={() => setIntervention(null)} wide><InterventionPanel client={client} user={user} reference={intervention} onChanged={load} /></Modal>}
    {snapshot && <Modal title="成交时的约定" close={() => setSnapshot(null)} wide><p className="muted">记录于 {new Date(snapshot.capturedAt).toLocaleString("zh-CN")}</p><h3>{snapshot.content?.title || snapshot.items?.map((item) => item.title).join("、")}</h3><p className="price">{credit(snapshot.totalAmount ?? snapshot.content?.reward)}<small>信用点</small></p>{snapshot.items?.map((item) => <div key={item.productId}><h3>{item.title} × {item.quantity}</h3><p style={{ whiteSpace: "pre-wrap" }}>{item.description || item.subtitle}</p><p>{credit(item.unitPrice)} 信用点 / 件</p></div>)}{snapshot.content && <><p style={{ whiteSpace: "pre-wrap" }}>{snapshot.content.description}</p><p>地点：{snapshot.content.location}</p><p>履约时限：{snapshot.content.workHours} 小时</p></>}{snapshot.delivery?.location && <p>交付地点：{snapshot.delivery.location}</p>}{(snapshot.acceptanceHours || snapshot.confirmationHours) && <p>验收时限：{snapshot.acceptanceHours || snapshot.confirmationHours} 小时</p>}</Modal>}
  </>;
}

function BusinessAction({ client, user, type, value, action, onUpdated, onIntervention }) {
  const [description, setDescription] = useState(""), [reasonCode, setReasonCode] = useState(action === "REQUEST_INTERVENTION" ? "REFUND_DISAGREEMENT" : "NO_LONGER_NEEDED"), [decision, setDecision] = useState("APPROVE"), [resolution, setResolution] = useState("FULL_REFUND"), [refundAmount, setRefundAmount] = useState(""), [evidence, setEvidence] = useState([]), [busy, setBusy] = useState(false), [error, setError] = useState(""), [messages, setMessages] = useState([]);
  const pending = useRef(null), reference = resourceId(value), base = resourcePath(type, reference), completing = ["COMPLETE", "COMPLETE_WORK"].includes(action), refund = action === "REQUEST_REFUND", resolve = action === "RESOLVE_REFUND", intervene = action === "REQUEST_INTERVENTION";
  const submit = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      let suffix = ({ SHIP: "ship", START_WORK: "start-work", COMPLETE_WORK: "complete-work", CONFIRM_RECEIPT: "confirm", CONFIRM_ACCEPTANCE: "confirm", ACCEPT: "accept", CANCEL: "cancel", COMPLETE: "complete", CONFIRM: "confirm", REQUEST_REFUND: "refunds", REQUEST_INTERVENTION: "interventions" })[action];
      const nestedRefund = ["WITHDRAW_REFUND", "RESOLVE_REFUND"].includes(action);
      if (nestedRefund) { if (!value.refund?.refundId || !value.refund?.version) throw new Error("退款状态已变化，请刷新详情。"); suffix = `refunds/${encodeURIComponent(value.refund.refundId)}/${resolve ? "resolve" : "withdraw"}`; }
      const body = { expectedVersion: nestedRefund ? value.refund.version : value.version };
      if (completing || refund || intervene) { body.description = description; body.evidenceAssetIds = evidence.map((asset) => asset.assetId); }
      if (refund || intervene) body.reasonCode = reasonCode;
      if (resolve) { body.decision = decision; body.reason = description; }
      if (intervene) { body.desiredResolution = resolution; body.relatedMessageIds = messages.map((message) => message.messageId); if (resolution === "PARTIAL_REFUND") body.requestedRefundAmount = refundAmount.trim(); }
      const fingerprint = JSON.stringify(body); if (pending.current?.fingerprint !== fingerprint) pending.current = { fingerprint, body: { clientRequestId: id(), ...body } };
      const result = await client.request(base + "/" + suffix, { method: "POST", body: pending.current.body });
      if (intervene) { const caseId = result.data.caseId || result.data.interventionCaseId; if (!caseId) throw new Error("平台尚未返回处理记录，请使用原请求重试。"); onIntervention(caseId); }
      else await onUpdated(result.data);
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={submit}><fieldset disabled={busy} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
    {refund && <p className="notice-box">退款申请只有一次机会。提交后即计入申请次数，撤回或被拒绝不会恢复次数。</p>}
    {action === "WITHDRAW_REFUND" && <p className="notice-box">撤回后不会恢复退款机会，暂停的确认计时会继续。</p>}
    {["CONFIRM", "CONFIRM_ACCEPTANCE", "CONFIRM_RECEIPT"].includes(action) && <p className="notice-box">确认后将按约定结算给对方，请确认已经收到商品或验收完成。</p>}
    {action === "ACCEPT" && <p>接取后会锁定约定与报酬，并开始履约计时。资金绑定未确认时会继续核对。</p>}
    {action === "CANCEL" && <p>确认取消这项委托？预付报酬将在退款确认后返还。</p>}
    {refund && <Field label="退款原因"><select value={reasonCode} onChange={(e) => setReasonCode(e.target.value)}><option value="NO_LONGER_NEEDED">不再需要</option><option value="DELIVERY_DELAY">交付延迟</option><option value="NOT_AS_DESCRIBED">与描述不符</option><option value="OTHER">其他</option></select></Field>}
    {resolve && <Field label="处理决定"><select value={decision} onChange={(e) => setDecision(e.target.value)}><option value="APPROVE">同意退款</option><option value="REJECT">拒绝退款</option></select></Field>}
    {intervene && <><Field label="介入原因"><select value={reasonCode} onChange={(e) => setReasonCode(e.target.value)}><option value="REFUND_DISAGREEMENT">退款存在争议</option><option value="NOT_DELIVERED">尚未交付</option><option value="NOT_AS_DESCRIBED">与描述不符</option><option value="OTHER">其他</option></select></Field><Field label="期望处理方式"><select value={resolution} onChange={(e) => setResolution(e.target.value)}><option value="FULL_REFUND">全额退款</option><option value="PARTIAL_REFUND">部分退款</option><option value="CONTINUE_FULFILLMENT">继续履约</option><option value="OTHER">其他</option></select></Field>{resolution === "PARTIAL_REFUND" && <Field label="期望退款金额" required inputMode="decimal" maxLength={20} value={refundAmount} onChange={(e) => setRefundAmount(e.target.value)} />}</>}
    {(completing || refund || resolve || intervene) && <Field label={completing ? "完成说明" : resolve ? "处理说明" : "详细说明"}><textarea rows={4} required={!resolve || decision === "REJECT"} minLength={intervene ? 10 : 2} maxLength={intervene ? 3000 : 500} value={description} onChange={(e) => setDescription(e.target.value)} /></Field>}
    {(completing || refund || intervene) && <MediaPicker client={client} images={evidence} onChange={setEvidence} purpose="DISPUTE_EVIDENCE" businessType={type} businessRef={reference} max={5} onBusy={setBusy} />}
    {intervene && <EvidenceMessagePicker client={client} selected={messages} onChange={setMessages} />}
  </fieldset>{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>{busy ? "正在提交…" : "确认提交"}</Button></form>;
}

export function InterventionDetail({ client, reference }) {
  const [value, setValue] = useState(null), [error, setError] = useState("");
  useEffect(() => { let active = true; client.request(`/api/v1/interventions/${encodeURIComponent(reference)}`).then((r) => { if (active) setValue(r.data); }).catch((e) => { if (active) setError(e.message); }); return () => { active = false; }; }, [reference]);
  if (!value) return <Empty title={error ? "暂时无法读取" : "正在读取"} text={error || "请稍候…"} />;
  return <><Badge>{value.status || value.state}</Badge><h3>平台处理记录</h3><p style={{ whiteSpace: "pre-wrap" }}>{value.description || value.reason || value.summary}</p>{value.resolution && <p className="notice-box">{value.resolution}</p>}<p className="muted">{value.updatedAt && new Date(value.updatedAt).toLocaleString("zh-CN")}</p></>;
}

export function CommissionForm({ client, user, onResource }) {
  const [value, setValue] = useState({ title: "", description: "", location: "", urgency: "NORMAL", reward: "", workHours: 24 }), [images, setImages] = useState([]), [confirming, setConfirming] = useState(false), [entry, setEntry] = useState(null), [result, setResult] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""); const request = useRef(null);
  const submit = async () => {
    if (busy) return; setBusy(true); setError("");
    const current = request.current;
    try { saveBusiness(user.userId, current); const r = await client.request("/api/v1/commissions", { method: "POST", body: current.body }); setResult(r.data); setEntry(current); }
    catch (e) { if (e.status >= 400 && e.status < 500 && ![408, 429].includes(e.status)) { clearBusiness(user.userId, current.body.clientRequestId); setError(e.message); } else { setError(e.message); setEntry(current); } }
    finally { setBusy(false); }
  };
  if (entry) return <CreationProgress client={client} user={user} entry={entry} initialResult={result} onResource={onResource} />;
  if (confirming) return <><h2>确认预付报酬</h2><p>{value.title}</p><p className="price">{credit(value.reward)}<small>信用点</small></p><p className="notice-box">预付报酬将由平台托管，资金确认后才会公开为待接取委托。</p>{error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button disabled={busy} onClick={submit}>确认预付并发布</Button><Button secondary disabled={busy} onClick={() => setConfirming(false)}>返回修改</Button></div></>;
  return <form onSubmit={(event) => { event.preventDefault(); setError(""); if (!images[0]) { setError("请先上传委托封面。"); return; } if (!/^(0|[1-9]\d*)(\.\d{1,2})?$/.test(value.reward) || /^0(?:\.0{1,2})?$/.test(value.reward)) { setError("请输入大于 0 的报酬，最多两位小数。"); return; } request.current = { userId: user.userId, origin: location.origin, path: "/api/v1/commissions", kind: "COMMISSION_PUBLISH", flow: "commission:" + id(), body: { clientRequestId: id(), content: { ...value, coverAssetId: images[0].assetId } } }; setConfirming(true); }}><fieldset disabled={busy} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
    <Field label="委托标题" required minLength={2} maxLength={60} value={value.title} onChange={(e) => setValue({ ...value, title: e.target.value })} /><Field label="委托说明"><textarea required minLength={5} maxLength={2000} rows={5} value={value.description} onChange={(e) => setValue({ ...value, description: e.target.value })} /></Field><Field label="地点" required minLength={2} maxLength={120} value={value.location} onChange={(e) => setValue({ ...value, location: e.target.value })} />
    <Field label="紧急程度"><select value={value.urgency} onChange={(e) => setValue({ ...value, urgency: e.target.value })}><option value="NORMAL">普通</option><option value="SOON">尽快</option><option value="URGENT">紧急</option></select></Field><Field label="履约时限（小时）" required type="number" min={1} max={8760} value={value.workHours} onChange={(e) => setValue({ ...value, workHours: Number(e.target.value) })} /><Field label="报酬（信用点）" required inputMode="decimal" maxLength={20} value={value.reward} onChange={(e) => setValue({ ...value, reward: e.target.value })} />
    <MediaPicker client={client} images={images} onChange={setImages} purpose="COMMISSION_COVER" businessType="COMMISSION" max={1} onBusy={setBusy} />
  </fieldset>{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>下一步</Button></form>;
}

export default function BusinessPages({ client, user, type = "ORDER", merchantStoreId }) {
  const [scope, setScope] = useState(type === "COMMISSION" ? "委托大厅" : "全部"), [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""), [selected, setSelected] = useState(null), [creating, setCreating] = useState(false), [pending, setPending] = useState(null);
  const generation = useRef(0), isCommission = type === "COMMISSION";
  const load = async (more = false) => {
    const current = ++generation.current; setBusy(true); setError("");
    try {
      const endpoint = isCommission ? scope === "委托大厅" ? "/commissions" : `/commissions/me?role=${scope === "我发布的" ? "PUBLISHER" : "WORKER"}` : merchantStoreId ? `/merchant/orders?storeId=${encodeURIComponent(merchantStoreId)}` : `/orders${scope === "全部" ? "" : `?role=${scope === "买入" ? "BUYER" : "SELLER"}`}`;
      const result = await client.request(`/api/v1${endpoint}${endpoint.includes("?") ? "&" : "?"}limit=30${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`);
      if (current !== generation.current) return; if (!Array.isArray(result.data)) throw new Error("业务列表格式不正确。"); setItems((old) => more ? [...old, ...result.data] : result.data); setCursor(result.page?.nextCursor);
    } catch (e) { if (current === generation.current) setError(e.message); } finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { setItems([]); load(); return () => { generation.current++; }; }, [scope, type, merchantStoreId]);
  useEffect(() => { const reference = new URLSearchParams(location.search).get(isCommission ? "commission" : "order"); if (reference) setSelected({ reference }); }, [type]);
  const onResource = (resource) => { setCreating(false); setPending(null); setSelected({ reference: resourceId(resource.value), value: resource.value }); load(); };
  const saved = Object.values(savedBusiness(user.userId)).filter((entry) => isCommission ? entry.kind === "COMMISSION_PUBLISH" : ["STORE_PURCHASE", "MARKET_PURCHASE"].includes(entry.kind));
  return <>
    <PageHead eyebrow={isCommission ? "COMMUNITY COMMISSIONS" : "YOUR ORDERS"} title={isCommission ? "委托大厅" : merchantStoreId ? "商店订单" : "我的订单"} subtitle={isCommission ? "共同完成想法，让每一份付出都有回报。" : "查看交付、验收和每一笔资金的进展。"}><Button secondary disabled={busy} onClick={() => load()}><RefreshCw size={16} />刷新</Button>{isCommission && <Button onClick={() => setCreating(true)}><Plus size={16} />发布委托</Button>}</PageHead>
    {!merchantStoreId && <Tabs values={isCommission ? ["委托大厅", "我发布的", "我接取的"] : ["全部", "买入", "卖出"]} value={scope} onChange={setScope} />}
    {saved.length > 0 && <div className="notice-box"><p>有 {saved.length} 笔提交等待核对。</p>{saved.map((entry, index) => <Button secondary key={entry.body.clientRequestId} onClick={() => setPending(entry)}>查看第 {index + 1} 笔</Button>)}</div>}
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="foundation-grid">{items.map((item) => <button className="foundation-card" style={{ textAlign: "left", color: "inherit" }} key={resourceId(item)} onClick={() => setSelected({ reference: resourceId(item), value: item })}>
      {item.cover?.url && <img src={item.cover.url} alt={item.content?.title || "委托封面"} style={{ width: "100%", aspectRatio: "16/10", objectFit: "cover", borderRadius: 12 }} />}
      <Badge>{(isCommission ? commissionStatus : orderStatus)[item.status] || item.status}</Badge><h3>{isCommission ? item.content?.title : item.items?.map((line) => line.title).join("、") || item.orderNo}</h3><p>{fundsLabels[item.fundsStatus] || item.fundsStatus}</p><strong className="price">{credit(item.amount ?? item.content?.reward)}<small>信用点</small></strong>
    </button>)}</div>
    {!busy && !error && !items.length && <Empty title={isCommission ? "当前没有委托" : "还没有订单"} text="有新的业务后会显示在这里。" />}{cursor && <Button secondary disabled={busy} onClick={() => load(true)}>加载更多</Button>}
    {selected && <Modal title={isCommission ? "委托详情" : "订单详情"} close={() => setSelected(null)} wide><BusinessDetail client={client} user={user} type={type} reference={selected.reference} initial={selected.value} onChanged={() => load()} /></Modal>}
    {creating && <Modal title="发布委托" close={() => setCreating(false)} wide><CommissionForm client={client} user={user} onResource={onResource} /></Modal>}
    {pending && <Modal title="核对原请求" close={() => setPending(null)}><CreationProgress client={client} user={user} entry={pending} onResource={onResource} /></Modal>}
  </>;
}

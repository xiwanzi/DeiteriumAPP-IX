import React, { useEffect, useRef, useState } from "react";
import { Badge, Button, Empty, Field } from "./components.jsx";
import { credit, id } from "./format.js";
import { businessOperation, businessResource, canReplayBusiness, clearBusiness, findBusiness, fundsLabels, fundsPending, paymentConfirmed, replayUnreceivedBusiness, resourceId, saveBusiness, savedBusiness } from "./business.js";

export function CreationProgress({ client, user, entry, initialResult, onResource, onRestart }) {
  const [operation, setOperation] = useState(() => businessOperation(initialResult)), [resource, setResource] = useState(() => businessResource(initialResult)), [error, setError] = useState(""), [busy, setBusy] = useState(false), [notFound, setNotFound] = useState(false), [expired, setExpired] = useState(false);
  const entryRef = useRef(entry), alive = useRef(true), resolving = useRef(false), finished = useRef(false);
  const apply = (op, result) => {
    if (!alive.current) return; setOperation(op); setResource(result); setError(""); setNotFound(false);
    const next = { ...entryRef.current, operationId: result?.value?.pendingOperationId || op?.operationId || entryRef.current.operationId, resourceType: result?.type || op?.resourceType || entryRef.current.resourceType, resourceId: resourceId(result?.value) || op?.resourceId || entryRef.current.resourceId };
    entryRef.current = next;
    finished.current = op?.status === "FAILED" || op?.status === "COMPLETED" && Boolean(result?.value) && !fundsPending(result.value);
    if (finished.current) clearBusiness(user.userId, next.body.clientRequestId);
    else saveBusiness(user.userId, next);
  };
  const lookup = async () => {
    if (resolving.current) return; resolving.current = true; setBusy(true);
    try { const value = await findBusiness(client, entryRef.current); apply(value.operation, value.resource); }
    catch (e) { if (alive.current) { if (e.knownOperation) apply(e.knownOperation, resource); setError(e.message); setNotFound(Boolean(e.originalOperationNotFound) && canReplayBusiness(entryRef.current, user.userId, location.origin)); } }
    finally { resolving.current = false; if (alive.current) setBusy(false); }
  };
  useEffect(() => {
    alive.current = true; if (initialResult) apply(businessOperation(initialResult), businessResource(initialResult)); else lookup();
    const timer = setInterval(() => { if (!finished.current && !document.hidden) lookup(); }, 5000);
    return () => { alive.current = false; clearInterval(timer); };
  }, []);
  const retryOriginal = async () => {
    if (resolving.current || !notFound) return; resolving.current = true; setBusy(true); setError("");
    try { const result = await replayUnreceivedBusiness(client, entryRef.current, user.userId, location.origin); apply(result.operation, result.resource); }
    catch (e) { if (e.knownOperation) apply(e.knownOperation, resource); setError(e.message); setNotFound(false); if (e.code === "QUOTE_EXPIRED") { setExpired(true); finished.current = true; clearBusiness(user.userId, entryRef.current.body.clientRequestId); } }
    finally { resolving.current = false; setBusy(false); }
  };
  const value = resource?.value, failed = operation?.status === "FAILED" || value?.fundsStatus === "UNPAID", confirmed = paymentConfirmed(value);
  return <div>
    <h2>{expired ? "报价已过期" : failed ? "这笔付款未完成" : confirmed ? "付款已确认" : "正在核对处理结果"}</h2>
    {value && <><Badge tone={confirmed ? "sage" : "neutral"}>{fundsLabels[value.fundsStatus] || value.fundsStatus}</Badge><p className="price">{credit(value.amount ?? value.content?.reward)}<small>信用点</small></p><p>{value.orderNo || value.content?.title}</p></>}
    {expired ? <p className="notice-box">原报价未能创建订单。请重新查看最新价格，并再次确认后付款。</p> : !confirmed && !failed && <p className="notice-box">请保留当前请求。系统会继续查询同一笔业务，结果未确认前不要重新付款。</p>}
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="button-row">{!expired && <Button secondary disabled={busy} onClick={lookup}>刷新处理结果</Button>}{notFound && <Button disabled={busy} onClick={retryOriginal}>重试原请求</Button>}{expired && onRestart && <Button onClick={onRestart}>重新获取报价</Button>}{resource && onResource && <Button onClick={() => onResource(resource)}>查看{resource.type === "ORDER" ? "订单" : "委托"}</Button>}</div>
  </div>;
}

export default function PurchaseFlow({ client, user, items, channel, listing, onResource }) {
  const flow = `${channel}:${items.map((item) => item.productId).sort().join(",")}`, pending = Object.values(savedBusiness(user.userId)).find((entry) => entry.flow === flow),
    [entry, setEntry] = useState(pending || null), [result, setResult] = useState(null), [quote, setQuote] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false),
    [method, setMethod] = useState(channel === "OFFICIAL_STORE" ? "MAILBOX" : listing?.categoryCode === "CONSTRUCTION" ? "WORKSITE" : listing?.deliveryMethods?.[0] || "PICKUP"),
    [location, setLocation] = useState(listing?.pickupLocation || ""), [project, setProject] = useState("");
  const submitting = useRef(false), request = useRef(null), quoteRequest = useRef(null);
  const getQuote = async (event) => {
    event.preventDefault(); setBusy(true); setError("");
    try { const body = { channel, items, delivery: { method, ...(method !== "MAILBOX" ? { location } : {}), ...(method === "WORKSITE" ? { projectName: project } : {}) } }; const fingerprint = JSON.stringify(body); if (quoteRequest.current?.fingerprint !== fingerprint) quoteRequest.current = { fingerprint, key: id() }; const r = await client.request("/api/v1/checkout/quotes", { method: "POST", body, idempotencyKey: quoteRequest.current.key }); if (!r.data.quoteId || !r.data.version || r.data.totalAmount === undefined) throw new Error("报价结果不完整，请重试。"); setQuote(r.data); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const pay = async () => {
    if (submitting.current) return; submitting.current = true; setBusy(true); setError("");
    if (!request.current) request.current = { userId: user.userId, origin: location.origin, flow, kind: channel === "OFFICIAL_STORE" ? "STORE_PURCHASE" : "MARKET_PURCHASE", path: `/api/v1/${channel === "OFFICIAL_STORE" ? "store" : "market"}/orders`, body: { clientRequestId: id(), quoteId: quote.quoteId, expectedQuoteVersion: quote.version } };
    const current = request.current;
    try { saveBusiness(user.userId, current); const r = await client.request(current.path, { method: "POST", body: current.body }); setResult(r.data); setEntry(current); }
    catch (e) {
      if (e.status >= 400 && e.status < 500 && ![408, 429].includes(e.status)) { clearBusiness(user.userId, current.body.clientRequestId); setError(e.message); }
      else { setError(e.message); setEntry(current); }
    } finally { submitting.current = false; setBusy(false); }
  };
  if (entry) return <CreationProgress client={client} user={user} entry={entry} initialResult={result} onResource={onResource} onRestart={() => { setEntry(null); setResult(null); setQuote(null); request.current = null; quoteRequest.current = null; }} />;
  if (quote) return <>
    <h2>确认付款</h2><p className="price">{credit(quote.totalAmount)}<small>信用点</small></p>
    {quote.items?.map((item, index) => <p key={index}>{item.title} × {item.quantity}</p>)}
    <p>{method === "MAILBOX" ? "通过游戏内邮箱交付，未领取前可按订单规则申请退款。" : "付款将由平台托管，完成履约后才会结算。"}</p>
    {quote.warnings?.length > 0 && <div className="notice-box">{quote.warnings.map((warning, index) => <p key={index}>{warning}</p>)}</div>}
    {error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button disabled={busy} onClick={pay}>{busy ? "正在提交…" : "确认付款"}</Button><Button secondary disabled={busy} onClick={() => { setQuote(null); request.current = null; quoteRequest.current = null; }}>返回修改</Button></div>
  </>;
  return <form onSubmit={getQuote}>
    {method === "MAILBOX" ? <p className="notice-box">商品将通过游戏内邮箱交付。</p> : <>
      <Field label="交付方式"><select value={method} onChange={(e) => setMethod(e.target.value)}>{(listing?.categoryCode === "CONSTRUCTION" ? ["WORKSITE"] : listing?.deliveryMethods || ["PICKUP"]).map((value) => <option key={value} value={value}>{{ DOOR: "送货上门", PICKUP: "约定自取", WORKSITE: "建筑工程" }[value]}</option>)}</select></Field>
      <Field label={method === "WORKSITE" ? "工程地点" : "交付地点"} required maxLength={120} value={location} onChange={(e) => setLocation(e.target.value)} />{method === "WORKSITE" && <Field label="工程名称" required maxLength={80} value={project} onChange={(e) => setProject(e.target.value)} />}
    </>}
    {error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>{busy ? "正在获取报价…" : "查看最终报价"}</Button>
  </form>;
}

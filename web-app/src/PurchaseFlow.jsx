import React, { useEffect, useRef, useState } from "react";
import { Mail, ShieldCheck, Check, LoaderCircle, CircleAlert } from "lucide-react";
import { Badge, Button, Empty, Field } from "./components.jsx";
import ProductArtwork from "./ProductArtwork.jsx";
import { credit, id } from "./format.js";
import { businessOperation, businessResource, canReplayBusiness, clearBusiness, findBusiness, fundsLabels, fundsPending, paymentConfirmed, replayUnreceivedBusiness, resourceId, saveBusiness, savedBusiness } from "./business.js";

function confirmedPayment(operation, resource) {
  return operation?.status !== "FAILED" && paymentConfirmed(resource?.value) &&
    (!(resource.orders?.length > 1) || (operation?.status === "COMPLETED" && resource.orders.every(paymentConfirmed)));
}

export function CreationProgress({ client, user, entry, initialResult, onResource, onRestart, onConfirmed }) {
  const [operation, setOperation] = useState(() => businessOperation(initialResult)), [resource, setResource] = useState(() => businessResource(initialResult)), [error, setError] = useState(""), [busy, setBusy] = useState(false), [notFound, setNotFound] = useState(false), [expired, setExpired] = useState(false);
  const entryRef = useRef(entry), alive = useRef(true), resolving = useRef(false), finished = useRef(false);
  const apply = (op, result) => {
    if (onConfirmed && confirmedPayment(op, result)) onConfirmed();
    if (!alive.current) return; setOperation(op); setResource(result); setError(""); setNotFound(false);
    const next = { ...entryRef.current, operationId: result?.value?.pendingOperationId || op?.operationId || entryRef.current.operationId, resourceType: result?.type || op?.resourceType || entryRef.current.resourceType, resourceId: resourceId(result?.value) || op?.resourceId || entryRef.current.resourceId };
    entryRef.current = next;
    finished.current = op?.status === "FAILED" || op?.status === "COMPLETED" && Boolean(result?.value) && (result.orders || [result.value]).every((value) => !fundsPending(value));
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
  const value = resource?.value, failed = operation?.status === "FAILED" || value?.fundsStatus === "UNPAID", grouped = resource?.orders?.length > 1, confirmed = confirmedPayment(operation, resource);
  return <div className="purchase-result">
    <div className={`purchase-result-icon ${confirmed ? "success" : failed || expired ? "failure" : "pending"}`}>{confirmed ? <Check size={32} /> : failed || expired ? <CircleAlert size={32} /> : <LoaderCircle size={32} />}</div>
    <h2>{expired ? "结算已过期" : failed ? "这笔付款未完成" : confirmed ? "付款已确认" : "正在核对处理结果"}</h2>
    {value && <><Badge tone={confirmed ? "sage" : "neutral"}>{grouped ? `已拆分为 ${resource.orders.length} 笔店铺订单` : fundsLabels[value.fundsStatus] || value.fundsStatus}</Badge><p className="price">{credit(grouped ? operation.amount : value.amount ?? value.content?.reward)}<small>信用点</small></p>{!grouped && <p>{value.orderNo || value.content?.title}</p>}</>}
    {grouped && <div className="checkout-lines">{resource.orders.map((order) => <div className="checkout-line" key={order.orderId}><div><strong>{order.seller?.displayName}</strong><small>{order.orderNo} · {fundsLabels[order.fundsStatus] || order.fundsStatus}</small></div><span>{credit(order.amount)} 信用点</span>{onResource && <Button secondary onClick={() => onResource({ type: "ORDER", value: order })}>查看订单</Button>}</div>)}</div>}
    {operation?.errorCode === "CHECKOUT_PAYMENT_REVERSED" && <p className="notice-box">合并付款未完成，已预留款项已退回。商品仍保留在购物袋中。</p>}
    {operation?.errorCode === "CHECKOUT_REFUND_REQUIRES_ATTENTION" && <p className="notice-box">合并付款未完成，部分预留款项仍待退款核对，请查看各店订单。</p>}
    {expired ? <p className="notice-box">原报价未能创建订单。请重新查看最新价格，并再次确认后付款。</p> : !confirmed && !failed && <p className="notice-box">请保留当前请求。系统会继续查询同一笔业务，结果未确认前不要重新付款。</p>}
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="button-row">{!expired && <Button secondary disabled={busy} onClick={lookup}>刷新处理结果</Button>}{notFound && <Button disabled={busy} onClick={retryOriginal}>继续处理</Button>}{expired && onRestart && <Button onClick={onRestart}>重新结算</Button>}{resource && onResource && !grouped && <Button onClick={() => onResource(resource)}>查看{resource.type === "ORDER" ? "订单" : "委托"}</Button>}</div>
  </div>;
}

export default function PurchaseFlow({ client, user, items, channel, listing, source = "DIRECT", previewProducts = [], onResource, onCartPaid }) {
  const flow = `${channel}:${items.map((item) => item.productId).sort().join(",")}`, pending = Object.values(savedBusiness(user.userId)).find((entry) => entry.flow === flow),
    [entry, setEntry] = useState(pending || null), [result, setResult] = useState(null), [priced, setPriced] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false),
    [method, setMethod] = useState(channel === "OFFICIAL_STORE" ? "MAILBOX" : listing?.categoryCode === "CONSTRUCTION" ? "WORKSITE" : listing?.deliveryMethods?.[0] || "PICKUP"),
    [deliveryLocation, setDeliveryLocation] = useState(listing?.pickupLocation || ""), [project, setProject] = useState("");
  const [revision, setRevision] = useState(0), [reviewMarket, setReviewMarket] = useState(false);
  const quoteBody = JSON.stringify({ channel, items, source, delivery: { method, ...(method !== "MAILBOX" ? { location: deliveryLocation } : {}), ...(method === "WORKSITE" ? { projectName: project } : {}) } });
  const quote = priced?.body === quoteBody && priced.revision === revision ? priced.value : null;
  const [expiresAt, setExpiresAt] = useState(null), [now, setNow] = useState(Date.now());
  useEffect(() => { if (!quote) return; const timer = setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(timer); }, [quote]);
  const expired = expiresAt !== null && now >= expiresAt;
  const submitting = useRef(false), request = useRef(null), quoteRequest = useRef(null);
  const refreshPrice = () => { request.current = null; quoteRequest.current = null; setRevision(value=>value+1); };
  useEffect(() => {
    if (entry || (method !== "MAILBOX" && !reviewMarket)) return;
    const controller = new AbortController();
    if (quoteRequest.current?.body !== quoteBody) quoteRequest.current = {body:quoteBody, key:id()};
    setBusy(true); setError("");
    client.request("/api/v1/checkout/quotes", {method:"POST", body:JSON.parse(quoteBody), idempotencyKey:quoteRequest.current.key, signal:controller.signal})
      .then(response => {
        if (controller.signal.aborted) return;
        const value = response.data;
        if (!value?.quoteId || !value.version || credit(value.totalAmount) === "—") throw new Error("暂时无法计算实付金额，请重试。");
        const expiration = Date.parse(value.expiresAt), serverNow = Date.parse(response.serverTime);
        setExpiresAt(Number.isFinite(expiration) ? Date.now()+expiration-(Number.isFinite(serverNow)?serverNow:Date.now()) : null);
        setNow(Date.now()); setPriced({body:quoteBody, revision, value});
      })
      .catch(e=>{if (!controller.signal.aborted) setError(e.message);})
      .finally(()=>{if (!controller.signal.aborted) setBusy(false);});
    return () => controller.abort();
  }, [client, quoteBody, revision, entry, method, reviewMarket]);
  const reportedPayment = useRef(false);
  const reportPayment = () => {
    if (source === "CART" && onCartPaid && !reportedPayment.current) {
      reportedPayment.current = true;
      onCartPaid();
    }
  };
  const pay = async () => {
    if (!quote || submitting.current || expired || (expiresAt !== null && Date.now() >= expiresAt)) return; submitting.current = true; setBusy(true); setError("");
    if (request.current?.body.quoteId !== quote.quoteId) request.current = { userId: user.userId, origin: window.location.origin, flow, kind: channel === "OFFICIAL_STORE" ? "STORE_PURCHASE" : "MARKET_PURCHASE", path: `/api/v1/${channel === "OFFICIAL_STORE" ? "store" : "market"}/orders`, body: { clientRequestId: id(), quoteId: quote.quoteId, expectedQuoteVersion: quote.version } };
    const current = request.current;
    try { saveBusiness(user.userId, current); const r = await client.request(current.path, { method: "POST", body: current.body }); if (confirmedPayment(businessOperation(r.data), businessResource(r.data))) reportPayment(); setResult(r.data); setEntry(current); }
    catch (e) {
      if (e.status >= 400 && e.status < 500 && ![408, 429].includes(e.status)) { clearBusiness(user.userId, current.body.clientRequestId); setError(e.message); if (["COUPON_EXPIRED", "COUPON_CHANGED", "COUPON_ALREADY_USED", "STATE_VERSION_CONFLICT", "QUOTE_EXPIRED", "PLAYER_PURCHASE_LIMIT"].includes(e.code)) { setPriced(null); request.current = null; quoteRequest.current = null; } }
      else { setError(e.message); setEntry(current); }
    } finally { submitting.current = false; setBusy(false); }
  };
  if (entry) return <CreationProgress client={client} user={user} entry={entry} initialResult={result} onResource={onResource} onConfirmed={reportPayment} onRestart={() => { setEntry(null); setResult(null); setPriced(null); refreshPrice(); }} />;
  const orderLines = (quote?.items || items).map((item, index) => {
    const product = previewProducts.find((p) => (p.productId || p.listingId) === item.productId) || previewProducts[index];
    const content = product?.content || product;
    const image = product?.images?.[0] || product?.photos?.[0];
    const unitPrice = item.unitPrice ?? product?.effectivePrice ?? content?.price;
    return <div className="checkout-line" key={item.productId || index}><ProductArtwork className="checkout-art" image={image} /><div><strong>{item.title || content?.title || "商品"}</strong><small>{unitPrice !== undefined ? `单价 ${credit(unitPrice)} 信用点` : ""}</small></div><span>× {item.quantity}</span></div>;
  });
  const warnings = (quote?.warnings || []).filter(warning => warning !== "报价不预留库存，付款时将重新校验库存及交付条件。");
  const deliveryCard = method === "MAILBOX" ? <div className="checkout-mailbox"><Mail size={20}/><div><strong>{user.gameId || "当前登录账号"} 的游戏邮箱</strong><p>付款后自动投递，未领取前可按订单规则申请退款。</p></div></div> : <div className="checkout-delivery"><ShieldCheck size={22} /><div><strong>平台担保交易</strong><p>款项由平台托管，完成履约后才会结算。</p></div></div>;
  if (quote) return <div className="checkout-flow checkout-summary">
    <div className="checkout-lines">{orderLines}</div>{deliveryCard}
    {Number(quote.discountTotal) > 0 && <div className="checkout-promotion">{Number(quote.productDiscount) > 0 && <div><span>商品优惠</span><strong>−{credit(quote.productDiscount)}</strong></div>}{quote.coupon && <><div><span>{quote.coupon.name}</span><strong>−{credit(quote.couponDiscount)}</strong></div><span className="coupon-auto"><Check size={14} />已自动选用最优惠的券{quote.coupon.type === "ITEM" ? " · 作用于一件商品" : ""}</span></>}</div>}
    {quote.storeOrders?.length > 1 && <dl className="checkout-facts">{quote.storeOrders.map((store) => <div key={store.storeId}><dt>{store.storeName}{store.coupon && <small> · {store.coupon.name} −{credit(store.couponDiscount)}</small>}</dt><dd>{credit(store.totalAmount)} 信用点</dd></div>)}</dl>}
    <dl className="checkout-facts">{quote.storeName && <div><dt>店铺</dt><dd>{quote.storeName}</dd></div>}{method !== "MAILBOX" && <div><dt>领取账号</dt><dd>{user.gameId || "当前登录账号"}</dd></div>}{method !== "MAILBOX" && <div><dt>交付地点</dt><dd>{deliveryLocation}</dd></div>}{expired && <div><dt>价格已过期</dt><dd>请更新后确认</dd></div>}</dl>
    {warnings.length > 0 && <div className="notice-box">{warnings.map((warning, index) => <p key={index}>{warning}</p>)}</div>}
    {error && <p className="auth-error" role="alert">{error}</p>}<footer className="checkout-footer"><div className="checkout-amount"><span>本次应付</span><strong>{credit(quote.totalAmount)}</strong><small>信用点</small>{Number(quote.discountTotal) > 0 && <del className="sale-original">{credit(quote.originalTotal)}</del>}</div><div className="checkout-actions">{method !== "MAILBOX" && <Button secondary disabled={busy} onClick={() => {setReviewMarket(false);setPriced(null);request.current=null;}}>修改交付</Button>}{expired ? <Button disabled={busy} onClick={refreshPrice}>更新价格</Button> : <Button disabled={busy} onClick={pay}>{busy ? "正在付款…" : "确认付款"}</Button>}</div></footer>
  </div>;
  if (method === "MAILBOX" || reviewMarket) return <div className="checkout-flow">
    <div className="checkout-lines">{orderLines}</div>
    {error ? <><p className="auth-error" role="alert">{error}</p><div className="button-row">{method !== "MAILBOX" && <Button secondary onClick={()=>setReviewMarket(false)}>修改交付</Button>}<Button disabled={busy} onClick={refreshPrice}>重新计算金额</Button></div></> : <p className="muted" role="status">正在计算实付金额…</p>}
  </div>;
  return <form className="checkout-flow" onSubmit={event=>{event.preventDefault();setReviewMarket(true);}}><h3>交付信息</h3><div className="checkout-lines">{orderLines}</div>{deliveryCard}
    {method !== "MAILBOX" && <>
      <Field label="交付方式"><select value={method} disabled={busy} onChange={(e) => setMethod(e.target.value)}>{(listing?.categoryCode === "CONSTRUCTION" ? ["WORKSITE"] : listing?.deliveryMethods || ["PICKUP"]).map((value) => <option key={value} value={value}>{{ DOOR: "送货上门", PICKUP: "约定自取", WORKSITE: "建筑工程" }[value]}</option>)}</select></Field>
      <Field label={method === "WORKSITE" ? "工程地点" : "交付地点"} required disabled={busy} maxLength={120} value={deliveryLocation} onChange={(e) => setDeliveryLocation(e.target.value)} />{method === "WORKSITE" && <Field label="工程名称" required disabled={busy} maxLength={80} value={project} onChange={(e) => setProject(e.target.value)} />}
    </>}
    {error && <p className="auth-error" role="alert">{error}</p>}<Button className="checkout-submit" type="submit" disabled={busy}>下一步</Button>
  </form>;
}

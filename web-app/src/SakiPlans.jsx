import React, { useEffect, useRef, useState } from "react";
import { Check, CheckCircle2, Circle, LoaderCircle } from "lucide-react";
import { Button, Modal, media } from "./components.jsx";
import { credit, id } from "./format.js";
import { savedBusiness, saveBusiness, clearBusiness, findBusiness, replayUnreceivedBusiness } from "./business.js";

export default function SakiPlans({ client, user, plans, account, onUpdated }) {
  const [selectedId, setSelectedId] = useState(null), [confirmation, setConfirmation] = useState(null), [busy, setBusy] = useState(false), [result, setResult] = useState(""), [error, setError] = useState("");
  const [pending, setPending] = useState(() => Object.values(savedBusiness(user.userId)).find((e) => e.kind === "AI_PURCHASE") || null);
  const selected=plans.find((p)=>p.planId===selectedId), setSelected=(p)=>setSelectedId(p.planId);
  const switching=Boolean(account?.expiresAt && selected && account.plan?.planId!==selected.planId);
  const downgrade=switching && plans.findIndex(p=>p.planId===selected.planId)<=plans.findIndex(p=>p.planId===account.plan?.planId);
  const [quoting,setQuoting]=useState(false);
  const submitting = useRef(false), alive = useRef(true), pendingRef = useRef(pending); pendingRef.current = pending;
  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  const persist = (entry) => { saveBusiness(user.userId, entry); pendingRef.current = entry; if (alive.current) setPending(entry); };
  const finish = async (entry, operation, order) => {
    if (!operation?.operationId) throw new Error("正在确认购买进度，请稍后查看。");
    persist({ ...entry, operationId: operation.operationId, resourceId: operation.resourceId, resourceType: "ORDER" });
    if (operation.status === "FAILED") { clearBusiness(user.userId, entry.body.clientRequestId); pendingRef.current = null; if (alive.current) setPending(null); throw new Error(operation.errorCode === "AI_PURCHASE_REVERSED" ? "套餐未能开通，信用点已退回。" : "购买未完成，请查看订单详情。"); }
    if (operation.status !== "COMPLETED" || order?.fundsStatus !== "SETTLED" || order?.orderType !== "AI_SUBSCRIPTION") { if (alive.current) setResult("pending"); return; }
    clearBusiness(user.userId, entry.body.clientRequestId); pendingRef.current = null;
    if (alive.current) { setPending(null); setResult("success"); } await onUpdated();
  };
  const recover = async () => {
    const entry = pendingRef.current; if (!entry || submitting.current) return;
    submitting.current = true; if (alive.current) setError("");
    try { const r = entry.operationId ? await findBusiness(client,entry) : await replayUnreceivedBusiness(client, entry, user.userId, window.location.origin); await finish(entry, r.operation, r.resource?.value); }
    catch (e) {
      if (["AI_PLAN_CHANGED", "AI_PLAN_UNAVAILABLE", "AI_PURCHASE_UNAVAILABLE", "AI_PLAN_ACTIVE", "AI_DOWNGRADE_NOT_ALLOWED", "AI_UPGRADE_UNAVAILABLE", "AI_QUOTE_REQUIRED", "AI_QUOTE_CHANGED", "QUOTE_EXPIRED", "QUOTE_ALREADY_USED", "AMOUNT_LIMIT", "CAPABILITY_UNAVAILABLE"].includes(e.code)) { clearBusiness(user.userId, entry.body.clientRequestId); pendingRef.current = null; if (alive.current) setPending(null); }
      if (alive.current) setError(e.message);
    } finally { submitting.current = false; }
  };
  useEffect(() => { if (!pending) return; const timer = setInterval(recover, 4000); return () => clearInterval(timer); }, [pending?.body?.clientRequestId]);
  const quote = async () => {
    if (!selected || quoting || busy || pendingRef.current) return;
    setQuoting(true); setError(""); setResult("");
    try { const r=await client.request("/api/v1/ai/purchase-quotes",{method:"POST",body:{clientRequestId:id(),planId:selected.planId,expectedPlanVersion:selected.version}}); if(alive.current)setConfirmation(r.data); }
    catch(e){if(alive.current)setError(e.message);}
    finally{if(alive.current)setQuoting(false);}
  };
  const buy = async () => {
    if (!confirmation || submitting.current || pendingRef.current) return; const confirmed=confirmation; if(Date.now()>=Date.parse(confirmed.expiresAt)){setConfirmation(null);setError("价格确认已过期，请重新查看后确认购买。");return;} submitting.current = true; setBusy(true); setError(""); setResult("paying");
    const entry = { kind: "AI_PURCHASE", path: "/api/v1/ai/purchases", userId: user.userId, origin: window.location.origin, resourceType: "ORDER", flow: "saki", body: { clientRequestId: id(), planId: confirmed.plan.planId, expectedPlanVersion: confirmed.plan.version, quoteId: confirmed.quoteId } };
    try { persist(entry); const [response] = await Promise.all([client.request(entry.path, { method: "POST", body: entry.body }), new Promise((resolve) => setTimeout(resolve, 1600))]); await finish(entry, response.data.operation, response.data.order); }
    catch (e) {
      if ([400, 401, 403, 404, 409, 422].includes(e.status) || ["AI_PURCHASE_UNAVAILABLE", "CAPABILITY_UNAVAILABLE"].includes(e.code)) { clearBusiness(user.userId, entry.body.clientRequestId); pendingRef.current = null; if (alive.current) setPending(null); }
      if (alive.current) { setError(e.message); setResult("error"); }
    } finally { submitting.current = false; if (alive.current) setBusy(false); }
  };
  return <div className="saki-plans"><div className="saki-hero"><img src={media("xiaoxiang_avatar.png")} alt="小祥" /><h2>Saki AI</h2><h3>让小祥，陪你多聊一点。</h3><p>选择适合你的额度，探索更多想法。</p></div>
    {account && <div className="saki-current"><strong>当前套餐 · {account.plan?.name}</strong><span>{account.quota?.unlimited ? "管理员 · 不限额度" : `${account.quota?.remaining} / ${account.quota?.limit} 次 · ${account.quota?.windowHours} 小时重置`}</span>{account.expiresAt && <small>有效期至 {new Date(account.expiresAt).toLocaleString("zh-CN")}</small>}</div>}
    <div className="saki-options" role="group" aria-label="选择 Saki 套餐">{plans.filter((p) => p.code !== "free").map((p) => <button type="button" key={p.planId} className={selected?.planId === p.planId ? "selected" : ""} aria-pressed={selected?.planId === p.planId} disabled={busy || Boolean(pending)} onClick={() => setSelected(p)}><div><strong>{p.name}</strong><span>{p.quotaPerWindow} 次 / {p.windowHours} 小时</span></div>{selected?.planId === p.planId ? <CheckCircle2 size={22} /> : <Circle size={22} />}<p>{p.description}</p><b>{credit(p.price)} <small>信用点 / {p.durationDays} 天</small></b>{!p.purchasable && <small>暂未开放购买</small>}</button>)}</div>
    <p className="saki-terms">付款后自动生效，不会自动续费。支持补差价升级，到期时间不变；同套餐续购可延长有效期。不支持降级与退款。</p>
    {result === "success" && <div className="saki-success" role="status"><CheckCircle2 size={24} /><div><strong>套餐已开通</strong><p>现在就可以继续与小祥聊天。</p></div></div>}{error && <p className="auth-error" role="alert">{error}</p>}
    <div className="saki-purchase-footer">{pending ? <Button className="full" secondary disabled={busy} onClick={recover}>查看开通进度</Button> : <Button className="full" disabled={!selected?.purchasable || downgrade || busy || quoting} onClick={quote}>{quoting ? "正在读取价格…" : downgrade ? "不支持降级" : selected?.purchasable ? `${switching ? "升级至" : account?.expiresAt ? "续购" : "购买"} ${selected.name}` : selected ? "暂未开放购买" : "选择一个套餐"}</Button>}</div>
    {confirmation && <Modal title="Saki AI" className="saki-checkout" close={() => {if(!busy)setConfirmation(null);}} dismissOnBackdrop={!busy}>
      {result ? <div className="saki-checkout-result" role="status" aria-live="polite">
        {busy ? <LoaderCircle className="saki-spinner" size={64}/> : result==="success" ? <CheckCircle2 size={64}/> : <Circle size={64}/>}
        <h3>{busy ? "正在确认付款" : result==="success" ? (confirmation.kind==="UPGRADE" ? "套餐已升级" : "套餐已开通") : result==="pending" ? "正在开通套餐" : "付款未完成"}</h3>
        <strong>{credit(confirmation.totalAmount)} <small>信用点</small></strong><p>{busy||result==="success" ? confirmation.plan.name : error||"稍后可在订单中查看进度。"}</p>
        <Button className="full" disabled={busy} onClick={()=>setConfirmation(null)}>{busy ? "请稍候…" : "完成"}</Button>
      </div> : <>
        <div className="saki-purchase-card">
          <div className="saki-purchase-product"><img src={media("xiaoxiang_avatar.png")} alt=""/><div><strong>{confirmation.plan.name}</strong><span>{confirmation.plan.quotaPerWindow} 次 / {confirmation.plan.windowHours} 小时</span><small>{confirmation.kind==="UPGRADE" ? "套餐升级" : confirmation.kind==="RENEWAL" ? "套餐续购" : "Saki AI 服务"}</small></div></div>
          <div><small>{confirmation.kind==="UPGRADE" ? "本次补差价" : "本次付款"}</small><p className="saki-checkout-amount">{credit(confirmation.totalAmount)} <small>信用点</small></p>
            {confirmation.kind!=="UPGRADE" && <p>{confirmation.plan.durationDays} 天使用权益</p>}</div>
          <div>{confirmation.kind==="UPGRADE" ? <><p>有效期至 {new Date(confirmation.entitlementExpiresAt).toLocaleString("zh-CN",{timeZone:"Asia/Shanghai"})}</p><p>升级立即生效，到期时间不变。</p></> : <p>{confirmation.kind==="RENEWAL" ? "付款后延长当前套餐有效期。" : "付款后立即生效，无需领取。"}</p>}<p>不会自动续费，不支持降级与退款。</p></div>
          <div><small>购买账号：{user.gameId || user.displayName}</small></div>
        </div><Button className="full" onClick={buy}>确认购买</Button><p className="saki-checkout-caption">使用信用点余额付款</p>
      </>}
    </Modal>}
  </div>;
}

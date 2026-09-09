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
      if (["AI_PLAN_CHANGED", "AI_PLAN_UNAVAILABLE", "AI_PURCHASE_UNAVAILABLE", "AI_PLAN_ACTIVE", "CAPABILITY_UNAVAILABLE"].includes(e.code)) { clearBusiness(user.userId, entry.body.clientRequestId); pendingRef.current = null; if (alive.current) setPending(null); }
      if (alive.current) setError(e.message);
    } finally { submitting.current = false; }
  };
  useEffect(() => { if (!pending) return; const timer = setInterval(recover, 4000); return () => clearInterval(timer); }, [pending?.body?.clientRequestId]);
  const buy = async () => {
    if (!confirmation || submitting.current || pendingRef.current) return; const confirmed=confirmation; submitting.current = true; setBusy(true); setError(""); setConfirmation(false); setResult("paying");
    const entry = { kind: "AI_PURCHASE", path: "/api/v1/ai/purchases", userId: user.userId, origin: window.location.origin, resourceType: "ORDER", flow: "saki", body: { clientRequestId: id(), planId: confirmed.planId, expectedPlanVersion: confirmed.version } };
    try { persist(entry); const [response] = await Promise.all([client.request(entry.path, { method: "POST", body: entry.body }), new Promise((resolve) => setTimeout(resolve, 1600))]); await finish(entry, response.data.operation, response.data.order); }
    catch (e) {
      if ([400, 401, 403, 404, 409, 422].includes(e.status) || ["AI_PURCHASE_UNAVAILABLE", "CAPABILITY_UNAVAILABLE"].includes(e.code)) { clearBusiness(user.userId, entry.body.clientRequestId); pendingRef.current = null; if (alive.current) setPending(null); }
      if (alive.current) { setError(e.message); setResult("error"); }
    } finally { submitting.current = false; if (alive.current) setBusy(false); }
  };
  return <div className="saki-plans"><div className="saki-hero"><img src={media("xiaoxiang_avatar.png")} alt="小祥" /><h2>Saki AI</h2><h3>让小祥，陪你多聊一点。</h3><p>选择适合你的额度，探索更多想法。</p></div>
    {account && <div className="saki-current"><strong>当前套餐 · {account.plan?.name}</strong><span>{account.quota?.unlimited ? "管理员 · 不限额度" : `${account.quota?.remaining} / ${account.quota?.limit} 次 · ${account.quota?.windowHours} 小时重置`}</span>{account.expiresAt && <small>有效期至 {new Date(account.expiresAt).toLocaleString("zh-CN")}</small>}</div>}
    <div className="saki-options" role="group" aria-label="选择 Saki 套餐">{plans.filter((p) => p.code !== "free").map((p) => <button type="button" key={p.planId} className={selected?.planId === p.planId ? "selected" : ""} aria-pressed={selected?.planId === p.planId} disabled={busy || Boolean(pending)} onClick={() => setSelected(p)}><div><strong>{p.name}</strong><span>{p.quotaPerWindow} 次 / {p.windowHours} 小时</span></div>{selected?.planId === p.planId ? <CheckCircle2 size={22} /> : <Circle size={22} />}<p>{p.description}</p><b>{credit(p.price)} <small>信用点 / {p.durationDays} 天</small></b>{!p.purchasable && <small>暂未开放购买</small>}</button>)}</div>
    <p className="saki-terms">付款后自动开通，无需领取。不支持退款，不会自动续费。同套餐续购可延长有效期。</p>
    {result === "success" && <div className="saki-success" role="status"><CheckCircle2 size={24} /><div><strong>套餐已开通</strong><p>现在就可以继续与小祥聊天。</p></div></div>}{error && <p className="auth-error" role="alert">{error}</p>}
    <div className="saki-purchase-footer">{pending ? <Button full secondary disabled={busy} onClick={recover}>查看开通进度</Button> : <Button full disabled={!selected?.purchasable || switching || busy} onClick={() => setConfirmation({...selected})}>{switching ? "当前套餐到期后可更换" : selected?.purchasable ? `购买 ${selected.name}` : selected ? "暂未开放购买" : "选择一个套餐"}</Button>}</div>
    {confirmation && <Modal title="确认付款" close={() => setConfirmation(false)}><div className="saki-confirm"><strong>{credit(confirmation.price)}</strong><span>信用点</span></div><div className="button-row"><Button secondary onClick={() => setConfirmation(false)}>取消</Button><Button onClick={buy}>确认付款</Button></div></Modal>}
    {busy && <div className="saki-payment" role="status" aria-live="polite"><div><LoaderCircle size={64} /><h3>正在确认付款</h3><p>Saki AI · {selected?.name}</p></div></div>}
  </div>;
}

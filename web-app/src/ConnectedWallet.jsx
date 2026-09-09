import React, { useEffect, useRef, useState } from "react";
import { ArrowUpRight, RefreshCw, WalletCards, Check, Search } from "lucide-react";
import { PageHead, SectionHead, Button, Empty, Badge, Modal, Field, Avatar } from "./components.jsx";
import { credit, transferAmount, id } from "./format.js";

const statusLabel = { success: "已完成", failed: "失败", processing: "处理中", unknown: "待核对" };
const journalKey = (user) => `deuterium-transfer:${user}`;
function readJournal(user) {
  try {
    const value = JSON.parse(sessionStorage.getItem(journalKey(user)));
    return value?.request?.clientRequestId && value.request.recipientPlayerRef ? value : null;
  } catch { return null; }
}

export default function ConnectedWallet({ client, user }) {
  const [balance, setBalance] = useState(null), [records, setRecords] = useState([]),
    [cursor, setCursor] = useState(null), [error, setError] = useState(""), [recordError, setRecordError] = useState(""),
    [busy, setBusy] = useState(false), [transferOpen, setTransferOpen] = useState(false),
    [journal, setJournal] = useState(() => readJournal(user.userId));
  const generation = useRef(0);
  const load = async (refresh = false) => {
    const current = ++generation.current;
    setBusy(true); setError(""); setRecordError("");
    const [b, r] = await Promise.allSettled([client.balance(refresh), client.records()]);
    if (current !== generation.current) return;
    if (b.status === "fulfilled") setBalance(b.value.data.balance);
    else { setError(b.reason.message); setBalance((old) => old ? { ...old, fresh: false } : null); }
    if (r.status === "fulfilled") { setRecords(r.value.data.records); setCursor(r.value.page?.nextCursor); }
    else setRecordError(r.reason.message);
    setBusy(false);
  };
  useEffect(() => { load(); return () => { generation.current++; }; }, [user.userId]);
  const saveJournal = (value) => {
    if (value) sessionStorage.setItem(journalKey(user.userId), JSON.stringify(value));
    else sessionStorage.removeItem(journalKey(user.userId));
    setJournal(value);
  };
  const more = async () => {
    setBusy(true); setRecordError("");
    try {
      const r = await client.records(cursor);
      setRecords((old) => [...new Map([...old, ...r.data.records].map((x) => [x.recordId, x])).values()]);
      setCursor(r.page?.nextCursor);
    } catch (e) { setRecordError(e.message); } finally { setBusy(false); }
  };
  return <>
    <PageHead eyebrow="YOUR WALLET" title={<>每一份积累，<span className="muted-heading">都有迹可循。</span></>} subtitle="查看信用点和每一笔往来。">
      <Button secondary disabled={busy} onClick={() => load(true)}><RefreshCw size={16} />刷新余额</Button>
    </PageHead>
    <div className="balance-card" style={{ maxWidth: 650 }}>
      <div className="balance-top"><span><WalletCards size={20} />可用信用点</span><span className="wallet-mark">D</span></div>
      <div className="balance-number">{credit(balance?.amount)}</div>
      <div className="balance-bottom"><span>DEUTERIUM CREDIT<small>{balance?.refreshedAt ? `${balance.fresh ? "更新于" : "上次已知余额"} ${new Date(balance.refreshedAt).toLocaleString("zh-CN")}` : error ? "余额暂不可用" : "正在获取服务器余额"}</small></span>
        <Button disabled={!balance || busy || Boolean(error)} onClick={() => setTransferOpen(true)}>转账<ArrowUpRight size={17} /></Button></div>
    </div>
    {error && <div className="notice-box" role="alert">{error}<Button secondary onClick={() => load(true)} disabled={busy}>重试</Button></div>}
    {journal && <div className="notice-box" role="status">有一笔转账等待核对。<Button secondary onClick={() => setTransferOpen(true)}>查看转账</Button></div>}
    <SectionHead title="账单明细" description="以服务器确认的交易记录为准。" />
    {recordError && <div className="notice-box" role="alert">{recordError}<Button secondary onClick={() => load()} disabled={busy}>重新读取账单</Button></div>}
    {records.length ? <div className="panel table-wrap"><table><thead><tr><th>交易</th><th>金额</th><th>状态</th><th>时间</th></tr></thead><tbody>
      {records.map((r) => <tr key={r.recordId}><td><strong>{r.otherPlayer?.gameId || r.note || "信用点交易"}</strong>{r.note && <small className="muted" style={{ display: "block" }}>{r.note}</small>}</td><td>{r.direction === "income" ? "+" : "−"}{credit(r.amount)}</td><td><Badge tone={r.status === "success" ? "sage" : "neutral"}>{statusLabel[r.status] || r.status}</Badge></td><td>{new Date(r.occurredAt).toLocaleString("zh-CN")}</td></tr>)}
    </tbody></table></div> : !busy && !recordError && <Empty title="还没有账单" text="完成交易后，可以在这里查看记录。" />}
    {busy && <p className="muted" role="status">正在读取…</p>}
    {cursor && <Button secondary disabled={busy} onClick={more}>加载更多账单</Button>}
    {transferOpen && <Modal title="信用点转账" close={() => setTransferOpen(false)}>
      <TransferForm client={client} user={user} journal={journal} saveJournal={saveJournal} onSuccess={() => load(true)} />
    </Modal>}
  </>;
}

function TransferForm({ client, user, journal, saveJournal, onSuccess }) {
  const [query, setQuery] = useState(""), [candidates, setCandidates] = useState([]), [searched, setSearched] = useState(false),
    [recipient, setRecipient] = useState(journal?.recipient || null), [amount, setAmount] = useState(""), [note, setNote] = useState(""),
    [confirmation, setConfirmation] = useState(null), [result, setResult] = useState(journal?.result || null),
    [error, setError] = useState(""), [busy, setBusy] = useState(false);
  const submitting = useRef(false);
  const search = async (event) => {
    event.preventDefault(); if (busy || !query.trim()) return;
    setBusy(true); setError(""); setCandidates([]); setSearched(false);
    try { const r = await client.recipients(query.trim()); setCandidates(r.data.candidates.filter((p) => p.playerRef !== user.playerRef)); setSearched(true); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const prepare = (event) => {
    event.preventDefault(); setError("");
    try {
      if (!recipient) throw new Error("请先选择收款玩家。");
      if (recipient.expiresAt && Date.parse(recipient.expiresAt) < Date.now()) throw new Error("收款玩家确认已过期，请重新搜索。");
      setConfirmation({ clientRequestId: id(), recipientPlayerRef: recipient.playerRef, amount: transferAmount(amount.trim()), note: note.trim() });
    } catch (e) { setError(e.message); }
  };
  const submit = async () => {
    if (submitting.current) return;
    submitting.current = true; setBusy(true); setError("");
    const entry = journal || { request: confirmation, recipient };
    if (!entry.request) { submitting.current = false; setBusy(false); return; }
    // Persist before the network call; a lost response must reuse this exact operation.
    try {
      saveJournal(entry);
      const r = entry.result?.transferId ? await client.transferResult(entry.result.transferId) : await client.transfer(entry.request);
      const transfer = r.data?.transfer;
      if (!transfer || !["success", "failed", "unknown", "processing"].includes(transfer.status)) throw new Error("转账结果尚未确认，请稍后核对。");
      setResult(transfer); setConfirmation(null);
      if (transfer.status === "success") { saveJournal(null); onSuccess(); }
      else saveJournal({ ...entry, result: transfer });
    } catch (e) {
      setError(e.message);
      if (["INVALID_REQUEST", "AMOUNT_INVALID", "BALANCE_INSUFFICIENT", "RECIPIENT_NOT_FOUND", "RECIPIENT_IDENTITY_UNCONFIRMED", "TRANSFER_FAILED", "PLUGIN_BRIDGE_UNAVAILABLE"].includes(e.code)) {
        const failed = { status: "failed" };
        setResult(failed); saveJournal({ ...entry, result: failed });
      }
    } finally { submitting.current = false; setBusy(false); }
  };
  const request = journal?.request || confirmation;
  if (result?.status === "success") return <div className="empty"><Check size={38} /><h2>转账成功</h2><p>{credit(result.amount)} 信用点已转给 {result.recipient?.gameId || recipient?.gameId}。</p><small className="mono">{result.transferId}</small></div>;
  if (request) return <>
    <div className="player-detail"><Avatar user={recipient || journal?.recipient} size="large" /><h2>{recipient?.gameId || journal?.recipient?.gameId}</h2><strong className="price">{credit(request.amount)}<small>信用点</small></strong></div>
    {request.note && <p className="notice-box">备注：{request.note}</p>}
    <p>{journal ? (result?.status === "failed" ? "服务器报告转账失败。" : "转账结果等待核对，请使用下方按钮查询同一笔交易。") : "请核对收款人和金额，确认后提交转账。"}</p>
    {error && <div className="auth-error" role="alert">{error}</div>}
    <div className="button-row"><Button disabled={busy} onClick={submit}>{busy ? "正在核对…" : journal ? "核对转账结果" : "确认转账"}</Button>
      {!journal && <Button secondary onClick={() => setConfirmation(null)}>返回修改</Button>}
      {result?.status === "failed" && <Button secondary disabled={busy} onClick={() => { saveJournal(null); setResult(null); setConfirmation(null); }}>返回</Button>}
    </div>
  </>;
  return <>
    <form className="recipient-search" onSubmit={search}><Field label="收款玩家" value={query} onChange={(e) => { setQuery(e.target.value); setRecipient(null); }} placeholder="输入游戏 ID 或 QQ" required maxLength={64} /><Button secondary type="submit" disabled={busy}><Search size={16} />查找玩家</Button></form>
    {searched && !candidates.length && <p className="muted">没有找到可转账的玩家。</p>}
    <div className="button-row">{candidates.map((p) => <Button key={p.playerRef} secondary={recipient?.playerRef !== p.playerRef} onClick={() => setRecipient(p)}><Avatar user={p} />{p.gameId}</Button>)}</div>
    <form onSubmit={prepare}><Field label="金额（信用点）" value={amount} onChange={(e) => setAmount(e.target.value)} inputMode="decimal" placeholder="0.00" required maxLength={10} /><Field label="备注" value={note} onChange={(e) => setNote(e.target.value)} maxLength={80} placeholder="选填" />
      {error && <div className="auth-error" role="alert">{error}</div>}<Button type="submit" disabled={!recipient || busy}>下一步<ArrowUpRight size={16} /></Button>
    </form>
  </>;
}

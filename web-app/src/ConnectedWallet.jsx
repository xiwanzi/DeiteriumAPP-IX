import { usePlayerNameState } from "./use-player-names.js";
import React, { useEffect, useRef, useState } from "react";
import { ArrowDownLeft, ArrowUpRight, ChevronRight, Clock3, RefreshCw, WalletCards, Check, Search } from "lucide-react";
import { Button, Modal, Field, Avatar } from "./components.jsx";
import { credit, transferAmount, id } from "./format.js";

import WalletLedger, { emptyFilters } from "./WalletLedger.jsx";
import { walletDay, walletTime } from "./wallet-ledger.js";
import "./wallet.css";

const journalKey = (user) => `deuterium-transfer:${user}`;
function readJournal(user) {
  try {
    const value = JSON.parse(sessionStorage.getItem(journalKey(user)));
    return value?.request?.clientRequestId && value.request.recipientPlayerRef ? value : null;
  } catch { return null; }
}

export default function ConnectedWallet({ client, user }) {
  const [balance, setBalance] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(true);
  const [filters, setFilters] = useState(emptyFilters), [revision, setRevision] = useState(0), [transferOpen, setTransferOpen] = useState(false);
  const [journal, setJournal] = usePlayerNameState(client, () => readJournal(user.userId));
  const [sending, setSending] = useState(false);
  const mounted = useRef(true), generation = useRef(0), ledger = useRef(null), transferLock = useRef(false);
  const loadBalance = async (refresh = false) => {
    const current = ++generation.current;
    setBusy(true); setError("");
    try {
      const response = await client.balance(refresh), value = response.data?.balance;
      if (current !== generation.current) return;
      if (credit(value?.availableAmount ?? value?.amount) === "—") throw new Error("可用余额暂不可用，请刷新后重试。");
      const serverTime = Date.parse(value.serverNow || response.serverTime);
      setBalance({...value, updatedAt:value.refreshedAt || value.serverNow || response.serverTime,
        clockOffset:Number.isFinite(serverTime) ? serverTime-Date.now() : 0});
    } catch (e) {
      if (current === generation.current) { setError(e.message); setBalance(old=>old ? {...old,fresh:false} : old); }
    } finally {
      if (current === generation.current) setBusy(false);
    }
  };
  useEffect(() => {
    mounted.current=true; loadBalance();
    return () => { mounted.current=false; generation.current++; };
  }, [client, user.userId]);
  const refreshWallet = () => { if (!mounted.current) return; loadBalance(true); setRevision(value=>value+1); };
  const saveJournal = (value, expectedRequestId) => {
    const stored = readJournal(user.userId);
    if ((stored?.request.clientRequestId ?? null) !== expectedRequestId) {
      setJournal(stored);
      return false;
    }
    if (value) sessionStorage.setItem(journalKey(user.userId), JSON.stringify(value));
    else sessionStorage.removeItem(journalKey(user.userId));
    setJournal(value);
    return true;
  };
  const today = walletDay(Date.now() + (balance?.clockOffset || 0));
  const statsDay = balance?.today?.date || walletDay(balance?.updatedAt) || today;
  const available = credit(balance?.availableAmount ?? balance?.amount);
  const updated = balance?.updatedAt;
  const inspectDay = direction => {
    setFilters({direction, kind:"", day:statsDay});
    ledger.current?.querySelector("h2")?.focus({preventScroll:true});
    ledger.current?.scrollIntoView({block:"start"});
  };
  return <div className="wallet-workspace">
    <header className="wallet-heading"><h1>钱包</h1><Button secondary disabled={busy} onClick={refreshWallet}><RefreshCw size={16} className={busy?"spin":undefined}/>刷新余额</Button></header>
    <div className="wallet-overview">
      <section className="wallet-balance" aria-label="信用点余额" aria-busy={busy}>
        <div className="wallet-account"><span><WalletCards size={20}/>可用余额</span><small>{user.gameId}</small></div>
        <div className="wallet-money"><strong className="balance-number">{available}</strong><span>信用点</span></div>
        {balance?.heldAmount!=null && <div className="wallet-held">托管中 <strong>{credit(balance.heldAmount)}</strong></div>}
        <div className="wallet-balance-actions"><small>{busy ? "正在更新余额…" : error && !balance ? "余额暂不可用" : balance?.fresh===false ? "上次已知余额" : "余额已更新"}{!busy && updated && walletDay(updated) && <time dateTime={updated}> · {walletDay(updated)!==today && walletDay(updated)+" "}{walletTime(updated)}</time>}</small>
          <Button disabled={!balance || busy || Boolean(error) || balance.fresh===false || !!journal} onClick={()=>setTransferOpen(true)}>转账<ArrowUpRight size={17}/></Button></div>
      </section>
      <section className="wallet-today" aria-label="日收支"><header><h2>{balance?.fresh===false?"上次收支":statsDay===today?"今日收支":"当日收支"}</h2><small>{statsDay} · 北京时间</small></header>
        <div className="wallet-today-rows">{[["income","收入",ArrowDownLeft],["expense","支出",ArrowUpRight]].map(([direction,label,Icon])=>{
          const amount=credit(balance?.today?.[direction]);
          return <button type="button" key={direction} aria-label={`查看当日${label}账单`} disabled={amount==="—"} onClick={()=>inspectDay(direction)}>
            <span className={`wallet-stat-icon ${direction}`}><Icon size={18}/></span><span>{label}</span><strong>{amount}</strong><ChevronRight size={15}/>
          </button>;
        })}</div>
      </section>
    </div>
    {error && <div className="wallet-notice" role="alert"><span>{error}</span><button type="button" onClick={refreshWallet} disabled={busy}>重试</button></div>}
    {journal && <div className="wallet-notice" role="status"><Clock3 size={18}/><span>{journal.result?.status==="failed" ? "上一笔转账未完成" : "有一笔转账等待核对"} · {credit(journal.request.amount)} 信用点</span><button type="button" disabled={sending} onClick={()=>setTransferOpen(true)}>查看转账</button></div>}
    <WalletLedger client={client} filters={filters} setFilters={setFilters} revision={revision} today={today} sectionRef={ledger}/>
    {transferOpen && <Modal title="信用点转账" close={()=>setTransferOpen(false)} className="wallet-modal">
      <TransferForm client={client} user={user} journal={journal} saveJournal={saveJournal} onSuccess={refreshWallet} onDone={()=>setTransferOpen(false)} submitting={transferLock} setSending={setSending} available={balance?.fresh===false?"—":available}/>
    </Modal>}
  </div>;
}

function TransferForm({ client, user, journal, saveJournal, onSuccess, onDone, submitting, setSending, available }) {
  const [query, setQuery] = useState(""), [candidates, setCandidates] = usePlayerNameState(client, []), [searched, setSearched] = useState(false),
    [recipient, setRecipient] = usePlayerNameState(client, journal?.recipient || null), [amount, setAmount] = useState(journal?.request?.amount || ""), [note, setNote] = useState(journal?.request?.note || ""),
    [confirmation, setConfirmation] = usePlayerNameState(client, null), [result, setResult] = usePlayerNameState(client, journal?.result || null),
    [error, setError] = useState(""), [busy, setBusy] = useState(false);
  const searchVersion = useRef(0), searchInput = useRef(null), amountInput = useRef(null);
  const changeRecipient = (message = "") => {
    setConfirmation(null); setRecipient(null); setCandidates([]); setSearched(false); setError(message);
    requestAnimationFrame(()=>searchInput.current?.focus());
  };
  const search = async (event) => {
    event.preventDefault(); if (busy || !query.trim()) return;
    const version = ++searchVersion.current;
    setBusy(true); setError(""); setCandidates([]); setSearched(false);
    try { const r = await client.recipients(query.trim()); if (version !== searchVersion.current) return; setCandidates(r.data.candidates.filter((p) => p.playerRef !== user.playerRef)); setSearched(true); }
    catch (e) { if (version === searchVersion.current) setError(e.message); } finally { if (version === searchVersion.current) setBusy(false); }
  };
  const prepare = (event) => {
    event.preventDefault(); setError("");
    try {
      if (!recipient) throw new Error("请先选择收款玩家。");
      if (recipient.expiresAt && Date.parse(recipient.expiresAt) <= Date.now()) { changeRecipient("收款玩家确认已过期，请重新查找。"); return; }
      setConfirmation({ clientRequestId: id(), recipientPlayerRef: recipient.playerRef, amount: transferAmount(amount.trim()), note: note.trim() });
    } catch (e) { setError(e.message); }
  };
  const submit = async () => {
    if (submitting.current) return;
    if (!journal && recipient?.expiresAt && Date.parse(recipient.expiresAt) <= Date.now()) {
      changeRecipient("收款玩家确认已过期，请重新查找。"); return;
    }
    const entry = journal || { request: confirmation, recipient };
    if (!entry.request) return;
    submitting.current = true; setSending(true); setBusy(true); setError("");
    // Persist before the network call; a lost response must reuse this exact operation.
    try {
      if (!saveJournal(entry, journal?.request.clientRequestId ?? null)) {
        setConfirmation(null); setError("转账进度已更新，请查看最新结果。"); return;
      }
      const r = entry.result?.transferId ? await client.transferResult(entry.result.transferId) : await client.transfer(entry.request);
      const transfer = r.data?.transfer;
      if (!transfer || !["success", "failed", "unknown", "processing"].includes(transfer.status)) throw new Error("转账结果尚未确认，请稍后核对。");
      setResult(transfer); setConfirmation(null);
      if (transfer.status === "success") { saveJournal(null, entry.request.clientRequestId); onSuccess(); }
      else saveJournal({ ...entry, result: transfer }, entry.request.clientRequestId);
    } catch (e) {
      setError(e.message);
      if (!entry.result?.transferId && ["RECIPIENT_INVALID", "INVALID_REQUEST", "AMOUNT_INVALID", "BALANCE_INSUFFICIENT", "RECIPIENT_NOT_FOUND", "RECIPIENT_IDENTITY_UNCONFIRMED", "TRANSFER_FAILED", "PLUGIN_BRIDGE_UNAVAILABLE"].includes(e.code)) {
        const failed = { status: "failed" };
        setResult(failed); saveJournal({ ...entry, result: failed }, entry.request.clientRequestId);
      }
    } finally { submitting.current = false; setSending(false); setBusy(false); }
  };
  const request = journal?.request || confirmation;
  const payee = journal?.recipient || recipient;
  const failed = result?.status === "failed";
  if (result?.status === "success") return <div className="wallet-transfer-success"><span><Check size={28}/></span><h2>转账成功</h2><strong>{credit(result.amount)} <small>信用点</small></strong>
    <p>已转给 {result.recipient?.gameId || payee?.gameId}。</p>{result.transferId && <code>{result.transferId}</code>}<Button onClick={onDone}>完成</Button></div>;
  if (request) return <div className="wallet-transfer-review">
    <div className="wallet-recipient"><Avatar user={payee}/><div><small>收款玩家</small><strong>{payee?.gameId || "玩家"}</strong></div></div>
    <div className="ledger-detail-amount"><span>转账金额</span><strong>{credit(request.amount)}</strong><small>信用点</small></div>
    {request.note && <dl className="wallet-facts"><div><dt>备注</dt><dd>{request.note}</dd></div></dl>}
    {journal && <p className="wallet-transfer-status" role="status">{failed ? "这笔转账未完成。" : "正在等待确认，可以核对这笔转账的结果。"}</p>}
    {error && <div className="auth-error" role="alert">{error}</div>}
    <div className="wallet-form-actions">{!journal && <Button secondary disabled={busy} onClick={()=>{setConfirmation(null);requestAnimationFrame(()=>amountInput.current?.focus());}}>返回修改</Button>}
      {failed ? <Button secondary disabled={busy} onClick={()=>{saveJournal(null, request.clientRequestId);setResult(null);setConfirmation(null);setError("");}}>返回</Button> :
      <Button disabled={busy} onClick={submit}>{busy ? "正在处理…" : journal ? "核对转账结果" : "确认转账"}</Button>}</div>
  </div>;
  return <div className="wallet-transfer-form">
    {recipient ? <div className="wallet-recipient"><Avatar user={recipient}/><div><small>收款玩家</small><strong>{recipient.gameId}</strong></div>
      <button type="button" onClick={()=>changeRecipient()}>更换</button></div> : <>
      <form className="wallet-payee-search" onSubmit={search}><Field label="收款玩家"><input ref={searchInput} value={query}
        onChange={event=>{searchVersion.current++;setQuery(event.target.value);setCandidates([]);setSearched(false);setBusy(false);setError("");}}
        placeholder="游戏 ID 或 QQ" required maxLength={64} autoComplete="off"/></Field>
        <Button secondary type="submit" disabled={busy || !query.trim()}><Search size={16}/>查找玩家</Button></form>
      {searched && !candidates.length && <p className="wallet-state">没有找到可转账的玩家。</p>}
      {candidates.length>0 && <ul className="wallet-payees">{candidates.map(player=><li key={player.playerRef}>
        <button type="button" aria-label={player.gameId} onClick={()=>{setRecipient(player);setError("");amountInput.current?.focus();}}><Avatar user={player}/><strong>{player.gameId}</strong><ChevronRight size={16}/></button></li>)}</ul>}
    </>}
    <form onSubmit={prepare}>
      <Field label="金额（信用点）" hint={available!=="—" ? `可用余额 ${available} 信用点` : undefined}>
        <input ref={amountInput} value={amount} onChange={e=>setAmount(e.target.value)} inputMode="decimal" placeholder="0.00" required maxLength={10}/>
      </Field>
      <Field label="备注" value={note} onChange={e=>setNote(e.target.value)} maxLength={80} placeholder="选填"/>
      {error && <div className="auth-error" role="alert">{error}</div>}
      <div className="wallet-form-actions"><Button type="submit" disabled={!recipient || !amount.trim() || busy}>下一步<ArrowUpRight size={16}/></Button></div>
    </form>
  </div>;
}

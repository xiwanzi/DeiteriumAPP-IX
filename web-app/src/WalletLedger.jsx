import React, { useEffect, useRef, useState } from "react";
import { ArrowLeftRight, Blocks, BriefcaseBusiness, ChevronRight, CircleDollarSign, ClipboardList, SlidersHorizontal, ShoppingBag, Store } from "lucide-react";
import { Button, Modal } from "./components.jsx";
import { credit } from "./format.js";
import { usePlayerNameState } from "./use-player-names.js";
import { businessLabels, ledgerQuery, recordAmount, recordTitle, statusLabels, walletDay, walletTime } from "./wallet-ledger.js";

const icons = { TRANSFER: ArrowLeftRight, GAME: Blocks, OFFICIAL_STORE: ShoppingBag, MARKET_ORDER: Store, COMMISSION: BriefcaseBusiness };
export const emptyFilters = {direction:"", kind:"", day:""};

function RecordDetail({ client, recordId, close }) {
  const [record, setRecord] = usePlayerNameState(client, null);
  const [error, setError] = useState(""), [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    setError("");
    client.request(`/api/v1/wallet/records/${encodeURIComponent(recordId)}`).then(response => {
      if (response.data?.recordId !== recordId) throw new Error("这笔账单暂时无法读取。");
      if (active) setRecord(response.data);
    }).catch(e => { if (active) setError(e.message); });
    return () => { active = false; };
  }, [client, recordId, attempt]);
  const party = record?.otherPlayer?.displayName || record?.otherPlayer?.gameId || record?.counterparty?.displayName;
  return <Modal title="账单详情" close={close} className="wallet-modal">
    {record ? <>
      <div className="ledger-detail-amount"><span>{recordTitle(record)}</span><strong className={record.direction==="income"?"wallet-income":""}>{recordAmount(record)}</strong><small>信用点</small></div>
      <dl className="wallet-facts">
        <div><dt>交易状态</dt><dd>{statusLabels[record.status?.toLowerCase()] || record.status || "—"}</dd></div>
        <div><dt>交易时间</dt><dd>{walletDay(record.occurredAt)} {walletTime(record.occurredAt)}（北京时间）</dd></div>
        {record.businessType && <div><dt>交易类型</dt><dd>{businessLabels[record.businessType] || record.businessType}</dd></div>}
        {party && <div><dt>交易对方</dt><dd>{party}</dd></div>}
        {record.note && record.note!==record.title && <div><dt>备注</dt><dd>{record.note}</dd></div>}
        {record.beforeBalance!=null && <div><dt>交易前余额</dt><dd>{credit(record.beforeBalance)}</dd></div>}
        {record.afterBalance!=null && <div><dt>交易后余额</dt><dd>{credit(record.afterBalance)}</dd></div>}
        <div><dt>流水号</dt><dd><code>{record.recordId}</code></dd></div>
        {record.businessRef && record.businessRef!==record.recordId && <div><dt>业务编号</dt><dd><code>{record.businessRef}</code></dd></div>}
      </dl>
    </> : error ? <div className="wallet-notice" role="alert"><span>{error}</span><button type="button" onClick={()=>setAttempt(n=>n+1)}>重试</button></div> : <p className="wallet-state" role="status">正在读取这笔账单…</p>}
  </Modal>;
}

function LedgerRows({ client, filters, revision, today, onSelect }) {
  const [records, setRecords] = usePlayerNameState(client, []), [cursor, setCursor] = useState(null);
  const [busy, setBusy] = useState(true), [error, setError] = useState("");
  const generation = useRef(0), loading = useRef(false), retryCursor = useRef(null);
  const read = async (next = null) => {
    if (next && loading.current) return;
    const current = ++generation.current;
    retryCursor.current = next;
    loading.current = true; setBusy(true); setError("");
    try {
      const response = await client.records(next, next ? {} : ledgerQuery(filters, today));
      if (current !== generation.current) return;
      const items = response.data?.records, after = response.page?.nextCursor;
      if (!Array.isArray(items) || (response.page?.hasMore && !after) || (next && next===after)) throw new Error("账单分页数据异常，请重试。");
      setRecords(old => next ? [...new Map([...old,...items].map(item=>[item.recordId,item])).values()] : items);
      setCursor(after || null);
    } catch (e) {
      if (current===generation.current) setError(e.message);
    } finally {
      if (current===generation.current) { loading.current = false; setBusy(false); }
    }
  };
  useEffect(() => {
    read();
    return () => { generation.current++; };
  }, [client, revision, today]);
  const filtered = Boolean(filters.direction || filters.kind || filters.day);
  return <>
    {records.length>0 && <ul className="ledger-list">{records.map((record,index)=>{
      const day = walletDay(record.occurredAt), title = recordTitle(record);
      const Icon = icons[record.businessType] || CircleDollarSign;
      const note = record.note!==record.title && record.note!==title ? record.note : "";
      return <li key={record.recordId}>
        {(index===0 || day!==walletDay(records[index-1].occurredAt)) && <h3 className="ledger-day">{day===today ? "今天 · " : ""}{day || "日期未知"}</h3>}
        <button type="button" className="ledger-row" onClick={()=>onSelect(record.recordId)}>
          <span className={`ledger-icon ledger-${record.businessType || "other"}`}><Icon size={19}/></span>
          <span className="ledger-description"><strong>{title}</strong><small><time dateTime={record.occurredAt}>{walletTime(record.occurredAt)}</time>{note && <span>{note}</span>}</small></span>
          <span className="ledger-numbers"><strong className={record.direction==="income"?"wallet-income":""}>{recordAmount(record)}</strong><small>{statusLabels[record.status?.toLowerCase()] || record.status}</small></span>
          <ChevronRight size={15} className="ledger-open"/>
        </button>
      </li>;
    })}</ul>}
    {!busy && !error && !records.length && <div className="wallet-empty"><ClipboardList size={24}/><h3>{filtered ? "没有符合条件的账单" : "还没有账单"}</h3><p>{filtered ? "可以调整日期或收支类型。" : "完成交易后，收支记录会出现在这里。"}</p></div>}
    {error && <div className="wallet-notice" role="alert"><span>{error}</span><button type="button" disabled={busy} onClick={()=>read(retryCursor.current)}>重试读取</button></div>}
    {busy && <p className="wallet-state" role="status">正在读取账单…</p>}
    {cursor && !busy && !error && <Button className="ledger-more" secondary onClick={()=>read(cursor)}>加载更多账单</Button>}
  </>;
}

export default function WalletLedger({ client, filters, setFilters, revision, today, sectionRef }) {
  const [selectedId, setSelectedId] = useState(null), [advanced, setAdvanced] = useState(false);
  const dateInput = useRef(null);
  const filtered = Boolean(filters.direction || filters.kind || filters.day);
  useEffect(() => {
    const focus = () => { setAdvanced(true); requestAnimationFrame(()=>dateInput.current?.focus()); };
    window.addEventListener("deuterium:search",focus);
    return () => window.removeEventListener("deuterium:search",focus);
  }, []);
  return <section className="wallet-ledger" aria-labelledby="wallet-bills-heading" ref={sectionRef}>
    <header><h2 id="wallet-bills-heading" tabIndex={-1}>账单明细</h2>
      {filtered && <button type="button" className="ledger-reset" onClick={()=>setFilters(emptyFilters)}>重置筛选</button>}</header>
    <div className="ledger-filters">
      <div className="ledger-directions" role="group" aria-label="收支方向">
        {[["","全部"],["income","收入"],["expense","支出"]].map(([value,label])=><button key={value} type="button" aria-pressed={filters.direction===value} onClick={()=>setFilters(old=>({...old,direction:value}))}>{label}</button>)}
      </div>
      <button type="button" className="ledger-filter-toggle" aria-label="日期与类型筛选" aria-controls="wallet-extra-filters" aria-expanded={advanced} onClick={()=>setAdvanced(value=>!value)}>
        <SlidersHorizontal size={15}/>筛选{(filters.day||filters.kind) && <b>{Number(Boolean(filters.day))+Number(Boolean(filters.kind))}</b>}
      </button>
      <div className={`ledger-extra ${advanced?"is-open":""}`} id="wallet-extra-filters">
      <label className="ledger-date"><span>日期</span><input ref={dateInput} type="date" aria-label="账单日期" value={filters.day} max={today} onChange={event=>{
        const day=event.target.value;
        if (!day || (event.target.validity.valid && /^\d{4}-\d{2}-\d{2}$/.test(day))) setFilters(old=>({...old,day}));
      }}/></label>
      <select aria-label="交易类型" value={filters.kind} onChange={event=>setFilters({...filters,kind:event.target.value})}><option value="">全部类型</option>{Object.entries(businessLabels).map(([value,label])=><option key={value} value={value}>{label}</option>)}</select>
      </div>
    </div>
    <LedgerRows key={JSON.stringify([filters.direction,filters.kind,filters.day])} client={client} filters={filters} revision={revision} today={today} onSelect={setSelectedId}/>
    {selectedId && <RecordDetail client={client} recordId={selectedId} close={()=>setSelectedId(null)}/>}
  </section>;
}

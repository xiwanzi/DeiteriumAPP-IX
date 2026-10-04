import React, { useEffect, useRef, useState } from "react";
import { ArrowLeft, ArrowDown, Check, ChevronLeft, ChevronRight, Inbox, LockKeyhole, MessageSquare, PanelRightClose, PanelRightOpen, Plus, RefreshCw, Search, Send, UserRound, X, Zap } from "lucide-react";
import { Avatar, Button, Field, Modal } from "./components.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";
import "./mail-support.css";
import "./support-workspace.css";

const states = { OPEN: "待处理", IN_PROGRESS: "处理中", WAITING_CUSTOMER: "等玩家回复", RESOLVED: "已解决", CLOSED: "已关闭" };
const priorities = { LOW: "低", NORMAL: "普通", HIGH: "高", URGENT: "紧急" };
const categories = { ACCOUNT: "账号与验证", GAME: "游戏问题", ORDER: "交易与订单", ACCESS: "通行与审核", OTHER: "其他问题" };
const sendLabels = { WAITING_CUSTOMER: "发送并等待回复", IN_PROGRESS: "发送并继续处理", RESOLVED: "发送并解决" };
const deliveryLabels = { PENDING: "等待发送", SENDING: "正在发送", SENT: "邮件服务器已接收", RETRY: "发送失败，将重试", FAILED: "发送失败" };
const time = value => new Date(value).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
const base = "/api/v1/admin/support/tickets";

export default function SupportWorkspace({ client, user }) {
  const [items, setItems] = useState([]);
  const [selected, setSelected] = useState(() => new URLSearchParams(location.search).get("ticket") || "");
  const [detail, setDetail] = useState(null);
  const [status, setStatus] = useState("OPEN"), [queue, setQueue] = useState(""), [category, setCategory] = useState("");
  const [query, setQuery] = useState(""), [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0), [more, setMore] = useState(false);
  const [loading, setLoading] = useState(false), [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState(""), [listError, setListError] = useState(""), [detailError, setDetailError] = useState(""), [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false), [tools, setTools] = useState({ agents: [], macros: [] });
  const [drafts, setDrafts] = useState({}), [visibility, setVisibility] = useState("PUBLIC");
  const [replyStatus, setReplyStatus] = useState("WAITING_CUSTOMER"), [advance, setAdvance] = useState(false);
  const [contextOpen, setContextOpen] = useState(false), [macro, setMacro] = useState(null), [newMessages, setNewMessages] = useState(false);
  const [creating, setCreating] = useState(false);
  const listGeneration = useRef(0), detailGeneration = useRef(0), toolsGeneration = useRef(0);
  const reloadList = useRef(null);
  const selectedRef = useRef(selected), busyRef = useRef(false), pending = useRef(new Map());
  const formRef = useRef(null), messagesRef = useRef(null), nearBottom = useRef(true), viewedMessages = useRef({ ticket: "", last: "" });
  const contextRef = useRef(null), contextToggleRef = useRef(null);
  const draftKey = `${selected}:${visibility}`, draft = drafts[draftKey] || "";
  const position = items.findIndex(ticket => ticket.ticketId === selected);
  const nextTicket = items[position + 1]?.ticketId;
  const ticket = detail?.ticket;
  useUnsavedChanges(drafts, busy);

  const choose = value => {
    if (busyRef.current || value === selectedRef.current) return;
    selectedRef.current = value;
    ++detailGeneration.current;
    setSelected(value); setDetail(null); setDetailError(""); setError(""); setNotice(""); setVisibility("PUBLIC");
    setReplyStatus("WAITING_CUSTOMER"); setContextOpen(false); setNewMessages(false);
    nearBottom.current = true;
    const next = new URL(location.href);
    if (value) next.searchParams.set("ticket", value); else next.searchParams.delete("ticket");
    history.replaceState({}, "", next.pathname + next.search);
  };
  const load = async () => {
    const generation = ++listGeneration.current;
    setLoading(true);
    try {
      const response = await client.request(`${base}?${new URLSearchParams({ status, queue, q: search, category, offset })}`);
      if (generation !== listGeneration.current) return;
      setItems(response.data.items); setMore(response.data.hasMore); setListError("");
      if (!selectedRef.current && innerWidth > 760 && response.data.items.length) choose(response.data.items[0].ticketId);
    } catch (e) { if (generation === listGeneration.current) setListError(e.message); }
    finally { if (generation === listGeneration.current) setLoading(false); }
  };
  reloadList.current = load;
  const loadDetail = async (ticketId = selectedRef.current) => {
    if (!ticketId) return;
    const generation = ++detailGeneration.current;
    setDetailLoading(true);
    try {
      const response = await client.request(`${base}/${encodeURIComponent(ticketId)}`);
      if (generation === detailGeneration.current && ticketId === selectedRef.current) {
        if (response.data.ticket.ticketId !== ticketId) { choose(response.data.ticket.ticketId); return; }
        setDetail(response.data); setDetailError("");
      }
    } catch (e) { if (generation === detailGeneration.current) setDetailError(e.message); }
    finally { if (generation === detailGeneration.current) setDetailLoading(false); }
  };
  const loadTools = async () => {
    const generation = ++toolsGeneration.current;
    try {
      const response = await client.request("/api/v1/admin/support/tools");
      if (generation === toolsGeneration.current) setTools(response.data);
    } catch (e) { if (generation === toolsGeneration.current) setError(e.message); }
  };
  const refresh = () => { load(); loadDetail(); loadTools(); };
  useEffect(() => {
    const timer = setTimeout(() => { setSearch(query.trim()); if (query.trim()) setStatus(""); setOffset(0); }, 250);
    return () => clearTimeout(timer);
  }, [query]);
  useEffect(() => { load(); return () => { ++listGeneration.current; }; }, [status, queue, search, category, offset]);
  useEffect(() => { loadDetail(); return () => { ++detailGeneration.current; }; }, [selected]);
  useEffect(() => { loadTools(); return () => { ++toolsGeneration.current; }; }, []);
  useEffect(() => {
    const timer = setInterval(() => { if (!document.hidden && !busyRef.current) refresh(); }, 15000);
    return () => clearInterval(timer);
  }, [status, queue, search, category, offset]);
  useEffect(() => {
    if (!detail || !messagesRef.current) return;
    const last = detail.messages.at(-1)?.messageId || "";
    const previous = viewedMessages.current;
    if (previous.ticket !== selected || previous.last !== last) {
      if (previous.ticket !== selected || nearBottom.current) {
        messagesRef.current.scrollTop = messagesRef.current.scrollHeight;
        setNewMessages(false);
      } else setNewMessages(true);
    }
    viewedMessages.current = { ticket: selected, last };
  }, [detail, selected]);
  useEffect(() => {
    if (!contextOpen) return;
    // At narrower widths the detail panel is a modal drawer, so keep keyboard focus inside it.
    const drawer = window.matchMedia("(max-width: 1279px)").matches;
    if (drawer) contextRef.current?.querySelector("button")?.focus();
    const keyboard = event => {
      if (event.key === "Escape") { setContextOpen(false); contextToggleRef.current?.focus(); }
      if (drawer && event.key === "Tab") {
        const elements = [...contextRef.current.querySelectorAll("button:not(:disabled), select:not(:disabled), a[href], summary")];
        const first = elements[0], last = elements.at(-1);
        if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
        else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
      }
    };
    window.addEventListener("keydown", keyboard);
    return () => window.removeEventListener("keydown", keyboard);
  }, [contextOpen]);

  const change = async (patch, clearDraft = false) => {
    if (busyRef.current || !detail || detail.ticket.ticketId !== selectedRef.current) return;
    busyRef.current = true; setBusy(true); setError(""); setNotice("");
    // Ignore any read started before this mutation, including background polls.
    ++detailGeneration.current;
    const ticketId = selected, sentDraftKey = draftKey;
    const following = clearDraft && advance && position >= 0 ? nextTicket : "";
    const fingerprint = JSON.stringify([ticketId, patch]);
    if (!pending.current.has(fingerprint)) pending.current.set(fingerprint, {
      expectedVersion: detail.ticket.version, ...patch, clientRequestId: id(),
    });
    let succeeded = false;
    try {
      await client.request(`${base}/${encodeURIComponent(ticketId)}`, { method: "POST", body: pending.current.get(fingerprint) });
      pending.current.delete(fingerprint);
      if (clearDraft) setDrafts(old => { const next = { ...old }; delete next[sentDraftKey]; return next; });
      nearBottom.current = clearDraft || nearBottom.current;
      setNotice(clearDraft ? patch.visibility === "INTERNAL" ? "内部备注已保存" : "回复已加入发送队列" : "处理信息已更新");
      await Promise.all([loadDetail(ticketId), reloadList.current(), loadTools()]);
      succeeded = true;
    } catch (e) {
      setError(e.message);
      if (e.code === "VERSION_CONFLICT" || e.code === "STATE_VERSION_CONFLICT") {
        pending.current.delete(fingerprint);
        await loadDetail(ticketId);
      }
    } finally { busyRef.current = false; setBusy(false); }
    if (succeeded && following) choose(following);
  };
  const retryMail = async event => {
    if (busyRef.current) return;
    busyRef.current = true; setBusy(true); setError("");
    try {
      await client.request(`${base}/${selected}/mail/${event.eventId}/retry`, { method: "POST", body: { clientRequestId: id() } });
      await loadDetail(); setNotice("邮件已重新加入发送队列");
    } catch (e) { setError(e.message); }
    finally { busyRef.current = false; setBusy(false); }
  };
  const updateDraft = value => setDrafts(old => {
    const next = { ...old };
    if (value) next[draftKey] = value; else delete next[draftKey];
    return next;
  });
  const clearFilters = () => { setStatus(""); setQueue(""); setCategory(""); setQuery(""); setSearch(""); setOffset(0); };
  const goLatest = () => {
    if (messagesRef.current) messagesRef.current.scrollTop = messagesRef.current.scrollHeight;
    nearBottom.current = true; setNewMessages(false);
  };
  const mailboxHealthy = tools.mailbox?.enabled && tools.mailbox?.lastPollAt && !tools.mailbox?.lastError;
  const mailboxTitle = tools.mailbox?.lastError || (tools.mailbox?.lastPollAt ? `最近同步 ${time(tools.mailbox.lastPollAt)}` : "等待收件同步");

  return <div className="support-page support-efficient">
    <div className="support-toolbar">
      <h1>工单</h1>
      <div className="support-status-tabs" role="group" aria-label="全站工单状态">
        {[["", "全部"], ...Object.entries(states)].map(([value, label]) => <button key={value} type="button" aria-pressed={status === value} className={status === value ? "active" : ""} onClick={() => { setStatus(value); setOffset(0); }}>
          {label}<span>{value ? tools.counts?.[value] || 0 : Object.entries(tools.counts || {}).reduce((sum, [state, count]) => sum + (state === "CLOSED" ? 0 : count), 0)}</span>
        </button>)}
      </div>
      <span className={`support-mailbox-health ${mailboxHealthy ? "online" : ""}`} title={`${tools.mailbox?.email || "玩家支持邮箱待配置"} · ${mailboxTitle}`}><i />{mailboxHealthy ? "收件正常" : "收件待检查"}</span>
      <button className="icon-button" aria-label="刷新工单" title="刷新工单" disabled={loading || busy} onClick={refresh}><RefreshCw size={17} /></button>
      <Button className="support-create-button" disabled={busy} onClick={() => setCreating(true)}><Plus size={16} />新建工单</Button>
    </div>
    {error && <div className="support-feedback error" role="alert"><span>{error}</span><button onClick={refresh} disabled={busy}>重新读取</button><button aria-label="关闭错误提示" onClick={() => setError("")}><X size={15} /></button></div>}
    {listError && <div className="support-feedback error" role="alert"><span>工单列表未更新：{listError}</span><button onClick={load} disabled={loading}>重试列表</button></div>}
    {notice && <div className="support-feedback" role="status"><Check size={15} /><span>{notice}</span><button aria-label="关闭操作提示" onClick={() => setNotice("")}><X size={15} /></button></div>}
    <div className={`support-workspace ${selected ? "has-selection" : ""} ${contextOpen && detail ? "context-open" : ""}`}>
      <aside className="support-queue" aria-label="工单队列">
        <div className="support-views" role="group" aria-label="工单范围">{[["", "全部人员"], ["mine", "分配给我"], ["unassigned", "未分配"]].map(([value, label]) => <button key={value} aria-pressed={queue === value} className={queue === value ? "active" : ""} onClick={() => { setQueue(value); setOffset(0); }}>{label}</button>)}</div>
        <form className="support-search" onSubmit={event => { event.preventDefault(); setSearch(query.trim()); setOffset(0); }}><Search size={16} /><input aria-label="搜索工单" placeholder="主题、编号、邮箱、游戏 ID" value={query} onChange={event => setQuery(event.target.value)} maxLength={120} />{query && <button type="button" aria-label="清空搜索" onClick={() => { setQuery(""); setSearch(""); setOffset(0); }}><X size={15} /></button>}</form>
        <div className="support-list-filters"><select aria-label="问题类型筛选" value={category} onChange={event => { setCategory(event.target.value); setOffset(0); }}><option value="">全部问题类型</option>{Object.entries(categories).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><button onClick={clearFilters} disabled={!status && !queue && !category && !query}>清空筛选</button></div>
        <div className="support-list-caption"><span>{states[status] || "全部状态"} · 本页 {items.length}{more ? "+" : ""} 件</span><span role="status">{loading ? "更新中…" : ""}</span></div>
        <div className="support-ticket-list" aria-busy={loading}>
          {items.map(item => <button key={item.ticketId} disabled={busy} aria-current={item.ticketId === selected ? "true" : undefined} className={`support-ticket ${item.ticketId === selected ? "selected" : ""}`} onClick={() => choose(item.ticketId)}>
            <div><span className={`support-state ${item.status.toLowerCase()}`}>{states[item.status]}</span><time>{time(item.updatedAt)}</time></div>
            <strong>{item.subject}</strong><p className="support-sender">{item.requesterName || item.requesterEmail}</p>
            <div className="support-ticket-tags"><span className={`support-category ${item.category.toLowerCase()}`}>{categories[item.category]}</span><small>#{item.ticketId}</small></div>
            <div className="support-ticket-flags"><span>{tools.agents.find(agent => agent.userId === item.assigneeId)?.name || (item.assigneeId ? "已分配" : "未分配")}</span>{["HIGH", "URGENT"].includes(item.priority) && <strong>{priorities[item.priority]}优先级</strong>}{Object.keys(drafts).some(key => key.startsWith(`${item.ticketId}:`)) && <span className="support-draft-marker">草稿</span>}</div>
          </button>)}
          {!items.length && !loading && <div className="support-queue-empty"><Inbox size={26} /><p>{search ? "没有匹配的工单" : "当前筛选下没有工单"}</p><button onClick={clearFilters}>查看全部工单</button></div>}
        </div>
        <div className="support-pagination"><button aria-label="上一页工单" disabled={!offset || loading || busy} onClick={() => setOffset(Math.max(0, offset - 50))}><ChevronLeft size={16} /></button><span>第 {offset / 50 + 1} 页</span><button aria-label="下一页工单" disabled={!more || loading || busy} onClick={() => setOffset(offset + 50)}><ChevronRight size={16} /></button></div>
      </aside>
      <section className="support-conversation" aria-label="工单对话">
        {!selected ? <div className="support-welcome"><Inbox size={28} /><h2>选择工单开始处理</h2><p>邮件回复、内部备注和处理状态都在这里。</p></div> : !detail ? <div className="support-loading" role="status">{detailError ? <><p>{detailError}</p><Button secondary disabled={detailLoading} onClick={() => loadDetail()}>重新读取工单</Button></> : "正在读取工单…"}</div> : <>
          <header className="support-conversation-head">
            <div className="support-subject-row"><button className="support-mobile-back icon-button" aria-label="返回工单列表" disabled={busy} onClick={() => choose("")}><ArrowLeft size={20} /></button><div><span className="support-number">#{ticket.ticketId}</span><h2 title={ticket.subject}>{ticket.subject}</h2></div><div className="support-ticket-navigation"><button className="icon-button" aria-label="上一条工单" title="上一条工单" disabled={busy || position <= 0} onClick={() => choose(items[position - 1].ticketId)}><ChevronLeft size={17} /></button><button className="icon-button" aria-label="下一条工单" title="下一条工单" disabled={busy || position < 0 || !nextTicket} onClick={() => choose(nextTicket)}><ChevronRight size={17} /></button><button ref={contextToggleRef} className="icon-button" aria-label={contextOpen ? "收起工单资料" : "展开工单资料"} title="用户资料、分类、优先级与投递记录" aria-expanded={contextOpen} aria-controls="support-context" onClick={() => setContextOpen(!contextOpen)}>{contextOpen ? <PanelRightClose size={18} /> : <PanelRightOpen size={18} />}</button></div></div>
            <div className="support-handling"><select aria-label="工单状态" value={ticket.status} disabled={busy} onChange={event => change({ status: event.target.value })}>{Object.entries(states).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><select aria-label="负责人" disabled={busy} value={ticket.assigneeId} onChange={event => change({ assigneeId: event.target.value })}><option value="">未分配</option>{tools.agents.map(agent => <option key={agent.userId} value={agent.userId}>{agent.name}</option>)}</select>{ticket.assigneeId !== user.userId && <button disabled={busy} onClick={() => change({ assigneeId: user.userId, status: "IN_PROGRESS" })}><UserRound size={14} />由我处理</button>}{!["RESOLVED", "CLOSED"].includes(ticket.status) && <button className="support-resolve" disabled={busy} onClick={() => change({ status: "RESOLVED" })}><Check size={15} />标记已解决</button>}</div>
          </header>
          {position < 0 && !loading && <div className="support-filter-note">此工单不在当前筛选中，仍可继续处理。<button onClick={clearFilters}>查看全部</button></div>}
          {detailError && <div className="support-feedback error" role="alert"><span>{detailError}</span><button onClick={() => loadDetail()}>重新读取</button></div>}
          <div ref={messagesRef} className="support-messages" onScroll={event => { const node = event.currentTarget; nearBottom.current = node.scrollHeight - node.scrollTop - node.clientHeight < 60; if (nearBottom.current) setNewMessages(false); }}>
            {detail.messages.map(message => <article key={message.messageId} className={`support-message ${message.visibility === "INTERNAL" ? "internal" : ""} ${message.actorId === user.userId ? "own" : ""}`}>
              <Avatar user={message.actorName} /><div><header><strong>{message.direction === "INBOUND" ? ticket.requesterName || ticket.requesterEmail : message.actorName}</strong><span>{message.visibility === "INTERNAL" ? <><LockKeyhole size={12} />内部备注</> : message.direction === "INBOUND" ? "玩家来信" : "玩家支持回复"}</span><time>{time(message.createdAt)}</time></header>
                {message.visibility !== "INTERNAL" && <details className="support-mail-addresses"><summary>{message.direction === "INBOUND" ? message.authorEmail : `发往 ${ticket.requesterEmail}`}</summary><p>发件人：{message.authorEmail || tools.mailbox?.email}<br />收件人：{message.direction === "INBOUND" ? tools.mailbox?.email : ticket.requesterEmail}{message.direction === "INBOUND" && ticket.source === "HELP" ? <><br />Help 转交，回复地址：{ticket.requesterEmail}</> : null}</p></details>}
                <MailBody text={message.body} />
                {message.attachments?.length > 0 && <div className="support-attachments">{message.attachments.map(attachment => <a key={attachment.attachmentId} href={`${base}/${ticket.ticketId}/attachments/${attachment.attachmentId}`} download>{attachment.filename}<small>{(attachment.size / 1024).toFixed(0)} KB</small></a>)}</div>}
                {message.direction === "OUTBOUND" && <small className="support-mail-receipt">{deliveryLabels[detail.delivery?.find(event => message.mailMessageId?.startsWith(event.eventId + "@"))?.status] || "已记录"}</small>}
              </div>
            </article>)}
          </div>
          {newMessages && <button className="support-new-messages" onClick={goLatest}><ArrowDown size={14} />查看新消息</button>}
          <form ref={formRef} className={`support-composer ${visibility === "INTERNAL" ? "internal" : ""}`} onSubmit={event => { event.preventDefault(); if (draft.trim()) change({ body: draft.trim(), visibility, ...(visibility === "PUBLIC" ? { status: replyStatus } : {}) }, true); }} onKeyDown={event => { if ((event.ctrlKey || event.metaKey) && event.key === "Enter") { event.preventDefault(); formRef.current?.requestSubmit(); } }}>
            <div className="support-composer-toolbar"><div role="group" aria-label="回复方式"><button type="button" disabled={busy} aria-pressed={visibility === "PUBLIC"} className={visibility === "PUBLIC" ? "active" : ""} onClick={() => setVisibility("PUBLIC")}><MessageSquare size={15} />邮件回复</button><button type="button" disabled={busy} aria-pressed={visibility === "INTERNAL"} className={visibility === "INTERNAL" ? "active" : ""} onClick={() => setVisibility("INTERNAL")}><LockKeyhole size={15} />内部备注</button></div><select aria-label="插入快捷回复" value="" disabled={busy} onChange={event => { const value = tools.macros.find(item => item.macroId === event.target.value); if (value) updateDraft([draft, value.body].filter(Boolean).join("\n\n")); if (event.target.value === "new") setMacro({}); }}><option value="">快捷回复</option>{tools.macros.map(item => <option key={item.macroId} value={item.macroId}>{item.title}</option>)}<option value="new">管理快捷回复…</option></select></div>
            <div className="support-recipient">{visibility === "INTERNAL" ? <><LockKeyhole size={12} />仅工作人员可见，不发送邮件</> : <>收件人：{ticket.requesterEmail}</>}<span>{draft ? "切换工单保留草稿" : "Ctrl / ⌘ + Enter 提交"}</span></div>
            <textarea aria-label={visibility === "INTERNAL" ? "内部备注内容" : "回复内容"} placeholder={visibility === "INTERNAL" ? "记录处理进展，仅内部可见…" : "输入邮件回复…"} value={draft} onChange={event => updateDraft(event.target.value)} maxLength={12000} rows={3} disabled={busy} />
            <footer><label className="support-advance"><input type="checkbox" checked={advance} onChange={event => setAdvance(event.target.checked)} disabled={busy} />提交后下一条</label><div className="support-send-actions">{visibility === "PUBLIC" && <select aria-label="发送后的状态" value={replyStatus} disabled={busy} onChange={event => setReplyStatus(event.target.value)}><option value="WAITING_CUSTOMER">等待玩家回复</option><option value="IN_PROGRESS">继续处理</option><option value="RESOLVED">已解决</option></select>}<Button type="submit" disabled={busy || !draft.trim()}><Send size={15} />{busy ? "正在提交…" : visibility === "INTERNAL" ? "保存内部备注" : sendLabels[replyStatus]}</Button></div></footer>
          </form>
        </>}
      </section>
      {contextOpen && detail && <><button className="support-context-backdrop" aria-label="关闭工单资料" onClick={() => { setContextOpen(false); contextToggleRef.current?.focus(); }} /><aside ref={contextRef} className="support-context" id="support-context" aria-label="工单资料"><div className="support-context-heading"><h2>工单资料</h2><button className="icon-button" aria-label="关闭工单资料面板" onClick={() => { setContextOpen(false); contextToggleRef.current?.focus(); }}><X size={18} /></button></div><h3>玩家信息</h3><div className="support-person"><Avatar user={ticket.requesterName} /><div><strong>{ticket.requesterName}</strong><small>{ticket.source === "HELP" ? "Help 转交" : ticket.source === "ADMIN" ? "玩家支持主动建单" : "邮箱来信"}</small></div></div><dl><div><dt>游戏 ID</dt><dd>{detail.requester.gameId || "未绑定"}</dd></div><div><dt>QQ</dt><dd>{detail.requester.qq || "未填写"}</dd></div><div><dt>邮箱</dt><dd>{detail.requester.email || ticket.requesterEmail}</dd></div><div><dt>历史工单</dt><dd>{detail.requester.ticketCount} 件</dd></div><div><dt>创建时间</dt><dd>{time(ticket.createdAt)}</dd></div></dl><h3>分类与优先级</h3><Field label="问题类型"><select disabled={busy} value={ticket.category} onChange={event => change({ category: event.target.value })}>{Object.entries(categories).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></Field><Field label="优先级"><select disabled={busy} value={ticket.priority} onChange={event => change({ priority: event.target.value })}>{Object.entries(priorities).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></Field>
        {detail.delivery?.length > 0 && <><h3>邮件投递</h3>{detail.delivery.map(event => <div className="support-delivery" key={event.eventId}><strong>{event.kind === "TICKET_CREATED" ? "收件回执" : "人工回复"}</strong><span>{deliveryLabels[event.status] || event.status}</span>{event.lastError && <small>{event.lastError}</small>}{event.status === "FAILED" && <Button secondary disabled={busy} onClick={() => retryMail(event)}>重新发送</Button>}</div>)}</>}
      </aside></>}
    </div>
    {macro && <Modal title="快捷回复" close={() => setMacro(null)} guardClose><MacroForm client={client} macros={tools.macros} onSaved={async () => { await loadTools(); setMacro(null); }} /></Modal>}
    {creating && <Modal title="新建工单" className="support-create-modal" close={() => setCreating(false)} guardClose dismissOnBackdrop={false}><NewTicketForm client={client} onCreated={result => { setCreating(false); choose(result.ticketId); clearFilters(); load(); loadTools(); setNotice(`工单 #${result.ticketId} 已创建，首封邮件已加入发送队列。`); }} /></Modal>}
  </div>;
}

function NewTicketForm({ client, onCreated }) {
  const [value, setValue] = useState({ email: "", subject: "", category: "OTHER", body: "" });
  const [busy, setBusy] = useState(false), [uncertain, setUncertain] = useState(false), [error, setError] = useState("");
  const pending = useRef(null), submitting = useRef(false);
  useUnsavedChanges(value, busy);
  const save = async event => {
    event.preventDefault(); if (submitting.current) return;
    const input = { ...value, email: value.email.trim(), subject: value.subject.trim(), body: value.body.trim() };
    const fingerprint = JSON.stringify(input);
    if (!pending.current || !uncertain && pending.current.fingerprint !== fingerprint) pending.current = { fingerprint, body: { ...input, clientRequestId: id() } };
    submitting.current = true; setBusy(true); setError("");
    try {
      const response = await client.request(base, { method: "POST", body: pending.current.body });
      onCreated(response.data);
    } catch (e) {
      setError(e.message);
      setUncertain(["NETWORK_ERROR", "INVALID_RESPONSE"].includes(e.code) || e.status >= 500);
    } finally { submitting.current = false; setBusy(false); }
  };
  return <form className="support-create-form" onSubmit={save}>
    <p>主动联系玩家。邮件顶部会显示工单编号，玩家可直接回复；不会另发“工单已收到”自动回执。</p>
    <fieldset disabled={busy || uncertain}>
      <Field label="收件邮箱" type="email" required maxLength={254} autoComplete="off" placeholder="player@example.com" value={value.email} onChange={event => setValue({ ...value, email: event.target.value })} />
      <Field label="主题" required minLength={2} maxLength={120} value={value.subject} onChange={event => setValue({ ...value, subject: event.target.value })} />
      <Field label="问题类型"><select value={value.category} onChange={event => setValue({ ...value, category: event.target.value })}>{Object.entries(categories).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></Field>
      <Field label="邮件正文"><textarea required minLength={2} maxLength={12000} rows={6} value={value.body} onChange={event => setValue({ ...value, body: event.target.value })} /></Field>
    </fieldset>
    {error && <p className="auth-error" role="alert">{error}</p>}
    {uncertain && <p role="status">提交结果尚未确认。请用下方按钮重试确认，将沿用原请求，不会重复建单。</p>}
    <footer><small>创建后由你负责，状态为“等玩家回复”。</small><Button type="submit" disabled={busy || !value.email.trim() || value.subject.trim().length < 2 || value.body.trim().length < 2}><Send size={15} />{busy ? "正在提交…" : uncertain ? "确认创建结果" : "创建并发送"}</Button></footer>
  </form>;
}

function MacroForm({ client, macros, onSaved }) {
  const [value, setValue] = useState({ macroId: id(), title: "", body: "", version: 0 }), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const pending = useRef(null); useUnsavedChanges(value, busy);
  const save = async event => {
    event.preventDefault(); if (busy) return; setBusy(true);
    const fingerprint = JSON.stringify(value);
    if (pending.current?.fingerprint !== fingerprint) pending.current = { fingerprint, body: { ...value, clientRequestId: id() } };
    try { await client.request(`/api/v1/admin/support/macros/${value.macroId}`, { method: "PUT", body: pending.current.body }); await onSaved(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={save}><Field label="选择快捷回复"><select value={value.version ? value.macroId : ""} onChange={event => setValue(macros.find(item => item.macroId === event.target.value) || { macroId: id(), title: "", body: "", version: 0 })}><option value="">新建</option>{macros.map(item => <option key={item.macroId} value={item.macroId}>{item.title}</option>)}</select></Field><Field label="名称" required maxLength={100} value={value.title} onChange={event => setValue({ ...value, title: event.target.value })} /><Field label="回复内容"><textarea required rows={6} maxLength={6000} value={value.body} onChange={event => setValue({ ...value, body: event.target.value })} /></Field>{error && <p className="auth-error">{error}</p>}<Button type="submit" disabled={busy}><Zap size={16} />保存快捷回复</Button></form>;
}

function MailBody({ text }) {
  const split = text.search(/\n(?:On .+wrote:|在.{0,100}写道[：:]|[-]{3,}.*(?:原始邮件|Original Message)|>)/i);
  const current = split < 0 ? text : text.slice(0, split);
  return <><p>{current || text}</p>{split >= 0 && current && <details className="support-quoted"><summary>查看引用的邮件内容</summary><pre>{text.slice(split)}</pre></details>}</>;
}

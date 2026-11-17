import React, { useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  Search,
  Hash,
  Send,
  Smile,
  Keyboard,
  Reply,
  X,
  PanelRight,
  ArrowDown,
  ArrowLeft,
  Check,
  Megaphone,
  ArrowUpRight,
  Copy,
  LoaderCircle,
  WifiOff,
  ChevronDown,
  Plus,
  Forward,
} from "lucide-react";
import { Avatar, Empty, Modal, media } from "./components.jsx";
import { id } from "./format.js";
import { deleteChatGrapheme } from "./chat-expressions.js";
import { ChatMediaBubble, ChatPictureButton, ChatExpressionPanel, ChatImageViewer } from "./ChatMedia.jsx";

function messageDate(value) {
  const date = new Date(value);
  if (!value || !Number.isFinite(date.getTime())) return "";
  const today = new Date(), yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  if (date.toDateString() === today.toDateString()) return "今天";
  if (date.toDateString() === yesterday.toDateString()) return "昨天";
  return date.toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric" });
}

function conversationTime(value) {
  const date = new Date(value), label = messageDate(value);
  if (!label) return "";
  return label === "今天" ? date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", hour12: false }) : label === "昨天" ? label : date.toLocaleDateString("zh-CN");
}

function sameMessageGroup(previous, message) {
  const gap = Date.parse(message.sentAt) - Date.parse(previous?.sentAt);
  return previous?.sender === message.sender && previous?.source === message.source &&
    gap >= 0 && gap < 5 * 60 * 1000 && messageDate(previous.sentAt) === messageDate(message.sentAt);
}

function senderTone(sender = "") {
  return [...String(sender || "")].reduce((hash, character) => (hash * 31 + character.codePointAt(0)) % 6, 0);
}

function Highlight({ text, query }) {
  if (!query || !text) return text;
  const literal = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return text.split(new RegExp(`(${literal})`, "gi")).map((part, index) =>
    index % 2 ? <mark key={index}>{part}</mark> : part);
}

function MessageActions({ message, copied, onCopy, onReply, onForward }) {
  const accepted = !message.id.startsWith("pending:");
  return <>
    {!message.media && <button type="button" aria-label={`复制消息 ${message.id}`} title="复制" onClick={() => onCopy(message)}>{copied ? <Check size={16} /> : <Copy size={16} />}<span>复制</span></button>}
    {accepted && onReply && <button type="button" aria-label={`回复消息 ${message.id}`} title="回复" onClick={() => onReply(message)}><Reply size={16} /><span>回复</span></button>}
    {accepted && !message.media && onForward && <button type="button" aria-label={`转发消息 ${message.id}`} title="转发" onClick={() => onForward(message)}><Forward size={16} /><span>转发</span></button>}
  </>;
}

function messageExcerpt(message) {
  return message.media ? (message.media.type === "sticker" ? "[贴纸]" : "[图片]") : message.text;
}

export default function Messenger({
  user,
  contacts,
  messages,
  drafts = {},
  readPositions = {},
  onDraft,
  onRead,
  onSend,
  onRetry,
  onLoadEarlier,
  hasMore = false,
  loadingEarlier = false,
  onProfile,
  announcements = [],
  onAnnouncement,
  onAnnouncements,
  connection = "connecting",
  onReconnect,
  maxLength = 2000,
  activeConversation,
  onConversationChange,
  onNewConversation,
  canReply = false,
  onForwardMessage,
  threadActions,
  conversationStatus,
  mediaService,
  mediaState,
}) {
  const [viewingMedia, setViewingMedia] = useState(null);
  const composerElement = useRef(null);
  const [active, setActive] = useState("public"),
    [contactQuery, setContactQuery] = useState(""),
    [contactFilter, setContactFilter] = useState("all"),
    [text, setText] = useState(drafts[`${user.id}:public`] || ""),
    [reply, setReply] = useState(null),
    [showSearch, setShowSearch] = useState(false),
    [search, setSearch] = useState(""),
    [details, setDetails] = useState(false),
    [mobileThread, setMobileThread] = useState(false),
    [compact, setCompact] = useState(() => window.matchMedia("(max-width: 767px)").matches),
    [emoji, setEmoji] = useState(false),
    [sending, setSending] = useState({}),
    [error, setError] = useState(""),
    [newBelow, setNewBelow] = useState(false),
    [awayFromBottom, setAwayFromBottom] = useState(false),
    [actionsId, setActionsId] = useState(null),
    [locatedId, setLocatedId] = useState(null),
    [arrivedId, setArrivedId] = useState(null),
    [copied, setCopied] = useState("");
  const selection = useRef({start: 0, end: 0}), dockPinUntil = useRef(0), beforeSearch = useRef(null), scrollInteraction = useRef(0);
  const scroll = useRef(null), stack = useRef(null), detailsToggle = useRef(null), searchToggle = useRef(null),
    input = useRef(null),
    searchInput = useRef(null),
    atBottom = useRef(true),
    lastAttempt = useRef(null),
    draftRef = useRef(text),
    prevLast = useRef(null),
    activeRef = useRef(null), savedDrafts = useRef(drafts);
  savedDrafts.current = drafts;
  draftRef.current = text;
  const current = contacts.find((c) => c.id === active) || contacts[0],
    channel = current?.channel || "public",
    all = messages.filter((m) => m.channel === channel),
    lastId = all.at(-1)?.id;
  activeRef.current = channel;
  const query = search.trim();
  const visible = query
    ? all.filter((m) =>
        `${m.text} ${m.senderName || m.sender}`
          .toLowerCase()
          .includes(query.toLowerCase()),
      )
    : all;
  const visibleById = new Map(visible.map(message => [message.id, message]));
  const actionMessage = visibleById.get(actionsId);
  const threadVisible = !compact || mobileThread;
  const ready = connection === "connected",
    unsupported = current?.unavailable;
  const connectionLabel = ready ? "实时连接正常" : connection === "connecting" ? "正在连接…" : connection === "reconnecting" ? "正在重连…" : "连接已断开";
  useEffect(() => {
    const viewport = window.matchMedia("(max-width: 767px)");
    const update = () => setCompact(viewport.matches);
    viewport.addEventListener("change", update);
    return () => viewport.removeEventListener("change", update);
  }, []);
  const toBottom = () => {
    if (scroll.current) scroll.current.scrollTop = scroll.current.scrollHeight;
    atBottom.current = true;
    setNewBelow(false);
    setAwayFromBottom(false);
  };
  const rememberSelection = () => { if (input.current) selection.current = {start: input.current.selectionStart, end: input.current.selectionEnd}; };
  const insertEmoji = value => {
    const start = Math.min(selection.current.start, text.length), finish = Math.min(selection.current.end, text.length);
    const next = text.slice(0, start) + value + text.slice(finish); if (next.length > maxLength) return false;
    setText(next); selection.current = {start: start + value.length, end: start + value.length};
    requestAnimationFrame(() => input.current?.setSelectionRange(selection.current.start, selection.current.end));
  };
  const backspaceEmoji = () => {
    const next = deleteChatGrapheme(text, Math.min(selection.current.start, text.length), Math.min(selection.current.end, text.length));
    setText(next.text); selection.current = {start: next.caret, end: next.caret};
    requestAnimationFrame(() => input.current?.setSelectionRange(next.caret, next.caret));
  };
  const toggleExpressions = () => {
    if (atBottom.current) dockPinUntil.current = performance.now() + 400;
    if (emoji) { setEmoji(false); input.current?.focus({preventScroll: true}); }
    else { rememberSelection(); input.current?.blur(); setEmoji(true); if (channel === "public") mediaService?.refreshStickers(); }
  };
  useEffect(() => {
    if (!emoji) return;
    const close = event => { if (!composerElement.current?.contains(event.target)) setEmoji(false); };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [emoji]);
  useLayoutEffect(() => { if (input.current) { input.current.style.height = "0px"; input.current.style.height = Math.min(132, Math.max(44, input.current.scrollHeight)) + "px"; } }, [text, channel]);
  useLayoutEffect(() => {
    const composer = composerElement.current;
    const observer = new ResizeObserver(() => {
      composer.closest(".pc-thread")?.style.setProperty("--chat-dock-height", `${composer.offsetHeight}px`);
      if (!beforeSearch.current && (atBottom.current || performance.now() < dockPinUntil.current)) toBottom();
    });
    observer.observe(composer);
    observer.observe(stack.current);
    return () => observer.disconnect();
  }, []);
  useLayoutEffect(() => {
    if (query) scroll.current.scrollTop = 0;
    else if (beforeSearch.current) {
      const position = beforeSearch.current;
      beforeSearch.current = null;
      if (position.bottom) toBottom();
      else {
        scroll.current.scrollTop = position.top;
        atBottom.current = false;
        setAwayFromBottom(true);
      }
    }
  }, [query]);
  useEffect(() => {
    if (query) return;
    const previous = prevLast.current;
    if (previous?.channel === channel && previous.lastId && previous.lastId !== lastId && all.at(-2)?.id === previous.lastId)
      setArrivedId(lastId);
    let frame;
    if (channel !== prevLast.current?.channel || atBottom.current) {
      frame = requestAnimationFrame(toBottom);
    } else if (lastId !== prevLast.current?.lastId) setNewBelow(true);
    prevLast.current = { channel, lastId };
    if (
      lastId &&
      onRead &&
      atBottom.current && !query &&
      threadVisible
    )
      onRead(channel, lastId);
    return () => cancelAnimationFrame(frame);
  }, [channel, lastId, threadVisible, query]);
  useEffect(() => {
    if (showSearch) searchInput.current?.focus();
    else setSearch("");
  }, [showSearch]);
  useEffect(() => {
    const timeout = setTimeout(() => onDraft?.(channel, text), 350);
    return () => clearTimeout(timeout);
  }, [channel, text]);
  useEffect(() => {
    const find = (e) => {
      if (document.querySelector('dialog[open]') || !scroll.current?.clientHeight) return;
      if ((e.ctrlKey || e.metaKey) && e.key === "f") {
        e.preventDefault();
        setShowSearch(true);
        searchInput.current?.focus();
      }
      if (e.key === "Escape" && !e.isComposing) {
        if (emoji) { setEmoji(false); input.current?.focus({preventScroll: true}); }
        else if (details) { setDetails(false); detailsToggle.current?.focus(); }
        else if (showSearch) { setShowSearch(false); searchToggle.current?.focus(); }
        else setReply(null);
      }
    };
    const searchEvent = () => setShowSearch(true);
    window.addEventListener("deuterium:search", searchEvent);
    window.addEventListener("keydown", find);
    return () => {
      window.removeEventListener("keydown", find);
      window.removeEventListener("deuterium:search", searchEvent);
    };
  }, [emoji, details, showSearch]);
  useEffect(() => {
    if (!locatedId) return;
    const timer = setTimeout(() => setLocatedId(null), 1600);
    return () => clearTimeout(timer);
  }, [locatedId]);
  const choose = (c) => {
    setMobileThread(true);
    if (c.id === active) return;
    beforeSearch.current = null;
    scrollInteraction.current++;
    onDraft?.(channel, text);
    setActive(c.id);
    onConversationChange?.(c.id);
    setText(drafts[`${user.id}:${c.channel}`] || "");
    setReply(null);
    setSearch("");
    setShowSearch(false);
    setError("");
    setDetails(false);
    setActionsId(null);
    setLocatedId(null);
    setEmoji(false);
    selection.current = {start: 0, end: 0};
    atBottom.current = true;
    lastAttempt.current = null;
    if (window.matchMedia("(min-width: 768px) and (pointer: fine)").matches)
      requestAnimationFrame(() => input.current?.focus({preventScroll: true}));
  };
  useEffect(() => {
    if (activeConversation && activeConversation !== active) {
      const next = contacts.find((c) => c.id === activeConversation);
      if (next) choose(next);
    }
  }, [activeConversation, contacts]);
  const send = async () => {
    if (!ready || !text.trim() || sending[channel] || unsupported) return;
    if (channel === "public" && /[\r\n\t]/.test(text)) {
      setError("当前公共频道暂不支持换行或制表符，请改为单段消息。");
      return;
    }
    beforeSearch.current = null;
    setShowSearch(false);
    toBottom();
    const submitted = text.trim();
    if (!lastAttempt.current || lastAttempt.current.text !== submitted || lastAttempt.current.channel !== channel || lastAttempt.current.replyId !== reply?.id)
      lastAttempt.current = { text: submitted, channel, replyId: reply?.id, id: id() };
    setSending((old) => ({...old,[channel]:true}));
    setError("");
    try {
      const ok = await onSend({
        channel,
        text: submitted,
        reply,
        clientMessageId: lastAttempt.current.id,
      });
      if (ok !== false) {
        if (activeRef.current === channel) {
          if (draftRef.current === text) { setText(""); onDraft?.(channel, ""); }
          setReply(currentReply => currentReply?.id === reply?.id ? null : currentReply);
          lastAttempt.current = null;
        } else if (savedDrafts.current[`${user.id}:${channel}`] === text) onDraft?.(channel, "");
      }
    } catch (e) {
      if (activeRef.current === channel) setError(e.message || "消息未能发送，输入内容已保留。");
    } finally {
      setSending((old) => ({...old,[channel]:false}));
    }
  };
  const unread = (c) => {
    if (typeof c.unreadCount === "number") return c.unreadCount;
    const msgs = messages.filter((m) => m.channel === c.channel);
    const last = readPositions[`${user.id}:${c.channel}`];
    if (!last) return msgs.filter((m) => m.sender !== user.id).length;
    const idx = msgs.findIndex((m) => m.id === last);
    return msgs.slice(idx + 1).filter((m) => m.sender !== user.id).length;
  };
  const copy = async (m) => {
    try {
      await navigator.clipboard.writeText(m.text);
      setCopied(m.id);
      setTimeout(() => setCopied(""), 1200);
    } catch {
      setActionsId(null);
      setError("浏览器未允许复制，请选中文字手动复制。");
    }
  };
  const replyTo = message => {
    setReply(message);
    setActionsId(null);
    requestAnimationFrame(() => input.current?.focus({preventScroll: true}));
  };
  const locate = messageId => {
    const row = scroll.current?.querySelector(`[data-message-id="${CSS.escape(messageId)}"]`);
    if (!row) return;
    row.scrollIntoView({block: "center", behavior: "instant"});
    setLocatedId(messageId);
  };
  const filteredContacts = contacts.filter(c =>
    `${c.name} ${c.caption || ""}`.toLowerCase().includes(contactQuery.trim().toLowerCase()) &&
    (contactFilter !== "unread" || ((!threadVisible || active !== c.id) && unread(c) > 0)) &&
    (contactFilter !== "direct" || (c.id !== "public" && c.id !== "assistant")),
  );
  return (
    <div
      className={`pc-chat ${details ? "with-details" : ""} ${mobileThread ? "thread-open" : ""}`}
    >
      <aside className="pc-conversations">
        <header>
          <h1>聊天</h1>
          {onNewConversation && <button className="icon-button new-conversation" aria-label="发起私聊" title="发起私聊" onClick={onNewConversation}><Plus size={22} /></button>}
          <button
            className="icon-button"
            aria-label="查看社区公告"
            title="查看社区公告"
            onClick={onAnnouncements}
          >
            <Megaphone size={19} />
          </button>
        </header>
        <label className="conversation-search">
          <Search size={16} />
          <input
            aria-label="搜索会话"
            value={contactQuery}
            onChange={(e) => setContactQuery(e.target.value)}
            placeholder="搜索会话或玩家"
          />
          {contactQuery && <button type="button" aria-label="清除会话搜索" onClick={() => setContactQuery("")}><X size={16} /></button>}
        </label>
        <div className="conversation-filters" role="group" aria-label="筛选会话">
          {[["all", "全部"], ["unread", "未读"], ["direct", "私聊"]].map(([value, label]) => <button key={value} aria-pressed={contactFilter === value} className={contactFilter === value ? "selected" : ""} onClick={() => setContactFilter(value)}>{label}</button>)}
        </div>
        <div className="pc-contact-scroll">
          {filteredContacts
            .map((c) => {
              const latest = messages
                  .filter((m) => m.channel === c.channel)
                  .at(-1),
                n = threadVisible && active === c.id ? 0 : unread(c);
              return (
                <button
                  className={`pc-contact ${threadVisible && active === c.id ? "selected" : ""}`}
                  key={c.id}
                  aria-pressed={threadVisible && active === c.id}
                  onClick={() => choose(c)}
                >
                  {c.id === "public" ? (
                    <span className="conversation-symbol">
                      <Hash size={22} />
                    </span>
                  ) : c.id === "assistant" ? (
                    <img
                      className="conversation-symbol"
                      src={media("xiaoxiang_avatar.png")}
                      alt="小祥"
                    />
                  ) : (
                    <Avatar user={c.player || c.avatar || c.name} />
                  )}
                  <span className="pc-contact-content">
                    <span>
                      <strong>{c.name}</strong>
                      <time dateTime={latest?.sentAt || c.updatedAt} className={n > 0 ? "unread-time" : ""}>{conversationTime(latest?.sentAt || c.updatedAt) || latest?.time || ""}</time>
                    </span>
                    <span>
                      <small>
                        {drafts[`${user.id}:${c.channel}`] &&
                        (!threadVisible || active !== c.id) ? (
                          <>
                            <em>[草稿]</em> {drafts[`${user.id}:${c.channel}`]}
                          </>
                        ) : (
                          latest ? <>{(latest.mine || latest.sender === user.id) ? "你：" : c.channel === "public" && latest.senderName ? `${latest.senderName}：` : ""}{messageExcerpt(latest)}</> : c.caption || "开始一段对话"
                        )}
                      </small>
                      {n > 0 && <b>{n > 99 ? "99+" : n}</b>}
                    </span>
                  </span>
                </button>
              );
            })}
          {!filteredContacts.length && <div className="conversation-empty" role="status"><Search size={24} /><strong>{contactQuery ? "没有找到会话" : contactFilter === "unread" ? "暂无未读消息" : "还没有私聊"}</strong><p>{contactQuery ? "换个名字或关键词试试。" : contactFilter === "unread" ? "新消息会出现在这里。" : "点击上方加号，找一位玩家聊聊。"}</p></div>}
        </div>
        <footer>
          <span className={`connection-dot ${ready ? "ready" : ""}`} />
          <span>
            {connectionLabel}
          </span>
          {!ready && onReconnect && <button onClick={onReconnect}>重连</button>}
        </footer>
      </aside>
      <section className={`pc-thread ${channel === "public" ? "public-thread" : "private-thread"}`}>
        <header className="pc-thread-header">
          <button
            className="icon-button pc-back"
            aria-label="返回会话列表"
            onClick={() => { setMobileThread(false); setEmoji(false); setShowSearch(false); setDetails(false); }}
          >
            <ArrowLeft size={20} />
          </button>
          {active === "public" ? (
            <span className="conversation-symbol">
              <Hash size={21} />
            </span>
          ) : active === "assistant" ? (
            <img
              className="conversation-symbol"
              src={media("xiaoxiang_avatar.png")}
              alt="小祥"
            />
          ) : (
            <Avatar user={current?.player || current?.name} />
          )}
          <div>
            <h2>{current?.name}</h2>
            <p>{unsupported || (active === "public" ? "服务器公共消息" : active === "assistant" ? current?.caption : "私聊 · 仅双方可见")}</p>
          </div>
          <div className="pc-thread-actions">
            {threadActions}
            <button
              className={`icon-button ${showSearch ? "selected" : ""}`}
              aria-label="搜索当前会话消息"
              aria-expanded={showSearch}
              ref={searchToggle}
              title="搜索当前会话 · Ctrl F"
              onClick={() => setShowSearch(!showSearch)}
            >
              <Search size={19} />
            </button>
            <button
              className={`icon-button ${details ? "selected" : ""}`}
              aria-label={details ? "收起会话详情" : "展开会话详情"}
              aria-expanded={details}
              ref={detailsToggle}
              onClick={() => setDetails(!details)}
            >
              <PanelRight size={19} />
            </button>
          </div>
        </header>
        {announcements[0] && active === "public" && (
          <button
            className="chat-announcement-strip"
            onClick={() => onAnnouncement(announcements[0])}
          >
            <Megaphone size={14} />
            <span>{announcements[0].title}</span>
            <ChevronDown size={14} />
          </button>
        )}
        {showSearch && (
          <div className="message-search-row">
            <Search size={16} />
            <input
              ref={searchInput}
              aria-label="搜索已加载消息"
              value={search}
              onChange={(e) => {
                if (!query && e.target.value.trim()) beforeSearch.current = { top: scroll.current.scrollTop, bottom: atBottom.current };
                setSearch(e.target.value);
              }}
              placeholder="搜索已加载的消息和发言人"
            />
            {query && <span role="status">{visible.length} 条结果</span>}
            <button
              className="icon-button"
              aria-label="关闭消息搜索"
              onClick={() => { setShowSearch(false); searchToggle.current?.focus(); }}
            >
              <X size={16} />
            </button>
          </div>
        )}
        {!ready && (
          <div className="chat-connection-banner">
            <WifiOff size={15} />
            <span role="status">{connection === "connecting" ? "正在连接会话…" : "实时连接暂不可用，已加载的消息仍可查看。"}</span>
            {onReconnect && <button onClick={onReconnect}>重新连接</button>}
          </div>
        )}
        <div
          ref={scroll}
          className="pc-message-scroll"
          onPointerDown={() => { dockPinUntil.current = 0; scrollInteraction.current++; }}
          onWheel={() => { dockPinUntil.current = 0; scrollInteraction.current++; }}
          onKeyDown={() => { scrollInteraction.current++; }}
          onScroll={() => {
            const el = scroll.current;
            if (!el.clientHeight) return;
            atBottom.current =
              el.scrollHeight - el.scrollTop - el.clientHeight < 90;
            setAwayFromBottom(!atBottom.current);
            if (atBottom.current) {
              setNewBelow(false);
              if (lastId && onRead && !query) onRead(channel, lastId);
            }
          }}
        >
          {hasMore && (
            <button
              className="load-earlier"
              disabled={loadingEarlier}
              onClick={async () => {
                const el = scroll.current;
                const interaction = scrollInteraction.current;
                const anchor = Array.from(stack.current.querySelectorAll('[data-message-id]')).find(row => row.getBoundingClientRect().bottom > el.getBoundingClientRect().top);
                const top = anchor?.getBoundingClientRect().top;
                await onLoadEarlier?.();
                requestAnimationFrame(() => {
                  if (interaction === scrollInteraction.current && anchor?.isConnected) el.scrollTop += anchor.getBoundingClientRect().top - top;
                });
              }}
            >
              {loadingEarlier ? "正在加载…" : "查看更早的消息"}
            </button>
          )}
          <div ref={stack} className="pc-message-stack">
          {unsupported ? (
            <Empty title="服务暂不可用" text={unsupported} />
          ) : !visible.length ? (
            <Empty
              title={
                search
                  ? "没有匹配的消息"
                  : active === "assistant"
                    ? "你好，我是小祥。"
                    : "从一句问候开始"
              }
              text={
                search
                  ? "试试其他关键词，或加载更早的消息。"
                  : channel === "public" ? "和大家分享游戏里的新鲜事。" : "发一条消息，开始这段对话。"
              }
            />
          ) : (
            visible.map((m, i) => {
              const mine = m.sender === user.id || m.mine,
                name =
                  m.senderName ||
                  contacts.find((c) => c.id === m.sender)?.name ||
                  m.sender;
              const grouped = !search && sameMessageGroup(visible[i - 1], m);
              const continues = !search && sameMessageGroup(m, visible[i + 1] || {});
              const date = messageDate(m.sentAt);
              const original = m.reply?.availability !== "UNAVAILABLE" && visibleById.get(m.reply?.id);
              const quotedName = original?.senderName || m.reply?.senderName || "引用消息";
              const quote = m.reply && <><strong>{quotedName}</strong><span>{m.reply.availability === "UNAVAILABLE" ? "原消息暂不可见" : messageExcerpt(original || m.reply) || "引用消息暂不可见"}</span></>;
              return (
                <React.Fragment key={m.id}>
                {date && date !== messageDate(visible[i - 1]?.sentAt) && <div className="pc-date-divider"><span>{date}</span></div>}
                <div
                  data-message-id={m.id}
                  data-located={locatedId === m.id || undefined}
                  data-arrived={!query && arrivedId === m.id || undefined}
                  className={`pc-message ${mine ? "mine" : ""} ${grouped ? "message-grouped" : ""} ${continues ? "group-continues" : ""} ${channel !== "public" ? "direct-message" : ""} ${m.status === "failed" || m.status === "unknown" ? "message-failed" : ""}`}
                >
                  {m.sender === "assistant" ? (
                    <img
                      className="avatar"
                      src={media("xiaoxiang_avatar.png")}
                      alt="小祥"
                    />
                  ) : (
                    <button
                      className="avatar-button"
                      aria-label={`查看 ${name} 的资料`}
                      onClick={() => onProfile?.(m.sender)}
                    >
                      <Avatar user={m.senderProfile || (mine ? user : contacts.find((c) => c.player?.playerRef === m.sender)?.player) || name} />
                    </button>
                  )}
                  <div className="pc-message-content">
                    <div className={`bubble ${m.media ? "has-media" : ""}`}>
                    {!mine && (channel === "public" || m.sender === "assistant") && !grouped && <div className={`pc-message-meta sender-tone-${senderTone(m.sender)}`}>
                      <strong>
                        <Highlight text={m.sender === "assistant" ? "小祥" : name} query={query} />
                      </strong>
                      {m.source && <span>{m.source}</span>}
                    </div>}
                      {quote && (original ? <button type="button" className="chat-quote" aria-label={`查看原消息：${quotedName}`} onClick={() => locate(m.reply.id)}>{quote}<Reply size={13} /></button> : <blockquote className="chat-quote">{quote}</blockquote>)}
                      {m.forwarded && <blockquote className="chat-quote"><strong>转发自 {m.forwarded.senderName || "玩家"}</strong><span>{m.forwarded.availability === "UNAVAILABLE" ? "原消息暂不可见" : m.forwarded.text}</span></blockquote>}
                      {m.media ? <ChatMediaBubble service={mediaService} media={m.media} messageID={m.isMediaPending ? null : m.id} localURL={m.localURL} onClick={(rect, preview, source) => { if (m.media.mediaId) setViewingMedia({ media: m.media, messageID: m.isMediaPending ? null : m.id, rect, preview, source }); }} /> : <Highlight text={m.text} query={query} />}
                      {m.sources?.length > 0 && <div className="ai-sources">{m.sources.map((source, n) => <a key={`${source.url}:${n}`} href={source.url} target="_blank" rel="noopener noreferrer">{source.origin === "provider_text" ? "正文链接" : "来源"} · {source.title}</a>)}</div>}
                      {m.searchUsed && <small className="muted">已使用联网搜索</small>}
                      {m.sender === "assistant" && ["failed", "unknown", "incomplete"].includes(m.aiStatus) && <p className="composer-error">这次回复未完成，可以开始新对话。</p>}
                      <span className="message-stamp"><time dateTime={m.sentAt} title={m.sentAt ? new Date(m.sentAt).toLocaleString("zh-CN") : undefined}>{m.time}</time>{mine && (m.status === "sending" ? <LoaderCircle className="spin" size={13} aria-label="发送中" /> : m.status === "sent" ? <Check size={14} aria-label="已发送" /> : null)}
                        {(!m.media || (canReply && !m.id.startsWith("pending:"))) && <button type="button" className="message-action-trigger" aria-label={`消息操作 ${m.id}`} aria-haspopup="dialog" onClick={() => setActionsId(m.id)}><ChevronDown size={13} /></button>}
                      </span>
                    </div>
                    {(m.status === "failed" || m.status === "unknown") && (
                      <div className="message-delivery">
                            <span>{m.error || "结果暂未确认"}</span>
                            <button
                              onClick={async () => {
                                try {
                                  await onRetry?.(m);
                                  if (activeRef.current !== channel) return;
                                  if (draftRef.current.trim() === m.text && lastAttempt.current?.id === m.clientMessageId) {
                                    setText("");
                                    lastAttempt.current = null;
                                  }
                                  setError("");
                                } catch (e) {
                                  if (activeRef.current === channel) setError(e.message);
                                }
                              }}
                            >
                              使用原消息重试
                            </button>
                      </div>
                    )}
                  </div>
                  <div className="pc-message-tools">
                    <MessageActions message={m} copied={copied === m.id} onCopy={copy} onReply={canReply && replyTo} onForward={onForwardMessage} />
                  </div>
                </div>
                </React.Fragment>
              );
            })
          )}
          </div>
        </div>
        {awayFromBottom && !query && !emoji && (
          <button className={`new-message-button ${newBelow ? "has-new" : ""}`} aria-label="回到最新消息" title="回到最新消息" onClick={toBottom}>
            <ArrowDown size={18} />{newBelow && "有新消息"}
          </button>
        )}
        <form
          ref={composerElement}
          className="pc-composer chat-composer"
          onSubmit={(e) => {
            e.preventDefault();
            send();
          }}
        >
          {conversationStatus && <p className="muted" role="status">{conversationStatus}</p>}
          {reply && (
            <div className="reply-preview">
              <Reply size={16} />
              <div>
                <strong>回复 {reply.senderName || reply.sender}</strong>
                <span>{messageExcerpt(reply)}</span>
              </div>
              <button
                type="button"
                aria-label="取消回复"
                onClick={() => setReply(null)}
              >
                <X size={16} />
              </button>
            </div>
          )}
          <div className="chat-composer-capsule">
            {channel === "public" && mediaService && <ChatPictureButton service={mediaService} reply={reply} onError={setError} onSent={() => setReply(null)} />}
            <textarea ref={input} aria-label="消息内容" placeholder={unsupported ? "暂不可发送" : "输入消息"} disabled={Boolean(unsupported)} value={text} maxLength={maxLength} rows={1}
              onChange={e => { setText(e.target.value); rememberSelection(); }} onSelect={rememberSelection}
              onFocus={() => { if (emoji) setEmoji(false); }} onBlur={() => { rememberSelection(); onDraft?.(channel, text); }}
              onKeyDown={e => { if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing && e.keyCode !== 229) { e.preventDefault(); send(); } }} />
            <button type="button" className="chat-expression-toggle" aria-label={emoji ? "切换键盘" : "选择表情"} aria-expanded={emoji} onMouseDown={e => e.preventDefault()} onClick={toggleExpressions}>
              {emoji ? <Keyboard size={22} /> : <Smile size={22} />}
            </button>
            <button type="submit" className="chat-send-button" aria-label="发送消息" onMouseDown={e => e.preventDefault()} disabled={sending[channel] || !text.trim() || !ready || Boolean(unsupported)}>
              <span>{sending[channel] ? <LoaderCircle className="spin" size={21} /> : <Send size={21} />}</span>
            </button>
          </div>
          {error && <p className="composer-error" role="alert">{error}</p>}
          {text.length > maxLength * .8 && <div className="chat-character-count">{text.length} / {maxLength}</div>}
          <ChatExpressionPanel service={channel === "public" ? mediaService : null} state={channel === "public" ? mediaState : undefined} open={emoji} onEmoji={insertEmoji} onBackspace={backspaceEmoji}
            onSticker={async media => { try { await mediaService.enqueue(null, reply, media); setReply(null); } catch (e) { setError(e.message); } }}
            onPreview={(media, rect, preview, source) => setViewingMedia({ media, rect, preview, source })} />
        </form>
      </section>
      {actionMessage && <Modal title="消息操作" className="chat-action-sheet" close={() => setActionsId(null)}>
        <div className="chat-action-excerpt"><strong>{actionMessage.senderName || actionMessage.sender}</strong><p>{messageExcerpt(actionMessage)}</p></div>
        <div className="chat-action-options"><MessageActions message={actionMessage} copied={copied === actionMessage.id} onCopy={copy} onReply={canReply && replyTo}
          onForward={onForwardMessage && (message => { setActionsId(null); onForwardMessage(message); })} /></div>
        {copied === actionMessage.id && <p role="status">已复制消息</p>}
      </Modal>}
      {viewingMedia && <ChatImageViewer service={mediaService} {...viewingMedia} close={() => setViewingMedia(null)} />}
      {details && (
        <aside className="pc-details">
          <header>
            <h3>会话详情</h3>
            <button
              className="icon-button"
              aria-label="关闭会话详情"
              onClick={() => { setDetails(false); detailsToggle.current?.focus(); }}
            >
              <X size={17} />
            </button>
          </header>
          <div className="chat-room-profile">
            {active === "public" ? (
              <span className="room-hash">
                <Hash size={34} />
              </span>
            ) : active === "assistant" ? (
              <img className="room-hash" src={media("xiaoxiang_avatar.png")} alt="小祥" />
            ) : (
              <Avatar user={current?.player || current?.name} size="large" />
            )}
            <h3>{current?.name}</h3>
            <p>
              {active === "public"
                ? "服务器公共消息"
                : active === "assistant" ? current?.caption : current?.player?.bio || "私聊 · 仅双方可见"}
            </p>
          </div>
          <div className="pc-detail-section">
            <h4>当前会话</h4>
            <dl>
              <div>
                <dt>消息记录</dt>
                <dd>{all.length} 条已加载</dd>
              </div>
              <div>
                <dt>来源</dt>
                <dd>服务器</dd>
              </div>
              <div>
                <dt>连接状态</dt>
                <dd>{connectionLabel}</dd>
              </div>
            </dl>
          </div>
          {active === "public" && (
            <div className="pc-detail-section">
              <h4>最近发言的玩家</h4>
              {Array.from(new Map(all.map(message => [message.sender, message])).values())
                .sort((left, right) => Date.parse(right.sentAt) - Date.parse(left.sentAt)).slice(0, 8).map(message => (
                <button
                  className="detail-member"
                  key={message.sender}
                  onClick={() => onProfile?.(message.sender)}
                >
                  <Avatar user={message.senderProfile || message.senderName} />
                  <span>{message.senderName || message.sender}</span>
                  <ArrowUpRight size={13} />
                </button>
              ))}
            </div>
          )}
          <div className="pc-detail-section">
            <h4>社区公告</h4>
            {announcements.length ? (
              announcements.slice(0, 3).map((a) => (
                <button
                  className="detail-announcement"
                  key={a.id}
                  onClick={() => onAnnouncement(a)}
                >
                  <Megaphone size={15} />
                  <span>{a.title}</span>
                  <ArrowUpRight size={13} />
                </button>
              ))
            ) : (
              <p className="muted">
                点击下方查看最新公告
              </p>
            )}
            <button className="text-button" onClick={onAnnouncements}>
              查看全部公告 <ArrowUpRight size={14} />
            </button>
          </div>
        </aside>
      )}
    </div>
  );
}

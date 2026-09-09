import React, { useEffect, useRef, useState } from "react";
import {
  Search,
  Hash,
  Send,
  Smile,
  Reply,
  X,
  PanelRight,
  ArrowDown,
  ArrowLeft,
  Check,
  CheckCheck,
  RefreshCw,
  MoreHorizontal,
  Users,
  MessageCircle,
  Megaphone,
  ArrowUpRight,
  Copy,
  LoaderCircle,
  Wifi,
  WifiOff,
  ChevronDown,
  Plus,
  Forward,
} from "lucide-react";
import { Avatar, Badge, Button, Empty, media } from "./components.jsx";
import { id } from "./format.js";

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
}) {
  const [active, setActive] = useState("public"),
    [contactQuery, setContactQuery] = useState(""),
    [text, setText] = useState(drafts[`${user.id}:public`] || ""),
    [reply, setReply] = useState(null),
    [showSearch, setShowSearch] = useState(false),
    [search, setSearch] = useState(""),
    [details, setDetails] = useState(() => window.innerWidth >= 1250),
    [mobileThread, setMobileThread] = useState(false),
    [emoji, setEmoji] = useState(false),
    [sending, setSending] = useState({}),
    [error, setError] = useState(""),
    [newBelow, setNewBelow] = useState(false),
    [copied, setCopied] = useState("");
  const scroll = useRef(null),
    end = useRef(null),
    input = useRef(null),
    searchInput = useRef(null),
    atBottom = useRef(true),
    lastAttempt = useRef(null),
    draftRef = useRef(text),
    prevLast = useRef(null),
    activeRef = useRef(active);
  activeRef.current = active;
  draftRef.current = text;
  const current = contacts.find((c) => c.id === active) || contacts[0],
    channel = current?.channel || "public",
    all = messages.filter((m) => m.channel === channel),
    lastId = all.at(-1)?.id;
  const visible = search
    ? all.filter((m) =>
        `${m.text} ${m.senderName || m.sender}`
          .toLowerCase()
          .includes(search.toLowerCase()),
      )
    : all;
  const isLive = true,
    ready = connection === "connected",
    unsupported = current?.unavailable;
  const toBottom = () => {
    end.current?.scrollIntoView({ block: "nearest", behavior: "instant" });
    atBottom.current = true;
    setNewBelow(false);
  };
  useEffect(() => {
    if (channel !== prevLast.current?.channel || atBottom.current) {
      requestAnimationFrame(toBottom);
    } else if (lastId !== prevLast.current?.lastId) setNewBelow(true);
    prevLast.current = { channel, lastId };
    if (
      lastId &&
      onRead &&
      atBottom.current &&
      (mobileThread || window.innerWidth > 767)
    )
      onRead(channel, lastId);
  }, [channel, lastId, mobileThread]);
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
      if ((e.ctrlKey || e.metaKey) && e.key === "f") {
        e.preventDefault();
        setShowSearch(true);
      }
      if (e.key === "Escape") {
        setSearch("");
        setShowSearch(false);
        setReply(null);
      }
    };
    const searchEvent = () => setShowSearch(true);
    window.addEventListener("deuterium:search", searchEvent);
    window.addEventListener("keydown", find);
    return () => {
      window.removeEventListener("keydown", find);
      window.removeEventListener("deuterium:search", searchEvent);
    };
  }, []);
  const choose = (c) => {
    onDraft?.(channel, text);
    setActive(c.id);
    onConversationChange?.(c.id);
    setText(drafts[`${user.id}:${c.channel}`] || "");
    setReply(null);
    setSearch("");
    setShowSearch(false);
    setError("");
    setMobileThread(true);
    atBottom.current = true;
    lastAttempt.current = null;
    requestAnimationFrame(() => input.current?.focus());
  };
  useEffect(() => {
    if (activeConversation && activeConversation !== active) {
      const next = contacts.find((c) => c.id === activeConversation);
      if (next) choose(next);
    }
  }, [activeConversation, contacts]);
  const send = async () => {
    if (!text.trim() || sending[channel] || unsupported) return;
    if (channel === "public" && /[\r\n\t]/.test(text)) {
      setError("当前公共频道暂不支持换行或制表符，请改为单段消息。");
      return;
    }
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
        setText("");
        setReply(null);
        onDraft?.(channel, "");
        lastAttempt.current = null;
        atBottom.current = true;
        requestAnimationFrame(toBottom);
      }
    } catch (e) {
      setError(e.message || "消息未能发送，输入内容已保留。");
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
      setError("浏览器未允许复制，请选中文字手动复制。");
    }
  };
  return (
    <div
      className={`pc-chat ${details ? "with-details" : ""} ${mobileThread ? "thread-open" : ""}`}
    >
      <aside className="pc-conversations">
        <header>
          <h1>信息</h1>
          {onNewConversation && <button className="icon-button" aria-label="发起私聊" onClick={onNewConversation}><Plus size={19} /></button>}
          <button
            className="icon-button"
            aria-label="查看社区公告"
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
        </label>
        <div className="conversation-section-label">
          会话 <span>{contacts.length}</span>
        </div>
        <div className="pc-contact-scroll">
          {contacts
            .filter((c) =>
              `${c.name}${c.caption || ""}`
                .toLowerCase()
                .includes(contactQuery.toLowerCase()),
            )
            .map((c) => {
              const latest = messages
                  .filter((m) => m.channel === c.channel)
                  .at(-1),
                n = active === c.id ? 0 : unread(c);
              return (
                <button
                  className={`pc-contact ${active === c.id ? "selected" : ""}`}
                  key={c.id}
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
                      <time>{latest?.time || ""}</time>
                    </span>
                    <span>
                      <small>
                        {drafts[`${user.id}:${c.channel}`] &&
                        active !== c.id ? (
                          <>
                            <em>[草稿]</em> {drafts[`${user.id}:${c.channel}`]}
                          </>
                        ) : (
                          latest?.text || c.caption || "开始一段对话"
                        )}
                      </small>
                      {n > 0 && <b>{n > 99 ? "99+" : n}</b>}
                    </span>
                  </span>
                </button>
              );
            })}
        </div>
        <footer>
          <span className={`connection-dot ${ready ? "ready" : ""}`} />
          <span>
            {connection === "connected"
                ? "实时连接正常"
                : connection === "connecting"
                  ? "正在连接…"
                  : connection === "reconnecting"
                    ? "正在重连…"
                    : "连接已断开"}
          </span>
          {!ready && onReconnect && <button onClick={onReconnect}>重连</button>}
        </footer>
      </aside>
      <section className="pc-thread">
        <header className="pc-thread-header">
          <button
            className="icon-button pc-back"
            aria-label="返回会话列表"
            onClick={() => setMobileThread(false)}
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
            <p>{unsupported || current?.caption || "私聊 · 仅双方可见"}</p>
          </div>
          <div className="pc-thread-actions">
            {threadActions}
            <button
              className={`icon-button ${showSearch ? "selected" : ""}`}
              aria-label="搜索当前会话消息"
              title="搜索当前会话 · Ctrl F"
              onClick={() => setShowSearch(!showSearch)}
            >
              <Search size={19} />
            </button>
            <button
              className={`icon-button ${details ? "selected" : ""}`}
              aria-label={details ? "收起会话详情" : "展开会话详情"}
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
              onChange={(e) => setSearch(e.target.value)}
              placeholder="搜索已加载的消息和发言人"
            />
            <span>{search ? `${visible.length} 条结果` : "Ctrl F"}</span>
            <button
              className="icon-button"
              aria-label="关闭消息搜索"
              onClick={() => setShowSearch(false)}
            >
              <X size={16} />
            </button>
          </div>
        )}
        {!ready && (
          <div className="chat-connection-banner">
            <WifiOff size={15} />
            <span>实时连接暂不可用，已加载的消息仍可查看。</span>
            {onReconnect && <button onClick={onReconnect}>重新连接</button>}
          </div>
        )}
        <div
          ref={scroll}
          className="pc-message-scroll"
          onScroll={() => {
            const el = scroll.current;
            atBottom.current =
              el.scrollHeight - el.scrollTop - el.clientHeight < 90;
            if (atBottom.current) {
              setNewBelow(false);
              if (lastId && onRead) onRead(channel, lastId);
            }
          }}
        >
          {hasMore && (
            <button
              className="load-earlier"
              disabled={loadingEarlier}
              onClick={async () => {
                const el = scroll.current,
                  h = el.scrollHeight;
                await onLoadEarlier?.();
                requestAnimationFrame(() => {
                  el.scrollTop += el.scrollHeight - h;
                });
              }}
            >
              {loadingEarlier ? "正在加载…" : "查看更早的消息"}
            </button>
          )}
          <div className="pc-date-divider">
            <span>{channel === "public" ? "公共消息记录" : "会话消息"}</span>
          </div>
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
                  : "这里将显示服务器公共聊天。"
              }
            />
          ) : (
            visible.map((m, i) => {
              const mine = m.sender === user.id || m.mine,
                name =
                  m.senderName ||
                  contacts.find((c) => c.id === m.sender)?.name ||
                  m.sender;
              return (
                <div
                  className={`pc-message ${mine ? "mine" : ""} ${m.status === "failed" || m.status === "unknown" ? "message-failed" : ""}`}
                  key={m.id}
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
                    <div className="pc-message-meta">
                      <strong>
                        {m.sender === "assistant" ? "小祥" : name}
                      </strong>
                      {m.source && <span>{m.source}</span>}
                      <time>{m.time}</time>
                    </div>
                    <div className="bubble">
                      {m.reply && (
                        <blockquote>
                          {m.reply.text || "引用消息暂不可见"}
                        </blockquote>
                      )}
                      {m.forwarded && <blockquote>转发自 {m.forwarded.senderName || "玩家"}<br />{m.forwarded.availability === "UNAVAILABLE" ? "原消息暂不可见" : m.forwarded.text}</blockquote>}
                      {m.text}
                      {m.sources?.length > 0 && <div className="ai-sources">{m.sources.map((source, n) => <a key={`${source.url}:${n}`} href={source.url} target="_blank" rel="noopener noreferrer">{source.origin === "provider_text" ? "正文链接" : "来源"} · {source.title}</a>)}</div>}
                      {m.searchUsed && <small className="muted">已使用联网搜索</small>}
                      {m.sender === "assistant" && ["failed", "unknown", "incomplete"].includes(m.aiStatus) && <p className="composer-error">这次回复未完成，可以开始新对话。</p>}
                    </div>
                    {m.status && (
                      <div className="message-delivery">
                        {m.status === "sending" ? (
                          <>
                            <LoaderCircle className="spin" size={11} />
                            发送中
                          </>
                        ) : m.status === "failed" || m.status === "unknown" ? (
                          <>
                            <span>{m.error || "结果暂未确认"}</span>
                            <button
                              onClick={async () => {
                                try {
                                  await onRetry?.(m);
                                  if (text.trim() === m.text) {
                                    setText("");
                                    lastAttempt.current = null;
                                  }
                                  setError("");
                                } catch (e) {
                                  setError(e.message);
                                }
                              }}
                            >
                              使用原消息重试
                            </button>
                          </>
                        ) : (
                          <>
                            <Check size={12} />
                            已发送
                          </>
                        )}
                      </div>
                    )}
                  </div>
                  <div className="pc-message-tools">
                    <button
                      aria-label={`复制消息 ${m.id}`}
                      onClick={() => copy(m)}
                    >
                      {copied === m.id ? (
                        <Check size={15} />
                      ) : (
                        <Copy size={15} />
                      )}
                    </button>
                    {canReply && !m.id.startsWith("pending:") && (
                      <button
                        aria-label={`回复消息 ${m.id}`}
                        onClick={() => {
                          setReply(m);
                          input.current?.focus();
                        }}
                      >
                        <Reply size={15} />
                      </button>
                    )}
                    {onForwardMessage && !m.id.startsWith("pending:") && <button aria-label={`转发消息 ${m.id}`} onClick={() => onForwardMessage(m)}><Forward size={15} /></button>}
                  </div>
                </div>
              );
            })
          )}
          <div ref={end} />
        </div>
        {newBelow && (
          <button className="new-message-button" onClick={toBottom}>
            <ArrowDown size={15} />
            回到最新消息
          </button>
        )}
        <form
          className="pc-composer"
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
                <span>{reply.text}</span>
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
          <div className="composer-toolbar">
            <div>
              <button
                type="button"
                aria-label="选择表情"
                aria-expanded={emoji}
                onClick={() => setEmoji(!emoji)}
              >
                <Smile size={19} />
              </button>
              <span>
                {unsupported
                  ? "暂不可发送"
                  : isLive
                    ? `${channel === "public" ? "公共频道" : channel === "assistant" ? "AI 会话" : "私聊"} · 最多 ${maxLength} 字`
                    : "Enter 发送，Shift + Enter 换行"}
              </span>
            </div>
            <span>
              {text.length} / {maxLength}
            </span>
          </div>
          {emoji && (
            <div className="emoji-picker">
              {[
                "🌿",
                "✨",
                "👍",
                "😊",
                "🎉",
                "🏡",
                "💙",
                "🌙",
                "⛏️",
                "🌸",
                "谢谢！",
                "收到！",
              ].map((e) => (
                <button
                  key={e}
                  type="button"
                  onClick={() => {
                    setText((t) => (t + e).slice(0, maxLength));
                    setEmoji(false);
                    input.current?.focus();
                  }}
                >
                  {e}
                </button>
              ))}
            </div>
          )}
          <textarea
            ref={input}
            aria-label="消息内容"
            placeholder={
              unsupported ? "等待后端接入此会话" : `发送给 ${current?.name}…`
            }
            disabled={Boolean(unsupported)}
            value={text}
            maxLength={maxLength}
            rows={2}
            onChange={(e) => setText(e.target.value)}
            onBlur={() => onDraft?.(channel, text)}
            onKeyDown={(e) => {
              if (
                e.key === "Enter" &&
                !e.shiftKey &&
                !e.nativeEvent.isComposing &&
                e.keyCode !== 229
              ) {
                e.preventDefault();
                send();
              }
            }}
          />
          {error && (
            <p className="composer-error" role="alert">
              {error}
            </p>
          )}
          <div className="composer-bottom">
            <span>
              <ShieldCheckSmall />
              通过 Deuterium ID 验证身份
            </span>
            <Button
              type="submit"
              disabled={
                sending[channel] || !text.trim() || !ready || Boolean(unsupported)
              }
            >
              {sending[channel] ? (
                <LoaderCircle className="spin" size={15} />
              ) : (
                <Send size={15} />
              )}
              发送
            </Button>
          </div>
        </form>
      </section>
      {details && (
        <aside className="pc-details">
          <header>
            <h3>会话详情</h3>
            <button
              className="icon-button"
              aria-label="关闭会话详情"
              onClick={() => setDetails(false)}
            >
              <X size={17} />
            </button>
          </header>
          <div className="chat-room-profile">
            {active === "public" ? (
              <span className="room-hash">
                <Hash size={34} />
              </span>
            ) : (
              <Avatar user={current?.player || current?.name} size="large" />
            )}
            <h3>{current?.name}</h3>
            <p>
              {active === "public"
                ? "在同一个世界里，分享每一天。"
                : current?.bio || "熟悉的朋友，随时相连。"}
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
                <dd>{ready ? "可用" : "重连中"}</dd>
              </div>
            </dl>
          </div>
          {active === "public" && (
            <div className="pc-detail-section">
              <h4>{isLive ? "最近发言的玩家" : "社区好友"}</h4>
              {(isLive
                ? [
                    ...new Map(
                      all.map((m) => [
                        m.sender,
                        { id: m.sender, name: m.senderName || m.sender, player: m.senderProfile },
                      ]),
                    ).values(),
                  ].slice(-8)
                : contacts.filter(
                    (c) => !["public", "assistant"].includes(c.id),
                  )
              ).map((p) => (
                <button
                  className="detail-member"
                  key={p.id}
                  onClick={() => onProfile?.(p.id)}
                >
                  <Avatar user={p.player || p.name} />
                  <span>{p.name}</span>
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
function ShieldCheckSmall() {
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
    >
      <path d="M12 3 4 6v6c0 5 8 9 8 9s8-4 8-9V6Z" />
      <path d="m8 12 3 3 5-6" />
    </svg>
  );
}

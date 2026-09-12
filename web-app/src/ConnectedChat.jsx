import React, { useEffect, useRef, useState } from "react";
import Messenger from "./Messenger.jsx";
import { Button, Field, Modal, Avatar, Empty } from "./components.jsx";
import { createChatConnection, normalizeMessage, mergeMessages, recoveryCursor } from "./api.js";
import { id } from "./format.js";
import useAiConversation from "./AiConversation.jsx";

const pendingKey = (user) => `deuterium-chat-pending:${user.userId}`;
function readPending(user) {
  try { const value = JSON.parse(sessionStorage.getItem(pendingKey(user))); return value && typeof value === "object" ? value : {}; }
  catch { return {}; }
}
function pendingMessage(user, entry) {
  return { id: `pending:${entry.clientMessageId}`, channel: entry.channel, text: entry.text, sender: user.playerRef, senderName: user.gameId, mine: true, sentAt: entry.sentAt, time: new Date(entry.sentAt).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }), status: "unknown", clientMessageId: entry.clientMessageId, reply: entry.reply, forwardRequest: entry.forward };
}

export default function ConnectedChat({ client, user, onProfile, onUnavailable, requestedConversation }) {
  const pending = useRef(readPending(user)),
    [messages, setMessages] = useState(() => Object.values(pending.current).filter((entry) => entry.clientMessageId && entry.channel).map((entry) => pendingMessage(user, entry))),
    [conversations, setConversations] = useState([]), [activeConversation, setActiveConversation] = useState(requestedConversation || "public"),
    [publicConnection, setPublicConnection] = useState("connecting"), [directConnection, setDirectConnection] = useState("connecting"),
    [cursors, setCursors] = useState({}), [error, setError] = useState(""), [loadingEarlier, setLoadingEarlier] = useState(false),
    [modal, setModal] = useState(null), [drafts, setDrafts] = useState({}), [readPositions, setReadPositions] = useState({});
  const transport = useRef(null), alive = useRef(true), messagesRef = useRef(messages), cursorsRef = useRef(cursors),
    selectedRef = useRef(activeConversation), loading = useRef(new Set()), readRef = useRef({});
  const conversationsRef = useRef(conversations), erasedChannels = useRef(new Set()); conversationsRef.current = conversations;
  messagesRef.current = messages; cursorsRef.current = cursors; selectedRef.current = activeConversation;
  const ai = useAiConversation({ client, user, active: activeConversation === "assistant" });
  const savePending = () => sessionStorage.setItem(pendingKey(user), JSON.stringify(pending.current));
  const forgetChannels = (channels) => {
    if (!channels.length) return;
    channels.forEach((channel) => erasedChannels.current.add(channel));
    setConversations((old) => old.filter((item) => !erasedChannels.current.has(item.conversationId)));
    setMessages((old) => old.filter((message) => !erasedChannels.current.has(message.channel)));
    setDrafts((old) => Object.fromEntries(Object.entries(old).filter(([key]) => !erasedChannels.current.has(key.slice(user.playerRef.length + 1)))));
    Object.entries(pending.current).forEach(([key, entry]) => { if (erasedChannels.current.has(entry.channel)) delete pending.current[key]; }); savePending();
    if (erasedChannels.current.has(selectedRef.current)) setActiveConversation("public");
  };
  const cleanMessage = (message) => {
    const quote = (value) => value && client.erasedPlayerRefs.has(value.sender) ? { ...value, text: "", senderName: "已注销用户", availability: "UNAVAILABLE" } : value;
    return { ...message, reply: quote(message.reply), forwarded: quote(message.forwarded), text: client.erasedPlayerRefs.has(message.forwarded?.sender) ? "原消息不可见" : message.text };
  };
  const integrate = (incoming) => {
    incoming = incoming.filter((message) => !client.erasedPlayerRefs.has(message.sender) && !erasedChannels.current.has(message.channel)).map(cleanMessage);
    for (const message of incoming) if (message.mine && message.clientMessageId && pending.current[message.clientMessageId]) { delete pending.current[message.clientMessageId]; savePending(); }
    setMessages((old) => mergeMessages(old, incoming));
  };
  const loadConversations = async () => {
    if (loading.current.has("conversations")) return;
    loading.current.add("conversations");
    const knownBefore = new Set([...conversationsRef.current.map((item) => item.conversationId), ...messagesRef.current.map((message) => message.channel), ...Object.values(pending.current).map((entry) => entry.channel)]);
    try {
      const r = await client.conversations();
      if (alive.current && Array.isArray(r.data)) {
        const current = r.data.filter((item) => !client.erasedPlayerRefs.has(item.otherPlayer.playerRef));
        if (r.page?.hasMore === false) { const ids = new Set(current.map((item) => item.conversationId)); forgetChannels([...knownBefore].filter((id) => id && id !== "public" && id !== "assistant" && !ids.has(id))); }
        setConversations(current);
      }
    } catch (e) { if (alive.current) setError(e.message); }
    finally { loading.current.delete("conversations"); }
  };
  const loadMessages = async (channel, more = false) => {
    if (channel === "assistant" || loading.current.has(channel)) return;
    loading.current.add(channel); if (more) setLoadingEarlier(true);
    try {
      const r = channel === "public" ? await client.history(more ? cursorsRef.current.public : undefined) : await client.directMessages(channel, more ? cursorsRef.current[channel] : undefined);
      if (!alive.current) return;
      const values = channel === "public" ? r.data.messages : r.data;
      if (!Array.isArray(values)) throw new Error("消息列表格式不正确，请重试。");
      const incoming = values.slice().reverse().map((m) => normalizeMessage(m, user)), previous = messagesRef.current.filter((m) => m.channel === channel);
      integrate(incoming);
      setCursors((old) => ({ ...old, [channel]: more ? r.page?.nextCursor : recoveryCursor(previous, incoming, old[channel], r.page?.nextCursor) }));
      if (channel !== "public" && selectedRef.current === channel) setDirectConnection("connected");
      setError("");
    } catch (e) {
      if (alive.current && e.status === 404 && channel !== "public" && channel !== "assistant") { forgetChannels([channel]); return; }
      if (alive.current) { setError(e.message); if (channel !== "public" && selectedRef.current === channel) setDirectConnection("disconnected"); }
    } finally { loading.current.delete(channel); if (alive.current && more) setLoadingEarlier(false); }
  };
  useEffect(() => {
    alive.current = true; loadConversations(); loadMessages("public");
    transport.current = createChatConnection({ client, onState: (value) => { if (alive.current) setPublicConnection(value); }, onMessage: (value) => { if (alive.current) integrate([normalizeMessage(value, user)]); }, onUnauthorized: () => client.onUnauthorized(), onRecovery: () => loadMessages("public") });
    const timer = setInterval(() => { if (!document.hidden) { loadConversations(); const selected = selectedRef.current; if (selected !== "public" && selected !== "assistant") loadMessages(selected); } }, 4000);
    return () => { alive.current = false; clearInterval(timer); transport.current?.close(); };
  }, [user.userId]);
  useEffect(() => {
    if (activeConversation !== "public" && activeConversation !== "assistant") { setDirectConnection("connecting"); loadMessages(activeConversation); }
  }, [activeConversation]);
  useEffect(() => client.onAccountDeletions((refs) => {
    forgetChannels(conversationsRef.current.filter((item) => refs.has(item.otherPlayer.playerRef)).map((item) => item.conversationId));
    setMessages((old) => old.filter((message) => !refs.has(message.sender) && !erasedChannels.current.has(message.channel)).map(cleanMessage));
  }), [client, user.userId]);
  useEffect(() => { if (requestedConversation) { setActiveConversation(requestedConversation); loadConversations(); } }, [requestedConversation]);
  const submitEntry = async (entry) => {
    const clientMessageId = entry.clientMessageId, pendingId = `pending:${clientMessageId}`;
    pending.current[clientMessageId] = entry; savePending();
    setMessages((old) => mergeMessages(old, [{ ...pendingMessage(user, entry), status: "sending" }]));
    try {
      let authoritative;
      if (entry.channel === "public") {
        const r = await transport.current.send(entry.text, clientMessageId, { ...(entry.reply?.id ? { replyToMessageId: entry.reply.id } : {}) });
        authoritative = { ...pendingMessage(user, entry), id: r.messageId, status: "sent" };
      } else {
        const r = entry.forward ? await client.forwardDirect(entry.channel, { clientMessageId, ...entry.forward }) : await client.sendDirect(entry.channel, { clientMessageId, content: entry.text, ...(entry.reply?.id ? { replyToMessageId: entry.reply.id } : {}) });
        if (!r.data?.messageId) throw new Error("消息结果尚未确认，请使用原消息重试。");
        authoritative = normalizeMessage(r.data, user);
      }
      if (alive.current) { delete pending.current[clientMessageId]; savePending(); setMessages((old) => mergeMessages(old.filter((m) => m.id !== pendingId), erasedChannels.current.has(authoritative.channel) ? [] : [cleanMessage(authoritative)])); loadConversations(); }
      return true;
    } catch (e) {
      if (alive.current) setMessages((old) => old.map((message) => message.id === pendingId ? { ...message, status: e.status >= 400 && e.status < 500 ? "failed" : "unknown", error: e.message } : message));
      throw e;
    }
  };
  const send = ({ channel, text, reply, clientMessageId }) => channel === "assistant" ? ai.send({ text, clientMessageId }) : submitEntry(pending.current[clientMessageId] || Object.values(pending.current).find((entry) => entry.channel === channel && entry.text === text && entry.reply?.id === reply?.id && !entry.forward) || { channel, text, reply, clientMessageId, sentAt: new Date().toISOString() });
  const retry = (message) => submitEntry(pending.current[message.clientMessageId] || { channel: message.channel, text: message.text, reply: message.reply, forward: message.forwardRequest, clientMessageId: message.clientMessageId, sentAt: message.sentAt });
  const markRead = async (channel, messageID) => {
    if (messageID.startsWith("pending:") || readRef.current[channel] === messageID) return;
    readRef.current[channel] = messageID;
    setReadPositions((old) => ({ ...old, [`${user.playerRef}:${channel}`]: messageID }));
    if (channel === "public" || channel === "assistant") return;
    try { await client.readDirect(channel, { clientRequestId: id(), lastReadMessageId: messageID }); if (alive.current) setConversations((old) => old.map((c) => c.conversationId === channel ? { ...c, unreadCount: 0 } : c)); }
    catch (e) { if (alive.current) { delete readRef.current[channel]; setError(e.message); } }
  };
  const contacts = [{ id: "public", channel: "public", name: "公共聊天", caption: "服务器公共消息" }, ...conversations.map((c) => ({ id: c.conversationId, channel: c.conversationId, name: c.otherPlayer.gameId, player: c.otherPlayer, caption: c.lastMessage?.content || c.otherPlayer.bio || "开始一段对话", unreadCount: c.unreadCount })), { id: "assistant", channel: "assistant", name: "小祥 AI", caption: ai.caption }];
  const [profiles, setProfiles] = useState({}), profileRequests = useRef(new Map());
  useEffect(() => {
    // Public chat intentionally carries a small identity. Resolve each player's
    // current signed avatar once, separately from message loading and sending.
    const refs = [...new Set(messages.filter((m) => m.sender && m.sender !== "assistant" && m.senderProfile?.registered !== false).map((m) => m.sender))].filter((ref) => (profileRequests.current.get(ref) || 0) < Date.now() - 5 * 60 * 1000);
    refs.forEach((ref) => profileRequests.current.set(ref, Date.now()));
    (async () => {
      for (let i = 0; i < refs.length && alive.current; i += 4) {
        const batch = refs.slice(i, i + 4);
        const results = await Promise.allSettled(batch.map((ref) => client.profile(ref)));
        if (alive.current) setProfiles((old) => ({ ...old, ...Object.fromEntries(results.flatMap((result, n) => result.status === "fulfilled" ? [[batch[n], result.value.data]] : [])) }));
      }
    })();
  }, [messages, client]);
  const startConversation = async (player, forward) => {
    const r = await client.createConversation({ clientRequestId: id(), otherPlayerRef: player.playerRef });
    const conversation = r.data;
    const previousForward = forward && pending.current[forward.clientMessageId];
    if (previousForward && previousForward.channel !== conversation.conversationId) throw new Error("上一条转发仍待核对，请先回到原会话确认结果。");
    setConversations((old) => [conversation, ...old.filter((c) => c.conversationId !== conversation.conversationId)]);
    setActiveConversation(conversation.conversationId);
    if (forward) await submitEntry({ channel: conversation.conversationId, clientMessageId: forward.clientMessageId, text: forward.message.text, sentAt: new Date().toISOString(), forward: { sourceMessageId: forward.message.id, ...(forward.message.channel !== "public" ? { sourceConversationId: forward.message.channel } : {}) } });
    setModal(null);
  };
  return <>
    {(activeConversation === "assistant" ? ai.error : error) && <div className="connected-history-error" role="alert"><span>{activeConversation === "assistant" ? ai.error : error}</span><button onClick={() => { if (activeConversation === "assistant") ai.load(); else { loadConversations(); loadMessages(activeConversation); } }}>重新连接</button></div>}
    <Messenger user={{ ...user, id: user.playerRef, name: user.gameId }} contacts={contacts} messages={[...messages.map((message) => ({ ...message, senderProfile: profiles[message.sender] || message.senderProfile })), ...ai.messages]} onSend={send} onRetry={retry}
      activeConversation={activeConversation} onConversationChange={setActiveConversation} onNewConversation={() => setModal({ type: "new" })}
      canReply={activeConversation !== "assistant"} onForwardMessage={activeConversation === "assistant" ? undefined : (message) => setModal({ type: "forward", message, clientMessageId: id() })}
      connection={activeConversation === "public" ? publicConnection : activeConversation === "assistant" ? ai.connection : directConnection} onReconnect={() => activeConversation === "public" ? transport.current?.reconnect() : activeConversation === "assistant" ? ai.load() : loadMessages(activeConversation)}
      drafts={drafts} onDraft={(channel, text) => setDrafts((old) => ({ ...old, [`${user.playerRef}:${channel}`]: text }))} readPositions={readPositions} onRead={markRead}
      hasMore={Boolean(cursors[activeConversation])} onLoadEarlier={() => loadMessages(activeConversation, true)} loadingEarlier={loadingEarlier} maxLength={activeConversation === "assistant" ? 2000 : 256} threadActions={activeConversation === "assistant" ? ai.actions : null} conversationStatus={activeConversation === "assistant" ? ai.status : ""}
      onProfile={(playerRef) => { const m = [...messages,...ai.messages].find((m) => m.sender === playerRef); if (m) onProfile({ id: playerRef, name: m.senderName }); }} onAnnouncement={onUnavailable} onAnnouncements={onUnavailable} />
    {modal && <Modal title={modal.type === "forward" ? "转发给玩家" : "发起私聊"} close={() => setModal(null)}><PlayerPicker client={client} user={user} onChoose={(player) => startConversation(player, modal.type === "forward" ? modal : null)} /></Modal>}
    {ai.plansModal}
  </>;
}

export function PlayerPicker({ client, user, onChoose }) {
  const [query, setQuery] = useState(""), [players, setPlayers] = useState([]), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const load = async () => { setBusy(true); setError(""); try { const r = await client.directory(query); setPlayers(r.data.players.filter((p) => p.playerRef !== user.playerRef)); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  useEffect(() => { load(); }, []);
  return <><form onSubmit={(e) => { e.preventDefault(); load(); }}><Field label="查找玩家" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="输入游戏 ID 或 QQ" maxLength={80} /><Button secondary type="submit" disabled={busy}>查找</Button></form>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="button-row">{players.map((player) => <Button key={player.playerRef} secondary disabled={busy} onClick={async () => { setBusy(true); setError(""); try { await onChoose(player); } catch (e) { setError(e.message); } finally { setBusy(false); } }}><Avatar user={player} />{player.gameId}</Button>)}</div>
    {!busy && !players.length && <Empty title="没有找到玩家" text="试试游戏 ID 或 QQ。" />}</>;
}

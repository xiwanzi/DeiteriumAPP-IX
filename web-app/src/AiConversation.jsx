import React, { useEffect, useRef, useState } from "react";
import { Button, Badge, Modal } from "./components.jsx";
import { credit } from "./format.js";
import { aiSources } from "./ai-stream.js";

function normalizeAi(message, user) {
  const mine = message.role === "user";
  return { id: message.messageId, channel: "assistant", text: message.content || "", sender: mine ? user.playerRef : "assistant", senderName: mine ? user.gameId : "小祥", mine, sentAt: message.createdAt, time: new Date(message.createdAt).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }), sources: aiSources(message.sources), searchUsed: Boolean(message.searchUsed), aiStatus: message.status, status: mine ? "sent" : undefined };
}
const pendingKey = (user) => `deuterium-ai-pending:${user.userId}`;
function loadPending(user) { try { return JSON.parse(sessionStorage.getItem(pendingKey(user))); } catch { return null; } }

export default function useAiConversation({ client, user, active }) {
  const [messages, setMessages] = useState([]), [account, setAccount] = useState(null), [plans, setPlans] = useState([]),
    [error, setError] = useState(""), [busy, setBusy] = useState(false), [connection, setConnection] = useState("connecting"),
    [pending, setPending] = useState(() => loadPending(user)), [showPlans, setShowPlans] = useState(false), [showOptions, setShowOptions] = useState(false), [status, setStatus] = useState("");
  const pendingRef = useRef(pending), alive = useRef(true), busyRef = useRef(false), loading = useRef(false), abort = useRef(null), messagesRef = useRef(messages);
  messagesRef.current = messages;
  const savePending = (value) => { pendingRef.current = value; if (value) sessionStorage.setItem(pendingKey(user), JSON.stringify(value)); else sessionStorage.removeItem(pendingKey(user)); setPending(value); };
  const load = async () => {
    if (loading.current || busyRef.current) return; loading.current = true;
    try {
      const [me, history, catalogue] = await Promise.all([client.request("/api/v1/ai/me"), client.request("/api/v1/ai/messages?limit=100"), client.request("/api/v1/ai/plans")]);
      if (!alive.current) return;
      setAccount(me.data); setPlans(catalogue.data.plans || []); setMessages((history.data.messages || []).map((message) => normalizeAi(message, user))); setConnection("connected"); setError("");
      const remote = me.data.pendingRequest;
      if (remote?.clientMessageId) savePending({ ...remote, content: remote.content, clientMessageId: remote.clientMessageId });
      else if (pendingRef.current?.assistantMessageId) {
        const completed = history.data.messages?.find((message) => message.messageId === pendingRef.current.assistantMessageId && message.status === "completed");
        if (completed) savePending(null);
      }
    } catch (e) { if (alive.current) { setError(e.message); setConnection("disconnected"); } }
    finally { loading.current = false; }
  };
  useEffect(() => { alive.current = true; return () => { alive.current = false; abort.current?.abort(); }; }, [user.userId]);
  useEffect(() => { if (!active) return; load(); const timer = setInterval(() => { if (!document.hidden) load(); }, 5000); return () => clearInterval(timer); }, [active]);
  const send = async ({ text, clientMessageId }) => {
    if (busyRef.current) throw new Error("小祥正在回复，请稍候。");
    if (text.trim() === "/new") { if (await reset()) return true; throw new Error("暂时无法开始新对话，请稍后重试。"); }
    let entry = pendingRef.current;
    if (!entry || (entry.content !== text && !["pending", "streaming"].includes(entry.status))) entry = { content: text, clientMessageId, status: "pending", createdAt: new Date().toISOString() };
    if (entry.content !== text) throw new Error("上一条回复仍在进行中，请先继续查看。");
    busyRef.current = true; setBusy(true); setError(""); setStatus("正在回复…"); savePending(entry); abort.current = new AbortController();
    try {
      await client.streamAi({ clientMessageId: entry.clientMessageId, content: entry.content }, (event, payload) => {
        if (!alive.current) return;
        if (event === "meta") {
          entry = { ...entry, conversationId: payload.conversationId, userMessageId: payload.userMessageId, assistantMessageId: payload.assistantMessageId, status: "streaming" }; savePending(entry);
          const userMessage = payload.userMessage || { messageId: payload.userMessageId, role: "user", content: entry.content, createdAt: entry.createdAt, status: "completed" };
          const assistant = { messageId: payload.assistantMessageId, role: "assistant", content: "", createdAt: entry.createdAt, status: "streaming" };
          setMessages((old) => [...old.filter((message) => message.id !== payload.userMessageId && message.id !== payload.assistantMessageId), normalizeAi(userMessage, user), normalizeAi(assistant, user)]);
          if (payload.quota) setAccount((old) => ({ ...old, quota: payload.quota }));
        } else if (event === "delta") {
          setMessages((old) => old.map((message) => message.id === entry.assistantMessageId ? { ...message, text: message.text + (payload.content || "") } : message));
        } else if (event === "sources") {
          setMessages((old) => old.map((message) => message.id === entry.assistantMessageId ? { ...message, sources: aiSources(payload.sources) } : message));
        } else if (event === "status") {
          setStatus(["searching", "web_search"].includes(payload.status) ? "正在查找资料…" : "正在回复…");
        } else if (event === "done") {
          if (!payload.message?.messageId) throw new Error("AI 回复结果尚未确认。");
          setMessages((old) => [...old.filter((message) => message.id !== payload.message.messageId), normalizeAi(payload.message, user)]);
          if (payload.quota) setAccount((old) => ({ ...old, quota: payload.quota }));
          savePending(null); setStatus("");
        }
      }, abort.current.signal);
      return true;
    } catch (e) {
      if (alive.current) { setError(e.message); setStatus(""); if (e.status >= 400 && e.status < 500 && !entry.assistantMessageId) savePending(null); }
      throw e;
    } finally { busyRef.current = false; if (alive.current) { setBusy(false); load(); } }
  };
  const resume = async () => { const current = pendingRef.current; if (!current) return; try { await send({ text: current.content, clientMessageId: current.clientMessageId }); } catch {} };
  const reset = async () => { if (busyRef.current) return false; setError(""); setBusy(true); try { await client.request("/api/v1/ai/conversation/reset", { method: "POST", body: {} }); savePending(null); setMessages([]); await load(); return true; } catch (e) { setError(e.message); return false; } finally { setBusy(false); } };
  const quota = account?.quota, caption = quota ? quota.unlimited ? "管理员 · 不限次数 · 可联网搜索" : `${quota.windowHours || 24} 小时额度 · ${quota.remaining} / ${quota.limit} · 可联网搜索` : "DeepSeek V4 Flash · 可联网搜索";
  return { messages, send, load, busy, connection, caption, error, status, pending,
    actions: <Button secondary onClick={() => setShowOptions(true)}>AI 选项</Button>,
    plansModal: <>{showOptions && <Modal title="AI 会话" close={() => setShowOptions(false)}><p>{caption}</p><div className="button-row">{pending && !busy && <Button onClick={() => { setShowOptions(false); resume(); }}>继续查看</Button>}<Button secondary disabled={busy} onClick={async () => { if (await reset()) setShowOptions(false); }}>新对话</Button><Button secondary onClick={() => { setShowOptions(false); setShowPlans(true); }}>额度与计划</Button></div></Modal>}{showPlans && <Modal title="AI 额度与计划" close={() => setShowPlans(false)}><p>当前使用 DeepSeek V4 Flash。根据问题需要使用联网搜索。</p>{quota && <div className="notice-box">{quota.unlimited ? "管理员账号不限制请求次数。" : `免费额度：${quota.limit} 次 / ${quota.windowHours} 小时，剩余 ${quota.remaining} 次。`}{!quota.unlimited && quota.resetsAt && <p>下次重置：{new Date(quota.resetsAt).toLocaleString("zh-CN")}</p>}</div>}
      <div className="foundation-grid">{plans.map((plan) => <div className="foundation-card" key={plan.planId}><h3>{plan.name}</h3><p>{plan.description}</p><strong className="price">{credit(plan.price)}<small>信用点</small></strong><p><Badge tone="neutral">{String(plan.price) === "0.00" || String(plan.price) === "0" ? "免费计划" : "暂未开放购买"}</Badge></p><Button disabled>暂不提供购买</Button></div>)}</div>
    </Modal>}</>,
  };
}

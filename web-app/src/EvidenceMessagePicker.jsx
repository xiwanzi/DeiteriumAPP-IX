import React, { useEffect, useState } from "react";
import { Button, Field } from "./components.jsx";

export default function EvidenceMessagePicker({ client, selected, onChange }) {
  const [open, setOpen] = useState(false), [conversations, setConversations] = useState([]), [channel, setChannel] = useState("public"), [messages, setMessages] = useState([]), [busy, setBusy] = useState(false), [error, setError] = useState("");
  useEffect(() => { if (!open) return; let alive = true; client.request("/api/v1/chat/conversations?limit=100").then((r) => { if (alive) setConversations(Array.isArray(r.data) ? r.data : []); }).catch((e) => { if (alive) setError(e.message); }); return () => { alive = false; }; }, [open]);
  useEffect(() => {
    if (!open) return; let alive = true; setBusy(true); setError(""); setMessages([]);
    client.request(channel === "public" ? "/api/v1/chat/messages?limit=100" : `/api/v1/chat/conversations/${encodeURIComponent(channel)}/messages?limit=100`).then((r) => { if (alive) setMessages(Array.isArray(r.data) ? r.data : r.data.messages || []); }).catch((e) => { if (alive) setError(e.message); }).finally(() => { if (alive) setBusy(false); }); return () => { alive = false; };
  }, [open, channel]);
  return <div style={{ margin: "16px 0" }}><Button secondary onClick={() => setOpen(!open)}>{open ? "收起聊天资料" : "选择相关聊天"}{selected.length > 0 && `（已选 ${selected.length} 条）`}</Button>
    {selected.length > 0 && <div className="panel" style={{ padding: 12, marginTop: 10 }}>{selected.map((message) => <div key={message.messageId} style={{ marginBottom: 8 }}><p style={{ whiteSpace: "pre-wrap" }}>{message.sender?.displayName || message.sender?.gameId}：{message.content}</p><Button secondary onClick={() => onChange(selected.filter((item) => item.messageId !== message.messageId))}>移除此条</Button></div>)}</div>}
    {open && <div className="panel" style={{ padding: 16, marginTop: 10 }}><p className="notice-box">提交后，所选消息的正文快照会随案件提供给交易双方及处理管理员。请选择与本次争议相关的内容。</p><Field label="资料来源"><select value={channel} onChange={(e) => setChannel(e.target.value)}><option value="public">公共聊天</option>{conversations.map((item) => <option key={item.conversationId} value={item.conversationId}>与 {item.otherPlayer?.gameId || item.otherPlayer?.displayName} 的私聊</option>)}</select></Field>
      {error && <p className="auth-error" role="alert">{error}</p>}{busy && <p role="status">正在读取可访问的消息…</p>}
      <div style={{ maxHeight: 300, overflowY: "auto" }}>{messages.map((message) => <label key={message.messageId} className="checkbox-row" style={{ alignItems: "start", margin: "12px 0" }}><input type="checkbox" checked={selected.some((item) => item.messageId === message.messageId)} disabled={selected.length >= 20 && !selected.some((item) => item.messageId === message.messageId)} onChange={(e) => onChange(e.target.checked ? [...selected, message] : selected.filter((item) => item.messageId !== message.messageId))} /><span style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}><strong>{message.sender?.displayName || message.sender?.gameId}</strong> · {new Date(message.sentAt).toLocaleString("zh-CN")}<br />{message.content}</span></label>)}</div>{!busy && !error && !messages.length && <p className="muted">没有可以选择的消息。</p>}
    </div>}
  </div>;
}

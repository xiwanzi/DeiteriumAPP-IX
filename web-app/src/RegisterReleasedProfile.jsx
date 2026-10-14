import React, { useRef, useState } from "react";
import { Button, Field } from "./components.jsx";
import { id } from "./format.js";

export default function RegisterReleasedProfile({ client, onRegistered }) {
  const [gameId, setGameId] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState(""), [uncertain, setUncertain] = useState(false);
  const pending = useRef(null), saving = useRef(false);
  const save = async event => {
    event.preventDefault(); if (saving.current) return;
    const name = gameId.trim();
    if (!pending.current || !uncertain && pending.current.gameId !== name) pending.current = { clientRequestId: id(), gameId: name };
    saving.current = true; setBusy(true); setError("");
    try { await client.request("/api/v1/account/game-profile", { method: "POST", body: pending.current }); await onRegistered(); }
    catch (e) { setError(e.message); setUncertain(e.status >= 500 || ["NETWORK_ERROR", "INVALID_RESPONSE"].includes(e.code)); }
    finally { saving.current = false; setBusy(false); }
  };
  return <form className="auth-form" onSubmit={save}>
    <p>原游戏身份登记已由支持人员解除，你的平台账号和资料仍然保留。请登记自己的正版游戏账号。</p>
    <Field label="Minecraft Java 版游戏 ID" value={gameId} onChange={e => setGameId(e.target.value)} required minLength={3} maxLength={16} pattern="[A-Za-z0-9_]{3,16}" autoComplete="off" disabled={busy || uncertain} />
    <p className="muted">登记后可等服务器开放，再进入游戏完成身份验证。</p>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <Button type="submit" disabled={busy}>{busy ? "正在登记…" : uncertain ? "确认登记结果" : "登记游戏身份"}</Button>
  </form>;
}

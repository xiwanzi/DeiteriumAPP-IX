import React, { useEffect, useRef, useState } from "react";
import { Monitor, Smartphone, RefreshCw } from "lucide-react";
import { Badge, Button } from "./components.jsx";
import "./account-sessions.css";

export default function DeviceSessions({ client }) {
  const [items, setItems] = useState(null), [error, setError] = useState("");
  const [busy, setBusy] = useState(false), [confirmation, setConfirmation] = useState(null), [message, setMessage] = useState("");
  const saving = useRef(false), generation = useRef(0);
  const load = async () => {
    const current = ++generation.current;
    setBusy(true); setError("");
    try { const r = await client.request("/api/v1/account/sessions"); if (current === generation.current) setItems(r.data.items); }
    catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { load(); return () => { generation.current++; }; }, [client]);
  const revoke = async () => {
    if (saving.current || !confirmation) return;
    saving.current = true; setBusy(true); setError("");
    try {
      if (confirmation === "others") await client.request("/api/v1/account/sessions/revoke-others", { method: "POST", body: {} });
      else await client.request(`/api/v1/account/sessions/${encodeURIComponent(confirmation.sessionId)}`, { method: "DELETE" });
      setConfirmation(null); setMessage("已退出所选设备。"); await load();
    } catch (e) { setError(e.message); }
    finally { saving.current = false; setBusy(false); }
  };
  return <section className="device-sessions">
    <p className="muted">查看登录中的设备。如果不再使用某台设备，可以在这里退出。</p>
    <div className="button-row"><Button secondary disabled={busy} onClick={load}><RefreshCw size={16} />刷新</Button><Button secondary disabled={busy || !items?.some(item => !item.current)} onClick={() => setConfirmation("others")}>退出其他设备</Button></div>
    {error && <p role="alert" className="auth-error">{error}</p>}{message && <p role="status" className="muted">{message}</p>}
    {confirmation && <div className="device-confirm" role="alertdialog" aria-label="确认退出设备"><strong>{confirmation === "others" ? "退出其他所有设备？" : `退出 ${confirmation.deviceName || (confirmation.clientKind === "web" ? "网页设备" : "App 设备")}？`}</strong><p>这台设备需要重新登录，当前设备仍可继续使用。</p><div className="button-row"><Button secondary disabled={busy} onClick={() => setConfirmation(null)}>取消</Button><Button danger disabled={busy} onClick={revoke}>{busy ? "正在退出…" : "确认退出"}</Button></div></div>}
    {!items && !error && <p role="status">正在读取登录设备…</p>}
    <ul className="device-list">{items?.map(item => <li key={item.sessionId}>
      <span className="device-icon">{item.clientKind === "web" ? <Monitor size={22} /> : <Smartphone size={22} />}</span>
      <div className="device-details"><strong>{item.deviceName || (item.clientKind === "web" ? "网页设备（旧版）" : "App 设备（旧版）")}</strong><span>最近活动 · {new Date(item.lastSeenAt).toLocaleString("zh-CN")}</span><span>{item.expiresAt ? `有效至 ${new Date(item.expiresAt).toLocaleDateString("zh-CN")}` : "长期登录"}</span></div>
      {item.current ? <Badge tone="sage">当前设备</Badge> : <Button secondary disabled={busy} onClick={() => setConfirmation(item)}>退出</Button>}
    </li>)}</ul>
    {items?.length === 200 && <p className="muted">显示最近使用的 200 个登录记录；“退出其他设备”会处理全部其他登录。</p>}
  </section>;
}

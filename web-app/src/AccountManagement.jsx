import React, { useEffect, useRef, useState } from "react";
import { Search, RefreshCw, ShieldCheck, Users } from "lucide-react";
import { Avatar, Badge, Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { id } from "./format.js";

const actions = { "grant-admin": "授予管理员", "revoke-admin": "撤销管理员", ban: "封禁账号", unban: "解封账号" };
export default function AccountManagement({ client, user }) {
  const [query, setQuery] = useState(""), [filter, setFilter] = useState(""), [search, setSearch] = useState(""), [offset, setOffset] = useState(0);
  const [data, setData] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""), [change, setChange] = useState(null), [reason, setReason] = useState(""), [saving, setSaving] = useState(false), [feedback, setFeedback] = useState("");
  const generation = useRef(0), pending = useRef(null);
  const load = async () => {
    const current = ++generation.current; setBusy(true); setError("");
    try { const r = await client.request(`/api/v1/admin/accounts?${new URLSearchParams({ q: search, status: filter, offset, limit: 30 })}`); if (current === generation.current) setData(r.data); }
    catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { load(); return () => { generation.current++; }; }, [client, search, filter, offset]);
  const save = async (event) => {
    event.preventDefault(); if (saving) return; setSaving(true); setError("");
    const content = { expectedVersion: change.account.version, action: change.action, reason: reason.trim() }, fingerprint = JSON.stringify([change.account.userId, content]);
    if (pending.current?.fingerprint !== fingerprint) pending.current = { fingerprint, body: { ...content, clientRequestId: id() } };
    try { await client.request(`/api/v1/admin/accounts/${encodeURIComponent(change.account.userId)}`, { method: "POST", body: pending.current.body }); setFeedback(`已${actions[change.action]}：${change.account.gameId}`); setChange(null); await load(); }
    catch (e) { setError(e.message); } finally { setSaving(false); }
  };
  return <div className="accounts-workspace">
    <PageHead eyebrow="ACCOUNTS" title="账号与权限" subtitle="查看社区成员，管理账号状态与平台权限。"><Button secondary disabled={busy} onClick={load}><RefreshCw size={16} />刷新</Button></PageHead>
    <div className="settings-banner"><Users size={24} /><div><strong>{data ? `${data.total} 位用户` : "社区用户"}</strong><p>管理员可管理商品、公告、平台介入及系统设置。请谨慎授予。</p></div></div>
    <form className="account-search" onSubmit={(e) => { e.preventDefault(); setOffset(0); setSearch(query.trim()); }}>
      <Field label="查找用户" value={query} maxLength={100} onChange={(e) => setQuery(e.target.value)} placeholder="游戏 ID、QQ 或玩家编号" />
      <Field label="账号状态"><select value={filter} onChange={(e) => { setFilter(e.target.value); setOffset(0); }}><option value="">全部用户</option><option value="active">正常</option><option value="disabled">已封禁</option><option value="locked">已锁定</option></select></Field>
      <Button type="submit" disabled={busy}><Search size={16} />查找</Button>
    </form>
    {feedback && <p className="notice-box" role="status">{feedback}</p>}{error && !change && <p role="alert" className="auth-error">{error}</p>}
    {busy && <p role="status" className="muted">正在读取用户…</p>}
    {data && <div className="panel table-wrap"><table><thead><tr><th>用户</th><th>QQ</th><th>权限</th><th>状态</th><th>管理</th></tr></thead><tbody>{data.items.map((a) => <tr key={a.userId}>
      <td><div className="account-identity"><Avatar user={a.gameId} /><div><strong>{a.gameId}</strong><small>加入于 {new Date(a.createdAt).toLocaleDateString("zh-CN")}</small></div></div></td><td>{a.qq || "—"}</td>
      <td><Badge tone={a.admin ? "blue" : "neutral"}>{a.admin ? "管理员" : "用户"}</Badge></td><td><Badge tone={a.status === "active" ? "sage" : "neutral"}>{{ active: "正常", disabled: "已封禁", locked: "已锁定" }[a.status]}</Badge>{a.banReason && <small className="account-reason">{a.banReason}</small>}</td>
      <td><div className="account-actions">{a.userId === user.userId ? <span className="muted">当前账号</span> : <>{(a.admin || a.status === "active") && <Button secondary disabled={busy} onClick={() => { setChange({ account: a, action: a.admin ? "revoke-admin" : "grant-admin" }); setReason(""); setError(""); }}><ShieldCheck size={15} />{a.admin ? "撤销管理" : "设为管理员"}</Button>}<Button secondary danger={a.status === "active"} disabled={busy} onClick={() => { setChange({ account: a, action: a.status === "active" ? "ban" : "unban" }); setReason(""); setError(""); }}>{a.status === "active" ? "封禁" : "解封"}</Button></>}</div></td>
    </tr>)}</tbody></table>{!data.items.length && <Empty title="没有找到用户" text="试试其他游戏 ID、QQ 或筛选条件。" />}
      <div className="catalog-pagination"><span>{data.total} 位用户 · 第 {Math.floor(offset / 30) + 1} 页</span><div><Button secondary disabled={busy || offset === 0} onClick={() => setOffset(offset - 30)}>上一页</Button><Button secondary disabled={busy || offset + 30 >= data.total} onClick={() => setOffset(offset + 30)}>下一页</Button></div></div>
    </div>}
    {change && <Modal title={actions[change.action]} close={() => { if (!saving) setChange(null); }} dismissOnBackdrop={false}><form className="settings-stack" onSubmit={save}>
      <p>将对 <strong>{change.account.gameId}</strong> 执行“{actions[change.action]}”。</p><p className="muted">{change.action === "ban" ? "封禁后将退出所有设备，无法登录或继续使用账号。" : change.action === "grant-admin" ? "该用户将获得平台管理权限，包括管理其他账号。" : change.action === "unban" ? "解封后用户可以重新登录。" : "该用户将退出所有设备，重新登录后按剩余权限使用。"}</p>
      {change.action === "ban" && <Field label="封禁原因"><textarea required maxLength={500} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} /></Field>}
      {error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button secondary disabled={saving} onClick={() => setChange(null)}>取消</Button><Button danger={change.action === "ban" || change.action === "revoke-admin"} disabled={saving} type="submit">{saving ? "正在保存…" : "确认"}</Button></div>
    </form></Modal>}
  </div>;
}

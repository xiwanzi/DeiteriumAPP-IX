import React, { useEffect, useRef, useState } from "react";
import { Search, RefreshCw, ShieldCheck, Users } from "lucide-react";
import { Avatar, Badge, Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { id } from "./format.js";

const actions = { "grant-admin": "授予超级管理员", "revoke-admin": "撤销超级管理员", "grant-support": "授予管理员", "revoke-support": "撤销管理员", ban: "封禁账号", unban: "解封账号", delete: "永久注销" };
export default function AccountManagement({ client, user }) {
  const [query, setQuery] = useState(""), [filter, setFilter] = useState(""), [search, setSearch] = useState(""), [offset, setOffset] = useState(0);
  const [data, setData] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState(""), [change, setChange] = useState(null), [reason, setReason] = useState(""), [saving, setSaving] = useState(false), [feedback, setFeedback] = useState("");
  const generation = useRef(0), pending = useRef(null);
  const [password, setPassword] = useState("");
  const openChange = (account, action) => { setChange({ account, action }); setReason(""); setPassword(""); setError(""); };
  const closeChange = () => { if (!saving) { setPassword(""); setChange(null); } };
  const load = async () => {
    const current = ++generation.current; setBusy(true); setError("");
    try { const r = await client.request(`/api/v1/admin/accounts?${new URLSearchParams({ q: search, status: filter, offset, limit: 30 })}`); if (current === generation.current) setData(r.data); }
    catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { load(); return () => { generation.current++; }; }, [client, search, filter, offset]);
  useEffect(() => { setPassword(""); setChange(null); pending.current = null; }, [client, user.userId]);
  const save = async (event) => {
    event.preventDefault(); if (saving || !change || (change.action === "delete" && !password)) return; setSaving(true); setError("");
    const deleting = change.action === "delete";
    const content = { expectedVersion: change.account.version, ...(deleting ? {} : { action: change.action, reason: reason.trim() }) }, fingerprint = JSON.stringify([change.account.userId, change.action, content]);
    if (pending.current?.fingerprint !== fingerprint) pending.current = { fingerprint, body: { ...content, clientRequestId: id() } };
    // The password is sent only in this request, never retained in the replay body.
    const body = { ...pending.current.body, ...(deleting ? { password } : {}) }; setPassword("");
    try { await client.request(`/api/v1/admin/accounts/${encodeURIComponent(change.account.userId)}${deleting ? "/delete" : ""}`, { method: "POST", body }); setFeedback(`已${actions[change.action]}：${change.account.gameId}`); setChange(null); await load(); }
    catch (e) { setError(e.message); } finally { setSaving(false); }
  };
  return <div className="accounts-workspace">
    <PageHead eyebrow="ACCOUNTS" title="账号与权限" subtitle="查看社区成员，管理账号状态与平台权限。"><Button secondary disabled={busy} onClick={load}><RefreshCw size={16} />刷新</Button></PageHead>
    <div className="settings-banner"><Users size={24} /><div><strong>{data ? `${data.total} 位用户` : "社区用户"}</strong><p>超级管理员可管理全部功能；管理员仅能使用工单系统。</p></div></div>
    <form className="account-search" onSubmit={(e) => { e.preventDefault(); setOffset(0); setSearch(query.trim()); }}>
      <Field label="查找用户" value={query} maxLength={100} onChange={(e) => setQuery(e.target.value)} placeholder="游戏 ID、邮箱、QQ 或玩家编号" />
      <Field label="账号状态"><select value={filter} onChange={(e) => { setFilter(e.target.value); setOffset(0); }}><option value="">全部用户</option><option value="active">正常</option><option value="disabled">已封禁</option><option value="locked">已锁定</option></select></Field>
      <Button type="submit" disabled={busy}><Search size={16} />查找</Button>
    </form>
    {feedback && <p className="notice-box" role="status">{feedback}</p>}{error && !change && <p role="alert" className="auth-error">{error}</p>}
    {busy && <p role="status" className="muted">正在读取用户…</p>}
    {data && <div className="panel table-wrap"><table><thead><tr><th>用户</th><th>QQ</th><th>权限</th><th>状态</th><th>管理</th></tr></thead><tbody>{data.items.map((a) => <tr key={a.userId}>
      <td><div className="account-identity"><Avatar user={a.gameId} /><div><strong>{a.gameId}</strong><small>{a.gameVerified ? "已验证" : "未验证"}{a.email ? ` · ${a.email}` : ""}</small><small>加入于 {new Date(a.createdAt).toLocaleDateString("zh-CN")}</small></div></div></td><td>{a.qq || "—"}</td>
      <td><Badge tone={a.admin ? "blue" : "neutral"}>{a.admin ? "超级管理员" : a.supportAdmin ? "管理员 · 工单" : "用户"}</Badge></td><td><Badge tone={a.status === "active" ? "sage" : "neutral"}>{{ active: "正常", disabled: "已封禁", locked: "已锁定" }[a.status]}</Badge>{a.banReason && <small className="account-reason">{a.banReason}</small>}</td>
      <td><div className="account-actions">{a.userId === user.userId ? <span className="muted">当前账号</span> : <>{(a.admin || a.status === "active") && <Button secondary disabled={busy || saving} onClick={() => openChange(a, a.admin ? "revoke-admin" : "grant-admin")}><ShieldCheck size={15} />{a.admin ? "撤销超级管理" : "设为超级管理员"}</Button>}{!a.admin && a.status === "active" && <Button secondary disabled={busy || saving} onClick={() => openChange(a, a.supportAdmin ? "revoke-support" : "grant-support")}>{a.supportAdmin ? "撤销工单管理" : "设为管理员"}</Button>}<Button secondary danger={a.status === "active"} disabled={busy || saving} onClick={() => openChange(a, a.status === "active" ? "ban" : "unban")}>{a.status === "active" ? "封禁" : "解封"}</Button><Button secondary danger disabled={busy || saving} onClick={() => openChange(a, "delete")}>永久注销</Button></>}</div></td>
    </tr>)}</tbody></table>{!data.items.length && <Empty title="没有找到用户" text="试试其他游戏 ID、QQ 或筛选条件。" />}
      <div className="catalog-pagination"><span>{data.total} 位用户 · 第 {Math.floor(offset / 30) + 1} 页</span><div><Button secondary disabled={busy || offset === 0} onClick={() => setOffset(offset - 30)}>上一页</Button><Button secondary disabled={busy || offset + 30 >= data.total} onClick={() => setOffset(offset + 30)}>下一页</Button></div></div>
    </div>}
    {change && <Modal title={actions[change.action]} close={closeChange} dismissOnBackdrop={false}><form className="settings-stack" onSubmit={save}>
      <p>将对 <strong>{change.account.gameId}</strong> 执行“{actions[change.action]}”。</p><p className="muted">{change.action === "delete" ? "该操作无法恢复。账号主页、联系人及私聊、市场商品将被清理，历史交易凭据保留为匿名记录。有未完成交易时无法注销。" : change.action === "ban" ? "封禁后将退出所有设备，无法登录或继续使用账号。" : change.action === "grant-support" ? "该用户仅获得工单系统管理权限。" : change.action === "grant-admin" ? "该用户将获得超级管理员权限，包括管理其他账号和系统配置。" : change.action === "unban" ? "解封后用户可以重新登录。" : "该用户将退出所有设备，重新登录后按剩余权限使用。"}</p>
      {change.action === "delete" && <Field label="当前管理员密码" hint={`请输入当前登录账户 ${user.gameId} 的密码进行确认。`} type="password" autoComplete="current-password" required maxLength={256} disabled={saving} value={password} onChange={(e) => setPassword(e.target.value)} />}
      {change.action === "ban" && <Field label="封禁原因"><textarea required maxLength={500} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} /></Field>}
      {error && <p className="auth-error" role="alert">{error}</p>}<div className="button-row"><Button secondary disabled={saving} onClick={closeChange}>取消</Button><Button danger={["ban", "revoke-admin", "delete"].includes(change.action)} disabled={saving || (change.action === "delete" && !password)} type="submit">{saving ? (change.action === "delete" ? "正在注销…" : "正在保存…") : change.action === "delete" ? "确认永久注销" : "确认"}</Button></div>
    </form></Modal>}
  </div>;
}

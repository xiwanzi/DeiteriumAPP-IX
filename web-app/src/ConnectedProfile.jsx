import React, { useEffect, useState, useRef } from "react";
import { LogOut, PenLine, Upload, RefreshCw, Bell, MessageCircle } from "lucide-react";
import { Avatar, Badge, Button, Empty, Field, Modal, PageHead, Toggle } from "./components.jsx";
import { id } from "./format.js";
import { uploadAsset } from "./assets.js";

export default function ConnectedProfile({ client, user, onLogout, navigate }) {
  const [profile, setProfile] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false),
    [editing, setEditing] = useState(false), [bio, setBio] = useState(""), [progress, setProgress] = useState(null);
  const pendingBio = useRef(null), fileInput = useRef(null);
  const load = async () => { setError(""); try { const r = await client.profile(user.playerRef); setProfile(r.data); } catch (e) { setError(e.message); } };
  useEffect(() => { load(); }, [user.playerRef]);
  const saveBio = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    if (!pendingBio.current || pendingBio.current.bio !== bio) pendingBio.current = { clientRequestId: id(), expectedVersion: profile.version, bio };
    try { const r = await client.patchProfile(pendingBio.current); setProfile(r.data); setEditing(false); pendingBio.current = null; }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const saveAvatar = async (file) => {
    if (!file || busy || !profile) return; setBusy(true); setError("");
    try {
      const asset = await uploadAsset(client, file, "AVATAR", "PROFILE", user.userId, (value, label) => setProgress({ value, label }));
      const r = await client.patchProfile({ clientRequestId: id(), expectedVersion: profile.version, avatarAssetId: asset.assetId });
      setProfile(r.data);
    } catch (e) { setError(e.message); } finally { setBusy(false); setProgress(null); if (fileInput.current) fileInput.current.value = ""; }
  };
  return <section className="connected-profile">
    <PageHead eyebrow="DEUTERIUM ID" title="你的账号" subtitle="同一个身份，连接 App、网页和游戏。"><Button secondary disabled={busy} onClick={load}><RefreshCw size={16} />刷新</Button></PageHead>
    <div className="profile-card"><div className="profile-identity"><Avatar user={profile || user} size="large" /><div><h2>{user.gameId}</h2><p>QQ · {user.qq}</p><small>Deuterium ID · {user.userId}</small></div></div></div>
    {error && <p className="notice-box" role="alert">{error}</p>}
    {profile && <>
      <div className="panel" style={{ padding: 24, marginTop: 24 }}><h3>个人简介</h3><p style={{ whiteSpace: "pre-wrap" }}>{profile.bio || "还没有填写个人简介。"}</p>
        <div className="button-row"><Button secondary disabled={busy} onClick={() => { setBio(profile.bio); setEditing(true); }}><PenLine size={16} />编辑简介</Button><Button secondary disabled={busy} onClick={() => fileInput.current?.click()}><Upload size={16} />更换头像</Button></div>
        <input ref={fileInput} type="file" hidden accept="image/png,image/jpeg,image/webp" onChange={(e) => saveAvatar(e.target.files?.[0])} />
        {progress && <div role="status"><p>{progress.label} · {progress.value}%</p><progress value={progress.value} max="100" style={{ width: "100%" }} aria-label="头像上传进度" /></div>}
      </div>
      <div className="button-row"><Button secondary onClick={() => navigate("/notification-settings")}><Bell size={16} />通知设置</Button><Button secondary onClick={() => navigate("/wallet")}>查看钱包</Button><Button secondary onClick={() => navigate("/orders")}>我的订单</Button><Button secondary onClick={() => navigate("/merchant")}>我的商店</Button><Button secondary onClick={onLogout}><LogOut size={16} />退出登录</Button></div>
    </>}
    {!profile && !error && <p className="muted">正在读取资料…</p>}
    {editing && <Modal title="编辑个人简介" close={() => { if (!busy) setEditing(false); }}><form onSubmit={saveBio}><Field label="个人简介"><textarea value={bio} onChange={(e) => { setBio(e.target.value); pendingBio.current = null; }} maxLength={200} rows={5} /></Field><p className="muted">{bio.length} / 200</p>{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>{busy ? "正在保存…" : "保存"}</Button></form></Modal>}
  </section>;
}

export function PlayerProfile({ client, user, playerRef, onMessage }) {
  const [profile, setProfile] = useState(null), [error, setError] = useState(""), [busy, setBusy] = useState(false);
  useEffect(() => { let active = true; client.profile(playerRef).then((r) => { if (active) setProfile(r.data); }).catch((e) => { if (active) setError(e.message); }); return () => { active = false; }; }, [playerRef]);
  const follow = async () => { setBusy(true); setError(""); try { const r = profile.followed ? await client.request(`/api/v1/chat/follows/${encodeURIComponent(playerRef)}`, { method: "DELETE" }) : await client.request("/api/v1/chat/follows", { method: "POST", body: { playerRef } }); setProfile({ ...profile, followed: r.data.followed }); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  if (!profile) return <Empty title={error ? "暂时无法读取资料" : "正在读取资料"} text={error || "请稍候…"} />;
  return <><div className="player-detail"><Avatar user={profile} size="large" /><h2>{profile.gameId}</h2><Badge tone="neutral">{profile.online ? "游戏在线" : profile.lastSeenAt ? `上次在线 ${new Date(profile.lastSeenAt).toLocaleString("zh-CN")}` : "玩家"}</Badge></div>
    <p className="description" style={{ whiteSpace: "pre-wrap" }}>{profile.bio || "还没有填写个人简介。"}</p><p>QQ · {profile.qq}</p>
    {user.playerRef !== playerRef && <div className="button-row"><Button disabled={busy} onClick={async () => { setBusy(true); try { await onMessage(profile); } catch (e) { setError(e.message); } finally { setBusy(false); } }}><MessageCircle size={16} />私聊</Button><Button secondary disabled={busy} onClick={follow}>{profile.followed ? "取消特别关心" : "特别关心"}</Button></div>}
    {error && <p className="auth-error" role="alert">{error}</p>}
  </>;
}

export function ConnectedNotificationSettings({ client }) {
  const [value, setValue] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const load = async () => { setError(""); try { const r = await client.request("/api/v1/notifications/preferences"); setValue(r.data); } catch (e) { setError(e.message); } };
  useEffect(() => { load(); }, []);
  const change = async (key, checked) => { setBusy(true); setError(""); try { const r = await client.request("/api/v1/notifications/preferences", { method: "PATCH", body: { clientRequestId: id(), expectedVersion: value.version, [key]: checked } }); setValue(r.data); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  const topics = [["enabled", "消息提醒", "控制账号的业务提醒"], ["showPreviews", "显示通知摘要", "允许系统通知显示消息内容"], ["directMessages", "私聊消息", "其他玩家发来的消息"], ["mentions", "有人提及我", "公共聊天或私聊中的提及"], ["followedPlayers", "特别关心", "关注玩家发布的新消息"], ["wallet", "钱包", "信用点变化与转账"], ["marketOrders", "市场订单", "购买、发货与退款"], ["commissions", "委托", "接取、完成与验收"], ["announcements", "官方公告", "社区的重要消息"], ["appUpdates", "应用更新", "应用与资源版本变化"]];
  return <><PageHead eyebrow="NOTIFICATIONS" title="通知设置" subtitle="提醒偏好在 App 与网页间同步。"><Button secondary onClick={load} disabled={busy}>刷新</Button></PageHead>{error && <p className="notice-box" role="alert">{error}</p>}
    {value && <fieldset className="panel" disabled={busy} style={{ padding: 24, minWidth: 0, margin: 0, border: "1px solid var(--line)" }}>{topics.map(([key, label, description]) => <Toggle key={key} label={label} description={description} checked={Boolean(value[key])} onChange={(checked) => change(key, checked)} />)}</fieldset>}
  </>;
}

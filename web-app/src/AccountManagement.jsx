import { usePlayerNameState } from "./use-player-names.js";
import React, { useEffect, useRef, useState } from "react";
import { Search, RefreshCw, ShieldCheck, Pencil, UserX, Trash2, KeyRound, X, Ticket } from "lucide-react";
import { Avatar, Badge, Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";
import ProfileConflictForm from "./ProfileConflictForm.jsx";
import { ActionMenu, CopyButton } from "./AdminPrimitives.jsx";

const actions = {
  "grant-admin": "授予超级管理员",
  "revoke-admin": "撤销超级管理员",
  "grant-support": "授予管理员",
  "revoke-support": "撤销管理员",
  ban: "封禁账号",
  unban: "解封账号",
  delete: "永久注销",
};

const statusOptions = [
  ["", "全部"],
  ["active", "正常"],
  ["disabled", "已封禁"],
  ["locked", "已锁定"],
];

export default function AccountManagement({ client, user }) {
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("");
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const [data, setData] = usePlayerNameState(client, null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [change, setChange] = useState(null);
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState("");
  const generation = useRef(0);
  const pending = useRef(null);
  const [editing, setEditing] = useState(null);
  const [resolving, setResolving] = useState(null);
  const [password, setPassword] = useState("");
  const isPlatformAdmin = user.permissions?.includes("platform.admin");

  const openChange = (account, action) => {
    setChange({ account, action });
    setReason("");
    setPassword("");
    setError("");
  };
  const closeChange = () => {
    if (!saving) {
      setPassword("");
      setChange(null);
    }
  };
  const load = async () => {
    const current = ++generation.current;
    setBusy(true);
    setError("");
    try {
      const r = await client.request(
        `/api/v1/admin/accounts?${new URLSearchParams({ q: search, status: filter, offset, limit: 30 })}`,
      );
      if (current === generation.current) setData(r.data);
    } catch (e) {
      if (current === generation.current) setError(e.message);
    } finally {
      if (current === generation.current) setBusy(false);
    }
  };
  useEffect(() => {
    load();
    return () => {
      generation.current++;
    };
  }, [client, search, filter, offset]);
  useEffect(() => {
    setPassword("");
    setChange(null);
    setEditing(null);
    setResolving(null);
    pending.current = null;
  }, [client, user.userId]);

  const save = async (event) => {
    event.preventDefault();
    if (saving || !change || (change.action === "delete" && !password)) return;
    setSaving(true);
    setError("");
    const deleting = change.action === "delete";
    const content = {
      expectedVersion: change.account.version,
      ...(deleting ? {} : { action: change.action, reason: reason.trim() }),
    };
    const fingerprint = JSON.stringify([change.account.userId, change.action, content]);
    if (pending.current?.fingerprint !== fingerprint)
      pending.current = { fingerprint, body: { ...content, clientRequestId: id() } };
    // The password is sent only in this request, never retained in the replay body.
    const body = { ...pending.current.body, ...(deleting ? { password } : {}) };
    setPassword("");
    try {
      await client.request(
        `/api/v1/admin/accounts/${encodeURIComponent(change.account.userId)}${deleting ? "/delete" : ""}`,
        { method: "POST", body },
      );
      setFeedback(`已${actions[change.action]}：${change.account.gameId}`);
      setChange(null);
      await load();
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="accounts-workspace">
      <PageHead
        eyebrow="ACCOUNTS"
        title="账号与权限"
        subtitle="超级管理员可管理全部功能；管理员仅能使用工单系统。"
      >
        {data && <Badge tone="neutral">共 {data.total} 位用户</Badge>}
        <Button secondary disabled={busy} onClick={load}>
          <RefreshCw size={15} />刷新
        </Button>
      </PageHead>

      <form
        className="adm-toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          setOffset(0);
          setSearch(query.trim());
        }}
      >
        <div className="adm-toolbar-left">
          <div className="adm-pill-group" role="group" aria-label="按账号状态筛选">
            {statusOptions.map(([value, label]) => (
              <button
                key={value || "all"}
                type="button"
                className={filter === value ? "is-active" : ""}
                aria-pressed={filter === value}
                onClick={() => {
                  setFilter(value);
                  setOffset(0);
                }}
              >
                {label}
              </button>
            ))}
          </div>
          <label className="adm-search-box">
            <Search size={15} />
            <input
              aria-label="查找用户"
              value={query}
              maxLength={100}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索游戏 ID、邮箱、QQ 或编号…"
            />
            {query && (
              <button
                type="button"
                className="adm-search-clear"
                aria-label="清空搜索"
                onClick={() => {
                  setQuery("");
                  setSearch("");
                  setOffset(0);
                }}
              >
                <X size={14} />
              </button>
            )}
          </label>
        </div>
        <div className="adm-toolbar-right">
          <Button type="submit" disabled={busy}>
            <Search size={15} />查找
          </Button>
        </div>
      </form>

      {feedback && (
        <p className="notice-box" role="status">
          {feedback}
        </p>
      )}
      {error && !change && (
        <p role="alert" className="auth-error">
          {error}
        </p>
      )}
      {busy && (
        <p role="status" className="muted">
          正在读取用户…
        </p>
      )}
      {data && (
        <div className="panel table-wrap">
          <table>
            <thead>
              <tr>
                <th>用户</th>
                <th>QQ</th>
                <th>权限</th>
                <th>状态</th>
                <th style={{ width: 196 }}>管理</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((a) => {
                const isSelf = a.userId === user.userId;
                const menuItems = isSelf
                  ? []
                  : [
                      { heading: "身份与权限" },
                      isPlatformAdmin &&
                        !a.gameVerified && {
                          key: "conflict",
                          label: "身份占用处理",
                          icon: KeyRound,
                          onClick: () => setResolving(a),
                        },
                      (a.admin || a.status === "active") && {
                        key: "admin",
                        label: a.admin ? "撤销超级管理" : "设为超级管理员",
                        icon: ShieldCheck,
                        onClick: () => openChange(a, a.admin ? "revoke-admin" : "grant-admin"),
                      },
                      !a.admin &&
                        a.status === "active" && {
                          key: "support",
                          label: a.supportAdmin ? "撤销工单管理" : "设为工单管理员",
                          icon: Ticket,
                          onClick: () => openChange(a, a.supportAdmin ? "revoke-support" : "grant-support"),
                        },
                      { divider: true },
                      { heading: "账号风控" },
                      {
                        key: "ban",
                        label: a.status === "active" ? "封禁账号" : "解封账号",
                        icon: UserX,
                        danger: a.status === "active",
                        onClick: () => openChange(a, a.status === "active" ? "ban" : "unban"),
                      },
                      {
                        key: "delete",
                        label: "永久注销账号",
                        icon: Trash2,
                        danger: true,
                        onClick: () => openChange(a, "delete"),
                      },
                    ];

                return (
                  <tr key={a.userId}>
                    <td>
                      <div className="account-identity">
                        <Avatar user={a.gameId} />
                        <div>
                          <strong>{a.gameId || "待登记游戏身份"}</strong>
                          <small>
                            {a.gameVerified ? "已验证" : "未验证"}
                            {a.email ? ` · ${a.email}` : ""}
                          </small>
                          <small>加入于 {new Date(a.createdAt).toLocaleDateString("zh-CN")}</small>
                        </div>
                      </div>
                    </td>
                    <td>
                      {a.qq ? (
                        <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                          <span>{a.qq}</span>
                          <CopyButton value={a.qq} label="QQ" compact />
                        </span>
                      ) : (
                        "—"
                      )}
                    </td>
                    <td>
                      <Badge tone={a.admin ? "blue" : "neutral"}>
                        {a.admin ? "超级管理员" : a.supportAdmin ? "管理员 · 工单" : "用户"}
                      </Badge>
                    </td>
                    <td>
                      <Badge tone={a.status === "active" ? "sage" : "neutral"}>
                        {{ active: "正常", disabled: "已封禁", locked: "已锁定" }[a.status]}
                      </Badge>
                      {a.banReason && <small className="account-reason">{a.banReason}</small>}
                    </td>
                    <td>
                      {isSelf ? (
                        <span className="muted">当前账号</span>
                      ) : (
                        <div className="adm-compact-row-actions">
                          {isPlatformAdmin && (
                            <Button
                              secondary
                              disabled={busy || saving}
                              onClick={() => setEditing(a)}
                            >
                              <Pencil size={14} />修改 ID
                            </Button>
                          )}
                          <ActionMenu
                            disabled={busy || saving}
                            label={`管理 ${a.gameId}`}
                            items={menuItems}
                          />
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {!data.items.length && (
            <Empty title="没有找到用户" text="试试其他游戏 ID、QQ 或筛选条件。" />
          )}
          <div className="catalog-pagination">
            <span>
              {data.total} 位用户 · 第 {Math.floor(offset / 30) + 1} 页
            </span>
            <div>
              <Button secondary disabled={busy || offset === 0} onClick={() => setOffset(offset - 30)}>
                上一页
              </Button>
              <Button
                secondary
                disabled={busy || offset + 30 >= data.total}
                onClick={() => setOffset(offset + 30)}
              >
                下一页
              </Button>
            </div>
          </div>
        </div>
      )}
      {editing && (
        <Modal
          title="修改游戏 ID"
          close={() => setEditing(null)}
          guardClose
          dismissOnBackdrop={false}
        >
          <GameIDForm
            client={client}
            account={editing}
            onSaved={async (result) => {
              setEditing(null);
              setFeedback(`游戏 ID 已更新：${result.previousGameId} → ${result.gameId}`);
              await load();
            }}
          />
        </Modal>
      )}
      {resolving && (
        <Modal
          title="身份占用处理"
          close={() => setResolving(null)}
          guardClose
          dismissOnBackdrop={false}
        >
          <ProfileConflictForm
            client={client}
            account={resolving}
            onSaved={async () => {
              setResolving(null);
              setFeedback("已解除未验证登记，原账号可通过邮箱登录，真正持有人可继续注册。");
              await load();
            }}
          />
        </Modal>
      )}
      {change && (
        <Modal title={actions[change.action]} close={closeChange} dismissOnBackdrop={false}>
          <form className="settings-stack" onSubmit={save}>
            <p>
              将对 <strong>{change.account.gameId}</strong> 执行“{actions[change.action]}”。
            </p>
            <p className="muted">
              {change.action === "delete"
                ? "该操作无法恢复。账号主页、联系人及私聊、市场商品将被清理，历史交易凭据保留为匿名记录。有未完成交易时无法注销。"
                : change.action === "ban"
                  ? "封禁后将退出所有设备，无法登录或继续使用账号。"
                  : change.action === "grant-support"
                    ? "该用户仅获得工单系统管理权限。"
                    : change.action === "grant-admin"
                      ? "该用户将获得超级管理员权限，包括管理其他账号和系统配置。"
                      : change.action === "unban"
                        ? "解封后用户可以重新登录。"
                        : "该用户将退出所有设备，重新登录后按剩余权限使用。"}
            </p>
            {change.action === "delete" && (
              <Field
                label="当前管理员密码"
                hint={`请输入当前登录账户 ${user.gameId} 的密码进行确认。`}
                type="password"
                autoComplete="current-password"
                required
                maxLength={256}
                disabled={saving}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            )}
            {change.action === "ban" && (
              <Field label="封禁原因">
                <textarea
                  required
                  maxLength={500}
                  rows={3}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </Field>
            )}
            {error && (
              <p className="auth-error" role="alert">
                {error}
              </p>
            )}
            <div className="button-row">
              <Button secondary disabled={saving} onClick={closeChange}>
                取消
              </Button>
              <Button
                danger={["ban", "revoke-admin", "delete"].includes(change.action)}
                disabled={saving || (change.action === "delete" && !password)}
                type="submit"
              >
                {saving
                  ? change.action === "delete"
                    ? "正在注销…"
                    : "正在保存…"
                  : change.action === "delete"
                    ? "确认永久注销"
                    : "确认"}
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}

function GameIDForm({ client, account, onSaved }) {
  const [current, setCurrent] = useState(account);
  const [gameId, setGameId] = useState(account.gameId);
  const [busy, setBusy] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [error, setError] = useState("");
  const pending = useRef(null);
  const saving = useRef(false);
  useUnsavedChanges(gameId, busy);
  const save = async (event) => {
    event.preventDefault();
    if (saving.current) return;
    const input = { expectedVersion: current.version, gameId: gameId.trim() };
    const fingerprint = JSON.stringify(input);
    if (!pending.current || (!uncertain && pending.current.fingerprint !== fingerprint))
      pending.current = { fingerprint, body: { ...input, clientRequestId: id() } };
    saving.current = true;
    setBusy(true);
    setError("");
    try {
      const response = await client.request(
        `/api/v1/admin/accounts/${encodeURIComponent(account.userId)}/game-id`,
        { method: "POST", body: pending.current.body },
      );
      await onSaved(response.data);
    } catch (e) {
      setError(e.message);
      setUncertain(["NETWORK_ERROR", "INVALID_RESPONSE"].includes(e.code) || e.status >= 500);
      if (["VERSION_CONFLICT", "STATE_VERSION_CONFLICT"].includes(e.code)) {
        try {
          const response = await client.request(
            `/api/v1/admin/accounts?${new URLSearchParams({ q: account.playerRef, limit: 30 })}`,
          );
          const latest = response.data.items.find((item) => item.userId === account.userId);
          if (latest) {
            setCurrent(latest);
            setError(`账号刚刚被更新，当前游戏 ID 为 ${latest.gameId}。请核对后重新保存。`);
          }
        } catch {
          /* Keep the original conflict visible; never guess a new version. */
        }
      }
    } finally {
      saving.current = false;
      setBusy(false);
    }
  };
  return (
    <form className="settings-stack" onSubmit={save}>
      <p>
        当前游戏 ID：<strong>{current.gameId}</strong>
      </p>
      <Field
        label="新的游戏 ID"
        hint="1–32 位英文字母、数字或下划线。"
        required
        maxLength={32}
        pattern="[A-Za-z0-9_]{1,32}"
        autoComplete="off"
        value={gameId}
        disabled={busy || uncertain}
        onChange={(event) => setGameId(event.target.value)}
      />
      <p className="muted">
        {current.gameVerified
          ? "已验证账号的新名称需要由服务器确认仍属于原玩家。"
          : "修改后仍需由玩家完成游戏身份验证。"}
        此处修改平台保存的游戏 ID，不修改游戏服务器用户名。账号资料、昵称和资产保留；原登录会话会退出，用户可用新游戏 ID 或原邮箱重新登录。
      </p>
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      {uncertain && <p role="status">保存结果尚未确认，请重试确认原请求。</p>}
      <Button
        type="submit"
        disabled={busy || !gameId.trim() || (!uncertain && gameId.trim() === current.gameId)}
      >
        {busy ? "正在保存…" : uncertain ? "确认保存结果" : "保存游戏 ID"}
      </Button>
    </form>
  );
}

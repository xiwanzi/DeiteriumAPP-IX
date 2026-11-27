import React, { useEffect, useRef, useState } from "react";
import { Plus, RefreshCw, Search, ExternalLink, History, CheckCircle2, XCircle, X } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { id } from "./format.js";
import { CopyButton } from "./AdminPrimitives.jsx";
import "./whitelist.css";

const labels = {
  PENDING: "待审核",
  APPROVED: "已通过",
  REJECTED: "未通过",
  ACTIVE: "可进入",
  REVOKED: "已移除",
};
const sources = { APPLICATION: "申请通过", MANUAL: "管理员添加", IMPORT: "原有玩家" };
const kicks = {
  PENDING: "在线移除待执行",
  DISCONNECTED: "已断开在线连接",
  NOT_ONLINE: "操作时玩家不在线",
  CANCELLED: "已取消旧移除指令",
};
const REJECT_PRESETS = [
  "未在服务器 QQ 群（490579956）内找到对应账号，请入群后重新申请。",
  "无法核实 Minecraft Java 正版账号归属，请确认游戏 ID 拼写无误。",
  "申请信息填写过于简略，请补充介绍你的游玩方向后再提交。",
  "当前暂不符合入服审核要求。",
];
const when = (value) => (value ? new Date(value).toLocaleString("zh-CN", { hour12: false }) : "—");

export default function WhitelistManagement({ client }) {
  const [view, setView] = useState("申请记录");
  const [filter, setFilter] = useState("PENDING");
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const [result, setResult] = useState(null);
  const [summary, setSummary] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");
  const [editor, setEditor] = useState(null);
  const [history, setHistory] = useState(null);
  const [historyBusy, setHistoryBusy] = useState(false);
  const generation = useRef(0);
  const historyGeneration = useRef(0);
  const applications = view === "申请记录";
  const listPath = `/api/v1/admin/whitelist/${applications ? "applications" : "entries"}?${new URLSearchParams({
    q: search,
    status: filter,
    offset,
    limit: 30,
  })}`;
  const data = result?.path === listPath ? result.data : null;

  const load = async (quiet = false) => {
    const current = ++generation.current;
    if (!quiet) setBusy(true);
    setError("");
    try {
      const [list, overview] = await Promise.all([
        client.request(listPath),
        client.request("/api/v1/admin/whitelist/summary"),
      ]);
      if (current === generation.current) {
        setResult({ path: listPath, data: list.data });
        setSummary(overview.data);
      }
    } catch (e) {
      if (current === generation.current) setError(e.message);
    } finally {
      if (current === generation.current) setBusy(false);
    }
  };
  useEffect(() => {
    load();
    const timer = setInterval(() => load(true), 15000);
    return () => {
      generation.current++;
      clearInterval(timer);
    };
  }, [client, listPath]);
  useEffect(() => () => {
    historyGeneration.current++;
  }, []);

  const changeView = (next) => {
    setView(next);
    setFilter(next === "申请记录" ? "PENDING" : "ACTIVE");
    setOffset(0);
  };
  const saved = async (message) => {
    setEditor(null);
    setFeedback(message);
    await load();
  };
  const openHistory = async (entry) => {
    const current = ++historyGeneration.current;
    setHistory({ entry, items: [] });
    setHistoryBusy(true);
    try {
      const r = await client.request(`/api/v1/admin/whitelist/entries/${entry.uuid}/history`);
      if (current === historyGeneration.current) setHistory({ entry, items: r.data.items });
    } catch (e) {
      if (current === historyGeneration.current)
        setHistory({ entry, items: [], error: e.message });
    } finally {
      if (current === historyGeneration.current) setHistoryBusy(false);
    }
  };

  const statusChoices = applications
    ? [
        ["PENDING", "待审核"],
        ["APPROVED", "已通过"],
        ["REJECTED", "未通过"],
        ["", "全部"],
      ]
    : [
        ["ACTIVE", "可进入"],
        ["REVOKED", "已移除"],
        ["", "全部"],
      ];

  return (
    <section className="whitelist-workspace">
      <PageHead eyebrow="ACCESS PERMITS" title="白名单管理" subtitle="审核入服申请，管理玩家的通行权限。">
        <Button secondary disabled={busy} onClick={() => load()}>
          <RefreshCw size={15} />刷新
        </Button>
        <Button onClick={() => setEditor({ type: "add" })}>
          <Plus size={16} />添加玩家
        </Button>
      </PageHead>

      <div className="whitelist-summary">
        <button
          className={applications && filter === "PENDING" ? "is-active" : ""}
          onClick={() => changeView("申请记录")}
        >
          <span>待审核</span>
          <strong>{summary?.pending ?? "—"}</strong>
        </button>
        <button
          className={!applications && filter === "ACTIVE" ? "is-active" : ""}
          onClick={() => changeView("当前白名单")}
        >
          <span>有通行权限</span>
          <strong>{summary?.active ?? "—"}</strong>
        </button>
        <button
          className={!applications && filter === "REVOKED" ? "is-active" : ""}
          onClick={() => {
            setView("当前白名单");
            setFilter("REVOKED");
            setOffset(0);
          }}
        >
          <span>已移除</span>
          <strong>{summary?.revoked ?? "—"}</strong>
        </button>
        <div className="whitelist-gateway">
          <Badge tone={summary?.gateway?.online ? "sage" : "neutral"}>
            {summary?.gateway?.online ? "游戏准入已连接" : "游戏准入未连接"}
          </Badge>
          <small>
            {summary?.gateway?.online
              ? `${summary.gateway.onlinePlayers} 位玩家在线`
              : "新连接需要准入服务可用"}
          </small>
          {summary?.applicationURL && (
            <a href={summary.applicationURL} target="_blank" rel="noopener noreferrer">
              打开申请页 <ExternalLink size={13} />
            </a>
          )}
        </div>
      </div>

      <Tabs values={["申请记录", "当前白名单"]} value={view} onChange={changeView} />

      <form
        className="adm-toolbar"
        onSubmit={(event) => {
          event.preventDefault();
          setSearch(query.trim());
          setOffset(0);
        }}
      >
        <div className="adm-toolbar-left">
          <div className="adm-pill-group" role="group" aria-label="按状态筛选">
            {statusChoices.map(([value, label]) => (
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
              aria-label="查找玩家"
              placeholder="搜索游戏 ID、QQ 或 UUID…"
              value={query}
              maxLength={100}
              onChange={(event) => setQuery(event.target.value)}
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
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      {busy && (
        <p className="muted" role="status">
          正在读取…
        </p>
      )}
      {data && data.items.length === 0 ? (
        <Empty
          title={applications ? "没有符合条件的申请" : "没有符合条件的玩家"}
          text={applications ? "新申请会出现在这里。" : "可以调整筛选条件，或手动添加玩家。"}
        />
      ) : (
        data && (
          <div className="panel table-wrap">
            <table>
              <thead>
                <tr>
                  <th>玩家</th>
                  <th>QQ</th>
                  <th>{applications ? "申请内容" : "来源"}</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {data.items.map((item) => (
                  <tr key={item.applicationId && applications ? item.applicationId : item.uuid}>
                    <td>
                      <strong>{item.gameId}</strong>
                      <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 4 }}>
                        <small className="whitelist-uuid" style={{ margin: 0 }}>
                          {item.uuid}
                        </small>
                        <CopyButton value={item.uuid} label="UUID" compact />
                      </div>
                      <small className="muted">
                        {when(applications ? item.createdAt : item.updatedAt)}
                      </small>
                    </td>
                    <td>
                      {item.qq ? (
                        <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                          <span>{item.qq}</span>
                          <CopyButton value={item.qq} label="QQ" compact />
                        </span>
                      ) : (
                        "未记录"
                      )}
                    </td>
                    <td>
                      {applications ? (
                        <>
                          <span>{item.interests.join(" · ") || "未选择偏好"}</span>
                          <p className="whitelist-excerpt">{item.message || "未填写补充记录"}</p>
                        </>
                      ) : (
                        <>
                          <span>{sources[item.source] || item.source}</span>
                          <p className="whitelist-excerpt">{item.reason || "—"}</p>
                        </>
                      )}
                    </td>
                    <td>
                      <Badge
                        tone={
                          ["ACTIVE", "APPROVED"].includes(item.status)
                            ? "sage"
                            : item.status === "PENDING"
                              ? "amber"
                              : "neutral"
                        }
                      >
                        {labels[item.status]}
                      </Badge>
                      {applications && item.reason && (
                        <small className="whitelist-excerpt">{item.reason}</small>
                      )}
                      {!applications && item.kickStatus && (
                        <small className="whitelist-excerpt">{kicks[item.kickStatus]}</small>
                      )}
                    </td>
                    <td>
                      <div className="adm-compact-row-actions">
                        {applications ? (
                          <Button
                            secondary={item.status !== "PENDING"}
                            onClick={() => setEditor({ type: "review", item })}
                          >
                            {item.status === "PENDING" ? "审批" : "详情"}
                          </Button>
                        ) : (
                          <>
                            <Button
                              secondary
                              danger={item.status === "ACTIVE"}
                              onClick={() =>
                                setEditor(
                                  item.status === "ACTIVE"
                                    ? { type: "revoke", item }
                                    : { type: "add", item },
                                )
                              }
                            >
                              {item.status === "ACTIVE" ? "移除" : "重新添加"}
                            </Button>
                            <Button secondary onClick={() => openHistory(item)}>
                              <History size={14} />记录
                            </Button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="whitelist-pagination">
              <span>共 {data.total} 条</span>
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
        )
      )}
      {editor && (
        <WhitelistEditor
          key={`${editor.type}:${editor.item?.uuid || "new"}:${editor.item?.applicationId || ""}`}
          client={client}
          editor={editor}
          close={() => setEditor(null)}
          onSaved={saved}
        />
      )}
      {history && (
        <Modal
          title={`${history.entry.gameId} · 操作记录`}
          close={() => {
            historyGeneration.current++;
            setHistory(null);
          }}
        >
          {historyBusy ? (
            <p role="status">正在读取操作记录…</p>
          ) : history.error ? (
            <p role="alert" className="auth-error">
              {history.error}
            </p>
          ) : history.items.length ? (
            <ol className="whitelist-history">
              {history.items.map((event) => (
                <li key={event.sequence}>
                  <strong>
                    {{
                      approve: "审核通过",
                      reject: "审核拒绝",
                      "manual-add": "手动添加",
                      revoke: "移除白名单",
                      import: "导入原有玩家",
                    }[event.action] || event.action}
                  </strong>
                  <small>
                    {event.actor} · {when(event.createdAt)}
                  </small>
                  <p>{event.reason || "未填写备注"}</p>
                </li>
              ))}
            </ol>
          ) : (
            <p className="muted">暂无操作记录。</p>
          )}
        </Modal>
      )}
    </section>
  );
}

function WhitelistEditor({ client, editor, close, onSaved }) {
  const item = editor.item;
  const type = editor.type;
  const [name, setName] = useState(item?.gameId || "");
  const [qq, setQQ] = useState(item?.qq || "");
  const [profile, setProfile] = useState(null);
  const [reason, setReason] = useState("");
  const [decision, setDecision] = useState("APPROVE");
  const [member, setMember] = useState(false);
  const [identity, setIdentity] = useState(false);
  const [kick, setKick] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const request = useRef(null);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const readOnly = type === "review" && item.status !== "PENDING";
  const title = type === "add" ? "添加白名单玩家" : type === "revoke" ? "移除白名单" : "申请审批";
  const requiresConfirmation = type === "add" || (type === "review" && decision === "APPROVE");

  const resolve = async () => {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const r = await client.request(
        `/api/v1/admin/whitelist/resolve?gameId=${encodeURIComponent(name.trim())}`,
      );
      if (mounted.current) {
        setProfile(r.data);
        setIdentity(false);
      }
    } catch (e) {
      if (mounted.current) setError(e.message);
    } finally {
      if (mounted.current) setBusy(false);
    }
  };
  const save = async (event) => {
    event.preventDefault();
    if (busy || readOnly) return;
    if (type === "add" && !profile) {
      setError("请先查询并核对正版账号。");
      return;
    }
    if (requiresConfirmation && (!member || !identity)) {
      setError("请核对QQ群成员身份和账号归属后勾选确认。");
      return;
    }
    const input =
      type === "add"
        ? {
            gameId: name.trim(),
            expectedUUID: profile.uuid,
            qq: qq.trim(),
            reason: reason.trim(),
            qqMemberConfirmed: member,
            identityConfirmed: identity,
          }
        : type === "revoke"
          ? { expectedVersion: item.version, reason: reason.trim(), kickOnline: kick }
          : {
              expectedVersion: item.version,
              decision,
              reason: reason.trim(),
              qqMemberConfirmed: member,
              identityConfirmed: identity,
            };
    const fingerprint = JSON.stringify(input);
    if (request.current?.fingerprint !== fingerprint)
      request.current = { fingerprint, body: { ...input, clientRequestId: id() } };
    const path =
      type === "add"
        ? "/api/v1/admin/whitelist/entries"
        : type === "revoke"
          ? `/api/v1/admin/whitelist/entries/${item.uuid}/revoke`
          : `/api/v1/admin/whitelist/applications/${item.applicationId}/review`;
    setBusy(true);
    setError("");
    try {
      const result = await client.request(path, { method: "POST", body: request.current.body });
      const message =
        type === "add"
          ? `已添加 ${profile.name} 的通行权限。`
          : type === "revoke"
            ? `已移除 ${item.gameId} 的通行权限。${kick ? "在线连接的移除结果可在列表中查看。" : "再次连接时将被拒绝。"}`
            : decision === "APPROVE"
              ? `已通过 ${item.gameId} 的申请。`
              : `已拒绝 ${item.gameId} 的申请，原因会显示在申请人的查询结果中。`;
      const mail =
        type === "review"
          ? result.data?.emailStatus === "PENDING"
            ? "审核结果邮件已排队。"
            : result.data?.emailStatus === "SKIPPED"
              ? "邮件提醒未开启，本次未发送邮件。"
              : ""
          : "";
      await onSaved(message + mail);
    } catch (e) {
      if (mounted.current) setError(e.message);
    } finally {
      if (mounted.current) setBusy(false);
    }
  };
  return (
    <Modal
      title={title}
      close={() => {
        if (!busy) close();
      }}
      dismissOnBackdrop={false}
    >
      <form className="settings-stack" onSubmit={save}>
        <fieldset disabled={busy || readOnly} className="whitelist-fieldset">
          {type === "add" ? (
            <>
              <div className="whitelist-resolve">
                <Field
                  label="正版玩家名"
                  required
                  pattern="[A-Za-z0-9_]{3,16}"
                  minLength={3}
                  maxLength={16}
                  value={name}
                  onChange={(event) => {
                    setName(event.target.value);
                    setProfile(null);
                  }}
                  placeholder="Minecraft Java 玩家名"
                />
                <Button
                  secondary
                  type="button"
                  disabled={busy || !/^[A-Za-z0-9_]{3,16}$/.test(name.trim())}
                  onClick={resolve}
                >
                  查询账号
                </Button>
              </div>
              {profile && (
                <div className="notice-box">
                  <strong>{profile.name}</strong>
                  <code className="whitelist-uuid">{profile.uuid}</code>
                  <small>查询结果仅确认正版身份存在，仍需核对申请者对该账号的使用权。</small>
                </div>
              )}
              <Field
                label="玩家 QQ"
                required
                inputMode="numeric"
                pattern="[1-9][0-9]{4,10}"
                maxLength={11}
                value={qq}
                onChange={(event) => setQQ(event.target.value)}
              />
            </>
          ) : (
            <div className="whitelist-detail">
              <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
                <h3 style={{ margin: 0 }}>{item.gameId}</h3>
                <CopyButton value={item.uuid} label="UUID" />
                {item.qq && <CopyButton value={item.qq} label="QQ" />}
              </div>
              <code>{item.uuid}</code>
              <dl>
                <div>
                  <dt>QQ</dt>
                  <dd>{item.qq || "未记录"}</dd>
                </div>
                {type === "review" && (
                  <>
                    <div>
                      <dt>申请时间</dt>
                      <dd>{when(item.createdAt)}</dd>
                    </div>
                    <div>
                      <dt>开拓方向</dt>
                      <dd>{item.interests.join(" · ") || "未选择"}</dd>
                    </div>
                    <div>
                      <dt>补充记录</dt>
                      <dd>{item.message || "未填写"}</dd>
                    </div>
                    <div>
                      <dt>公约</dt>
                      <dd>已同意 · {item.covenantVersion}</dd>
                    </div>
                    {readOnly && (
                      <>
                        <div>
                          <dt>审核结果</dt>
                          <dd>{labels[item.status]}</dd>
                        </div>
                        <div>
                          <dt>审核人</dt>
                          <dd>{item.reviewer || "—"}</dd>
                        </div>
                        <div>
                          <dt>审核时间</dt>
                          <dd>{when(item.reviewedAt)}</dd>
                        </div>
                        <div>
                          <dt>原因</dt>
                          <dd>{item.reason || "—"}</dd>
                        </div>
                      </>
                    )}
                  </>
                )}
              </dl>
            </div>
          )}

          {type === "review" && !readOnly && (
            <>
              <div className="adm-decision-cards" role="radiogroup" aria-label="审批结果">
                <button
                  type="button"
                  role="radio"
                  aria-checked={decision === "APPROVE"}
                  className={`adm-decision-card is-approve ${decision === "APPROVE" ? "is-selected" : ""}`}
                  onClick={() => setDecision("APPROVE")}
                >
                  <strong>
                    <CheckCircle2 size={16} /> 通过申请
                  </strong>
                  <small>签发通行权限并通知 {item.qq}@qq.com</small>
                </button>
                <button
                  type="button"
                  role="radio"
                  aria-checked={decision === "REJECT"}
                  className={`adm-decision-card is-reject ${decision === "REJECT" ? "is-selected" : ""}`}
                  onClick={() => setDecision("REJECT")}
                >
                  <strong>
                    <XCircle size={16} /> 拒绝申请
                  </strong>
                  <small>附带拒绝说明，允许重新提交</small>
                </button>
              </div>
            </>
          )}

          {!readOnly && (
            <>
              {type === "review" && decision === "REJECT" && (
                <div className="adm-preset-chips" aria-label="常用拒绝原因">
                  {REJECT_PRESETS.map((preset) => (
                    <button
                      key={preset}
                      type="button"
                      className="adm-preset-chip"
                      onClick={() => setReason(preset)}
                    >
                      {preset.slice(0, 14)}…
                    </button>
                  ))}
                </div>
              )}
              <Field
                label={
                  type === "revoke"
                    ? "移除原因"
                    : type === "add"
                      ? "添加原因"
                      : decision === "REJECT"
                        ? "拒绝原因（申请人可见）"
                        : "审核备注（选填）"
                }
              >
                <textarea
                  required={type !== "review" || decision === "REJECT"}
                  maxLength={500}
                  rows={3}
                  value={reason}
                  onChange={(event) => setReason(event.target.value)}
                />
              </Field>
            </>
          )}

          {requiresConfirmation && !readOnly && (
            <div className="whitelist-confirmations">
              <label>
                <input
                  type="checkbox"
                  required
                  checked={member}
                  onChange={(event) => setMember(event.target.checked)}
                />
                <span>已核对：此 QQ 当前在服务器群 490579956 内。</span>
              </label>
              <label>
                <input
                  type="checkbox"
                  required
                  checked={identity}
                  onChange={(event) => setIdentity(event.target.checked)}
                />
                <span>已核对：该玩家拥有并使用此正版游戏账号。</span>
              </label>
            </div>
          )}
          {type === "revoke" && (
            <>
              <p className="muted">移除后，新连接将被拒绝。玩家账号、申请和历史操作记录会保留。</p>
              <label className="checkbox-row">
                <input
                  type="checkbox"
                  checked={kick}
                  onChange={(event) => setKick(event.target.checked)}
                />
                同时断开该玩家当前的在线连接
              </label>
            </>
          )}
        </fieldset>
        {error && (
          <p className="auth-error" role="alert">
            {error}
          </p>
        )}
        <div className="button-row">
          <Button secondary disabled={busy} onClick={close}>
            {readOnly ? "关闭" : "取消"}
          </Button>
          {!readOnly && (
            <Button
              type="submit"
              danger={type === "revoke" || (type === "review" && decision === "REJECT")}
              disabled={busy || (type === "add" && !profile)}
            >
              {busy
                ? "正在处理…"
                : type === "revoke"
                  ? "确认移除"
                  : type === "add"
                    ? "确认添加"
                    : decision === "APPROVE"
                      ? "确认通过"
                      : "确认拒绝"}
            </Button>
          )}
        </div>
      </form>
    </Modal>
  );
}

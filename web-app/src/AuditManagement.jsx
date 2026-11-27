import React, { useEffect, useState } from "react";
import { RefreshCw, Search, RotateCcw } from "lucide-react";
import { Button, Empty, Field, PageHead } from "./components.jsx";
import { CopyButton } from "./AdminPrimitives.jsx";

function localDateTimeInput(date) {
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

export default function AuditManagement({ client, embedded = false }) {
  const [filters, setFilters] = useState({ actorId: "", action: "", resourceId: "", from: "", to: "" });
  const [items, setItems] = useState([]);
  const [cursor, setCursor] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = async (more = false, overrideFilters = null) => {
    const activeFilters = overrideFilters || filters;
    setBusy(true);
    setError("");
    if (!more) {
      setItems([]);
      setCursor(null);
    }
    try {
      const query = new URLSearchParams({ limit: "30" });
      if (more && cursor) query.set("cursor", cursor);
      else {
        for (const [key, value] of Object.entries(activeFilters))
          if (value.trim())
            query.set(
              key,
              key === "from" || key === "to" ? new Date(value).toISOString() : value.trim(),
            );
        const from = query.get("from");
        const to = query.get("to") || new Date().toISOString();
        if (from && (new Date(to) <= new Date(from) || new Date(to) - new Date(from) > 90 * 86400000))
          throw new Error("请选择起点早于终点、最多 90 天的时间范围。");
        if (new Date(to) > new Date()) throw new Error("结束时间不能晚于当前时间。");
      }
      const r = await client.request(`/api/v1/admin/audit-events?${query}`);
      if (!Array.isArray(r.data)) throw new Error("审计响应格式不正确，请重新读取。");
      setItems((old) =>
        more
          ? [...old, ...r.data.filter((item) => !old.some((prior) => prior.eventId === item.eventId))]
          : r.data,
      );
      setCursor(r.page?.nextCursor || null);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  useEffect(() => {
    load();
  }, []);

  const applyQuickRange = (days) => {
    const now = new Date();
    const from = new Date(now.getTime() - days * 86400000);
    const next = {
      ...filters,
      from: localDateTimeInput(from),
      to: localDateTimeInput(now),
    };
    setFilters(next);
    load(false, next);
  };

  const resetAll = () => {
    const empty = { actorId: "", action: "", resourceId: "", from: "", to: "" };
    setFilters(empty);
    load(false, empty);
  };

  return (
    <>
      {!embedded && (
        <PageHead
          eyebrow="PLATFORM AUDIT"
          title="管理审计"
          subtitle="查看已记录的管理操作。默认最近 30 天，单次最多查询 90 天。"
        >
          <Button secondary disabled={busy} onClick={() => load()}>
            <RefreshCw size={15} />刷新记录
          </Button>
        </PageHead>
      )}
      <form
        className="panel admin-form audit-filters"
        onSubmit={(event) => {
          event.preventDefault();
          load();
        }}
      >
        <div className="adm-quick-dates">
          <span>快捷时间：</span>
          {[
            [1, "近 24 小时"],
            [7, "近 7 天"],
            [30, "近 30 天"],
          ].map(([days, label]) => (
            <button
              key={days}
              type="button"
              className="adm-preset-chip"
              disabled={busy}
              onClick={() => applyQuickRange(days)}
            >
              {label}
            </button>
          ))}
        </div>
        <div className="form-grid">
          {[
            ["actorId", "操作者 ID", 64],
            ["action", "操作名称", 64],
            ["resourceId", "资源 ID", 128],
          ].map(([key, label, max]) => (
            <Field
              key={key}
              label={label}
              maxLength={max}
              value={filters[key]}
              onChange={(event) => setFilters((old) => ({ ...old, [key]: event.target.value }))}
              placeholder="精确匹配，可留空"
            />
          ))}
          {[
            ["from", "起始时间"],
            ["to", "结束时间"],
          ].map(([key, label]) => (
            <Field
              key={key}
              label={label}
              type="datetime-local"
              value={filters[key]}
              onChange={(event) => setFilters((old) => ({ ...old, [key]: event.target.value }))}
            />
          ))}
        </div>
        <div className="button-row">
          <Button type="submit" disabled={busy}>
            <Search size={15} />查询记录
          </Button>
          <Button secondary disabled={busy} onClick={resetAll}>
            <RotateCcw size={15} />清空条件
          </Button>
          <Button secondary disabled={busy} onClick={() => load()}>
            <RefreshCw size={15} />刷新
          </Button>
          <span className="muted">默认最近 30 天 · 单次最多查询 90 天</span>
        </div>
      </form>
      {error && (
        <p className="notice-box" role="alert">
          {error}
        </p>
      )}
      <div className="panel table-wrap">
        <table>
          <thead>
            <tr>
              <th>时间</th>
              <th>操作者</th>
              <th>操作</th>
              <th>资源</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.eventId}>
                <td>{new Date(item.createdAt).toLocaleString("zh-CN", { hour12: false })}</td>
                <td className="mono">{item.actorId}</td>
                <td className="mono">{item.action}</td>
                <td className="mono" style={{ overflowWrap: "anywhere" }}>
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                    <span>{item.resourceId}</span>
                    {item.resourceId && <CopyButton value={item.resourceId} label="资源 ID" compact />}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!busy && !error && !items.length && (
          <Empty
            title="没有符合条件的记录"
            text="可以调整操作者、操作名称、资源或时间范围后重新查询。"
          />
        )}
      </div>
      {busy && (
        <p className="muted" role="status">
          正在读取审计记录…
        </p>
      )}
      {cursor && (
        <div style={{ marginTop: 14 }}>
          <Button secondary disabled={busy} onClick={() => load(true)}>
            加载更多记录
          </Button>
        </div>
      )}
    </>
  );
}

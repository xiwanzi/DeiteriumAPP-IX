import React, { useEffect, useState } from "react";
import { RefreshCw, Search } from "lucide-react";
import { Button, Empty, Field, PageHead } from "./components.jsx";

export default function AuditManagement({ client }) {
  const [filters, setFilters] = useState({ actorId: "", action: "", resourceId: "", from: "", to: "" }),
    [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const load = async (more = false) => {
    setBusy(true); setError("");
    if (!more) { setItems([]); setCursor(null); }
    try {
      const query = new URLSearchParams({ limit: "30" });
      if (more && cursor) query.set("cursor", cursor);
      else {
        for (const [key, value] of Object.entries(filters)) if (value.trim()) query.set(key, key === "from" || key === "to" ? new Date(value).toISOString() : value.trim());
        const from = query.get("from"), to = query.get("to") || new Date().toISOString();
        if (from && (new Date(to) <= new Date(from) || new Date(to) - new Date(from) > 90 * 86400000)) throw new Error("请选择起点早于终点、最多 90 天的时间范围。");
        if (new Date(to) > new Date()) throw new Error("结束时间不能晚于当前时间。");
      }
      const r = await client.request(`/api/v1/admin/audit-events?${query}`);
      if (!Array.isArray(r.data)) throw new Error("审计响应格式不正确，请重新读取。");
      setItems((old) => more ? [...old, ...r.data.filter((item) => !old.some((prior) => prior.eventId === item.eventId))] : r.data);
      setCursor(r.page?.nextCursor || null);
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  useEffect(() => { load(); }, []);
  return <>
    <PageHead eyebrow="PLATFORM AUDIT" title="管理审计" subtitle="查看已记录的管理操作。默认最近 30 天，单次最多查询 90 天。"><Button secondary disabled={busy} onClick={() => load()}><RefreshCw size={16} />刷新记录</Button></PageHead>
    <form className="panel" style={{ padding: 20, marginBottom: 20 }} onSubmit={(event) => { event.preventDefault(); load(); }}>
      <div className="foundation-grid">{[["actorId", "操作者 ID", 64], ["action", "操作名称", 64], ["resourceId", "资源 ID", 128]].map(([key, label, max]) => <Field key={key} label={label} maxLength={max} value={filters[key]} onChange={(event) => setFilters((old) => ({ ...old, [key]: event.target.value }))} placeholder="精确匹配，可留空" />)}
        {[["from", "起始时间"], ["to", "结束时间"]].map(([key, label]) => <Field key={key} label={label} type="datetime-local" value={filters[key]} onChange={(event) => setFilters((old) => ({ ...old, [key]: event.target.value }))} />)}
      </div><Button type="submit" disabled={busy}><Search size={16} />查询记录</Button>
    </form>
    {error && <p className="notice-box" role="alert">{error}</p>}
    <div className="panel table-wrap"><table><thead><tr><th>时间</th><th>操作者</th><th>操作</th><th>资源</th></tr></thead><tbody>{items.map((item) => <tr key={item.eventId}><td>{new Date(item.createdAt).toLocaleString("zh-CN", { hour12: false })}</td><td className="mono">{item.actorId}</td><td className="mono">{item.action}</td><td className="mono" style={{ overflowWrap: "anywhere" }}>{item.resourceId}</td></tr>)}</tbody></table>
      {!busy && !error && !items.length && <Empty title="没有符合条件的记录" text="可以调整操作者、操作名称、资源或时间范围后重新查询。" />}
    </div>{busy && <p className="muted" role="status">正在读取审计记录…</p>}{cursor && <Button secondary disabled={busy} onClick={() => load(true)}>加载更多记录</Button>}
  </>;
}

import React, { useState, useEffect, useMemo } from "react";
import { RefreshCw, Search, X } from "lucide-react";
import { Button, Empty, Badge, PageHead } from "./components.jsx";
import { CopyButton } from "./AdminPrimitives.jsx";

export default function CoreAdmin({ client }) {
  const [nodes, setNodes] = useState(null);
  const [items, setItems] = useState([]);
  const [cursor, setCursor] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [serverFilter, setServerFilter] = useState("ALL");

  const load = async (searchQuery = "") => {
    setBusy(true);
    setError("");
    try {
      const [n, i] = await Promise.all([client.nodes(), client.items(null, searchQuery)]);
      setNodes(n.data.nodes);
      setItems(i.data.items);
      setCursor(i.data.next);
    } catch (e) {
      setError(
        e.status === 403 ? "此 Deuterium ID 没有 core.read 管理权限。" : e.message,
      );
    } finally {
      setBusy(false);
    }
  };
  useEffect(() => {
    load();
  }, []);

  const more = async () => {
    setBusy(true);
    try {
      const r = await client.items(cursor, query.trim());
      setItems((old) => [...old, ...r.data.items]);
      setCursor(r.data.next);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const serverChoices = useMemo(() => {
    const ids = new Set();
    (nodes || []).forEach((n) => ids.add(n.serverId));
    items.forEach((i) => (i.compatibleServerIds || []).forEach((s) => ids.add(s)));
    return [...ids];
  }, [nodes, items]);

  const filteredItems = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter((item) => {
      if (serverFilter !== "ALL" && !(item.compatibleServerIds || []).includes(serverFilter)) {
        return false;
      }
      if (!q) return true;
      return `${item.displayName} ${item.itemRef}`.toLowerCase().includes(q);
    });
  }, [items, query, serverFilter]);

  const onlineCount = (nodes || []).filter((n) => n.online).length;

  return (
    <>
      <PageHead
        eyebrow="CORE INFRASTRUCTURE"
        title="Core 管理"
        subtitle="实时查看游戏节点运行健康度与已发布物品模板库。"
      >
        {nodes && (
          <Badge tone={onlineCount === nodes.length ? "sage" : "amber"}>
            {onlineCount} / {nodes.length} 节点在线
          </Badge>
        )}
        <Button secondary onClick={() => load(query.trim())} disabled={busy}>
          <RefreshCw size={15} />刷新
        </Button>
      </PageHead>

      {error ? (
        <Empty title="暂时无法读取管理数据" text={error} />
      ) : nodes === null ? (
        <p className="muted">正在读取…</p>
      ) : (
        <>
          <div className="adm-node-grid">
            {nodes.map((n) => {
              const checks = n.runtime
                ? [
                    ["存储", n.runtime.storageHealthy],
                    ["共享存储", n.runtime.sharedStorage],
                    ["数据同步", n.runtime.playerDataReady],
                    ["经济服务", n.runtime.economy],
                    ["经济权威", n.runtime.economyAuthority],
                    ["游戏邮箱", n.runtime.mailbox?.available],
                    ["邮箱交易", n.runtime.mailbox?.commerceReady],
                    ["邮箱存储", n.runtime.mailbox?.storageReady],
                  ]
                : [];
              return (
                <article className="adm-node-card" key={n.serverId}>
                  <div className="adm-node-head">
                    <h3>{n.serverId}</h3>
                    <Badge tone={n.online ? "sage" : "neutral"}>
                      {n.online ? "在线" : "离线"}
                    </Badge>
                  </div>
                  <div className="adm-node-meta">
                    <span>背包域：{n.inventoryDomain}</span>
                    <span>{n.claimEnabled ? "允许领取" : "禁止领取"}</span>
                    <span>聊天：{n.chat ? "开" : "关"}</span>
                    <span>发布：{n.itemPublisher ? "开" : "关"}</span>
                  </div>
                  {n.runtime ? (
                    <>
                      <div className="adm-health-chips">
                        {checks.map(([label, flag]) => (
                          <div className="adm-health-chip" key={label}>
                            <span>{label}</span>
                            <span
                              className={`adm-health-dot ${
                                flag === true ? "is-ok" : flag === false ? "is-bad" : "is-unknown"
                              }`}
                            >
                              <i />
                              {flag === true ? "就绪" : flag === false ? "异常" : "待上报"}
                            </span>
                          </div>
                        ))}
                      </div>
                      {(n.runtime.playerDataProvider || n.runtime.mailbox?.clusterId) && (
                        <small className="muted">
                          {n.runtime.playerDataProvider && `同步：${n.runtime.playerDataProvider}`}
                          {n.runtime.playerDataProvider && n.runtime.mailbox?.clusterId ? " · " : ""}
                          {n.runtime.mailbox?.clusterId && `集群：${n.runtime.mailbox.clusterId}`}
                        </small>
                      )}
                    </>
                  ) : (
                    <p className="muted" style={{ margin: 0, fontSize: 12 }}>
                      节点运行状态待上报
                    </p>
                  )}
                </article>
              );
            })}
          </div>

          <div className="workspace-section-head">
            <div>
              <h2>物品库版本</h2>
              <p>只展示批准的元数据，不读取原始 NBT 或游戏命令。</p>
            </div>
            <Badge tone="neutral">当前显示 {filteredItems.length} 项</Badge>
          </div>

          <form
            className="adm-toolbar"
            onSubmit={(e) => {
              e.preventDefault();
              load(query.trim());
            }}
          >
            <div className="adm-toolbar-left">
              <label className="adm-search-box">
                <Search size={15} />
                <input
                  aria-label="搜索物品模板"
                  placeholder="搜索物品名称或模板引用…"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
                {query && (
                  <button
                    type="button"
                    className="adm-search-clear"
                    aria-label="清空搜索"
                    onClick={() => {
                      setQuery("");
                      load("");
                    }}
                  >
                    <X size={14} />
                  </button>
                )}
              </label>
              <select
                aria-label="按兼容服务器筛选"
                value={serverFilter}
                onChange={(e) => setServerFilter(e.target.value)}
              >
                <option value="ALL">全部服务器</option>
                {serverChoices.map((id) => (
                  <option key={id} value={id}>
                    {id}
                  </option>
                ))}
              </select>
            </div>
            <div className="adm-toolbar-right">
              <Button type="submit" secondary disabled={busy}>
                <Search size={15} />服务端检索
              </Button>
            </div>
          </form>

          <div className="panel table-wrap">
            <table>
              <thead>
                <tr>
                  <th>物品</th>
                  <th>模板引用</th>
                  <th>版本</th>
                  <th>允许服务器</th>
                  <th>最大数量</th>
                </tr>
              </thead>
              <tbody>
                {filteredItems.map((i) => (
                  <tr key={`${i.itemRef}:${i.revision}`}>
                    <td>
                      <strong>{i.displayName}</strong>
                    </td>
                    <td className="mono">
                      <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                        <span>{i.itemRef}</span>
                        <CopyButton value={i.itemRef} label="模板引用" compact />
                      </span>
                    </td>
                    <td>v{i.revision}</td>
                    <td>{i.compatibleServerIds.join("、")}</td>
                    <td>{i.maxQuantity}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!filteredItems.length && (
              <Empty
                title={items.length ? "没有匹配的物品模板" : "尚未发布物品模板"}
                text={
                  items.length
                    ? "调整关键词或服务器筛选后重试。"
                    : "Core 发布元数据后将在这里显示。"
                }
              />
            )}
          </div>
          {cursor && (
            <div style={{ marginTop: 14 }}>
              <Button secondary disabled={busy} onClick={more}>
                加载更多版本
              </Button>
            </div>
          )}
        </>
      )}
    </>
  );
}

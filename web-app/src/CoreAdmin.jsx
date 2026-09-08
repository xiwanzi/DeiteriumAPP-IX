import React, {useState,useEffect} from "react";
import {RefreshCw} from "lucide-react";
import {Button,Empty,Badge} from "./components.jsx";
export default function CoreAdmin({ client }) {
  const [nodes, setNodes] = useState(null),
    [items, setItems] = useState([]),
    [cursor, setCursor] = useState(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const load = async () => {
    setBusy(true);
    setError("");
    try {
      const [n, i] = await Promise.all([client.nodes(), client.items()]);
      setNodes(n.data.nodes);
      setItems(i.data.items);
      setCursor(i.data.next);
    } catch (e) {
      setError(
        e.status === 403
          ? "此 Deuterium ID 没有 core.read 管理权限。"
          : e.message,
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
      const r = await client.items(cursor);
      setItems((old) => [...old, ...r.data.items]);
      setCursor(r.data.next);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <div className="workspace-section-head">
        <div>
          <p className="eyebrow">OFFICIAL WORKSPACE</p>
          <h1>Core 管理</h1>
          <p>查看游戏节点连接状态与已发布物品。</p>
        </div>
        <Button secondary onClick={load} disabled={busy}>
          <RefreshCw size={16} />
          刷新
        </Button>
      </div>
      {error ? (
        <Empty title="暂时无法读取管理数据" text={error} />
      ) : nodes === null ? (
        <p className="muted">正在读取…</p>
      ) : (
        <>
          <div className="foundation-grid">
            {nodes.map((n) => (
              <div className="foundation-card" key={n.serverId}>
                <Badge tone={n.online ? "sage" : "neutral"}>
                  {n.online ? "已连接" : "离线"}
                </Badge>
                <h3 style={{ marginTop: 16 }}>{n.serverId}</h3>
                <p>
                  聊天 {n.chat ? "启用" : "关闭"} · 物品发布{" "}
                  {n.itemPublisher ? "启用" : "关闭"}
                </p>
                <small className="muted">背包域 · {n.inventoryDomain}</small>
                <p className="muted">领取策略 · {n.claimEnabled ? "允许领取" : "禁止领取"}</p>
                {n.runtime ? <dl className="detail-list">
                  {[["存储",n.runtime.storageHealthy],["共享存储",n.runtime.sharedStorage],["玩家数据同步",n.runtime.playerDataReady],["经济服务",n.runtime.economy],["经济权威",n.runtime.economyAuthority],["邮箱",n.runtime.mailbox?.available],["邮箱交易接口",n.runtime.mailbox?.commerceReady],["邮箱存储",n.runtime.mailbox?.storageReady]].map(([label,flag])=><div key={label}><dt>{label}</dt><dd>{flag===true?"已就绪":flag===false?"未就绪":"待上报"}</dd></div>)}
                  {n.runtime.playerDataProvider&&<div><dt>同步服务</dt><dd>{n.runtime.playerDataProvider}</dd></div>}
                  {n.runtime.mailbox?.clusterId&&<div><dt>邮箱集群</dt><dd>{n.runtime.mailbox.clusterId}</dd></div>}
                </dl> : <p className="muted">节点运行状态待上报</p>}
              </div>
            ))}
          </div>
          <div className="workspace-section-head">
            <div>
              <h2>物品库版本</h2>
              <p>只展示批准的元数据，不读取原始 NBT 或游戏命令。</p>
            </div>
          </div>
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
                {items.map((i) => (
                  <tr key={`${i.itemRef}:${i.revision}`}>
                    <td>{i.displayName}</td>
                    <td className="mono">{i.itemRef}</td>
                    <td>{i.revision}</td>
                    <td>{i.compatibleServerIds.join("、")}</td>
                    <td>{i.maxQuantity}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!items.length && (
              <Empty
                title="尚未发布物品模板"
                text="Core 发布元数据后将在这里显示。"
              />
            )}
          </div>
          {cursor && (
            <Button secondary disabled={busy} onClick={more}>
              加载更多版本
            </Button>
          )}
        </>
      )}

    </>
  );
}

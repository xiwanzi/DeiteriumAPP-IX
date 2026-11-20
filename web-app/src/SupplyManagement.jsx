import React, { useEffect, useRef, useState } from "react";
import { Plus, RefreshCw, Copy, CalendarClock, Pause, Play, ArrowLeft } from "lucide-react";
import { Badge, Button, Empty, Field, PageHead } from "./components.jsx";
import { newPool, poolKinds, dateText, creditMinor, creditText } from "./supply-model.js";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";
import SupplyPoolEditor from "./SupplyPoolEditor.jsx";
import SupplyTicketAccounts from "./SupplyTicketAccounts.jsx";
import SupplyRecords from "./SupplyRecords.jsx";
import "./supply-admin.css";

const tabs = [["pools", "卡池管理"], ["accounts", "玩家与票券"], ["orders", "抽卡订单"], ["ledger", "票券流水"], ["settings", "兑换设置"]];

export default function SupplyManagement({ client, user }) {
  const [tab, setTab] = useState("pools"), [pools, setPools] = useState([]), [assets, setAssets] = useState({}), [models, setModels] = useState([]);
  const [edit, setEdit] = useState(null), [query, setQuery] = useState(""), [error, setError] = useState(""), [busy, setBusy] = useState(false), [loading, setLoading] = useState(true);
  const [player, setPlayer] = useState(""), [notice, setNotice] = useState("");
  const alive = useRef(true), scope = String(user.userId || user.id || user.playerRef || user.gameId);
  const load = async () => {
    const results = await Promise.allSettled([client.request("/api/v1/admin/supply/pools"), client.request("/api/v1/admin/supply/assets")]);
    if (!alive.current) return;
    if (results[0].status === "fulfilled") setPools(results[0].value.data.items); else throw results[0].reason;
    if (results[1].status === "fulfilled") {
      const data = results[1].value.data; setAssets({ ...data.builtins, ...Object.fromEntries(data.items.map(a => [a.assetId, a])) }); setModels(data.models);
    } else setNotice("素材服务暂不可用，已有卡池和订单仍可查询；上传及素材预览等待恢复。");
  };
  useEffect(() => { alive.current = true; load().catch(e => { if (alive.current) setError(e.message); }).finally(() => { if (alive.current) setLoading(false); }); return () => { alive.current = false; }; }, [client]);
  const changed = async value => { if (value?.id) setEdit(value); await load(); setNotice("修改已保存"); };
  const switchState = async (pool, action, retry = false) => {
    setBusy(true); setError("");
    try {
      const key = `${scope}:pool:${pool.id}`;
      if (retry) await retrySupply(client, key);
      else await mutateSupply(client, key, `/api/v1/admin/supply/pools/${pool.id}`, { action, expectedVersion: pool.version });
      await load();
    }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const duplicate = async (pool, rerun) => {
    const next = structuredClone(pool);
    if (rerun) {
      try { const r = await client.request(`/api/v1/admin/supply/pools/${pool.id}/versions/${pool.publishedRevision}`); next.definition = r.data; }
      catch (e) { setError(e.message); return; }
    }
    next.id = ""; next.version = 0; next.publishedRevision = 0; next.enabled = false;
    next.sourcePoolId = rerun ? pool.id : ""; next.definition.name += rerun ? " · 复刻" : " · 副本";
    if (!rerun) next.definition.seriesId = "";
    next.definition.startsAt = null; next.definition.endsAt = null; next.definition.visibleAt = null;
    setEdit(next); setError("");
  };
  const shown = pools.filter(p => `${p.definition.name} ${p.id} ${poolKinds[p.definition.kind]}`.toLowerCase().includes(query.toLowerCase()));
  return <section className="supply-admin">
    <PageHead eyebrow="GAME OPERATIONS" title="武器补给" subtitle="管理卡池、素材与票券，追踪每一笔抽取和交付。">
      <Button secondary disabled={busy || loading} onClick={() => { setError(""); load().catch(e => setError(e.message)); }}><RefreshCw size={16} />刷新</Button>
      {!edit && tab === "pools" && <Button onClick={() => { setEdit(newPool()); setError(""); }}><Plus size={16} />新建卡池</Button>}
    </PageHead>
    {error && <p className="form-error" role="alert">{error}</p>}
    {notice && <p className="supply-notice" role="status">{notice}</p>}
    {!edit && <nav className="supply-tabs" aria-label="补给管理功能">{tabs.map(([id, label]) => <button key={id} className={tab === id ? "active" : ""} aria-current={tab === id ? "page" : undefined} onClick={() => { setTab(id); setError(""); }}>{label}</button>)}</nav>}
    {edit ? <SupplyPoolEditor key={`${edit.id || "new"}-${edit.version}`} client={client} initial={edit} assets={assets} models={models} scope={scope} onSaved={changed} onAsset={asset => setAssets(old => ({ ...old, [asset.assetId]: asset }))} close={() => setEdit(null)} /> : <>
      {tab === "pools" && <>
        <div className="supply-summary"><div><strong>{pools.length}</strong><span>卡池配置</span></div><div><strong>{pools.filter(p => p.enabled).length}</strong><span>启用的发布版本</span></div><div><strong>80 / 100–120</strong><span>限定两组独立共享保底</span></div><div><strong>1 / 10</strong><span>单抽 / 十连消耗</span></div></div>
        <Field label="查找卡池"><input value={query} onChange={e => setQuery(e.target.value)} placeholder="名称、类型或编号" /></Field>
        {loading ? <p role="status">正在读取卡池…</p> : !shown.length ? <Empty title="还没有匹配的卡池" text="创建卡池草稿，配置奖品和美术素材后发布。" /> : <div className="table-wrap"><table><thead><tr><th>卡池</th><th>类型 / 方案</th><th>草稿排期 · UTC+8</th><th>发布状态</th><th>操作</th></tr></thead><tbody>{shown.map(pool => <tr key={pool.id}>
          <td><div className="supply-pool-cell">{assets[pool.definition.art.rail]?.url && <img src={assets[pool.definition.art.rail].url} alt="" />}<div><strong>{pool.definition.name}</strong><small title={pool.id}>{pool.id}</small></div></div></td>
          <td>{poolKinds[pool.definition.kind]}<small>{pool.definition.kind === "LIMITED" ? "80 + 120 两个页面" : pool.definition.kind === "STANDARD" ? "80 抽保底" : `${pool.definition.eventPity} 抽保底`}</small></td>
          <td>{dateText(pool.definition.startsAt)}<small>至 {dateText(pool.definition.endsAt)}</small></td>
          <td><Badge>{pool.publishedRevision ? pool.enabled ? "已启用" : "已暂停" : "草稿"}</Badge><small>发布 v{pool.publishedRevision} · 编辑 v{pool.version}</small></td>
          <td><div className="supply-row-actions"><Button secondary onClick={() => setEdit(pool)}>编辑</Button><Button secondary aria-label={`复制 ${pool.definition.name}`} onClick={() => duplicate(pool, false)}><Copy size={14} /></Button>
            {pendingSupplyRequest(`${scope}:pool:${pool.id}`) && <Button secondary disabled={busy} onClick={() => switchState(pool, "", true)}>恢复原请求</Button>}
            {pool.definition.kind === "EVENT" && pool.publishedRevision > 0 && <Button secondary onClick={() => duplicate(pool, true)}><CalendarClock size={14} />复刻</Button>}
            {pool.publishedRevision > 0 && <Button secondary disabled={busy} onClick={() => switchState(pool, pool.enabled ? "pause" : "resume")}>{pool.enabled ? <Pause size={14} /> : <Play size={14} />}{pool.enabled ? "暂停" : "恢复"}</Button>}</div></td>
        </tr>)}</tbody></table></div>}
      </>}
      {tab === "accounts" && <SupplyTicketAccounts client={client} scope={scope} onRecords={(id, type) => { setPlayer(id); setTab(type); }} />}
      {(tab === "orders" || tab === "ledger") && <SupplyRecords key={`${tab}-${player}`} client={client} kind={tab} pools={pools} initialPlayer={player} scope={scope} />}
      {tab === "settings" && <SupplyExchangeSettings client={client} scope={scope} />}
    </>}
  </section>;
}

function SupplyExchangeSettings({ client, scope }) {
  const [value, setValue] = useState(null), [standard, setStandard] = useState(""), [limited, setLimited] = useState(""), [busy, setBusy] = useState(false), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const apply = data => { setValue(data); setStandard(creditText(data.standardPrice)); setLimited(creditText(data.limitedPrice)); };
  useEffect(() => { let live = true; client.request("/api/v1/admin/supply/settings").then(r => { if (live) apply(r.data); }).catch(e => { if (live) setError(e.message); }); return () => { live = false; }; }, [client]);
  const key = `${scope}:settings`;
  const save = async retry => {
    setBusy(true); setError(""); setNotice("");
    try { const result = retry ? await retrySupply(client, key) : await mutateSupply(client, key, "/api/v1/admin/supply/settings", { settings: { version: value.version, standardPrice: creditMinor(standard), limitedPrice: creditMinor(limited) } }); apply(result); setNotice("兑换价格已更新；已创建的兑换订单保留原价格。"); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <div className="supply-panel supply-settings"><h2>信用点兑换票券</h2><p>价格为兑换 1 张券所需信用点。填 0 关闭该券的兑换入口；已有票券仍可正常使用。</p>
    {error && <p className="form-error" role="alert">{error}</p>}{notice && <p role="status">{notice}</p>}
    {value && <form onSubmit={e => { e.preventDefault(); save(false); }}><Field label="1 张常驻券"><input required inputMode="decimal" value={standard} onChange={e => setStandard(e.target.value)} disabled={busy} /></Field><Field label="1 张限定券"><input required inputMode="decimal" value={limited} onChange={e => setLimited(e.target.value)} disabled={busy} /></Field><p className="muted">两种券永久有效、全服共享，仅供本人使用。资金通过现有信用点服务结算，票券入账可追踪。</p><div className="button-row"><Button type="submit" disabled={busy}>保存兑换价格</Button>{pendingSupplyRequest(key) && <Button secondary disabled={busy} onClick={() => save(true)}>恢复原请求</Button>}</div></form>}
  </div>;
}

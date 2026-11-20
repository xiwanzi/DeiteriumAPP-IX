import React, { useState } from "react";
import { Button, Field, Modal } from "./components.jsx";
import { ticketKinds } from "./supply-model.js";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";

export default function SupplyTicketAccounts({ client, scope, onRecords }) {
  const [query, setQuery] = useState(""), [items, setItems] = useState([]), [account, setAccount] = useState(null), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [adjust, setAdjust] = useState(null), [reason, setReason] = useState("");
  const search = async () => {
    setBusy(true); setError("");
    try {
      if (/^[0-9a-f-]{36}$/i.test(query.trim())) { const r = await client.request(`/api/v1/admin/supply/players/${query.trim().toLowerCase()}`); setAccount(r.data); setItems([]); }
      else { const r = await client.request(`/api/v1/admin/supply/players?q=${encodeURIComponent(query.trim())}`); setItems(r.data.items); setAccount(null); }
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const load = async id => { setError(""); try { const r = await client.request(`/api/v1/admin/supply/players/${id}`); setAccount(r.data); } catch (e) { setError(e.message); } };
  const save = async retry => {
    setBusy(true); setError("");
    try {
      const key = `${scope}:account:${account.playerUuid}`;
      const amount = retry ? 0 : Number(adjust.amount);
      if (!retry && (!Number.isSafeInteger(amount) || amount < 0 || amount > 1e9 || !reason.trim())) throw new Error("请填写有效数量和调整原因");
      const result = retry ? await retrySupply(client, key) : await mutateSupply(client, key, "/api/v1/admin/supply/tickets", { playerUuid: account.playerUuid, currency: adjust.currency, action: adjust.action, amount, expectedVersion: account.version, reason: reason.trim() });
      setAccount(result); setAdjust(null); setReason("");
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <div className="supply-panel"><h2>玩家票券与保底</h2><form className="supply-filter-row" onSubmit={e => { e.preventDefault(); search(); }}><Field label="玩家名称或 UUID"><input required value={query} onChange={e => setQuery(e.target.value)} placeholder="名称前缀；未建立票券账户的玩家可直接查 UUID" /></Field><Button type="submit" disabled={busy}>查询玩家</Button></form>
    {error && <p role="alert" className="form-error">{error}</p>}
    {!!items.length && <div className="table-wrap"><table><thead><tr><th>玩家</th><th>常驻券</th><th>限定券</th><th /></tr></thead><tbody>{items.map(item => <tr key={item.playerUuid}><td>{item.gameId || "尚未记录名称"}<small>{item.playerUuid}</small></td><td>{item.standard}</td><td>{item.limited}</td><td><Button secondary onClick={() => load(item.playerUuid)}>查看账户</Button></td></tr>)}</tbody></table></div>}
    {account && <><div className="supply-section-head"><div><h3>{account.gameId || "玩家账户"}</h3><small>{account.playerUuid}</small></div><Button secondary onClick={() => load(account.playerUuid)}>刷新账户</Button></div><div className="supply-summary"><div><strong>{account.standard}</strong><span>常驻券</span></div><div><strong>{account.limited}</strong><span>限定券</span></div></div>
      <div className="button-row"><Button onClick={() => { setAdjust({ currency: "LIMITED", action: "add", amount: "" }); setReason(""); }}>调整票券</Button><Button secondary onClick={() => onRecords(account.playerUuid, "ledger")}>查看票券流水</Button><Button secondary onClick={() => onRecords(account.playerUuid, "orders")}>查看抽卡订单</Button>{pendingSupplyRequest(`${scope}:account:${account.playerUuid}`) && <Button secondary disabled={busy} onClick={() => save(true)}>恢复原调整请求</Button>}</div>
      <h3>保底进度</h3><div className="table-wrap"><table><thead><tr><th>保底组</th><th>已累计</th><th>本轮保底目标</th><th>周期</th></tr></thead><tbody>{Object.entries(account.pity || {}).map(([group, value]) => <tr key={group}><td>{{ standard: "常驻", "limited:80": "全部限定 · 80", "limited:120": "全部限定 · 120" }[group] || `活动 ${group.replace(/^event:/, "")}`}</td><td>{value.count}</td><td>{value.target}</td><td>{value.cycle + 1}</td></tr>)}</tbody></table></div><p className="muted">随机保底目标只在管理审计中展示。玩家端显示最多 120 抽的保障；获得 6 星后开始新周期。</p></>}
    {adjust && <Modal title="调整玩家票券" close={() => { if (!busy) setAdjust(null); }}><form onSubmit={e => { e.preventDefault(); save(false); }}><p>{account.gameId || account.playerUuid}</p><Field label="券种"><select disabled={busy} value={adjust.currency} onChange={e => setAdjust({ ...adjust, currency: e.target.value })}>{Object.entries(ticketKinds).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></Field><Field label="操作"><select disabled={busy} value={adjust.action} onChange={e => setAdjust({ ...adjust, action: e.target.value })}><option value="add">增加</option><option value="take">扣除</option><option value="set">设为指定余额</option></select></Field><Field label={adjust.action === "set" ? "新的余额" : "调整数量"}><input required type="number" min={0} max={1000000000} step={1} disabled={busy} value={adjust.amount} onChange={e => setAdjust({ ...adjust, amount: e.target.value })} /></Field><Field label="调整原因"><textarea required maxLength={200} rows={3} disabled={busy} value={reason} onChange={e => setReason(e.target.value)} placeholder="记录发放依据或异常处理原因" /></Field><p className="muted">操作会记录当前管理员、数量及调整前后的余额。账户发生并发变化时需要刷新后重新确认。</p>{error && <p role="alert" className="form-error">{error}</p>}<div className="button-row"><Button secondary disabled={busy} onClick={() => setAdjust(null)}>取消</Button><Button type="submit" disabled={busy}>{busy ? "提交中…" : "确认调整"}</Button></div></form></Modal>}
  </div>;
}

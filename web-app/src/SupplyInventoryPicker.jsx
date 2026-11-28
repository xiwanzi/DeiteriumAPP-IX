import React, { useState } from "react";
import { Box, RefreshCw } from "lucide-react";
import { Button, Field, Modal } from "./components.jsx";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";
import { supplyGrades } from "./supply-model.js";

function slotName(slot) {
  if (typeof slot !== "string") return "原槽位";
  if (slot.startsWith("storage:")) {
    const index = Number(slot.slice(8));
    return index < 9 ? `快捷栏 ${index + 1}` : `背包 ${index - 8}`;
  }
  return { head: "头部", chest: "胸部", legs: "腿部", feet: "脚部", offhand: "副手" }[slot] || slot;
}

export default function SupplyInventoryPicker({ client, scope, remaining, onAdded, close, targetName = "" }) {
  const binding = Boolean(targetName);
  const key = `${scope}:inventory-capture`, playerKey = `deuterium.supply.inventory-player.${scope}`;
  const [player, setPlayer] = useState(() => sessionStorage.getItem(playerKey) || "");
  const [snapshot, setSnapshot] = useState(null), [selected, setSelected] = useState(null);
  const [stars, setStars] = useState(() => { const value = Number(sessionStorage.getItem(playerKey + ".stars")); return [4, 5, 6].includes(value) ? value : 4; });
  const [busy, setBusy] = useState(false), [error, setError] = useState(""), [notice, setNotice] = useState("");
  const [added, setAdded] = useState(new Set());
  const pending = pendingSupplyRequest(key);

  const read = async () => {
    setBusy(true); setError(""); setNotice(""); setSnapshot(null); setSelected(null);
    try {
      const value = player.trim();
      if (!value) throw new Error("请输入玩家游戏 ID 或 UUID");
      const response = await client.request(`/api/v1/admin/supply/inventory?player=${encodeURIComponent(value)}`);
      setSnapshot(response.data); setAdded(new Set()); sessionStorage.setItem(playerKey, value);
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };

  const save = async recover => {
    setBusy(true); setError(""); setNotice("");
    try {
      const original = recover ? pendingSupplyRequest(key)?.body : null;
      const body = recover ? null : {
        serverId: snapshot.serverId, playerUuid: snapshot.playerUuid, sessionEpoch: snapshot.sessionEpoch,
        slot: selected.slot, fingerprint: selected.fingerprint,
      };
      const item = recover ? await retrySupply(client, key) : await mutateSupply(client, key, "/api/v1/admin/supply/inventory/capture", body);
      onAdded(item, stars);
      if (binding) { close(); return; }
      const saved = original || body;
      if (saved && snapshot?.playerUuid === saved.playerUuid && snapshot?.sessionEpoch === saved.sessionEpoch)
        setAdded(old => new Set([...old, `${saved.slot}:${saved.fingerprint}`]));
      setSelected(null); setNotice(`已保存「${item.displayName}」并加入当前卡池，可继续选择其他物品。`);
    } catch (e) {
      setError(e.message);
      if (["INVENTORY_CHANGED", "PLAYER_SESSION_CHANGED"].includes(e.code)) { setSelected(null); setSnapshot(null); }
    } finally { setBusy(false); }
  };

  const slotButton = slot => {
    const done = added.has(`${slot.slot}:${slot.fingerprint}`);
    return <button type="button" key={slot.slot} className={`supply-inventory-slot ${selected?.slot === slot.slot ? "selected" : ""}`}
      aria-pressed={selected?.slot === slot.slot} aria-label={`${slotName(slot.slot)}：${slot.name || "空"}${done ? "，已加入" : ""}`}
      disabled={busy || !slot.savable || done} title={slot.reason || `${slot.name}\n${slot.itemId}`}
      onClick={() => { setSelected(slot); setError(""); }}>
      <span className="supply-slot-position">{slotName(slot.slot)}</span>
      {slot.amount > 0 ? <><Box size={18} aria-hidden="true" /><strong>{slot.name || slot.itemId}</strong><span className="supply-slot-count">×{slot.amount}</span></> : <span className="muted">空</span>}
      {done ? <small>已加入</small> : slot.reason && <small>不可入库</small>}
    </button>;
  };

  return <Modal title={binding ? `为「${targetName}」绑定背包物品` : "从玩家背包选取奖品"} className="supply-inventory-modal" wide close={() => { if (!busy) close(); }}>
    <div className="supply-inventory-content">
    <p>{binding ? "读取在线玩家背包并保存单件模板，将其绑定到该奖品（保留原奖品名称、星级、权重与模型配置）。原背包物品不会被取走。" : "读取在线玩家当前背包，选中物品后保存到物品库并加入卡池。原背包物品不会被取走。"}</p>
    <form className="supply-filter-row" onSubmit={e => { e.preventDefault(); read(); }}>
      <Field label="玩家游戏 ID / UUID"><input autoFocus required value={player} disabled={busy} onChange={e => setPlayer(e.target.value)} placeholder="输入正在游戏中的玩家 ID" /></Field>
      <Button type="submit" disabled={busy}><RefreshCw size={15} />{busy ? "正在处理…" : "读取背包"}</Button>
    </form>
    {error && <p className="form-error" role="alert">{error}</p>}{notice && <p className="supply-notice" role="status">{notice}</p>}
    {snapshot && <>
      <div className="supply-section-head"><strong>{snapshot.gameId}</strong><span>{snapshot.serverId} · {snapshot.slots.filter(slot => slot.amount > 0).length} 个非空槽位</span></div>
      <p className="muted">玩家整理背包或切服后，请重新读取。带外部存储的容器等不支持入库的物品会显示原因。</p>
      <h3>主背包</h3><div className="supply-inventory-grid">{snapshot.slots.filter(slot => slot.slot.startsWith("storage:") && Number(slot.slot.slice(8)) >= 9).map(slotButton)}</div>
      <h3>快捷栏</h3><div className="supply-inventory-grid">{snapshot.slots.filter(slot => slot.slot.startsWith("storage:") && Number(slot.slot.slice(8)) < 9).map(slotButton)}</div>
      <h3>装备与副手</h3><div className="supply-inventory-grid equipment">{snapshot.slots.filter(slot => !slot.slot.startsWith("storage:")).map(slotButton)}</div>
    </>}
    </div>
    <div className="supply-inventory-selection">
      <div><strong>{selected ? `已选：${selected.name}` : "点击背包中的物品进行选择"}</strong>{selected && <small>{slotName(selected.slot)} · {selected.itemId} · 将保存单件奖品模板</small>}</div>
      {!binding && <Field label="加入卡池后的品质"><select value={stars} disabled={busy || !!pending} onChange={e => { const value = Number(e.target.value); setStars(value); sessionStorage.setItem(playerKey + ".stars", String(value)); }}>{supplyGrades.map(grade => <option key={grade.stars} value={grade.stars}>{grade.label}</option>)}</select></Field>}
      <div className="button-row"><Button secondary disabled={busy} onClick={close}>{binding ? "取消" : "完成选择"}</Button><Button disabled={busy || !selected || (!binding && remaining <= 0) || !!pending} onClick={() => save(false)}>{binding ? "保存并绑定此奖品" : "保存并加入卡池"}</Button></div>
      {remaining <= 0 && <p className="form-error">当前卡池已达到 100 个奖品上限。</p>}
      {pending && <div className="supply-notice"><p>有一笔保存结果尚未确认{pending.body ? `：${pending.body.playerUuid} · ${slotName(pending.body.slot)}` : ""}。</p><Button secondary disabled={busy || remaining <= 0} onClick={() => save(true)}>恢复原保存请求</Button></div>}
    </div>
  </Modal>;
}

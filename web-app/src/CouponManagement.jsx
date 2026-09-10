import React, { useEffect, useRef, useState } from "react";
import { Plus, Search, Ticket, CalendarDays, Users, Send, FileText } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs, Toggle } from "./components.jsx";
import SearchPicker from "./SearchPicker.jsx";
import { CouponCard } from "./CouponWallet.jsx";
import { couponBenefit, couponConditions, couponDate, couponStatus, couponPublishSelection } from "./promotions.js";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

export default function CouponManagement({ client }) {
  const [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [query, setQuery] = useState(""), [filter, setFilter] = useState("草稿"),
    [busy, setBusy] = useState(false), [error, setError] = useState(""), [editor, setEditor] = useState(null),
    [selected, setSelected] = useState(new Map()), [confirmation, setConfirmation] = useState(null), [publishing, setPublishing] = useState(false),
    [publishError, setPublishError] = useState(""), [message, setMessage] = useState(""),
    [deletion, setDeletion] = useState(null), [deleting, setDeleting] = useState(false), [deleteError, setDeleteError] = useState("");
  const generation = useRef(0), publication = useRef(null);
  const load = async (more = false) => {
    const current = ++generation.current; setBusy(true); setError("");
    try { const status = { "草稿": "DRAFT", "进行中": "ACTIVE", "未开始": "SCHEDULED", "已结束": "EXPIRED", "已停用": "INACTIVE" }[filter] || ""; const r = await client.request(`/api/v1/admin/coupons?limit=30&status=${status}&q=${encodeURIComponent(query)}${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); if (current === generation.current) { setItems((old) => more ? [...old, ...r.data] : r.data); setCursor(r.page?.nextCursor); } }
    catch (e) { if (current === generation.current) setError(e.message); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { const timer = setTimeout(load, 220); return () => { clearTimeout(timer); generation.current++; }; }, [query, filter]);
  const visible = items.filter((c) => filter === "全部" || couponStatus(c) === filter);
  const drafts = visible.filter((c) => c.publicationState === "DRAFT"), allSelected = drafts.length > 0 && drafts.every((c) => selected.has(c.couponId));
  const toggle = (coupon) => {
    if (!selected.has(coupon.couponId) && selected.size >= 100) { setError("每批最多发放 100 张草稿。"); return; }
    setSelected((old) => { const next = new Map(old); if (next.has(coupon.couponId)) next.delete(coupon.couponId); else next.set(coupon.couponId, coupon); return next; });
  };
  const selectVisible = () => setSelected((old) => { const next = new Map(old); drafts.forEach((c) => { if (allSelected) next.delete(c.couponId); else if (next.size < 100) next.set(c.couponId, c); }); return next; });
  const reviewPublication = () => {
    try {
      const coupons = couponPublishSelection([...selected.values()]), fingerprint = JSON.stringify(coupons);
      if (publication.current?.fingerprint !== fingerprint) publication.current = { fingerprint, body: { clientRequestId: id(), coupons } };
      setPublishError(""); setConfirmation({ items: [...selected.values()], body: publication.current.body });
    } catch (e) { setError(e.message); }
  };
  const publish = async () => {
    if (publishing || !confirmation) return;
    setPublishing(true); setPublishError("");
    try {
      const response = await client.request("/api/v1/admin/coupons/publish", { method: "POST", body: confirmation.body });
      if (!response.data?.releaseBatchId) throw new Error("发放结果尚未确认，请重试以核对同一批次。");
      setMessage(`已统一发放 ${confirmation.items.length} 张优惠券，符合条件的玩家会收到一次合并提醒。`);
      setSelected(new Map()); setConfirmation(null); publication.current = null;
      if (filter === "全部") await load(); else setFilter("全部");
    } catch (e) { setPublishError(e.message); } finally { setPublishing(false); }
  };
  const remove = async () => {
    if (!deletion || deleting) return;
    setDeleting(true); setDeleteError("");
    try {
      const response = await client.request(`/api/v1/admin/coupons/${encodeURIComponent(deletion.coupon.couponId)}/delete`, { method: "POST", body: deletion.body });
      if (!response.data?.deleted) throw new Error("删除结果尚未确认，请重试核对。");
      setSelected((old) => { const next = new Map(old); next.delete(deletion.coupon.couponId); return next; });
      setItems((old) => old.filter((coupon) => coupon.couponId !== deletion.coupon.couponId));
      setDeletion(null); setMessage("优惠券已删除。"); await load();
    } catch (e) { setDeleteError(e.message); } finally { setDeleting(false); }
  };
  return <div className="coupons-workspace">
    <PageHead eyebrow="OFFERS" title="优惠券" subtitle="先保存草稿，再勾选要一起发放的优惠券。"><Button onClick={() => { setMessage(""); setEditor({}); }}><Plus size={17} />新建草稿</Button></PageHead>
    <div className="template-toolbar"><label className="picker-search"><Search size={18} /><input aria-label="搜索优惠券" placeholder="搜索优惠券名称" value={query} maxLength={100} onChange={(e) => setQuery(e.target.value)} /></label><Button secondary disabled={busy} onClick={() => load()}>刷新</Button></div>
    <Tabs values={["草稿", "全部", "进行中", "未开始", "已结束", "已停用"]} value={filter} onChange={setFilter} />
    {message && <p className="notice-box" role="status">{message}</p>}
    {error && <p className="auth-error" role="alert">{error}</p>}
    {(drafts.length > 0 || selected.size > 0) && <div className="coupon-selection-bar">
      <label className="coupon-select-all"><input type="checkbox" aria-label="全选当前列表的草稿" checked={allSelected} ref={(node) => { if (node) node.indeterminate = !allSelected && drafts.some((c) => selected.has(c.couponId)); }} disabled={busy || !drafts.length} onChange={selectVisible} />全选当前列表</label>
      <span aria-live="polite">已选 <strong>{selected.size}</strong> 张草稿</span>
      <div><Button secondary disabled={!selected.size} onClick={() => setSelected(new Map())}>清除选择</Button><Button disabled={!selected.size || publishing} onClick={reviewPublication}><Send size={16} />批量发放{selected.size > 0 ? `（${selected.size}）` : ""}</Button></div>
    </div>}
    <div className="coupon-row-list">{visible.map((coupon) => <article className={`coupon-admin-row ${selected.has(coupon.couponId) ? "selected" : ""}`} key={coupon.couponId}>
      {coupon.publicationState === "DRAFT" ? <label className="coupon-row-select"><input type="checkbox" aria-label={`选择 ${coupon.name}`} checked={selected.has(coupon.couponId)} disabled={busy} onChange={() => toggle(coupon)} /></label> : <span className="coupon-row-select"><Ticket size={18} aria-hidden="true" /></span>}
      <div className="coupon-row-summary"><Badge tone={couponStatus(coupon) === "进行中" ? "sage" : "neutral"}>{couponStatus(coupon)}</Badge><h3>{coupon.name}</h3><p>{coupon.type === "ITEM" ? "单品折扣券" : "整单优惠券"} · {coupon.stackWithProductDiscount ? "可叠加商品折扣" : "与商品折扣择优"}</p></div>
      <div className="coupon-row-benefit"><div className="coupon-row-value">{couponBenefit(coupon)}</div><small>{couponConditions(coupon)}</small></div>
      <div className="coupon-row-audience"><p>{coupon.audience === "ALL" ? "全体玩家 · 含新注册玩家" : `${coupon.playerRefs.length} 位指定玩家`}</p><small>{couponDate(coupon.startsAt)} — {couponDate(coupon.endsAt)}</small></div>
      <div className="coupon-row-edit coupon-row-actions"><Button secondary disabled={busy} onClick={() => setEditor(coupon)}>编辑</Button>{!coupon.active && <button type="button" className="text-button danger-text" disabled={busy || deleting} onClick={() => { setDeleteError(""); setDeletion({ coupon, body: { clientRequestId: id(), expectedVersion: coupon.version } }); }}>删除</button>}</div>
    </article>)}</div>
    {!busy && !visible.length && <Empty title={filter === "草稿" ? "先准备好你的优惠" : "还没有匹配的优惠券"} text={filter === "草稿" ? "保存几张不同的优惠券草稿，再勾选它们，一次发放给玩家。" : "换个筛选条件，或新建一张优惠券草稿。"} />}
    {cursor && <div className="button-row"><Button secondary disabled={busy} onClick={() => load(true)}>加载更多优惠券</Button></div>}
    {deletion && <Modal title="删除优惠券" dismissOnBackdrop={false} close={() => { if (!deleting) setDeletion(null); }}>
      <p>确认删除“{deletion.coupon.name}”？删除后将从优惠券列表移除，无法再次启用。已有订单记录保留。</p>
      {deleteError && <p className="auth-error" role="alert">{deleteError}</p>}
      <div className="editor-actions"><Button secondary disabled={deleting} onClick={() => setDeletion(null)}>取消</Button><Button danger disabled={deleting} onClick={remove}>{deleting ? "正在删除…" : "确认删除"}</Button></div>
    </Modal>}
    {editor && <Modal title={!editor.couponId ? "新建优惠券草稿" : editor.publicationState === "DRAFT" ? "编辑草稿" : "编辑优惠券"} wide guardClose dismissOnBackdrop={false} className="coupon-editor-modal" close={() => setEditor(null)}><CouponForm client={client} initial={editor} onSaved={async (saved) => {
      setEditor(null); setSelected((old) => { if (!old.has(saved.couponId)) return old; const next = new Map(old); next.set(saved.couponId, saved); return next; });
      setMessage(saved.publicationState === "DRAFT" ? "草稿已保存。勾选需要一起发放的草稿后，点击批量发放。" : "优惠券已保存。");
      if (saved.publicationState === "DRAFT" && filter !== "草稿") setFilter("草稿"); else await load();
    }} /></Modal>}
    {confirmation && <Modal title={`发放这 ${confirmation.items.length} 张优惠券？`} dismissOnBackdrop={false} close={() => { if (!publishing) setConfirmation(null); }} className="coupon-batch-modal">
      <p className="coupon-batch-intro">所选草稿会统一发布，每张券保留自己的优惠规则、发放对象和有效期。玩家会收到一次合并提醒。</p>
      <div className="coupon-batch-review">{confirmation.items.map((coupon) => <div key={coupon.couponId}><div><strong>{coupon.name}</strong><span>{coupon.audience === "ALL" ? "全体玩家" : `${coupon.playerRefs.length} 位指定玩家`} · {couponConditions(coupon)}</span><small>{couponDate(coupon.startsAt)} — {couponDate(coupon.endsAt)}</small></div><b>{couponBenefit(coupon)}</b></div>)}</div>
      <p className="coupon-batch-note">未到开始时间的券会按计划生效；同批券只提醒一次。</p>
      {publishError && <p className="auth-error" role="alert">{publishError}</p>}
      <div className="editor-actions"><Button secondary disabled={publishing} onClick={() => setConfirmation(null)}>取消</Button><Button disabled={publishing} onClick={publish}>{publishing ? "正在统一发放…" : publishError ? "重试发放" : "确认发放"}</Button></div>
    </Modal>}
  </div>;
}

function localTime(value) { return new Date(Date.parse(value) + 8 * 3600000).toISOString().slice(0, 16); }
function serverTime(value) { return new Date(value + ":00+08:00").toISOString(); }
function CouponForm({ client, initial, onSaved }) {
  const draft = !initial.couponId || initial.publicationState === "DRAFT";
  const [value, setValue] = useState({ name: initial.name || "", type: initial.type || "ORDER", benefit: initial.benefit || "FIXED", amountOff: initial.amountOff || "10.00", discountRate: initial.discountRate || 9000, minimumSpend: initial.minimumSpend || "100.00", maxDiscount: initial.maxDiscount || "0.00", stackWithProductDiscount: initial.stackWithProductDiscount ?? true, audience: initial.audience || "ALL", startsAt: initial.startsAt || new Date().toISOString(), endsAt: initial.endsAt || new Date(Date.now() + 7 * 86400000).toISOString(), active: initial.active ?? true });
  const [stores, setStores] = useState((initial.storeIds || []).map((storeId) => ({ storeId, name: storeId }))),
    [products, setProducts] = useState((initial.productIds || []).map((productId) => ({ productId, title: productId }))),
    [players, setPlayers] = useState((initial.playerRefs || []).map((playerRef) => ({ playerRef, gameId: playerRef }))),
    [busy, setBusy] = useState(false), [error, setError] = useState("");
  const editorSnapshot = { value, storeIds: stores.map((v) => v.storeId), productIds: products.map((v) => v.productId), playerRefs: players.map((v) => v.playerRef) };
  const mutation = useRef(null), guard = useUnsavedChanges(editorSnapshot, busy);
  useEffect(() => {
    let live = true;
    Promise.all(stores.map(async (v) => (await client.request(`/api/v1/store/stores/${encodeURIComponent(v.storeId)}`)).data)).then((v) => { if (live && v.length) setStores(v); }).catch(() => {});
    Promise.all(products.map(async (v) => { const r = await client.request(`/api/v1/store/products/${encodeURIComponent(v.productId)}`); return { ...r.data, title: r.data.content.title }; })).then((v) => { if (live && v.length) setProducts(v); }).catch(() => {});
    Promise.all(players.map(async (v) => { const r = await client.request(`/api/v1/admin/accounts?q=${encodeURIComponent(v.playerRef)}&limit=10`); return r.data.items.find((p) => p.playerRef === v.playerRef) || v; })).then((v) => { if (live && v.length) setPlayers(v); }).catch(() => {});
    return () => { live = false; };
  }, []);
  const update = (patch) => setValue((old) => ({ ...old, ...patch }));
  const preview = { ...value, storeIds: stores.map((v) => v.storeId), productIds: products.map((v) => v.productId) };
  const save = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      if (Date.parse(value.endsAt) <= Date.parse(value.startsAt)) throw new Error("结束时间需要晚于开始时间。");
      if (value.audience === "PLAYERS" && !players.length) throw new Error("请至少选择一名玩家。");
      const content = { ...preview, active: draft ? false : value.active, playerRefs: value.audience === "ALL" ? [] : players.map((v) => v.playerRef), ...(initial.couponId ? { expectedVersion: initial.version } : {}) }, fingerprint = JSON.stringify(content);
      if (mutation.current?.fingerprint !== fingerprint) mutation.current = { fingerprint, body: { clientRequestId: id(), ...content } };
      const saved = await client.request(`/api/v1/admin/coupons${initial.couponId ? `/${encodeURIComponent(initial.couponId)}` : ""}`, { method: initial.couponId ? "PUT" : "POST", body: mutation.current.body });
      guard.markSaved(editorSnapshot); await onSaved(saved.data);
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={save}><div className="promotion-form-layout"><fieldset disabled={busy}>
    <section className="promotion-editor-section"><h3>优惠方式</h3><p>每单自动选用一张，让玩家获得最大的实际优惠。</p>
      <Field label="优惠券名称" required maxLength={80} placeholder="例如：旗舰店开业礼遇" value={value.name} onChange={(e) => update({ name: e.target.value })} />
      <Tabs values={["整单优惠", "单品折扣"]} value={value.type === "ITEM" ? "单品折扣" : "整单优惠"} onChange={(v) => update(v === "单品折扣" ? { type: "ITEM", benefit: "PERCENT", minimumSpend: "0.00" } : { type: "ORDER" })} />
      {value.type === "ORDER" && <Field label="优惠形式"><select value={value.benefit} onChange={(e) => update({ benefit: e.target.value })}><option value="FIXED">满额立减</option><option value="PERCENT">满额折扣</option></select></Field>}
      <div className="form-grid"><Field label={value.type === "ITEM" ? "单件门槛（信用点，0 为无门槛）" : "订单门槛（信用点）"} required inputMode="decimal" pattern="(0|[1-9][0-9]*)(\.[0-9]{1,2})?" value={value.minimumSpend} onChange={(e) => update({ minimumSpend: e.target.value })} />
        {value.benefit === "FIXED" ? <Field label="立减金额（信用点）" required inputMode="decimal" pattern="(0|[1-9][0-9]*)(\.[0-9]{1,2})?" value={value.amountOff} onChange={(e) => update({ amountOff: e.target.value })} /> : <Field label="折扣（折）" type="number" required min="0.01" max="9.99" step="0.01" value={value.discountRate / 1000} onChange={(e) => update({ discountRate: Math.round(Number(e.target.value) * 1000) })} />}
      </div>
      {value.benefit === "PERCENT" && <Field label="最多优惠（信用点，0 为不限）" required inputMode="decimal" pattern="(0|[1-9][0-9]*)(\.[0-9]{1,2})?" value={value.maxDiscount} onChange={(e) => update({ maxDiscount: e.target.value })} />}
      <Toggle label="与商品折扣叠加" description={value.stackWithProductDiscount ? "在商品折后价基础上，再使用这张券。" : "比较原价用券和商品折扣，自动采用更低的应付金额。"} checked={value.stackWithProductDiscount} onChange={(checked) => update({ stackWithProductDiscount: checked })} />
    </section>
    <section className="promotion-editor-section"><h3>适用范围</h3><p>留空表示全部。店铺和商品同时指定时，只作用于两者都符合的商品。</p>
      <div className="field"><label>适用店铺</label><SearchPicker label="选择店铺 · 默认全部" multiple values={stores} onChange={setStores} getKey={(v) => v.storeId} getTitle={(v) => v.name} getDescription={(v) => v.intro} load={async (query, cursor) => { const r = await client.request(`/api/v1/store/stores?limit=30&q=${encodeURIComponent(query)}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); return { items: r.data, nextCursor: r.page?.nextCursor }; }} /></div>
      <div className="field"><label>适用商品</label><SearchPicker label="选择商品 · 默认全部" multiple values={products} onChange={setProducts} getKey={(v) => v.productId} getTitle={(v) => v.title} getDescription={(v) => v.content?.subtitle} load={async (query, cursor) => { const r = await client.request(`/api/v1/store/products?limit=30&q=${encodeURIComponent(query)}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); return { items: r.data.map((v) => ({ ...v, title: v.content.title, imageUrl: v.images?.[0]?.url })), nextCursor: r.page?.nextCursor }; }} /></div>
    </section>
    <section className="promotion-editor-section"><h3>发放给谁</h3><p>每位玩家获得一张，无需领取，有效期内结算自动使用。</p>
      <Tabs values={["全体玩家", "指定玩家"]} value={value.audience === "ALL" ? "全体玩家" : "指定玩家"} onChange={(v) => update({ audience: v === "全体玩家" ? "ALL" : "PLAYERS" })} />
      {value.audience === "ALL" ? <p className="notice-box"><Users size={16} />有效期内新注册的玩家也会自动获得。</p> : <SearchPicker label="选择玩家" placeholder="搜索玩家名或 QQ" multiple max={1000} values={players} onChange={setPlayers} getKey={(v) => v.playerRef} getTitle={(v) => v.gameId} getDescription={(v) => v.qq ? `QQ ${v.qq}` : "已绑定玩家"} load={async (query, cursor) => { const offset = cursor || 0; const r = await client.request(`/api/v1/admin/accounts?status=active&limit=30&offset=${offset}&q=${encodeURIComponent(query)}`); return { items: r.data.items, nextCursor: Number(offset) + 30 < r.data.total ? Number(offset) + 30 : null }; }} />}
    </section>
    <section className="promotion-editor-section"><h3>有效时间</h3><p>使用北京时间。过期后直接从玩家入口消失。</p>
      <div className="form-grid"><Field label="开始时间" type="datetime-local" required value={localTime(value.startsAt)} onChange={(e) => { if (e.target.value) update({ startsAt: serverTime(e.target.value) }); }} /><Field label="结束时间" type="datetime-local" required value={localTime(value.endsAt)} onChange={(e) => { if (e.target.value) update({ endsAt: serverTime(e.target.value) }); }} /></div>
      <div className="button-row">{[1, 7, 30].map((days) => <Button key={days} secondary onClick={() => update({ endsAt: new Date(Date.parse(value.startsAt) + days * 86400000).toISOString() })}>{days} 天</Button>)}</div>
      {draft ? <p className="notice-box"><FileText size={17} />草稿不会出现在玩家端。保存后可在列表中勾选，统一发放。</p> : <Toggle label="启用优惠券" description="开启后，在有效期内可用；关闭后停止使用，不重新发放到账提醒。" checked={value.active} onChange={(active) => update({ active })} />}
    </section>
  </fieldset><aside className="promotion-preview"><p>玩家看到的优惠券</p><CouponCard coupon={preview} stores={stores} products={products} preview /><p><Ticket size={14} />每笔结算自动选择一张最优券。<br /><CalendarDays size={14} />有效期结束后，玩家入口不保留过期记录。</p></aside></div>
    {error && <p className="auth-error" role="alert">{error}</p>}<div className="editor-actions"><Button type="submit" disabled={busy}>{busy ? "正在保存…" : draft ? "保存草稿" : "保存优惠券"}</Button></div>
  </form>;
}

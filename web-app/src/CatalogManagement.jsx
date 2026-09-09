import React, { useEffect, useRef, useState } from "react";
import { Plus, RefreshCw, Upload, Trash2, ArrowUp } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { credit, id } from "./format.js";
import ProductForm from "./ProductEditor.jsx";
import {useUnsavedChanges} from "./unsaved-changes.js";
import { uploadAsset } from "./assets.js";
import DeliveryTemplateManagement from "./DeliveryTemplateManagement.jsx";
import BusinessPages from "./BusinessPages.jsx";
import {MediaPicker} from "./MediaPicker.jsx";
export {MediaPicker} from "./MediaPicker.jsx";

export const marketCategories = [["MATERIALS", "建材"], ["EQUIPMENT", "装备"], ["SUPPLIES", "补给"], ["DECORATION", "装饰"], ["CONSTRUCTION", "建筑服务"], ["OTHER", "其他"]];
const moneyInput = (value) => { const text = String(value).trim(); if (text.length > 20 || !/^(0|[1-9]\d*)(\.\d{1,2})?$/.test(text) || /^0(?:\.0{1,2})?$/.test(text)) throw new Error("请输入大于 0 的金额，最多两位小数。"); return text; };
function useStableMutation() {
  const ref = useRef(null);
  return (body) => { const fingerprint = JSON.stringify(body); if (ref.current?.fingerprint !== fingerprint) ref.current = { fingerprint, body: { clientRequestId: id(), ...body } }; return ref.current.body; };
}

export function ListingForm({ client, user, initial = {}, onSaved }) {
  const [value, setValue] = useState({ title: initial.title || "", subtitle: initial.subtitle || "", description: initial.description || "", categoryCode: initial.categoryCode || "MATERIALS", price: initial.price || "", stock: initial.stock || 1, contactQq: initial.contactQq || user.qq, pickupLocation: initial.pickupLocation || "", workHours: initial.workHours || 1 }),
    [images, setImages] = useState(initial.photos || []), [delivery, setDelivery] = useState(initial.deliveryMethods?.join(",") || "PICKUP"), [busy, setBusy] = useState(false), [error, setError] = useState("");
  useUnsavedChanges({value,images:images.map(image=>image.assetId),delivery},busy);
  const stable = useStableMutation(), construction = value.categoryCode === "CONSTRUCTION";
  const field = (name, label, props = {}) => <Field key={name} label={label} value={value[name]} onChange={(e) => setValue((old) => ({ ...old, [name]: props.type === "number" ? Number(e.target.value) : e.target.value }))} {...props} />;
  const save = async (event) => {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      if (!images.length) throw new Error("请至少上传一张商品图片。");
      const content = { ...value, price: moneyInput(value.price), stock: Number(value.stock), workHours: Number(value.workHours), photoAssetIds: images.map((asset) => asset.assetId), deliveryMethods: construction ? ["WORKSITE"] : delivery.split(",") };
      const body = stable({ ...(initial.listingId ? { expectedVersion: initial.version } : {}), content });
      const path = initial.listingId ? `/api/v1/market/listings/${encodeURIComponent(initial.listingId)}${initial.active === false ? "/republish" : ""}` : "/api/v1/market/listings";
      await client.request(path, { method: !initial.listingId || initial.active === false ? "POST" : "PUT", body }); await onSaved();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <form onSubmit={save}><fieldset disabled={busy} style={{ border: 0, margin: 0, padding: 0, minWidth: 0 }}>
    {field("title", "商品标题", { required: true, maxLength: 80 })}{field("subtitle", "一句话简介", { required: true, maxLength: 200 })}
    <Field label="详细介绍"><textarea required rows={5} maxLength={10000} value={value.description} onChange={(e) => setValue({ ...value, description: e.target.value })} /></Field>
    <Field label="分类"><select value={value.categoryCode} onChange={(e) => setValue({ ...value, categoryCode: e.target.value })}>{marketCategories.map(([code, label]) => <option key={code} value={code}>{label}</option>)}</select></Field>
    {field("price", "售价（信用点）", { required: true, inputMode: "decimal", maxLength: 20 })}{field("stock", "库存", { required: true, type: "number", min: 1, max: 999 })}
    <MediaPicker client={client} images={images} onChange={setImages} purpose="MARKET_PHOTO" businessType="MARKET_LISTING" businessRef={initial.listingId || ""} max={5} onBusy={setBusy} />
    {field("contactQq", "联系 QQ", { required: true, inputMode: "numeric", pattern: "[0-9]{5,12}", maxLength: 12 })}
    {!construction && <Field label="交付方式"><select value={delivery} onChange={(e) => setDelivery(e.target.value)}><option value="PICKUP">约定自取</option><option value="DOOR">送货上门</option><option value="DOOR,PICKUP">两种均可</option></select></Field>}
    {(construction || delivery.includes("PICKUP")) && field("pickupLocation", construction ? "工程地点" : "自取地点", { required: true, maxLength: 120 })}
    {construction && field("workHours", "总工期（小时，包含验收预留）", { required: true, type: "number", min: 1, max: 8760 })}
  </fieldset>{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>{busy ? "正在处理…" : initial.listingId ? initial.active === false ? "重新上架" : "保存商品" : "发布商品"}</Button></form>;
}

export default function CatalogManagement({ client, user }) {
  const [stores, setStores] = useState([]), [selected, setSelected] = useState(""), [products, setProducts] = useState([]), [brands, setBrands] = useState([]), [categories, setCategories] = useState([]), [templates, setTemplates] = useState([]),
    [error, setError] = useState(""), [busy, setBusy] = useState(false), [modal, setModal] = useState(null), [reason, setReason] = useState("");
  const all = (user.permissions || []).includes("platform.admin"), current = stores.find((store) => store.storeId === selected), actionKeys = useRef({});
  const loadStores = async () => {
    setBusy(true); setError("");
    try { const r = await client.request("/api/v1/merchant/me"); const values = await Promise.all(r.data.storeIds.map((storeId) => client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}`))); const next = values.map((item) => item.data); setStores(next); setSelected((old) => next.some((store) => store.storeId === old) ? old : next[0]?.storeId || ""); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const loadStoreContent = async () => {
    if (!selected) return; setBusy(true); setError("");
    try { const [p, b, c, t] = await Promise.all(["products", "brands", "categories", "delivery-templates"].map((kind) => client.request(`/api/v1/merchant/stores/${encodeURIComponent(selected)}/${kind}?limit=100`))); setProducts(p.data); setBrands(b.data); setCategories(c.data); setTemplates(t.data); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  useEffect(() => { loadStores(); }, []); useEffect(() => { loadStoreContent(); }, [selected]);
  const action = async (product, kind, explanation = "") => {
    setBusy(true); setError(""); const scope = `${product.productId}:${product.version}:${kind}:${explanation}`; actionKeys.current[scope] ||= id();
    try { await client.request(`/api/v1/merchant/products/${encodeURIComponent(product.productId)}/${kind}`, { method: "POST", body: { clientRequestId: actionKeys.current[scope], expectedVersion: product.version, ...(kind !== "publish" ? { reason: explanation } : {}) } }); setModal(null); await loadStoreContent(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  return <>
    <PageHead eyebrow="MERCHANT WORKSPACE" title="商店管理" subtitle="商品、分类与店铺资料，在这里统一维护。"><Button secondary disabled={busy} onClick={loadStores}><RefreshCw size={16} />刷新</Button>{all && <Button onClick={() => setModal({ type: "store-create" })}><Plus size={16} />创建商店</Button>}</PageHead>
    {error && <p className="notice-box" role="alert">{error}</p>}
    {stores.length > 0 && <Field label="当前商店"><select value={selected} onChange={(e) => setSelected(e.target.value)}>{stores.map((store) => <option key={store.storeId} value={store.storeId}>{store.name}</option>)}</select></Field>}
    {current && <>
      <div className="panel" style={{ padding: 22 }}><h2>{current.name}</h2><p>{current.intro}</p><div className="button-row"><Button secondary onClick={() => setModal({ type: "store-edit", item: current })}>编辑商店资料</Button><Button secondary onClick={() => setModal({ type: "brand" })}>新增品牌</Button><Button secondary onClick={() => setModal({ type: "category" })}>新增分类</Button><Button secondary onClick={() => setModal({ type: "templates" })}>交付模板</Button><Button secondary onClick={() => setModal({ type: "orders" })}>商店订单</Button><Button onClick={() => setModal({ type: "product", item: {} })}>新增商品</Button></div></div>
      {(!brands.some((x)=>x.active) || !categories.some((x)=>x.active) || !templates.some((x)=>x.active)) && <p className="notice-box">发布商品前，请准备好{!brands.some((x)=>x.active) ? "品牌、" : ""}{!categories.some((x)=>x.active) ? "分类、" : ""}{!templates.some((x)=>x.active) ? "游戏内邮箱交付模板" : "交付资料"}。</p>}
      <div className="foundation-grid">{products.map((product) => <article key={product.productId} className="foundation-card"><Badge tone={product.visibility === "ACTIVE" ? "sage" : "neutral"}>{{ ACTIVE: "已上架", DRAFT: "草稿", UNLISTED: "已下架", ARCHIVED: "已归档" }[product.visibility]}</Badge><h3>{product.draft.title}</h3><p>{product.draft.subtitle}</p><strong className="price">{credit(product.draft.price)}<small>信用点</small></strong><div className="button-row"><Button secondary disabled={busy || product.visibility === "ARCHIVED"} onClick={() => setModal({ type: "product", item: product })}>编辑</Button><Button disabled={busy || product.visibility === "ARCHIVED"} onClick={() => action(product, "publish")}>发布</Button>{product.visibility === "ACTIVE" && <Button secondary disabled={busy} onClick={() => { setModal({ type: "unlist", item: product }); setReason(""); }}>下架</Button>}</div></article>)}</div>
      {!busy && !products.length && <Empty title="还没有商品" text="保存商品草稿并发布后，玩家就能在官方商城看到。" />}
    </>}
    {modal?.type.startsWith("store-") && <Modal title={modal.type === "store-create" ? "创建商店" : "商店资料"} close={() => setModal(null)}><StoreForm client={client} user={user} initial={modal.item || {}} onSaved={async () => { setModal(null); await loadStores(); }} /></Modal>}
    {(modal?.type === "brand" || modal?.type === "category") && <Modal title={modal.type === "brand" ? "新增品牌" : "新增分类"} close={() => setModal(null)}><ClassificationForm client={client} storeId={selected} kind={modal.type} onSaved={async () => { setModal(null); await loadStoreContent(); }} /></Modal>}
    {modal?.type === "product" && <Modal title={modal.item.productId ? "编辑商品" : "新增商品"} close={() => setModal(null)} wide guardClose dismissOnBackdrop={false}><ProductForm client={client} storeId={selected} initial={modal.item} brands={brands} categories={categories} templates={templates} onSaved={async () => { setModal(null); await loadStoreContent(); }} /></Modal>}
    {modal?.type === "templates" && <Modal title="游戏内邮箱交付模板" close={() => setModal(null)} wide><DeliveryTemplateManagement client={client} storeId={selected} onChanged={loadStoreContent}/></Modal>}
    {modal?.type === "orders" && <Modal title="商店订单" close={() => setModal(null)} wide><BusinessPages client={client} user={user} type="ORDER" merchantStoreId={selected}/></Modal>}
    {modal?.type === "unlist" && <Modal title="下架商品" close={() => setModal(null)}><form onSubmit={(e) => { e.preventDefault(); action(modal.item, "unlist", reason); }}><p>确认下架“{modal.item.draft.title}”？已经成交的订单不受影响。</p><Field label="下架原因"><textarea required minLength={2} maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} /></Field>{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>确认下架</Button></form></Modal>}
  </>;
}

function StoreForm({ client, user, initial, onSaved }) {
  const [value, setValue] = useState({ name: initial.name || "", intro: initial.intro || "", contactQq: initial.contactQq || user.qq, serviceHours: initial.serviceHours || "", notice: initial.notice || "" }), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const stable = useStableMutation();
  const save = async (e) => { e.preventDefault(); setBusy(true); setError(""); try { const body = stable({ ...value, logoAssetId: initial.logo?.assetId || null, coverAssetId: initial.cover?.assetId || null, ...(initial.storeId ? { expectedVersion: initial.version } : {}) }); await client.request(initial.storeId ? `/api/v1/merchant/stores/${encodeURIComponent(initial.storeId)}` : "/api/v1/admin/stores", { method: initial.storeId ? "PUT" : "POST", body }); await onSaved(); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  return <form onSubmit={save}>{[["name", "商店名称", 60], ["intro", "商店简介", 300], ["contactQq", "联系 QQ", 12], ["serviceHours", "服务时间", 100], ["notice", "商店公告", 1000]].map(([key, label, max]) => <Field key={key} label={label} value={value[key]} onChange={(e) => setValue({ ...value, [key]: e.target.value })} required={["name", "contactQq"].includes(key)} maxLength={max} pattern={key === "contactQq" ? "[0-9]{5,12}" : undefined} />)}{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>保存商店</Button></form>;
}
function ClassificationForm({ client, storeId, kind, onSaved }) {
  const [name, setName] = useState(""), [order, setOrder] = useState(0), [busy, setBusy] = useState(false), [error, setError] = useState(""); const stable = useStableMutation();
  return <form onSubmit={async (e) => { e.preventDefault(); setBusy(true); setError(""); try { await client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}/${kind === "brand" ? "brands" : "categories"}`, { method: "POST", body: stable({ name, sortOrder: Number(order), active: true, ...(kind === "brand" ? { logoAssetId: null } : {}) }) }); await onSaved(); } catch (e) { setError(e.message); } finally { setBusy(false); } }}><Field label="名称" value={name} onChange={(e) => setName(e.target.value)} required maxLength={kind === "brand" ? 60 : 40} /><Field label="排序" type="number" min={0} max={9999} value={order} onChange={(e) => setOrder(e.target.value)} />{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>保存</Button></form>;
}

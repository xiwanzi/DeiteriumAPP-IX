import React, { useEffect, useRef, useState } from "react";
import { Plus, RefreshCw, Upload, Trash2, ArrowUp, Search, Package, ChevronLeft, ChevronRight, Settings2 } from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead, Tabs } from "./components.jsx";
import { credit, id } from "./format.js";
import ProductForm from "./ProductEditor.jsx";
import {useUnsavedChanges} from "./unsaved-changes.js";
import { uploadAsset } from "./assets.js";
import DeliveryTemplateManagement from "./DeliveryTemplateManagement.jsx";
import BusinessPages from "./BusinessPages.jsx";
import {MediaPicker} from "./MediaPicker.jsx";
import StoreAvatarPicker from "./StoreAvatarPicker.jsx";
import { salePrice } from "./promotions.js";
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
  const [moreTemplates,setMoreTemplates]=useState(false);
  const [stores, setStores] = useState([]), [selected, setSelected] = useState(""), [products, setProducts] = useState([]), [brands, setBrands] = useState([]), [categories, setCategories] = useState([]), [templates, setTemplates] = useState([]),
    [error, setError] = useState(""), [busy, setBusy] = useState(false), [modal, setModal] = useState(null), [reason, setReason] = useState(""), [query, setQuery] = useState(""), [status, setStatus] = useState("ALL"), [categoryFilter, setCategoryFilter] = useState("ALL"), [page, setPage] = useState(1), [cursor, setCursor] = useState(null);
  const all = (user.permissions || []).includes("platform.admin"), current = stores.find((store) => store.storeId === selected), actionKeys = useRef({});
  const loadStores = async () => {
    setBusy(true); setError("");
    try { const r = await client.request("/api/v1/merchant/me"); const values = await Promise.all(r.data.storeIds.map((storeId) => client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}`))); const next = values.map((item) => item.data); setStores(next); setSelected((old) => next.some((store) => store.storeId === old) ? old : next[0]?.storeId || ""); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const generation = useRef(0);
  const loadStoreContent = async (more = false) => {
    const requestGeneration = ++generation.current;
    if (!selected) return; setBusy(true); setError("");
    try { const [p, b, c, t] = await Promise.all(["products", "brands", "categories", "delivery-templates"].map((kind) => client.request(`/api/v1/merchant/stores/${encodeURIComponent(selected)}/${kind}?limit=100${kind === "products" && more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`))); if (requestGeneration !== generation.current) return; setProducts((old) => more ? [...new Map([...old, ...p.data].map((product) => [product.productId, product])).values()] : p.data); setCursor(p.page?.nextCursor); setBrands(b.data); setCategories(c.data); setTemplates(t.data);setMoreTemplates(Boolean(t.page?.hasMore)); }
    catch (e) { if (requestGeneration === generation.current) setError(e.message); } finally { if (requestGeneration === generation.current) setBusy(false); }
  };
  useEffect(() => { loadStores(); }, []); useEffect(() => { setProducts([]); setBrands([]); setCategories([]); setTemplates([]); setCursor(null); setPage(1); loadStoreContent(); return () => { generation.current++; }; }, [selected]);
  useEffect(() => { setPage(1); }, [query, status, categoryFilter]);
  const action = async (product, kind, explanation = "") => {
    if (busy) return;
    setBusy(true); setError(""); const scope = `${product.productId}:${product.version}:${kind}:${explanation}`; actionKeys.current[scope] ||= id();
    try { const response = await client.request(`/api/v1/merchant/products/${encodeURIComponent(product.productId)}/${kind}`, { method: "POST", body: { clientRequestId: actionKeys.current[scope], expectedVersion: product.version, ...(kind !== "publish" && kind !== "delete" ? { reason: explanation } : {}) } }); if (kind === "delete" && !response.data?.deleted) throw new Error("删除结果尚未确认，请重试核对。"); setModal(null); await loadStoreContent(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const states = { ACTIVE: "已上架", DRAFT: "草稿", UNLISTED: "已下架", ARCHIVED: "已归档" };
  const filtered = products.filter((product) => (status === "ALL" || product.visibility === status) && (categoryFilter === "ALL" || product.draft.categoryId === categoryFilter) && `${product.draft.title} ${product.draft.subtitle} ${product.productId}`.toLowerCase().includes(query.trim().toLowerCase()));
  const pages = Math.max(1, Math.ceil(filtered.length / 12)), currentPage = Math.min(page, pages);
  return <div className="catalog-workspace">
    <PageHead eyebrow="MERCHANT WORKSPACE" title="商店管理" subtitle="商品、分类与店铺资料，在这里统一维护。"><Button secondary disabled={busy} onClick={async () => { await loadStores(); await loadStoreContent(); }}><RefreshCw size={16} />刷新</Button>{all && <Button onClick={() => setModal({ type: "store-create" })}><Plus size={16} />创建商店</Button>}</PageHead>
    {error && <p className="notice-box" role="alert">{error}</p>}
    {stores.length > 1 && <Field label="当前商店"><select disabled={busy} value={selected} onChange={(e) => setSelected(e.target.value)}>{stores.map((store) => <option key={store.storeId} value={store.storeId}>{store.name}</option>)}</select></Field>}
    {current && <>
      <section className="store-overview"><div className="store-identity"><span className="store-mark">{current.logo?.url ? <img src={current.logo.url} alt="商店头像" /> : <Package size={25} />}</span><div><h2>{current.name}</h2><p>{current.intro || "在这里管理店铺与商品交付"}</p></div></div><Button secondary disabled={busy} onClick={() => setModal({ type: "store-edit", item: current })}><Settings2 size={16} />店铺资料</Button></section>
      <div className="catalog-stats">{[["已加载商品", products.length], ["已上架", products.filter((p) => p.visibility === "ACTIVE").length], ["草稿", products.filter((p) => p.visibility === "DRAFT").length], ["可用交付模板", `${templates.filter((t) => t.active).length}${moreTemplates?"+":""}`]].map(([label, count]) => <div key={label}><span>{label}</span><strong>{count}</strong></div>)}</div>
      <div className="catalog-actions"><div className="button-row"><Button secondary disabled={busy} onClick={() => setModal({ type: "brand" })}>新增品牌</Button><Button secondary disabled={busy} onClick={() => setModal({ type: "category" })}>新增分类</Button><Button secondary disabled={busy} onClick={() => setModal({ type: "templates" })}>交付模板</Button><Button secondary disabled={busy} onClick={() => setModal({ type: "orders" })}>商店订单</Button></div><Button disabled={busy} onClick={() => setModal({ type: "product", item: {} })}><Plus size={17} />新增商品</Button></div>
      {(!brands.some((x)=>x.active) || !categories.some((x)=>x.active) || !templates.some((x)=>x.active)) && <p className="notice-box">发布商品前，请准备好{!brands.some((x)=>x.active) ? "品牌、" : ""}{!categories.some((x)=>x.active) ? "分类、" : ""}{!templates.some((x)=>x.active) ? "游戏内邮箱交付模板" : "交付资料"}。</p>}
      <section className="catalog-table-panel">
        <div className="catalog-filterbar"><label className="search-input"><Search size={17} /><input aria-label="搜索管理商品" placeholder="搜索已加载的商品名称、简介或编号" value={query} onChange={(e) => setQuery(e.target.value)} /></label><select aria-label="商品状态" value={status} onChange={(e) => setStatus(e.target.value)}><option value="ALL">全部状态</option>{Object.entries(states).map(([key, label]) => <option value={key} key={key}>{label}</option>)}</select><select aria-label="商品分类筛选" value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)}><option value="ALL">全部分类</option>{categories.map((c) => <option value={c.categoryId} key={c.categoryId}>{c.name}</option>)}</select></div>
        <div className="table-wrap"><table className="catalog-table"><thead><tr><th scope="col">商品</th><th scope="col">状态</th><th scope="col">分类</th><th scope="col">售价 / 信用点</th><th scope="col">库存</th><th scope="col">操作</th></tr></thead><tbody>{filtered.slice((currentPage - 1) * 12, currentPage * 12).map((product) => {
          const image = product.draftImages?.[0] || product.published?.images?.[0];
          return <tr key={product.productId}><td><button className="catalog-product-cell" disabled={busy || product.visibility === "ARCHIVED"} onClick={() => setModal({ type: "product", item: product })}>{image?.url ? <img src={image.url} alt="" /> : <span className="catalog-thumbnail"><Package size={22} /></span>}<span><strong>{product.draft.title}</strong><small>{product.draft.subtitle}</small></span></button></td><td><Badge tone={product.visibility === "ACTIVE" ? "sage" : "neutral"}>{states[product.visibility]}</Badge>{product.hasUnpublishedChanges && <small className="unpublished-hint">有未发布修改</small>}</td><td>{categories.find((c) => c.categoryId === product.draft.categoryId)?.name || "—"}</td><td className="catalog-price">{credit(salePrice(product.draft.price,product.draft.discountRate||10000))}{product.draft.discountRate<10000&&<del className="sale-original">{credit(product.draft.price)}</del>}</td><td>{product.draft.inventoryPolicy === "UNLIMITED" ? "不限量" : product.availableStock ?? product.draft.stock}</td><td><div className="catalog-row-actions"><Button secondary disabled={busy || product.visibility === "ARCHIVED"} onClick={() => setModal({ type: "product", item: product })}>编辑 / 预览</Button>{product.visibility !== "ACTIVE" && <Button disabled={busy || product.visibility === "ARCHIVED"} onClick={() => action(product, "publish")}>发布</Button>}{product.visibility === "ACTIVE" && <button type="button" className="text-button" disabled={busy} onClick={() => { setModal({ type: "unlist", item: product }); setReason(""); }}>下架</button>}<button type="button" className="text-button danger-text" disabled={busy} onClick={() => { setError(""); setModal({ type: "delete", item: product }); }}>删除</button></div></td></tr>;
        })}</tbody></table></div>
        {!busy && !filtered.length && <Empty title={products.length ? "没有匹配的商品" : "还没有商品"} text={products.length ? "调整关键词、状态或分类后重试。" : "添加图片与商品信息，预览后保存并发布。"} />}
        {busy && <p className="catalog-loading" role="status">正在读取商品…</p>}
        <footer className="catalog-pagination"><span>已加载 {products.length} 件 · 筛选结果 {filtered.length} 件</span><div><Button secondary aria-label="上一页商品" disabled={currentPage <= 1} onClick={() => setPage(currentPage - 1)}><ChevronLeft size={16} /></Button><span>{currentPage} / {pages}</span><Button secondary aria-label="下一页商品" disabled={currentPage >= pages} onClick={() => setPage(currentPage + 1)}><ChevronRight size={16} /></Button>{cursor && <Button secondary disabled={busy} onClick={() => loadStoreContent(true)}>加载更多商品</Button>}</div></footer>
      </section>
    </>}
    {!busy && !error && !stores.length && <Empty title="还没有可管理的商店" text={all ? "先创建商店，再准备分类与交付模板。" : "获得商店授权后，就能在这里维护商品。"} />}
    {modal?.type.startsWith("store-") && <Modal title={modal.type === "store-create" ? "创建商店" : "商店资料"} close={() => setModal(null)}><StoreForm client={client} user={user} initial={modal.item || {}} onSaved={async () => { setModal(null); await loadStores(); }} /></Modal>}
    {(modal?.type === "brand" || modal?.type === "category") && <Modal title={modal.type === "brand" ? "新增品牌" : "新增分类"} close={() => setModal(null)}><ClassificationForm client={client} storeId={selected} kind={modal.type} onSaved={async () => { setModal(null); await loadStoreContent(); }} /></Modal>}
    {modal?.type === "product" && <Modal title={modal.item.productId ? "编辑商品" : "新增商品"} close={() => setModal(null)} wide className="product-editor-modal" guardClose dismissOnBackdrop={false}><ProductForm client={client} storeId={selected} initial={modal.item} brands={brands} categories={categories} templates={templates} onSaved={async ({ close }) => { if (close) setModal(null); await loadStoreContent(); }} /></Modal>}
    {modal?.type === "templates" && <Modal title="游戏内邮箱交付模板" close={() => setModal(null)} wide><DeliveryTemplateManagement client={client} storeId={selected} onChanged={loadStoreContent}/></Modal>}
    {modal?.type === "orders" && <Modal title="商店订单" close={() => setModal(null)} wide><BusinessPages client={client} user={user} type="ORDER" merchantStoreId={selected}/></Modal>}
    {modal?.type === "delete" && <Modal title="删除商品" dismissOnBackdrop={false} close={() => { if (!busy) setModal(null); }}><p>确认删除“{modal.item.draft.title}”？商品会停止销售并从管理列表移除，无法重新发布。已有订单仍可交付、领取或退款。</p>{error && <p className="auth-error" role="alert">{error}</p>}<div className="editor-actions"><Button secondary disabled={busy} onClick={() => setModal(null)}>取消</Button><Button danger disabled={busy} onClick={() => action(modal.item, "delete")}>{busy ? "正在删除…" : "确认删除"}</Button></div></Modal>}
    {modal?.type === "unlist" && <Modal title="下架商品" close={() => setModal(null)}><form onSubmit={(e) => { e.preventDefault(); action(modal.item, "unlist", reason); }}><p>确认下架“{modal.item.draft.title}”？已经成交的订单不受影响。</p><Field label="下架原因"><textarea required minLength={2} maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} /></Field>{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>确认下架</Button></form></Modal>}
  </div>;
}

function StoreForm({ client, user, initial, onSaved }) {
  const [value, setValue] = useState({ name: initial.name || "", intro: initial.intro || "", contactQq: initial.contactQq || user.qq, serviceHours: initial.serviceHours || "", notice: initial.notice || "" }), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const [logo,setLogo]=useState(initial.logo || null),[uploading,setUploading]=useState(false);
  useUnsavedChanges({value,logo:logo?.assetId},busy||uploading);
  const stable = useStableMutation();
  const save = async (e) => { e.preventDefault(); if(busy||uploading)return; setBusy(true); setError(""); try { const body = stable({ ...value, logoAssetId: logo?.assetId || null, coverAssetId: initial.cover?.assetId || null, ...(initial.storeId ? { expectedVersion: initial.version } : {}) }); await client.request(initial.storeId ? `/api/v1/merchant/stores/${encodeURIComponent(initial.storeId)}` : "/api/v1/admin/stores", { method: initial.storeId ? "PUT" : "POST", body }); await onSaved(); } catch (e) { setError(e.message); } finally { setBusy(false); } };
  return <form onSubmit={save} className="settings-stack"><StoreAvatarPicker client={client} storeId={initial.storeId} name={value.name} value={logo} onChange={setLogo} onBusy={setUploading} />{[["name", "商店名称", 60], ["intro", "商店简介", 300], ["contactQq", "联系 QQ", 12], ["serviceHours", "服务时间", 100], ["notice", "商店公告", 1000]].map(([key, label, max]) => <Field key={key} label={label} value={value[key]} onChange={(e) => setValue({ ...value, [key]: e.target.value })} required={["name", "contactQq"].includes(key)} maxLength={max} pattern={key === "contactQq" ? "[0-9]{5,12}" : undefined} />)}{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy||uploading}>保存商店</Button></form>;
}
function ClassificationForm({ client, storeId, kind, onSaved }) {
  const [name, setName] = useState(""), [order, setOrder] = useState(0), [busy, setBusy] = useState(false), [error, setError] = useState(""); const stable = useStableMutation();
  return <form onSubmit={async (e) => { e.preventDefault(); setBusy(true); setError(""); try { await client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}/${kind === "brand" ? "brands" : "categories"}`, { method: "POST", body: stable({ name, sortOrder: Number(order), active: true, ...(kind === "brand" ? { logoAssetId: null } : {}) }) }); await onSaved(); } catch (e) { setError(e.message); } finally { setBusy(false); } }}><Field label="名称" value={name} onChange={(e) => setName(e.target.value)} required maxLength={kind === "brand" ? 60 : 40} /><Field label="排序" type="number" min={0} max={9999} value={order} onChange={(e) => setOrder(e.target.value)} />{error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>保存</Button></form>;
}

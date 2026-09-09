import React, { useRef, useState } from "react";
import { Check, Save, Upload } from "lucide-react";
import { Badge, Button, Field } from "./components.jsx";
import { MediaPicker } from "./MediaPicker.jsx";
import { credit, id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

const onlyChoice = (values, key) => values.filter((v) => v.active).length === 1 ? values.find((v) => v.active)[key] : "";
export default function ProductEditor({ client, storeId, initial, brands, categories, templates, onSaved }) {
  const d = initial.draft || {}, templateRef = d.deliveryTemplateRef || onlyChoice(templates, "templateRef"), template = templates.find((t) => t.templateRef === templateRef);
  const [value, setValue] = useState({ title: d.title || "", subtitle: d.subtitle || "", description: d.description || "", brandId: d.brandId || onlyChoice(brands, "brandId"), categoryId: d.categoryId || onlyChoice(categories, "categoryId"), price: d.price || "", deliveryTemplateRef: templateRef, deliverySummary: d.deliverySummary || "通过游戏内邮箱领取", estimatedDelivery: d.estimatedDelivery || "付款后自动发放，以游戏内邮箱为准", inventoryPolicy: d.inventoryPolicy || "FINITE", stock: d.stock ?? 1, limitPerOrder: d.limitPerOrder || 1, posterTone: d.posterTone || "LIGHT", accentColor: d.accentColor || "#4C78BD", sortOrder: d.sortOrder || 0 }),
    [included, setIncluded] = useState(d.includedItems?.join("\n") || template?.summary || ""), [images, setImages] = useState(initial.draftImages || (d.galleryAssetIds || []).map((assetId) => initial.published?.images?.find((a) => a.assetId === assetId) || { assetId, altText: "" })), [busy, setBusy] = useState(false), [error, setError] = useState(""), [saved, setSaved] = useState(false);
  const requests = useRef(new Map()), savedDraft = useRef(null), lastProduct = useRef(initial);
  const editorValue = { value, included, imageIds: images.map((image) => image.assetId) };
  const guard = useUnsavedChanges(editorValue, busy);
  const stable = (key, body) => { const fingerprint = `${key}:${JSON.stringify(body)}`; if (!requests.current.has(fingerprint)) requests.current.set(fingerprint, { ...body, clientRequestId: id() }); return requests.current.get(fingerprint); };
  const field = (key, label, props = {}) => <Field label={label} value={value[key]} onChange={(e) => setValue((old) => ({ ...old, [key]: props.type === "number" ? Number(e.target.value) : e.target.value }))} {...props} />;
  const save = async (event) => {
    event.preventDefault(); if (busy) return; const publish = event.nativeEvent.submitter?.value === "publish"; setBusy(true); setError("");
    try {
      if (!images.length) throw new Error("请至少上传一张商品图片。");
      const price = String(value.price).trim(); if (!/^(0|[1-9]\d*)(\.\d{1,2})?$/.test(price) || /^0(?:\.0{1,2})?$/.test(price) || price.length > 20) throw new Error("请输入大于 0 的售价，最多两位小数。");
      const content = { ...value, price, coverAssetId: images[0].assetId, galleryAssetIds: images.map((a) => a.assetId), galleryAltTexts: images.map((a) => a.altText || value.title), contentBlocks: d.contentBlocks || [], includedItems: included.split("\n").map((s) => s.trim()).filter(Boolean), badges: d.badges || [] };
      const fingerprint = JSON.stringify(content);
      let product = savedDraft.current?.fingerprint === fingerprint ? savedDraft.current.product : null;
      if (!product) {
        const existing = lastProduct.current;
        const body = stable("save", { ...(existing.productId ? { expectedVersion: existing.version } : {}), content });
        const r = await client.request(existing.productId ? `/api/v1/merchant/products/${encodeURIComponent(existing.productId)}` : `/api/v1/merchant/stores/${encodeURIComponent(storeId)}/products`, { method: existing.productId ? "PUT" : "POST", body });
        product = r.data; lastProduct.current = product; savedDraft.current = { fingerprint, product }; setSaved(true); guard.markSaved(editorValue);
      }
      if (publish) await client.request(`/api/v1/merchant/products/${encodeURIComponent(product.productId)}/publish`, { method: "POST", body: stable("publish", { expectedVersion: product.version }) });
      await onSaved();
    } catch (e) { setError(`${savedDraft.current ? "草稿已保存。" : ""}${e.message}`); } finally { setBusy(false); }
  };
  const chooseTemplate = (e) => { const selected = templates.find((t) => t.templateRef === e.target.value); setValue({ ...value, deliveryTemplateRef: e.target.value }); if (!included.trim() && selected?.summary) setIncluded(selected.summary); };
  return <form className="product-editor" onSubmit={save} onInvalidCapture={(event) => { const section=event.target.closest("details");if(section)section.open=true; }}>
    <fieldset disabled={busy}>
      <section className="editor-section"><div className="editor-section-title"><span>1</span><h3>商品信息</h3></div>
        <div className="form-grid">{field("title", "商品标题", { required: true, maxLength: 80 })}{field("subtitle", "一句话简介", { required: true, maxLength: 200 })}</div>
        <Field label="详细介绍"><textarea rows={4} required maxLength={10000} value={value.description} onChange={(e) => setValue({ ...value, description: e.target.value })} /></Field>
        <div className="form-grid"><Field label="品牌"><select required value={value.brandId} onChange={(e) => setValue({ ...value, brandId: e.target.value })}><option value="">选择品牌</option>{brands.filter((x) => x.active).map((x) => <option key={x.brandId} value={x.brandId}>{x.name}</option>)}</select></Field><Field label="分类"><select required value={value.categoryId} onChange={(e) => setValue({ ...value, categoryId: e.target.value })}><option value="">选择分类</option>{categories.filter((x) => x.active).map((x) => <option key={x.categoryId} value={x.categoryId}>{x.name}</option>)}</select></Field></div>
        <MediaPicker client={client} images={images} onChange={setImages} purpose="STORE_MEDIA" businessType="STORE" businessRef={storeId} max={20} onBusy={setBusy} />
      </section>
      <section className="editor-section"><div className="editor-section-title"><span>2</span><h3>售价与交付</h3></div>
        <div className="form-grid">{field("price", "售价（信用点）", { required: true, inputMode: "decimal", maxLength: 20 })}{field("stock", "库存数量", { type: "number", required: true, min: 0, max: 999999, disabled: value.inventoryPolicy === "UNLIMITED" })}</div>
        <Field label="游戏内邮箱交付模板"><select required value={value.deliveryTemplateRef} onChange={chooseTemplate}><option value="">选择交付模板</option>{templates.filter((x) => x.active).map((x) => <option key={x.templateRef} value={x.templateRef}>{x.name} · {x.summary}</option>)}</select></Field>
        {!templates.some((x) => x.active) && <p className="notice-box">商店尚未配置交付模板，请先在商店管理中准备模板。</p>}
        <Field label="包含内容（每行一项）"><textarea required rows={3} value={included} onChange={(e) => setIncluded(e.target.value)} /></Field>
        <div className="form-grid">{field("deliverySummary", "交付说明", { required: true, maxLength: 500 })}{field("estimatedDelivery", "预计送达", { required: true, maxLength: 150 })}</div>
      </section>
      <details className="editor-section"><summary>更多设置 · 限购、库存方式与展示</summary><div className="form-grid">
        <Field label="库存方式"><select value={value.inventoryPolicy} onChange={(e) => setValue({ ...value, inventoryPolicy: e.target.value })}><option value="FINITE">有限库存</option><option value="UNLIMITED">不限库存</option></select></Field>
        {field("limitPerOrder", "单笔限购", { type: "number", required: true, min: 1, max: 999 })}{field("sortOrder", "展示排序", { type: "number", required: true, min: 0, max: 99999 })}
        <Field label="商品展示底色"><select value={value.posterTone} onChange={(e) => setValue({ ...value, posterTone: e.target.value })}><option value="LIGHT">浅色</option><option value="DARK">深色</option></select></Field>{field("accentColor", "强调色", { type: "color" })}
      </div></details>
    </fieldset>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="editor-actions"><span className="editor-save-hint">{busy ? "正在保存，请稍候…" : saved && !guard.dirty ? <><Check size={15} />草稿已保存</> : guard.dirty ? "有未保存的修改" : "填写完成后即可发布"}</span><Button secondary type="submit" value="draft" disabled={busy}><Save size={16} />保存草稿</Button><Button type="submit" value="publish" disabled={busy}><Upload size={16} />保存并发布</Button></div>
  </form>;
}

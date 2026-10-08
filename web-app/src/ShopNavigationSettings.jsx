import React, { useEffect, useRef, useState } from "react";
import { Button, Field } from "./components.jsx";
import { MediaPicker } from "./MediaPicker.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

export function CategoryHierarchy({ client, storeId, categories, onSaved }) {
  const [selected, setSelected] = useState(categories[0]?.categoryId || "");
  const current = categories.find(c => c.categoryId === selected);
  return <div className="settings-stack">
    <p>一级分类显示在 App 常驻分类栏；二级分类显示在对应分类页，点击后搜索该分类名称。</p>
    <Field label="要编辑的分类"><select value={selected} onChange={e => setSelected(e.target.value)}>{categories.map(c => <option key={c.categoryId} value={c.categoryId}>{c.name}</option>)}</select></Field>
    {current && <CategoryFields key={`${current.categoryId}:${current.version}`} client={client} storeId={storeId} kind="category" initial={current} categories={categories} onSaved={onSaved} />}
  </div>;
}

export function CategoryFields({ client, storeId, kind, initial = {}, categories = [], onSaved }) {
  const [name, setName] = useState(initial.name || ""), [order, setOrder] = useState(initial.sortOrder || 0), [parent, setParent] = useState(initial.parentCategoryId || ""), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const request = useRef(null);
  const guard = useUnsavedChanges({ name, order, parent }, busy);
  const roots = categories.filter(c => c.active && !c.parentCategoryId && c.categoryId !== initial.categoryId);
  const hasChildren = categories.some(c => c.parentCategoryId === initial.categoryId);
  return <form className="settings-stack" onSubmit={async e => {
    e.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      const body = { name, sortOrder: Number(order), active: initial.active ?? true, ...(kind === "brand" ? { logoAssetId: null } : { parentCategoryId: parent || null }), ...(initial.categoryId ? { expectedVersion: initial.version } : {}) };
      const fingerprint = JSON.stringify(body);
      if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, body: { ...body, clientRequestId: id() } };
      await client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}/${kind === "brand" ? "brands" : "categories"}${initial.categoryId ? `/${encodeURIComponent(initial.categoryId)}` : ""}`, { method: initial.categoryId ? "PUT" : "POST", body: request.current.body });
      guard.markSaved({ name, order, parent }); await onSaved();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  }}>
    <fieldset disabled={busy} style={{ border: 0, padding: 0, margin: 0 }}>
      <Field label="名称" value={name} onChange={e => setName(e.target.value)} required maxLength={kind === "brand" ? 60 : 40} />
      {kind === "category" && <Field label="所属一级分类" hint={hasChildren ? "当前分类包含子分类，需先移出子分类才能更改归属。" : "选择“作为一级分类”，即可在商城顶栏展示。"}><select disabled={hasChildren} value={parent} onChange={e => setParent(e.target.value)}><option value="">作为一级分类</option>{roots.map(c => <option key={c.categoryId} value={c.categoryId}>{c.name}</option>)}</select></Field>}
      <Field label="排序" type="number" min={0} max={9999} value={order} onChange={e => setOrder(e.target.value)} />
    </fieldset>
    {error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>保存</Button>
  </form>;
}

export function ShopBannerSettings({ client, storeId, onSaved }) {
  const [home, setHome] = useState(null), [images, setImages] = useState([]), [loading, setLoading] = useState(true), [busy, setBusy] = useState(false), [error, setError] = useState("");
  const request = useRef(null);
  const path = `/api/v1/merchant/stores/${encodeURIComponent(storeId)}/homepage`;
  useEffect(() => {
    let alive = true;
    client.request(path).then(({ data }) => {
      if (!alive) return;
      setHome(data);
      setImages((data.sections || []).filter(s => s.layout === "BANNER").sort((a,b) => a.sortOrder-b.sortOrder).map(s => ({ ...data.banners?.find(b => b.sectionId === s.sectionId)?.image, assetId: s.bannerAssetId, altText: s.title, sectionId: s.sectionId })));
    }).catch(e => { if (!alive) return; if (e.status === 404) setHome({ version: 1, intro: "", sections: [], brandIds: [], categoryIds: [] }); else setError(e.message); }).finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, [client, path]);
  if (loading) return <p role="status">正在读取活动图…</p>;
  if (!home) return <p className="auth-error" role="alert">{error}</p>;
  return <form className="settings-stack" onSubmit={async e => {
    e.preventDefault(); if (busy) return; setBusy(true); setError("");
    try {
      const sections = [...home.sections.filter(s => s.layout !== "BANNER"), ...images.map((image, index) => ({ sectionId: image.sectionId || `banner_${image.assetId}`, title: (image.altText || "商城活动").slice(0,80), layout: "BANNER", productIds: [], bannerAssetId: image.assetId, sortOrder: index }))];
      if (!sections.length) sections.push({ sectionId: "products", title: "商品", layout: "PRODUCT_GRID", productIds: [], bannerAssetId: null, sortOrder: 0 });
      const body = { expectedVersion: home.version, intro: home.intro || "", brandIds: home.brandIds || [], categoryIds: home.categoryIds || [], sections };
      const fingerprint = JSON.stringify(body);
      if (request.current?.fingerprint !== fingerprint) request.current = { fingerprint, body: { ...body, clientRequestId: id() } };
      const result = await client.request(path, { method: "PUT", body: request.current.body });
      setHome(result.data); await onSaved();
    } catch (e) { setError(e.message); } finally { setBusy(false); }
  }}>
    <p>活动图显示在商城首页分类栏下方，支持横向滑动。建议使用约 2.65 : 1 的横图；图片说明用于无障碍阅读。</p>
    <MediaPicker client={client} images={images} onChange={setImages} purpose="STORE_MEDIA" businessType="STORE" businessRef={storeId} max={Math.min(8,20-home.sections.filter(s => s.layout !== "BANNER").length)} onBusy={setBusy} />
    {error && <p className="auth-error" role="alert">{error}</p>}<Button type="submit" disabled={busy}>保存活动图</Button>
  </form>;
}

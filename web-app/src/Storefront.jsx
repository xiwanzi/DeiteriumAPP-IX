import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { ArrowRight, CircleAlert, CircleCheck, RotateCw, Search, X, ShoppingBag, LoaderCircle, LayoutGrid, List } from "lucide-react";
import { credit } from "./format.js";
import { productInventory } from "./productInventory.js";
import ProductArtwork from "./ProductArtwork.jsx";
import "./storefront.css";

function ProductTile({ item, categories, select, onAdd, adding, compact = false }) {
  const content = item.content;
  const sale = content.discountRate != null && content.discountRate < 10000;
  const price = item.effectivePrice ?? content.price;
  const soldOut = productInventory(item).stock <= 0;
  const art = <ProductArtwork className="shop-art" image={item.images?.[0]} lazy />;
  const category = categories.get(content.categoryId);
  const priceLine = <span className="shop-price"><strong>{credit(price)}</strong> <small>信用点</small>{sale && !compact && <del>{credit(content.price)}</del>}</span>;
  if (compact) return <button type="button" className="shop-pick" onClick={()=>select(item)}>
    {art}<span className="shop-copy"><strong className="shop-title">{content.title}</strong>{priceLine}</span>
  </button>;
  return <article className="shop-item">
    <button type="button" className="shop-tile foundation-card" aria-label={`查看${content.title}，${credit(price)} 信用点`} onClick={()=>select(item)}>
      {art}<span className="shop-copy">{category && <small>{category}</small>}<strong className="shop-title">{content.title}</strong></span>
    </button>
    <div className="shop-tile-footer">
      {priceLine}
      <button type="button" className="shop-add" aria-label={`加购 ${content.title}`} title={soldOut?"已售罄":"加入购物袋"} disabled={soldOut || !!adding} onClick={()=>onAdd(item,1)}>
        {soldOut ? <span>售罄</span> : adding===item.productId ? <LoaderCircle size={16} className="spin"/> : <ShoppingBag size={17}/>}
      </button>
    </div>
  </article>;
}

function CartFeedback({ message, error, dismiss, openCart }) {
  const [paused, setPaused] = useState(false);
  useEffect(() => {
    if (error || paused) return;
    const timer = setTimeout(dismiss, 4500);
    return () => clearTimeout(timer);
  }, [message, error, paused, dismiss]);
  const notice = <aside className={`shop-cart-notice ${error ? "has-error" : ""}`} role={error ? "alert" : "status"}
    onPointerEnter={() => setPaused(true)} onPointerLeave={e => { if (!e.currentTarget.contains(document.activeElement)) setPaused(false); }}
    onFocus={() => setPaused(true)} onBlur={e => { if (!e.currentTarget.contains(e.relatedTarget) && !e.currentTarget.matches(":hover")) setPaused(false); }}>
    {error ? <CircleAlert size={16}/> : <CircleCheck size={16}/>}<span>{message}</span>
    <button type="button" onClick={openCart} aria-label="查看购物袋"><span>查看购物袋</span><ArrowRight size={15}/></button>
    <button type="button" onClick={dismiss} aria-label="关闭加购提示"><X size={15}/></button>
  </aside>;
  const host = !error && document.getElementById("shop-feedback-host");
  return host ? createPortal(notice, host) : notice;
}

export default function Storefront({ client, items, select, onAdd, adding, cartNotice, noticeIsError, dismissNotice, openCart, query, setQuery, filter, setFilter, busy, error, retry }) {
  const search = useRef(null);
  const [layout, setLayout] = useState(() => {
    try { return localStorage.getItem("deuterium-store-layout") === "list" ? "list" : "grid"; } catch { return "grid"; }
  });
  const chooseLayout = value => {
    setLayout(value);
    try { localStorage.setItem("deuterium-store-layout",value); } catch {}
  };
  useEffect(() => {
    const focus = () => search.current?.focus();
    window.addEventListener("deuterium:search", focus);
    return () => window.removeEventListener("deuterium:search", focus);
  }, []);
  const [categories, setCategories] = useState([]), [categoryError, setCategoryError] = useState(""), [categoryLoad, setCategoryLoad] = useState(0);
  const [picks, setPicks] = useState([]), [pickError, setPickError] = useState(""), [batch, setBatch] = useState(0), [picking, setPicking] = useState(true);
  useEffect(() => {
    const controller = new AbortController();
    const load = async () => {
      const rows = [], seen = new Set();
      let cursor;
      do {
        const response = await client.request("/api/v1/store/categories?limit=100" + (cursor ? "&cursor=" + encodeURIComponent(cursor) : ""), { signal: controller.signal });
        rows.push(...response.data);
        cursor = response.page?.nextCursor;
        if (cursor && seen.has(cursor)) throw new Error("分类加载中断，请重试。");
        seen.add(cursor);
      } while (cursor && !controller.signal.aborted);
      if (!controller.signal.aborted) setCategories(rows.sort((left,right) => (left.sortOrder ?? 0) - (right.sortOrder ?? 0) || left.name.localeCompare(right.name, "zh-CN")));
    };
    setCategoryError("");
    load().catch(e => { if (!controller.signal.aborted) setCategoryError(e.message); });
    return () => controller.abort();
  }, [client, categoryLoad]);
  useEffect(() => {
    const controller = new AbortController();
    setPicking(true); setPickError("");
    client.request("/api/v1/store/recommendations", {signal:controller.signal})
      .then(response => { if (!controller.signal.aborted) setPicks(response.data); })
      .catch(e => { if (!controller.signal.aborted) setPickError(e.message); })
      .finally(() => { if (!controller.signal.aborted) setPicking(false); });
    return () => controller.abort();
  }, [client, batch]);
  const names = new Map(categories.map(category => [category.categoryId, category.name]));
  const groups = [...new Set(categories.filter(category => !category.parentCategoryId).map(category => category.name))];
  const children = filter.group ? [...new Set(categories.filter(category => names.get(category.parentCategoryId) === filter.group).map(category => category.name))] : [];
  const home = !filter.group && !query.trim() && !filter.inStock;
  return <section className="storefront" aria-label="官方商城">
    <header className="shop-toolbar"><h1>官方商城</h1>
      <label className="shop-search"><Search size={18}/><input ref={search} aria-label="搜索官方商城" placeholder="搜索商品、品牌或分类" value={query} maxLength={80} onChange={event=>setQuery(event.target.value)} />
      {query && <button type="button" aria-label="清除商品搜索" onClick={()=>setQuery("")}><X size={16}/></button>}</label>
    </header>
    {cartNotice && <CartFeedback message={cartNotice} error={noticeIsError} dismiss={dismissNotice} openCart={openCart} />}
    <nav className="shop-groups" aria-label="商品分类">{["", ...groups].map(group=><button type="button" key={group} aria-pressed={group===filter.group} onClick={()=>setFilter({...filter,group, child:""})}>{group || "全部"}</button>)}</nav>
    {children.length>0 && <div className="shop-children" role="group" aria-label="子分类">{["", ...children].map(child=><button type="button" key={child} aria-pressed={child===filter.child} onClick={()=>setFilter({...filter,child})}>{child || "不限子类"}</button>)}</div>}
    {categoryError && <p className="shop-error" role="alert">分类暂未加载 <button onClick={()=>setCategoryLoad(value=>value+1)}>重试</button></p>}
    <div className={`shop-results-layout ${home && (picks.length || picking || pickError) ? "with-picks" : ""}`}>
    {home && (picks.length>0 || picking || pickError) && <section className="shop-discovery" aria-label="首页推荐">
      <header><div><h2>首页推荐</h2><small>近 30 天热销 · 随机发现</small></div><button type="button" className="shop-change" disabled={picking} onClick={()=>setBatch(value=>value+1)}><RotateCw size={15}/>换一批</button></header>
      {picks.length>0 && <div className="shop-picks">{picks.map(item=><ProductTile key={item.productId} compact item={item} categories={names} select={select}/>)}</div>}
      {picking && !picks.length && <p role="status">正在挑选商品…</p>}
      {pickError && <p role="alert" className="shop-error">推荐暂未更新，可点击「换一批」重试。</p>}
    </section>}
    <section className="shop-results" aria-label="商品目录"><header className="shop-section-head"><h2>{filter.child || filter.group || (query.trim() ? "搜索结果" : "全部商品")}</h2>
      {(filter.group || filter.inStock || query.trim()) && <button type="button" onClick={()=>{setFilter({group:"",child:""});setQuery("");}}>清除筛选 <X size={14}/></button>}
      <label className="shop-availability"><input type="checkbox" checked={!!filter.inStock} onChange={event=>setFilter({...filter,inStock:event.target.checked})}/>仅看有货</label>
      <div className="shop-view" role="group" aria-label="商品展示方式">{[["grid","商品网格",LayoutGrid],["list","紧凑列表",List]].map(([value,label,Icon])=><button key={value} type="button" aria-label={label} title={label} aria-pressed={layout===value} onClick={()=>chooseLayout(value)}><Icon size={17}/></button>)}</div></header>
    {error && <p className="shop-error" role="alert">{error} <button disabled={busy} onClick={retry}>重试</button></p>}
    <div className={`shop-catalog-grid shop-products ${layout==="list"?"is-list":""}`}>{items.map(item=><ProductTile key={item.productId} item={item} categories={names} select={select} onAdd={onAdd} adding={adding}/>)}</div>
    </section></div>
  </section>;
}

import { usePlayerNameState } from "./use-player-names.js";
import React, { useCallback, useContext, useEffect, useRef, useState } from "react";
import { ShoppingCartContext } from "./ShoppingCartContext.jsx";
import { RefreshCw, Search, Package, Plus } from "lucide-react";
import { PageHead, Button, Empty, Badge, Modal } from "./components.jsx";
import { credit } from "./format.js";
import { id } from "./format.js";
import { ListingForm } from "./CatalogManagement.jsx";
import { productInventory } from "./productInventory.js";
import Storefront from "./Storefront.jsx";
import ProductDetails from "./ProductDetails.jsx";
import PurchaseFlow from "./PurchaseFlow.jsx";
import { resourceId } from "./business.js";

const collections = {
  "/": { endpoint: "/store/products", title: "官方商城", subtitle: "发现来自官方商店的精选好物。", key: "productId", empty: "还没有上架商品" },
  "/market": { endpoint: "/market/listings", title: "玩家市场", subtitle: "交换好物，分享创造。", key: "listingId", empty: "还没有在售商品" },
  "/commissions": { endpoint: "/commissions", title: "委托大厅", subtitle: "让想法遇见愿意一起完成它的人。", key: "commissionId", empty: "当前没有待接取委托" },
  "/announcements": { endpoint: "/announcements", title: "社区公告", subtitle: "关于这个世界的最新消息。", key: "announcementId", empty: "当前没有公告" },
  "/notifications": { endpoint: "/notifications", title: "通知", subtitle: "属于你的消息与业务提醒。", key: "notificationId", empty: "还没有通知" },
};
export function serviceError(error) {
  return error.status === 404 || error.code === "NOT_IMPLEMENTED" || error.code === "CAPABILITY_UNAVAILABLE"
    ? "这项服务暂未开放，请稍后再来。"
    : error.message;
}
export function ContentBlocks({ blocks = [], media = [] }) {
  return blocks.map((block, index) => {
    const key = block.blockId || index;
    if (block.type === "HEADING") return <h3 key={key}>{block.heading}</h3>;
    if (block.type === "KEY_VALUE_LIST") return <dl key={key} className="detail-list">{block.rows?.map((row, n) => <div key={n}><dt>{row.label}</dt><dd>{row.value}</dd></div>)}</dl>;
    if (block.type === "IMAGE") { const asset=media.find((item)=>item.assetId===block.assetId); return asset?.url ? <img key={key} src={asset.url} alt={block.altText || asset.altText || "正文图片"} style={{width:"100%",borderRadius:12}} /> : <p key={key} className="muted">{block.altText || "图片暂不可用"}</p>; }
    return <p key={key} style={{ whiteSpace: "pre-wrap" }}>{block.text}</p>;
  });
}

export default function RemoteCollection({ client, path, user, onNotificationTarget, navigate }) {
  const shoppingCart = useContext(ShoppingCartContext);
  const config = collections[path] || collections["/"],
    [items, setItems] = usePlayerNameState(client, []), [cursor, setCursor] = useState(null), [busy, setBusy] = useState(true),
    [error, setError] = useState(""), [query, setQuery] = useState(""), [serverQuery, setServerQuery] = useState(""), [selected, setSelected] = usePlayerNameState(client, null), [mine, setMine] = useState(false), [editor, setEditor] = useState(null), [unlist, setUnlist] = useState(null), [purchase, setPurchase] = usePlayerNameState(client, null), [quantity, setQuantity] = useState(1), [bagMessage, setBagMessage] = useState("");
  const [filter, setFilter] = useState({group:"", child:""});
  const [adding, setAdding] = useState(""), [bagError, setBagError] = useState("");
  const dismissNotice = useCallback(() => { setBagError(""); setBagMessage(""); }, []);
  const actionKeys = useRef({});
  const generation = useRef(0);
  const loading = useRef(false), sentinel = useRef(null);
  const catalog = path === "/" || path === "/market";
  const load = async (more = false) => {
    if (more && (loading.current || !cursor)) return;
    const current = ++generation.current;
    loading.current = true;
    setBusy(true); setError("");
    try {
      const endpoint = path === "/market" && mine ? "/market/me/listings" : config.endpoint;
      const params = new URLSearchParams({limit:"40"});
      if (catalog && serverQuery) params.set("q", serverQuery);
      if (path === "/" && filter.group) {
        params.set("categoryGroup", filter.group);
        if (filter.child) params.set("subcategory", filter.child);
      }
      if (path === "/" && filter.inStock) params.set("inStock", "true");
      if (more && cursor) params.set("cursor", cursor);
      const r = await client.request(`/api/v1${endpoint}?${params}`);
      if (current !== generation.current) return;
      if (!Array.isArray(r.data)) throw new Error("服务返回的列表格式不正确，请稍后重试。");
      if (r.page?.hasMore && (!r.page.nextCursor || (more && r.page.nextCursor === cursor))) throw new Error("分页结果异常，请刷新重试。");
      setItems((old) => more ? [...new Map([...old, ...r.data].map((item) => [item[config.key], item])).values()] : r.data);
      setCursor(r.page?.nextCursor);
    } catch (e) { if (current === generation.current) setError(serviceError(e)); }
    finally { if (current === generation.current) { loading.current = false; setBusy(false); } }
  };
  useEffect(() => { setQuery(""); setServerQuery(""); setFilter({group:"",child:""}); setSelected(null); }, [path, mine]);
  useEffect(() => { if (!catalog) return; const timer = setTimeout(() => setServerQuery(query.trim()), 250); return () => clearTimeout(timer); }, [query, catalog]);
  useEffect(() => { setItems([]); setCursor(null); load(); return () => { generation.current++; }; }, [path, mine, serverQuery, filter.group, filter.child, filter.inStock]);
  useEffect(() => {
    if (!catalog || !cursor || busy || error || !sentinel.current || !globalThis.IntersectionObserver) return;
    const observer = new IntersectionObserver(([entry]) => { if (entry.isIntersecting) load(true); }, { rootMargin: "1000px" });
    observer.observe(sentinel.current);
    return () => observer.disconnect();
  }, [catalog, cursor, busy, error, serverQuery, path, filter.group, filter.child, filter.inStock]);
  useEffect(() => client.onAccountDeletions((refs) => {
    setItems((old) => old.filter((item) => !refs.has(item.seller?.playerRef)));
    setSelected((old) => refs.has(old?.seller?.playerRef) ? null : old);
    setPurchase((old) => refs.has(old?.item?.seller?.playerRef) ? null : old);
    load();
  }), [client, path, mine, serverQuery, filter.group, filter.child, filter.inStock]);
  const markNotification = async (item) => {
    if (item.readAt) return; const scope = `notification:${item.notificationId}`; actionKeys.current[scope] ||= id();
    try { const r = await client.request("/api/v1/notifications/read", {method:"POST",body:{clientRequestId:actionKeys.current[scope],notificationIds:[item.notificationId]}}); if (typeof r.data.readCount !== "number") throw new Error("通知状态尚未确认。"); const changed={...item,readAt:r.serverTime || new Date().toISOString()};setItems((old)=>old.map((x)=>x.notificationId===item.notificationId?changed:x));setSelected(changed); }
    catch(e){setError(e.message);}
  };
  const select = (item) => { setSelected(item); setQuantity(1); setBagMessage(""); setBagError(""); if (path === "/notifications") markNotification(item); };
  const bagLock = useRef(false), uncertainAdds = useRef(new Map());
  const addToBag = async (product, count = 1) => {
    if (bagLock.current) return;
    const productId = product.productId;
    bagLock.current = true; setAdding(productId); setBagMessage(""); setBagError("");
    try {
      let pending = uncertainAdds.current.get(productId);
      if (pending && pending.count !== count) throw new Error("上一次加购仍待确认，请先查看购物袋。");
      if (!pending) {
        const cart = shoppingCart ? await shoppingCart.readCart() : (await client.request("/api/v1/store/cart")).data;
        const previous = cart.items.find(row=>row.productId===productId)?.quantity || 0;
        const next = previous + count;
        if (next > productInventory(product).max) throw new Error("购物袋数量将超过可购上限。");
        pending = { count, body: {clientRequestId:id(), expectedVersion:cart.version, quantity:next} };
        uncertainAdds.current.set(productId, pending);
      }
      const response = await client.request(`/api/v1/store/cart/items/${encodeURIComponent(productId)}`, {method:"PUT", body:pending.body, expectedPlayerRef:user.playerRef});
      shoppingCart?.applyCart(response.data);
      uncertainAdds.current.delete(productId);
      setBagMessage("已加入购物袋");
    } catch(e) {
      if (e.status >= 400 && e.status < 500 && ![408,429].includes(e.status)) uncertainAdds.current.delete(productId);
      setBagError(e.message);
    } finally { bagLock.current = false; setAdding(""); }
  };
  const unlistItem = async () => { setBusy(true);setError("");const scope=`unlist:${unlist.listingId}:${unlist.version}`;actionKeys.current[scope] ||= id();try{await client.request(`/api/v1/market/listings/${encodeURIComponent(unlist.listingId)}/unlist`,{method:"POST",body:{clientRequestId:actionKeys.current[scope],expectedVersion:unlist.version}});setUnlist(null);setSelected(null);await load();}catch(e){setError(e.message);}finally{setBusy(false);} };
  const content = selected?.content || selected;
  const visible = catalog ? items : items.filter((item) => `${item.title || item.content?.title || ""} ${item.summary || item.subtitle || item.content?.description || ""}`.toLowerCase().includes(query.toLowerCase()));
  return <>
    {path === "/" ? <Storefront client={client} items={visible} select={select} onAdd={addToBag} adding={adding} cartNotice={!selected && (bagError || bagMessage)} noticeIsError={Boolean(bagError)} dismissNotice={dismissNotice} openCart={()=>navigate("/cart")} query={query} setQuery={setQuery} filter={filter} setFilter={setFilter} busy={busy} error={error} retry={()=>load()} /> : <>
    <PageHead eyebrow="DEUTERIUM COMMUNITY" title={config.title} subtitle={config.subtitle}><Button secondary disabled={busy} onClick={() => load()}><RefreshCw size={16} />刷新</Button>{path==="/market"&&<><Button secondary onClick={()=>setMine(!mine)}>{mine?"发现好物":"我的发布"}</Button><Button onClick={()=>setEditor({})}><Plus size={16}/>发布商品</Button></>}</PageHead>
    <label className="search-input" style={{ maxWidth: 480, marginBottom: 24 }}><Search size={17} /><input aria-label={`搜索${config.title}`} placeholder={`搜索${catalog ? "全部商品" : `已加载的${config.title}`}`} maxLength={catalog ? 80 : undefined} value={query} onChange={(e) => setQuery(e.target.value)} /></label>
    {error && <div className="notice-box" role="alert">{error}</div>}
    <div className={`foundation-grid ${path === "/market" ? "shop-catalog-grid" : ""}`}>{visible.map((item) => {
      const c = item.content || item, image = item.images?.[0] || item.photos?.[0] || item.cover;
      return <button className="foundation-card" style={{ textAlign: "left", color: "inherit", cursor: "pointer" }} key={item[config.key]} onClick={() => select(item)}>
        {image?.url ? <img src={image.url} loading="lazy" decoding="async" alt={image.altText || c.title} style={{ width: "100%", aspectRatio: "16/10", objectFit: path === "/market" ? "contain" : "cover", borderRadius: 12 }} /> : <Package size={24} />}
        {item.pinned && <Badge>置顶</Badge>}{path==="/market"&&mine&&<Badge tone={item.active?"sage":"neutral"}>{item.active?"在售":"已下架"}</Badge>}{path==="/notifications"&&!item.readAt&&<Badge>未读</Badge>}<h3 style={{ marginTop: 16 }}>{c.title}</h3><p>{c.subtitle || c.description || item.summary || item.body}</p>
        {(c.price !== undefined || c.reward !== undefined) && <strong className="price">{credit(item.effectivePrice ?? c.price ?? c.reward)}{c.discountRate < 10000 && <del className="sale-original">{credit(c.price)}</del>}<small>信用点</small></strong>}
      </button>;
    })}</div></>}
    {!busy && !error && !visible.length && <Empty title={query || filter.group || filter.inStock ? "没有匹配的结果" : config.empty} text={query || filter.group || filter.inStock ? "试试其他分类或关键词。" : "新的内容发布后会出现在这里。"} />}
    {busy && <p className="muted" role="status">正在读取…</p>}
    {cursor && <div ref={sentinel}><Button secondary disabled={busy} onClick={() => load(true)}>{error ? "重试加载" : "加载更多"}</Button></div>}
    {selected && <Modal title={path === "/" || path === "/market" ? "商品详情" : content.title} close={() => setSelected(null)} wide={path === "/" || path === "/market"} className={path === "/" || path === "/market" ? "product-detail-modal" : ""}>
      {path === "/" || path === "/market" ? <ProductDetails key={selected.productId || selected.listingId} product={selected} official={path === "/"} quantity={quantity} setQuantity={setQuantity} busy={Boolean(adding) || (path !== "/" && busy)} error={bagError || error} bagMessage={bagMessage} onAdd={()=>addToBag(selected,quantity)} onBuy={() => {setPurchase({item:selected,quantity});setSelected(null);}} owned={path === "/market" && selected.seller?.playerRef === user?.playerRef} onEdit={() => {setEditor(selected);setSelected(null);}} onUnlist={() => {setUnlist(selected);setSelected(null);}} /> : <>
        {(selected.images || selected.photos || (selected.cover ? [selected.cover] : [])).map((asset) => <img key={asset.assetId} src={asset.url} alt={asset.altText || content.title} style={{width:"100%",borderRadius:12,marginBottom:12}} />)}
        <p className="description" style={{whiteSpace:"pre-wrap"}}>{content.description || content.subtitle || selected.summary || selected.body}</p>
        <ContentBlocks blocks={content.contentBlocks || content.detailBlocks || []} media={selected.media || selected.images || []} />
        {selected.publishedAt && <p className="muted">{new Date(selected.publishedAt).toLocaleString("zh-CN")}</p>}
        {path === "/notifications" && <><div className="button-row">{!selected.readAt && <Button secondary onClick={() => markNotification(selected)}>标记已读</Button>}{selected.target && onNotificationTarget && <Button onClick={() => onNotificationTarget(selected.target)}>查看相关内容</Button>}</div>{error && <p className="auth-error" role="alert">{error}</p>}</>}
      </>}
    </Modal>}
    {editor&&<Modal title={editor.listingId?"编辑玩家商品":"发布玩家商品"} close={()=>setEditor(null)} wide guardClose dismissOnBackdrop={false}><ListingForm client={client} user={user} initial={editor} onSaved={async()=>{setEditor(null);await load();}}/></Modal>}
    {unlist&&<Modal title="确认下架" close={()=>setUnlist(null)}><p>下架“{unlist.title}”后，其他玩家将无法购买。</p>{error&&<p className="auth-error" role="alert">{error}</p>}<Button disabled={busy} onClick={unlistItem}>确认下架</Button></Modal>}
    {purchase&&<Modal title="确认购买" close={()=>setPurchase(null)} className="checkout-modal"><PurchaseFlow client={client} user={user} channel={path==="/market"?"PLAYER_MARKET":"OFFICIAL_STORE"} listing={path==="/market"?purchase.item:undefined} previewProducts={[purchase.item]} items={[{productId:purchase.item.productId||purchase.item.listingId,quantity:purchase.quantity,expectedProductVersion:purchase.item.version}]} onResource={(resource)=>navigate(`/orders?order=${encodeURIComponent(resourceId(resource.value))}`)}/></Modal>}
  </>;
}

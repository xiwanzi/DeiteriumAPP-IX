import React, { useEffect, useRef, useState } from "react";
import { RefreshCw, Search, Package, Plus } from "lucide-react";
import { PageHead, Button, Empty, Badge, Modal } from "./components.jsx";
import { credit } from "./format.js";
import { id } from "./format.js";
import { ListingForm } from "./CatalogManagement.jsx";
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
  const config = collections[path] || collections["/"],
    [items, setItems] = useState([]), [cursor, setCursor] = useState(null), [busy, setBusy] = useState(true),
    [error, setError] = useState(""), [query, setQuery] = useState(""), [selected, setSelected] = useState(null), [mine, setMine] = useState(false), [editor, setEditor] = useState(null), [unlist, setUnlist] = useState(null), [purchase, setPurchase] = useState(null), [quantity, setQuantity] = useState(1), [bagMessage, setBagMessage] = useState("");
  const actionKeys = useRef({});
  const generation = useRef(0);
  const load = async (more = false) => {
    const current = ++generation.current;
    setBusy(true); setError("");
    try {
      const endpoint = path === "/market" && mine ? "/market/me/listings" : config.endpoint;
      const r = await client.request(`/api/v1${endpoint}?limit=30${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`);
      if (current !== generation.current) return;
      if (!Array.isArray(r.data)) throw new Error("服务返回的列表格式不正确，请稍后重试。");
      setItems((old) => more ? [...new Map([...old, ...r.data].map((item) => [item[config.key], item])).values()] : r.data);
      setCursor(r.page?.nextCursor);
    } catch (e) { if (current === generation.current) setError(serviceError(e)); }
    finally { if (current === generation.current) setBusy(false); }
  };
  useEffect(() => { setItems([]); setQuery(""); setSelected(null); load(); return () => { generation.current++; }; }, [path, mine]);
  const markNotification = async (item) => {
    if (item.readAt) return; const scope = `notification:${item.notificationId}`; actionKeys.current[scope] ||= id();
    try { const r = await client.request("/api/v1/notifications/read", {method:"POST",body:{clientRequestId:actionKeys.current[scope],notificationIds:[item.notificationId]}}); if (typeof r.data.readCount !== "number") throw new Error("通知状态尚未确认。"); const changed={...item,readAt:r.serverTime || new Date().toISOString()};setItems((old)=>old.map((x)=>x.notificationId===item.notificationId?changed:x));setSelected(changed); }
    catch(e){setError(e.message);}
  };
  const select = (item) => { setSelected(item); setQuantity(1); setBagMessage(""); if (path === "/notifications") markNotification(item); };
  const addToBag = async () => { setBusy(true);setError("");setBagMessage("");try{const r=await client.request("/api/v1/store/cart");const old=r.data.items.find((item)=>item.productId===selected.productId);await client.request(`/api/v1/store/cart/items/${encodeURIComponent(selected.productId)}`,{method:"PUT",body:{clientRequestId:id(),expectedVersion:r.data.version,quantity:(old?.quantity||0)+quantity}});setBagMessage("已加入购物袋");}catch(e){setError(e.message);}finally{setBusy(false);} };
  const unlistItem = async () => { setBusy(true);setError("");const scope=`unlist:${unlist.listingId}:${unlist.version}`;actionKeys.current[scope] ||= id();try{await client.request(`/api/v1/market/listings/${encodeURIComponent(unlist.listingId)}/unlist`,{method:"POST",body:{clientRequestId:actionKeys.current[scope],expectedVersion:unlist.version}});setUnlist(null);setSelected(null);await load();}catch(e){setError(e.message);}finally{setBusy(false);} };
  const content = selected?.content || selected;
  const validQuantity = Number.isInteger(quantity) && quantity >= 1 && quantity <= Math.min(selected?.availableStock ?? selected?.stock ?? 999, selected?.content?.limitPerOrder ?? 999);
  const visible = items.filter((item) => `${item.title || item.content?.title || ""} ${item.summary || item.subtitle || item.content?.description || ""}`.toLowerCase().includes(query.toLowerCase()));
  return <>
    <PageHead eyebrow="DEUTERIUM COMMUNITY" title={config.title} subtitle={config.subtitle}><Button secondary disabled={busy} onClick={() => load()}><RefreshCw size={16} />刷新</Button>{path==="/market"&&<><Button secondary onClick={()=>setMine(!mine)}>{mine?"发现好物":"我的发布"}</Button><Button onClick={()=>setEditor({})}><Plus size={16}/>发布商品</Button></>}</PageHead>
    <label className="search-input" style={{ maxWidth: 480, marginBottom: 24 }}><Search size={17} /><input aria-label={`搜索${config.title}`} placeholder={`搜索已加载的${config.title}`} value={query} onChange={(e) => setQuery(e.target.value)} /></label>
    {error && <div className="notice-box" role="alert">{error}</div>}
    <div className="foundation-grid">{visible.map((item) => {
      const c = item.content || item, image = item.images?.[0] || item.photos?.[0] || item.cover;
      return <button className="foundation-card" style={{ textAlign: "left", color: "inherit", cursor: "pointer" }} key={item[config.key]} onClick={() => select(item)}>
        {image?.url ? <img src={image.url} alt={image.altText || c.title} style={{ width: "100%", aspectRatio: "16/10", objectFit: "cover", borderRadius: 12 }} /> : <Package size={24} />}
        {item.pinned && <Badge>置顶</Badge>}{path==="/market"&&mine&&<Badge tone={item.active?"sage":"neutral"}>{item.active?"在售":"已下架"}</Badge>}{path==="/notifications"&&!item.readAt&&<Badge>未读</Badge>}<h3 style={{ marginTop: 16 }}>{c.title}</h3><p>{c.subtitle || c.description || item.summary || item.body}</p>
        {(c.price !== undefined || c.reward !== undefined) && <strong className="price">{credit(c.price ?? c.reward)}<small>信用点</small></strong>}
      </button>;
    })}</div>
    {!busy && !error && !visible.length && <Empty title={query ? "没有匹配的结果" : config.empty} text={query ? "试试其他关键词。" : "新的内容发布后会出现在这里。"} />}
    {busy && <p className="muted" role="status">正在读取…</p>}
    {cursor && <Button secondary disabled={busy} onClick={() => load(true)}>加载更多</Button>}
    {selected && <Modal title={content.title} close={() => setSelected(null)}>
      {(selected.images || selected.photos || (selected.cover ? [selected.cover] : [])).map((asset) => <img key={asset.assetId} src={asset.url} alt={asset.altText || content.title} style={{ width: "100%", borderRadius: 12, marginBottom: 12 }} />)}
      <p className="description" style={{ whiteSpace: "pre-wrap" }}>{content.description || content.subtitle || selected.summary || selected.body}</p>
      <ContentBlocks blocks={content.contentBlocks || content.detailBlocks || []} media={selected.media || selected.images || []} />
      {content.location && <p>地点：{content.location}</p>}
      {selected.contactQq && <p>联系 QQ：{selected.contactQq}</p>}
      {(content.price !== undefined || content.reward !== undefined) && <strong className="price">{credit(content.price ?? content.reward)}<small>信用点</small></strong>}
      {selected.publishedAt && <p className="muted">{new Date(selected.publishedAt).toLocaleString("zh-CN")}</p>}
      {path==="/market"&&selected.seller?.playerRef===user?.playerRef&&<div className="button-row"><Button onClick={()=>{setEditor(selected);setSelected(null);}}>{selected.active?"编辑商品":"重新上架"}</Button>{selected.active&&<Button secondary onClick={()=>{setUnlist(selected);setSelected(null);}}>下架</Button>}</div>}
      {(path==="/"||path==="/market"&&selected.seller?.playerRef!==user?.playerRef)&&<><div className="field"><label htmlFor="purchase-quantity">购买数量</label><input id="purchase-quantity" type="number" min={1} max={Math.min(selected.availableStock??selected.stock??999,selected.content?.limitPerOrder??999)} value={quantity} onChange={(e)=>setQuantity(Number(e.target.value))}/></div>{bagMessage&&<p role="status">{bagMessage}</p>}{error&&<p className="auth-error" role="alert">{error}</p>}<div className="button-row">{path==="/"&&<Button secondary disabled={busy||!validQuantity} onClick={addToBag}>加入购物袋</Button>}<Button disabled={busy||!validQuantity} onClick={()=>{setPurchase({item:selected,quantity});setSelected(null);}}>立即购买</Button></div></>}
      {path==="/notifications"&&<><div className="button-row">{!selected.readAt&&<Button secondary onClick={()=>markNotification(selected)}>标记已读</Button>}{selected.target&&onNotificationTarget&&<Button onClick={()=>onNotificationTarget(selected.target)}>查看相关内容</Button>}</div>{error&&<p className="auth-error" role="alert">{error}</p>}</>}
    </Modal>}
    {editor&&<Modal title={editor.listingId?"编辑玩家商品":"发布玩家商品"} close={()=>setEditor(null)} wide guardClose dismissOnBackdrop={false}><ListingForm client={client} user={user} initial={editor} onSaved={async()=>{setEditor(null);await load();}}/></Modal>}
    {unlist&&<Modal title="确认下架" close={()=>setUnlist(null)}><p>下架“{unlist.title}”后，其他玩家将无法购买。</p>{error&&<p className="auth-error" role="alert">{error}</p>}<Button disabled={busy} onClick={unlistItem}>确认下架</Button></Modal>}
    {purchase&&<Modal title="确认购买" close={()=>setPurchase(null)}><PurchaseFlow client={client} user={user} channel={path==="/market"?"PLAYER_MARKET":"OFFICIAL_STORE"} listing={path==="/market"?purchase.item:undefined} items={[{productId:purchase.item.productId||purchase.item.listingId,quantity:purchase.quantity,expectedProductVersion:purchase.item.version}]} onResource={(resource)=>navigate(`/orders?order=${encodeURIComponent(resourceId(resource.value))}`)}/></Modal>}
  </>;
}

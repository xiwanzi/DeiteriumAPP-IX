import React, { useEffect, useRef, useState } from "react";
import { ShoppingBag, Trash2 } from "lucide-react";
import { Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { credit, id } from "./format.js";
import PurchaseFlow from "./PurchaseFlow.jsx";
import { resourceId } from "./business.js";

export default function ConnectedCart({ client, user, navigate }) {
  const [cart, setCart] = useState(null), [products, setProducts] = useState({}), [error, setError] = useState(""), [busy, setBusy] = useState(false), [checkout, setCheckout] = useState(false);
  const requests = useRef({}), mounted = useRef(true);
  const load = async () => {
    setBusy(true); setError("");
    try {
      const result = await client.request("/api/v1/store/cart"); if (!mounted.current) return; setCart(result.data);
      const values = await Promise.allSettled(result.data.items.map((item) => client.request(`/api/v1/store/products/${encodeURIComponent(item.productId)}`)));
      if (mounted.current) setProducts(Object.fromEntries(result.data.items.map((item, index) => [item.productId, values[index].status === "fulfilled" ? values[index].value.data : null])));
    } catch (e) { if (mounted.current) setError(e.message); } finally { if (mounted.current) setBusy(false); }
  };
  useEffect(() => { mounted.current = true; load(); return () => { mounted.current = false; }; }, [user.userId]);
  const update = async (item, quantity, remove = false) => {
    if (busy) return; setBusy(true); setError(""); const scope = `${item.productId}:${cart.version}:${remove ? "remove" : quantity}`; requests.current[scope] ||= id();
    try { await client.request(`/api/v1/store/cart/items/${encodeURIComponent(item.productId)}${remove ? "/remove" : ""}`, { method: remove ? "POST" : "PUT", body: { clientRequestId: requests.current[scope], expectedVersion: cart.version, ...(!remove ? { quantity } : {}) } }); await load(); }
    catch (e) { setError(e.message); } finally { setBusy(false); }
  };
  const rows = cart?.items || [];
  return <><PageHead eyebrow="SHOPPING BAG" title="购物袋" subtitle="商品价格和库存将在结算时再次确认。"><Button secondary onClick={load} disabled={busy}>刷新</Button></PageHead>
    {error && <p className="auth-error" role="alert">{error}</p>}
    <div className="cart-product-list">{rows.map((item) => { const product = products[item.productId]; return <div className="cart-product-row" key={item.productId}>
      {product?.images?.[0]?.url && <img className="cart-product-image" src={product.images[0].url} alt="" />}<div className="cart-product-copy">
      <h3>{product?.content?.title || "商品暂不可用"}</h3>{product && <p className="price">{credit(product.content.price)}<small>信用点 / 件</small></p>}
      </div><Field label="数量"><select value={item.quantity} disabled={busy || !product} onChange={(e) => update(item, Number(e.target.value))}>{Array.from({ length: Math.max(item.quantity, Math.min(999, product?.content?.limitPerOrder || 1, product?.availableStock ?? 999)) }, (_, index) => <option key={index + 1} value={index + 1}>{index + 1}</option>)}</select></Field>
      <Button secondary disabled={busy} onClick={() => update(item, 0, true)}><Trash2 size={15} />移出购物袋</Button>
    </div>; })}</div>
    {!busy && !rows.length && <Empty title="购物袋还是空的" text="挑选喜欢的商品后，可以加入这里。"><Button onClick={() => navigate("/")}>前往商城</Button></Empty>}
    {rows.length > 0 && <div className="button-row"><Button disabled={busy || rows.some((item) => !products[item.productId])} onClick={() => setCheckout(true)}><ShoppingBag size={16} />去结算</Button></div>}
    {checkout && <Modal title="商城结算" close={() => setCheckout(false)} className="checkout-modal"><PurchaseFlow client={client} user={user} channel="OFFICIAL_STORE" previewProducts={rows.map((item) => products[item.productId])} items={rows.map((item) => ({ productId: item.productId, quantity: item.quantity, expectedProductVersion: products[item.productId].version }))} onResource={(resource) => navigate(`/orders?order=${encodeURIComponent(resourceId(resource.value))}`)} /></Modal>}
  </>;
}

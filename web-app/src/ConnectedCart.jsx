import React, { useContext, useEffect, useRef, useState } from "react";
import { Minus, Plus, RefreshCw, ShoppingBag, Ticket, Trash2 } from "lucide-react";
import { Button, Empty, Modal } from "./components.jsx";
import { credit, id } from "./format.js";
import { productInventory } from "./productInventory.js";
import { ShoppingCartContext } from "./ShoppingCartContext.jsx";
import ProductArtwork from "./ProductArtwork.jsx";
import PurchaseFlow from "./PurchaseFlow.jsx";
import { resourceId } from "./business.js";
import CouponWallet from "./CouponWallet.jsx";
import "./shopping-cart.css";

function subtotal(items, products) {
  let total = 0n;
  for (const item of items) {
    const product = products[item.productId];
    const price = String(product?.effectivePrice ?? product?.content?.price ?? "");
    if (!/^\d+(\.\d{1,2})?$/.test(price)) return null;
    const [whole, fraction = ""] = price.split(".");
    total += (BigInt(whole) * 100n + BigInt(fraction.padEnd(2, "0"))) * BigInt(item.quantity);
  }
  return `${total / 100n}.${String(total % 100n).padStart(2, "0")}`;
}

function QuantityControl({ title, quantity, max, disabled, onChange }) {
  const [draft, setDraft] = useState(null), [hint, setHint] = useState("");
  useEffect(() => { setDraft(null); setHint(""); }, [quantity, max]);
  const commit = value => {
    setDraft(null);
    const next = Number(value);
    if (!Number.isInteger(next) || next < 1 || next > max) {
      setHint(`请输入 1 至 ${max} 的整数`);
      return;
    }
    setHint("");
    if (next !== quantity) onChange(next);
  };
  const typed = draft === null || !/^\d+$/.test(draft) ? quantity : Number(draft);
  return <div className="bag-quantity">
    <div className="cart-stepper" role="group" aria-label={`调整${title}数量`}
      onBlur={event => { if (draft !== null && !event.currentTarget.contains(event.relatedTarget)) commit(draft); }}>
      <button type="button" aria-label={`减少${title}数量`} disabled={disabled || typed <= 1} onClick={() => commit(Math.min(max, typed - 1))}><Minus size={15}/></button>
      <input type="number" inputMode="numeric" min="1" max={max} step="1" aria-label={`${title}数量`} disabled={disabled} value={draft ?? quantity}
        onChange={event => { setDraft(event.target.value); setHint(""); }}
        onKeyDown={event => {
          if (event.key === "Enter") { event.preventDefault(); event.currentTarget.blur(); }
          if (event.key === "Escape") { setDraft(null); setHint(""); }
        }} />
      <button type="button" aria-label={`增加${title}数量`} disabled={disabled || typed >= max} onClick={() => commit(Math.max(1, typed + 1))}><Plus size={15}/></button>
    </div>
    {hint && <small className="bag-row-note" role="status">{hint}</small>}
  </div>;
}

export default function ConnectedCart({ client, user, navigate }) {
  const { cart, count, readCart, applyCart } = useContext(ShoppingCartContext);
  const [details, setDetails] = useState(null), [error, setError] = useState(""), [reading, setReading] = useState(true);
  const [writing, setWriting] = useState(false), [refresh, setRefresh] = useState(0);
  const [checkout, setCheckout] = useState(null), [coupons, setCoupons] = useState(false);
  const request = useRef(null), locked = useRef(false);
  const rows = cart?.items ?? [];
  const productKey = JSON.stringify(rows.map(item => [item.productId, item.productVersion]));
  const products = details?.products ?? {};
  const detailsReady = details?.key === productKey && details.refresh === refresh;
  const busy = reading || writing;
  const total = detailsReady ? subtotal(rows, products) : null;
  const canCheckout = !busy && !!rows.length && total !== null && rows.every(item =>
    products[item.productId] && item.quantity <= productInventory(products[item.productId]).max);

  useEffect(() => {
    let active = true;
    setReading(true); setError("");
    readCart(refresh > 0)
      .catch(e => { if (active) setError(e.message); })
      .finally(() => { if (active) setReading(false); });
    return () => { active = false; };
  }, [readCart, refresh]);

  useEffect(() => {
    const controller = new AbortController();
    const refs = JSON.parse(productKey);
    Promise.allSettled(refs.map(([productId]) => client.request(
      `/api/v1/store/products/${encodeURIComponent(productId)}`, { signal: controller.signal },
    ))).then(results => {
      if (controller.signal.aborted) return;
      const loaded = results.map((result, index) => [refs[index][0], result.status === "fulfilled" ? result.value.data : null]);
      setDetails({ key: productKey, refresh, products: Object.fromEntries(loaded) });
    });
    return () => controller.abort();
  }, [client, productKey, refresh]);

  const update = async (item, quantity, remove = false) => {
    if (busy || locked.current) return;
    locked.current = true; setWriting(true); setError("");
    const scope = `${item.productId}:${cart.version}:${remove ? "remove" : quantity}`;
    if (request.current?.scope !== scope) request.current = { scope, key: id() };
    try {
      const response = await client.request(`/api/v1/store/cart/items/${encodeURIComponent(item.productId)}${remove ? "/remove" : ""}`, {
        method: remove ? "POST" : "PUT", expectedPlayerRef: user.playerRef, body: { clientRequestId: request.current.key, expectedVersion: cart.version, ...(!remove ? { quantity } : {}) },
      });
      applyCart(response.data);
      request.current = null;
    } catch (e) {
      setError(e.message);
      if (e.status === 409) await readCart(true).catch(() => {});
    } finally { locked.current = false; setWriting(false); }
  };
  const beginCheckout = () => {
    if (!canCheckout || locked.current) return;
    setCheckout({
      products: rows.map(item => products[item.productId]),
      items: rows.map(item => ({productId:item.productId, quantity:item.quantity, expectedProductVersion:products[item.productId].version})),
    });
  };
  return <section className="shopping-bag" aria-label="购物袋">
    <header className="bag-heading"><div><h1>购物袋</h1>
      {!!rows.length && <div className="bag-meta"><p aria-live="polite" aria-atomic="true">{rows.length} 种商品 <span>·</span> 共 {count} 件</p>
        <button type="button" onClick={() => navigate("/")}>继续选购</button></div>}</div>
      <div className="bag-tools"><Button secondary onClick={() => setCoupons(true)}><Ticket size={17}/>优惠券</Button>
        <button type="button" className="icon-button" aria-label="刷新购物袋" title="刷新购物袋" disabled={busy} onClick={() => setRefresh(value => value + 1)}><RefreshCw size={17}/></button>
      </div>
    </header>
    {error && <p className="bag-error" role="alert">{error}</p>}
    {!cart && reading && <p role="status" className="muted">正在读取购物袋…</p>}
    {cart && !rows.length && <div className="bag-empty"><Empty title="购物袋还是空的" text="喜欢的商品可以先加入购物袋，一起结算。"><Button onClick={() => navigate("/")}>前往商城</Button></Empty></div>}
    {!!rows.length && <div className="bag-layout">
      <ul className="bag-items" aria-label="袋中商品">{rows.map(item => {
        const product = products[item.productId], title = product?.content?.title || (detailsReady ? "商品暂不可用" : "正在读取商品…");
        const max = product ? productInventory(product).max : 0;
        return <li className="bag-row" key={item.productId}>
          <ProductArtwork className="bag-art" imageClassName="cart-product-image" image={product?.images?.[0]} />
          <div className="bag-item-copy"><h2>{title}</h2>
            {product && <p><strong>{credit(product.effectivePrice ?? product.content.price)}</strong> 信用点 / 件{product.content.discountRate != null && product.content.discountRate < 10000 && <del>{credit(product.content.price)}</del>}</p>}
            {detailsReady && (!product || item.quantity > max) && <small className="bag-row-note">{!product ? "可以刷新商品信息或移除这件商品。" : max === 0 ? "暂时售罄" : `当前最多可购 ${max} 件`}</small>}
          </div>
          <QuantityControl title={title} quantity={item.quantity} max={max} disabled={busy || !detailsReady || max < 1} onChange={quantity => update(item, quantity)} />
          <button type="button" className="bag-remove" aria-label={`将${title}移出购物袋`} title="移出购物袋" disabled={busy || !detailsReady} onClick={() => update(item, 0, true)}><Trash2 size={16}/><span>移除</span></button>
        </li>;
      })}</ul>
      <aside className="bag-summary" aria-label="结算明细">
        <div><h2>商品小计</h2><p className="bag-amount"><strong>{credit(total)}</strong> 信用点</p>
          <p className="bag-summary-note">可用优惠将在结算时抵扣。</p></div>
        <Button disabled={!canCheckout} onClick={beginCheckout}><ShoppingBag size={17}/>去结算</Button>
      </aside>
    </div>}
    {checkout && <Modal title="商城结算" close={() => { setCheckout(null); readCart(true).catch(e => setError(e.message)); }} className="checkout-modal">
      <PurchaseFlow client={client} user={user} channel="OFFICIAL_STORE" source="CART" onCartPaid={() => { readCart(true).catch(e => setError(e.message)); }} previewProducts={checkout.products} items={checkout.items} onResource={resource => navigate(`/orders?order=${encodeURIComponent(resourceId(resource.value))}`)} />
    </Modal>}
    {coupons && <Modal title="优惠券" close={() => setCoupons(false)} wide><CouponWallet client={client} /></Modal>}
  </section>;
}

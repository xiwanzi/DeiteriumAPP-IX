import React, { createContext, useCallback, useEffect, useMemo, useRef, useState } from "react";

export const ShoppingCartContext = createContext(null);

export function ShoppingCartProvider({ client, playerRef, children }) {
  const [cart, setCart] = useState(null);
  const latest = useRef(null), pending = useRef(null);

  const applyCart = useCallback(snapshot => {
    if (!Number.isSafeInteger(snapshot?.version) || snapshot.version < 1 || !Array.isArray(snapshot.items) ||
        !snapshot.items.every(item => item.productId && Number.isSafeInteger(item.quantity) && item.quantity > 0)) {
      throw new Error("购物袋数据不完整，请刷新重试。");
    }
    if (!latest.current || snapshot.version > latest.current.version) {
      latest.current = snapshot;
      setCart(snapshot);
    }
    return latest.current;
  }, []);

  const readCart = useCallback((fresh = false) => {
    if (pending.current && !fresh) return pending.current;
    const request = client.request("/api/v1/store/cart", { expectedPlayerRef: playerRef })
      .then(response => applyCart(response.data))
      .finally(() => { if (pending.current === request) pending.current = null; });
    pending.current = request;
    return request;
  }, [client, playerRef, applyCart]);

  useEffect(() => {
    const sync = () => { if (!document.hidden) readCart().catch(() => {}); };
    sync();
    document.addEventListener("visibilitychange", sync);
    window.addEventListener("focus", sync);
    window.addEventListener("online", sync);
    return () => {
      document.removeEventListener("visibilitychange", sync);
      window.removeEventListener("focus", sync);
      window.removeEventListener("online", sync);
    };
  }, [readCart]);

  const value = useMemo(() => ({
    cart, applyCart, readCart,
    count: cart ? cart.items.reduce((total, item) => total + item.quantity, 0) : null,
  }), [cart, applyCart, readCart]);
  return <ShoppingCartContext.Provider value={value}>{children}</ShoppingCartContext.Provider>;
}

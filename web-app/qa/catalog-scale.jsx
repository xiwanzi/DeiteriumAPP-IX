// Isolated browser fixture; Vite's production entry does not import this file.
import React from "react";
import { createRoot } from "react-dom/client";
import RemoteCollection from "../src/RemoteCollection.jsx";
import "../src/styles.css";
import "../src/v2.css";
import "../src/desktop-admin.css";
import "../src/commerce-v209.css";

const texture = "data:image/svg+xml," + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="300" height="188"><rect width="300" height="188" fill="#e8edf2"/><path d="M110 70h80v65h-80zM100 60h100v20H100z" fill="#839896"/><path d="M144 60h12v75h-12z" fill="#c5d6cf"/></svg>');
const products = Array.from({ length: 3085 }, (_, i) => ({ productId: `fixture_${i}`, storeId: "fixture_store", version: 1, availableStock: null, effectivePrice: "100.00", content: { title: `家居道具 ${String(i).padStart(4, "0")}`, subtitle: "隔离测试商品", description: "仅用于验证分页、库存和搜索。", price: "100.00", stock: 0, inventoryPolicy: "UNLIMITED", limitPerOrder: 64, galleryAssetIds: [`image_${i}`] }, images: [{ assetId: `image_${i}`, url: texture }] }));
const calls = [];
window.catalogQA = { calls, failNextPage: false };
const client = {
  onAccountDeletions: () => () => {},
  request: async (path, options) => {
    calls.push({ path, method: options?.method || "GET" });
    const url = new URL(path, location.origin);
    await new Promise((resolve) => setTimeout(resolve, url.searchParams.get("q") === "0001" ? 900 : 100));
    if (url.pathname === "/api/v1/store/products") {
      const start = Number(url.searchParams.get("cursor") || 0), limit = Number(url.searchParams.get("limit") || 40);
      if (start && window.catalogQA.failNextPage) { window.catalogQA.failNextPage = false; throw new Error("隔离测试：续页连接中断"); }
      const query = url.searchParams.get("q") || "", all = products.filter((p) => p.content.title.includes(query));
      const data = all.slice(start, start + limit), next = start + data.length;
      return { data, page: { nextCursor: next < all.length ? String(next) : null, hasMore: next < all.length } };
    }
    if (url.pathname === "/api/v1/store/cart") return { data: { version: 1, items: [] } };
    if (url.pathname.startsWith("/api/v1/store/cart/items/")) return { data: { version: 2, items: [] } };
    throw new Error(`Unexpected fixture request: ${url.pathname}`);
  },
};
createRoot(document.getElementById("root")).render(<main style={{ maxWidth: 1160, margin: "32px auto", padding: "0 24px" }}><RemoteCollection client={client} path="/" user={{ playerRef: "fixture" }} /></main>);

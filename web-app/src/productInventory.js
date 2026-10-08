// Null is the public API's unlimited-stock value. A draft's stock=0 is not live stock.
export function productInventory(product, official = true) {
  const content = product.content || product;
  const unlimited = official && content.inventoryPolicy === "UNLIMITED";
  const stock = unlimited ? Infinity : (product.availableStock ?? product.stock ?? content.stock ?? 0);
  const max = Math.max(0, Math.min(stock, content.limitPerOrder ?? 999, 999));
  return { unlimited, stock, max };
}

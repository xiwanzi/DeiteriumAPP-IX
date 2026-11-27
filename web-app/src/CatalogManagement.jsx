import React, { useEffect, useRef, useState } from "react";
import {
  Plus,
  RefreshCw,
  Search,
  Package,
  ChevronLeft,
  ChevronRight,
  Settings2,
  FolderTree,
  Mail,
  ReceiptText,
  X,
  Copy,
  Trash2,
  EyeOff,
  Upload,
} from "lucide-react";
import { Badge, Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { credit, id } from "./format.js";
import ProductForm from "./ProductEditor.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";
import DeliveryTemplateManagement from "./DeliveryTemplateManagement.jsx";
import BusinessPages from "./BusinessPages.jsx";
import { MediaPicker } from "./MediaPicker.jsx";
import StoreAvatarPicker from "./StoreAvatarPicker.jsx";
import { salePrice } from "./promotions.js";
import { CategoryBrandWorkspace, ShopBannerSettings } from "./ShopNavigationSettings.jsx";
import { ActionMenu, CopyButton } from "./AdminPrimitives.jsx";
export { MediaPicker } from "./MediaPicker.jsx";

export const marketCategories = [
  ["MATERIALS", "建材"],
  ["EQUIPMENT", "装备"],
  ["SUPPLIES", "补给"],
  ["DECORATION", "装饰"],
  ["CONSTRUCTION", "建筑服务"],
  ["OTHER", "其他"],
];
const moneyInput = (value) => {
  const text = String(value).trim();
  if (
    text.length > 20 ||
    !/^(0|[1-9]\d*)(\.\d{1,2})?$/.test(text) ||
    /^0(?:\.0{1,2})?$/.test(text)
  )
    throw new Error("请输入大于 0 的金额，最多两位小数。");
  return text;
};
function useStableMutation() {
  const ref = useRef(null);
  return (body) => {
    const fingerprint = JSON.stringify(body);
    if (ref.current?.fingerprint !== fingerprint)
      ref.current = { fingerprint, body: { clientRequestId: id(), ...body } };
    return ref.current.body;
  };
}

export function ListingForm({ client, user, initial = {}, onSaved }) {
  const [value, setValue] = useState({
    title: initial.title || "",
    subtitle: initial.subtitle || "",
    description: initial.description || "",
    categoryCode: initial.categoryCode || "MATERIALS",
    price: initial.price || "",
    stock: initial.stock || 1,
    contactQq: initial.contactQq || user.qq,
    pickupLocation: initial.pickupLocation || "",
    workHours: initial.workHours || 1,
  });
  const [images, setImages] = useState(initial.photos || []);
  const [delivery, setDelivery] = useState(initial.deliveryMethods?.join(",") || "PICKUP");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useUnsavedChanges({ value, images: images.map((image) => image.assetId), delivery }, busy);
  const stable = useStableMutation();
  const construction = value.categoryCode === "CONSTRUCTION";
  const field = (name, label, props = {}) => (
    <Field
      key={name}
      label={label}
      value={value[name]}
      onChange={(e) =>
        setValue((old) => ({
          ...old,
          [name]: props.type === "number" ? Number(e.target.value) : e.target.value,
        }))
      }
      {...props}
    />
  );
  const save = async (event) => {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      if (!images.length) throw new Error("请至少上传一张商品图片。");
      const content = {
        ...value,
        price: moneyInput(value.price),
        stock: Number(value.stock),
        workHours: Number(value.workHours),
        photoAssetIds: images.map((asset) => asset.assetId),
        deliveryMethods: construction ? ["WORKSITE"] : delivery.split(","),
      };
      const body = stable({
        ...(initial.listingId ? { expectedVersion: initial.version } : {}),
        content,
      });
      const path = initial.listingId
        ? `/api/v1/market/listings/${encodeURIComponent(initial.listingId)}${initial.active === false ? "/republish" : ""}`
        : "/api/v1/market/listings";
      await client.request(path, {
        method: !initial.listingId || initial.active === false ? "POST" : "PUT",
        body,
      });
      await onSaved();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={save}>
      <fieldset disabled={busy} style={{ border: 0, margin: 0, padding: 0, minWidth: 0 }}>
        {field("title", "商品标题", { required: true, maxLength: 80 })}
        {field("subtitle", "一句话简介", { required: true, maxLength: 200 })}
        <Field label="详细介绍">
          <textarea
            required
            rows={5}
            maxLength={10000}
            value={value.description}
            onChange={(e) => setValue({ ...value, description: e.target.value })}
          />
        </Field>
        <Field label="分类">
          <select
            value={value.categoryCode}
            onChange={(e) => setValue({ ...value, categoryCode: e.target.value })}
          >
            {marketCategories.map(([code, label]) => (
              <option key={code} value={code}>
                {label}
              </option>
            ))}
          </select>
        </Field>
        {field("price", "售价（信用点）", { required: true, inputMode: "decimal", maxLength: 20 })}
        {field("stock", "库存", { required: true, type: "number", min: 1, max: 999 })}
        <MediaPicker
          client={client}
          images={images}
          onChange={setImages}
          purpose="MARKET_PHOTO"
          businessType="MARKET_LISTING"
          businessRef={initial.listingId || ""}
          max={5}
          onBusy={setBusy}
        />
        {field("contactQq", "联系 QQ", {
          required: true,
          inputMode: "numeric",
          pattern: "[0-9]{5,12}",
          maxLength: 12,
        })}
        {!construction && (
          <Field label="交付方式">
            <select value={delivery} onChange={(e) => setDelivery(e.target.value)}>
              <option value="PICKUP">约定自取</option>
              <option value="DOOR">送货上门</option>
              <option value="DOOR,PICKUP">两种均可</option>
            </select>
          </Field>
        )}
        {(construction || delivery.includes("PICKUP")) &&
          field("pickupLocation", construction ? "工程地点" : "自取地点", {
            required: true,
            maxLength: 120,
          })}
        {construction &&
          field("workHours", "总工期（小时，包含验收预留）", {
            required: true,
            type: "number",
            min: 1,
            max: 8760,
          })}
      </fieldset>
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      <Button type="submit" disabled={busy}>
        {busy
          ? "正在处理…"
          : initial.listingId
            ? initial.active === false
              ? "重新上架"
              : "保存商品"
            : "发布商品"}
      </Button>
    </form>
  );
}

const WORKSPACE_TABS = [
  { id: "products", label: "商品目录", icon: Package },
  { id: "categories", label: "分类与品牌", icon: FolderTree },
  { id: "templates", label: "交付模板", icon: Mail },
  { id: "orders", label: "商店订单", icon: ReceiptText },
  { id: "settings", label: "店铺与首页", icon: Settings2 },
];

export default function CatalogManagement({ client, user }) {
  const [moreTemplates, setMoreTemplates] = useState(false);
  const [stores, setStores] = useState([]);
  const [selected, setSelected] = useState("");
  const [workspaceTab, setWorkspaceTab] = useState("products");
  const [products, setProducts] = useState([]);
  const [brands, setBrands] = useState([]);
  const [categories, setCategories] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [loadingAll, setLoadingAll] = useState(false);
  const [modal, setModal] = useState(null);
  const [reason, setReason] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("ALL");
  const [categoryFilter, setCategoryFilter] = useState("ALL");
  const [page, setPage] = useState(1);
  const [cursor, setCursor] = useState(null);

  const all = (user.permissions || []).includes("platform.admin");
  const current = stores.find((store) => store.storeId === selected);
  const actionKeys = useRef({});
  const generation = useRef(0);

  const loadStores = async () => {
    setBusy(true);
    setError("");
    try {
      const r = await client.request("/api/v1/merchant/me");
      const values = await Promise.all(
        r.data.storeIds.map((storeId) =>
          client.request(`/api/v1/merchant/stores/${encodeURIComponent(storeId)}`),
        ),
      );
      const next = values.map((item) => item.data);
      setStores(next);
      setSelected((old) =>
        next.some((store) => store.storeId === old) ? old : next[0]?.storeId || "",
      );
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const loadStoreContent = async (more = false) => {
    const requestGeneration = ++generation.current;
    if (!selected) return;
    setBusy(true);
    setError("");
    try {
      const [p, b, c, t] = await Promise.all(
        ["products", "brands", "categories", "delivery-templates"].map((kind) =>
          client.request(
            `/api/v1/merchant/stores/${encodeURIComponent(selected)}/${kind}?limit=100${kind === "products" && more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
          ),
        ),
      );
      if (requestGeneration !== generation.current) return;
      setProducts((old) =>
        more
          ? [...new Map([...old, ...p.data].map((product) => [product.productId, product])).values()]
          : p.data,
      );
      setCursor(p.page?.nextCursor);
      setBrands(b.data);
      setCategories(c.data);
      setTemplates(t.data);
      setMoreTemplates(Boolean(t.page?.hasMore));
    } catch (e) {
      if (requestGeneration === generation.current) setError(e.message);
    } finally {
      if (requestGeneration === generation.current) setBusy(false);
    }
  };

  // Batch load up to 10 pages (1000 items) at once so searching large catalogs doesn't require repeated manual clicks
  const loadMoreBatch = async () => {
    if (!selected || !cursor || busy || loadingAll) return;
    const requestGeneration = ++generation.current;
    setLoadingAll(true);
    setError("");
    let nextCursor = cursor;
    let collected = [];
    try {
      for (let i = 0; i < 10 && nextCursor; i++) {
        const p = await client.request(
          `/api/v1/merchant/stores/${encodeURIComponent(selected)}/products?limit=100&cursor=${encodeURIComponent(nextCursor)}`,
        );
        if (requestGeneration !== generation.current) return;
        collected = [...collected, ...p.data];
        nextCursor = p.page?.nextCursor || null;
      }
      setProducts((old) => [
        ...new Map([...old, ...collected].map((product) => [product.productId, product])).values(),
      ]);
      setCursor(nextCursor);
    } catch (e) {
      if (requestGeneration === generation.current) setError(e.message);
    } finally {
      if (requestGeneration === generation.current) setLoadingAll(false);
    }
  };

  useEffect(() => {
    loadStores();
  }, []);
  useEffect(() => {
    setProducts([]);
    setBrands([]);
    setCategories([]);
    setTemplates([]);
    setCursor(null);
    setPage(1);
    loadStoreContent();
    return () => {
      generation.current++;
    };
  }, [selected]);
  useEffect(() => {
    setPage(1);
  }, [query, status, categoryFilter]);

  const action = async (product, kind, explanation = "") => {
    if (busy) return;
    setBusy(true);
    setError("");
    const scope = `${product.productId}:${product.version}:${kind}:${explanation}`;
    actionKeys.current[scope] ||= id();
    try {
      const response = await client.request(
        `/api/v1/merchant/products/${encodeURIComponent(product.productId)}/${kind}`,
        {
          method: "POST",
          body: {
            clientRequestId: actionKeys.current[scope],
            expectedVersion: product.version,
            ...(kind !== "publish" && kind !== "delete" ? { reason: explanation } : {}),
          },
        },
      );
      if (kind === "delete" && !response.data?.deleted)
        throw new Error("删除结果尚未确认，请重试核对。");
      setModal(null);
      await loadStoreContent();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const duplicateProduct = (product) => {
    const clonedDraft = {
      ...(product.draft || {}),
      title: `${product.draft?.title || "商品"} · 副本`.slice(0, 80),
    };
    setModal({
      type: "product",
      item: {
        draft: clonedDraft,
        draftImages: product.draftImages || product.published?.images || [],
      },
    });
  };

  const states = { ACTIVE: "已上架", DRAFT: "草稿", UNLISTED: "已下架", ARCHIVED: "已归档" };
  const unpublishedCount = products.filter((p) => p.hasUnpublishedChanges).length;
  const activeCount = products.filter((p) => p.visibility === "ACTIVE").length;
  const draftCount = products.filter((p) => p.visibility === "DRAFT").length;

  const filtered = products.filter((product) => {
    if (status === "UNPUBLISHED") {
      if (!product.hasUnpublishedChanges) return false;
    } else if (status !== "ALL" && product.visibility !== status) {
      return false;
    }
    if (categoryFilter !== "ALL" && product.draft.categoryId !== categoryFilter) return false;
    return `${product.draft.title} ${product.draft.subtitle} ${product.productId}`
      .toLowerCase()
      .includes(query.trim().toLowerCase());
  });
  const pages = Math.max(1, Math.ceil(filtered.length / 15));
  const currentPage = Math.min(page, pages);

  const missingPrereqs = [
    !brands.some((x) => x.active) && "品牌",
    !categories.some((x) => x.active) && "分类",
    !templates.some((x) => x.active) && "交付模板",
  ].filter(Boolean);

  return (
    <div className="catalog-workspace">
      <PageHead
        eyebrow="MERCHANT WORKSPACE"
        title={current ? current.name : "商店管理"}
        subtitle={
          current?.intro || "商品、分类层级、交付模板与订单，在同一工作区按标签直接切换维护。"
        }
      >
        {stores.length > 1 && (
          <select
            aria-label="切换当前商店"
            disabled={busy}
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            style={{ minWidth: 160 }}
          >
            {stores.map((store) => (
              <option key={store.storeId} value={store.storeId}>
                {store.name}
              </option>
            ))}
          </select>
        )}
        <Button
          secondary
          disabled={busy || loadingAll}
          onClick={async () => {
            await loadStores();
            await loadStoreContent();
          }}
        >
          <RefreshCw size={15} />刷新
        </Button>
        {all && (
          <Button secondary onClick={() => setModal({ type: "store-create" })}>
            <Plus size={15} />新建商店
          </Button>
        )}
        {current && (
          <Button disabled={busy} onClick={() => setModal({ type: "product", item: {} })}>
            <Plus size={16} />新增商品
          </Button>
        )}
      </PageHead>

      {error && (
        <p className="notice-box" role="alert">
          {error}
        </p>
      )}

      {current && (
        <>
          <nav className="adm-subtabs" aria-label="商店管理分区">
            {WORKSPACE_TABS.map(({ id: tabId, label, icon: Icon }) => {
              const count =
                tabId === "products"
                  ? products.length
                  : tabId === "categories"
                    ? categories.length
                    : tabId === "templates"
                      ? `${templates.filter((t) => t.active).length}${moreTemplates ? "+" : ""}`
                      : null;
              return (
                <button
                  key={tabId}
                  type="button"
                  className={workspaceTab === tabId ? "is-active" : ""}
                  aria-current={workspaceTab === tabId ? "page" : undefined}
                  onClick={() => setWorkspaceTab(tabId)}
                >
                  <Icon size={15} />
                  <span>{label}</span>
                  {count !== null && <span className="adm-subtab-count">{count}</span>}
                </button>
              );
            })}
          </nav>

          {missingPrereqs.length > 0 && workspaceTab === "products" && (
            <div className="notice-box" style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
              <span>发布商品前，请先准备好：{missingPrereqs.join("、")}。</span>
              <div className="adm-compact-row-actions">
                {(!brands.some((x) => x.active) || !categories.some((x) => x.active)) && (
                  <Button secondary onClick={() => setWorkspaceTab("categories")}>
                    去配置分类与品牌
                  </Button>
                )}
                {!templates.some((x) => x.active) && (
                  <Button secondary onClick={() => setWorkspaceTab("templates")}>
                    去配置交付模板
                  </Button>
                )}
              </div>
            </div>
          )}

          {workspaceTab === "products" && (
            <>
              <div className="adm-kpi-grid" style={{ gridTemplateColumns: "repeat(4, minmax(0, 1fr))", marginBottom: 16 }}>
                <button
                  type="button"
                  className={`adm-kpi-card ${status === "ALL" ? "tone-blue" : ""}`}
                  onClick={() => setStatus("ALL")}
                >
                  <div className="adm-kpi-top">
                    <span className="adm-kpi-label">已加载商品</span>
                  </div>
                  <strong className="adm-kpi-value">
                    {products.length}
                    {cursor ? "+" : ""}
                  </strong>
                  <small className="adm-kpi-hint">
                    {cursor ? "可点击下方一键继续加载" : "当前商店全部已载入"}
                  </small>
                </button>
                <button
                  type="button"
                  className={`adm-kpi-card ${status === "ACTIVE" ? "tone-sage" : ""}`}
                  onClick={() => setStatus("ACTIVE")}
                >
                  <div className="adm-kpi-top">
                    <span className="adm-kpi-label">已上架</span>
                  </div>
                  <strong className="adm-kpi-value">{activeCount}</strong>
                  <small className="adm-kpi-hint">正在商城展示并可下单</small>
                </button>
                <button
                  type="button"
                  className={`adm-kpi-card ${status === "DRAFT" ? "tone-blue" : ""}`}
                  onClick={() => setStatus("DRAFT")}
                >
                  <div className="adm-kpi-top">
                    <span className="adm-kpi-label">草稿</span>
                  </div>
                  <strong className="adm-kpi-value">{draftCount}</strong>
                  <small className="adm-kpi-hint">编辑完成后可一键发布</small>
                </button>
                <button
                  type="button"
                  className={`adm-kpi-card ${status === "UNPUBLISHED" || unpublishedCount > 0 ? "tone-amber" : ""}`}
                  onClick={() => setStatus("UNPUBLISHED")}
                >
                  <div className="adm-kpi-top">
                    <span className="adm-kpi-label">有未发布修改</span>
                  </div>
                  <strong className="adm-kpi-value">{unpublishedCount}</strong>
                  <small className="adm-kpi-hint">草稿已改动但尚未同步线上</small>
                </button>
              </div>

              <div className="adm-toolbar">
                <div className="adm-toolbar-left">
                  <div className="adm-pill-group" role="group" aria-label="按商品状态筛选">
                    {[
                      ["ALL", "全部"],
                      ["ACTIVE", "已上架"],
                      ["DRAFT", "草稿"],
                      ["UNPUBLISHED", "待发布修改"],
                      ["UNLISTED", "已下架"],
                    ].map(([key, label]) => (
                      <button
                        key={key}
                        type="button"
                        className={status === key ? "is-active" : ""}
                        aria-pressed={status === key}
                        onClick={() => setStatus(key)}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                  <label className="adm-search-box">
                    <Search size={15} />
                    <input
                      aria-label="搜索管理商品"
                      placeholder="搜索商品名称、简介或编号…"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                    />
                    {query && (
                      <button
                        type="button"
                        className="adm-search-clear"
                        aria-label="清空搜索"
                        onClick={() => setQuery("")}
                      >
                        <X size={14} />
                      </button>
                    )}
                  </label>
                  <select
                    aria-label="商品分类筛选"
                    value={categoryFilter}
                    onChange={(e) => setCategoryFilter(e.target.value)}
                  >
                    <option value="ALL">全部分类</option>
                    {categories.map((c) => (
                      <option value={c.categoryId} key={c.categoryId}>
                        {c.parentCategoryId ? `↳ ${c.name}` : c.name}
                      </option>
                    ))}
                  </select>
                </div>
                {cursor && (
                  <div className="adm-toolbar-right">
                    <Button secondary disabled={busy || loadingAll} onClick={loadMoreBatch}>
                      {loadingAll ? "正在批量加载…" : "批量加载更多商品"}
                    </Button>
                  </div>
                )}
              </div>

              <section className="catalog-table-panel">
                <div className="table-wrap">
                  <table className="catalog-table">
                    <thead>
                      <tr>
                        <th scope="col">商品</th>
                        <th scope="col">状态</th>
                        <th scope="col">分类</th>
                        <th scope="col">售价 / 信用点</th>
                        <th scope="col">库存</th>
                        <th scope="col" style={{ width: 220 }}>
                          操作
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {filtered
                        .slice((currentPage - 1) * 15, currentPage * 15)
                        .map((product) => {
                          const image =
                            product.draftImages?.[0] || product.published?.images?.[0];
                          const archived = product.visibility === "ARCHIVED";
                          const menuItems = [
                            !archived && {
                              key: "duplicate",
                              label: "复制为新商品草稿",
                              icon: Copy,
                              onClick: () => duplicateProduct(product),
                            },
                            product.visibility === "ACTIVE" &&
                              product.hasUnpublishedChanges && {
                                key: "republish",
                                label: "发布草稿修改",
                                icon: Upload,
                                onClick: () => action(product, "publish"),
                              },
                            product.visibility === "ACTIVE" && {
                              key: "unlist",
                              label: "下架商品",
                              icon: EyeOff,
                              onClick: () => {
                                setModal({ type: "unlist", item: product });
                                setReason("");
                              },
                            },
                            { divider: true },
                            {
                              key: "delete",
                              label: "删除商品",
                              icon: Trash2,
                              danger: true,
                              onClick: () => {
                                setError("");
                                setModal({ type: "delete", item: product });
                              },
                            },
                          ];
                          return (
                            <tr key={product.productId}>
                              <td>
                                <button
                                  className="catalog-product-cell"
                                  disabled={busy || archived}
                                  onClick={() => setModal({ type: "product", item: product })}
                                >
                                  {image?.url ? (
                                    <img src={image.url} alt="" />
                                  ) : (
                                    <span className="catalog-thumbnail">
                                      <Package size={20} />
                                    </span>
                                  )}
                                  <span>
                                    <strong>{product.draft.title}</strong>
                                    <small>{product.draft.subtitle}</small>
                                  </span>
                                </button>
                              </td>
                              <td>
                                <Badge tone={product.visibility === "ACTIVE" ? "sage" : "neutral"}>
                                  {states[product.visibility]}
                                </Badge>
                                {product.hasUnpublishedChanges && (
                                  <small className="unpublished-hint">有未发布修改</small>
                                )}
                              </td>
                              <td>
                                {categories.find((c) => c.categoryId === product.draft.categoryId)
                                  ?.name || "—"}
                              </td>
                              <td className="catalog-price">
                                {credit(
                                  salePrice(product.draft.price, product.draft.discountRate || 10000),
                                )}
                                {product.draft.discountRate < 10000 && (
                                  <del className="sale-original">{credit(product.draft.price)}</del>
                                )}
                              </td>
                              <td>
                                {product.draft.inventoryPolicy === "UNLIMITED"
                                  ? "不限量"
                                  : (product.availableStock ?? product.draft.stock)}
                              </td>
                              <td>
                                <div className="adm-compact-row-actions">
                                  <Button
                                    secondary
                                    disabled={busy || archived}
                                    onClick={() => setModal({ type: "product", item: product })}
                                  >
                                    编辑
                                  </Button>
                                  {product.visibility !== "ACTIVE" && !archived && (
                                    <Button
                                      disabled={busy}
                                      onClick={() => action(product, "publish")}
                                    >
                                      发布
                                    </Button>
                                  )}
                                  <ActionMenu
                                    disabled={busy}
                                    label={`更多操作 ${product.draft.title}`}
                                    items={menuItems}
                                  />
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                    </tbody>
                  </table>
                </div>
                {!busy && !filtered.length && (
                  <Empty
                    title={products.length ? "没有匹配的商品" : "还没有商品"}
                    text={
                      products.length
                        ? cursor
                          ? "当前仅筛选已加载商品；可点击上方“批量加载更多商品”或调整关键词。"
                          : "调整关键词、状态或分类后重试。"
                        : "添加图片与商品信息，预览后保存并发布。"
                    }
                  />
                )}
                {(busy || loadingAll) && (
                  <p className="catalog-loading" role="status">
                    正在读取商品…
                  </p>
                )}
                <footer className="catalog-pagination">
                  <span>
                    已加载 {products.length} 件 · 筛选结果 {filtered.length} 件
                    {cursor ? "（还有更多未加载）" : ""}
                  </span>
                  <div>
                    <Button
                      secondary
                      aria-label="上一页商品"
                      disabled={currentPage <= 1}
                      onClick={() => setPage(currentPage - 1)}
                    >
                      <ChevronLeft size={16} />
                    </Button>
                    <span>
                      {currentPage} / {pages}
                    </span>
                    <Button
                      secondary
                      aria-label="下一页商品"
                      disabled={currentPage >= pages}
                      onClick={() => setPage(currentPage + 1)}
                    >
                      <ChevronRight size={16} />
                    </Button>
                    {cursor && (
                      <Button secondary disabled={busy || loadingAll} onClick={() => loadStoreContent(true)}>
                        加载下 100 件
                      </Button>
                    )}
                  </div>
                </footer>
              </section>
            </>
          )}

          {workspaceTab === "categories" && (
            <CategoryBrandWorkspace
              client={client}
              storeId={selected}
              categories={categories}
              brands={brands}
              onSaved={loadStoreContent}
            />
          )}

          {workspaceTab === "templates" && (
            <section className="panel adm-card-panel">
              <DeliveryTemplateManagement
                client={client}
                storeId={selected}
                onChanged={loadStoreContent}
              />
            </section>
          )}

          {workspaceTab === "orders" && (
            <section className="panel adm-card-panel">
              <BusinessPages client={client} user={user} type="ORDER" merchantStoreId={selected} />
            </section>
          )}

          {workspaceTab === "settings" && (
            <div className="adm-split-grid">
              <section className="panel adm-card-panel">
                <div className="adm-panel-head">
                  <div>
                    <h2>店铺基本资料</h2>
                    <p>修改商店头像、简介、客服 QQ 与公告。</p>
                  </div>
                  <CopyButton value={current.storeId} label="商店 ID" />
                </div>
                <StoreForm
                  key={`${current.storeId}:${current.version}`}
                  client={client}
                  user={user}
                  initial={current}
                  onSaved={loadStores}
                />
              </section>
              <section className="panel adm-card-panel">
                <div className="adm-panel-head">
                  <div>
                    <h2>商城首页活动图</h2>
                    <p>直接管理首页顶栏下方的横向轮播活动横幅。</p>
                  </div>
                </div>
                <ShopBannerSettings client={client} storeId={selected} onSaved={loadStoreContent} />
              </section>
            </div>
          )}
        </>
      )}

      {!busy && !error && !stores.length && (
        <Empty
          title="还没有可管理的商店"
          text={all ? "先创建商店，再准备分类与交付模板。" : "获得商店授权后，就能在这里维护商品。"}
        />
      )}

      {modal?.type === "store-create" && (
        <Modal title="创建商店" close={() => setModal(null)}>
          <StoreForm
            client={client}
            user={user}
            initial={{}}
            onSaved={async () => {
              setModal(null);
              await loadStores();
            }}
          />
        </Modal>
      )}
      {modal?.type === "product" && (
        <Modal
          title={modal.item.productId ? "编辑商品" : "新增商品"}
          close={() => setModal(null)}
          wide
          className="product-editor-modal"
          guardClose
          dismissOnBackdrop={false}
        >
          <ProductForm
            client={client}
            storeId={selected}
            initial={modal.item}
            brands={brands}
            categories={categories}
            templates={templates}
            onSaved={async ({ close }) => {
              if (close) setModal(null);
              await loadStoreContent();
            }}
          />
        </Modal>
      )}
      {modal?.type === "delete" && (
        <Modal
          title="删除商品"
          dismissOnBackdrop={false}
          close={() => {
            if (!busy) setModal(null);
          }}
        >
          <p>
            确认删除“{modal.item.draft.title}
            ”？商品会停止销售并从管理列表移除，无法重新发布。已有订单仍可交付、领取或退款。
          </p>
          {error && (
            <p className="auth-error" role="alert">
              {error}
            </p>
          )}
          <div className="editor-actions">
            <Button secondary disabled={busy} onClick={() => setModal(null)}>
              取消
            </Button>
            <Button danger disabled={busy} onClick={() => action(modal.item, "delete")}>
              {busy ? "正在删除…" : "确认删除"}
            </Button>
          </div>
        </Modal>
      )}
      {modal?.type === "unlist" && (
        <Modal title="下架商品" close={() => setModal(null)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              action(modal.item, "unlist", reason);
            }}
          >
            <p>确认下架“{modal.item.draft.title}”？已经成交的订单不受影响。</p>
            <Field label="下架原因">
              <textarea
                required
                minLength={2}
                maxLength={500}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </Field>
            {error && (
              <p className="auth-error" role="alert">
                {error}
              </p>
            )}
            <Button type="submit" disabled={busy}>
              确认下架
            </Button>
          </form>
        </Modal>
      )}
    </div>
  );
}

function StoreForm({ client, user, initial, onSaved }) {
  const [value, setValue] = useState({
    name: initial.name || "",
    intro: initial.intro || "",
    contactQq: initial.contactQq || user.qq,
    serviceHours: initial.serviceHours || "",
    notice: initial.notice || "",
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [savedMsg, setSavedMsg] = useState("");
  const [logo, setLogo] = useState(initial.logo || null);
  const [uploading, setUploading] = useState(false);
  useUnsavedChanges({ value, logo: logo?.assetId }, busy || uploading);
  const stable = useStableMutation();
  const save = async (e) => {
    e.preventDefault();
    if (busy || uploading) return;
    setBusy(true);
    setError("");
    setSavedMsg("");
    try {
      const body = stable({
        ...value,
        logoAssetId: logo?.assetId || null,
        coverAssetId: initial.cover?.assetId || null,
        ...(initial.storeId ? { expectedVersion: initial.version } : {}),
      });
      await client.request(
        initial.storeId
          ? `/api/v1/merchant/stores/${encodeURIComponent(initial.storeId)}`
          : "/api/v1/admin/stores",
        { method: initial.storeId ? "PUT" : "POST", body },
      );
      setSavedMsg("商店资料已保存。");
      await onSaved();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={save} className="settings-stack">
      <StoreAvatarPicker
        client={client}
        storeId={initial.storeId}
        name={value.name}
        value={logo}
        onChange={setLogo}
        onBusy={setUploading}
      />
      {[
        ["name", "商店名称", 60],
        ["intro", "商店简介", 300],
        ["contactQq", "联系 QQ", 12],
        ["serviceHours", "服务时间", 100],
        ["notice", "商店公告", 1000],
      ].map(([key, label, max]) => (
        <Field
          key={key}
          label={label}
          value={value[key]}
          onChange={(e) => setValue({ ...value, [key]: e.target.value })}
          required={["name", "contactQq"].includes(key)}
          maxLength={max}
          pattern={key === "contactQq" ? "[0-9]{5,12}" : undefined}
        />
      ))}
      {savedMsg && (
        <p className="notice-box" role="status">
          {savedMsg}
        </p>
      )}
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      <Button type="submit" disabled={busy || uploading}>
        {busy ? "正在保存…" : "保存商店资料"}
      </Button>
    </form>
  );
}

import React, { useEffect, useMemo, useRef, useState } from "react";
import { Plus, Pencil, FolderTree, CornerDownRight, Tag } from "lucide-react";
import { Badge, Button, Empty, Field, Modal } from "./components.jsx";
import { MediaPicker } from "./MediaPicker.jsx";
import { id } from "./format.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

/**
 * Full in-page Visual Category Tree + Brand manager (replaces the old blind dropdown modal).
 */
export function CategoryBrandWorkspace({ client, storeId, categories = [], brands = [], onSaved }) {
  const [modal, setModal] = useState(null);

  const tree = useMemo(() => {
    const sorted = [...categories].sort(
      (a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0) || a.name.localeCompare(b.name, "zh"),
    );
    const roots = sorted.filter((c) => !c.parentCategoryId);
    const childrenByParent = new Map();
    for (const c of sorted) {
      if (c.parentCategoryId) {
        const list = childrenByParent.get(c.parentCategoryId) || [];
        list.push(c);
        childrenByParent.set(c.parentCategoryId, list);
      }
    }
    const orphans = sorted.filter(
      (c) => c.parentCategoryId && !roots.some((r) => r.categoryId === c.parentCategoryId),
    );
    return { roots, childrenByParent, orphans };
  }, [categories]);

  const sortedBrands = useMemo(
    () =>
      [...brands].sort(
        (a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0) || a.name.localeCompare(b.name, "zh"),
      ),
    [brands],
  );

  return (
    <div className="adm-split-grid">
      <section className="panel adm-card-panel">
        <div className="adm-panel-head">
          <div>
            <h2>
              <FolderTree size={17} /> 分类层级树
            </h2>
            <p>一级分类展示在常驻分类栏，二级分类归属对应一级分类下，按排序值升序排列。</p>
          </div>
          <Button onClick={() => setModal({ kind: "category", initial: {} })}>
            <Plus size={15} />新建一级分类
          </Button>
        </div>

        {!tree.roots.length && !tree.orphans.length ? (
          <Empty title="还没有商品分类" text="先创建一级分类，再按需要添加二级子分类。" />
        ) : (
          <div className="adm-cat-tree">
            {tree.roots.map((root) => {
              const children = tree.childrenByParent.get(root.categoryId) || [];
              return (
                <div className="adm-cat-branch" key={root.categoryId}>
                  <div className="adm-cat-node is-root">
                    <div className="adm-cat-info">
                      <strong>{root.name}</strong>
                      <div className="adm-cat-meta">
                        <Badge tone={root.active ? "sage" : "neutral"}>
                          {root.active ? "启用" : "停用"}
                        </Badge>
                        <span>排序 {root.sortOrder ?? 0}</span>
                        <span>{children.length} 个子分类</span>
                      </div>
                    </div>
                    <div className="adm-compact-row-actions">
                      <Button
                        secondary
                        onClick={() =>
                          setModal({
                            kind: "category",
                            initial: { parentCategoryId: root.categoryId },
                          })
                        }
                      >
                        <Plus size={14} />加子分类
                      </Button>
                      <Button
                        secondary
                        onClick={() => setModal({ kind: "category", initial: root })}
                      >
                        <Pencil size={14} />编辑
                      </Button>
                    </div>
                  </div>
                  {children.length > 0 && (
                    <div className="adm-cat-children">
                      {children.map((child) => (
                        <div className="adm-cat-node is-child" key={child.categoryId}>
                          <CornerDownRight size={14} className="adm-cat-elbow" />
                          <div className="adm-cat-info">
                            <strong>{child.name}</strong>
                            <div className="adm-cat-meta">
                              <Badge tone={child.active ? "sage" : "neutral"}>
                                {child.active ? "启用" : "停用"}
                              </Badge>
                              <span>排序 {child.sortOrder ?? 0}</span>
                            </div>
                          </div>
                          <Button
                            secondary
                            onClick={() => setModal({ kind: "category", initial: child })}
                          >
                            <Pencil size={14} />编辑
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}

            {tree.orphans.map((orphan) => (
              <div className="adm-cat-node is-root" key={orphan.categoryId}>
                <div className="adm-cat-info">
                  <strong>{orphan.name}</strong>
                  <div className="adm-cat-meta">
                    <Badge tone="amber">父分类缺失</Badge>
                    <span>排序 {orphan.sortOrder ?? 0}</span>
                  </div>
                </div>
                <Button
                  secondary
                  onClick={() => setModal({ kind: "category", initial: orphan })}
                >
                  <Pencil size={14} />编辑
                </Button>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="panel adm-card-panel">
        <div className="adm-panel-head">
          <div>
            <h2>
              <Tag size={17} /> 商店品牌
            </h2>
            <p>品牌用于商品卡片归属与商城检索。</p>
          </div>
          <Button secondary onClick={() => setModal({ kind: "brand", initial: {} })}>
            <Plus size={15} />新增品牌
          </Button>
        </div>

        {!sortedBrands.length ? (
          <Empty title="还没有品牌" text="添加一个官方或合作系列品牌后即可关联商品。" />
        ) : (
          <div className="adm-brand-list">
            {sortedBrands.map((brand) => (
              <div className="adm-brand-row" key={brand.brandId}>
                <div>
                  <strong>{brand.name}</strong>
                  <small className="muted">
                    {brand.active ? "已启用" : "已停用"} · 排序 {brand.sortOrder ?? 0}
                  </small>
                </div>
                <Badge tone={brand.active ? "sage" : "neutral"}>
                  {brand.active ? "启用" : "停用"}
                </Badge>
              </div>
            ))}
          </div>
        )}
      </section>

      {modal && (
        <Modal
          title={
            modal.kind === "brand"
              ? "新增品牌"
              : modal.initial?.categoryId
                ? `编辑分类 · ${modal.initial.name}`
                : modal.initial?.parentCategoryId
                  ? "新增二级子分类"
                  : "新增一级分类"
          }
          close={() => setModal(null)}
        >
          <CategoryFields
            key={`${modal.kind}:${modal.initial?.categoryId || modal.initial?.parentCategoryId || "new"}`}
            client={client}
            storeId={storeId}
            kind={modal.kind}
            initial={modal.initial}
            categories={categories}
            onSaved={async () => {
              setModal(null);
              await onSaved?.();
            }}
          />
        </Modal>
      )}
    </div>
  );
}

export function CategoryHierarchy({ client, storeId, categories, onSaved }) {
  const [selected, setSelected] = useState(categories[0]?.categoryId || "");
  const current = categories.find((c) => c.categoryId === selected);
  return (
    <div className="settings-stack">
      <p>一级分类显示在 App 常驻分类栏；二级分类显示在对应分类页，点击后搜索该分类名称。</p>
      <Field label="要编辑的分类">
        <select value={selected} onChange={(e) => setSelected(e.target.value)}>
          {categories.map((c) => (
            <option key={c.categoryId} value={c.categoryId}>
              {c.name}
            </option>
          ))}
        </select>
      </Field>
      {current && (
        <CategoryFields
          key={`${current.categoryId}:${current.version}`}
          client={client}
          storeId={storeId}
          kind="category"
          initial={current}
          categories={categories}
          onSaved={onSaved}
        />
      )}
    </div>
  );
}

export function CategoryFields({ client, storeId, kind, initial = {}, categories = [], onSaved }) {
  const [name, setName] = useState(initial.name || "");
  const [order, setOrder] = useState(initial.sortOrder || 0);
  const [parent, setParent] = useState(initial.parentCategoryId || "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const request = useRef(null);
  const guard = useUnsavedChanges({ name, order, parent }, busy);
  const roots = categories.filter(
    (c) => c.active && !c.parentCategoryId && c.categoryId !== initial.categoryId,
  );
  const hasChildren = categories.some((c) => c.parentCategoryId === initial.categoryId);
  return (
    <form
      className="settings-stack"
      onSubmit={async (e) => {
        e.preventDefault();
        if (busy) return;
        setBusy(true);
        setError("");
        try {
          const body = {
            name,
            sortOrder: Number(order),
            active: initial.active ?? true,
            ...(kind === "brand" ? { logoAssetId: null } : { parentCategoryId: parent || null }),
            ...(initial.categoryId ? { expectedVersion: initial.version } : {}),
          };
          const fingerprint = JSON.stringify(body);
          if (request.current?.fingerprint !== fingerprint)
            request.current = { fingerprint, body: { ...body, clientRequestId: id() } };
          await client.request(
            `/api/v1/merchant/stores/${encodeURIComponent(storeId)}/${kind === "brand" ? "brands" : "categories"}${initial.categoryId ? `/${encodeURIComponent(initial.categoryId)}` : ""}`,
            { method: initial.categoryId ? "PUT" : "POST", body: request.current.body },
          );
          guard.markSaved({ name, order, parent });
          await onSaved();
        } catch (e) {
          setError(e.message);
        } finally {
          setBusy(false);
        }
      }}
    >
      <fieldset disabled={busy} style={{ border: 0, padding: 0, margin: 0 }}>
        <Field
          label="名称"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          maxLength={kind === "brand" ? 60 : 40}
        />
        {kind === "category" && (
          <Field
            label="所属一级分类"
            hint={
              hasChildren
                ? "当前分类包含子分类，需先移出子分类才能更改归属。"
                : "选择“作为一级分类”，即可在商城顶栏展示。"
            }
          >
            <select
              disabled={hasChildren}
              value={parent}
              onChange={(e) => setParent(e.target.value)}
            >
              <option value="">作为一级分类</option>
              {roots.map((c) => (
                <option key={c.categoryId} value={c.categoryId}>
                  {c.name}
                </option>
              ))}
            </select>
          </Field>
        )}
        <Field
          label="排序（数字越小越靠前）"
          type="number"
          min={0}
          max={9999}
          value={order}
          onChange={(e) => setOrder(e.target.value)}
        />
      </fieldset>
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      <Button type="submit" disabled={busy}>
        保存
      </Button>
    </form>
  );
}

export function ShopBannerSettings({ client, storeId, onSaved }) {
  const [home, setHome] = useState(null);
  const [images, setImages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [savedMsg, setSavedMsg] = useState("");
  const request = useRef(null);
  const path = `/api/v1/merchant/stores/${encodeURIComponent(storeId)}/homepage`;
  useEffect(() => {
    let alive = true;
    client
      .request(path)
      .then(({ data }) => {
        if (!alive) return;
        setHome(data);
        setImages(
          (data.sections || [])
            .filter((s) => s.layout === "BANNER")
            .sort((a, b) => a.sortOrder - b.sortOrder)
            .map((s) => ({
              ...data.banners?.find((b) => b.sectionId === s.sectionId)?.image,
              assetId: s.bannerAssetId,
              altText: s.title,
              sectionId: s.sectionId,
            })),
        );
      })
      .catch((e) => {
        if (!alive) return;
        if (e.status === 404)
          setHome({ version: 1, intro: "", sections: [], brandIds: [], categoryIds: [] });
        else setError(e.message);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [client, path]);
  if (loading) return <p role="status">正在读取活动图…</p>;
  if (!home)
    return (
      <p className="auth-error" role="alert">
        {error}
      </p>
    );
  return (
    <form
      className="settings-stack"
      onSubmit={async (e) => {
        e.preventDefault();
        if (busy) return;
        setBusy(true);
        setError("");
        setSavedMsg("");
        try {
          const sections = [
            ...home.sections.filter((s) => s.layout !== "BANNER"),
            ...images.map((image, index) => ({
              sectionId: image.sectionId || `banner_${image.assetId}`,
              title: (image.altText || "商城活动").slice(0, 80),
              layout: "BANNER",
              productIds: [],
              bannerAssetId: image.assetId,
              sortOrder: index,
            })),
          ];
          if (!sections.length)
            sections.push({
              sectionId: "products",
              title: "商品",
              layout: "PRODUCT_GRID",
              productIds: [],
              bannerAssetId: null,
              sortOrder: 0,
            });
          const body = {
            expectedVersion: home.version,
            intro: home.intro || "",
            brandIds: home.brandIds || [],
            categoryIds: home.categoryIds || [],
            sections,
          };
          const fingerprint = JSON.stringify(body);
          if (request.current?.fingerprint !== fingerprint)
            request.current = { fingerprint, body: { ...body, clientRequestId: id() } };
          const result = await client.request(path, {
            method: "PUT",
            body: request.current.body,
          });
          setHome(result.data);
          setSavedMsg("首页活动图已保存。");
          await onSaved?.();
        } catch (e) {
          setError(e.message);
        } finally {
          setBusy(false);
        }
      }}
    >
      <p className="muted" style={{ margin: 0, fontSize: 13 }}>
        活动图显示在商城首页分类栏下方，支持横向滑动。建议使用约 2.65 : 1 的横图。
      </p>
      <MediaPicker
        client={client}
        images={images}
        onChange={setImages}
        purpose="STORE_MEDIA"
        businessType="STORE"
        businessRef={storeId}
        max={Math.min(8, 20 - home.sections.filter((s) => s.layout !== "BANNER").length)}
        onBusy={setBusy}
      />
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
      <Button type="submit" disabled={busy}>
        保存活动图
      </Button>
    </form>
  );
}

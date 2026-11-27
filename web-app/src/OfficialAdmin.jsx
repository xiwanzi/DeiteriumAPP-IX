import React, { useEffect, useRef, useState } from "react";
import { Plus, Upload, ArrowUp, ArrowDown, Trash2, Search, X, RefreshCw } from "lucide-react";
import CoreAdmin from "./CoreAdmin.jsx";
import { Badge, Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import { ContentBlocks } from "./RemoteCollection.jsx";
import { id } from "./format.js";
import { uploadAsset } from "./assets.js";
import InterventionManagement from "./Interventions.jsx";
import TradeAudit from "./TradeAudit.jsx";
import MailSystem from "./MailSystem.jsx";
import SupportWorkspace from "./SupportWorkspace.jsx";
import LauncherIconSettings from "./LauncherIconSettings.jsx";
import DesktopLauncherSettings from "./DesktopLauncherSettings.jsx";
import AiSettings from "./AiSettings.jsx";
import AccountManagement from "./AccountManagement.jsx";
import WhitelistManagement from "./WhitelistManagement.jsx";
import KeybindingManagement from "./KeybindingManagement.jsx";
import TitleManagement from "./TitleManagement.jsx";
import SupplyManagement from "./SupplyManagement.jsx";
import ImageReview from "./ImageReview.jsx";
import YsmManagement from "./YsmCloudManagement.jsx";
import CatalogManagement from "./CatalogManagement.jsx";
import CouponManagement from "./CouponManagement.jsx";
import AdminOverview from "./AdminOverview.jsx";
import { adminNavigationGroups } from "./admin-navigation.js";
import { useUnsavedChanges } from "./unsaved-changes.js";

export default function OfficialAdmin({
  client,
  user,
  navigate,
  search = "",
  adminSignals = {},
  refreshingSignals = false,
  onRefreshSignals,
  pinnedPaths = [],
  onTogglePin,
  recentPaths = [],
  onOpenPalette,
}) {
  const permissions = user.permissions || [];
  const all = permissions.includes("platform.admin");
  const { hasOverview } = adminNavigationGroups(permissions);

  const tabs = [
    ...(hasOverview ? ["工作台总览"] : []),
    ...(all || permissions.includes("support.manage") ? ["工单系统"] : []),
    ...(all || permissions.includes("intervention.manage") ? ["平台介入"] : []),
    ...(all || permissions.includes("announcements.manage") ? ["公告管理"] : []),
    ...(all || permissions.includes("audit.read") ? ["管理审计"] : []),
    ...(all
      ? [
          "补给管理",
          "小祥设置",
          "账号与权限",
          "白名单管理",
          "邮件系统",
          "应用图标",
          "优惠券",
          "启动器",
          "模型管理",
          "头衔管理",
          "玩家键位",
          "图片审核",
        ]
      : []),
    ...(all ||
    permissions.some((p) =>
      ["store.manage", "store.members.manage", "commerce.manage"].includes(p),
    )
      ? ["商店管理"]
      : []),
    ...(all || permissions.includes("core.read") ? ["Core 管理"] : []),
  ];

  const requested = {
    overview: "工作台总览",
    supply: "补给管理",
    keybindings: "玩家键位",
    "image-review": "图片审核",
    titles: "头衔管理",
    support: "工单系统",
    ysm: "模型管理",
    interventions: "平台介入",
    announcements: "公告管理",
    audit: "管理审计",
    email: "邮件系统",
    ai: "小祥设置",
    accounts: "账号与权限",
    whitelist: "白名单管理",
    core: "Core 管理",
    coupons: "优惠券",
    "launcher-icon": "应用图标",
    "desktop-launcher": "启动器",
  }[new URLSearchParams(search).get("section")];

  const tab = tabs.includes(requested) ? requested : tabs[0];
  const pending = adminSignals.interventionsPending ?? 0;

  return (
    <>
      {pending > 0 &&
        tab !== "平台介入" &&
        tab !== "工单系统" &&
        tab !== "工作台总览" && (
          <div className="intervention-alert" role="status">
            <span>
              <strong>{pending} 起平台介入</strong>需要继续跟进
            </span>
            <Button secondary onClick={() => navigate("/admin?section=interventions")}>
              查看案件
            </Button>
          </div>
        )}
      {tab === "工作台总览" ? (
        <AdminOverview
          user={user}
          navigate={navigate}
          signals={adminSignals}
          refreshingSignals={refreshingSignals}
          onRefreshSignals={onRefreshSignals}
          pinnedPaths={pinnedPaths}
          onTogglePin={onTogglePin}
          recentPaths={recentPaths}
          onOpenPalette={onOpenPalette}
        />
      ) : tab === "补给管理" ? (
        <SupplyManagement client={client} user={user} />
      ) : tab === "玩家键位" ? (
        <KeybindingManagement client={client} />
      ) : tab === "图片审核" ? (
        <ImageReview client={client} navigate={navigate} search={search} />
      ) : tab === "头衔管理" ? (
        <TitleManagement client={client} />
      ) : tab === "工单系统" ? (
        <SupportWorkspace client={client} user={user} admin />
      ) : tab === "模型管理" ? (
        <YsmManagement client={client} />
      ) : tab === "白名单管理" ? (
        <WhitelistManagement client={client} />
      ) : tab === "启动器" ? (
        <DesktopLauncherSettings client={client} />
      ) : tab === "优惠券" ? (
        <CouponManagement client={client} />
      ) : tab === "应用图标" ? (
        <LauncherIconSettings client={client} />
      ) : tab === "小祥设置" ? (
        <AiSettings client={client} />
      ) : tab === "账号与权限" ? (
        <AccountManagement client={client} user={user} />
      ) : tab === "Core 管理" ? (
        <CoreAdmin client={client} />
      ) : tab === "公告管理" ? (
        <AnnouncementManagement client={client} />
      ) : tab === "平台介入" ? (
        <InterventionManagement client={client} user={user} />
      ) : tab === "管理审计" ? (
        <TradeAudit client={client} />
      ) : tab === "邮件系统" ? (
        <MailSystem client={client} />
      ) : (
        <CatalogManagement client={client} user={user} />
      )}
    </>
  );
}

function AnnouncementManagement({ client }) {
  const [items, setItems] = useState([]);
  const [cursor, setCursor] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [editor, setEditor] = useState(null);
  const [withdraw, setWithdraw] = useState(null);
  const [reason, setReason] = useState("");
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [query, setQuery] = useState("");
  const actionKeys = useRef({});
  const [deleting, setDeleting] = useState(null);

  const load = async (more = false) => {
    setBusy(true);
    setError("");
    try {
      const r = await client.request(
        `/api/v1/admin/announcements?limit=30${more && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
      );
      setItems((old) => (more ? [...old, ...r.data] : r.data));
      setCursor(r.page?.nextCursor);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  useEffect(() => {
    load();
  }, []);

  const action = async (item, actionName, explanation = "") => {
    if (busy) return;
    setBusy(true);
    setError("");
    const scope = `${item.announcementId}:${item.version}:${actionName}:${explanation}`;
    actionKeys.current[scope] ||= id();
    try {
      await client.request(
        `/api/v1/admin/announcements/${encodeURIComponent(item.announcementId)}/${actionName}`,
        {
          method: "POST",
          body: {
            clientRequestId: actionKeys.current[scope],
            expectedVersion: item.version,
            ...(actionName === "unpublish" ? { reason: explanation } : {}),
          },
        },
      );
      setWithdraw(null);
      setDeleting(null);
      await load();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const filtered = items.filter((item) => {
    if (statusFilter !== "ALL" && item.status !== statusFilter) return false;
    const q = query.trim().toLowerCase();
    if (!q) return true;
    return `${item.title || ""} ${item.summary || ""}`.toLowerCase().includes(q);
  });

  const counts = {
    ALL: items.length,
    PUBLISHED: items.filter((i) => i.status === "PUBLISHED").length,
    DRAFT: items.filter((i) => i.status === "DRAFT").length,
    WITHDRAWN: items.filter((i) => i.status === "WITHDRAWN").length,
  };

  return (
    <>
      <PageHead
        eyebrow="OFFICIAL ANNOUNCEMENTS"
        title="公告管理"
        subtitle="保存草稿，确认后再向社区发布。"
      >
        <Button secondary disabled={busy} onClick={() => load()}>
          <RefreshCw size={15} />刷新
        </Button>
        <Button onClick={() => setEditor({})}>
          <Plus size={16} />新建公告
        </Button>
      </PageHead>

      <div className="adm-toolbar">
        <div className="adm-toolbar-left">
          <div className="adm-pill-group" role="group" aria-label="按公告状态筛选">
            {[
              ["ALL", "全部"],
              ["PUBLISHED", "已发布"],
              ["DRAFT", "草稿"],
              ["WITHDRAWN", "已撤下"],
            ].map(([key, label]) => (
              <button
                key={key}
                type="button"
                className={statusFilter === key ? "is-active" : ""}
                aria-pressed={statusFilter === key}
                onClick={() => setStatusFilter(key)}
              >
                {label} ({counts[key]})
              </button>
            ))}
          </div>
          <label className="adm-search-box">
            <Search size={15} />
            <input
              aria-label="搜索公告"
              placeholder="搜索公告标题或摘要…"
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
        </div>
      </div>

      {error && (
        <p className="notice-box" role="alert">
          {error}
        </p>
      )}
      {filtered.length > 0 && (
        <div className="announcement-admin-list">
          {filtered.map((item) => (
            <article key={item.announcementId}>
              {item.cover?.url && (
                <div className="admin-ann-cover">
                  <img src={item.cover.url} alt={item.title} />
                </div>
              )}
              <div className="admin-ann-copy">
                <div>
                  <Badge tone={item.status === "PUBLISHED" ? "sage" : "neutral"}>
                    {{ DRAFT: "草稿", PUBLISHED: "已发布", WITHDRAWN: "已撤下" }[item.status] ||
                      "公告"}
                  </Badge>
                  {item.pinned && <Badge>置顶</Badge>}
                  {item.hasUnpublishedChanges && item.status === "PUBLISHED" && (
                    <Badge tone="amber">有未发布修改</Badge>
                  )}
                </div>
                <h3>{item.title}</h3>
                <p>{item.summary}</p>
              </div>
              <div className="adm-compact-row-actions">
                <Button secondary disabled={busy} onClick={() => setEditor(item)}>
                  编辑
                </Button>
                <Button disabled={busy} onClick={() => action(item, "publish")}>
                  {item.status === "PUBLISHED" ? "发布更新" : "发布"}
                </Button>
                {item.status === "PUBLISHED" && (
                  <Button
                    secondary
                    disabled={busy}
                    onClick={() => {
                      setWithdraw(item);
                      setReason("");
                    }}
                  >
                    撤下
                  </Button>
                )}
                <Button
                  danger
                  secondary
                  disabled={busy}
                  aria-label={`永久删除 ${item.title}`}
                  onClick={() => {
                    setDeleting(item);
                    setError("");
                  }}
                >
                  <Trash2 size={15} />
                </Button>
              </div>
            </article>
          ))}
        </div>
      )}
      {!busy && !filtered.length && (
        <Empty
          title={items.length ? "没有匹配的公告" : "还没有公告"}
          text={
            items.length
              ? "切换状态标签或清空关键词重试。"
              : "创建并发布后，玩家就能在 App 和网页看到。"
          }
        />
      )}
      {cursor && (
        <Button secondary disabled={busy} onClick={() => load(true)}>
          加载更多
        </Button>
      )}
      {editor && (
        <Modal
          title={editor.announcementId ? "编辑公告" : "新建公告"}
          close={() => setEditor(null)}
          wide
          guardClose
          dismissOnBackdrop={false}
        >
          <AnnouncementForm
            client={client}
            initial={editor}
            onSaved={async () => {
              setEditor(null);
              await load();
            }}
          />
        </Modal>
      )}
      {deleting && (
        <Modal
          title="永久删除公告"
          close={() => {
            if (!busy) setDeleting(null);
          }}
          dismissOnBackdrop={false}
        >
          <p>确认永久删除“{deleting.title}”？</p>
          <p>正文、草稿、发布版本和关联通知都会删除，无法恢复。</p>
          {error && (
            <p className="auth-error" role="alert">
              {error}
            </p>
          )}
          <div className="button-row">
            <Button secondary disabled={busy} onClick={() => setDeleting(null)}>
              保留公告
            </Button>
            <Button danger disabled={busy} onClick={() => action(deleting, "delete")}>
              {busy ? "正在删除…" : "确认永久删除"}
            </Button>
          </div>
        </Modal>
      )}
      {withdraw && (
        <Modal title="撤下公告" close={() => setWithdraw(null)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              action(withdraw, "unpublish", reason);
            }}
          >
            <p>撤下“{withdraw.title}”后，玩家将无法查看此公告。</p>
            <Field label="撤下原因">
              <textarea
                required
                maxLength={500}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </Field>
            {error && (
              <p role="alert" className="auth-error">
                {error}
              </p>
            )}
            <Button type="submit" disabled={busy}>
              确认撤下
            </Button>
          </form>
        </Modal>
      )}
    </>
  );
}

function AnnouncementForm({ client, initial, onSaved }) {
  const [title, setTitle] = useState(initial.title || "");
  const [summary, setSummary] = useState(initial.summary || "");
  const [blocks, setBlocks] = useState(
    initial.contentBlocks || [{ blockId: id(), type: "PARAGRAPH", text: "" }],
  );
  const [cover, setCover] = useState(initial.cover || null);
  const [media, setMedia] = useState(initial.media || []);
  const [pinned, setPinned] = useState(initial.pinned || false);
  const [priority, setPriority] = useState(initial.priority || "NORMAL");
  const [preview, setPreview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(null);
  const [error, setError] = useState("");
  useUnsavedChanges({ title, summary, blocks, cover: cover?.assetId, pinned, priority }, busy);
  const request = useRef(null);
  const coverInput = useRef(null);
  const imageInput = useRef(null);
  const update = (index, patch) =>
    setBlocks((old) => old.map((block, n) => (n === index ? { ...block, ...patch } : block)));
  const move = (index, delta) =>
    setBlocks((old) => {
      const next = [...old];
      const target = index + delta;
      if (target < 0 || target >= next.length) return old;
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  const upload = async (file, asCover) => {
    if (!file || busy) return;
    setBusy(true);
    setError("");
    try {
      const asset = await uploadAsset(
        client,
        file,
        "ANNOUNCEMENT_MEDIA",
        "ANNOUNCEMENT",
        initial.announcementId || "",
        (value, label) => setProgress({ value, label }),
      );
      setMedia((old) => [...old, asset]);
      if (asCover) setCover(asset);
      else
        setBlocks((old) => [
          ...old,
          { blockId: id(), type: "IMAGE", assetId: asset.assetId, altText: file.name },
        ]);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
      setProgress(null);
      if (coverInput.current) coverInput.current.value = "";
      if (imageInput.current) imageInput.current.value = "";
    }
  };
  const save = async (event) => {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    const input = {
      title,
      summary,
      contentBlocks: blocks,
      coverAssetId: cover?.assetId || null,
      pinned,
      priority,
      ...(initial.announcementId ? { expectedVersion: initial.version } : {}),
    };
    const fingerprint = JSON.stringify(input);
    if (request.current?.fingerprint !== fingerprint)
      request.current = { fingerprint, body: { clientRequestId: id(), ...input } };
    try {
      await client.request(
        initial.announcementId
          ? `/api/v1/admin/announcements/${encodeURIComponent(initial.announcementId)}`
          : "/api/v1/admin/announcements",
        { method: initial.announcementId ? "PUT" : "POST", body: request.current.body },
      );
      await onSaved();
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={save}>
      <fieldset disabled={busy} style={{ padding: 0, border: 0, margin: 0, minWidth: 0 }}>
        <Field label="标题" required maxLength={100} value={title} onChange={(e) => setTitle(e.target.value)} />
        <Field label="摘要">
          <textarea required maxLength={300} rows={2} value={summary} onChange={(e) => setSummary(e.target.value)} />
        </Field>
        <div className="button-row">
          <Button secondary onClick={() => coverInput.current?.click()}>
            <Upload size={16} />
            {cover ? "更换封面" : "上传封面"}
          </Button>
          {cover && (
            <Button secondary onClick={() => setCover(null)}>
              移除封面
            </Button>
          )}
          <Button secondary onClick={() => setPreview(!preview)}>
            {preview ? "继续编辑" : "预览正文"}
          </Button>
        </div>
        <input hidden ref={coverInput} type="file" accept="image/png,image/jpeg,image/webp" onChange={(e) => upload(e.target.files?.[0], true)} />
        <input hidden ref={imageInput} type="file" accept="image/png,image/jpeg,image/webp" onChange={(e) => upload(e.target.files?.[0], false)} />
        {cover?.url && (
          <img
            src={cover.url}
            alt="公告封面"
            style={{ width: "100%", maxHeight: 250, objectFit: "cover", borderRadius: 12, marginBottom: 20 }}
          />
        )}
        {preview ? (
          <div className="panel" style={{ padding: 20 }}>
            <h2>{title}</h2>
            <p>{summary}</p>
            <ContentBlocks blocks={blocks} media={media} />
          </div>
        ) : (
          blocks.map((block, index) => (
            <div key={block.blockId} className="panel" style={{ padding: 18, margin: "12px 0" }}>
              <div className="button-row">
                <Badge tone="neutral">
                  {{ HEADING: "小标题", PARAGRAPH: "段落", IMAGE: "图片", KEY_VALUE_LIST: "信息列表" }[block.type]}
                </Badge>
                <Button secondary aria-label={`上移第 ${index + 1} 段`} onClick={() => move(index, -1)}>
                  <ArrowUp size={14} />
                </Button>
                <Button secondary aria-label={`下移第 ${index + 1} 段`} onClick={() => move(index, 1)}>
                  <ArrowDown size={14} />
                </Button>
                <Button
                  secondary
                  disabled={blocks.length < 2}
                  aria-label={`删除第 ${index + 1} 段`}
                  onClick={() => setBlocks((old) => old.filter((_, n) => n !== index))}
                >
                  <Trash2 size={14} />
                </Button>
              </div>
              {block.type === "HEADING" && (
                <Field label="小标题" required maxLength={100} value={block.heading || ""} onChange={(e) => update(index, { heading: e.target.value })} />
              )}
              {block.type === "PARAGRAPH" && (
                <Field label="段落内容">
                  <textarea required rows={4} maxLength={3000} value={block.text || ""} onChange={(e) => update(index, { text: e.target.value })} />
                </Field>
              )}
              {block.type === "IMAGE" && (
                <>
                  <ContentBlocks blocks={[block]} media={media} />
                  <Field label="图片说明" maxLength={200} value={block.altText || ""} onChange={(e) => update(index, { altText: e.target.value })} />
                </>
              )}
              {block.type === "KEY_VALUE_LIST" &&
                block.rows?.map((row, n) => (
                  <div key={n}>
                    <Field
                      label="名称"
                      required
                      maxLength={80}
                      value={row.label}
                      onChange={(e) =>
                        update(index, {
                          rows: block.rows.map((value, i) => (i === n ? { ...value, label: e.target.value } : value)),
                        })
                      }
                    />
                    <Field
                      label="内容"
                      required
                      maxLength={500}
                      value={row.value}
                      onChange={(e) =>
                        update(index, {
                          rows: block.rows.map((value, i) => (i === n ? { ...value, value: e.target.value } : value)),
                        })
                      }
                    />
                  </div>
                ))}
            </div>
          ))
        )}
        {!preview && (
          <div className="button-row">
            <Button secondary disabled={blocks.length >= 100} onClick={() => setBlocks((old) => [...old, { blockId: id(), type: "PARAGRAPH", text: "" }])}>
              添加段落
            </Button>
            <Button secondary disabled={blocks.length >= 100} onClick={() => setBlocks((old) => [...old, { blockId: id(), type: "HEADING", heading: "" }])}>
              添加小标题
            </Button>
            <Button secondary disabled={blocks.length >= 100} onClick={() => imageInput.current?.click()}>
              添加图片
            </Button>
          </div>
        )}
        <Field label="重要程度">
          <select value={priority} onChange={(e) => setPriority(e.target.value)}>
            <option value="NORMAL">普通</option>
            <option value="IMPORTANT">重要</option>
          </select>
        </Field>
        <label className="checkbox-row">
          <input type="checkbox" checked={pinned} onChange={(e) => setPinned(e.target.checked)} />
          置顶公告
        </label>
      </fieldset>
      {progress && (
        <p role="status">
          {progress.label} · {progress.value}%
        </p>
      )}
      {error && (
        <p className="auth-error" role="alert">
          {error}
        </p>
      )}
      <div className="button-row">
        <Button type="submit" disabled={busy}>
          {busy ? "正在处理…" : "保存草稿"}
        </Button>
      </div>
    </form>
  );
}

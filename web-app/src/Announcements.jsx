import React, { useState } from "react";
import {
  Plus,
  Search,
  Eye,
  PenLine,
  Send,
  Archive,
  Pin,
  Megaphone,
  ArrowUpRight,
  ChevronLeft,
} from "lucide-react";
import {
  Button,
  Field,
  Tabs,
  Badge,
  Empty,
  Modal,
  media,
} from "./components.jsx";
export const publishedAnnouncements = (state) =>
  (state.announcements || [])
    .filter((a) => a.status === "PUBLISHED" && a.published)
    .map((a) => a.published)
    .sort(
      (a, b) =>
        Number(b.pinned) - Number(a.pinned) ||
        (b.publishedAt || "").localeCompare(a.publishedAt || ""),
    );
export function AnnouncementAdmin({ state, open }) {
  const [tab, setTab] = useState("全部公告"),
    [query, setQuery] = useState("");
  const status = { DRAFT: "草稿", PUBLISHED: "已发布", WITHDRAWN: "已撤下" };
  const list = (state.announcements || []).filter(
    (a) =>
      (tab === "全部公告" || status[a.status] === tab) &&
      `${a.title}${a.summary}`.includes(query),
  );
  return (
    <>
      <div className="workspace-section-head">
        <div>
          <h2>公告管理</h2>
          <p>写下社区的新消息，让每一位玩家及时了解。</p>
        </div>
        <Button onClick={() => open({ type: "announcement-edit" })}>
          <Plus size={17} />
          新建公告
        </Button>
      </div>
      <div className="admin-filterbar">
        <Tabs
          values={["全部公告", "草稿", "已发布", "已撤下"]}
          value={tab}
          onChange={setTab}
        />
        <label className="search-input">
          <Search size={16} />
          <input
            aria-label="搜索管理公告"
            placeholder="搜索标题或摘要"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
      </div>
      <div className="announcement-admin-list">
        {list.map((a) => (
          <article key={a.id}>
            <div className="admin-ann-cover">
              <img
                src={media(a.cover || "street-daylight.png")}
                alt="公告封面"
              />
            </div>
            <div className="admin-ann-copy">
              <div>
                <Badge tone={a.status === "PUBLISHED" ? "sage" : "neutral"}>
                  {status[a.status]}
                </Badge>
                {a.pinned && (
                  <Badge tone="neutral">
                    <Pin size={11} />
                    置顶
                  </Badge>
                )}
                {a.hasDraftChanges && <Badge tone="amber">有未发布修改</Badge>}
              </div>
              <h3>{a.title}</h3>
              <p>{a.summary}</p>
              <small>
                更新于 {new Date(a.updatedAt).toLocaleString("zh-CN")} · 第{" "}
                {a.version} 版
              </small>
            </div>
            <div className="admin-ann-actions">
              <button
                className="text-button"
                onClick={() => open({ type: "announcement-edit", item: a })}
              >
                <PenLine size={14} />
                编辑
              </button>
              <button
                className="text-button"
                onClick={() =>
                  open({
                    type: "announcement",
                    item: { ...a, intro: a.summary, preview: true },
                  })
                }
              >
                <Eye size={14} />
                预览草稿
              </button>
              <button
                className="text-button"
                onClick={() =>
                  open({
                    type: "confirm",
                    title:
                      a.status === "PUBLISHED" ? "发布公告新版本" : "发布公告",
                    description: `「${a.title}」将出现在玩家公告页，并生成通知。`,
                    action: {
                      type: "ANNOUNCEMENT_PUBLISH",
                      id: a.id,
                      expectedVersion: a.version,
                    },
                    success: "公告已发布",
                  })
                }
              >
                <Send size={14} />
                {a.status === "PUBLISHED" ? "发布更新" : "发布"}
              </button>
              {a.status === "PUBLISHED" && (
                <button
                  className="text-button red-text"
                  onClick={() =>
                    open({
                      type: "confirm",
                      title: "撤下公告",
                      description: "撤下后不再向玩家展示，公告与历史记录保留。",
                      action: {
                        type: "ANNOUNCEMENT_WITHDRAW",
                        id: a.id,
                        expectedVersion: a.version,
                      },
                      success: "公告已撤下",
                      danger: true,
                    })
                  }
                >
                  <Archive size={14} />
                  撤下
                </button>
              )}
            </div>
          </article>
        ))}
      </div>
      {!list.length && (
        <Empty
          title="还没有这一类公告"
          text="新建一份草稿，预览确认后再发布。"
        />
      )}
    </>
  );
}
export function AnnouncementEditor({ item, act, close }) {
  const original = {
    title: item?.title || "",
    summary: item?.summary || "",
    body: item?.body || "",
    cover: item?.cover || "street-daylight.png",
    pinned: item?.pinned || false,
    priority: item?.priority || "NORMAL",
  };
  const [form, setForm] = useState(original),
    [preview, setPreview] = useState(false),
    [discard, setDiscard] = useState(false);
  const changed = JSON.stringify(form) !== JSON.stringify(original),
    update = (key, value) => setForm((f) => ({ ...f, [key]: value }));
  const exit = () => (changed ? setDiscard(true) : close());
  return (
    <Modal title={item ? "编辑公告草稿" : "新建公告"} close={exit} wide>
      {discard ? (
        <div className="confirm">
          <h2>保留未保存的内容？</h2>
          <p>离开将丢失当前编辑，已经保存或发布的版本不会改变。</p>
          <div className="button-row">
            <Button secondary onClick={() => setDiscard(false)}>
              继续编辑
            </Button>
            <Button danger onClick={close}>
              放弃修改
            </Button>
          </div>
        </div>
      ) : (
        <>
          <Tabs
            values={["编辑内容", "预览公告"]}
            value={preview ? "预览公告" : "编辑内容"}
            onChange={(v) => setPreview(v === "预览公告")}
          />
          {preview ? (
            <AnnouncementArticle item={{ ...form, preview: true }} />
          ) : (
            <form
              id="announcement-form"
              className="form-stack"
              onSubmit={(e) => {
                e.preventDefault();
                if (
                  act(
                    item
                      ? {
                          type: "ANNOUNCEMENT_SAVE",
                          id: item.id,
                          expectedVersion: item.version,
                          data: form,
                        }
                      : { type: "ANNOUNCEMENT_CREATE", data: form },
                    "公告草稿已保存",
                  )
                )
                  close();
              }}
            >
              <div className="form-grid">
                <Field
                  label="公告标题"
                  required
                  maxLength={100}
                  value={form.title}
                  onChange={(e) => update("title", e.target.value)}
                  placeholder="给社区带来了什么新消息？"
                />
                <Field label="重要程度">
                  <select
                    value={form.priority}
                    onChange={(e) => update("priority", e.target.value)}
                  >
                    <option value="NORMAL">普通公告</option>
                    <option value="IMPORTANT">重要公告</option>
                  </select>
                </Field>
              </div>
              <Field
                label="摘要"
                required
                maxLength={300}
                value={form.summary}
                onChange={(e) => update("summary", e.target.value)}
                placeholder="用于公告列表与通知预览"
              />
              <Field
                label="正文"
                hint={`${form.body.length} / 12000 · 以纯文本安全展示，可分段，不执行 HTML`}
              >
                <textarea
                  required
                  rows={9}
                  maxLength={12000}
                  value={form.body}
                  onChange={(e) => update("body", e.target.value)}
                  placeholder="把需要说明的内容写在这里…"
                />
              </Field>
              <div className="form-grid">
                <Field label="封面素材">
                  <select
                    value={form.cover}
                    onChange={(e) => update("cover", e.target.value)}
                  >
                    <option value="street-daylight.png">主城花桥</option>
                    <option value="plaza-sunset.png">中心广场</option>
                    <option value="friends-group.png">好友合影</option>
                    <option value="flower-shop.png">花店日常</option>
                  </select>
                </Field>
                <label className="checkbox-field">
                  <input
                    type="checkbox"
                    checked={form.pinned}
                    onChange={(e) => update("pinned", e.target.checked)}
                  />
                  置顶这份公告
                </label>
              </div>
              <div className="notice-box">
                保存后仍为草稿；从公告列表点击“发布”才会向玩家公开。编辑已发布公告也不会立即改动公开版本。
              </div>
            </form>
          )}
          <div className="editor-footer">
            <Button secondary onClick={exit}>
              取消
            </Button>
            {preview ? (
              <Button
                key="return-editor"
                onClick={(e) => {
                  e.preventDefault();
                  setPreview(false);
                }}
              >
                返回编辑
              </Button>
            ) : (
              <Button key="save-draft" type="submit" form="announcement-form">
                保存草稿
              </Button>
            )}
          </div>
        </>
      )}
    </Modal>
  );
}
export function AnnouncementArticle({ item: a }) {
  return (
    <article className="announcement-article">
      <div className="announcement-article-cover">
        <img
          src={media(a.cover || "street-daylight.png")}
          alt="Deuterium 公告封面"
        />
      </div>
      <div className="announcement-article-meta">
        <span>DEUTERIUM 官方公告</span>
        {a.preview ? (
          <Badge tone="amber">草稿预览</Badge>
        ) : (
          <time>
            {a.publishedAt
              ? new Date(a.publishedAt).toLocaleDateString("zh-CN")
              : a.date}
          </time>
        )}
      </div>
      <h2>{a.title || "公告标题"}</h2>
      <p className="announcement-summary">{a.summary || a.intro}</p>
      <div className="announcement-body">
        {a.body?.split("\n").map((p, i) => (
          <p key={i}>{p || "\u00a0"}</p>
        ))}
      </div>
    </article>
  );
}
export function PublicAnnouncements({ state, open }) {
  const [query, setQuery] = useState("");
  const list = publishedAnnouncements(state).filter((a) =>
    `${a.title}${a.summary}`.includes(query),
  );
  return (
    <div className="public-announcements">
      <div className="workspace-section-head">
        <div>
          <p className="eyebrow">COMMUNITY UPDATES</p>
          <h1>社区公告</h1>
          <p>新的进展，重要的约定。</p>
        </div>
        <label className="search-input">
          <Search size={16} />
          <input
            aria-label="搜索社区公告"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索公告"
          />
        </label>
      </div>
      <div className="announcement-grid">
        {list.map((a) => (
          <button
            key={a.id}
            className="announcement-card"
            onClick={() => open({ type: "announcement", item: a })}
          >
            <div className="announcement-image">
              <img
                src={media(a.cover || "street-daylight.png")}
                alt="公告封面"
              />
            </div>
            <div>
              <span className="tiny-label">
                {a.publishedAt
                  ? new Date(a.publishedAt).toLocaleDateString("zh-CN")
                  : a.date}{" "}
                · 官方公告
              </span>
              {a.pinned && <Badge tone="neutral">置顶</Badge>}
              <h2>{a.title}</h2>
              <p>{a.summary || a.intro}</p>
              <span className="text-button">
                阅读全文 <ArrowUpRight size={16} />
              </span>
            </div>
          </button>
        ))}
      </div>
      {!list.length && (
        <Empty title="没有匹配的已发布公告" text="未发布草稿不会出现在这里。" />
      )}
    </div>
  );
}

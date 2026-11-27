import React, { useEffect, useRef, useState } from "react";
import { Plus, RefreshCw, Search, Sparkles, X } from "lucide-react";
import { Button, Field, Modal, PageHead, Empty, Badge } from "./components.jsx";
import { useUnsavedChanges } from "./unsaved-changes.js";
import { CopyButton } from "./AdminPrimitives.jsx";

export function TitleText({ text }) {
  let color = "#A9B3B8";
  return (
    <>
      {String(text || "")
        .split(/(#[0-9a-fA-F]{6})/g)
        .map((part, i) => {
          if (/^#[0-9a-fA-F]{6}$/.test(part)) {
            color = part;
            return null;
          }
          return (
            <span key={i} style={{ color }}>
              {part}
            </span>
          );
        })}
    </>
  );
}

function stripTitleColors(text) {
  return String(text || "").replace(/#[0-9a-fA-F]{6}/g, "");
}

function parseHex(hex) {
  const clean = hex.replace("#", "");
  return [
    parseInt(clean.slice(0, 2), 16) || 0,
    parseInt(clean.slice(2, 4), 16) || 0,
    parseInt(clean.slice(4, 6), 16) || 0,
  ];
}

function toHex(r, g, b) {
  return (
    "#" +
    [r, g, b]
      .map((n) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, "0"))
      .join("")
  );
}

function generateGradientTitle(plainText, startHex, endHex) {
  const chars = [...plainText];
  if (!chars.length) return "";
  if (chars.length === 1 || startHex.toLowerCase() === endHex.toLowerCase()) {
    return `${startHex.toLowerCase()}${plainText}`;
  }
  const [r1, g1, b1] = parseHex(startHex);
  const [r2, g2, b2] = parseHex(endHex);
  return chars
    .map((ch, idx) => {
      const t = idx / (chars.length - 1);
      const hex = toHex(r1 + (r2 - r1) * t, g1 + (g2 - g1) * t, b1 + (b2 - b1) * t);
      return `${hex}${ch}`;
    })
    .join("");
}

const GRADIENT_PRESETS = [
  { name: "星云粉蓝", from: "#d9afd9", to: "#97d9e1" },
  { name: "流金岁月", from: "#f6d365", to: "#fda085" },
  { name: "翡翠薄荷", from: "#84fab0", to: "#8fd3f4" },
  { name: "暮霞绯红", from: "#ff9a9e", to: "#fecfef" },
  { name: "极光青蓝", from: "#66a6ff", to: "#89f7fe" },
  { name: "纯白银灰", from: "#ffffff", to: "#b8c2cc" },
];

const states = { DRAFT: "草稿", PUBLISHED: "已发布", DISABLED: "已停用" };
const blank = {
  internalId: "",
  displayName: "",
  description: "",
  public: true,
  weight: 0,
  status: "DRAFT",
  version: 0,
};

export default function TitleManagement({ client }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [edit, setEdit] = useState(null);
  const [busy, setBusy] = useState(false);
  const [remove, setRemove] = useState(null);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [gradText, setGradText] = useState("");
  const [gradFrom, setGradFrom] = useState("#d9afd9");
  const [gradTo, setGradTo] = useState("#97d9e1");

  const alive = useRef(true);
  const acting = useRef(false);
  const retry = useRef(null);
  useUnsavedChanges(!!edit, busy);

  const load = async () => {
    const r = await client.request("/api/v1/admin/titles");
    if (alive.current) setItems(r.data.items);
  };
  useEffect(() => {
    alive.current = true;
    load()
      .catch((e) => {
        if (alive.current) setError(e.message);
      })
      .finally(() => {
        if (alive.current) setLoading(false);
      });
    return () => {
      alive.current = false;
    };
  }, [client]);

  const openEditor = (target) => {
    setEdit({ ...target });
    setGradText(stripTitleColors(target.displayName || ""));
    setError("");
  };

  const applyGradient = (from = gradFrom, to = gradTo, text = gradText) => {
    const clean = text.trim() || stripTitleColors(edit?.displayName || "") || "溯行者";
    setGradText(clean);
    setGradFrom(from);
    setGradTo(to);
    setEdit((old) => (old ? { ...old, displayName: generateGradientTitle(clean, from, to) } : old));
  };

  const save = async (value) => {
    if (acting.current) return;
    acting.current = true;
    setBusy(true);
    setError("");
    const data = {
      internalId: value.internalId,
      displayName: value.displayName,
      description: value.description || "",
      public: !!value.public,
      weight: Number(value.weight),
      status: value.status,
      expectedVersion: value.version,
    };
    const key = JSON.stringify([value.id, data]);
    if (retry.current?.key !== key) retry.current = { key, id: crypto.randomUUID() };
    try {
      await client.request(`/api/v1/admin/titles${value.id ? `/${value.id}` : ""}`, {
        method: "POST",
        body: { ...data, clientRequestId: retry.current.id },
      });
      retry.current = null;
      if (alive.current) {
        setEdit(null);
        setRemove(null);
      }
      await load();
    } catch (e) {
      if (alive.current) setError(e.message);
    } finally {
      acting.current = false;
      if (alive.current) setBusy(false);
    }
  };

  const shown = items.filter((t) => {
    if (statusFilter !== "ALL" && t.status !== statusFilter) return false;
    return `${t.internalId} ${t.displayName} ${t.id}`
      .toLowerCase()
      .includes(query.trim().toLowerCase());
  });

  const visibleCount = edit ? [...stripTitleColors(edit.displayName)].length : 0;

  return (
    <>
      <PageHead
        eyebrow="PLAYER TITLES"
        title="头衔管理"
        subtitle="修改显示名会同步到正在佩戴该头衔的玩家。权重越高越靠前，同权重后发布的优先。"
      >
        <Button
          secondary
          disabled={busy}
          onClick={() =>
            load()
              .then(() => setError(""))
              .catch((e) => setError(e.message))
          }
        >
          <RefreshCw size={15} />刷新
        </Button>
        <Button onClick={() => openEditor(blank)}>
          <Plus size={16} />创建头衔
        </Button>
      </PageHead>

      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}

      <div className="adm-toolbar">
        <div className="adm-toolbar-left">
          <div className="adm-pill-group" role="group" aria-label="按头衔状态筛选">
            {[
              ["ALL", `全部 (${items.length})`],
              ["PUBLISHED", `已发布 (${items.filter((i) => i.status === "PUBLISHED").length})`],
              ["DRAFT", `草稿 (${items.filter((i) => i.status === "DRAFT").length})`],
              ["DISABLED", `已停用 (${items.filter((i) => i.status === "DISABLED").length})`],
            ].map(([key, label]) => (
              <button
                key={key}
                type="button"
                className={statusFilter === key ? "is-active" : ""}
                aria-pressed={statusFilter === key}
                onClick={() => setStatusFilter(key)}
              >
                {label}
              </button>
            ))}
          </div>
          <label className="adm-search-box">
            <Search size={15} />
            <input
              aria-label="搜索头衔"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索显示名、内部 ID 或头衔 ID…"
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

      {loading ? (
        <p>正在加载…</p>
      ) : !shown.length ? (
        <Empty title="没有匹配的头衔" />
      ) : (
        <div className="panel table-wrap">
          <table>
            <thead>
              <tr>
                <th>显示名</th>
                <th>内部 ID / 权限节点</th>
                <th>权重</th>
                <th>状态</th>
                <th>可见性</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {shown.map((t) => {
                const permNode =
                  t.id === "title_default" ? "" : `deuterium.title.${t.internalId}`;
                return (
                  <tr key={t.id}>
                    <td>
                      <span
                        style={{
                          display: "inline-block",
                          background: "#17181c",
                          padding: "7px 12px",
                          borderRadius: 8,
                          fontWeight: 550,
                        }}
                      >
                        <TitleText text={t.displayName} />
                      </span>
                    </td>
                    <td>
                      <strong>{t.internalId}</strong>
                      <div style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 4 }}>
                        <small className="muted">
                          {permNode || "所有玩家默认可用"}
                        </small>
                        {permNode && <CopyButton value={permNode} label="权限节点" compact />}
                      </div>
                    </td>
                    <td>{t.weight}</td>
                    <td>
                      <Badge tone={t.status === "PUBLISHED" ? "sage" : "neutral"}>
                        {states[t.status]}
                      </Badge>
                    </td>
                    <td>{t.public ? "公开" : "未公开"}</td>
                    <td>
                      <Button secondary onClick={() => openEditor(t)}>
                        管理
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {edit && (
        <Modal
          title={edit.id ? "管理头衔" : "创建头衔"}
          wide
          close={() => {
            if (!busy) setEdit(null);
          }}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              save(edit);
            }}
          >
            {error && (
              <p role="alert" className="form-error">
                {error}
              </p>
            )}
            <div className="form-grid">
              <Field
                label="内部 ID"
                hint={
                  edit.id === "title_default"
                    ? "默认头衔无需权限。"
                    : `权限节点：deuterium.title.${edit.internalId || "xxx"}（创建后不可修改）`
                }
              >
                <input
                  required
                  disabled={!!edit.id || busy}
                  pattern="[a-z0-9][a-z0-9_-]{0,47}"
                  maxLength={48}
                  value={edit.internalId}
                  onChange={(e) => setEdit({ ...edit, internalId: e.target.value })}
                />
              </Field>
              <div className="form-grid">
                <Field label="权重">
                  <input
                    type="number"
                    required
                    min={-1000000}
                    max={1000000}
                    step={1}
                    disabled={busy}
                    value={edit.weight}
                    onChange={(e) => setEdit({ ...edit, weight: e.target.value })}
                  />
                </Field>
                <Field label="状态">
                  <select
                    disabled={busy || edit.id === "title_default"}
                    value={edit.status}
                    onChange={(e) => setEdit({ ...edit, status: e.target.value })}
                  >
                    {Object.entries(states).map(([v, name]) => (
                      <option key={v} value={v}>
                        {name}
                      </option>
                    ))}
                  </select>
                </Field>
              </div>
            </div>

            <div className="adm-gradient-studio">
              <div className="adm-gradient-studio-head">
                <strong>
                  <Sparkles size={14} style={{ verticalAlign: "-2px", marginRight: 4 }} />
                  可视化渐变配色生成器
                </strong>
                <small>不用手写 #RRGGBB 色码，输入文字选色即可一键生成</small>
              </div>
              <div className="adm-gradient-row">
                <Field label="头衔纯文本">
                  <input
                    value={gradText}
                    maxLength={32}
                    disabled={busy}
                    placeholder="例如：星穹溯行者"
                    onChange={(e) => setGradText(e.target.value)}
                  />
                </Field>
                <label className="adm-color-pick">
                  <span>起始色</span>
                  <input
                    type="color"
                    value={gradFrom}
                    disabled={busy}
                    onChange={(e) => setGradFrom(e.target.value)}
                  />
                </label>
                <label className="adm-color-pick">
                  <span>结束色</span>
                  <input
                    type="color"
                    value={gradTo}
                    disabled={busy}
                    onChange={(e) => setGradTo(e.target.value)}
                  />
                </label>
                <Button
                  secondary
                  disabled={busy}
                  onClick={() => applyGradient(gradFrom, gradTo, gradText)}
                >
                  生成渐变色
                </Button>
              </div>
              <div className="adm-gradient-presets">
                {GRADIENT_PRESETS.map((preset) => (
                  <button
                    key={preset.name}
                    type="button"
                    className="adm-gradient-preset"
                    disabled={busy}
                    onClick={() => applyGradient(preset.from, preset.to, gradText)}
                  >
                    <span
                      className="adm-gradient-swatch"
                      style={{
                        background: `linear-gradient(90deg, ${preset.from}, ${preset.to})`,
                      }}
                    />
                    {preset.name}
                  </button>
                ))}
              </div>
            </div>

            <div className="adm-title-dual-preview" aria-label="头衔双背景实时预览">
              <div className="adm-title-stage is-dark">
                <TitleText text={edit.displayName || "溯行者"} />
                <small>深色 / 游戏内聊天预览</small>
              </div>
              <div className="adm-title-stage is-light">
                <TitleText text={edit.displayName || "溯行者"} />
                <small>浅色页面预览</small>
              </div>
            </div>

            <Field
              label={`色码文本（可见字符 ${visibleCount} / 32）`}
              hint="支持手动微调 #RRGGBB 逐字配色。"
            >
              <input
                required
                disabled={busy}
                maxLength={512}
                value={edit.displayName}
                onChange={(e) => setEdit({ ...edit, displayName: e.target.value })}
                placeholder="#d9afd9参#c3bddc考#adcbde这#97d9e1样"
              />
            </Field>
            <Field label="介绍及获取方式">
              <textarea
                rows={3}
                maxLength={1000}
                disabled={busy}
                value={edit.description || ""}
                onChange={(e) => setEdit({ ...edit, description: e.target.value })}
                placeholder="介绍这个头衔，并说明如何获得"
              />
            </Field>
            <label className="checkbox-row">
              <input
                type="checkbox"
                disabled={busy}
                checked={!!edit.public}
                onChange={(e) => setEdit({ ...edit, public: e.target.checked })}
              />
              公开展示（未公开的头衔仅对已拥有的玩家可见）
            </label>
            <div className="button-row">
              {edit.id && edit.id !== "title_default" && (
                <Button danger secondary disabled={busy} onClick={() => setRemove(edit)}>
                  删除
                </Button>
              )}
              <Button secondary disabled={busy} onClick={() => setEdit(null)}>
                取消
              </Button>
              <Button type="submit" disabled={busy}>
                {busy ? "保存中…" : "保存"}
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {remove && (
        <Modal
          title="删除头衔"
          close={() => {
            if (!busy) setRemove(null);
          }}
        >
          <p>
            正在佩戴的玩家将回退到此前仍可用的头衔。内部 ID「{remove.internalId}」删除后不可复用。
          </p>
          <div className="button-row">
            <Button secondary disabled={busy} onClick={() => setRemove(null)}>
              取消
            </Button>
            <Button danger disabled={busy} onClick={() => save({ ...remove, status: "DELETED" })}>
              确认删除
            </Button>
          </div>
        </Modal>
      )}
    </>
  );
}

import React, { useEffect, useMemo, useState } from "react";
import { Keyboard, RefreshCw, Search, X } from "lucide-react";
import { Button, Empty, Field, Modal, PageHead } from "./components.jsx";
import {
  choices,
  keyClashes,
  keyName,
  keyStatus,
  unbound,
} from "./keybindings.js";
import "./keybindings.css";

const base = "/api/v1/admin/keybindings";
const date = (value) =>
  new Date(value).toLocaleString("zh-CN", { hour12: false });

const CODE_TO_MC = {
  Space: "key.keyboard.space",
  Tab: "key.keyboard.tab",
  Enter: "key.keyboard.enter",
  Backspace: "key.keyboard.backspace",
  Minus: "key.keyboard.minus",
  Equal: "key.keyboard.equal",
  BracketLeft: "key.keyboard.left.bracket",
  BracketRight: "key.keyboard.right.bracket",
  Backslash: "key.keyboard.backslash",
  Semicolon: "key.keyboard.semicolon",
  Quote: "key.keyboard.apostrophe",
  Comma: "key.keyboard.comma",
  Period: "key.keyboard.period",
  Slash: "key.keyboard.slash",
  Backquote: "key.keyboard.grave.accent",
  ArrowUp: "key.keyboard.up",
  ArrowDown: "key.keyboard.down",
  ArrowLeft: "key.keyboard.left",
  ArrowRight: "key.keyboard.right",
  Home: "key.keyboard.home",
  End: "key.keyboard.end",
  PageUp: "key.keyboard.page.up",
  PageDown: "key.keyboard.page.down",
  Insert: "key.keyboard.insert",
  Delete: "key.keyboard.delete",
  CapsLock: "key.keyboard.caps.lock",
};

function mapDomEventToMcKey(e) {
  const c = e.code || "";
  if (/^Key([A-Z])$/.test(c)) return `key.keyboard.${c.slice(3).toLowerCase()}`;
  if (/^Digit([0-9])$/.test(c)) return `key.keyboard.${c.slice(5)}`;
  if (/^F([1-9]|1[0-2])$/.test(c)) return `key.keyboard.${c.toLowerCase()}`;
  if (/^Numpad([0-9])$/.test(c)) return `key.keyboard.keypad.${c.slice(6)}`;
  return CODE_TO_MC[c] || null;
}

function SearchBox({ label, value, change }) {
  return (
    <label className="keys-search">
      <Search size={16} />
      <input
        type="search"
        aria-label={label}
        placeholder={label}
        value={value}
        onChange={(e) => change(e.target.value)}
      />
      {value && (
        <button
          type="button"
          className="adm-search-clear"
          aria-label="清空搜索"
          onClick={() => change("")}
        >
          <X size={14} />
        </button>
      )}
    </label>
  );
}

export default function KeybindingManagement({ client }) {
  const [query, setQuery] = useState("");
  const [offset, setOffset] = useState(0);
  const [page, setPage] = useState(null);
  const [selected, select] = useState("");
  const [error, setError] = useState("");
  const [refresh, reload] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setError("");
    setPage(null);
    const timer = setTimeout(
      async () => {
        try {
          const { data } = await client.request(
            base + "?" + new URLSearchParams({ q: query.trim(), offset }),
            { signal: controller.signal },
          );
          if (controller.signal.aborted) return;
          setPage(data);
          select((id) =>
            data.items.some((p) => p.playerUuid === id)
              ? id
              : data.items[0]?.playerUuid || "",
          );
        } catch (e) {
          if (!controller.signal.aborted) {
            setError(e.message);
            select("");
          }
        }
      },
      query ? 180 : 0,
    );
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [client, query, offset, refresh]);
  return (
    <section className="keys-workspace">
      <PageHead
        eyebrow="PLAYER KEYBINDINGS"
        title="玩家键位"
        subtitle="按分类快速定位玩家按键冲突，支持直接按下键盘录入目标按键。"
      />
      <div className="keys-layout">
        <aside className="panel keys-players" aria-label="玩家列表">
          <SearchBox
            label="搜索游戏 ID 或 UUID"
            value={query}
            change={(q) => {
              setQuery(q);
              setOffset(0);
              select("");
            }}
          />
          {error && (
            <p role="alert" className="auth-error">
              {error}{" "}
              <button
                className="text-button"
                onClick={() => reload((i) => i + 1)}
              >
                重试
              </button>
            </p>
          )}
          {!page && !error && <p role="status">正在读取玩家…</p>}
          {page?.items.length === 0 && (
            <p>{query ? "没有匹配的玩家" : "还没有玩家同步键位"}</p>
          )}
          {page?.items.map((p) => (
            <button
              key={p.playerUuid}
              className="keys-player"
              aria-pressed={selected === p.playerUuid}
              onClick={() => select(p.playerUuid)}
            >
              <strong>{p.gameId}</strong>
              <small>{p.playerUuid}</small>
            </button>
          ))}
          {(offset > 0 || page?.hasMore) && (
            <div className="keys-pages">
              <Button
                secondary
                disabled={offset === 0 || !page}
                onClick={() => setOffset(offset - 30)}
              >
                上一页
              </Button>
              <Button
                secondary
                disabled={!page?.hasMore}
                onClick={() => setOffset(offset + 30)}
              >
                下一页
              </Button>
            </div>
          )}
        </aside>
        {selected ? (
          <PlayerKeys key={selected} uuid={selected} client={client} />
        ) : (
          <div className="panel">
            <Empty
              title="尚未选择玩家"
              text="首次进服同步后，可在这里查看和调整键位。"
            />
          </div>
        )}
      </div>
    </section>
  );
}

function PlayerKeys({ uuid, client }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [refresh, reload] = useState(0);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("全部");
  const [categoryFilter, setCategoryFilter] = useState("ALL");
  const [edit, setEdit] = useState(null);

  useEffect(() => {
    const controller = new AbortController();
    let timer, latest;
    const load = async () => {
      try {
        const suffix = latest ? "?since=" + latest.revision : "";
        const response = await client.request(base + "/" + uuid + suffix, {
          signal: controller.signal,
        });
        if (controller.signal.aborted) return;
        const delta = response.data;
        latest =
          latest && !delta.full
            ? { ...delta, rows: { ...latest.rows, ...delta.rows } }
            : delta;
        setData(latest);
        setError("");
        if (
          latest.online &&
          latest.allowRemote &&
          Object.values(latest.rows).some((r) => keyStatus(r) === "待生效")
        )
          timer = setTimeout(load, 5000);
      } catch (e) {
        if (!controller.signal.aborted) setError(e.message);
      }
    };
    load();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [uuid, client, refresh]);

  const all = Object.entries(data?.rows || {});
  const categories = useMemo(
    () =>
      [...new Set(all.map(([, r]) => r.category).filter(Boolean))].sort((a, b) =>
        a.localeCompare(b, "zh"),
      ),
    [all],
  );
  const search = query.trim().toLowerCase();
  const rows = all
    .filter(
      ([id, r]) =>
        (filter === "全部" || keyStatus(r) === filter) &&
        (categoryFilter === "ALL" || r.category === categoryFilter) &&
        [id, r.label, r.category, keyName(r.value), keyName(r.observed)].some((t) =>
          t.toLowerCase().includes(search),
        ),
    )
    .sort(
      (a, b) =>
        a[1].category.localeCompare(b[1].category, "zh") ||
        a[1].label.localeCompare(b[1].label, "zh"),
    );

  const statusCounts = {
    全部: all.length,
    待生效: all.filter(([, r]) => keyStatus(r) === "待生效").length,
    修改冲突: all.filter(([, r]) => keyStatus(r) === "修改冲突").length,
    已生效: all.filter(([, r]) => keyStatus(r) === "已生效").length,
  };

  return (
    <div className="panel keys-detail">
      <div className="keys-heading">
        <div>
          <h2>{data?.gameId || "读取键位…"}</h2>
          <small>{uuid}</small>
        </div>
        <Button
          secondary
          aria-label="刷新键位"
          onClick={() => reload((v) => v + 1)}
        >
          <RefreshCw size={16} /> 刷新
        </Button>
      </div>
      {error && (
        <p role="alert" className="auth-error">
          {error}
        </p>
      )}
      {data && (
        <>
          <p className="keys-meta">
            {data.online ? "在线" : "离线"} · 远程协助
            {data.allowRemote ? "已开启" : "已关闭"} · 最近同步{" "}
            {date(data.syncedAt)} · 共 {all.length} 项动作
          </p>
          <div className="adm-toolbar" style={{ marginBottom: 14 }}>
            <div className="adm-toolbar-left">
              <div className="adm-pill-group" role="group" aria-label="按同步状态筛选">
                {["全部", "待生效", "修改冲突", "已生效"].map((s) => (
                  <button
                    key={s}
                    type="button"
                    className={filter === s ? "is-active" : ""}
                    onClick={() => setFilter(s)}
                  >
                    {s} ({statusCounts[s] ?? 0})
                  </button>
                ))}
              </div>
              <select
                aria-label="按模组分类筛选"
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                <option value="ALL">全部模组/分类 ({categories.length})</option>
                {categories.map((cat) => (
                  <option key={cat} value={cat}>
                    {cat}
                  </option>
                ))}
              </select>
              <SearchBox label="搜索动作或按键" value={query} change={setQuery} />
            </div>
          </div>
          {rows.length === 0 ? (
            <Empty title="没有匹配的键位" text="可以调整搜索词、模组分类或状态筛选。" />
          ) : (
            <div className="keys-table">
              <table>
                <thead>
                  <tr>
                    <th>动作</th>
                    <th>当前键位</th>
                    <th>同步状态</th>
                    <th>
                      <span className="sr-only">修改键位</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map(([id, r]) => (
                    <tr key={id}>
                      <td>
                        <strong>{r.label}</strong>
                        <small>{r.category}</small>
                      </td>
                      <td>
                        <kbd>{keyName(r.observed)}</kbd>
                        {r.value !== r.observed && (
                          <small>目标 {keyName(r.value)}</small>
                        )}
                      </td>
                      <td>
                        <span
                          className={
                            "keys-status " +
                            (r.conflict
                              ? "conflict"
                              : r.value !== r.observed
                                ? "pending"
                                : "")
                          }
                        >
                          {keyStatus(r)}
                        </span>
                      </td>
                      <td>
                        <Button
                          secondary
                          aria-label={"修改 " + r.label}
                          onClick={() => setEdit({ id, row: r })}
                        >
                          修改
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
      {edit && (
        <EditKey
          client={client}
          uuid={uuid}
          edit={edit}
          data={data}
          close={() => setEdit(null)}
          refresh={() => reload((i) => i + 1)}
        />
      )}
    </div>
  );
}

function EditKey({ client, uuid, edit, data, close, refresh }) {
  const { id, row } = edit;
  const [initial, modifier = ""] = row.value.split(":");
  const [key, setKey] = useState(initial);
  const [mod, setMod] = useState(modifier);
  const [listening, setListening] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [stale, setStale] = useState(false);

  useEffect(() => {
    if (!listening) return;
    const onKeyDown = (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.key === "Escape") {
        setListening(false);
        return;
      }
      // Ignore pure modifier presses until the actual key is pressed
      if (["Control", "Shift", "Alt", "Meta"].includes(e.key)) return;
      const mapped = mapDomEventToMcKey(e);
      if (mapped) {
        setKey(mapped);
        setMod(e.ctrlKey ? "CONTROL" : e.shiftKey ? "SHIFT" : e.altKey ? "ALT" : "");
        setListening(false);
      }
    };
    window.addEventListener("keydown", onKeyDown, true);
    return () => window.removeEventListener("keydown", onKeyDown, true);
  }, [listening]);

  const value = key === unbound || !mod ? key : key + ":" + mod;
  const alternatives = [
    ...new Set([
      ...choices,
      key,
      ...Object.values(data.rows).flatMap((r) => [
        r.value.split(":")[0],
        r.observed.split(":")[0],
      ]),
    ]),
  ];
  const clashes = keyClashes(data.rows, id, value);
  const save = async (e) => {
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await client.request(base + "/" + uuid, {
        method: "PUT",
        body: { action: id, value, expectedRevision: row.revision },
      });
      refresh();
      close();
    } catch (ex) {
      setError(ex.message);
      if (ex.status === 409) {
        setStale(true);
        refresh();
      }
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      title={"修改 · " + row.label}
      close={() => {
        if (!busy) close();
      }}
    >
      <form onSubmit={save} className="keys-edit">
        <p>
          {data.gameId} · 当前 <kbd>{keyName(row.observed)}</kbd>
        </p>

        <div className={`adm-key-capture ${listening ? "is-listening" : ""}`}>
          <div>
            <strong>{listening ? "请直接按下目标键盘按键（按 Esc 取消）…" : "快速按键录入"}</strong>
            <small>
              {listening
                ? "支持组合键（如 Ctrl+K、Shift+R）"
                : `当前已选：${keyName(value)}`}
            </small>
          </div>
          <div className="adm-compact-row-actions">
            <Button
              secondary={!listening}
              type="button"
              disabled={busy}
              onClick={() => setListening((v) => !v)}
            >
              <Keyboard size={15} />
              {listening ? "正在监听…" : "按下键盘录入"}
            </Button>
            <Button
              secondary
              type="button"
              disabled={busy}
              onClick={() => {
                setListening(false);
                setKey(unbound);
                setMod("");
              }}
            >
              解绑
            </Button>
          </div>
        </div>

        <div className="form-grid">
          <Field label="按键（或手动选择）">
            <select value={key} onChange={(e) => setKey(e.target.value)}>
              {alternatives.map((k) => (
                <option key={k} value={k}>
                  {keyName(k)}
                </option>
              ))}
            </select>
          </Field>
          <Field label="组合键">
            <select
              disabled={key === unbound}
              value={mod}
              onChange={(e) => setMod(e.target.value)}
            >
              <option value="">无</option>
              <option value="CONTROL">Ctrl</option>
              <option value="SHIFT">Shift</option>
              <option value="ALT">Alt</option>
            </select>
          </Field>
        </div>

        {clashes.length > 0 && (
          <p className="keys-warning">
            可能与「{clashes.join("」、「")}」的按键冲突。
          </p>
        )}
        {row.missing && (
          <p>当前客户端未注册这个动作。装回对应模组后可继续使用保存的键位。</p>
        )}
        {row.conflict && (
          <p>玩家和后台改过同一动作。可选择玩家当前键位，或重新下发目标键。</p>
        )}
        {!data.allowRemote ? (
          <p>变更会保留到玩家在「按键绑定 → 云键位」开启远程协助。</p>
        ) : (
          !data.online && <p>玩家离线，下次进服后应用。</p>
        )}
        {error && (
          <p role="alert" className="auth-error">
            {error}
          </p>
        )}
        <div className="button-row">
          <Button secondary disabled={busy} onClick={close}>
            关闭
          </Button>
          <Button
            type="submit"
            disabled={busy || stale || (!row.conflict && value === row.value)}
          >
            <Keyboard size={16} />
            {busy ? "正在保存…" : "保存键位"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

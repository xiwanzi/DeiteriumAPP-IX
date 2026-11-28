import React, { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Box, CheckCircle2, Database, FolderUp, Package, RefreshCw, Search, Terminal, UserRound } from "lucide-react";
import { Button, Field, Modal } from "./components.jsx";
import { defaultWeightForStars, extractTaczMeta, supplyGrades } from "./supply-model.js";
import { mutateSupply, pendingSupplyRequest, retrySupply } from "./supply-requests.js";
import SupplyItemTooltip from "./SupplyItemTooltip.jsx";

const PAGE_SIZE = 63;
const RESOURCE_ID = /^[a-z0-9_.-]+:[a-z0-9_./-]+$/;
const TAB_GROUPS = [
  ["vanilla", "Minecraft"],
  ["tacz", "TaCZ"],
  ["mods", "其他模组"],
];
const PICKER_STATE_KEY = "deuterium.supply.picker.state.v1";

function loadPickerState() {
  try {
    const raw = sessionStorage.getItem(PICKER_STATE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function savePickerState(patch) {
  try {
    const prev = loadPickerState();
    sessionStorage.setItem(PICKER_STATE_KEY, JSON.stringify({ ...prev, ...patch }));
  } catch {
    // ignore storage failures
  }
}

let importedSources = [];

function slotName(slot) {
  if (typeof slot !== "string") return "原槽位";
  if (slot.startsWith("storage:")) {
    const index = Number(slot.slice(8));
    return index < 9 ? `快捷栏 ${index + 1}` : `背包 ${index - 8}`;
  }
  return { head: "头部", chest: "胸部", legs: "腿部", feet: "脚部", offhand: "副手" }[slot] || slot;
}

function parseSnapshotFile(data) {
  if (
    !data ||
    data.format !== "deuterium.reward-items.v1" ||
    !["creative", "inventory"].includes(data.kind) ||
    typeof data.name !== "string" ||
    !data.name.trim() ||
    !Array.isArray(data.items) ||
    data.items.length > 50_000
  ) {
    throw new Error("请选择由游戏导出的奖励物品快照 JSON（deuterium.reward-items.v1）。");
  }
  const icons = new Map(
    Object.entries(data.icons ?? {})
      .filter(([, png]) => typeof png === "string" && png.length <= 32_768 && /^iVBORw0KGgo[A-Za-z\d+/]+=*$/.test(png))
      .map(([id, png]) => [id, `data:image/png;base64,${png}`]),
  );
  const items = data.items
    .filter((item) => item && typeof item.itemId === "string" && RESOURCE_ID.test(item.itemId))
    .map((item) => ({
      itemId: item.itemId,
      stackData: typeof item.stackData === "string" ? item.stackData : "",
      name: typeof item.name === "string" && item.name.trim() ? item.name : item.itemId,
      count: Number.isSafeInteger(item.count) && item.count > 0 ? item.count : 1,
      slot: item.slot,
      tooltip: Array.isArray(item.tooltip) ? item.tooltip.slice(0, 48) : undefined,
    }));
  return { kind: data.kind, name: data.name.trim(), items, icons };
}

function ItemThumb({ item, icons }) {
  const url = item?.preview?.icon
    ? `data:image/png;base64,${item.preview.icon}`
    : icons?.get(item?.itemId);
  return url ? <img className="supply-picker-icon" src={url} alt="" /> : <Box size={20} className="supply-picker-fallback-icon" />;
}

export default function SupplyItemPickerModal({
  client,
  scope,
  remaining = 100,
  initialMode = "creative",
  targetName = "",
  onAdded,
  close,
}) {
  const binding = Boolean(targetName);
  const savedState = useRef(loadPickerState()).current;
  const [mode, setMode] = useState(initialMode); // creative | inventory | library | manual
  const [stars, setStars] = useState(() => ([4, 5, 6].includes(Number(savedState.stars)) ? Number(savedState.stars) : 5));
  const [weight, setWeight] = useState(() => (Number(savedState.weight) > 0 ? Number(savedState.weight) : defaultWeightForStars(5)));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  // --- Mode 1: Creative Catalog + Snapshot (Middleware RewardItemPicker UX) ---
  const [creativeSource, setCreativeSource] = useState("server"); // server | snapshot
  const [snapshotList, setSnapshotList] = useState(() => importedSources);
  const [selectedSnapshot, setSelectedSnapshot] = useState(() => importedSources[0]?.name || "");
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState(() => savedState.category || "minecraft:buildingBlocks");
  const [categoryPage, setCategoryPage] = useState(() => (Number.isInteger(savedState.page) && savedState.page >= 0 ? savedState.page : 0));
  const [searchPage, setSearchPage] = useState(0);
  const [catalogTabs, setCatalogTabs] = useState([]);
  const [catalogData, setCatalogData] = useState(null);
  const [catalogError, setCatalogError] = useState("");
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [selectedCreative, setSelectedCreative] = useState(null);
  const [hovered, setHovered] = useState(null);
  const viewportRef = useRef(null);

  // --- Mode 2: Online Player Inventory ---
  const invKey = `${scope}:inventory-capture`;
  const playerStorageKey = `deuterium.supply.inventory-player.${scope}`;
  const [playerInput, setPlayerInput] = useState(() => sessionStorage.getItem(playerStorageKey) || "");
  const [invSnapshot, setInvSnapshot] = useState(null);
  const [selectedSlot, setSelectedSlot] = useState(null);
  const [addedSlots, setAddedSlots] = useState(() => new Set());
  const pendingInv = pendingSupplyRequest(invKey);

  // --- Mode 3: Existing Server Item Library ---
  const [libQuery, setLibQuery] = useState("");
  const [libItems, setLibItems] = useState([]);
  const [libLoading, setLibLoading] = useState(false);
  const [selectedLib, setSelectedLib] = useState(null);

  // --- Mode 4: Manual Item ID / SNBT ---
  const [manualItemId, setManualItemId] = useState("");
  const [manualName, setManualName] = useState("");
  const [manualStackData, setManualStackData] = useState("");

  const usingServerCatalog = mode === "creative" && creativeSource === "server";
  const activeSnapshot = snapshotList.find((s) => s.name === selectedSnapshot) || snapshotList[0] || null;
  const needle = query.trim().toLowerCase();
  const isGlobalSearch = needle.length > 0;
  const activePage = isGlobalSearch ? searchPage : categoryPage;
  const effectiveCategory = isGlobalSearch ? "" : category;
  const snapshotFiltered = activeSnapshot
    ? activeSnapshot.items.filter((item) => `${item.name} ${item.itemId}`.toLowerCase().includes(needle))
    : [];

  const categoryGroup = catalogTabs.find((tab) => tab.id === category)?.group ?? "vanilla";
  const shownPage = usingServerCatalog ? catalogData?.page ?? 0 : activePage;
  const totalPages = usingServerCatalog
    ? Math.max(1, catalogData?.pages ?? 1)
    : Math.max(1, Math.ceil(snapshotFiltered.length / PAGE_SIZE));
  const visibleCreative = usingServerCatalog
    ? catalogData?.items ?? []
    : snapshotFiltered.slice(activePage * PAGE_SIZE, (activePage + 1) * PAGE_SIZE);

  // Persist category, page, stars, weight so reopening restores exact browsing position
  useEffect(() => {
    savePickerState({ category, page: categoryPage, stars, weight });
  }, [category, categoryPage, stars, weight]);

  // Load server creative item catalog when in creative/server mode (global search when query non-empty)
  useEffect(() => {
    if (!usingServerCatalog) return;
    let alive = true;
    const timer = window.setTimeout(() => {
      setCatalogLoading(true);
      setCatalogError("");
      const params = new URLSearchParams({
        query: needle,
        page: String(activePage),
        category: effectiveCategory,
      });
      client
        .request(`/api/v1/admin/supply/item-catalog?${params}`)
        .then((res) => {
          if (!alive) return;
          const data = res.data;
          if (data?.categories?.length && !catalogTabs.length) {
            setCatalogTabs(data.categories);
          }
          setCatalogData(data);
        })
        .catch((err) => {
          if (alive) {
            setCatalogError(err.message || "无法读取客户端物品目录");
            setCatalogData(null);
          }
        })
        .finally(() => {
          if (alive) setCatalogLoading(false);
        });
    }, needle ? 180 : 0);
    return () => {
      alive = false;
      window.clearTimeout(timer);
    };
  }, [usingServerCatalog, needle, activePage, effectiveCategory, client]);

  // Auto-load item library when entering library tab
  const loadLibrary = async (q = libQuery) => {
    setLibLoading(true);
    setError("");
    try {
      const r = await client.request(`/api/v1/admin/core/items?limit=60&q=${encodeURIComponent(q.trim())}`);
      setLibItems((r.data?.items || []).filter((item) => !item.archived));
    } catch (e) {
      setError(e.message);
    } finally {
      setLibLoading(false);
    }
  };

  useEffect(() => {
    if (mode === "library" && libItems.length === 0 && !libLoading) {
      void loadLibrary("");
    }
  }, [mode]);

  useLayoutEffect(() => {
    if (viewportRef.current) viewportRef.current.scrollTop = 0;
  }, [usingServerCatalog, catalogData, activeSnapshot, needle, shownPage]);

  const switchCategory = (next) => {
    setCategory(next);
    setQuery("");
    setCategoryPage(0);
    setSearchPage(0);
    setSelectedCreative(null);
    setHovered(null);
    setError("");
  };

  const setCurrentPage = (next) => {
    if (isGlobalSearch) setSearchPage(next);
    else setCategoryPage(next);
  };

  const handleStarsChange = (nextStars) => {
    setStars(nextStars);
    setWeight(defaultWeightForStars(nextStars));
  };

  const handleImportSnapshot = async (file) => {
    if (!file) return;
    setError("");
    try {
      if (file.size > 32 * 1024 * 1024) throw new Error("物品快照不能超过 32 MB。");
      const parsed = parseSnapshotFile(JSON.parse(await file.text()));
      importedSources = [
        ...importedSources.filter((s) => !(s.kind === parsed.kind && s.name === parsed.name)),
        parsed,
      ];
      setSnapshotList(importedSources);
      setSelectedSnapshot(parsed.name);
      setCreativeSource("snapshot");
      setCategoryPage(0);
      setSearchPage(0);
      setQuery("");
      setSelectedCreative(null);
      setNotice(`已载入快照「${parsed.name}」（共 ${parsed.items.length} 项）`);
    } catch (e) {
      setError(e instanceof SyntaxError ? "无法解析此 JSON 文件。" : e.message);
    }
  };

  const readPlayerInventory = async () => {
    setBusy(true);
    setError("");
    setNotice("");
    setInvSnapshot(null);
    setSelectedSlot(null);
    try {
      const val = playerInput.trim();
      if (!val) throw new Error("请输入在线玩家的游戏 ID 或 UUID");
      const res = await client.request(`/api/v1/admin/supply/inventory?player=${encodeURIComponent(val)}`);
      setInvSnapshot(res.data);
      setAddedSlots(new Set());
      sessionStorage.setItem(playerStorageKey, val);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const importAndCommit = async (itemId, stackData, displayName) => {
    const trimmedId = String(itemId || "").trim();
    if (!RESOURCE_ID.test(trimmedId) || trimmedId === "minecraft:air") {
      throw new Error("物品 ID 格式无效（例如 minecraft:diamond 或 tacz:modern_kinetic_gun）。");
    }
    const res = await client.request("/api/v1/admin/supply/items/import", {
      method: "POST",
      body: {
        itemId: trimmedId,
        stackData: String(stackData || "").trim(),
        name: String(displayName || "").trim() || trimmedId,
      },
    });
    const savedItem = res.data;
    const tacz = extractTaczMeta(stackData || savedItem.stackData);
    onAdded(savedItem, {
      stars,
      weight,
      name: String(displayName || "").trim() || savedItem.displayName,
      stackData: stackData || savedItem.stackData || "",
      gunId: tacz.gunId,
      displayId: tacz.displayId,
    });
    return savedItem;
  };

  const handleConfirm = async (recoverInv = false) => {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      if (mode === "creative") {
        if (!selectedCreative) throw new Error("请先在物品网格中点击选中一件物品。");
        const saved = await importAndCommit(
          selectedCreative.itemId,
          selectedCreative.stackData,
          selectedCreative.preview?.name || selectedCreative.name,
        );
        if (binding) {
          close();
          return;
        }
        setNotice(`已入库「${saved.displayName}」（${saved.itemRef}）并添加到当前卡池，可继续勾选下一件。`);
        setSelectedCreative(null);
      } else if (mode === "inventory") {
        const body = recoverInv
          ? null
          : {
              serverId: invSnapshot.serverId,
              playerUuid: invSnapshot.playerUuid,
              sessionEpoch: invSnapshot.sessionEpoch,
              slot: selectedSlot.slot,
              fingerprint: selectedSlot.fingerprint,
            };
        const saved = recoverInv
          ? await retrySupply(client, invKey)
          : await mutateSupply(client, invKey, "/api/v1/admin/supply/inventory/capture", body);
        onAdded(saved, { stars, weight, name: saved.displayName });
        if (binding) {
          close();
          return;
        }
        const orig = recoverInv ? pendingSupplyRequest(invKey)?.body : body;
        if (orig && invSnapshot?.playerUuid === orig.playerUuid && invSnapshot?.sessionEpoch === orig.sessionEpoch) {
          setAddedSlots((old) => new Set([...old, `${orig.slot}:${orig.fingerprint}`]));
        }
        setSelectedSlot(null);
        setNotice(`已保存背包物品「${saved.displayName}」入库并加入当前卡池。`);
      } else if (mode === "library") {
        if (!selectedLib) throw new Error("请先选择一条物品库版本。");
        onAdded(selectedLib, { stars, weight, name: selectedLib.displayName });
        if (binding) {
          close();
          return;
        }
        setNotice(`已从物品库添加「${selectedLib.displayName}」到当前卡池。`);
        setSelectedLib(null);
      } else if (mode === "manual") {
        const saved = await importAndCommit(manualItemId, manualStackData, manualName);
        if (binding) {
          close();
          return;
        }
        setNotice(`已校验入库「${saved.displayName}」（${saved.itemRef}）并加入当前卡池。`);
        setManualItemId("");
        setManualName("");
        setManualStackData("");
      }
    } catch (e) {
      setError(e.message || "操作失败，请重试。");
      if (mode === "inventory" && ["INVENTORY_CHANGED", "PLAYER_SESSION_CHANGED"].includes(e.code)) {
        setSelectedSlot(null);
        setInvSnapshot(null);
      }
    } finally {
      setBusy(false);
    }
  };

  const canConfirm = (() => {
    if (busy || (!binding && remaining <= 0)) return false;
    if (mode === "creative") return Boolean(selectedCreative);
    if (mode === "inventory") return Boolean(selectedSlot) && !pendingInv;
    if (mode === "library") return Boolean(selectedLib);
    if (mode === "manual") return Boolean(manualItemId.trim());
    return false;
  })();

  const selectedSummary = (() => {
    if (mode === "creative" && selectedCreative) {
      const tacz = extractTaczMeta(selectedCreative.stackData);
      return {
        name: selectedCreative.preview?.name || selectedCreative.name,
        sub: tacz.gunId ? `${selectedCreative.itemId} · 枪械 ${tacz.gunId}` : selectedCreative.itemId,
        thumb: selectedCreative,
      };
    }
    if (mode === "inventory" && selectedSlot) {
      return {
        name: selectedSlot.name || selectedSlot.itemId,
        sub: `${slotName(selectedSlot.slot)} · ${selectedSlot.itemId}`,
      };
    }
    if (mode === "library" && selectedLib) {
      return {
        name: selectedLib.displayName,
        sub: `${selectedLib.itemRef} @ v${selectedLib.revision}`,
      };
    }
    if (mode === "manual" && manualItemId.trim()) {
      return {
        name: manualName.trim() || manualItemId.trim(),
        sub: manualStackData.trim() ? `${manualItemId.trim()} · 含自定义 SNBT` : manualItemId.trim(),
      };
    }
    return null;
  })();

  const confirmText = busy
    ? "正在入库…"
    : binding
      ? mode === "library"
        ? "绑定此物品版本"
        : "入库并绑定此奖品"
      : mode === "library"
        ? "加入当前奖池"
        : "入库并加入奖池";

  const renderSlotButton = (slot) => {
    const done = addedSlots.has(`${slot.slot}:${slot.fingerprint}`);
    return (
      <button
        type="button"
        key={slot.slot}
        className={`supply-inventory-slot ${selectedSlot?.slot === slot.slot ? "selected" : ""}`}
        aria-pressed={selectedSlot?.slot === slot.slot}
        disabled={busy || !slot.savable || done}
        title={slot.reason || `${slot.name}\n${slot.itemId}`}
        onClick={() => {
          setSelectedSlot(slot);
          setError("");
        }}
      >
        <span className="supply-slot-position">{slotName(slot.slot)}</span>
        {slot.amount > 0 ? (
          <>
            <Box size={18} aria-hidden="true" />
            <strong>{slot.name || slot.itemId}</strong>
            <span className="supply-slot-count">×{slot.amount}</span>
          </>
        ) : (
          <span className="muted">空</span>
        )}
        {done ? <small>已加入</small> : slot.reason && <small>不可入库</small>}
      </button>
    );
  };

  return (
    <Modal
      title={binding ? `为「${targetName}」选择并绑定物品` : "物品选择器 · 入库并添加到奖池"}
      className="supply-picker-modal"
      wide
      close={() => {
        if (!busy) close();
      }}
    >
      <div className="supply-picker-tabs" role="tablist" aria-label="物品来源">
        {[
          ["creative", "创造物品栏", Package],
          ["inventory", "在线玩家背包", UserRound],
          ["library", "已有物品库", Database],
          ["manual", "手动 ID / SNBT", Terminal],
        ].map(([id, label, IconComp]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={mode === id}
            className={mode === id ? "is-active" : ""}
            onClick={() => {
              setMode(id);
              setError("");
              setHovered(null);
            }}
          >
            <IconComp size={15} />
            <span>{label}</span>
          </button>
        ))}
      </div>

      {error && (
        <p className="form-error supply-picker-banner" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="supply-notice supply-picker-banner" role="status">
          <CheckCircle2 size={15} />
          <span>{notice}</span>
        </p>
      )}

      <div className="supply-picker-stage">
        {mode === "creative" && (
          <div className="supply-creative-pane">
            <div className="supply-picker-searchbar">
              <div className="supply-search-input">
                <Search size={15} />
                <input
                  value={query}
                  maxLength={120}
                  onChange={(e) => {
                    setQuery(e.target.value);
                    setSearchPage(0);
                    setSelectedCreative(null);
                  }}
                  placeholder="全局搜索全部物品（支持中文名、英文 ID、枪械名，清空自动返回上次分类）…"
                />
                {isGlobalSearch && (
                  <button
                    type="button"
                    className="supply-search-clear"
                    onClick={() => {
                      setQuery("");
                      setSearchPage(0);
                      setSelectedCreative(null);
                    }}
                  >
                    清除
                  </button>
                )}
              </div>
              {snapshotList.length > 0 && (
                <select
                  className="supply-source-select"
                  value={creativeSource === "server" ? "__server__" : selectedSnapshot}
                  onChange={(e) => {
                    if (e.target.value === "__server__") {
                      setCreativeSource("server");
                    } else {
                      setCreativeSource("snapshot");
                      setSelectedSnapshot(e.target.value);
                    }
                    setCategoryPage(0);
                    setSearchPage(0);
                    setSelectedCreative(null);
                  }}
                >
                  <option value="__server__">服务器资源目录</option>
                  {snapshotList.map((s) => (
                    <option key={s.name} value={s.name}>
                      快照：{s.name}
                    </option>
                  ))}
                </select>
              )}
              <label className="supply-import-btn">
                <FolderUp size={15} />
                <span>导入快照 JSON</span>
                <input
                  type="file"
                  accept=".json,application/json"
                  onChange={(e) => {
                    void handleImportSnapshot(e.currentTarget.files?.[0]);
                    e.currentTarget.value = "";
                  }}
                />
              </label>
            </div>

            {usingServerCatalog && catalogTabs.length > 0 && !isGlobalSearch && (
              <div className="supply-creative-tabs">
                <div className="supply-creative-groups" role="group" aria-label="物品分组">
                  {TAB_GROUPS.map(([group, label]) => {
                    const firstTab = catalogTabs.find((tab) => tab.group === group);
                    if (!firstTab) return null;
                    return (
                      <button
                        key={group}
                        type="button"
                        aria-pressed={categoryGroup === group}
                        onClick={() => switchCategory(firstTab.id)}
                      >
                        {label}
                      </button>
                    );
                  })}
                </div>
                <div className="supply-creative-cats" role="group" aria-label="物品分类">
                  {catalogTabs
                    .filter((tab) => tab.group === categoryGroup)
                    .map((tab) => (
                      <button
                        key={tab.id}
                        type="button"
                        aria-pressed={tab.id === category}
                        onClick={() => switchCategory(tab.id)}
                      >
                        {tab.name}
                      </button>
                    ))}
                </div>
              </div>
            )}

            {usingServerCatalog && isGlobalSearch && (
              <div className="supply-global-search-bar">
                <span>
                  正在全部分类中全局搜索 <strong>「{query.trim()}」</strong>
                </span>
                <button
                  type="button"
                  onClick={() => {
                    setQuery("");
                    setSearchPage(0);
                    setSelectedCreative(null);
                  }}
                >
                  返回上次分类（{catalogTabs.find((t) => t.id === category)?.name || "建筑方块"}）
                </button>
              </div>
            )}

            <div
              className="supply-creative-viewport"
              ref={viewportRef}
              onScroll={() => setHovered(null)}
            >
              {visibleCreative.length > 0 ? (
                <div className="supply-creative-grid">
                  {visibleCreative.map((item, idx) => {
                    const active = selectedCreative === item;
                    const displayName = item.preview?.name || item.name;
                    return (
                      <button
                        type="button"
                        key={`${shownPage}:${idx}:${item.itemId}`}
                        className={`supply-creative-cell ${active ? "is-selected" : ""}`}
                        aria-pressed={active}
                        onMouseEnter={(e) =>
                          setHovered({
                            item,
                            anchor: { x: e.clientX, y: e.clientY, shift: e.shiftKey },
                          })
                        }
                        onMouseLeave={() => setHovered(null)}
                        onClick={() => {
                          setSelectedCreative(item);
                          setError("");
                        }}
                      >
                        <ItemThumb item={item} icons={activeSnapshot?.icons} />
                        <span>{displayName}</span>
                      </button>
                    );
                  })}
                </div>
              ) : catalogLoading ? (
                <div className="supply-picker-empty">正在读取服务器资源物品目录…</div>
              ) : catalogError ? (
                <div className="supply-picker-empty is-error">
                  <strong>{catalogError}</strong>
                  <span>你仍可点击右上角「导入快照 JSON」，或切换到「在线玩家背包 / 已有物品库」。</span>
                </div>
              ) : (
                <div className="supply-picker-empty">
                  {needle ? "没有找到匹配的物品，可尝试切换至「全局搜索」分类。" : "当前分类下暂无物品。"}
                </div>
              )}
            </div>

            <div className="supply-picker-pager">
              <span className="muted">
                {usingServerCatalog
                  ? catalogData
                    ? `服务器资源目录 · 共 ${catalogData.total} 项`
                    : "服务器资源目录"
                  : activeSnapshot
                    ? `快照 ${activeSnapshot.name} · 共 ${snapshotFiltered.length} 项`
                    : ""}
              </span>
              <div className="supply-pager-btns">
                <Button
                  type="button"
                  secondary
                  disabled={shownPage <= 0 || catalogLoading}
                  onClick={() => {
                    setHovered(null);
                    setCurrentPage(Math.max(0, shownPage - 1));
                  }}
                >
                  上一页
                </Button>
                <span className="supply-pager-num">
                  {shownPage + 1} / {totalPages}
                </span>
                <Button
                  type="button"
                  secondary
                  disabled={shownPage + 1 >= totalPages || catalogLoading}
                  onClick={() => {
                    setHovered(null);
                    setCurrentPage(shownPage + 1);
                  }}
                >
                  下一页
                </Button>
              </div>
            </div>
          </div>
        )}

        {mode === "inventory" && (
          <div className="supply-scroll-pane">
            <p className="muted" style={{ marginBottom: 12 }}>
              读取在线玩家当前背包，点击槽位后自动生成不可变物品模板入库并加入奖池；不会消耗玩家原背包物品。
            </p>
            <form
              className="supply-picker-searchbar"
              onSubmit={(e) => {
                e.preventDefault();
                void readPlayerInventory();
              }}
            >
              <div className="supply-search-input">
                <UserRound size={15} />
                <input
                  value={playerInput}
                  disabled={busy}
                  onChange={(e) => setPlayerInput(e.target.value)}
                  placeholder="输入在线玩家的游戏 ID 或 UUID"
                />
              </div>
              <Button type="submit" disabled={busy}>
                <RefreshCw size={15} />
                {busy ? "读取中…" : "读取背包"}
              </Button>
            </form>
            {pendingInv && (
              <div className="supply-notice" style={{ marginTop: 10 }}>
                <span>有一笔背包入库请求尚未确认，可点击恢复避免重复建库。</span>
                <Button type="button" secondary disabled={busy} onClick={() => void handleConfirm(true)}>
                  恢复原保存请求
                </Button>
              </div>
            )}
            {invSnapshot && (
              <div style={{ marginTop: 12 }}>
                <div className="supply-section-head">
                  <strong>
                    {invSnapshot.gameId}（节点：{invSnapshot.serverId}）
                  </strong>
                  <small>{invSnapshot.slots.filter((s) => s.amount > 0).length} 个非空槽位</small>
                </div>
                <h4 className="supply-subhead">主背包</h4>
                <div className="supply-inventory-grid">
                  {invSnapshot.slots
                    .filter((s) => s.slot.startsWith("storage:") && Number(s.slot.slice(8)) >= 9)
                    .map(renderSlotButton)}
                </div>
                <h4 className="supply-subhead">快捷栏</h4>
                <div className="supply-inventory-grid">
                  {invSnapshot.slots
                    .filter((s) => s.slot.startsWith("storage:") && Number(s.slot.slice(8)) < 9)
                    .map(renderSlotButton)}
                </div>
                <h4 className="supply-subhead">装备与副手</h4>
                <div className="supply-inventory-grid equipment">
                  {invSnapshot.slots.filter((s) => !s.slot.startsWith("storage:")).map(renderSlotButton)}
                </div>
              </div>
            )}
          </div>
        )}

        {mode === "library" && (
          <div className="supply-scroll-pane">
            <form
              className="supply-picker-searchbar"
              onSubmit={(e) => {
                e.preventDefault();
                void loadLibrary(libQuery);
              }}
            >
              <div className="supply-search-input">
                <Search size={15} />
                <input
                  value={libQuery}
                  onChange={(e) => setLibQuery(e.target.value)}
                  placeholder="搜索已入库的物品名称、物品 ID 或 itemRef…"
                />
              </div>
              <Button type="submit" disabled={libLoading}>
                {libLoading ? "搜索中…" : "搜索物品库"}
              </Button>
            </form>
            {libLoading && !libItems.length ? (
              <div className="supply-picker-empty">正在加载服务器物品库…</div>
            ) : !libItems.length ? (
              <div className="supply-picker-empty">未找到匹配的物品库版本。</div>
            ) : (
              <div className="supply-lib-list">
                {libItems.map((item) => {
                  const active = selectedLib?.itemRef === item.itemRef && selectedLib?.revision === item.revision;
                  return (
                    <button
                      type="button"
                      key={`${item.itemRef}-${item.revision}`}
                      className={`supply-lib-row ${active ? "is-selected" : ""}`}
                      onClick={() => {
                        setSelectedLib(item);
                        setError("");
                      }}
                    >
                      <div className="supply-lib-main">
                        <strong>{item.displayName}</strong>
                        <small>{item.itemRef} · v{item.revision}</small>
                      </div>
                      <span className="supply-lib-servers">
                        {(item.compatibleServerIds || []).join(" / ") || "全服"}
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {mode === "manual" && (
          <div className="supply-scroll-pane">
            <p className="muted" style={{ marginBottom: 12 }}>
              直接输入 Minecraft / 模组物品 ID 与可选完整 ItemStack SNBT，点击确认后由在线子服校验并写入物品库。
            </p>
            <div className="supply-form-grid">
              <Field label="物品 ID（必填）">
                <input
                  value={manualItemId}
                  onChange={(e) => setManualItemId(e.target.value)}
                  placeholder="例如 minecraft:diamond 或 tacz:modern_kinetic_gun"
                />
              </Field>
              <Field label="展示名称（可选）">
                <input
                  value={manualName}
                  maxLength={128}
                  onChange={(e) => setManualName(e.target.value)}
                  placeholder="留空则默认使用物品 ID"
                />
              </Field>
            </div>
            <Field label="ItemStack SNBT（可选）">
              <textarea
                rows={6}
                value={manualStackData}
                onChange={(e) => setManualStackData(e.target.value)}
                placeholder='{id:"tacz:modern_kinetic_gun",count:1,components:{"minecraft:custom_data":{GunId:"tacz:ak47"}}}'
              />
            </Field>
          </div>
        )}
      </div>

      <footer className="supply-picker-footer">
        <div className="supply-picker-selected">
          {selectedSummary ? (
            <>
              {selectedSummary.thumb && (
                <ItemThumb item={selectedSummary.thumb} icons={activeSnapshot?.icons} />
              )}
              <div>
                <strong>{selectedSummary.name}</strong>
                <small>{selectedSummary.sub}</small>
              </div>
            </>
          ) : (
            <span className="muted">请在上方选择一件物品</span>
          )}
        </div>

        {!binding && (
          <div className="supply-picker-grade">
            <label>
              <span>品质</span>
              <select
                value={stars}
                disabled={busy}
                onChange={(e) => handleStarsChange(Number(e.target.value))}
              >
                {supplyGrades.map((g) => (
                  <option key={g.stars} value={g.stars}>
                    {g.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="supply-picker-weight">
              <span>权重</span>
              <input
                type="number"
                min={1}
                max={1000000}
                value={weight}
                disabled={busy}
                onChange={(e) => setWeight(Math.max(1, Number(e.target.value) || 1))}
              />
            </label>
          </div>
        )}

        <div className="supply-picker-actions">
          <Button type="button" secondary disabled={busy} onClick={close}>
            {binding ? "取消" : "完成"}
          </Button>
          <Button type="button" disabled={!canConfirm} onClick={() => void handleConfirm(false)}>
            {confirmText}
          </Button>
        </div>
      </footer>

      {hovered && (
        <SupplyItemTooltip
          client={client}
          name={hovered.item.name}
          itemId={hovered.item.itemId}
          anchor={hovered.anchor}
          preview={hovered.item.preview}
          native={hovered.item.tooltip}
        />
      )}
    </Modal>
  );
}
